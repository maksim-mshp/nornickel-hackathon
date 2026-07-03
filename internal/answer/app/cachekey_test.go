package app

import (
	"bytes"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
)

func TestPlanCacheKeyVariesByDocAccess(t *testing.T) {
	t.Parallel()
	plan := &kmapv1.QueryPlan{Intent: "technology_search", Geography: "ru"}
	if bytes.Equal(planCacheKey(plan, "public"), planCacheKey(plan, "restricted")) {
		t.Fatal("different doc access must not share cache key")
	}
}

func TestPlanCacheKeyDeterministic(t *testing.T) {
	t.Parallel()
	plan := &kmapv1.QueryPlan{
		Intent:    "technology_search",
		Geography: "ru",
		ParamConstraints: []*kmapv1.ParamConstraint{
			{Parameter: "parameter:temperature", Op: "lte", Vmax: 90, Unit: "°C"},
		},
	}
	if !bytes.Equal(planCacheKey(plan, "public"), planCacheKey(plan, "public")) {
		t.Fatal("identical plan and doc access must yield identical key")
	}
}

func TestPlanCacheKeyVariesByConstraint(t *testing.T) {
	t.Parallel()
	base := &kmapv1.QueryPlan{Intent: "technology_search"}
	withConstraint := &kmapv1.QueryPlan{
		Intent: "technology_search",
		ParamConstraints: []*kmapv1.ParamConstraint{
			{Parameter: "parameter:tds", Op: "lte", Vmax: 1000, Unit: "мг/л"},
		},
	}
	if bytes.Equal(planCacheKey(base, "public"), planCacheKey(withConstraint, "public")) {
		t.Fatal("distinct numeric constraints must produce distinct cache keys")
	}
}
