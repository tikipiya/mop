package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"mc-server-checker/internal/domain"
)

func TestCheckServiceSuccess(t *testing.T) {
	t.Parallel()
	checker := checkerFunc(func(ctx context.Context, target domain.Target) (domain.Result, error) {
		if target != (domain.Target{Host: "play.example.com", Port: 25565}) {
			t.Fatalf("target = %+v", target)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("checker context has no deadline")
		}
		return domain.Result{VersionName: "Test"}, nil
	})
	service := NewCheckService(checker, time.Second)
	result, err := service.Check(context.Background(), CheckInput{Address: " play.example.com "})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Status != domain.StatusOnline || result.Target.Host != "play.example.com" {
		t.Fatalf("result = %+v", result)
	}
}

func TestCheckServiceValidationDoesNotCallChecker(t *testing.T) {
	t.Parallel()
	called := false
	service := NewCheckService(checkerFunc(func(context.Context, domain.Target) (domain.Result, error) {
		called = true
		return domain.Result{}, nil
	}), time.Second)
	result, err := service.Check(context.Background(), CheckInput{Address: "", Port: "25565"})
	if err == nil || err.Kind != domain.ErrorValidation || result.Status != domain.StatusError || called {
		t.Fatalf("result=%+v error=%#v called=%v", result, err, called)
	}
}

func TestCheckServiceTimeout(t *testing.T) {
	t.Parallel()
	service := NewCheckService(checkerFunc(func(ctx context.Context, _ domain.Target) (domain.Result, error) {
		<-ctx.Done()
		return domain.Result{}, ctx.Err()
	}), 20*time.Millisecond)
	result, err := service.Check(context.Background(), CheckInput{Address: "localhost"})
	if err == nil || err.Kind != domain.ErrorTimeout || result.Status != domain.StatusOffline {
		t.Fatalf("result=%+v error=%#v", result, err)
	}
}

func TestCheckServiceParentCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewCheckService(checkerFunc(func(ctx context.Context, _ domain.Target) (domain.Result, error) {
		return domain.Result{}, ctx.Err()
	}), time.Second)
	result, err := service.Check(ctx, CheckInput{Address: "localhost"})
	if err == nil || err.Kind != domain.ErrorCancelled || result.Status != domain.StatusUnknown {
		t.Fatalf("result=%+v error=%#v", result, err)
	}
}

func TestCheckAsyncAssignsUniqueRequestIDs(t *testing.T) {
	t.Parallel()
	service := NewCheckService(checkerFunc(func(_ context.Context, target domain.Target) (domain.Result, error) {
		return domain.Result{Target: target, Status: domain.StatusOnline}, nil
	}), time.Second)

	const requests = 32
	ids := make(chan uint64, requests)
	var wait sync.WaitGroup
	for i := 0; i < requests; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			id, outcomes := service.CheckAsync(context.Background(), CheckInput{Address: "localhost"})
			outcome := <-outcomes
			if outcome.RequestID != id || outcome.Error != nil {
				t.Errorf("outcome=%+v id=%d", outcome, id)
			}
			ids <- id
		}()
	}
	wait.Wait()
	close(ids)
	seen := make(map[uint64]bool, requests)
	for id := range ids {
		if id == 0 || seen[id] {
			t.Fatalf("invalid or duplicate request ID: %d", id)
		}
		seen[id] = true
	}
}

func TestNewCheckServiceClampsTimeout(t *testing.T) {
	t.Parallel()
	checker := checkerFunc(func(context.Context, domain.Target) (domain.Result, error) {
		return domain.Result{}, errors.New("unused")
	})
	if got := NewCheckService(checker, 0).timeout; got != DefaultTimeout {
		t.Fatalf("default timeout = %v", got)
	}
	if got := NewCheckService(checker, time.Hour).timeout; got != MaxTimeout {
		t.Fatalf("max timeout = %v", got)
	}
}

type checkerFunc func(context.Context, domain.Target) (domain.Result, error)

func (f checkerFunc) Check(ctx context.Context, target domain.Target) (domain.Result, error) {
	return f(ctx, target)
}
