package app

import (
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestBuildPlanAppliesFilters(t *testing.T) {
	t.Parallel()

	filters, err := structpb.NewStruct(map[string]any{
		"geography": "ru",
		"params": []any{
			map[string]any{"parameter": "parameter:temperature", "op": "range", "vmin": 60.0, "vmax": 80.0, "unit": "°C"},
		},
		"year_from": 2019.0,
		"year_to":   2024.0,
	})
	if err != nil {
		t.Fatalf("build filters: %v", err)
	}

	plan, err := buildPlan("оптимальная скорость циркуляции католита", filters)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.GetGeography() != "ru" {
		t.Fatalf("geography override failed: %q", plan.GetGeography())
	}

	var constraint *kmapv1.ParamConstraint
	for _, item := range plan.GetParamConstraints() {
		if item.GetParameter() == "parameter:temperature" {
			constraint = item
		}
	}
	if constraint == nil {
		t.Fatal("temperature constraint from filters not applied")
	}
	if constraint.GetOp() != "range" || constraint.GetVmin() != 60 || constraint.GetVmax() != 80 {
		t.Fatalf("unexpected constraint: %+v", constraint)
	}
	if constraint.GetVminSi() != 333.15 || constraint.GetVmaxSi() != 353.15 {
		t.Fatalf("SI conversion not applied: %v..%v", constraint.GetVminSi(), constraint.GetVmaxSi())
	}
	if plan.GetTimeRange() == nil {
		t.Fatal("time range not set from filters")
	}
}
