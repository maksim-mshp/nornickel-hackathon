# Runbook: инциденты kmap (single-VM)

Все команды из `/opt/kmap/deploy/vm` (`COMPOSE="docker compose -f compose.prod.yml"`).

## Диагностика
- Состояние: `$COMPOSE ps` — все сервисы должны быть `healthy`.
- Логи сервиса: `$COMPOSE logs --since=15m <service>`.
- Быстрый smoke: `bash /opt/kmap/ops/k6/gold_smoke.sh https://kmap.example.org demo-admin`.

## Сервис unhealthy / перезапускается
1. `$COMPOSE logs --tail=200 <service>` — найти причину.
2. Частые причины: недоступен postgres/nats/minio (проверить их health), исчерпан mem_limit (OOM → поднять лимит в compose.prod.yml), битый конфиг (`configs/prod/<service>.yml`).
3. Точечный рестарт: `$COMPOSE up -d --force-recreate <service>`.

## Зацикленные ошибки parse (NoSuchKey)
- Битый документ помечается `failed` консьюмером catalog (`parse-failed`). Если цикл — проверить, что образы parse+catalog свежие, и что нет registered-события без блоба (см. `#16`).

## LLM upstream недоступен
- Ответы деградируют в экстрактивный режим (guard-пломба amber). Проверить `configs/secrets.yml` (Yandex ключ+folder), доступность `ai.api.cloud.yandex.net`. failover переключит на fallback_model автоматически.

## Откат релиза
1. `cat deploy/vm/releases.log` — предыдущие digest'ы.
2. Прописать предыдущие digest'ы в `compose.prod.yml` (или `docker tag`), `$COMPOSE up -d --wait`.
3. Проверить `gold_smoke.sh`.

## Восстановление из бэкапа
1. Последний дамп: `mc ls local/kmap-backups` (или off-VM `rclone ls offvm:kmap-backups`).
2. `pg_restore -h postgres -U kmap -d kmap -c <dump>`.
3. Ежемесячная repetiция восстановления — `task restore-drill` на dev-машине.

## Эскалация
- Критично (сервис down >2m, бэкап stale >7h): alertmanager → Telegram/email дежурному.
