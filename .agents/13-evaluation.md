# 13. Оценка качества: gold-set, метрики, регрессии

Философия (из выбора решения): на витрину — **достоверные и дифференцирующие** метрики, которые нельзя надуть черри-пикингом. Приоритет: детерминированно измеримые (числа, единицы, guard) > размеченные (retrieval, противоречия) > нарративные («дни → минуты»).

## 1. Gold-set v1 (расширение dev v0 после получения датасета хакатона)

`eval/gold/questions.yaml` — 24–30 вопросов; структура записи:

```yaml
- id: q01_desalination
  category: numeric_multiparam          # numeric_multiparam|comparison|factual_search|
                                        # gap|contradiction|expert|literature_review
  question_ru: >
    Какие методы обессоливания воды подходят для обогатительной фабрики, если исходная
    вода содержит сульфаты, хлориды, Ca, Mg, Na по 200–300 мг/л, а требуемый сухой
    остаток — ≤1000 мг/дм³?
  plan_expected:                        # проверка QueryPlan (парсер)
    intent: technology_search
    param_constraints:
      - {parameter: sulfate_concentration, op: range, vmin: 200, vmax: 300, unit: mg_per_l}
      - {parameter: tds, op: lte, vmax: 1000, unit: mg_per_dm3}
  expected:
    must_contain_methods: [обратный осмос, ионный обмен]
    must_not_contain_methods: [дистилляция]     # если корпус не поддерживает
    numeric_claims:                             # каждое число ответа должно быть из этого множества источников
      - {parameter: tds, op: lte, vmax: 1000, unit: mg_per_dm3, source_doc: doc_017}
  sources:
    relevant_docs: [doc_017, doc_042, doc_101]  # для precision/recall@K
  notes: «метки согласованы двумя разметчиками, конфликты решает третий»
```

Распределение категорий: 8 numeric_multiparam (ядро), 4 comparison (в т.ч. RU vs мир), 4 factual_search (+временной фильтр), 4 gap, 4 contradiction, 4 expert, 2 literature_review. Q1–Q6 из ТЗ входят обязательно.

Разметка извлечения: `eval/gold/numeric_facts.yaml` — ≥150 числовых фактов из ≥20 документов разных типов (протоколы, статьи RU/EN, таблицы); `entities.yaml` — ≥100 сущностей; `relations.yaml` — ≥40 связей; `contradiction_pairs.yaml` — ≥40 пар (положительные/отрицательные, включая «разные условия — не противоречие»).

## 2. Метрики и целевые пороги

### Уровень A — детерминированные (витрина, козырь перед жюри)
| Метрика | Цель | Метод |
|---|---|---|
| numeric_extraction_precision | ≥ 0.98 | numcore vs разметка (значение+единица+оператор) |
| numeric_extraction_recall | ≥ 0.94 | то же |
| unit_normalization_accuracy | ≥ 0.99 | SI-конверсия vs разметка |
| **hallucinated_numbers_rate** | **= 0** | guard-отчёты всех ответов gold-set (архитектурная гарантия) |
| query_plan_numeric_accuracy | = 1.0 | числа/операторы/единицы плана vs plan_expected (numcore-контур) |

### Уровень B — размеченные
| Метрика | Цель | Метод |
|---|---|---|
| retrieval precision@10 / recall@20 | ≥ 0.80 / ≥ 0.85 | evidence vs relevant_docs |
| MRR | ≥ 0.75 | позиция первого релевантного |
| entity extraction F1 | ≥ 0.85 | по типам, macro |
| relation extraction F1 | ≥ 0.75 | |
| contradiction precision (judge_confirmed) | ≥ 0.80 | vs contradiction_pairs |
| contradiction recall | ≥ 0.55 | осознанный трейд-офф (precision-first) |
| expert_discovery top-3 hit-rate | ≥ 0.90 | vs разметка экспертов |
| answer faithfulness (выборочно) | ≥ 0.9 | каждая фраза ответа подтверждена цитатой — ручная проверка 10 ответов |

### Уровень C — продуктовые/системные
| Метрика | Цель |
|---|---|
| time-to-answer p50 / p95 | ≤ 3 c / ≤ 5 c |
| first token p95 | ≤ 1.5 c |
| sources per answer (медиана) | ≥ 5 |
| gap detection: ручная валидность 20 ячеек | ≥ 0.85 |
| стоимость ответа (токены LLM) | отчёт по моделям — аргумент «экономии ресурсов» для жюри |

## 3. Harness (kmap-eval)

- Прогон через публичный API (как реальный клиент): `kmap-eval run --env stage --gold eval/gold --out .tmp/eval/<ts>/`.
- Проверки плана — точное сравнение канонизированных структур; методы ответа — маппинг на канонические слаги (не строковый матч); числа ответа — numcore-парс + сверка с expected.numeric_claims и guard-отчётом.
- Отчёт: markdown + JSON (сводная таблица, диффы к прошлому прогону, примеры провалов со ссылками на трейсы) → артефакт CI; итоги пишутся в `eval.runs`.
- Юнит-контур (без сервисов): numcore golden-тесты — в каждом PR (см. CI).

## 4. Регрессионная политика

- PR не мержится при падении метрик уровня A ниже целей или уровня B более чем на 2 п.п. от базовой линии main.
- Смена промпта/модели = обязательный полный прогон (метка PR `llm-change`).
- Ночной прогон на stage — тренды в Grafana (метрики пишутся в Prometheus pushgateway).
- A/B матрица моделей (Qwen3-4B/8B/30B-A3B по задачам) — отчёт «качество vs токены» в `.agents`-приложение перед питчем: прямой ответ на критерий жюри «меньше модель при том же качестве».

## 5. Демо-нарратив метрик (слайд питча)

1. «Извлечение чисел: precision 0.98+, единицы 0.99+ — детерминированное ядро, проверьте на любом документе» (live: показать провенанс до цитаты).
2. «0 галлюцинированных чисел — не обещание, а guard: система физически не выпускает число без источника» (показать плашку guard).
3. «Противоречия: precision 0.8+ на размеченных парах; различие контекста ≠ конфликт» (карточка судьи с конфаундером).
4. «Ответ 3–5 с; evidence — раньше текста» (live-замер).
5. Сравнение с plain RAG (E0-бейзлайн на том же корпусе): точность чисел 82% → 100% guarded; фильтры/сравнения/эксперты — есть/нет.
