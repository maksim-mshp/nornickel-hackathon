package app

import "testing"

func TestReadableSlug(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"material:catholyte":            "catholyte",
		"process:nickel-electrowinning": "nickel electrowinning",
		"copper-production-capacity":    "copper production capacity",
		"":                              "",
		"plain":                         "plain",
	}
	for input, want := range cases {
		if got := readableSlug(input); got != want {
			t.Errorf("readableSlug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRankingTopDefaults(t *testing.T) {
	t.Parallel()
	ranking := DefaultRanking()
	if ranking.FinalTop != 30 || ranking.FTSTop != 200 || ranking.RRFK != 60 {
		t.Fatalf("defaults: final=%d fts=%d rrf=%d", ranking.FinalTop, ranking.FTSTop, ranking.RRFK)
	}
	if ranking.finalTop() != 30 || ranking.ftsTop() != 200 {
		t.Fatalf("helpers: finalTop=%d ftsTop=%d", ranking.finalTop(), ranking.ftsTop())
	}
	empty := Ranking{}
	if empty.finalTop() != 30 {
		t.Errorf("empty finalTop fallback = %d, want 30", empty.finalTop())
	}
	if empty.ftsTop() != 200 {
		t.Errorf("empty ftsTop fallback = %d, want 200", empty.ftsTop())
	}
}
