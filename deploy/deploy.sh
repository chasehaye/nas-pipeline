#!/usr/bin/env bash
# Deploy — run ON THE SERVER (k3s + docker). Apps only by default; infra opt-in.
#
#   ./deploy/deploy.sh                 # build + staggered-roll all app services
#   ./deploy/deploy.sh api filter      # only the named services
#   NO_CACHE=1  ./deploy/deploy.sh     # clean rebuild (default: cached)
#   APPLY_INFRA=1 ./deploy/deploy.sh   # also apply Kafka/Redis/Postgres/PV
set -euo pipefail

SERVICES=(api normalizer cache-writer filter web bridge ladd-admin database-writer)
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INFRA_OVERLAY="deploy/k8s/overlays/prod/infra"
APPS_OVERLAY="deploy/k8s/overlays/prod/apps"
NS=nas

KUBECTL="sudo k3s kubectl"
CTR="sudo k3s ctr"

# Optional: restrict to the services named as arguments (default: all).
# A literal "all" means every service (CI passes "all" for a full deploy).
if [ "$#" -gt 0 ] && [ "$1" != "all" ]; then
  SERVICES=("$@")
fi

cd "$REPO"
echo "==> repo: $REPO"
echo "==> services: ${SERVICES[*]}"

echo "==> git pull"
git pull origin main

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

# Flaky-build-host stopgap: this server intermittently crashes the Go compiler
# mid-build (suspected unstable RAM). Retry a failed build with backoff so a
# transient fault doesn't abort the whole deploy; the growing delay doubles as a
# cooldown if the instability is thermal.
# NOTE: a bridge, not a fix -- stabilize the memory (BIOS XMP/EXPO off, then
# memtest86+). A build that doesn't crash could still be silently bit-flipped.
build_one() {
  local svc="$1"; shift
  docker "$@" && return 0
  for d in 5 15 30 60; do
    echo "!! [$svc] build failed -- retry in ${d}s (suspect flaky build host)"
    sleep "$d"
    docker "$@" && return 0
  done
  echo "!! [$svc] build failed after retries"
  return 1
}

for svc in "${SERVICES[@]}"; do
  echo "==> [$svc] build + import"
  case "$svc" in
    bridge|web)
      # Self-contained: build with the service dir as context.
      build_one "$svc" build "${build_args[@]}" -t "nas-$svc:latest" "./$svc"
      ;;
    *)
      # Go services import the shared ./platform module, so the build context
      # must be the repo root; select the service's Dockerfile with -f.
      build_one "$svc" build "${build_args[@]}" -t "nas-$svc:latest" -f "./$svc/Dockerfile" .
      ;;
  esac
  docker save "nas-$svc:latest" | $CTR images import -
  echo "==> [$svc] rollout"
  $KUBECTL rollout restart "deploy/$svc" -n "$NS"
  $KUBECTL rollout status "deploy/$svc" -n "$NS" --timeout=180s || true
done

echo "==> done. pods:"
$KUBECTL get pods -n "$NS"
