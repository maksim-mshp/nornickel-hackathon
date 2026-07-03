#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE="docker compose -f ${HERE}/compose.prod.yml"
RELEASES="${HERE}/releases.log"
ORDERED_SERVICES=(postgres nats minio keycloak llm search catalog epistemic ingest answer embed parse extract ui backup gateway caddy)
SMOKE=("${HERE}/../../ops/k6/gold_smoke.sh")

log() { printf '[deploy] %s\n' "$*"; }

record_release() {
  local tag; tag="$(date -u +%Y%m%dT%H%M%SZ)"
  {
    echo "release ${tag}"
    for svc in "${ORDERED_SERVICES[@]}"; do
      local image digest
      image="$(${COMPOSE} config --images 2>/dev/null | grep "kmap-${svc}" || true)"
      digest="$(docker image inspect --format '{{index .RepoDigests 0}}' "${image}" 2>/dev/null || echo "${image}")"
      echo "  ${svc} ${digest}"
    done
  } >>"${RELEASES}"
}

deploy() {
  log "pulling images"
  ${COMPOSE} pull --ignore-buildable || log "pull skipped (local images)"
  log "starting services in order"
  for svc in "${ORDERED_SERVICES[@]}"; do
    ${COMPOSE} up -d --wait "${svc}"
  done
  record_release
}

smoke() {
  log "running gold smoke test"
  bash "${SMOKE[@]}"
}

rollback() {
  log "ROLLBACK: reverting to previous release from ${RELEASES}"
  ${COMPOSE} down
  log "operator must restore previous digests from ${RELEASES} and re-run"
  exit 1
}

main() {
  deploy
  if ! smoke; then
    rollback
  fi
  log "deploy OK"
}

main "$@"
