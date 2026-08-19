package ui

import (
	"context"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	appsvc "mc-server-checker/internal/app"
	"mc-server-checker/internal/domain"
	"mc-server-checker/internal/platform"
	"mc-server-checker/internal/storage"
)

func TestMainWindowInitialStateAndSuccessfulCheck(t *testing.T) {
	application := test.NewApp()
	checker := uiCheckerFunc(func(_ context.Context, target domain.Target) (domain.Result, error) {
		latency := 10 * time.Millisecond
		online, max := 1, 10
		return domain.Result{
			Target: target, Status: domain.StatusOnline, Latency: &latency,
			VersionName: "Fake", PlayersOnline: &online, PlayersMax: &max,
			MOTD: "Test Server", CheckedAt: time.Now(),
		}, nil
	})
	service := appsvc.NewCheckService(checker, time.Second)
	preferences := storage.NewPreferences(application.Preferences())
	window := NewMainWindow(application, service, preferences, platform.BuildInfo{Version: "test"})
	defer window.rootCancel()
	dispatcher := &serialDispatcher{}
	window.dispatch = dispatcher.Do
	outcomeApplied := make(chan struct{}, 1)
	window.afterOutcome = func() { outcomeApplied <- struct{}{} }

	if window.addressEntry.Text != "" || window.portEntry.Text != "" || !window.checkButton.Disabled() || window.statusLabel.Text != "○ 未確認" {
		t.Fatalf("unexpected initial state: address=%q port=%q disabled=%v status=%q", window.addressEntry.Text, window.portEntry.Text, window.checkButton.Disabled(), window.statusLabel.Text)
	}
	dispatcher.Do(func() { window.addressEntry.SetText("localhost") })
	if window.checkButton.Disabled() {
		t.Fatal("check button stayed disabled for valid input")
	}
	dispatcher.Do(window.startCheck)
	select {
	case <-outcomeApplied:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for outcome")
	}
	dispatcher.Do(func() {
		if window.statusLabel.Text != "● ONLINE" || window.pingLabel.Text != "10 ms" || window.motdLabel.Text != "Test Server" || window.copyButton.Disabled() {
			t.Fatalf("unexpected completed state: status=%q ping=%q motd=%q copyDisabled=%v", window.statusLabel.Text, window.pingLabel.Text, window.motdLabel.Text, window.copyButton.Disabled())
		}
		window.copyResult()
	})
	if got := application.Clipboard().Content(); got == "" {
		t.Fatal("clipboard is empty")
	}
	address, port := preferences.Load()
	if address != "localhost" || port != "25565" {
		t.Fatalf("saved preferences = %q, %q", address, port)
	}
}

func TestMainWindowIgnoresCancelledStaleResult(t *testing.T) {
	application := test.NewApp()
	release := make(chan struct{})
	checker := uiCheckerFunc(func(ctx context.Context, target domain.Target) (domain.Result, error) {
		select {
		case <-release:
			return domain.Result{Target: target, Status: domain.StatusOnline, MOTD: "stale"}, nil
		case <-ctx.Done():
			return domain.Result{}, ctx.Err()
		}
	})
	window := NewMainWindow(application, appsvc.NewCheckService(checker, time.Second), storage.NewPreferences(application.Preferences()), platform.BuildInfo{})
	defer window.rootCancel()
	dispatcher := &serialDispatcher{}
	window.dispatch = dispatcher.Do
	outcomeApplied := make(chan struct{}, 1)
	window.afterOutcome = func() { outcomeApplied <- struct{}{} }
	dispatcher.Do(func() {
		window.addressEntry.SetText("localhost")
		window.startCheck()
		window.addressEntry.SetText("example.com")
	})
	close(release)
	select {
	case <-outcomeApplied:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stale outcome")
	}
	dispatcher.Do(func() {
		if window.statusLabel.Text != "○ 未確認" || window.motdLabel.Text == "stale" {
			t.Fatalf("stale result was applied: status=%q motd=%q", window.statusLabel.Text, window.motdLabel.Text)
		}
	})
}

func TestMainWindowShowsResolvedTargetOnlyWhenPresent(t *testing.T) {
	application := test.NewApp()
	window := NewMainWindow(application, nil, storage.NewPreferences(application.Preferences()), platform.BuildInfo{})
	defer window.rootCancel()

	window.applyView(ResultView{Status: domain.StatusOnline, Resolved: "backend.example.net:25570", ShowResolved: true})
	if !window.resolvedRow.Visible() || window.resolvedLabel.Text != "backend.example.net:25570" {
		t.Fatalf("resolved row: visible=%v text=%q", window.resolvedRow.Visible(), window.resolvedLabel.Text)
	}

	window.applyView(InitialResultView())
	if window.resolvedRow.Visible() || window.resolvedLabel.Text != "" {
		t.Fatalf("resolved row remained visible: visible=%v text=%q", window.resolvedRow.Visible(), window.resolvedLabel.Text)
	}
}

type uiCheckerFunc func(context.Context, domain.Target) (domain.Result, error)

func (f uiCheckerFunc) Check(ctx context.Context, target domain.Target) (domain.Result, error) {
	return f(ctx, target)
}

type serialDispatcher struct{ mutex sync.Mutex }

func (d *serialDispatcher) Do(callback func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	callback()
}
