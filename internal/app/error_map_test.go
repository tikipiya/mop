package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"
	"time"

	"mc-server-checker/internal/domain"
	"mc-server-checker/internal/protocol/java"
)

func TestMapError(t *testing.T) {
	t.Parallel()
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	deadlineContext, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer deadlineCancel()
	tests := []struct {
		name      string
		ctx       context.Context
		err       error
		kind      domain.ErrorKind
		retryable bool
	}{
		{"cancelled", cancelledContext, errors.New("socket closed"), domain.ErrorCancelled, false},
		{"deadline", deadlineContext, errors.New("i/o timeout"), domain.ErrorTimeout, true},
		{"dns", context.Background(), &net.DNSError{Name: "missing.invalid", Err: "no such host"}, domain.ErrorDNS, true},
		{"refused", context.Background(), fmt.Errorf("dial: %w", syscall.ECONNREFUSED), domain.ErrorRefused, true},
		{"payload limit", context.Background(), fmt.Errorf("read: %w", java.ErrPacketTooLarge), domain.ErrorPayload, false},
		{"invalid motd", context.Background(), java.ErrInvalidMOTDType, domain.ErrorPayload, false},
		{"short protocol", context.Background(), io.ErrUnexpectedEOF, domain.ErrorProtocol, false},
		{"network", context.Background(), errors.New("route unavailable"), domain.ErrorNetwork, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapError(tt.ctx, tt.err)
			if got.Kind != tt.kind || got.Retryable != tt.retryable || got.Message == "" || !errors.Is(got, tt.err) {
				t.Fatalf("MapError = %#v, want kind=%s retryable=%v", got, tt.kind, tt.retryable)
			}
		})
	}
}

func TestMapErrorPreservesAppError(t *testing.T) {
	t.Parallel()
	want := &domain.AppError{Kind: domain.ErrorValidation, Message: "stable"}
	if got := MapError(context.Background(), want); got != want {
		t.Fatalf("MapError returned a different AppError: %#v", got)
	}
}
