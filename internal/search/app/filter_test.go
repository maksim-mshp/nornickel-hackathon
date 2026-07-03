package app

import (
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
)

func TestFactFilterKeepsGeographyAndSIConstraints(t *testing.T) {
	t.Parallel()
	plan := &kmapv1.QueryPlan{
		Geography: "ru",
		ParamConstraints: []*kmapv1.ParamConstraint{
			{Parameter: "parameter:temperature", Op: "range", SiUnit: "K", VminSi: 333.15, VmaxSi: 353.15},
			{Parameter: "parameter:unresolved", Op: "range", SiUnit: ""},
		},
	}
	filter := factFilter(plan)

	if filter.Geography != "ru" {
		t.Fatalf("want geography ru, got %q", filter.Geography)
	}
	if len(filter.ParamSlugs) != 1 || filter.ParamSlugs[0] != "parameter:temperature" {
		t.Fatalf("want only the SI-resolved constraint, got %v", filter.ParamSlugs)
	}
	if filter.RangeLo[0] != 333.15 || filter.RangeHi[0] != 353.15 {
		t.Fatalf("want SI bounds 333.15..353.15, got %v..%v", filter.RangeLo[0], filter.RangeHi[0])
	}
}

func TestFactFilterIgnoresNonDirectionalGeography(t *testing.T) {
	t.Parallel()
	for _, geo := range []string{"any", "compare", ""} {
		filter := factFilter(&kmapv1.QueryPlan{Geography: geo})
		if filter.Geography != "" {
			t.Fatalf("geography %q must not filter, got %q", geo, filter.Geography)
		}
	}
}

func TestFactFilterUnboundedOperators(t *testing.T) {
	t.Parallel()
	plan := &kmapv1.QueryPlan{
		ParamConstraints: []*kmapv1.ParamConstraint{
			{Parameter: "parameter:tds", Op: "lte", SiUnit: "kg/m^3", VmaxSi: 1.0},
		},
	}
	filter := factFilter(plan)
	if len(filter.ParamSlugs) != 1 {
		t.Fatalf("want one constraint, got %d", len(filter.ParamSlugs))
	}
	if filter.RangeHi[0] != 1.0 {
		t.Fatalf("want upper bound 1.0, got %v", filter.RangeHi[0])
	}
	if filter.RangeLo[0] >= 0 {
		t.Fatalf("lte must be lower-unbounded, got lo=%v", filter.RangeLo[0])
	}
}
