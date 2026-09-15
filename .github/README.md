# GitHub Actions

Two workflows, both in [`workflows/`](workflows):

| Workflow | Trigger | Purpose |
| --- | --- | --- |
| [`ci.yml`](workflows/ci.yml) | every push and PR | run Go unit tests |
| [`deploy.yml`](workflows/deploy.yml) | push to `main`, or manual | build images and roll out to the k3s server |

## CI

`ci.yml` runs `go test -race` for each Go module (`ladd-admin`, `normalizer`,
`filter`, `cache-writer`, `api`) as a matrix, on every push and PR. There's nothing
to run by hand. A red check on a PR means a test failed.

## Deploy

`deploy.yml` SSHes into the k3s server over the Cloudflare tunnel and runs
[`deploy/deploy.sh`](../deploy/deploy.sh). That script builds each image, imports it
into k3s, applies the prod overlay, and rolls the pods.

Pushing to `main` deploys automatically with safe defaults: all app services, a
cached build, apps only (infra is left alone). Merging is the deploy.

For anything else, use Actions → deploy → Run workflow:

| Input | Default | Effect |
| --- | --- | --- |
| Service | `all` | deploy one service (e.g. `api`) instead of everything |
| Clean rebuild | off | build with `--no-cache`; slower but fully fresh |
| Also apply infra | off | re-apply the Kafka/Redis/Postgres/monitoring manifests |

Infra is skipped on a normal deploy on purpose, so a routine code push never
restarts the stateful stores. Only tick "Also apply infra" when you mean to.

## Setup (one-time)

Repo secrets, under Settings → Secrets and variables → Actions:

| Secret | What it is |
| --- | --- |
| `SERVER_HOST` | Cloudflare tunnel hostname for the k3s box (not the LAN IP) |
| `SERVER_USER` | SSH user on the server |
| `SERVER_SSH_KEY` | private key for that user |
| `APP_DIR` | path to this repo on the server |

On the server you also need a Cloudflare tunnel with SSH access (the runner can't
reach a LAN IP), and passwordless sudo for the deploy user. `deploy.sh` runs
`sudo k3s kubectl`, `sudo k3s ctr`, and `docker`, and a non-interactive SSH session
can't answer a sudo password prompt, so give that user NOPASSWD for those (or the
right group membership). A deploy that hangs is usually this.

## Notes

Builds run on the server, not the GitHub runner, so a deploy loads the k3s box's CPU
and RAM; a clean rebuild is the heaviest. Deploys don't overlap (a `concurrency`
group serializes them). The hand-created cluster secrets (`ladd`, `swim`,
`postgres`) aren't managed here; `deploy.sh` just warns if one is missing.
