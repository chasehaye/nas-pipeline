# GitHub Actions

Two workflows live in [`workflows/`](workflows):

| Workflow | Trigger | Purpose |
| --- | --- | --- |
| [`ci.yml`](workflows/ci.yml) | every push & PR | run Go unit tests |
| [`deploy.yml`](workflows/deploy.yml) | push to `main`, or manual | build images, roll out to the k3s server |

---

## CI — `ci.yml`

Runs automatically on every push and pull request. It runs `go test -race` for each
Go module in a matrix (`ladd-admin`, `normalizer`, `filter`, `cache-writer`, `api`).
Nothing to trigger by hand — a red check on a PR means a test failed.

---

## Deploy — `deploy.yml`

SSHes into the k3s server through the Cloudflare tunnel and runs
[`deploy/deploy.sh`](../deploy/deploy.sh), which builds each image, imports it into
k3s, applies the prod overlay, and rolls out the pods.

### Automatic (push to `main`)

Every push to `main` deploys with the safe defaults:

- **all** app services
- **cached** build (fast)
- **apps only** — infra (Kafka/Redis/Postgres) is left untouched

You don't do anything — merging to `main` ships it.

### Manual (the "Run workflow" button)

For anything non-default, go to **Actions → deploy → Run workflow** and set the inputs:

| Input | Default | Effect |
| --- | --- | --- |
| **Service** | `all` | Deploy just one service (e.g. `api`) instead of everything |
| **Clean rebuild** | off | Build with `--no-cache` (no Docker layer cache) — slower, fully fresh |
| **Also apply infra** | off | Re-apply Kafka/Redis/Postgres/monitoring manifests |

### Common tasks

- **Ship the latest code** → just push to `main`.
- **Force a clean rebuild of one service** → Run workflow → Service: `<name>`, Clean rebuild: ✓.
- **Apply an infra change** (e.g. edited `deploy/k8s/base/infra/redis.yaml`) → Run workflow → Also apply infra: ✓.

> Infra is intentionally **not** touched on a normal deploy, so a routine code push
> never restarts your stateful stores. Use the infra checkbox on purpose.

---

## Prerequisites (one-time setup)

These are configured in **Settings → Secrets and variables → Actions**:

| Secret | What it is |
| --- | --- |
| `SERVER_HOST` | the **Cloudflare tunnel hostname** for the k3s box (not the LAN IP) |
| `SERVER_USER` | SSH user on the server |
| `SERVER_SSH_KEY` | private key for that user |
| `APP_DIR` | path to this repo on the server |

Also required on the server itself:

- **Cloudflare tunnel** with SSH access to the k3s box (the runner can't reach a LAN IP directly).
- **Passwordless sudo** for the deploy user — `deploy.sh` runs `sudo k3s kubectl`,
  `sudo k3s ctr`, and `docker`. A non-interactive SSH session can't answer a sudo
  password prompt, so the deploy user needs `NOPASSWD` for those (or the right group
  membership). If a deploy hangs, this is the usual cause.

---

## Notes

- **Builds run on the server**, not the GitHub runner — so deploys load the k3s box's
  CPU/RAM. `Clean rebuild` is the heaviest option.
- Deploys never overlap: a `concurrency` group serializes them.
- Hand-created secrets on the cluster (`ladd`, `swim`, `postgres`) are **not** managed
  here; `deploy.sh` only warns if one is missing.
