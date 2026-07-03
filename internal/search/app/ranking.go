package app

import "math"

type Ranking struct {
	Weights           RankingWeights     `koanf:"weights"`
	SourceReliability map[string]float64 `koanf:"source_reliability"`
	ValidationLevel   map[string]float64 `koanf:"validation_level"`
	FreshnessLambda   float64            `koanf:"freshness_lambda"`
	RRFK              int                `koanf:"rrf_k"`
	VectorTop         int                `koanf:"vector_top"`
	FTSTop            int                `koanf:"fts_top"`
	RerankTop         int                `koanf:"rerank_top"`
	FinalTop          int                `koanf:"final_top"`
}

type RankingWeights struct {
	MatchStrength     float64 `koanf:"match_strength"`
	RerankScore       float64 `koanf:"rerank_score"`
	SourceReliability float64 `koanf:"source_reliability"`
	ValidationLevel   float64 `koanf:"validation_level"`
	Freshness         float64 `koanf:"freshness"`
}

func DefaultRanking() Ranking {
	return Ranking{
		Weights: RankingWeights{
			MatchStrength: 0.35, RerankScore: 0.25, SourceReliability: 0.15,
			ValidationLevel: 0.15, Freshness: 0.10,
		},
		SourceReliability: map[string]float64{
			"protocol": 1.0, "report": 1.0, "article": 0.9, "patent": 0.8,
			"handbook": 0.9, "normative": 0.9, "dataset": 1.0, "web": 0.5,
		},
		ValidationLevel: map[string]float64{
			"expert_validated": 1.0, "multi_source": 0.9, "machine_extracted": 0.6,
			"weak_evidence": 0.5, "contradicted": 0.3,
		},
		FreshnessLambda: 0.1,
		RRFK:            60,
		VectorTop:       200,
		FTSTop:          200,
		RerankTop:       80,
		FinalTop:        30,
	}
}

func (ranking Ranking) finalTop() int {
	if ranking.FinalTop <= 0 {
		return 30
	}
	return ranking.FinalTop
}

func (ranking Ranking) ftsTop() int {
	if ranking.FTSTop <= 0 {
		return 200
	}
	return ranking.FTSTop
}

func (ranking Ranking) score(matchStrength float64, confidence float64, docType string, validation string, docYear int, currentYear int) ScoreComponents {
	source := lookup(ranking.SourceReliability, docType, 0.7)
	validationLevel := lookup(ranking.ValidationLevel, validation, 0.6)
	freshness := math.Exp(-ranking.FreshnessLambda * float64(currentYear-docYear))
	if docYear == 0 {
		freshness = 0.5
	}
	rerank := 0.5 + 0.5*confidence
	return ScoreComponents{
		Match:      round2(matchStrength),
		Rerank:     round2(rerank),
		Source:     round2(source),
		Validation: round2(validationLevel),
		Freshness:  round2(freshness),
	}
}

func (ranking Ranking) finalScore(components ScoreComponents) float64 {
	weights := ranking.Weights
	total := weights.MatchStrength*components.Match +
		weights.RerankScore*components.Rerank +
		weights.SourceReliability*components.Source +
		weights.ValidationLevel*components.Validation +
		weights.Freshness*components.Freshness
	return round2(total)
}

func lookup(table map[string]float64, key string, fallback float64) float64 {
	if value, ok := table[key]; ok {
		return value
	}
	return fallback
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
