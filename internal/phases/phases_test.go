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

func TestManagedWorkflowsAreSequentialAndComplete(t *testing.T) {
	for _, mode := range []string{ModeRefactor, ModeGreenfield, ModeFeature, ModeDebug} {
		workflow, err := ForMode(mode)
		if err != nil {
			t.Fatal(err)
		}
		if len(workflow.Phases) == 0 || len(workflow.Stages) != len(workflow.Phases) {
			t.Fatalf("%s workflow is incomplete: %+v", mode, workflow)
		}
		for i, stage := range workflow.Stages {
			if len(stage) != 1 || stage[0] != i {
				t.Fatalf("%s stage %d is not a human-gated sequential phase: %v", mode, i, stage)
			}
		}
		for i, id := range workflow.PhaseIDs() {
			if workflow.IndexOf(id) != i {
				t.Fatalf("%s phase %q has the wrong index", mode, id)
			}
		}
	}
	if _, err := ForMode(ModeAuto); err == nil {
		t.Fatal("auto must be resolved before selecting a workflow")
	}
}
