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
	"Составь связный, структурированный обзор на русском по вопросу пользователя, опираясь на приведённые фрагменты источников [Ci] и числовые факты [Fi]. Правила: " +
	"1) описывай методы, технические решения и выводы своими словами по содержанию фрагментов [Ci], каждое утверждение сопровождай ссылкой на источник [Ci] или [Fi]; " +
	"2) ВСЕ числовые значения и единицы бери ТОЛЬКО из фрагментов [Ci] и фактов [Fi] и копируй дословно — не округляй, не пересчитывай, не усредняй, не суммируй и не придумывай новых чисел; " +
	"3) количество источников, пунктов, этапов и лет указывай словами (например «три источника»), а не цифрами; " +
	"4) не нумеруй пункты цифрами; " +
	"5) противоречия не сглаживай — опиши обе стороны; " +
	"6) если фрагменты не покрывают часть вопроса — честно скажи об этом словами. " +
	"Пиши как эксперт: сгруппируй изложение по методам/подходам и по источникам, будь конкретным, опирайся на содержание фрагментов."

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
	cviews := chunkViews(pack)
	if len(views) == 0 && len(cviews) == 0 {
		return extractiveSynthesis(pack), nil
	}

	payload, err := synthesisPayload(question, pack, views, cviews)
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

func synthesisPayload(question string, pack *kmapv1.EvidencePack, views []factView, cviews []chunkView) (*structpb.Struct, error) {
	var builder strings.Builder
	builder.WriteString("Вопрос: ")
	builder.WriteString(question)
	if len(cviews) > 0 {
		builder.WriteString("\n\nФрагменты источников:\n")
		for _, view := range cviews {
			source := shortTitle(view.sourceTitle)
			if source == "" {
				source = "источник"
			}
			if view.page > 0 {
				fmt.Fprintf(&builder, "[%s] %s (стр. %d): %s\n", view.ref, source, view.page, view.text)
			} else {
				fmt.Fprintf(&builder, "[%s] %s: %s\n", view.ref, source, view.text)
			}
		}
	}
	if len(views) > 0 {
		builder.WriteString("\nЧисловые факты:\n")
	}
	for _, view := range views {
		source := view.sourceTitle
		if source == "" {
			source = view.subjectName
		}
		fmt.Fprintf(&builder, "[%s] %s — %s (источник: %s, %s)\n",
			view.ref, paramLabel(view.parameterName), view.valueText, shortTitle(source), applicabilityLabel(view.geography))
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
