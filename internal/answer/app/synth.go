package app

import (
	"regexp"
	"strconv"
	"strings"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

var wordPattern = regexp.MustCompile(`\S+\s*`)

type factView struct {
	ref           string
	subjectSlug   string
	subjectName   string
	sourceTitle   string
	parameterName string
	valueText     string
	geography     string
}

type methodView struct {
	Name          string   `json:"name"`
	Applicability string   `json:"applicability"`
	Citations     []string `json:"citations"`
}

func synthesize(pack *kmapv1.EvidencePack) (summary string, methods []methodView, confidence float64) {
	views := factViews(pack)
	if len(views) == 0 {
		return "По запросу не найдено фактов с числовыми значениями в доступном корпусе.", nil, 0.3
	}

	var paragraphs []string
	paragraphs = append(paragraphs, sourcesDigest(views))
	if consensus := consensusParagraph(pack); consensus != "" {
		paragraphs = append(paragraphs, consensus)
	}
	if contradiction := contradictionParagraph(pack); contradiction != "" {
		paragraphs = append(paragraphs, contradiction)
	}
	if gap := gapParagraph(pack); gap != "" {
		paragraphs = append(paragraphs, gap)
	}

	summary = strings.Join(paragraphs, " ")
	methods = deriveMethods(views)
	confidence = deriveConfidence(pack)
	return summary, methods, confidence
}

var paramDisplay = map[string]string{
	"ratio":               "доля",
	"content":             "содержание",
	"size":                "размер",
	"duration":            "длительность",
	"concentration":       "концентрация",
	"molar concentration": "молярная концентрация",
	"pressure":            "давление",
	"throughput":          "производительность",
	"flow rate":           "расход",
	"volumetric flow":     "объёмный расход",
	"mass flow":           "массовый расход",
	"energy intensity":    "энергозатраты",
	"current density":     "плотность тока",
	"rotation speed":      "скорость вращения",
	"velocity":            "скорость",
	"length":              "длина",
	"temperature":         "температура",
	"acidity":             "кислотность",
	"ph":                  "кислотность",
	"cost":                "стоимость",
	"mass fraction":       "массовая доля",
}

func paramLabel(name string) string {
	if label, ok := paramDisplay[strings.ToLower(name)]; ok {
		return label
	}
	return name
}

var docExtensions = []string{".docx", ".pptx", ".xlsx", ".pdf", ".xls", ".doc", ".txt"}

func shortTitle(title string) string {
	title = strings.TrimSpace(title)
	for _, ext := range docExtensions {
		title = strings.TrimSuffix(title, ext)
	}
	runes := []rune(strings.TrimSpace(title))
	if len(runes) > 70 {
		return string(runes[:70]) + "…"
	}
	return string(runes)
}

func sourcesDigest(views []factView) string {
	groups := map[string][]factView{}
	var order []string
	for _, view := range views {
		key := view.sourceTitle
		if key == "" {
			key = view.subjectName
		}
		if key == "" {
			continue
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], view)
	}
	if len(order) == 0 {
		return "По запросу найдены числовые факты в источниках."
	}

	var clauses []string
	for _, source := range order {
		var items []string
		seen := map[string]bool{}
		for _, view := range groups[source] {
			item := paramLabel(view.parameterName) + " " + view.valueText + " [" + view.ref + "]"
			if seen[item] {
				continue
			}
			seen[item] = true
			items = append(items, item)
			if len(items) >= 6 {
				break
			}
		}
		clauses = append(clauses, "«"+shortTitle(source)+"» — "+strings.Join(items, ", "))
	}
	return "По данным источников: " + strings.Join(clauses, "; ") + "."
}

func consensusParagraph(pack *kmapv1.EvidencePack) string {
	if len(pack.GetConsensus()) == 0 {
		return ""
	}
	fields := pack.GetConsensus()[0].GetFields()
	parameter := fields["parameter"].GetStructValue().GetFields()["name"].GetStringValue()
	unit := fields["unit"].GetStringValue()
	agreedMin := formatNumber(fields["agreedMin"].GetNumberValue())
	agreedMax := formatNumber(fields["agreedMax"].GetNumberValue())
	verdict := verdictLabel(fields["verdict"].GetStringValue())
	return "Согласованный диапазон параметра «" + parameter + "» — " + agreedMin + "–" + agreedMax + " " + unit +
		" (" + verdict + ")."
}

func contradictionParagraph(pack *kmapv1.EvidencePack) string {
	if len(pack.GetContradictions()) == 0 {
		return ""
	}
	fields := pack.GetContradictions()[0].GetFields()
	if fields["status"].GetStringValue() == "suspected" {
		return ""
	}
	aRef := fields["aFactRef"].GetStringValue()
	bRef := fields["bFactRef"].GetStringValue()
	aStatement := fields["aStatement"].GetStringValue()
	bStatement := fields["bStatement"].GetStringValue()
	cause := fields["cause"].GetStringValue()
	return "Подтверждённое противоречие [" + aRef + "]↔[" + bRef + "]: " + aStatement + " против " + bStatement +
		". Вероятная причина — " + cause + "."
}

func gapParagraph(pack *kmapv1.EvidencePack) string {
	if len(pack.GetGaps()) == 0 {
		return ""
	}
	fields := pack.GetGaps()[0].GetFields()
	label := fields["label"].GetStringValue()
	var reasons []string
	for _, reason := range fields["reasons"].GetListValue().GetValues() {
		reasons = append(reasons, reason.GetStringValue())
	}
	if label == "" {
		return ""
	}
	text := "Пробел в данных: " + label
	if len(reasons) > 0 {
		text += " (" + strings.Join(reasons, ", ") + ")"
	}
	return text + "."
}

func deriveMethods(views []factView) []methodView {
	groups := map[string]*methodView{}
	var order []string
	for _, view := range views {
		if !strings.HasPrefix(view.subjectSlug, "process:") && !strings.HasPrefix(view.subjectSlug, "technology:") {
			continue
		}
		method, ok := groups[view.subjectSlug]
		if !ok {
			method = &methodView{Name: view.subjectName, Applicability: applicabilityLabel(view.geography)}
			groups[view.subjectSlug] = method
			order = append(order, view.subjectSlug)
		}
		method.Citations = append(method.Citations, view.ref)
	}

	methods := make([]methodView, 0, len(order))
	for _, slug := range order {
		methods = append(methods, *groups[slug])
	}
	return methods
}

func deriveConfidence(pack *kmapv1.EvidencePack) float64 {
	confidence := 0.7
	if len(pack.GetConsensus()) > 0 {
		confidence += 0.15
	}
	if len(pack.GetContradictions()) > 0 {
		confidence -= 0.1
	}
	if confidence < 0.3 {
		confidence = 0.3
	}
	if confidence > 0.95 {
		confidence = 0.95
	}
	return confidence
}

const summaryFactLimit = 12

func factViews(pack *kmapv1.EvidencePack) []factView {
	facts := pack.GetFacts()
	if len(facts) > summaryFactLimit {
		facts = facts[:summaryFactLimit]
	}
	views := make([]factView, 0, len(facts))
	for _, item := range facts {
		fields := item.GetPayload().GetFields()
		subject := fields["subject"].GetStructValue().GetFields()
		parameter := fields["parameter"].GetStructValue().GetFields()
		provenance := fields["provenance"].GetStructValue().GetFields()
		views = append(views, factView{
			ref:           fields["ref"].GetStringValue(),
			subjectSlug:   subject["slug"].GetStringValue(),
			subjectName:   subject["name"].GetStringValue(),
			sourceTitle:   provenance["title"].GetStringValue(),
			parameterName: parameter["name"].GetStringValue(),
			valueText:     formatValue(fields["value"].GetStructValue()),
			geography:     fields["geography"].GetStringValue(),
		})
	}
	return views
}

func formatValue(value *structpb.Struct) string {
	fields := value.GetFields()
	operator := fields["operator"].GetStringValue()
	unit := fields["unit"].GetStringValue()
	vmin, hasMin := numberField(fields, "vmin")
	vmax, hasMax := numberField(fields, "vmax")

	suffix := ""
	if unit != "" {
		suffix = " " + unit
	}

	switch operator {
	case "range":
		return formatNumber(vmin) + "–" + formatNumber(vmax) + suffix
	case "lte":
		return "≤" + formatNumber(vmax) + suffix
	case "lt":
		return "<" + formatNumber(vmax) + suffix
	case "gte":
		return "≥" + formatNumber(vmin) + suffix
	case "gt":
		return ">" + formatNumber(vmin) + suffix
	case "from":
		return "от " + formatNumber(vmin) + suffix
	case "to":
		return "до " + formatNumber(vmax) + suffix
	case "approx":
		return "≈" + formatNumber(vmin) + suffix
	default:
		if hasMin {
			return formatNumber(vmin) + suffix
		}
		if hasMax {
			return formatNumber(vmax) + suffix
		}
		return strings.TrimSpace(suffix)
	}
}

func numberField(fields map[string]*structpb.Value, key string) (float64, bool) {
	value, ok := fields[key]
	if !ok {
		return 0, false
	}
	if _, isNumber := value.GetKind().(*structpb.Value_NumberValue); !isNumber {
		return 0, false
	}
	return value.GetNumberValue(), true
}

func formatNumber(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	return strings.ReplaceAll(text, ".", ",")
}

func verdictLabel(verdict string) string {
	switch verdict {
	case "consensus":
		return "консенсус"
	case "majority":
		return "большинство источников"
	case "split":
		return "расхождение источников"
	default:
		return "недостаточно данных"
	}
}

func applicabilityLabel(geography string) string {
	switch geography {
	case "ru":
		return "отечественная практика"
	case "foreign":
		return "зарубежная практика"
	default:
		return "подтверждено источниками"
	}
}

func chunkDeltas(summary string, wordsPerChunk int) []string {
	words := wordPattern.FindAllString(summary, -1)
	if len(words) == 0 {
		return []string{summary}
	}
	var chunks []string
	for start := 0; start < len(words); start += wordsPerChunk {
		end := min(start+wordsPerChunk, len(words))
		chunks = append(chunks, strings.Join(words[start:end], ""))
	}
	return chunks
}
