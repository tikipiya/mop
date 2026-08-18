package java

import (
	"errors"
	"testing"
)

func TestDecodeStatus(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"version":{"name":"Paper 1.21.x","protocol":769},
		"players":{"online":4,"max":20,"sample":[]},
		"description":{"text":"§aExample ","extra":[{"text":"Server"}]},
		"forgeData":{"mods":[{"modId":"a"},{"modId":"b"}]},
		"unknown":true
	}`)
	payload, motd, mod, err := DecodeStatus(data, NewMOTDNormalizer())
	if err != nil {
		t.Fatalf("DecodeStatus: %v", err)
	}
	if payload.Version.Name != "Paper 1.21.x" || payload.Version.Protocol == nil || *payload.Version.Protocol != 769 {
		t.Fatalf("version = %+v", payload.Version)
	}
	if payload.Players.Online == nil || *payload.Players.Online != 4 || payload.Players.Max == nil || *payload.Players.Max != 20 {
		t.Fatalf("players = %+v", payload.Players)
	}
	if motd != "Example Server" {
		t.Fatalf("motd = %q", motd)
	}
	if !mod.Detected || mod.Loader != "Forge" || mod.Count == nil || *mod.Count != 2 {
		t.Fatalf("mod = %+v", mod)
	}
}

func TestDecodeStatusAllowsMissingOptionalFields(t *testing.T) {
	t.Parallel()
	payload, motd, mod, err := DecodeStatus([]byte(`{"version":{},"players":{}}`), NewMOTDNormalizer())
	if err != nil {
		t.Fatalf("DecodeStatus: %v", err)
	}
	if payload.Version.Protocol != nil || payload.Players.Online != nil || payload.Players.Max != nil || motd != "" || mod.Detected {
		t.Fatalf("unexpected values: payload=%+v motd=%q mod=%+v", payload, motd, mod)
	}
}

func TestDecodeStatusRejectsInvalidCoreData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		json string
		want error
	}{
		{"negative online", `{"players":{"online":-1}}`, ErrInvalidPlayerCount},
		{"wrong type", `{"players":{"online":"many"}}`, nil},
		{"wrong motd type", `{"description":42}`, ErrInvalidMOTDType},
		{"invalid json", `{`, nil},
		{"trailing json", `{} {}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := DecodeStatus([]byte(tt.json), NewMOTDNormalizer())
			if err == nil {
				t.Fatal("DecodeStatus returned nil error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestDecodeStatusKeepsCoreResultWhenModInfoIsInvalid(t *testing.T) {
	t.Parallel()
	payload, _, mod, err := DecodeStatus([]byte(`{"version":{"name":"NeoForge 1.21"},"forgeData":"bad"}`), NewMOTDNormalizer())
	if err != nil {
		t.Fatalf("DecodeStatus: %v", err)
	}
	if payload.Version.Name != "NeoForge 1.21" || !mod.Detected || mod.Warning == "" {
		t.Fatalf("payload=%+v mod=%+v", payload, mod)
	}
}
