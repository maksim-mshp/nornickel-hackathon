package app

import (
	"strings"
	"unicode"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type ref struct {
	slug string
	name string
}

type route struct {
	intent     string
	materials  []ref
	processes  []ref
	properties []ref
}

func buildPlan(question string) (*kmapv1.QueryPlan, error) {
	selected := routeQuestion(question)
	entities, err := entitiesStruct(selected)
	if err != nil {
		return nil, err
	}
	quality, err := structpb.NewStruct(map[string]any{
		"parser":     "rules",
		"confidence": 0.9,
	})
	if err != nil {
		return nil, err
	}

	return &kmapv1.QueryPlan{
		Schema:           "queryplan/1",
		Intent:           selected.intent,
		Lang:             detectLang(question),
		Entities:         entities,
		ParamConstraints: extractConstraints(question),
		Geography:        detectGeography(question),
		Quality:          quality,
	}, nil
}

func routeQuestion(question string) route {
	lower := strings.ToLower(question)
	switch {
	case containsAny(lower, "католит", "циркуляц", "электроэкстракц", "electrowinning", "диафрагмен"):
		intent := "technology_search"
		if containsAny(lower, "кто ", "эксперт", "работал", "компетенц") {
			intent = "expert_search"
		}
		return route{
			intent:     intent,
			materials:  []ref{{"material:catholyte", "католит"}},
			processes:  []ref{{"process:nickel-electrowinning", "электроэкстракция никеля"}},
			properties: []ref{{"parameter:catholyte-flow-rate", "скорость потока"}},
		}
	case containsAny(lower, "обессолива", "сухой остаток", "сульфат", "обогатительн", "хлорид", "минерализац"):
		return route{
			intent:     "technology_search",
			materials:  []ref{{"material:sulfates", "сульфаты"}, {"material:chlorides", "хлориды"}},
			processes:  []ref{{"process:desalination", "обессоливание воды"}},
			properties: []ref{{"property:tds", "сухой остаток"}},
		}
	case containsAny(lower, "кучн", "выщелачив", "холодном климат", "холодный климат", "заполярь"):
		return route{
			intent:     "gap_analysis",
			materials:  []ref{{"material:nickel-ore", "никелевая руда"}},
			processes:  []ref{{"process:heap-leaching", "кучное выщелачивание"}},
			properties: []ref{{"climate:cold", "холодный климат"}},
		}
	default:
		return route{
			intent:     "technology_search",
			materials:  []ref{{"material:catholyte", "католит"}},
			processes:  []ref{{"process:nickel-electrowinning", "электроэкстракция никеля"}},
			properties: []ref{{"parameter:catholyte-flow-rate", "скорость потока"}},
		}
	}
}

func entitiesStruct(selected route) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]any{
		"materials":  refsToList(selected.materials),
		"processes":  refsToList(selected.processes),
		"properties": refsToList(selected.properties),
	})
}

func refsToList(refs []ref) []any {
	list := make([]any, 0, len(refs))
	for _, item := range refs {
		list = append(list, map[string]any{"slug": item.slug, "name": item.name})
	}
	return list
}

type constraintRule struct {
	keywords  []string
	parameter string
	name      string
}

var constraintRules = []constraintRule{
	{keywords: []string{"сухой остаток", "сухого остатка", "tds"}, parameter: "property:tds", name: "сухой остаток"},
	{keywords: []string{"сульфат"}, parameter: "parameter:sulfate-concentration", name: "концентрация сульфатов"},
	{keywords: []string{"хлорид"}, parameter: "parameter:chloride-concentration", name: "концентрация хлоридов"},
	{keywords: []string{"скорост"}, parameter: "parameter:catholyte-flow-rate", name: "скорость потока"},
	{keywords: []string{"температ"}, parameter: "parameter:temperature", name: "температура"},
	{keywords: []string{"плотност тока", "плотности тока"}, parameter: "parameter:current-density", name: "плотность тока"},
}

func extractConstraints(question string) []*kmapv1.ParamConstraint {
	lower := strings.ToLower(question)
	segments := splitClauses(lower)
	var constraints []*kmapv1.ParamConstraint
	seen := map[string]bool{}
	for _, rule := range constraintRules {
		for _, segment := range segments {
			if !containsAny(segment, rule.keywords...) {
				continue
			}
			constraint := parseConstraint(segment, rule)
			if constraint == nil || seen[rule.parameter] {
				continue
			}
			seen[rule.parameter] = true
			constraints = append(constraints, constraint)
			break
		}
	}
	return constraints
}

func parseConstraint(segment string, rule constraintRule) *kmapv1.ParamConstraint {
	numbers := numericLiterals(segment)
	if len(numbers) == 0 {
		return nil
	}
	unit := detectUnit(segment)
	op, hasUpper, hasLower := detectOperator(segment)

	constraint := &kmapv1.ParamConstraint{Parameter: rule.parameter, Unit: unit}
	switch {
	case op == "range" && len(numbers) >= 2:
		constraint.Op = "range"
		constraint.Vmin = numbers[0]
		constraint.Vmax = numbers[1]
	case hasUpper:
		constraint.Op = "lte"
		constraint.Vmax = numbers[len(numbers)-1]
	case hasLower:
		constraint.Op = "gte"
		constraint.Vmin = numbers[0]
	case len(numbers) >= 2:
		constraint.Op = "range"
		constraint.Vmin = numbers[0]
		constraint.Vmax = numbers[1]
	default:
		constraint.Op = "eq"
		constraint.Vmin = numbers[0]
		constraint.Vmax = numbers[0]
	}
	applySI(constraint)
	return constraint
}

type unitSI struct {
	factor float64
	offset float64
	siUnit string
}

var querySIUnits = map[string]unitSI{
	"м/с":    {1, 0, "m/s"},
	"m/s":    {1, 0, "m/s"},
	"°c":     {1, 273.15, "K"},
	"мг/дм³": {1e-3, 0, "kg/m^3"},
	"мг/дм3": {1e-3, 0, "kg/m^3"},
	"мг/л":   {1e-3, 0, "kg/m^3"},
	"mg/l":   {1e-3, 0, "kg/m^3"},
	"%":      {1, 0, "ratio"},
	"а/м²":   {1, 0, "A/m^2"},
	"мпа":    {1e6, 0, "Pa"},
}

func applySI(constraint *kmapv1.ParamConstraint) {
	unit, ok := querySIUnits[strings.ToLower(constraint.Unit)]
	if !ok {
		return
	}
	constraint.SiUnit = unit.siUnit
	switch constraint.Op {
	case "range", "eq":
		constraint.VminSi = constraint.Vmin*unit.factor + unit.offset
		constraint.VmaxSi = constraint.Vmax*unit.factor + unit.offset
	case "lte":
		constraint.VmaxSi = constraint.Vmax*unit.factor + unit.offset
	case "gte":
		constraint.VminSi = constraint.Vmin*unit.factor + unit.offset
	}
}

func detectOperator(segment string) (op string, hasUpper bool, hasLower bool) {
	if containsAny(segment, "≤", "не более", "не выше", "до ", "<") {
		hasUpper = true
	}
	if containsAny(segment, "≥", "не менее", "не ниже", "от ", "свыше", "выше", ">") {
		hasLower = true
	}
	if containsAny(segment, "–", "—", "…", " до ") {
		op = "range"
	}
	return op, hasUpper, hasLower
}

func detectUnit(segment string) string {
	units := []string{"мг/дм³", "мг/дм3", "мг/л", "г/л", "м/с", "°c", "а/м²", "а/дм²", "мпа", "%"}
	for _, unit := range units {
		if strings.Contains(segment, unit) {
			return strings.ReplaceAll(unit, "°c", "°C")
		}
	}
	return ""
}

func splitClauses(text string) []string {
	replacer := strings.NewReplacer(",", "|", ";", "|", ".", "|", " а ", "|", " и ", "|")
	parts := strings.Split(replacer.Replace(text), "|")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			segments = append(segments, trimmed)
		}
	}
	return segments
}

func detectLang(question string) string {
	for _, symbol := range question {
		if unicode.Is(unicode.Cyrillic, symbol) {
			return "ru"
		}
	}
	return "en"
}

func detectGeography(question string) string {
	lower := strings.ToLower(question)
	hasRu := containsAny(lower, "росси", "отечествен", "заполярь")
	hasForeign := containsAny(lower, "зарубеж", "мировой практик", "за рубеж", "world")
	switch {
	case hasRu && hasForeign:
		return "compare"
	case hasRu:
		return "ru"
	case hasForeign:
		return "foreign"
	default:
		return "any"
	}
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
