package java

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrInvalidPlayerCount = errors.New("status contains a negative player count")

const maxModEntries = 1024

type StatusPayload struct {
	Version     StatusVersion
	Players     StatusPlayers
	Description json.RawMessage
	Favicon     string
	ModInfo     json.RawMessage
	ForgeData   json.RawMessage
}

type StatusVersion struct {
	Name     string
	Protocol *int
}

type StatusPlayers struct {
	Max    *int
	Online *int
}

type ModInfo struct {
	Detected bool
	Loader   string
	Count    *int
	Warning  string
}

func DecodeStatus(data []byte, normalizer MOTDNormalizer) (StatusPayload, string, ModInfo, error) {
	var wire struct {
		Version     StatusVersion   `json:"version"`
		Players     StatusPlayers   `json:"players"`
		Description json.RawMessage `json:"description"`
		Favicon     string          `json:"favicon"`
		ModInfo     json.RawMessage `json:"modinfo"`
		ForgeData   json.RawMessage `json:"forgeData"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&wire); err != nil {
		return StatusPayload{}, "", ModInfo{}, fmt.Errorf("decode status json: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return StatusPayload{}, "", ModInfo{}, errors.New("status json contains trailing data")
	}
	if wire.Players.Online != nil && *wire.Players.Online < 0 || wire.Players.Max != nil && *wire.Players.Max < 0 {
		return StatusPayload{}, "", ModInfo{}, ErrInvalidPlayerCount
	}
	motd, err := normalizer.Normalize(wire.Description)
	if err != nil {
		return StatusPayload{}, "", ModInfo{}, fmt.Errorf("normalize motd: %w", err)
	}
	payload := StatusPayload{
		Version:     wire.Version,
		Players:     wire.Players,
		Description: wire.Description,
		Favicon:     wire.Favicon,
		ModInfo:     wire.ModInfo,
		ForgeData:   wire.ForgeData,
	}
	return payload, motd, decodeModInfo(wire.ModInfo, wire.ForgeData, wire.Version.Name), nil
}

func decodeModInfo(legacy, forge json.RawMessage, versionName string) ModInfo {
	if presentJSON(legacy) {
		var value struct {
			Type    string            `json:"type"`
			ModList []json.RawMessage `json:"modList"`
		}
		if err := json.Unmarshal(legacy, &value); err != nil {
			return ModInfo{Detected: true, Warning: "MOD拡張情報を解釈できません。"}
		}
		count := len(value.ModList)
		loader := strings.TrimSpace(value.Type)
		if loader == "" {
			loader = "Forge"
		}
		if count > maxModEntries {
			return ModInfo{Detected: true, Loader: loader, Warning: "MOD件数が表示上限を超えています。"}
		}
		return ModInfo{Detected: true, Loader: loader, Count: &count}
	}
	if presentJSON(forge) {
		var value struct {
			Mods []json.RawMessage `json:"mods"`
		}
		if err := json.Unmarshal(forge, &value); err != nil {
			return ModInfo{Detected: true, Warning: "MOD拡張情報を解釈できません。"}
		}
		count := len(value.Mods)
		loader := "Forge"
		if strings.Contains(strings.ToLower(versionName), "neoforge") {
			loader = "NeoForge"
		}
		if count > maxModEntries {
			return ModInfo{Detected: true, Loader: loader, Warning: "MOD件数が表示上限を超えています。"}
		}
		return ModInfo{Detected: true, Loader: loader, Count: &count}
	}
	return ModInfo{}
}

func presentJSON(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null"))
}
