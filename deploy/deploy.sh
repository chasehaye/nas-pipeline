#!/usr/bin/env bash
# Deploy — run ON THE SERVER (k3s + docker). Apps only by default; infra opt-in.
#
#   ./deploy/deploy.sh                 # build + staggered-roll all app services
#   ./deploy/deploy.sh api web         # only these
#   SKIP_PULL=1 ./deploy/deploy.sh     # skip git pull
#   NO_CACHE=1  ./deploy/deploy.sh     # clean rebuild (default: cached)
#   APPLY_INFRA=1 ./deploy/deploy.sh   # also apply Kafka/Redis/Postgres/PV
set -euo pipefail

ALL_SERVICES=(api normalizer cache-writer filter web bridge ladd-admin database-writer)
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INFRA_OVERLAY="deploy/k8s/overlays/prod/infra"
APPS_OVERLAY="deploy/k8s/overlays/prod/apps"
NS=nas

KUBECTL="sudo k3s kubectl"
CTR="sudo k3s ctr"

if [ "$#" -gt 0 ]; then
  SERVICES=("$@")
else
  SERVICES=("${ALL_SERVICES[@]}")
fi

cd "$REPO"
echo "==> repo: $REPO"
echo "==> services: ${SERVICES[*]}"

if [ "${SKIP_PULL:-0}" != "1" ]; then
  echo "==> git pull"
  git pull origin main
fi

$KUBECTL get namespace "$NS" >/dev/null 2>&1 || $KUBECTL create namespace "$NS"

# Infra is long-lived; a routine app deploy must not restart it.
if [ "${APPLY_INFRA:-0}" = "1" ]; then
  echo "==> apply infra: $INFRA_OVERLAY"
  $KUBECTL apply -k "$INFRA_OVERLAY"
fi

# Secrets are created by hand; warn if a required one is missing.
for s in ladd swim postgres; do
  if ! $KUBECTL get secret "$s" -n "$NS" >/dev/null 2>&1; then
    echo "!!  WARNING: secret '$s' is missing in namespace '$NS'."
    if [ "$s" = "ladd" ]; then
      echo "    kubectl create secret generic ladd -n $NS --from-file=filter/data/ladd/active/<LADD_file>.txt"
    elif [ "$s" = "swim" ]; then
      echo "    kubectl create secret generic swim -n $NS --from-env-file=bridge/.env"
    else
      echo "    kubectl create secret generic postgres -n $NS \\"
      echo "      --from-literal=POSTGRES_PASSWORD='<pw>' \\"
      echo "      --from-literal=DATABASE_URL='postgres://naspipeline:<pw>@postgres:5432/naspipeline?sslmode=disable'"
    fi
  fi
done

# Declarative — unchanged resources are a no-op (no restarts).
echo "==> apply apps: $APPS_OVERLAY"
$KUBECTL apply -k "$APPS_OVERLAY"

# Build and roll out one service at a time (staggered).
NO_CACHE="${NO_CACHE:-0}"
build_args=()
if [ "$NO_CACHE" = "1" ]; then
  build_args=(--no-cache)
fi

for svc in "${SERVICES[@]}"; do
  echo "==> [$svc] build + import"
  case "$svc" in
    bridge|web)
      # Self-contained: build with the service dir as context.
      docker build "${build_args[@]}" -t "nas-$svc:latest" "./$svc"
      ;;
    *)
      # Go services import the shared ./platform module, so the build context
      # must be the repo root; select the service's Dockerfile with -f.
      docker build "${build_args[@]}" -t "nas-$svc:latest" -f "./$svc/Dockerfile" .
      ;;
  esac
  docker save "nas-$svc:latest" | $CTR images import -
  echo "==> [$svc] rollout"
  $KUBECTL rollout restart "deploy/$svc" -n "$NS"
  $KUBECTL rollout status "deploy/$svc" -n "$NS" --timeout=180s || true
done

echo "==> done. pods:"
$KUBECTL get pods -n "$NS"
