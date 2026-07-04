package app

import (
	"context"
	"strings"
	"time"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const taskParseQuery = "parse_query"

const defaultParseTimeout = 35 * time.Second

const parseQuerySystemPrompt = "Ты — разборщик запросов системы знаний R&D горно-металлургической отрасли. " +
	"По вопросу пользователя верни СТРОГО JSON-объект без пояснений: " +
	`{"intent":"...","materials":[],"processes":[],"properties":[],"geography":"any|ru|foreign|compare"}. ` +
	"intent — одно из: technology_search, literature_review, comparison, gap_analysis, expert_search, experiment_search, contradiction_analysis, entity_lookup. " +
	"В materials — материалы и вещества; в processes — процессы, технологии, оборудование; в properties — свойства и параметры. " +
	"Значения — короткие русские термины из вопроса (например: медный штейн, шахтные воды, техногенный гипс, диоксид серы, свинцово-цинковое сырьё, электроэкстракция). Числа не извлекай. " +
	"geography: ru — отечественная практика, foreign — зарубежная/мировая, compare — обе, any — не указана."

type ResolvedEntity struct {
	Slug  string
	Name  string
	Etype string
}

type EntityResolver interface {
	ResolveEntities(ctx context.Context, terms []string) ([]ResolvedEntity, error)
}

type LLMPlanner struct {
	llm      LLMClient
	resolver EntityResolver
	timeout  time.Duration
}

func NewLLMPlanner(llm LLMClient, resolver EntityResolver, timeout time.Duration) *LLMPlanner {
	if timeout <= 0 {
		timeout = defaultParseTimeout
	}
	return &LLMPlanner{llm: llm, resolver: resolver, timeout: timeout}
}

func (planner *LLMPlanner) Plan(ctx context.Context, question string, filters *structpb.Struct) (*kmapv1.QueryPlan, []string, bool) {
	payload, err := parseQueryPayload(question)
	if err != nil {
		return nil, nil, false
	}

	llmCtx, cancel := context.WithTimeout(ctx, planner.timeout)
	defer cancel()
	response, err := planner.llm.Complete(llmCtx, &kmapv1.CompleteRequest{Task: taskParseQuery, Payload: payload})
	if err != nil {
		return nil, nil, false
	}

	parsed := parseLLMPlan(response.GetJson())
	terms := parsed.terms()
	if len(terms) == 0 {
		return nil, nil, false
	}

	entities, err := planner.resolveEntities(ctx, terms)
	if err != nil {
		entities, _ = structpb.NewStruct(map[string]any{})
	}

	geography := detectGeography(question)
	if geography == "any" && validGeography(parsed.geography) {
		geography = parsed.geography
	}

	quality, err := structpb.NewStruct(map[string]any{"parser": "llm", "confidence": 0.85})
	if err != nil {
		return nil, nil, false
	}

	plan := &kmapv1.QueryPlan{
		Schema:           "queryplan/1",
		Intent:           chooseIntent(parsed.intent, question),
		Lang:             detectLang(question),
		Entities:         entities,
		ParamConstraints: extractConstraints(question),
		Geography:        geography,
		Quality:          quality,
	}
	if err := applyFilters(plan, filters); err != nil {
		return nil, nil, false
	}
	return plan, terms, true
}

func (planner *LLMPlanner) resolveEntities(ctx context.Context, terms []string) (*structpb.Struct, error) {
	if planner.resolver == nil {
		return structpb.NewStruct(map[string]any{})
	}
	resolved, err := planner.resolver.ResolveEntities(ctx, terms)
	if err != nil {
		return nil, err
	}
	materials, processes, properties := []any{}, []any{}, []any{}
	seen := map[string]bool{}
	for _, entity := range resolved {
		if entity.Slug == "" || seen[entity.Slug] {
			continue
		}
		seen[entity.Slug] = true
		item := map[string]any{"slug": entity.Slug, "name": entity.Name}
		switch entity.Etype {
		case "material":
			materials = append(materials, item)
		case "property", "parameter", "climate", "economic_indicator":
			properties = append(properties, item)
		case "process", "technology", "equipment", "experiment", "topic":
			processes = append(processes, item)
		}
	}
	return structpb.NewStruct(map[string]any{
		"materials":  materials,
		"processes":  processes,
		"properties": properties,
	})
}

func parseQueryPayload(question string) (*structpb.Struct, error) {
	messages := []any{
		map[string]any{"role": "system", "content": parseQuerySystemPrompt},
		map[string]any{"role": "user", "content": question},
	}
	return structpb.NewStruct(map[string]any{"messages": messages})
}

type llmPlanFields struct {
	intent     string
	geography  string
	materials  []string
	processes  []string
	properties []string
}

func parseLLMPlan(js *structpb.Struct) llmPlanFields {
	out := llmPlanFields{}
	if js == nil {
		return out
	}
	fields := js.GetFields()
	out.intent = strings.TrimSpace(fields["intent"].GetStringValue())
	out.geography = strings.TrimSpace(fields["geography"].GetStringValue())
	out.materials = stringListValue(fields["materials"])
	out.processes = stringListValue(fields["processes"])
	out.properties = stringListValue(fields["properties"])
	return out
}

func stringListValue(value *structpb.Value) []string {
	if value == nil {
		return nil
	}
	list := value.GetListValue()
	if list == nil {
		return nil
	}
	out := make([]string, 0, len(list.GetValues()))
	for _, item := range list.GetValues() {
		text := strings.TrimSpace(item.GetStringValue())
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func (fields llmPlanFields) terms() []string {
	seen := map[string]bool{}
	terms := make([]string, 0, 12)
	for _, group := range [][]string{fields.materials, fields.processes, fields.properties} {
		for _, term := range group {
			key := strings.ToLower(term)
			if seen[key] {
				continue
			}
			seen[key] = true
			terms = append(terms, term)
			if len(terms) >= 12 {
				return terms
			}
		}
	}
	return terms
}

func validGeography(value string) bool {
	switch value {
	case "ru", "foreign", "compare":
		return true
	default:
		return false
	}
}

var validIntents = map[string]bool{
	"technology_search":      true,
	"literature_review":      true,
	"comparison":             true,
	"gap_analysis":           true,
	"expert_search":          true,
	"experiment_search":      true,
	"contradiction_analysis": true,
	"entity_lookup":          true,
}

func chooseIntent(llmIntent string, question string) string {
	lower := strings.ToLower(question)
	switch {
	case containsAny(lower, "кто ", "работал", "эксперт", "компетенц", "специалист", "who ", "expert", "specialist"):
		return "expert_search"
	case containsAny(lower, "есть ли данные", "пробел", "не изучен", "белые пятна", "gap", "unexplored"):
		return "gap_analysis"
	}
	if validIntents[llmIntent] {
		return llmIntent
	}
	return "technology_search"
}
