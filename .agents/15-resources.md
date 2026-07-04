# 15. Ресурсы, доступы, открытые вопросы

Правило для всех агентов и участников: **не блокируйся молча — запрашивай ресурс у владельца проекта явно** (списком: что, зачем, в каком виде). Владелец подтвердил готовность предоставлять ресурсы по запросу.

## 1. Уже подтверждено владельцем

| Ресурс | Статус | Детали |
|---|---|---|
| **LLM (chat/synthesis) — Yandex AI Studio** (`https://ai.api.cloud.yandex.net/v1`, **Responses API**, OpenAI-совместимый) | **выдан и протестирован 03.07** | ключ и `folder_id` — в `configs/secrets.yml` (в .gitignore); схема авторизации `Api-Key`; клиент — официальный `openai-go` SDK; **только open-source модели** (`gpt-oss-20b/120b`, `deepseek-v4-flash`, `qwen3.6-35b-a3b`) — `yandexgpt`/`aliceai` запрещены (не open-weight); модель задаётся как `gpt://<folder_id>/<модель>` |
| **Эмбеддинги/rerank — DigitalOcean Gradient** (`https://inference.do-ai.run/v1`, OpenAI-совместимый) | выдан и протестирован 02.07 | ключ — в `configs/secrets.yml`; `bge-m3` (эмбеддинги) + `bge-reranker-v2-m3` (rerank) — open-weight; Yandex-эмбеддинги проприетарны и не используются |
| **«Реестр моделей» организаторов** | подтверждён в чате капитанов 02.07, выдача — утром 03.07 | что это (inference API / хостинг / каталог весов), какие модели и размеры, OpenAI-совместимость, лимиты; после получения — сравнить с DO по ценам/качеству и решить порядок upstream'ов в `llm-routes.yml` |

### 1.1. Выданные модели (DO Gradient) и цены ($ за 1M токенов, in/out)

| Модель | Цена | Роль у нас |
|---|---|---|
| `openai-gpt-oss-20b` (128k ctx, max_out 4k) | 0.05 / 0.45 | extraction, parse_query, bind_numbers, aliases |
| `openai-gpt-oss-120b` (128k ctx, max_out 4k) | 0.10 / 0.70 | judge, эскалация extraction |
| `deepseek-4-flash` (65k ctx/out) | 0.112 / 0.224 | синтез ответов (дёшев на output, длинный вывод) |
| `mimo-v2.5` (32k) | 0.105 / 0.28 | резерв/альтернатива для дешёвых задач |
| `alibaba-qwen3-32b` (32k, thinking) | 0.25 / 0.55 | альтернативный judge |
| `gemma-4-31B-it` (256k ctx) | 0.18 / 0.50 | literature_review по длинным документам |
| `mistral-3-14B` (262k ctx) | 0.20 / 0.20 | дешёвые батч-задачи |
| `nvidia-nemotron-3-super-120b` (1M ctx) | 0.21 / 0.455 | сверхдлинный контекст (резерв) |
| `qwen3.5-397b-a17b` (131k, thinking) | 0.385 / 2.45 | эскалация judge (спорные случаи) |
| `bge-m3` (embeddings, 1024d) | 0.02 | **эмбеддинги — работает** (`embed.backend: remote`) |
| `bge-reranker-v2-m3` (`/v1/rerank`) | 0.01 | **rerank — работает** (remote) |

Оценка стоимости: полная индексация 5 тыс. документов ≈ 90M токенов ≈ **$10–15** (+эмбеддинги ~$1); один ответ ≈ $0.002–0.01. Бюджет — незначимый.

### 1.2. Результаты проверки ключа v2 (02.07)

- ✅ `/v1/chat/completions`: все 11 чат-моделей отвечают, RU/EN; стриминг работает; у gpt-oss/mimo/qwen `reasoning_content` (kmap-llm отбрасывает его; для дешёвых задач `reasoning_effort: low` — reasoning-токены биллятся как output; thinking-моделям qwen давать запас max_tokens).
- ✅ `/v1/embeddings` (bge-m3, 1024d, батчи, кириллица) и ✅ `/v1/rerank` (bge-reranker-v2-m3, relevance_score) — работают: **remote-режим kmap-embed доступен с первого дня**.
- ✅ `response_format: json_object` — работает на gpt-oss; ⚠️ на deepseek-4-flash недоступен (403) → синтез через prompt-based JSON + schema-валидация с repair.
- ⚠️ **Дисциплина UTF-8**: шлюз DO возвращает обманчивый `403 Forbidden` на битые не-ASCII байты в теле (ловушка ручных curl из Windows-консоли — слать `--data-binary @file.json` в UTF-8). Go/Python клиенты шлют корректный UTF-8 — проблемы нет.
- ❗ Первый ключ (02.07, 6 моделей) **отозван** (401) — везде используется только ключ v2 из `configs/secrets.yml`.
- ❗ В `/v1/models` виден полный каталог DO, включая **проприетарные `openai-gpt-5*`, `anthropic-claude-*` — их использовать НЕЛЬЗЯ** (правила хакатона запрещают OpenAI/Anthropic). В `llm-routes.yml` — жёсткий allowlist только open-weight моделей.

### 1.3. Порядок upstream'ов LLM (fallback Yandex → DigitalOcean)

`configs/base/llm-routes.yml`: `default_provider: yandex`, `fallback_providers: [do_gradient]`. kmap-llm сначала обращается к Yandex AI Studio; при ошибке провайдера (например, `PermissionDenied`, недоступность) тот же запрос идёт в DigitalOcean Gradient — тем же клиентом `openai-go` (Responses API, проверено: DO его поддерживает). Провайдер без `base_url`/`api_key` в цепочке пропускается, поэтому стек работает и когда ключ Yandex не выдан и остаётся только DO. Канонические слаги (allowlist/tasks — яндексовые) переводятся в имена моделей DO через `providers.do_gradient.models`:

| Канонический слаг (Yandex) | Модель DigitalOcean |
|---|---|
| `gpt-oss-20b/latest` | `openai-gpt-oss-20b` |
| `gpt-oss-120b/latest` | `openai-gpt-oss-120b` |
| `deepseek-v4-flash/latest` | `deepseek-4-flash` |
| `qwen3.6-35b-a3b/latest` | `alibaba-qwen3-32b` |
| `qwen3-235b-a22b-fp8/latest` | `qwen3.5-397b-a17b` |

Ключ DO для LLM-фолбэка — `llm.providers.do_gradient.api_key` в `configs/secrets.yml` (тот же ключ, что и для эмбеддингов/rerank).

## 2. Запросить у владельца (по мере необходимости)

| # | Ресурс | Зачем | Когда нужен |
|---|---|---|---|
| R1 | Датасет хакатона (корпус отчётов/статей, каталог экспериментов, справочники, сотрудники/лаборатории, таксономия тегов) — складывается в `data-sources/` (в .gitignore) | ingestion, gold-set v1, калибровка numcore | выдача доступов — утром 03.07 (подтверждено в чате капитанов) |
| ~~R2~~ | ~~GPU-машина~~ | **не требуется** (решено 02.07): LLM+embeddings+rerank полностью закрыты DO API, парсинг Docling работает на CPU. GPU вернётся в повестку только при сценарии «полный офлайн» (запрет внешних API → vLLM on-prem) | — |
| R3 | Хост/VM для демо (8 vCPU/32 ГБ) либо договорённость «демо с ноутбука» | стабильный показ жюри | день 2 |
| R4 | Ключ routerai.ru (резервный LLM-провайдер, разрешён организаторами) | failover, если API владельца недоступен с площадки | опционально |
| ~~R7~~ | ~~Включить embeddings и rerank~~ | **✅ закрыт 02.07**: ключ v2 покрывает `/v1/embeddings` и `/v1/rerank` | — |
| ~~R8~~ | ~~Добавить недорогие модели~~ | **✅ закрыт 02.07**: qwen3-32b, gemma-4-31B, mistral-3-14B, nemotron-3-super-120b, qwen3.5-397b добавлены в ключ v2 | — |
| R5 | Git-репозиторий команды (URL, доступы CI) | монорепо, GitHub Actions | до старта |
| R6 | Критерии оценки жюри (если опубликуют) | расстановка акцентов демо | как появятся |

## 3. Открытые вопросы к организаторам (канал: чат капитанов; разборы — открытие и Q&A 03.07)

1. Состав «реестра моделей»: какие модели/размеры, это inference API или хостинг весов, OpenAI-совместимость, лимиты RPS/TPM, доступен ли извне площадки.
2. Объём и состав датасета: сколько документов, есть ли EN-часть, реальные ли ФИО сотрудников (или анонимизированные) — влияет на expert-слой и PII-режим.
3. Формат каталога экспериментов (колонки CSV/XLSX) и справочников — для маппингов ingestion.
4. Формат сдачи решения (ссылка на GitHub / платформа / архив) и требования к воспроизводимости — обещали рассказать на открытии.
5. Как будет проходить демо: своя машина / выделенный стенд / доступ в интернет с площадки (влияет на выбор LLM-upstream).
6. Критерии оценки и вес «экономии ресурсов» — калибровка матрицы моделей.
7. Допустим ли внешний managed-роутер (routerai.ru) с точки зрения ИБ данных датасета, или данные не должны покидать периметр (тогда — только vLLM on-prem / реестр организаторов).

## 4. Технические константы (чтобы не переспрашивать)

- LLM-модели по умолчанию (Yandex AI Studio, только open-source): `gpt-oss-20b/latest` (extraction, parse_query, bind_numbers), `deepseek-v4-flash/latest` (synthesis), `gpt-oss-120b/latest` (judge, эскалации), `qwen3.6-35b-a3b/latest` (aliases); маршрутизация — `configs/base/llm-routes.yml`. Полная матрица — [06-extraction.md](06-extraction.md) §2.1. Фолбэк-провайдер при недоступности Yandex — DigitalOcean (§1.3).
- Эмбеддинги/rerank: bge-m3 (1024d) + bge-reranker-v2-m3, **режим по умолчанию — remote через DO** (`embed.backend: remote`, проверено); локальный режим (torch/onnx-int8) — офлайн-фолбэк.
- Секреты (ключи API) — только в `configs/secrets.yml` (в .gitignore); `.env` в проекте не используется.
- Стор: PostgreSQL 18 + pgvector 0.8; шина: NATS JetStream ≥2.12; объектное: MinIO.
- Временные артефакты (выгрузки eval, дампы, черновики) — только в `.tmp/`.
