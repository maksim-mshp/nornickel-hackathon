#!/usr/bin/env sh
set -eu

INTERVAL_SECONDS=21600
PGHOST=postgres
PGUSER=kmap
PGDATABASE=kmap
MINIO_ALIAS=local
MINIO_BUCKET=kmap-backups
OFFVM_REMOTE=offvm:kmap-backups

export PGPASSWORD="$(cat /run/secrets/postgres_password)"

log() { printf '[backup] %s\n' "$*"; }

run_once() {
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  dump="/tmp/kmap-${stamp}.dump"
  log "pg_dump -> ${dump}"
  pg_dump -h "${PGHOST}" -U "${PGUSER}" -d "${PGDATABASE}" -Fc -f "${dump}"

  log "upload to MinIO ${MINIO_BUCKET}"
  mc alias set "${MINIO_ALIAS}" http://minio:9000 kmap "$(cat /run/secrets/minio_password)" >/dev/null 2>&1 || true
  mc mb -p "${MINIO_ALIAS}/${MINIO_BUCKET}" >/dev/null 2>&1 || true
  mc cp "${dump}" "${MINIO_ALIAS}/${MINIO_BUCKET}/"

  log "off-VM copy via rclone -> ${OFFVM_REMOTE}"
  rclone --config /run/secrets/offvm_rclone copy "${dump}" "${OFFVM_REMOTE}" || log "off-VM copy FAILED (alert)"

  rm -f "${dump}"
  log "done ${stamp}"
}

while true; do
  run_once || log "backup cycle failed"
  sleep "${INTERVAL_SECONDS}"
done
