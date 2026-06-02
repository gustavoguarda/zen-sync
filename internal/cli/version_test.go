package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion_PrintsExpectedFields(t *testing.T) {
	var buf bytes.Buffer
	err := Version(&buf, "v0.1.0-test", "abc1234", "2026-06-01")
	if err != nil {
		t.Fatalf("Version returned error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"v0.1.0-test", "abc1234", "2026-06-01"} {
		if !strings.Contains(out, want) {
			t.Errorf("Version output missing %q; got: %q", want, out)
		}
	}
}
