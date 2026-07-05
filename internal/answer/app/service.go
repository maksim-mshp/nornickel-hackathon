package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

const deltaWordsPerChunk = 8

const defaultSynthesisTimeout = 35 * time.Second

type SearchClient interface {
	Search(ctx context.Context, in *kmapv1.SearchRequest, opts ...grpc.CallOption) (*kmapv1.SearchResponse, error)
}

type Planner interface {
	Plan(ctx context.Context, question string, filters *structpb.Struct) (*kmapv1.QueryPlan, []string, bool)
}

type FactRetriever interface {
	FactsByText(ctx context.Context, termsQuery string, question string, limit int) ([]*kmapv1.Fact, error)
}

type ChunkRetriever interface {
	ChunksByText(ctx context.Context, termsQuery string, question string, limit int) ([]*kmapv1.Chunk, error)
	VectorChunks(ctx context.Context, vector []float32, limit int) ([]*kmapv1.Chunk, error)
}

type Embedder interface {
	Embed(ctx context.Context, in *kmapv1.EmbedRequest, opts ...grpc.CallOption) (*kmapv1.EmbedResponse, error)
}

type CachedAnswer struct {
	Plan     *kmapv1.QueryPlan
	Evidence *kmapv1.EvidencePack
	Answer   *kmapv1.AnswerDoc
}

type Cache interface {
	Get(ctx context.Context, key []byte) (*CachedAnswer, bool, error)
	Put(ctx context.Context, key []byte, value *CachedAnswer) error
}

type Emit func(*kmapv1.AskResponse) error

type Synthesis struct {
	Summary    string
	Methods    []methodView
	Confidence float64
}

type Synthesizer interface {
	Synthesize(ctx context.Context, question string, pack *kmapv1.EvidencePack) (Synthesis, error)
}

type extractiveSynthesizer struct{}

func (extractiveSynthesizer) Synthesize(_ context.Context, _ string, pack *kmapv1.EvidencePack) (Synthesis, error) {
	return extractiveSynthesis(pack), nil
}

func extractiveSynthesis(pack *kmapv1.EvidencePack) Synthesis {
	summary, methods, confidence := synthesize(pack)
	return Synthesis{Summary: summary, Methods: methods, Confidence: confidence}
}

type Service struct {
	search         SearchClient
	cache          Cache
	synth          Synthesizer
	planner        Planner
	retriever      FactRetriever
	chunkRetriever ChunkRetriever
	embedder       Embedder
	synthTimeout   time.Duration
}

type Option func(*Service)

func WithCache(cache Cache) Option {
	return func(service *Service) {
		service.cache = cache
	}
}

func WithSynthesizer(synth Synthesizer) Option {
	return func(service *Service) {
		service.synth = synth
	}
}

func WithPlanner(planner Planner) Option {
	return func(service *Service) {
		service.planner = planner
	}
}

func WithRetriever(retriever FactRetriever) Option {
	return func(service *Service) {
		service.retriever = retriever
	}
}

func WithChunkRetriever(retriever ChunkRetriever) Option {
	return func(service *Service) {
		service.chunkRetriever = retriever
	}
}

func WithEmbedder(embedder Embedder) Option {
	return func(service *Service) {
		service.embedder = embedder
	}
}

func WithSynthesisTimeout(timeout time.Duration) Option {
	return func(service *Service) {
		if timeout > 0 {
			service.synthTimeout = timeout
		}
	}
}

func NewService(search SearchClient, options ...Option) *Service {
	service := &Service{search: search, synth: extractiveSynthesizer{}, synthTimeout: defaultSynthesisTimeout}
	for _, option := range options {
		option(service)
	}
	return service
}

func (service *Service) ParseQuery(question string) (*kmapv1.QueryPlan, error) {
	if question == "" {
		return nil, status.Error(codes.InvalidArgument, "question is required")
	}
	plan, err := buildPlan(question, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build plan: %v", err)
	}
	return plan, nil
}

const (
	minFactsForAnswer = 3
	ftsFactLimit      = 24
)

func (service *Service) planAndTerms(ctx context.Context, question string, filters *structpb.Struct) (*kmapv1.QueryPlan, []string, error) {
	if service.planner != nil {
		if plan, terms, ok := service.planner.Plan(ctx, question, filters); ok {
			return plan, terms, nil
		}
	}
	plan, err := buildPlan(question, filters)
	if err != nil {
		return nil, nil, err
	}
	return plan, ruleTerms(plan), nil
}

func ruleTerms(plan *kmapv1.QueryPlan) []string {
	var terms []string
	fields := plan.GetEntities().GetFields()
	for _, group := range []string{"materials", "processes", "properties"} {
		list := fields[group].GetListValue()
		if list == nil {
			continue
		}
		for _, item := range list.GetValues() {
			if name := item.GetStructValue().GetFields()["name"].GetStringValue(); name != "" {
				terms = append(terms, name)
			}
		}
	}
	return terms
}

func (service *Service) augmentFacts(ctx context.Context, pack *kmapv1.EvidencePack, terms []string, question string) *kmapv1.EvidencePack {
	if service.retriever == nil || len(pack.GetFacts()) >= minFactsForAnswer {
		return pack
	}
	facts, err := service.retriever.FactsByText(ctx, ftsTermsQuery(terms, question), question, ftsFactLimit)
	if err != nil || len(facts) == 0 {
		return pack
	}
	pack.Facts = facts
	pack.Contradictions = nil
	if stats, err := structpb.NewStruct(factStats(facts)); err == nil {
		pack.Stats = stats
	}
	return pack
}

func factStats(facts []*kmapv1.Fact) map[string]any {
	return evidenceStats(facts, nil)
}

const chunkTextLimit = 10

func (service *Service) augmentChunks(ctx context.Context, pack *kmapv1.EvidencePack, terms []string, question string) *kmapv1.EvidencePack {
	if service.chunkRetriever == nil {
		return pack
	}
	var chunks []*kmapv1.Chunk
	if vector := service.embedQuery(ctx, question); len(vector) > 0 {
		if semantic, err := service.chunkRetriever.VectorChunks(ctx, vector, chunkTextLimit); err == nil {
			chunks = semantic
		}
	}
	if len(chunks) == 0 {
		if len(pack.GetChunks()) > 0 {
			return pack
		}
		if lexical, err := service.chunkRetriever.ChunksByText(ctx, ftsTermsQuery(terms, question), question, chunkTextLimit); err == nil {
			chunks = lexical
		}
	}
	if len(chunks) == 0 {
		return pack
	}
	pack.Chunks = chunks
	if stats, err := structpb.NewStruct(evidenceStats(pack.GetFacts(), chunks)); err == nil {
		pack.Stats = stats
	}
	return pack
}

func (service *Service) embedQuery(ctx context.Context, question string) []float32 {
	if service.embedder == nil || strings.TrimSpace(question) == "" {
		return nil
	}
	response, err := service.embedder.Embed(ctx, &kmapv1.EmbedRequest{Texts: []string{question}, Mode: "query"})
	if err != nil || len(response.GetVectors()) == 0 {
		return nil
	}
	return response.GetVectors()[0].GetValues()
}

func evidenceStats(facts []*kmapv1.Fact, chunks []*kmapv1.Chunk) map[string]any {
	docGeo := map[string]string{}
	docYear := map[string]int{}
	for _, fact := range facts {
		fields := fact.GetPayload().GetFields()
		prov := fields["provenance"].GetStructValue().GetFields()
		key := prov["documentId"].GetStringValue()
		if key == "" {
			key = prov["title"].GetStringValue()
		}
		if key == "" {
			continue
		}
		docGeo[key] = fields["geography"].GetStringValue()
		if year := int(prov["year"].GetNumberValue()); year > 0 {
			docYear[key] = year
		}
	}
	for _, chunk := range chunks {
		meta := chunk.GetMeta().GetFields()
		key := chunk.GetDocumentId()
		if key == "" {
			key = meta["title"].GetStringValue()
		}
		if key == "" {
			continue
		}
		if _, ok := docGeo[key]; !ok {
			docGeo[key] = meta["geography"].GetStringValue()
		}
		if year := int(meta["year"].GetNumberValue()); year > 0 {
			if _, ok := docYear[key]; !ok {
				docYear[key] = year
			}
		}
	}
	ru, foreign, yearFrom, yearTo := 0, 0, 0, 0
	for key, geo := range docGeo {
		switch geo {
		case "ru":
			ru++
		case "foreign":
			foreign++
		}
		if year := docYear[key]; year > 0 {
			if yearFrom == 0 || year < yearFrom {
				yearFrom = year
			}
			if year > yearTo {
				yearTo = year
			}
		}
	}
	return map[string]any{
		"sources":        float64(len(docGeo)),
		"ruSources":      float64(ru),
		"foreignSources": float64(foreign),
		"yearFrom":       float64(yearFrom),
		"yearTo":         float64(yearTo),
	}
}

var ftsStopTerms = map[string]bool{
	"переработка": true, "переработки": true, "обзор": true, "анализ": true,
	"метод": true, "методы": true, "методов": true, "способ": true, "способы": true, "способов": true,
	"технология": true, "технологии": true, "технологий": true, "техническое": true, "технических": true, "технической": true,
	"решение": true, "решения": true, "решений": true, "практика": true, "практике": true, "практики": true,
	"мировая": true, "мировой": true, "отечественная": true, "отечественной": true,
	"современные": true, "современных": true, "существующих": true, "информация": true, "информации": true,
	"производство": true, "производства": true, "предприятие": true, "предприятий": true,
	"промышленность": true, "промышленности": true, "использование": true, "использования": true,
	"организация": true, "организации": true, "показатель": true, "показатели": true,
	"данные": true, "данных": true, "источник": true, "источники": true, "источников": true,
	"литературный": true, "литературного": true,
}

func ftsTermsQuery(terms []string, question string) string {
	seen := map[string]bool{}
	out := make([]string, 0, len(terms)+8)
	add := func(value string) {
		value = strings.TrimSpace(value)
		lower := strings.ToLower(value)
		if value == "" || ftsStopTerms[lower] || seen[lower] {
			return
		}
		seen[lower] = true
		out = append(out, value)
	}
	for _, term := range terms {
		add(term)
	}
	if len(out) >= 2 {
		return strings.Join(out, " OR ")
	}
	for _, word := range strings.FieldsFunc(question, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len([]rune(word)) >= 3 {
			add(word)
		}
		if len(out) >= 10 {
			break
		}
	}
	return strings.Join(out, " OR ")
}

func (service *Service) runSynthesis(ctx context.Context, question string, pack *kmapv1.EvidencePack) (Synthesis, error) {
	if service.synthTimeout <= 0 {
		return service.synth.Synthesize(ctx, question, pack)
	}
	synthCtx, cancel := context.WithTimeout(ctx, service.synthTimeout)
	defer cancel()
	return service.synth.Synthesize(synthCtx, question, pack)
}

func (service *Service) Ask(ctx context.Context, req *kmapv1.AskRequest, emit Emit) error {
	question := req.GetQuestion()
	if question == "" {
		return status.Error(codes.InvalidArgument, "question is required")
	}

	plan, terms, err := service.planAndTerms(ctx, question, req.GetFilters())
	if err != nil {
		return status.Errorf(codes.Internal, "build plan: %v", err)
	}

	key := planCacheKey(plan, req.GetPrincipal().GetDocAccess())
	if service.cache != nil {
		if cached, ok, err := service.cache.Get(ctx, key); err == nil && ok {
			return replayCached(ctx, cached, emit)
		}
	}

	if err := emit(&kmapv1.AskResponse{Type: "plan", Plan: plan}); err != nil {
		return err
	}

	searchResp, err := service.search.Search(ctx, &kmapv1.SearchRequest{Plan: plan, Principal: req.GetPrincipal()})
	if err != nil {
		return status.Errorf(codes.Internal, "search: %v", err)
	}
	pack := searchResp.GetEvidence()
	if pack == nil {
		pack = &kmapv1.EvidencePack{}
	}
	pack = service.augmentFacts(ctx, pack, terms, question)
	pack = service.augmentChunks(ctx, pack, terms, question)
	if err := emit(&kmapv1.AskResponse{Type: "evidence", Evidence: pack}); err != nil {
		return err
	}

	result, err := service.runSynthesis(ctx, question, pack)
	degraded := false
	if err != nil {
		result = extractiveSynthesis(pack)
		degraded = true
	}
	guard := runGuard(result.Summary, pack)
	if guard.violations > 0 {
		result = extractiveSynthesis(pack)
		guard = runGuard(result.Summary, pack)
		degraded = true
	}

	for _, delta := range chunkDeltas(result.Summary, deltaWordsPerChunk) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(&kmapv1.AskResponse{Type: "answer.delta", Delta: delta}); err != nil {
			return err
		}
	}

	payload, err := methodsStruct(result.Methods)
	if err != nil {
		return status.Errorf(codes.Internal, "encode answer: %v", err)
	}
	answer := &kmapv1.AnswerDoc{
		Summary:    result.Summary,
		Confidence: result.Confidence,
		Payload:    payload,
		Guard: &kmapv1.GuardReport{
			NumbersChecked: uint32(guard.numbersChecked),
			Violations:     uint32(guard.violations),
			Degraded:       degraded,
		},
	}

	if service.cache != nil && guard.violations == 0 && !degraded {
		_ = service.cache.Put(ctx, key, &CachedAnswer{Plan: plan, Evidence: pack, Answer: answer})
	}

	return emit(&kmapv1.AskResponse{Type: "answer.done", Answer: answer})
}

func replayCached(ctx context.Context, cached *CachedAnswer, emit Emit) error {
	if err := emit(&kmapv1.AskResponse{Type: "plan", Plan: cached.Plan}); err != nil {
		return err
	}
	if err := emit(&kmapv1.AskResponse{Type: "evidence", Evidence: cached.Evidence}); err != nil {
		return err
	}
	for _, delta := range chunkDeltas(cached.Answer.GetSummary(), deltaWordsPerChunk) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(&kmapv1.AskResponse{Type: "answer.delta", Delta: delta}); err != nil {
			return err
		}
	}
	return emit(&kmapv1.AskResponse{Type: "answer.done", Answer: cached.Answer})
}

func planCacheKey(plan *kmapv1.QueryPlan, docAccess string) []byte {
	digest := sha256.Sum256([]byte(canonicalPlan(plan) + "\x00" + docAccess))
	return digest[:]
}

func canonicalPlan(plan *kmapv1.QueryPlan) string {
	slugs := planEntitySlugs(plan)
	sort.Strings(slugs)
	constraints := make([]string, 0, len(plan.GetParamConstraints()))
	for _, constraint := range plan.GetParamConstraints() {
		constraints = append(constraints, fmt.Sprintf("%s|%s|%g|%g|%s",
			constraint.GetParameter(), constraint.GetOp(), constraint.GetVmin(), constraint.GetVmax(), constraint.GetUnit()))
	}
	sort.Strings(constraints)
	years := ""
	if tr := plan.GetTimeRange(); tr != nil {
		fields := tr.GetFields()
		years = fmt.Sprintf("%g-%g", fields["year_from"].GetNumberValue(), fields["year_to"].GetNumberValue())
	}
	return strings.Join([]string{
		plan.GetIntent(), plan.GetGeography(),
		strings.Join(slugs, ","), strings.Join(constraints, ";"), years,
	}, "\x1f")
}

func planEntitySlugs(plan *kmapv1.QueryPlan) []string {
	var slugs []string
	fields := plan.GetEntities().GetFields()
	for _, group := range []string{"materials", "processes", "properties"} {
		list := fields[group].GetListValue()
		if list == nil {
			continue
		}
		for _, item := range list.GetValues() {
			if slug := item.GetStructValue().GetFields()["slug"].GetStringValue(); slug != "" {
				slugs = append(slugs, slug)
			}
		}
	}
	return slugs
}

func methodsStruct(methods []methodView) (*structpb.Struct, error) {
	data, err := json.Marshal(map[string]any{"methods": methods})
	if err != nil {
		return nil, fmt.Errorf("marshal methods: %w", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(data, &asMap); err != nil {
		return nil, fmt.Errorf("unmarshal methods: %w", err)
	}
	return structpb.NewStruct(asMap)
}
