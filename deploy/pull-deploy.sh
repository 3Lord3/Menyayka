#!/usr/bin/env bash
# Pull-based deploy: runs on the VPS via the menyayka-deploy systemd timer.
# CI only builds and pushes images to GHCR, so no SSH keys live in GitHub.
#
# Flow: update the checkout -> login to GHCR -> pull images -> if any image
# changed: migrate -> up -d -> probe Caddy -> prune. Re-running with no new
# images is a no-op.
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/menyayka}"
COMPOSE="docker compose -f docker-compose.prod.yml"

cd "$DEPLOY_DIR"

if [ ! -f .env ]; then
  echo "ERROR: $DEPLOY_DIR/.env is missing; copy .env.deploy.example and fill it" >&2
  exit 1
fi

# Read single values from .env without sourcing it: values like
# MAIL_FROM=Menyayka <noreply@example.com> are not valid shell.
env_get() {
  grep -E "^${1}=" .env | tail -n 1 | cut -d= -f2- || true
}

DOCKER_IMAGE_PREFIX="$(env_get DOCKER_IMAGE_PREFIX)"
IMAGE_TAG="$(env_get IMAGE_TAG)"
IMAGE_TAG="${IMAGE_TAG:-latest}"
GHCR_USER="$(env_get GHCR_USER)"
GHCR_TOKEN="$(env_get GHCR_TOKEN)"
HEALTHCHECK_HOST="$(env_get HEALTHCHECK_HOST)"

if [ -z "$DOCKER_IMAGE_PREFIX" ]; then
  echo "ERROR: DOCKER_IMAGE_PREFIX is not set in $DEPLOY_DIR/.env" >&2
  exit 1
fi

# 1. Repo files (compose, Caddyfile, migrations) come from the checkout.
#    The script owns them — keep local settings in .env only.
if [ -d .git ]; then
  git fetch --quiet origin main
  git reset --hard --quiet origin/main
fi

# 2. GHCR login for private packages; skipped when the packages are public.
if [ -n "$GHCR_TOKEN" ]; then
  login_user="${GHCR_USER:-${DOCKER_IMAGE_PREFIX#ghcr.io/}}"
  login_user="${login_user%%/*}"
  printf '%s\n' "$GHCR_TOKEN" | docker login ghcr.io -u "$login_user" --password-stdin
fi

# 3. Pull and compare image IDs: nothing new means nothing to restart.
images_digest() {
  docker image inspect -f '{{.Id}}' \
    "$DOCKER_IMAGE_PREFIX/backend:$IMAGE_TAG" \
    "$DOCKER_IMAGE_PREFIX/frontend:$IMAGE_TAG" 2>/dev/null | paste -sd ':'
}
before="$(images_digest || true)"
$COMPOSE pull --quiet
after="$(images_digest || true)"
if [ -n "$before" ] && [ "$before" = "$after" ]; then
  echo "No new images ($IMAGE_TAG is already deployed)."
  exit 0
fi

# 4. Apply: the DB first (migrate waits for its healthcheck), then the stack.
$COMPOSE up -d db
$COMPOSE run --rm migrate
$COMPOSE up -d --remove-orphans
# Caddy resolves upstreams by container name; recreate it after the backend
# and frontend got new addresses.
$COMPOSE up -d --force-recreate caddy

# 5. Post-deploy probe: Caddy must answer the backend health via the public domain.
if [ -n "$HEALTHCHECK_HOST" ]; then
  curl -fsS --max-time 15 "https://${HEALTHCHECK_HOST}/healthz" | grep -q '"status"'
  echo "Deployed $IMAGE_TAG; https://${HEALTHCHECK_HOST}/healthz is healthy."
else
  echo "Deployed $IMAGE_TAG (HEALTHCHECK_HOST is not set, probe skipped)."
fi

docker image prune -f >/dev/null
