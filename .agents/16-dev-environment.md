# 16. Стандарты кода и dev-окружение

> Команда работает на **Windows и macOS** — все инструменты и скрипты обязаны быть кроссплатформенными (правила — §7).

## 1. Go — стандарты (обязательные)

- **Версия в `go.mod` — строго `go 1.25.0`** (тулчейн на машинах — Go 1.25.x). Пример шапки модуля:
  ```
  module github.com/<org>/kmap
  go 1.25.0
  ```
- **golangci-lint обязателен, с включённым `modernize`** (линтер покрывает правила gopls/`go fix` — модернизация кода до актуальных идиом). Нужна версия ≥ v2.6.0. Эталонный `.golangci.yml` в корне репо:
  ```yaml
  version: "2"
  linters:
    enable:
      - modernize
      - govet
      - staticcheck
      - errcheck
      - revive
      - gocritic
      - gosec
      - sqlclosecheck
      - rowserrcheck
      - unconvert
      - misspell
  issues:
    max-issues-per-linter: 0
    max-same-issues: 0
  ```
  Локально: `golangci-lint run --fix` (modernize применяет автофиксы); в CI — `golangci-lint run` (без `--fix`, расхождение = красный билд).
- Весь Go-бэкенд — **один проект**: точки входа `cmd/<service>/main.go`, bounded contexts — `internal/<service>/{domain,app,ports,adapters}`, общее — `internal/platform` (эталон — [03-architecture.md](03-architecture.md) §9). Домен не импортирует инфраструктуру; контексты не импортируют друг друга (depguard); никаких «всё в main.go».
- Базовые Go-библиотеки платформы: `koanf/v2` — загрузка YAML-конфигурации из `configs/base` + оверлеев `configs/<env>` + `configs/secrets.yml`, выбран вместо самописного merge из-за готовой поддержки provider/parser/strict merge; `chi/v5` — HTTP routing для gateway, health/readiness и будущих REST-хендлеров, выбран как уже зафиксированный в архитектуре лёгкий роутер, совместимый с `net/http`; `slog` — стандартный structured logging без внешней зависимости.
- Прочее: `gofmt`/`goimports` через golangci-lint; ошибки оборачиваются `fmt.Errorf("...: %w", err)`; контексты сквозные; тесты рядом с кодом, интеграционные — testcontainers.
- **Комментарии в коде запрещены — в YAML-файлах тоже** (Go/Python/TS/SQL/YAML): самодокументируемые имена и структура; пояснения — в `.agents/`, не в исходниках.
- **YAML-файлы именуются `.yml`** (не `.yaml`) — везде: configs, compose, CI, Taskfile.

## 2. Python — стандарты (обязательные)

- Python 3.13, менеджер — **uv** (`uv.lock` в репо, `uv sync` для окружения).
- Линт и формат — **ruff** (`ruff check --fix`, `ruff format`), конфиг в `pyproject.toml`; типизация — mypy (strict для `domain`/`app`).
- Та же дисциплина слоёв, что и в Go (зеркалим чистую архитектуру):
  ```
  services/extract/
  ├── src/extract/
  │   ├── domain/        # модели и инварианты (NumericCandidate, правила грамматики)
  │   ├── app/           # use-cases (ExtractDocument, BuildBundle)
  │   ├── ports/         # grpc-сервер, NATS-консьюмер
  │   └── adapters/      # s3, llm-gw клиент, embed-клиент
  ├── tests/             # pytest; golden-тесты numcore
  └── pyproject.toml
  ```
- Никакой бизнес-логики в хендлерах/консьюмерах; промпты — только из `prompts/` (версионированные), не инлайном.

## 3. Docker — политика образов

- **База — alpine, когда это возможно**: все Go-сервисы собираются `CGO_ENABLED=0` → multi-stage: `golang:1.25-alpine` (build) → `alpine:3.22` (runtime, nonroot, `ca-certificates` + `tzdata`).
- Где alpine невозможен (musl ломает ML-колёса): Python-сервисы (parse/extract/embed) — `python:3.13-slim` (bookworm); UI — `node:22-alpine` (build) → standalone-вывод Next.js на `node:22-alpine`.
- Общие правила: multi-stage всегда; nonroot user; HEALTHCHECK; пин версий базовых образов; без кэшей пакетных менеджеров в финальном слое; образы публикуем `linux/amd64` (+`linux/arm64` при необходимости — у части команды Apple Silicon; официальные образы зависимостей postgres/nats/minio — multi-arch).

## 4. Инвентаризация dev-машины владельца (проверено 02.07.2026 — **всё обязательное установлено**)

| Инструмент | Версия | Роль |
|---|---|---|
| go | 1.25.1 | тулчейн (совместим с `go 1.25.0`) |
| golangci-lint | v2.12.2 | линт Go (modernize поддержан) |
| swag | v2.0.0 | генерация OpenAPI 3.1 |
| **gh** | 2.95.0, **авторизован** (maksim-mshp) | GitHub: репо/PR/secrets/Actions |
| **buf** | 1.71.0 | proto: lint, breaking, кодоген |
| protoc-gen-go / -go-grpc | установлены | плагины кодогена |
| **sqlc** | v1.31.1 | типобезопасные PG-запросы |
| **mockgen** | v0.6.0 | моки портов |
| **nats** (natscli) | v0.4.0 | инспекция JetStream, DLQ redrive |
| **grpcurl** | dev build | ручные вызовы gRPC |
| **lefthook** | 1.13.6 | git-хуки (линт + формат коммитов) |
| **gotestsum** | v1.13.0 | читаемый вывод тестов |
| **pnpm** | 10.34.4 | пакеты UI |
| **ruff** | 0.15.20 | линт/формат Python |
| **mypy** | 2.1.0 | типы Python |
| migrate (golang-migrate) | есть | **миграции БД — стандарт проекта** (SQL-first, `NNNN_name.up/.down.sql`, embed через iofs) |
| task (go-task) | 3.48.0 | канонический раннер (`Taskfile.yml`) |
| docker / compose | 29.2.0 | контуры demo |
| node / npm | 22.12 | UI |
| python / uv / pytest | 3.13.1 / 0.9.3 | ML-сервисы |
| psql | 18 | клиент PostgreSQL |
| dlv | есть | отладчик Go |
| protoc | 25.7 | резерв (основной кодоген — buf) |
| kubectl | есть | прод-фаза |
| git, openssl, curl | есть | базовые |

Диск C: ~38 ГБ свободно (02.07.2026). Docker Desktop (WSL2 vhdx) живёт на C:; полный dev-стек — ~8–10 ГБ образов + кэш сборки buildx (растёт!). Гигиена: периодически `docker system prune` и `docker builder prune`; если C: станет тесно — перенести Docker-диск на D: (Docker Desktop → Settings → Resources → Disk image location).
| jq / yq | 1.8.2 / v4.53.3 | JSON/YAML в скриптах |
| oasdiff | есть | REST breaking-check локально |
| air | есть | hot-reload Go |
| mc (MinIO client) | RELEASE.2025-08-13 | инспекция бакетов |

### Осталось поставить (опционально, по мере надобности)

| Инструмент | Windows | macOS | Когда |
|---|---|---|---|
| k6 | `winget install k6 --source winget` | `brew install k6` | нагрузочные (пре-прод) |
| trivy / syft | choco/scoop | `brew install trivy syft` | сканы — обязательны в CI, локально опционально |
| helm / kubectl | — | — | только если вернёмся к k8s (сейчас деплой — Docker на одной VM, см. [12-deployment.md](12-deployment.md)) |

## 5. Онбординг разработчика (Windows и macOS)

1. Базовые рантаймы: Go 1.25.x, Python 3.13 + uv, Node 22, Docker Desktop, git.
   - Windows: `winget install GoLang.Go Python.Python.3.13 OpenJS.NodeJS.LTS Docker.DockerDesktop Git.Git GitHub.cli Task.Task`
   - macOS: `brew install go python@3.13 node@22 git gh go-task && brew install --cask docker`
2. Кроссплатформенные (одинаково на обеих ОС):
   ```
   go install github.com/bufbuild/buf/cmd/buf@latest \
     google.golang.org/protobuf/cmd/protoc-gen-go@latest \
     google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest \
     github.com/sqlc-dev/sqlc/cmd/sqlc@latest \
     go.uber.org/mock/mockgen@latest \
     github.com/nats-io/natscli/nats@latest \
     github.com/fullstorydev/grpcurl/cmd/grpcurl@latest \
     github.com/evilmartians/lefthook@latest \
     gotest.tools/gotestsum@latest \
     github.com/golang-migrate/migrate/v4/cmd/migrate@latest   # с тегами: -tags 'postgres'
   uv tool install ruff && uv tool install mypy
   npm install -g pnpm
   ```
   golangci-lint — по [официальной инструкции](https://golangci-lint.run/docs/welcome/install/) (brew/бинарь; версия ≥2.6). swag: `go install github.com/swaggo/swag/v2/cmd/swag@latest`.
   Убедиться, что `$(go env GOPATH)/bin` и `~/.local/bin` в PATH.
3. `gh auth login` → `task tools` (доустановка) → `lefthook install` (хуки) → `task dev`.

## 6. gh — рабочий процесс

- Репо: `gh repo create <org>/kmap --private --source . --push`.
- PR: `gh pr create --fill`, ревью: `gh pr view --web`, `gh pr checks`, merge: `gh pr merge --squash`.
- Secrets CI: `gh secret set LLM_API_KEY`; статусы Actions: `gh run list`, `gh run watch`.
- Релизы демо-дня: `gh release create v0.1.0 --generate-notes`.

## 7. Инструменты агентов

Агентам разрешены **любые CLI-команды** без ограничения по списку (сборка, docker, gh, сеть, диагностика). Для фронтенда доступен **chrome-devtools MCP**: открытие страниц, **скриншоты результата**, чтение консоли и DOM — обязательный цикл визуальной верификации описан в [17-frontend.md](17-frontend.md) §7. Скриншоты и прочие артефакты проверки — в `.tmp/`.

## 8. Кроссплатформенность (Windows + macOS) — правила

1. **Раннер — только Taskfile** (`task <target>`): встроенный shell-интерпретатор go-task работает одинаково на Windows/macOS; **не писать** `.sh`/`.ps1`/Makefile. Сложная логика — `go run ./tools/...` или `python`-скрипт, не шелл.
2. В командах Taskfile — без bash-измов и юникс-путей: пути только относительные и через `/` (go-task нормализует), никаких хардкодов `C:\` или `/home`.
3. Line endings: в корне репо `.gitattributes` с `* text=auto eol=lf` (+ `core.autocrlf=input` в онбординге) — иначе golden-тесты и линтеры расходятся между ОС.
4. Git-хуки — через **lefthook** (Go-бинарь, кроссплатформенный), не сырые `.git/hooks`-скрипты: pre-commit — `golangci-lint run --fix`, `ruff check --fix`; commit-msg — проверка `^(feat|fix|docs|refactor|test|perf|ci|chore): .+`.
5. Всё «тяжёлое» окружение (PG, NATS, MinIO, vLLM) — только в Docker; локально на хост ничего не ставим. На Apple Silicon vLLM/CUDA-профиль недоступен — LLM-upstream = реестр организаторов/API владельца, эмбеддинги в CPU-режиме.
6. Кодоген (buf, swag, sqlc, mockgen) детерминирован и коммитится — разработчику без генераторов достаточно `task dev`; CI проверяет свежесть генерации (`task gen && git diff --exit-code`).
