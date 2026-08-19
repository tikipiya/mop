package ui

import (
	"strings"
	"testing"
	"time"

	"mc-server-checker/internal/domain"
)

func TestCompletedResultViewOnline(t *testing.T) {
	t.Parallel()
	latency := 24500 * time.Microsecond
	online, max, protocol, mods := 4, 20, 776, 12
	result := domain.Result{
		Target:          domain.Target{Host: "play.example.com", Port: 25565},
		Status:          domain.StatusOnline,
		Latency:         &latency,
		VersionName:     "Paper 26.2",
		Protocol:        &protocol,
		PlayersOnline:   &online,
		PlayersMax:      &max,
		MOTD:            "Example Server",
		CheckedAt:       time.Date(2026, 8, 19, 12, 34, 56, 0, time.Local),
		ModInfoDetected: true,
		ModLoader:       "NeoForge",
		ModCount:        &mods,
	}
	view := CompletedResultView(result, nil)
	if view.StatusText != "● ONLINE" || view.Ping != "25 ms" || view.Players != "4 / 20" || view.Mod != "検出（NeoForge、12 mods）" || !view.ShowMod {
		t.Fatalf("view = %+v", view)
	}
	for _, want := range []string{"Target: play.example.com:25565", "Status: ONLINE", "Ping: 25 ms", "MOTD: Example Server"} {
		if !strings.Contains(view.CopyText, want) {
			t.Fatalf("copy text %q does not contain %q", view.CopyText, want)
		}
	}
	if !strings.HasSuffix(view.CopyText, "Developed by: @tikipiya (tikisan)") {
		t.Fatalf("copy credit is not the final line: %q", view.CopyText)
	}
	if strings.Contains(view.CopyText, "Resolved:") || view.ShowResolved {
		t.Fatalf("direct connection unexpectedly shows resolved target: %+v", view)
	}
}

func TestCompletedResultViewShowsSRVEndpoint(t *testing.T) {
	t.Parallel()
	resolved := domain.Target{Host: "backend.example.net", Port: 25570}
	view := CompletedResultView(domain.Result{
		Target:         domain.Target{Host: "play.example.com", Port: 25565, UseSRV: true},
		ResolvedTarget: &resolved,
		Status:         domain.StatusOnline,
	}, nil)

	if !view.ShowResolved || view.Resolved != "backend.example.net:25570" {
		t.Fatalf("view = %+v", view)
	}
	want := "Target: play.example.com\nResolved: backend.example.net:25570\nStatus: ONLINE"
	if !strings.Contains(view.CopyText, want) {
		t.Fatalf("copy text %q does not contain %q", view.CopyText, want)
	}
}

func TestCompletedResultViewErrorAndMissingValues(t *testing.T) {
	t.Parallel()
	appError := &domain.AppError{Kind: domain.ErrorTimeout, Message: "時間内に応答がありませんでした。"}
	view := CompletedResultView(domain.Result{
		Target:    domain.Target{Host: "localhost", Port: 25565},
		Status:    domain.StatusOffline,
		CheckedAt: time.Now(),
	}, appError)
	if view.StatusText != "● OFFLINE" || view.Version != "不明" || view.Ping != unavailable || view.Players != unavailable || !view.ShowMessage || view.Message != appError.Message {
		t.Fatalf("view = %+v", view)
	}
}

func TestCheckingAndPreviousViewsKeepOldValues(t *testing.T) {
	t.Parallel()
	previous := ResultView{Status: domain.StatusOnline, StatusText: "● ONLINE", MOTD: "old", HasResult: true, CopyText: "copy"}
	checking := CheckingResultView(previous)
	if checking.Status != domain.StatusChecking || checking.MOTD != "old" || !checking.ShowMessage {
		t.Fatalf("checking = %+v", checking)
	}
	changed := PreviousResultView(checking)
	if changed.Status != domain.StatusUnknown || changed.StatusText != "○ 未確認（前回結果）" || changed.MOTD != "old" || changed.ShowMessage {
		t.Fatalf("changed = %+v", changed)
	}
}

func TestWarningWithoutLatencyShowsPingFailure(t *testing.T) {
	t.Parallel()
	view := CompletedResultView(domain.Result{
		Target:  domain.Target{Host: "localhost", Port: 25565},
		Status:  domain.StatusOnline,
		Warning: "ping failed",
	}, nil)
	if view.Ping != "取得失敗" || view.Message != "ping failed" {
		t.Fatalf("view = %+v", view)
	}
}
