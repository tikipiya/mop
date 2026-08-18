package java

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultMaxMOTDBytes    = 4 << 10
	DefaultMaxMOTDDepth    = 32
	DefaultMaxMOTDElements = 1024
)

var (
	ErrInvalidMOTDType = errors.New("motd has an unsupported type")
	ErrMOTDTooComplex  = errors.New("motd exceeds complexity limit")
)

type MOTDNormalizer struct {
	MaxBytes    int
	MaxDepth    int
	MaxElements int
}

func NewMOTDNormalizer() MOTDNormalizer {
	return MOTDNormalizer{
		MaxBytes:    DefaultMaxMOTDBytes,
		MaxDepth:    DefaultMaxMOTDDepth,
		MaxElements: DefaultMaxMOTDElements,
	}
}

func (n MOTDNormalizer) Normalize(raw json.RawMessage) (string, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	switch value.(type) {
	case string, []any, map[string]any:
	default:
		return "", ErrInvalidMOTDType
	}

	var text strings.Builder
	elements := 0
	if err := n.appendText(&text, value, 0, &elements); err != nil {
		return "", err
	}
	return truncateUTF8(cleanMOTD(text.String()), n.maxBytes()), nil
}

func (n MOTDNormalizer) appendText(dst *strings.Builder, value any, depth int, elements *int) error {
	if depth > n.maxDepth() {
		return ErrMOTDTooComplex
	}
	(*elements)++
	if *elements > n.maxElements() {
		return ErrMOTDTooComplex
	}

	switch typed := value.(type) {
	case string:
		dst.WriteString(typed)
	case []any:
		for _, item := range typed {
			if err := n.appendText(dst, item, depth+1, elements); err != nil {
				return err
			}
		}
	case map[string]any:
		if text, ok := typed["text"].(string); ok {
			dst.WriteString(text)
		}
		if extra, ok := typed["extra"]; ok {
			if err := n.appendText(dst, extra, depth+1, elements); err != nil {
				return err
			}
		}
	}
	return nil
}

func cleanMOTD(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	runes := []rune(value)
	cleaned := make([]rune, 0, len(runes))
	for i := 0; i < len(runes); i++ {
		if runes[i] == '§' {
			if i+1 < len(runes) {
				i++
			}
			continue
		}
		if unicode.IsControl(runes[i]) && runes[i] != '\n' && runes[i] != '\t' {
			continue
		}
		cleaned = append(cleaned, runes[i])
	}
	return string(cleaned)
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut]
}

func (n MOTDNormalizer) maxBytes() int {
	if n.MaxBytes <= 0 {
		return DefaultMaxMOTDBytes
	}
	return n.MaxBytes
}

func (n MOTDNormalizer) maxDepth() int {
	if n.MaxDepth <= 0 {
		return DefaultMaxMOTDDepth
	}
	return n.MaxDepth
}

func (n MOTDNormalizer) maxElements() int {
	if n.MaxElements <= 0 {
		return DefaultMaxMOTDElements
	}
	return n.MaxElements
}
