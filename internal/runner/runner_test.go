package runner

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCodexEventToLines(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"agent message", `{"type":"item.completed","item":{"type":"agent_message","text":"Done."}}`, []string{"Done."}},
		{"command start", `{"type":"item.started","item":{"type":"command_execution","command":"go test ./..."}}`, []string{"  ⚙  go test ./..."}},
		{"command output", `{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"PASS"}}`, []string{"  ↳  PASS"}},
		{"error", `{"type":"error","message":"authentication failed"}`, []string{"  [error] authentication failed"}},
		{"unknown", `{"type":"turn.started"}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event codexEvent
			if err := json.Unmarshal([]byte(tt.raw), &event); err != nil {
				t.Fatal(err)
			}
			if got := codexEventToLines(event); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
