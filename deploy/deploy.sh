#!/usr/bin/env bash
# Manual deploy — run this ON THE SERVER (k3s + docker required).
#
#   ./deploy/deploy.sh              # pull, build ALL images, import, apply, roll out
#   ./deploy/deploy.sh api web      # only rebuild/roll out these services (faster)
#   SKIP_PULL=1 ./deploy/deploy.sh  # skip git pull (deploy current working tree)
#   NO_CACHE=0 ./deploy/deploy.sh   # use Docker's build cache (faster); default
#                                     forces a full --no-cache rebuild of every image
# What it does:
#   1. git pull (latest manifests + source)
#   2. docker build each selected service
#   3. import each image into k3s's containerd (so IfNotPresent finds it)
#   4. kubectl apply -k the prod overlay
#   5. rollout restart the rebuilt services so new pods pick up the new images
#
# Secrets (ladd, swim) are NOT created here — they are uploaded by hand (CUI /
# credentials). The script warns if they're missing.
set -euo pipefail

ALL_SERVICES=(api normalizer cache-writer filter web bridge ladd-admin)
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OVERLAY="deploy/k8s/overlays/prod"
NS=nas

# k3s ships its own kubectl/ctr; use them via sudo so KUBECONFIG isn't needed.
KUBECTL="sudo k3s kubectl"
CTR="sudo k3s ctr"

# Which services to build: args if given, else all.
if [ "$#" -gt 0 ]; then
  SERVICES=("$@")
else
  SERVICES=("${ALL_SERVICES[@]}")
fi

cd "$REPO"
echo "==> repo: $REPO"
echo "==> services: ${SERVICES[*]}"

# 1. latest code + manifests
if [ "${SKIP_PULL:-0}" != "1" ]; then
  echo "==> git pull"
  git pull origin main
fi

# 2 + 3. build each image and import it into k3s's containerd
# NO_CACHE=1 (default) forces a full rebuild of every image, so a deploy always
# reflects the latest source. Set NO_CACHE=0 for faster cached builds once the
# code stabilizes. This only affects image builds — k8s ConfigMaps/Secrets/env
# are untouched.
NO_CACHE="${NO_CACHE:-1}"
build_args=()
if [ "$NO_CACHE" = "1" ]; then
  build_args=(--no-cache)
fi

for svc in "${SERVICES[@]}"; do
  echo "==> build nas-$svc:latest"
  docker build "${build_args[@]}" -t "nas-$svc:latest" "./$svc"
  echo "==> import nas-$svc into k3s"
  docker save "nas-$svc:latest" | $CTR images import -
done

# 4. apply the prod overlay (creates namespace + all resources, or updates them)
echo "==> kubectl apply -k $OVERLAY"
$KUBECTL apply -k "$OVERLAY"

# secret sanity check (manual uploads; filter/bridge crashloop without them)
for s in ladd swim; do
  if ! $KUBECTL get secret "$s" -n "$NS" >/dev/null 2>&1; then
    echo "!!  WARNING: secret '$s' is missing in namespace '$NS'."
    if [ "$s" = "ladd" ]; then
      echo "    kubectl create secret generic ladd -n $NS --from-file=filter/data/ladd/active/<LADD_file>.txt"
    else
      echo "    kubectl create secret generic swim -n $NS --from-env-file=bridge/.env"
    fi
  fi
done

# 5. roll out only the services we rebuilt (new pods pull the fresh local image)
for svc in "${SERVICES[@]}"; do
  echo "==> rollout restart deploy/$svc"
  $KUBECTL rollout restart "deploy/$svc" -n "$NS"
done

echo "==> waiting for rollouts..."
for svc in "${SERVICES[@]}"; do
  $KUBECTL rollout status "deploy/$svc" -n "$NS" --timeout=180s || true
done

echo "==> done. pods:"
$KUBECTL get pods -n "$NS"
