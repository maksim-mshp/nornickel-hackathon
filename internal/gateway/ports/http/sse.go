package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type askBody struct {
	Question string          `json:"question"`
	Lang     string          `json:"lang"`
	Filters  json.RawMessage `json:"filters"`
}

func (server *Server) askHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	body, err := readBody(w, r)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}
	var request askBody
	if err := json.Unmarshal(body, &request); err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}
	if strings.TrimSpace(request.Question) == "" {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "question is required")
		return
	}

	filters, err := parseFilters(request.Filters)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	flusher, ok := w.(stdhttp.Flusher)
	if !ok {
		writeProblem(w, r, stdhttp.StatusInternalServerError, "streaming_unsupported", "Streaming unsupported", "")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(stdhttp.StatusOK)

	stream, err := server.answer.Ask(r.Context(), &kmapv1.AskRequest{
		Question:  request.Question,
		Lang:      request.Lang,
		Filters:   filters,
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeSSEProblem(w, flusher, r, err)
		return
	}

	for {
		message, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			writeSSEProblem(w, flusher, r, err)
			return
		}
		if err := writeAskEvent(w, message); err != nil {
			return
		}
		flusher.Flush()
	}
}

func parseFilters(raw json.RawMessage) (*structpb.Struct, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("invalid filters: %w", err)
	}
	if decoded == nil {
		return nil, nil
	}
	filters, err := structpb.NewStruct(decoded)
	if err != nil {
		return nil, fmt.Errorf("invalid filters: %w", err)
	}
	return filters, nil
}

func writeAskEvent(w io.Writer, message *kmapv1.AskResponse) error {
	switch message.GetType() {
	case "plan":
		return writeSSE(w, "plan", mapPlan(message.GetPlan()))
	case "evidence":
		return writeSSE(w, "evidence", mapEvidence(message.GetEvidence()))
	case "answer.delta":
		return writeSSE(w, "answer.delta", map[string]any{"text": message.GetDelta()})
	case "answer.done":
		return writeSSE(w, "answer.done", mapAnswer(message.GetAnswer()))
	default:
		return nil
	}
}

func writeSSE(w io.Writer, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
	return err
}

func writeSSEProblem(w io.Writer, flusher stdhttp.Flusher, r *stdhttp.Request, err error) {
	_ = writeSSE(w, "error", sseProblem(r, err))
	flusher.Flush()
}

func sseProblem(r *stdhttp.Request, err error) problem {
	code := stdhttp.StatusBadGateway
	title := "Upstream error"
	problemType := "upstream_error"
	if st, ok := status.FromError(err); ok {
		code = grpcHTTPStatus(st.Code())
		title = st.Message()
		problemType = st.Code().String()
	}
	return problem{
		Type:      "https://kmap.local/problems/" + problemType,
		Title:     title,
		Status:    code,
		Detail:    err.Error(),
		Instance:  r.URL.Path,
		RequestID: r.Header.Get("X-Request-Id"),
	}
}

func mapPlan(plan *kmapv1.QueryPlan) map[string]any {
	result := map[string]any{
		"intent":           plan.GetIntent(),
		"entities":         structMap(plan.GetEntities()),
		"paramConstraints": mapConstraints(plan.GetParamConstraints()),
		"geography":        defaultString(plan.GetGeography(), "any"),
		"parser":           "rules",
		"confidence":       0.9,
	}
	if quality := plan.GetQuality(); quality != nil {
		fields := quality.GetFields()
		if parser := fields["parser"].GetStringValue(); parser != "" {
			result["parser"] = parser
		}
		if _, ok := fields["confidence"]; ok {
			result["confidence"] = fields["confidence"].GetNumberValue()
		}
	}
	return result
}

func mapConstraints(constraints []*kmapv1.ParamConstraint) []map[string]any {
	result := make([]map[string]any, 0, len(constraints))
	for _, constraint := range constraints {
		value := map[string]any{"operator": constraint.GetOp(), "unit": constraint.GetUnit()}
		if constraintHasMin(constraint.GetOp()) {
			value["vmin"] = constraint.GetVmin()
		}
		if constraintHasMax(constraint.GetOp()) {
			value["vmax"] = constraint.GetVmax()
		}
		result = append(result, map[string]any{
			"parameter": map[string]any{
				"slug": constraint.GetParameter(),
				"name": parameterLabel(constraint.GetParameter()),
			},
			"value": value,
		})
	}
	return result
}

func constraintHasMin(op string) bool {
	switch op {
	case "lte", "lt", "to":
		return false
	default:
		return true
	}
}

func constraintHasMax(op string) bool {
	switch op {
	case "gte", "gt", "from":
		return false
	default:
		return true
	}
}

var parameterLabels = map[string]string{
	"property:tds":                     "сухой остаток",
	"parameter:sulfate-concentration":  "концентрация сульфатов",
	"parameter:chloride-concentration": "концентрация хлоридов",
	"parameter:catholyte-flow-rate":    "скорость потока",
	"parameter:temperature":            "температура",
	"parameter:current-density":        "плотность тока",
}

func parameterLabel(slug string) string {
	if label, ok := parameterLabels[slug]; ok {
		return label
	}
	if _, after, ok := strings.Cut(slug, ":"); ok {
		return strings.ReplaceAll(after, "-", " ")
	}
	return slug
}

func mapEvidence(pack *kmapv1.EvidencePack) map[string]any {
	facts := make([]any, 0, len(pack.GetFacts()))
	for _, item := range pack.GetFacts() {
		facts = append(facts, structMap(item.GetPayload()))
	}
	experts := make([]any, 0, len(pack.GetExperts()))
	for _, item := range pack.GetExperts() {
		experts = append(experts, structMap(item.GetEvidence()))
	}
	return map[string]any{
		"facts":          facts,
		"consensus":      structList(pack.GetConsensus()),
		"contradictions": structList(pack.GetContradictions()),
		"gaps":           structList(pack.GetGaps()),
		"experts":        experts,
		"stats":          structMap(pack.GetStats()),
	}
}

func mapAnswer(answer *kmapv1.AnswerDoc) map[string]any {
	methods := []any{}
	if payload := answer.GetPayload(); payload != nil {
		if list, ok := payload.AsMap()["methods"].([]any); ok {
			methods = list
		}
	}
	guard := answer.GetGuard()
	return map[string]any{
		"summary":    answer.GetSummary(),
		"confidence": answer.GetConfidence(),
		"methods":    methods,
		"guard": map[string]any{
			"numbersChecked": guard.GetNumbersChecked(),
			"violations":     guard.GetViolations(),
			"degraded":       guard.GetDegraded(),
		},
	}
}

func structMap(value *structpb.Struct) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value.AsMap()
}

func structList(values []*structpb.Struct) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, structMap(value))
	}
	return result
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
