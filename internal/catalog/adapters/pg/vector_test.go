package pg

import "testing"

func TestVectorLiteral(t *testing.T) {
	t.Parallel()
	if got := vectorLiteral(nil); got != nil {
		t.Errorf("nil embedding = %v, want nil", got)
	}
	if got := vectorLiteral([]float32{}); got != nil {
		t.Errorf("empty embedding = %v, want nil", got)
	}
	got := vectorLiteral([]float32{0.1, -0.25, 1})
	if got != "[0.1,-0.25,1]" {
		t.Errorf("vectorLiteral = %v, want [0.1,-0.25,1]", got)
	}
}
