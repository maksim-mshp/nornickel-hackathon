package app

import (
	"strings"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
)

func selectScenario(plan *kmapv1.QueryPlan) scenario {
	slugs := planSlugs(plan)
	switch {
	case matchesAny(slugs, "process:nickel-electrowinning", "material:catholyte", "parameter:catholyte-flow-rate"):
		return catholyteScenario()
	case matchesAny(slugs, "process:desalination", "material:sulfates", "material:chlorides", "property:tds"):
		return desalinationScenario()
	case matchesAny(slugs, "process:heap-leaching", "material:nickel-ore", "climate:cold"):
		return coldLeachScenario()
	default:
		return catholyteScenario()
	}
}

func planSlugs(plan *kmapv1.QueryPlan) []string {
	if plan == nil {
		return nil
	}
	var slugs []string
	entities := plan.GetEntities()
	if entities == nil {
		return slugs
	}
	for _, group := range []string{"materials", "processes", "properties"} {
		list := entities.GetFields()[group].GetListValue()
		if list == nil {
			continue
		}
		for _, item := range list.GetValues() {
			slug := item.GetStructValue().GetFields()["slug"].GetStringValue()
			if slug != "" {
				slugs = append(slugs, slug)
			}
		}
	}
	return slugs
}

func matchesAny(slugs []string, targets ...string) bool {
	for _, slug := range slugs {
		for _, target := range targets {
			if strings.EqualFold(slug, target) {
				return true
			}
		}
	}
	return false
}

func catholyteScenario() scenario {
	base := scoreComponents{Match: 1.0, Rerank: 0.72, Source: 1.0, Validation: 0.9, Freshness: 0.82}
	process := entityRef{Slug: "process:nickel-electrowinning", Name: "электроэкстракция никеля"}
	flow := entityRef{Slug: "parameter:catholyte-flow-rate", Name: "скорость циркуляции католита"}

	facts := []fact{
		{
			ID: "0197f1a0-1", Ref: "F1", Subject: process, Parameter: flow,
			Value:      numericValue{Operator: "range", Vmin: ptr(0.8), Vmax: ptr(1.0), Unit: "м/с"},
			SI:         numericValue{Operator: "range", Vmin: ptr(0.8), Vmax: ptr(1.0), Unit: "m/s"},
			Conditions: map[string]string{"температура": "60–70 °C", "плотность тока": "220 А/м²"},
			Geography:  "foreign",
			Provenance: provenance{DocumentID: "doc_017", Title: "Отчёт: оптимизация циркуляции католита", DocType: "report", Page: 12, Year: 2023,
				Quote: "при скорости циркуляции католита 0,8–1,0 м/с достигалась максимальная равномерность осаждения и чистота катода"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.98,
			ValidationStatus: "multi_source", Score: 0.91, ScoreComponents: base,
		},
		{
			ID: "0197f1a0-2", Ref: "F2", Subject: process, Parameter: flow,
			Value:      numericValue{Operator: "gt", Vmin: ptr(0.7), Unit: "м/с"},
			SI:         numericValue{Operator: "gt", Vmin: ptr(0.7), Unit: "m/s"},
			Conditions: map[string]string{"температура": "55–60 °C", "плотность тока": "320 А/м²"},
			Geography:  "ru",
			Provenance: provenance{DocumentID: "doc_042", Title: "Протокол опытной серии ЭН-7", DocType: "protocol", Page: 4, Year: 2021,
				Quote: "при скорости потока выше 0,7 м/с наблюдался рост дефектности катодного осадка"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.97,
			ValidationStatus: "contradicted", Score: 0.84, ScoreComponents: scoreComponents{Match: 1.0, Rerank: 0.72, Source: 1.0, Validation: 0.3, Freshness: 0.7},
		},
		{
			ID: "0197f1a0-3", Ref: "F3", Subject: process, Parameter: flow,
			Value:      numericValue{Operator: "eq", Vmin: ptr(0.8), Vmax: ptr(0.8), Unit: "м/с"},
			SI:         numericValue{Operator: "eq", Vmin: ptr(0.8), Vmax: ptr(0.8), Unit: "m/s"},
			Conditions: map[string]string{"температура": "65 °C", "среда": "сульфатная"},
			Geography:  "ru",
			Provenance: provenance{DocumentID: "doc_058", Title: "Отчёт: режимы диафрагменных ячеек", DocType: "report", Page: 21, Year: 2023,
				Quote: "скорость циркуляции католита составляла 0,8 м/с"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.99,
			ValidationStatus: "expert_validated", Score: 0.93, ScoreComponents: scoreComponents{Match: 1.0, Rerank: 0.72, Source: 1.0, Validation: 1.0, Freshness: 0.82},
		},
		{
			ID: "0197f1a0-4", Ref: "F4", Subject: process, Parameter: flow,
			Value:      numericValue{Operator: "range", Vmin: ptr(0.6), Vmax: ptr(0.9), Unit: "м/с"},
			SI:         numericValue{Operator: "range", Vmin: ptr(0.6), Vmax: ptr(0.9), Unit: "m/s"},
			Conditions: map[string]string{"температура": "60–80 °C"},
			Geography:  "foreign",
			Provenance: provenance{DocumentID: "doc_101", Title: "Nickel electrowinning practice review", DocType: "article", Page: 7, Year: 2022,
				Quote: "catholyte circulation velocities of 0.6–0.9 m/s are typical in modern tankhouses"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.98,
			ValidationStatus: "multi_source", Score: 0.88, ScoreComponents: scoreComponents{Match: 1.0, Rerank: 0.66, Source: 1.0, Validation: 0.9, Freshness: 0.82},
		},
		{
			ID: "0197f1a0-5", Ref: "F5", Subject: process, Parameter: entityRef{Slug: "parameter:temperature", Name: "температура электролита"},
			Value:      numericValue{Operator: "range", Vmin: ptr(60), Vmax: ptr(70), Unit: "°C"},
			SI:         numericValue{Operator: "range", Vmin: ptr(333.15), Vmax: ptr(343.15), Unit: "K"},
			Conditions: map[string]string{},
			Geography:  "foreign",
			Provenance: provenance{DocumentID: "doc_017", Title: "Отчёт: оптимизация циркуляции католита", DocType: "report", Page: 9, Year: 2023,
				Quote: "температура электролита поддерживалась в диапазоне 60–70 °C"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.98,
			ValidationStatus: "multi_source", Score: 0.8, ScoreComponents: scoreComponents{Match: 0.6, Rerank: 0.72, Source: 1.0, Validation: 0.9, Freshness: 0.82},
		},
		{
			ID: "0197f1a0-6", Ref: "F6", Subject: entityRef{Slug: "experiment:exp-014", Name: "эксперимент EXP-014"},
			Parameter:  entityRef{Slug: "parameter:cathode-purity-gain", Name: "прирост чистоты катода"},
			Value:      numericValue{Operator: "eq", Vmin: ptr(4.2), Vmax: ptr(4.2), Unit: "%"},
			SI:         numericValue{Operator: "eq", Vmin: ptr(4.2), Vmax: ptr(4.2), Unit: "%"},
			Conditions: map[string]string{"скорость потока": "0.8 м/с", "температура": "65 °C"},
			Geography:  "ru",
			Provenance: provenance{DocumentID: "doc_058", Title: "Отчёт: режимы диафрагменных ячеек", DocType: "report", Page: 23, Year: 2023,
				Quote: "чистота катода повысилась на 4,2 % относительно базового режима"},
			ExtractionMethod: "catalog", ExtractorVersion: "catalog-1.0", Confidence: 0.99,
			ValidationStatus: "expert_validated", Score: 0.86, ScoreComponents: scoreComponents{Match: 0.8, Rerank: 0.72, Source: 1.0, Validation: 0.9, Freshness: 0.82},
		},
	}

	return scenario{
		slug:       "catholyte",
		intent:     "technology_search",
		materials:  []entityRef{{Slug: "material:catholyte", Name: "католит"}},
		processes:  []entityRef{process},
		properties: []entityRef{{Slug: "parameter:catholyte-flow-rate", Name: "скорость потока"}},
		facts:      facts,
		consensus: []consensus{{
			Parameter: flow, Unit: "м/с", Verdict: "majority",
			AgreedMin: 0.8, AgreedMax: 0.9, OverlapIndex: 0.42,
			Sources: []consensusSource{
				{Title: "Отчёт 2023 (doc_017)", Year: 2023, Geography: "foreign", Vmin: 0.8, Vmax: 1.0},
				{Title: "Протокол ЭН-7 (doc_042)", Year: 2021, Geography: "ru", Vmin: 0.3, Vmax: 0.7},
				{Title: "Отчёт 2023 (doc_058)", Year: 2023, Geography: "ru", Vmin: 0.8, Vmax: 0.8},
				{Title: "Review 2022 (doc_101)", Year: 2022, Geography: "foreign", Vmin: 0.6, Vmax: 0.9},
			},
		}},
		contradictions: []contradiction{{
			ID: "ctr-1", AFactRef: "F1", BFactRef: "F2",
			AStatement:  "0,8–1,0 м/с улучшает равномерность осаждения и чистоту катода",
			BStatement:  "выше 0,7 м/с растёт дефектность катодного осадка",
			Cause:       "различие плотности тока: 220 А/м² против 320 А/м²",
			Confounders: []string{"плотность тока", "температура электролита"},
			Status:      "judge_confirmed", Confidence: 0.86,
		}},
		gaps: []gapCell{{
			Label:   "циркуляция католита · плотность тока >350 А/м²",
			Score:   18,
			Reasons: []string{"нет экспериментов", "только зарубежные данные"},
			Neighbors: []string{
				"электроэкстракция меди при высокой плотности тока",
				"электроэкстракция никеля при 250–350 А/м²",
			},
		}},
		experts: []expert{
			{ID: "exp-1", Name: "Иванов И. И.", Lab: "Лаборатория гидрометаллургии", Weight: 0.83, Reports: 7, Experiments: 12, LastYear: 2025},
			{ID: "exp-2", Name: "Петрова А. А.", Lab: "Лаборатория электрохимии", Weight: 0.76, Reports: 5, Experiments: 8, LastYear: 2024},
		},
		stats: evidenceStats{Sources: 4, RuSources: 2, ForeignSources: 2, YearFrom: 2021, YearTo: 2023},
	}
}

func desalinationScenario() scenario {
	base := scoreComponents{Match: 1.0, Rerank: 0.7, Source: 0.95, Validation: 0.85, Freshness: 0.8}
	ro := entityRef{Slug: "technology:reverse-osmosis", Name: "обратный осмос"}
	ie := entityRef{Slug: "technology:ion-exchange", Name: "ионный обмен"}
	tds := entityRef{Slug: "property:tds", Name: "сухой остаток"}

	facts := []fact{
		{
			ID: "0197f2b0-1", Ref: "F1", Subject: ro, Parameter: tds,
			Value:      numericValue{Operator: "lte", Vmax: ptr(500), Unit: "мг/дм³"},
			SI:         numericValue{Operator: "lte", Vmax: ptr(0.5), Unit: "kg/m^3"},
			Conditions: map[string]string{"исходный солесодержание": "1000–1500 мг/дм³"},
			Geography:  "foreign",
			Provenance: provenance{DocumentID: "doc_201", Title: "Обзор технологий обессоливания оборотных вод", DocType: "article", Page: 15, Year: 2022,
				Quote: "обратный осмос обеспечивает сухой остаток не более 500 мг/дм³ на пермеате"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.98,
			ValidationStatus: "multi_source", Score: 0.9, ScoreComponents: base,
		},
		{
			ID: "0197f2b0-2", Ref: "F2", Subject: ie, Parameter: entityRef{Slug: "parameter:sulfate-removal", Name: "степень удаления сульфатов"},
			Value:      numericValue{Operator: "gte", Vmin: ptr(95), Unit: "%"},
			SI:         numericValue{Operator: "gte", Vmin: ptr(95), Unit: "%"},
			Conditions: map[string]string{"сульфаты на входе": "200–300 мг/л"},
			Geography:  "ru",
			Provenance: provenance{DocumentID: "doc_215", Title: "Ионообменная очистка сточных вод обогатительной фабрики", DocType: "report", Page: 8, Year: 2023,
				Quote: "степень удаления сульфат-ионов ионным обменом составляла не менее 95 %"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.97,
			ValidationStatus: "expert_validated", Score: 0.88, ScoreComponents: scoreComponents{Match: 1.0, Rerank: 0.68, Source: 1.0, Validation: 1.0, Freshness: 0.85},
		},
		{
			ID: "0197f2b0-3", Ref: "F3", Subject: ro, Parameter: entityRef{Slug: "parameter:recovery", Name: "коэффициент выхода пермеата"},
			Value:      numericValue{Operator: "range", Vmin: ptr(70), Vmax: ptr(80), Unit: "%"},
			SI:         numericValue{Operator: "range", Vmin: ptr(70), Vmax: ptr(80), Unit: "%"},
			Conditions: map[string]string{"давление": "1.5–2.0 МПа"},
			Geography:  "foreign",
			Provenance: provenance{DocumentID: "doc_201", Title: "Обзор технологий обессоливания оборотных вод", DocType: "article", Page: 17, Year: 2022,
				Quote: "рабочий коэффициент выхода пермеата составляет 70–80 %"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.96,
			ValidationStatus: "multi_source", Score: 0.82, ScoreComponents: scoreComponents{Match: 0.8, Rerank: 0.66, Source: 0.9, Validation: 0.9, Freshness: 0.8},
		},
	}

	return scenario{
		slug:       "desalination",
		intent:     "technology_search",
		materials:  []entityRef{{Slug: "material:sulfates", Name: "сульфаты"}, {Slug: "material:chlorides", Name: "хлориды"}},
		processes:  []entityRef{{Slug: "process:desalination", Name: "обессоливание воды"}},
		properties: []entityRef{tds},
		facts:      facts,
		consensus: []consensus{{
			Parameter: tds, Unit: "мг/дм³", Verdict: "consensus",
			AgreedMin: 300, AgreedMax: 500, OverlapIndex: 0.61,
			Sources: []consensusSource{
				{Title: "Обзор 2022 (doc_201)", Year: 2022, Geography: "foreign", Vmin: 300, Vmax: 500},
				{Title: "Отчёт 2023 (doc_215)", Year: 2023, Geography: "ru", Vmin: 350, Vmax: 500},
			},
		}},
		contradictions: []contradiction{},
		gaps: []gapCell{{
			Label:     "ионный обмен · Mg >300 мг/л",
			Score:     22,
			Reasons:   []string{"нет российских данных", "мало экспериментов"},
			Neighbors: []string{"обратный осмос при высокой минерализации", "нанофильтрация сульфатных вод"},
		}},
		experts: []expert{
			{ID: "exp-3", Name: "Смирнова Е. В.", Lab: "Лаборатория водоподготовки", Weight: 0.79, Reports: 6, Experiments: 9, LastYear: 2024},
		},
		stats: evidenceStats{Sources: 2, RuSources: 1, ForeignSources: 1, YearFrom: 2022, YearTo: 2023},
	}
}

func coldLeachScenario() scenario {
	base := scoreComponents{Match: 0.6, Rerank: 0.55, Source: 0.8, Validation: 0.6, Freshness: 0.6}
	process := entityRef{Slug: "process:heap-leaching", Name: "кучное выщелачивание"}

	facts := []fact{
		{
			ID: "0197f3c0-1", Ref: "F1", Subject: process, Parameter: entityRef{Slug: "parameter:temperature", Name: "температура процесса"},
			Value:      numericValue{Operator: "gte", Vmin: ptr(15), Unit: "°C"},
			SI:         numericValue{Operator: "gte", Vmin: ptr(288.15), Unit: "K"},
			Conditions: map[string]string{"климат": "умеренный"},
			Geography:  "foreign",
			Provenance: provenance{DocumentID: "doc_310", Title: "Heap leaching of nickel laterites", DocType: "article", Page: 4, Year: 2019,
				Quote: "effective bacterial leaching requires temperatures at or above 15 °C"},
			ExtractionMethod: "deterministic", ExtractorVersion: "numcore-1.4.0", Confidence: 0.95,
			ValidationStatus: "weak_evidence", Score: 0.62, ScoreComponents: base,
		},
	}

	return scenario{
		slug:           "coldleach",
		intent:         "gap_analysis",
		materials:      []entityRef{{Slug: "material:nickel-ore", Name: "никелевая руда"}},
		processes:      []entityRef{process},
		properties:     []entityRef{{Slug: "climate:cold", Name: "холодный климат"}},
		facts:          facts,
		consensus:      []consensus{},
		contradictions: []contradiction{},
		gaps: []gapCell{
			{
				Label:     "кучное выщелачивание никеля · холодный климат",
				Score:     6,
				Reasons:   []string{"нет экспериментов", "нет российской практики", "устаревшие источники"},
				Neighbors: []string{"кучное выщелачивание меди в умеренном климате", "чановое выщелачивание никеля в Заполярье"},
			},
		},
		experts: []expert{},
		stats:   evidenceStats{Sources: 1, RuSources: 0, ForeignSources: 1, YearFrom: 2019, YearTo: 2019},
	}
}
