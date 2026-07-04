package http

import (
	"encoding/json"
	"math"
	stdhttp "net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
)

func pathParam(r *stdhttp.Request, key string) string {
	value := chi.URLParam(r, key)
	if decoded, err := url.PathUnescape(value); err == nil {
		return decoded
	}
	return value
}

type itemsResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	Total      uint32 `json:"total,omitempty"`
}

type entitySummaryDTO struct {
	ID     string         `json:"id"`
	Slug   string         `json:"slug"`
	Name   string         `json:"name"`
	NameEn string         `json:"nameEn"`
	Etype  string         `json:"etype"`
	Counts map[string]any `json:"counts,omitempty"`
}

type expertProfile struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Lab         string           `json:"lab"`
	Weight      float64          `json:"weight"`
	Reports     int              `json:"reports"`
	Experiments int              `json:"experiments"`
	LastYear    int              `json:"lastYear"`
	Topics      []string         `json:"topics"`
	Activity    []activityPoint  `json:"activity"`
	Evidence    []expertEvidence `json:"evidence"`
}

type activityPoint struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

type expertEvidence struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	Kind  string `json:"kind"`
}

type experimentRow struct {
	ID         string         `json:"id"`
	Code       string         `json:"code"`
	Material   string         `json:"material"`
	Process    string         `json:"process"`
	Conditions map[string]any `json:"conditions"`
	Result     string         `json:"result"`
	Source     string         `json:"source"`
	DocType    string         `json:"docType"`
	Confidence float64        `json:"confidence"`
}

type documentRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	DocType     string `json:"docType"`
	Lang        string `json:"lang"`
	Geography   string `json:"geography"`
	AccessLevel string `json:"accessLevel"`
	Status      string `json:"status"`
	Facts       uint32 `json:"facts"`
	Year        int32  `json:"year"`
}

type coverageCell struct {
	ID         string         `json:"id"`
	Domain     string         `json:"domain"`
	Material   string         `json:"material"`
	Process    string         `json:"process"`
	Condition  string         `json:"condition"`
	Score      float64        `json:"score"`
	GapFlag    bool           `json:"gap_flag"`
	Reasons    []string       `json:"reasons"`
	Counters   map[string]any `json:"counters"`
	Components map[string]any `json:"scoreComponents"`
}

type contradictionDTO struct {
	ID          string   `json:"id"`
	ClusterID   string   `json:"clusterId"`
	Status      string   `json:"status"`
	Dtype       string   `json:"dtype"`
	Severity    float64  `json:"severity"`
	Subject     string   `json:"subject"`
	Parameter   string   `json:"parameter"`
	AStatement  string   `json:"aStatement"`
	BStatement  string   `json:"bStatement"`
	Cause       string   `json:"cause"`
	Confounders []string `json:"confounders"`
}

type graphDTO struct {
	Nodes []graphNodeDTO `json:"nodes"`
	Edges []graphEdgeDTO `json:"edges"`
}

type graphNodeDTO struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type graphEdgeDTO struct {
	ID            string  `json:"id"`
	Src           string  `json:"src"`
	Dst           string  `json:"dst"`
	Rel           string  `json:"rel"`
	Weight        float64 `json:"weight"`
	Confidence    float64 `json:"confidence"`
	Contradiction bool    `json:"contradiction"`
}

type factStatusBody struct {
	Status   string `json:"status"`
	Comment  string `json:"comment"`
	FactKind string `json:"fact_kind"`
}

type mergeEntityBody struct {
	IntoID  string `json:"into_id"`
	Comment string `json:"comment"`
}

type contradictionDecisionBody struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}

func (server *Server) entitiesHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	resp, err := server.search.ListEntities(r.Context(), &kmapv1.ListEntitiesRequest{
		Type:      r.URL.Query().Get("type"),
		Query:     r.URL.Query().Get("q"),
		Page:      pageRequest(r),
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	items := make([]entitySummaryDTO, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, mapEntitySummary(item))
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[entitySummaryDTO]{Items: items, NextCursor: resp.GetPage().GetNextCursor()})
}

func (server *Server) entityHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	resp, err := server.search.GetEntity(r.Context(), &kmapv1.GetEntityRequest{
		EntityId:  pathParam(r, "id"),
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	writeDataJSON(w, stdhttp.StatusOK, mapEntityCard(resp.GetEntity()))
}

func (server *Server) entityFactsHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	resp, err := server.search.ListEntityFacts(r.Context(), &kmapv1.ListEntityFactsRequest{
		EntityId:  pathParam(r, "id"),
		Parameter: r.URL.Query().Get("param"),
		Page:      pageRequest(r),
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(resp.GetFacts()))
	for _, item := range resp.GetFacts() {
		items = append(items, structMap(item.GetPayload()))
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[map[string]any]{Items: items, NextCursor: resp.GetPage().GetNextCursor()})
}

func (server *Server) experimentsHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	resp, err := server.search.ListExperiments(r.Context(), &kmapv1.ListExperimentsRequest{
		Material:  r.URL.Query().Get("material"),
		Process:   r.URL.Query().Get("process"),
		YearFrom:  int32Value(r.URL.Query().Get("year_from")),
		Parameter: r.URL.Query().Get("param"),
		Op:        r.URL.Query().Get("op"),
		Value:     floatValue(r.URL.Query().Get("value")),
		Unit:      r.URL.Query().Get("unit"),
		Page:      pageRequest(r),
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	items := make([]experimentRow, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, mapExperiment(item))
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[experimentRow]{Items: items, NextCursor: resp.GetPage().GetNextCursor()})
}

func (server *Server) expertsHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	req := &kmapv1.ListExpertsRequest{
		Topic:     r.URL.Query().Get("topic"),
		EntityId:  r.URL.Query().Get("entity_id"),
		Page:      pageRequest(r),
		Principal: principalFromContext(r),
	}
	resp, err := server.search.ListExperts(r.Context(), req)
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}

	items := make([]expertProfile, 0, len(resp.GetExperts()))
	for _, item := range resp.GetExperts() {
		items = append(items, mapExpert(item))
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[expertProfile]{Items: items, NextCursor: resp.GetPage().GetNextCursor()})
}

func (server *Server) documentsHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	resp, err := server.ingest.ListDocuments(r.Context(), &kmapv1.ListDocumentsRequest{
		Page:      pageRequest(r),
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	items := make([]documentRow, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, mapDocument(item))
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[documentRow]{Items: items, Total: resp.GetPage().GetTotal()})
}

func mapEntitySummary(item *kmapv1.EntitySummary) entitySummaryDTO {
	return entitySummaryDTO{
		ID:     item.GetId(),
		Slug:   item.GetSlug(),
		Name:   item.GetName(),
		NameEn: item.GetNameEn(),
		Etype:  item.GetEtype(),
		Counts: map[string]any{"facts": item.GetFacts(), "relations": item.GetRelations()},
	}
}

func mapEntityCard(item *kmapv1.EntityCard) map[string]any {
	return map[string]any{
		"id":        item.GetId(),
		"slug":      item.GetSlug(),
		"nameRu":    item.GetNameRu(),
		"nameEn":    item.GetNameEn(),
		"type":      item.GetType(),
		"synonyms":  item.GetSynonyms(),
		"counters":  structMap(item.GetCounters()),
		"consensus": structList(item.GetConsensus()),
		"relations": graphEdges(item.GetRelations()),
		"experts":   mapExperts(item.GetExperts()),
		"timeline":  structList(item.GetTimeline()),
	}
}

func mapExperiment(item *kmapv1.ExperimentSummary) experimentRow {
	return experimentRow{
		ID:         item.GetId(),
		Code:       item.GetCode(),
		Material:   item.GetMaterial(),
		Process:    item.GetProcess(),
		Conditions: structMap(item.GetConditions()),
		Result:     item.GetResult(),
		Source:     item.GetSource(),
		DocType:    item.GetDocType(),
		Confidence: item.GetConfidence(),
	}
}

func mapDocument(item *kmapv1.DocumentSummary) documentRow {
	return documentRow{
		ID:          item.GetId(),
		Title:       item.GetTitle(),
		DocType:     item.GetDocType(),
		Lang:        item.GetLang(),
		Geography:   item.GetGeography(),
		AccessLevel: item.GetAccessLevel(),
		Status:      item.GetStatus(),
		Facts:       item.GetFacts(),
		Year:        item.GetYear(),
	}
}

func (server *Server) coverageHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	resp, err := server.epistemic.GetCoverage(r.Context(), &kmapv1.GetCoverageRequest{
		Domain:    r.URL.Query().Get("domain"),
		Axis1:     defaultString(r.URL.Query().Get("axis1"), "material"),
		Axis2:     defaultString(r.URL.Query().Get("axis2"), "process"),
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}

	items := make([]coverageCell, 0, len(resp.GetCells()))
	for _, item := range resp.GetCells() {
		items = append(items, mapCoverageCell(item))
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[coverageCell]{Items: items})
}

func (server *Server) contradictionsHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	resp, err := server.epistemic.GetContradictions(r.Context(), &kmapv1.GetContradictionsRequest{
		ClusterId: r.URL.Query().Get("cluster_id"),
		EntityId:  r.URL.Query().Get("entity_id"),
		Status:    r.URL.Query().Get("status"),
		Page:      pageRequest(r),
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}

	items := make([]contradictionDTO, 0, len(resp.GetContradictions()))
	for _, item := range resp.GetContradictions() {
		items = append(items, mapContradiction(item))
	}
	writeDataJSON(w, stdhttp.StatusOK, itemsResponse[contradictionDTO]{Items: items, NextCursor: resp.GetPage().GetNextCursor()})
}

func (server *Server) graphHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	resp, err := server.search.EgoGraph(r.Context(), &kmapv1.EgoGraphRequest{
		EntityId:  r.URL.Query().Get("entity_id"),
		Depth:     boundedUint(r.URL.Query().Get("depth"), 1, 3, 1),
		TopN:      boundedUint(r.URL.Query().Get("top_n"), 1, 100, 50),
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	writeDataJSON(w, stdhttp.StatusOK, mapGraph(resp.GetGraph()))
}

func (server *Server) updateFactStatusHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body factStatusBody
	if !readJSONBody(w, r, &body) {
		return
	}
	if body.Status == "" {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "status is required")
		return
	}
	resp, err := server.catalog.UpdateFactStatus(r.Context(), &kmapv1.UpdateFactStatusRequest{
		FactId:    chi.URLParam(r, "id"),
		FactKind:  defaultString(body.FactKind, "numeric"),
		Status:    body.Status,
		Comment:   body.Comment,
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	writeProto(w, resp)
}

func (server *Server) mergeEntityHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body mergeEntityBody
	if !readJSONBody(w, r, &body) {
		return
	}
	if body.IntoID == "" {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "into_id is required")
		return
	}
	resp, err := server.catalog.MergeEntities(r.Context(), &kmapv1.MergeEntitiesRequest{
		EntityId:  chi.URLParam(r, "id"),
		IntoId:    body.IntoID,
		Comment:   body.Comment,
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	writeProto(w, resp)
}

func (server *Server) decideContradictionHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var body contradictionDecisionBody
	if !readJSONBody(w, r, &body) {
		return
	}
	if body.Decision == "" {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "decision is required")
		return
	}
	resp, err := server.epistemic.DecideContradiction(r.Context(), &kmapv1.DecideContradictionRequest{
		ContradictionId: chi.URLParam(r, "id"),
		Decision:        body.Decision,
		Comment:         body.Comment,
		Principal:       principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}
	writeDataJSON(w, stdhttp.StatusOK, mapContradiction(resp.GetContradiction()))
}

func mapExpert(item *kmapv1.Expert) expertProfile {
	evidence := structMap(item.GetEvidence())
	return expertProfile{
		ID:          item.GetPersonId(),
		Name:        item.GetName(),
		Lab:         stringValue(evidence, "lab"),
		Weight:      item.GetWeight(),
		Reports:     intValue(evidence, "reports"),
		Experiments: intValue(evidence, "experiments"),
		LastYear:    intValue(evidence, "lastYear"),
		Topics:      stringList(evidence["topics"]),
		Activity:    activityList(evidence["activity"]),
		Evidence:    evidenceList(evidence["evidence"]),
	}
}

func mapCoverageCell(item *kmapv1.CoverageCell) coverageCell {
	counters := structMap(item.GetCounters())
	return coverageCell{
		ID:         item.GetId(),
		Domain:     item.GetDomain(),
		Material:   defaultString(stringValue(counters, "material"), item.GetMaterialId()),
		Process:    defaultString(stringValue(counters, "process"), item.GetProcessId()),
		Condition:  item.GetConditionKey(),
		Score:      item.GetScore(),
		GapFlag:    item.GetGapFlag(),
		Reasons:    nonNilStrings(item.GetGapReasons()),
		Counters:   counters,
		Components: structMap(item.GetScoreComponents()),
	}
}

func mapContradiction(item *kmapv1.Contradiction) contradictionDTO {
	payload := structMap(item.GetPayload())
	return contradictionDTO{
		ID:          item.GetId(),
		ClusterID:   item.GetClusterId(),
		Status:      item.GetStatus(),
		Dtype:       item.GetDtype(),
		Severity:    item.GetSeverity(),
		Subject:     stringValue(payload, "subject"),
		Parameter:   stringValue(payload, "parameter"),
		AStatement:  stringValue(payload, "aStatement"),
		BStatement:  stringValue(payload, "bStatement"),
		Cause:       stringValue(payload, "cause"),
		Confounders: stringList(payload["confounders"]),
	}
}

func mapGraph(graph *kmapv1.Graph) graphDTO {
	nodes := make([]graphNodeDTO, 0, len(graph.GetNodes()))
	for _, node := range graph.GetNodes() {
		nodes = append(nodes, graphNodeDTO{ID: node.GetId(), Type: node.GetType(), Label: node.GetLabel()})
	}
	edges := make([]graphEdgeDTO, 0, len(graph.GetEdges()))
	for _, edge := range graph.GetEdges() {
		edges = append(edges, graphEdgeDTO{
			ID: edge.GetId(), Src: edge.GetSrc(), Dst: edge.GetDst(), Rel: edge.GetRel(),
			Weight: edge.GetWeight(), Confidence: edge.GetConfidence(), Contradiction: edge.GetContradiction(),
		})
	}
	return graphDTO{Nodes: nodes, Edges: edges}
}

func graphEdges(edges []*kmapv1.GraphEdge) []graphEdgeDTO {
	items := make([]graphEdgeDTO, 0, len(edges))
	for _, edge := range edges {
		items = append(items, graphEdgeDTO{
			ID: edge.GetId(), Src: edge.GetSrc(), Dst: edge.GetDst(), Rel: edge.GetRel(),
			Weight: edge.GetWeight(), Confidence: edge.GetConfidence(), Contradiction: edge.GetContradiction(),
		})
	}
	return items
}

func mapExperts(experts []*kmapv1.Expert) []expertProfile {
	items := make([]expertProfile, 0, len(experts))
	for _, item := range experts {
		items = append(items, mapExpert(item))
	}
	return items
}

func readJSONBody(w stdhttp.ResponseWriter, r *stdhttp.Request, target any) bool {
	body, err := readBody(w, r)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return false
	}
	if err := json.Unmarshal(body, target); err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return false
	}
	return true
}

func writeDataJSON(w stdhttp.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func pageRequest(r *stdhttp.Request) *kmapv1.PageRequest {
	return &kmapv1.PageRequest{
		Cursor: r.URL.Query().Get("cursor"),
		Limit:  boundedUint(r.URL.Query().Get("limit"), 0, math.MaxUint32, 50),
		Offset: boundedUint(r.URL.Query().Get("offset"), 0, math.MaxUint32, 0),
	}
}

func boundedUint(raw string, minValue uint32, maxValue uint32, fallback uint32) uint32 {
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return fallback
	}
	value := uint32(parsed)
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func intValue(values map[string]any, key string) int {
	switch value := values[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

func int32Value(raw string) int32 {
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0
	}
	return int32(value)
}

func floatValue(raw string) float64 {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func stringList(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func activityList(value any) []activityPoint {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]activityPoint, 0, len(raw))
	for _, item := range raw {
		asMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, activityPoint{Year: intValue(asMap, "year"), Count: intValue(asMap, "count")})
	}
	return result
}

func evidenceList(value any) []expertEvidence {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]expertEvidence, 0, len(raw))
	for _, item := range raw {
		asMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, expertEvidence{
			Title: stringValue(asMap, "title"),
			Year:  intValue(asMap, "year"),
			Kind:  stringValue(asMap, "kind"),
		})
	}
	return result
}
