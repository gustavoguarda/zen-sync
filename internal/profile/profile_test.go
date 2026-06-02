package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetect_FindsReleaseProfile(t *testing.T) {
	base := t.TempDir()
	want := filepath.Join(base, "abc.Default (release)")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Detect(base)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDetect_FallsBackToDefaultWithoutSuffix(t *testing.T) {
	base := t.TempDir()
	want := filepath.Join(base, "xyz.Default")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Detect(base)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDetect_PrefersReleaseOverPlainDefault(t *testing.T) {
	base := t.TempDir()
	release := filepath.Join(base, "rrr.Default (release)")
	plain := filepath.Join(base, "ppp.Default")
	for _, d := range []string{release, plain} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Detect(base)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got != release {
		t.Errorf("got %q, want %q (release should win)", got, release)
	}
}

func TestDetect_ErrorsOnMissingBase(t *testing.T) {
	_, err := Detect(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDetect_ErrorsOnMultiple(t *testing.T) {
	base := t.TempDir()
	for _, d := range []string{"a.Default (release)", "b.Default (release)"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Detect(base)
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if !strings.Contains(err.Error(), "multiple") {
		t.Errorf("error should mention 'multiple'; got: %v", err)
	}
}

func TestRunning_FalseWhenNoMatch(t *testing.T) {
	if Running("/this/path/will/never/match/anything") {
		t.Error("Running returned true for an impossible pattern")
	}
}
