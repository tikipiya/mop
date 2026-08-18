package domain

import "testing"

func TestTargetAddress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		target Target
		want   string
	}{
		{Target{Host: "play.example.com", Port: 25565}, "play.example.com:25565"},
		{Target{Host: "127.0.0.1", Port: 25565}, "127.0.0.1:25565"},
		{Target{Host: "2001:db8::1", Port: 25565}, "[2001:db8::1]:25565"},
	}
	for _, tt := range tests {
		if got := tt.target.Address(); got != tt.want {
			t.Errorf("Address() = %q, want %q", got, tt.want)
		}
	}
}
