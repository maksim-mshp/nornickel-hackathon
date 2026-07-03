#!/usr/bin/env bash
set -euo pipefail

BASE="${1:-http://localhost:8080}"
TOKEN="${2:-demo-admin}"
QUESTIONS=(
  "Какая скорость циркуляции католита оптимальна при электроэкстракции никеля?"
  "Мировая практика и консенсус по обессоливанию оборотных вод"
  "Эксперименты и публикации по электроэкстракции никеля за 2019–2024"
  "Россия против зарубежной практики электроэкстракции никеля"
  "Где пробелы в данных по кучному выщелачиванию в холодном климате?"
  "Кто в организации работал с электроэкстракцией никеля?"
)

fail=0
for q in "${QUESTIONS[@]}"; do
  code="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${BASE}/v1/ask" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    --max-time 30 \
    -d "{\"question\":\"${q}\"}")"
  if [ "${code}" != "200" ]; then
    echo "SMOKE FAIL (${code}): ${q}"
    fail=1
  else
    echo "SMOKE OK: ${q}"
  fi
done

exit "${fail}"
