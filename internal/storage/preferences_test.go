package storage

import (
	"testing"

	"mc-server-checker/internal/domain"
)

func TestPreferencesLoadDefaultsAndSave(t *testing.T) {
	t.Parallel()
	values := memoryPreferences{}
	preferences := NewPreferences(values)
	address, port := preferences.Load()
	if address != "" || port != "" {
		t.Fatalf("defaults = %q, %q", address, port)
	}

	preferences.Save(domain.Target{Host: "play.example.com", Port: 25566})
	address, port = preferences.Load()
	if address != "play.example.com" || port != "25566" {
		t.Fatalf("saved = %q, %q", address, port)
	}

	preferences.Save(domain.Target{Host: "srv.example.com", Port: 25565, UseSRV: true})
	address, port = preferences.Load()
	if address != "srv.example.com" || port != "" {
		t.Fatalf("SRV preferences = %q, %q", address, port)
	}
}

func TestNilPreferencesAreSafe(t *testing.T) {
	t.Parallel()
	preferences := NewPreferences(nil)
	address, port := preferences.Load()
	if address != "" || port != "" {
		t.Fatalf("defaults = %q, %q", address, port)
	}
	preferences.Save(domain.Target{Host: "ignored", Port: 25565})
}

type memoryPreferences map[string]string

func (m memoryPreferences) StringWithFallback(key, fallback string) string {
	if value, ok := m[key]; ok {
		return value
	}
	return fallback
}

func (m memoryPreferences) SetString(key, value string) { m[key] = value }
