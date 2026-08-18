package app

import (
	"errors"
	"testing"

	"mc-server-checker/internal/domain"
)

func TestNormalizeTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		address string
		port    string
		want    domain.Target
	}{
		{"hostname and default port", "  play.example.com  ", "", domain.Target{Host: "play.example.com", Port: 25565}},
		{"hostname and explicit port", "localhost", " 25566 ", domain.Target{Host: "localhost", Port: 25566}},
		{"ipv4", "127.0.0.1", "1", domain.Target{Host: "127.0.0.1", Port: 1}},
		{"bracketed ipv6", "[2001:0db8::1]", "65535", domain.Target{Host: "2001:db8::1", Port: 65535}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeTarget(tt.address, tt.port)
			if err != nil {
				t.Fatalf("NormalizeTarget: %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeTarget = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNormalizeTargetRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		address string
		port    string
		cause   error
	}{
		{"empty address", "  ", "25565", ErrAddressRequired},
		{"host with port", "example.com:25565", "25565", ErrInvalidAddress},
		{"unbracketed ipv6", "2001:db8::1", "25565", ErrInvalidAddress},
		{"unclosed ipv6", "[2001:db8::1", "25565", ErrInvalidAddress},
		{"bracketed hostname", "[example.com]", "25565", ErrInvalidAddress},
		{"empty label", "a..example", "25565", ErrInvalidAddress},
		{"label starts hyphen", "-bad.example", "25565", ErrInvalidAddress},
		{"port zero", "example.com", "0", ErrInvalidPort},
		{"port too high", "example.com", "65536", ErrInvalidPort},
		{"port sign", "example.com", "+25565", ErrInvalidPort},
		{"port text", "example.com", "abc", ErrInvalidPort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeTarget(tt.address, tt.port)
			if err == nil || err.Kind != domain.ErrorValidation || !errors.Is(err, tt.cause) {
				t.Fatalf("error = %#v, want validation wrapping %v", err, tt.cause)
			}
		})
	}
}
