package java

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestMOTDNormalizer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"string", `"Hello server"`, "Hello server"},
		{"text and extra", `{"text":"Hello ","extra":[{"text":"world"},"!"]}`, "Hello world!"},
		{"array", `[{"text":"one"}," two"]`, "one two"},
		{"legacy colors", `"§aGreen §lBold§r"`, "Green Bold"},
		{"controls and newlines", `"a\u0000b\r\nc\rd"`, "ab\nc\nd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMOTDNormalizer().Normalize(json.RawMessage(tt.raw))
			if err != nil {
				t.Fatalf("Normalize: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Normalize = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMOTDNormalizerLimitsComplexity(t *testing.T) {
	t.Parallel()
	normalizer := MOTDNormalizer{MaxBytes: 8, MaxDepth: 2, MaxElements: 3}
	if _, err := normalizer.Normalize(json.RawMessage(`{"text":"x","extra":[{"text":"y","extra":["z"]}]}`)); !errors.Is(err, ErrMOTDTooComplex) {
		t.Fatalf("depth error = %v", err)
	}
	if _, err := normalizer.Normalize(json.RawMessage(`["1","2","3","4"]`)); !errors.Is(err, ErrMOTDTooComplex) {
		t.Fatalf("element error = %v", err)
	}
}

func TestMOTDNormalizerTruncatesAtUTF8Boundary(t *testing.T) {
	t.Parallel()
	normalizer := MOTDNormalizer{MaxBytes: 5}
	got, err := normalizer.Normalize(json.RawMessage(`"あいう"`))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if !utf8.ValidString(got) || got != "あ" || len(got) > 5 {
		t.Fatalf("Normalize = %q (%d bytes)", got, len(got))
	}
	if strings.ContainsRune(got, utf8.RuneError) {
		t.Fatalf("Normalize contains replacement rune: %q", got)
	}
}
