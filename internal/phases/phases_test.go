package phases

import "testing"

func TestIndexOf(t *testing.T) {
	for i, phase := range All {
		if got := IndexOf(phase.ID); got != i {
			t.Fatalf("IndexOf(%q)=%d, want %d", phase.ID, got, i)
		}
	}
	if got := IndexOf("missing"); got != -1 {
		t.Fatalf("missing phase index=%d", got)
	}
}
