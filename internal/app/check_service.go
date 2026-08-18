package app

import (
	"context"
	"sync/atomic"
	"time"

	"mc-server-checker/internal/domain"
)

const (
	DefaultTimeout = 5 * time.Second
	MaxTimeout     = 30 * time.Second
)

type Checker interface {
	Check(ctx context.Context, target domain.Target) (domain.Result, error)
}

type CheckService struct {
	checker Checker
	timeout time.Duration
	nextID  atomic.Uint64
}

type CheckInput struct {
	Address string
	Port    string
}

type CheckOutcome struct {
	RequestID uint64
	Result    domain.Result
	Error     *domain.AppError
}

func NewCheckService(checker Checker, timeout time.Duration) *CheckService {
	if timeout <= 0 {
		timeout = DefaultTimeout
	} else if timeout > MaxTimeout {
		timeout = MaxTimeout
	}
	return &CheckService{checker: checker, timeout: timeout}
}

func (s *CheckService) Check(ctx context.Context, input CheckInput) (domain.Result, *domain.AppError) {
	target, validationErr := NormalizeTarget(input.Address, input.Port)
	if validationErr != nil {
		return domain.Result{Status: domain.StatusError}, validationErr
	}

	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result, err := s.checker.Check(checkCtx, target)
	if err != nil {
		appError := MapError(checkCtx, err)
		return domain.Result{Target: target, Status: statusForError(appError)}, appError
	}
	if result.Target.Host == "" {
		result.Target = target
	}
	if result.Status == "" || result.Status == domain.StatusUnknown {
		result.Status = domain.StatusOnline
	}
	return result, nil
}

// CheckAsync runs one check without blocking the caller. The buffered result
// channel ensures the worker can always finish even if a window is closed.
func (s *CheckService) CheckAsync(ctx context.Context, input CheckInput) (uint64, <-chan CheckOutcome) {
	requestID := s.nextID.Add(1)
	outcomes := make(chan CheckOutcome, 1)
	go func() {
		defer close(outcomes)
		result, err := s.Check(ctx, input)
		outcomes <- CheckOutcome{RequestID: requestID, Result: result, Error: err}
	}()
	return requestID, outcomes
}
