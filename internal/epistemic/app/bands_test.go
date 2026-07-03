package app

import "testing"

func testBands() Bands {
	return Bands{
		Numeric: map[string]NumericBand{
			"temperature_c":        {Edges: []float64{0, 25, 60, 90, 200, 700}},
			"current_density_a_m2": {Edges: []float64{150, 250, 400}},
		},
		Categorical: map[string][]string{
			"climate": {"cold", "temperate", "unspecified"},
			"medium":  {"sulfate", "chloride", "mixed", "unspecified"},
		},
	}
}

func TestNumericBandLabel(t *testing.T) {
	t.Parallel()
	edges := []float64{0, 25, 60, 90, 200, 700}
	cases := map[float64]string{
		-10: "<0",
		0:   "0-25",
		65:  "60-90",
		200: "200-700",
		900: ">700",
	}
	for value, want := range cases {
		if got := numericBandLabel(edges, value); got != want {
			t.Errorf("numericBandLabel(%v) = %q, want %q", value, got, want)
		}
	}
}

func TestClassifyConditions(t *testing.T) {
	t.Parallel()
	bands := testBands()

	class := bands.ClassifyConditions(map[string]any{
		"temperature_c":        65.0,
		"current_density_a_m2": "320",
		"climate":              "cold",
		"medium":               "acidic",
	})

	if class["temperature_c"] != "60-90" {
		t.Errorf("temperature = %q, want 60-90", class["temperature_c"])
	}
	if class["current_density_a_m2"] != "250-400" {
		t.Errorf("current_density = %q, want 250-400", class["current_density_a_m2"])
	}
	if class["climate"] != "cold" {
		t.Errorf("climate = %q, want cold", class["climate"])
	}
	if class["medium"] != "unspecified" {
		t.Errorf("unknown medium = %q, want unspecified", class["medium"])
	}
}

func TestClassifyConditionsMissingIsUnspecified(t *testing.T) {
	t.Parallel()
	class := testBands().ClassifyConditions(map[string]any{"climate": "cold"})
	if class["temperature_c"] != "unspecified" {
		t.Errorf("missing temperature must be unspecified, got %q", class["temperature_c"])
	}
	if class["medium"] != "unspecified" {
		t.Errorf("missing medium must be unspecified, got %q", class["medium"])
	}
}
