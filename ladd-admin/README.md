# ladd-admin

Control-plane service + CLI for securely updating the **LADD** aircraft-block
list from anywhere. The LADD Industry file is CUI, so every upload is:

- **encrypted** to the server's public key on the operator's machine
  (confidentiality — plaintext never touches the transport, Cloudflare, or any
  proxy; it is decrypted only inside the cluster), and
- **signed** with the operator's private key (authenticity — the server rejects
  anything not signed by the authorized operator).

---

## How it works

```
operator machine                         in-cluster (trusted)
────────────────                         ────────────────────
ladd-upload                              ladd-admin service
  encrypt → server pubkey                  1. verify signature (operator pubkey)
  sign    → operator privkey  ──POST──►    2. decrypt (server privkey)
        (ciphertext + signature)           3. validate (fail-closed)
                                           4. patch the `ladd` Secret
                                                   │  (k8s syncs the mount)
                                                   ▼
                                           filter hot-reloads → new list live
```

Two keypairs, each with one job:

| Keypair | Private half | Public half | Purpose |
|---|---|---|---|
| **server** (age / X25519) | in the cluster Secret | with the CLI (`--recipient`) | encryption |
| **operator** (ed25519) | with the CLI (`--sign-key`) | in the cluster Secret | signatures |

The server verifies the signature **before** it decrypts, and it **refuses to
start** without the operator public key — authentication is not optional.

---

## One-time setup

### 1. Build the CLI

```bash
cd ladd-admin && go build -o ladd-upload ./cmd/ladd-upload
```

### 2. Generate all key material (no external tools needed)

```bash
./ladd-upload keygen --out ./keys
```

This writes four files and prints exactly where each goes:

| File | Where it goes |
|---|---|
| `operator-signing.key` | **keep on your machine** → `upload --sign-key-file` |
| `server-recipient.pub` | **keep on your machine** → `upload --recipient-file` |
| `operator-signing.pub` | into the cluster Secret as `operator.pub` |
| `server-identity.txt` | into the cluster Secret as `identity.txt` |

Guard `operator-signing.key` and `server-identity.txt` — they are the private
keys. (They're written `0600`.)

### 3. Create the key Secret in the cluster

Both keys live in one Secret, mounted at `/keys`:

```bash
kubectl create secret generic ladd-admin-keys -n nas \
  --from-file=identity.txt=./keys/server-identity.txt \
  --from-file=operator.pub=./keys/operator-signing.pub
```

On the server, prefix with `sudo k3s`.

### 4. Deploy the service

```bash
./deploy/deploy.sh ladd-admin        # server; local k3d: docker build + k3d import + apply -k
kubectl get pods -n nas -l app=ladd-admin      # want Running 1/1
kubectl logs -n nas deploy/ladd-admin          # "listening on :8092"
```

> The `ladd` Secret must already exist (from your normal deploy) — the server
> updates it, it does not create it.

---

## Uploading a LADD file

```bash
kubectl port-forward -n nas svc/ladd-admin 8092:8092    # one terminal

./ladd-upload upload \
  --file LADD_Industry_Filter_CUI_SP_PRVCY_20260811.txt \
  --recipient-file ./keys/server-recipient.pub \
  --sign-key-file ./keys/operator-signing.key \
  --url http://localhost:8092/upload
```

**Success:**

```
uploaded LADD_Industry_Filter_CUI_SP_PRVCY_20260811.txt: 70655 entries
```

The server rejects (and never touches the Secret) if the signature is invalid
(`401`), the ciphertext won't decrypt (`400`), or the file is malformed / empty /
stale / future-dated / misnamed (`422`) — the fail-closed guarantee.

---

## Verifying it took effect

The server updated the `ladd` Secret immediately, but the filter re-reads on its
`LADD_CHECK_EVERY` interval (default 1h). To confirm **now**:

```bash
kubectl rollout restart deploy/filter -n nas
kubectl logs -n nas deploy/filter --tail=5      # "LADD list loaded: N entries, effective <date>"
```

---

## Weekly update (steady state)

```bash
kubectl port-forward -n nas svc/ladd-admin 8092:8092
./ladd-upload upload --file <new file>.txt \
  --recipient-file ./keys/server-recipient.pub \
  --sign-key-file ./keys/operator-signing.key \
  --url http://localhost:8092/upload
```

---

## Exposing it publicly (upload from anywhere)

Now that uploads are **signature-authenticated**, the endpoint can be safely
exposed. Add a Cloudflare tunnel route to the `ladd-admin` service and point
`--url` at the public hostname:

```yaml
# cloudflared ingress
- hostname: admin.<yourdomain>
  service: http://<node-ip>:<port>   # via a LoadBalancer/NodePort on ladd-admin, or the tunnel → service
```

Until you add that route, the service stays ClusterIP (internal-only) and you
reach it with `port-forward` as above. Security still rests on the signature —
an attacker at the public endpoint cannot produce a valid signature without your
`operator-signing.key`.

---

## Configuration (server env vars — all optional)

| Variable | Default | Meaning |
|---|---|---|
| `LADD_ADMIN_ADDR` | `:8092` | HTTP listen address |
| `LADD_ADMIN_IDENTITY_PATH` | `/keys/identity.txt` | server private key (decrypt) |
| `LADD_OPERATOR_PUBKEY_PATH` | `/keys/operator.pub` | operator public key (verify) |
| `LADD_SECRET_NAME` | `ladd` | Secret to update |
| `LADD_SECRET_NAMESPACE` | `nas` | its namespace |
| `LADD_MAX_AGE` | `216h` | reject files older than this (9 days) |
| `LADD_MAX_UPLOAD_BYTES` | `4194304` | request-body cap (4 MiB) |

---

## Troubleshooting

| Symptom | Likely cause / fix |
|---|---|
| pod `CrashLoopBackOff`, log `load identity` / `operator public key` | `ladd-admin-keys` Secret missing a key — it needs both `identity.txt` and `operator.pub` |
| pod log `get secret … forbidden` | RBAC not applied, or the `ladd` Secret doesn't exist yet |
| CLI `upload rejected (401)` | signature invalid — wrong `--sign-key-file`, or the Secret's `operator.pub` doesn't match it |
| CLI `upload rejected (400)` | `--recipient` doesn't match the server's private key |
| CLI `upload rejected (422)` | the file failed validation (stale/empty/misnamed) — expected fail-closed |
| upload OK but filter unchanged | filter hasn't hit its reload interval — `rollout restart deploy/filter` |
| `connection refused` on `localhost:8092` | the `port-forward` isn't running |

---

## Security model (summary)

- **Confidentiality:** age (X25519 + ChaCha20). Plaintext exists only in-memory
  inside the server pod; the private key never leaves the cluster.
- **Authenticity:** ed25519 signature verified before decryption. Only the
  holder of `operator-signing.key` can produce accepted uploads.
- **Least privilege:** the service's RBAC allows `get`+`update` on the **single**
  `ladd` Secret and nothing else.
- **Fail-closed:** malformed / stale / unsigned uploads are rejected and never
  reach the Secret.

Key rotation: regenerate with `keygen`, update the Secret + your local files.
For zero-downtime rotation you'd support two operator keys at once — a future
enhancement.
