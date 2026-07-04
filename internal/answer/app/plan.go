package app

import (
	"regexp"
	"strings"
	"unicode"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

var decimalSeparator = regexp.MustCompile(`([0-9])[.,]([0-9])`)

type ref struct {
	slug string
	name string
}

type route struct {
	intent     string
	materials  []ref
	processes  []ref
	properties []ref
	matched    bool
}

func buildPlan(question string, filters *structpb.Struct) (*kmapv1.QueryPlan, error) {
	selected := routeQuestion(question)
	entities, err := entitiesStruct(selected)
	if err != nil {
		return nil, err
	}
	quality, err := structpb.NewStruct(map[string]any{
		"parser":     "rules",
		"confidence": routeConfidence(selected),
	})
	if err != nil {
		return nil, err
	}

	geography := detectGeography(question)
	plan := &kmapv1.QueryPlan{
		Schema:           "queryplan/1",
		Intent:           refineIntent(question, selected.intent, geography),
		Lang:             detectLang(question),
		Entities:         entities,
		ParamConstraints: extractConstraints(question),
		Geography:        geography,
		Quality:          quality,
	}
	if err := applyFilters(plan, filters); err != nil {
		return nil, err
	}
	return plan, nil
}

func applyFilters(plan *kmapv1.QueryPlan, filters *structpb.Struct) error {
	if filters == nil {
		return nil
	}
	fields := filters.GetFields()

	if geo, ok := filterString(fields, "geography"); ok && geo != "" {
		plan.Geography = geo
	}

	if params := fields["params"].GetListValue(); params != nil {
		index := map[string]int{}
		for position, constraint := range plan.ParamConstraints {
			index[constraint.GetParameter()] = position
		}
		for _, item := range params.GetValues() {
			constraint := constraintFromFilter(item.GetStructValue())
			if constraint == nil {
				continue
			}
			if position, exists := index[constraint.Parameter]; exists {
				plan.ParamConstraints[position] = constraint
			} else {
				index[constraint.Parameter] = len(plan.ParamConstraints)
				plan.ParamConstraints = append(plan.ParamConstraints, constraint)
			}
		}
	}

	timeRange := map[string]any{}
	if year, ok := filterNumber(fields, "year_from"); ok {
		timeRange["year_from"] = year
	}
	if year, ok := filterNumber(fields, "year_to"); ok {
		timeRange["year_to"] = year
	}
	if len(timeRange) > 0 {
		encoded, err := structpb.NewStruct(timeRange)
		if err != nil {
			return err
		}
		plan.TimeRange = encoded
	}
	return nil
}

func constraintFromFilter(item *structpb.Struct) *kmapv1.ParamConstraint {
	if item == nil {
		return nil
	}
	fields := item.GetFields()
	parameter, _ := filterString(fields, "parameter")
	if parameter == "" {
		return nil
	}
	op, _ := filterString(fields, "op")
	if op == "" {
		op = "eq"
	}
	unit, _ := filterString(fields, "unit")
	value, hasValue := filterNumber(fields, "value")
	vmin, hasMin := filterNumber(fields, "vmin")
	vmax, hasMax := filterNumber(fields, "vmax")
	if !hasMin && hasValue {
		vmin, hasMin = value, true
	}
	if !hasMax && hasValue {
		vmax, hasMax = value, true
	}

	constraint := &kmapv1.ParamConstraint{Parameter: parameter, Op: op, Unit: unit}
	switch op {
	case "range", "pm":
		if !hasMin || !hasMax {
			return nil
		}
		constraint.Op = "range"
		constraint.Vmin, constraint.Vmax = vmin, vmax
	case "lte", "lt", "to":
		if !hasMax {
			return nil
		}
		constraint.Vmax = vmax
	case "gte", "gt", "from":
		if !hasMin {
			return nil
		}
		constraint.Vmin = vmin
	default:
		if !hasMin {
			return nil
		}
		constraint.Vmin, constraint.Vmax = vmin, vmin
	}
	applySI(constraint)
	return constraint
}

func filterString(fields map[string]*structpb.Value, key string) (string, bool) {
	value, ok := fields[key]
	if !ok {
		return "", false
	}
	if str, isString := value.GetKind().(*structpb.Value_StringValue); isString {
		return str.StringValue, true
	}
	return "", false
}

func filterNumber(fields map[string]*structpb.Value, key string) (float64, bool) {
	value, ok := fields[key]
	if !ok {
		return 0, false
	}
	if number, isNumber := value.GetKind().(*structpb.Value_NumberValue); isNumber {
		return number.NumberValue, true
	}
	return 0, false
}

func routeQuestion(question string) route {
	lower := strings.ToLower(question)
	switch {
	case containsAny(lower, "католит", "циркуляц", "электроэкстракц", "electrowinning", "catholyte", "electrowin", "диафрагмен"):
		return route{
			intent:     "technology_search",
			materials:  []ref{{"material:catholyte", "католит"}},
			processes:  []ref{{"process:nickel-electrowinning", "электроэкстракция никеля"}},
			properties: electrowinningParameters(lower),
			matched:    true,
		}
	case containsAny(lower, "обессолива", "сухой остаток", "сульфат", "обогатительн", "хлорид", "минерализац",
		"desalination", "sulfate", "sulphate", "chloride", "reverse osmosis", "ion exchange", "brine"):
		return route{
			intent:     "technology_search",
			materials:  []ref{{"material:sulfates", "сульфаты"}, {"material:chlorides", "хлориды"}},
			processes:  []ref{{"process:desalination", "обессоливание воды"}},
			properties: []ref{{"property:tds", "сухой остаток"}},
			matched:    true,
		}
	case containsAny(lower, "кучн", "выщелачив", "холодном климат", "холодный климат", "заполярь",
		"heap leach", "leaching", "cold climate"):
		return route{
			intent:     "gap_analysis",
			materials:  []ref{{"material:nickel-ore", "никелевая руда"}},
			processes:  []ref{{"process:heap-leaching", "кучное выщелачивание"}},
			properties: []ref{{"climate:cold", "холодный климат"}},
			matched:    true,
		}
	default:
		return route{
			intent:  "technology_search",
			matched: false,
		}
	}
}

func electrowinningParameters(lower string) []ref {
	var params []ref
	if containsAny(lower, "температ", "temperature") {
		params = append(params, ref{"parameter:temperature", "температура электролита"})
	}
	if containsAny(lower, "плотност тока", "плотности тока", "плотность тока", "current density", "current-density") {
		params = append(params, ref{"parameter:current-density", "плотность тока"})
	}
	if containsAny(lower, "скорост", "циркуляц", "flow rate", "flow-rate", "velocity") {
		params = append(params, ref{"parameter:catholyte-flow-rate", "скорость циркуляции католита"})
	}
	if len(params) == 0 {
		params = append(params, ref{"parameter:catholyte-flow-rate", "скорость циркуляции католита"})
	}
	return params
}

func refineIntent(question string, base string, geography string) string {
	lower := strings.ToLower(question)
	switch {
	case containsAny(lower, "кто ", "работал", "эксперт", "компетенц", "специалист", "who ", "expert", "specialist", "competenc"):
		return "expert_search"
	case containsAny(lower, "есть ли данные", "пробел", "не изучен", "белые пятна", "изучал ли",
		"is there data", "any data", "gap", "unexplored", "understudied"):
		return "gap_analysis"
	case geography == "compare":
		return "comparison"
	default:
		return base
	}
}

func routeConfidence(selected route) float64 {
	if selected.matched {
		return 0.9
	}
	return 0.4
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
	{keywords: []string{"сухой остаток", "сухого остатка", "tds", "dry residue"}, parameter: "property:tds", name: "сухой остаток"},
	{keywords: []string{"сульфат", "sulfate", "sulphate"}, parameter: "parameter:sulfate-concentration", name: "концентрация сульфатов"},
	{keywords: []string{"хлорид", "chloride"}, parameter: "parameter:chloride-concentration", name: "концентрация хлоридов"},
	{keywords: []string{"скорост", "flow rate", "velocity"}, parameter: "parameter:catholyte-flow-rate", name: "скорость потока"},
	{keywords: []string{"температ", "temperature"}, parameter: "parameter:temperature", name: "температура"},
	{keywords: []string{"плотност тока", "плотности тока", "current density"}, parameter: "parameter:current-density", name: "плотность тока"},
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
	"mg/dm3": {1e-3, 0, "kg/m^3"},
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
	units := []string{"мг/дм³", "мг/дм3", "мг/л", "mg/dm3", "mg/l", "г/л", "м/с", "m/s", "°c", "а/м²", "а/дм²", "мпа", "%"}
	for _, unit := range units {
		if strings.Contains(segment, unit) {
			return strings.ReplaceAll(unit, "°c", "°C")
		}
	}
	return ""
}

func splitClauses(text string) []string {
	protected := decimalSeparator.ReplaceAllString(text, "$1\x00$2")
	replacer := strings.NewReplacer(",", "|", ";", "|", ".", "|", " а ", "|", " и ", "|")
	parts := strings.Split(replacer.Replace(protected), "|")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(strings.ReplaceAll(part, "\x00", "."))
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
	hasRu := containsAny(lower, "росси", "отечествен", "заполярь", "russia", "domestic")
	hasForeign := containsAny(lower, "зарубеж", "мировой практик", "за рубеж", "world", "foreign", "abroad", "international", "global")
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
