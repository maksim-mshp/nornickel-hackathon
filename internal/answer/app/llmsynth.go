package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

const taskSynthesize = "synthesize_answer"

const synthesisSystemPrompt = "Ты — аналитик R&D горно-металлургической отрасли. " +
	"Составь связный обзор на русском СТРОГО по приведённым фактам [Fi]. Правила: " +
	"1) каждое утверждение сопровождай ссылкой [Fi]; " +
	"2) числа и единицы бери ТОЛЬКО из фактов [Fi] и копируй дословно — не округляй, не пересчитывай, не усредняй, не суммируй; " +
	"3) НЕ пиши никаких других чисел: количество источников, пунктов, этапов, лет указывай словами (например «три источника»), а не цифрами; " +
	"4) не нумеруй пункты цифрами; " +
	"5) противоречия не сглаживай — опиши обе стороны; " +
	"6) если релевантных данных мало — скажи об этом словами, без цифр. " +
	"Группируй изложение по темам и источникам, будь конкретным, опирайся на цитаты."

var errEmptySynthesis = errors.New("empty synthesis from llm")

type LLMClient interface {
	Complete(ctx context.Context, in *kmapv1.CompleteRequest, opts ...grpc.CallOption) (*kmapv1.CompleteResponse, error)
}

type LLMSynthesizer struct {
	llm LLMClient
}

func NewLLMSynthesizer(llm LLMClient) *LLMSynthesizer {
	return &LLMSynthesizer{llm: llm}
}

func (synth *LLMSynthesizer) Synthesize(ctx context.Context, question string, pack *kmapv1.EvidencePack) (Synthesis, error) {
	views := factViews(pack)
	if len(views) == 0 {
		return extractiveSynthesis(pack), nil
	}

	payload, err := synthesisPayload(question, pack, views)
	if err != nil {
		return Synthesis{}, err
	}
	response, err := synth.llm.Complete(ctx, &kmapv1.CompleteRequest{Task: taskSynthesize, Payload: payload})
	if err != nil {
		return Synthesis{}, err
	}
	text := strings.TrimSpace(response.GetJson().GetFields()["text"].GetStringValue())
	if text == "" {
		return Synthesis{}, errEmptySynthesis
	}
	return Synthesis{
		Summary:    text,
		Methods:    deriveMethods(views),
		Confidence: deriveConfidence(pack),
	}, nil
}

func synthesisPayload(question string, pack *kmapv1.EvidencePack, views []factView) (*structpb.Struct, error) {
	var builder strings.Builder
	builder.WriteString("Вопрос: ")
	builder.WriteString(question)
	builder.WriteString("\n\nФакты:\n")
	for _, view := range views {
		fmt.Fprintf(&builder, "[%s] %s — %s (%s, %s)\n",
			view.ref, view.parameterName, view.valueText, view.subjectName, applicabilityLabel(view.geography))
	}
	if consensus := consensusParagraph(pack); consensus != "" {
		builder.WriteString("\n" + consensus + "\n")
	}
	if contradiction := contradictionParagraph(pack); contradiction != "" {
		builder.WriteString(contradiction + "\n")
	}
	if gap := gapParagraph(pack); gap != "" {
		builder.WriteString(gap + "\n")
	}

	messages := []any{
		map[string]any{"role": "system", "content": synthesisSystemPrompt},
		map[string]any{"role": "user", "content": builder.String()},
	}
	return structpb.NewStruct(map[string]any{"messages": messages})
}
