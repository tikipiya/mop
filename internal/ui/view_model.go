package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"mc-server-checker/internal/domain"
)

const unavailable = "—"

type ResultView struct {
	Status      domain.Status
	StatusText  string
	CheckedAt   string
	Ping        string
	Version     string
	Players     string
	MOTD        string
	Mod         string
	Message     string
	CopyText    string
	ShowMod     bool
	ShowMessage bool
	HasResult   bool
}

func InitialResultView() ResultView {
	return ResultView{
		Status:     domain.StatusUnknown,
		StatusText: "○ 未確認",
		CheckedAt:  unavailable,
		Ping:       unavailable,
		Version:    unavailable,
		Players:    unavailable,
		MOTD:       unavailable,
	}
}

func CheckingResultView(previous ResultView) ResultView {
	previous.Status = domain.StatusChecking
	previous.StatusText = "◌ 確認中…"
	previous.Message = "サーバーへ問い合わせています。"
	previous.ShowMessage = true
	return previous
}

func PreviousResultView(previous ResultView) ResultView {
	previous.Status = domain.StatusUnknown
	if previous.HasResult {
		previous.StatusText = "○ 未確認（前回結果）"
	} else {
		previous.StatusText = "○ 未確認"
	}
	previous.Message = ""
	previous.ShowMessage = false
	return previous
}

func CompletedResultView(result domain.Result, appError *domain.AppError) ResultView {
	view := InitialResultView()
	view.Status = result.Status
	view.HasResult = result.Target.Host != ""
	view.CheckedAt = formatCheckedAt(result.CheckedAt)
	view.Ping = formatLatency(result.Latency, result.Warning)
	view.Version = valueOrUnknown(result.VersionName)
	view.Players = formatPlayers(result.PlayersOnline, result.PlayersMax)
	view.MOTD = valueOrDash(result.MOTD)
	view.Mod, view.ShowMod = formatMod(result)

	switch result.Status {
	case domain.StatusOnline:
		view.StatusText = "● ONLINE"
	case domain.StatusOffline:
		view.StatusText = "● OFFLINE"
	case domain.StatusError:
		view.StatusText = "▲ ERROR"
	default:
		view.StatusText = "○ 未確認"
	}
	if appError != nil {
		view.Message = appError.Message
		view.ShowMessage = true
	} else if result.Warning != "" {
		view.Message = result.Warning
		view.ShowMessage = true
	} else if result.ModInfoWarning != "" {
		view.Message = result.ModInfoWarning
		view.ShowMessage = true
	}
	view.CopyText = formatCopyText(result, view)
	return view
}

func formatCheckedAt(value time.Time) string {
	if value.IsZero() {
		return unavailable
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func formatLatency(value *time.Duration, warning string) string {
	if value == nil {
		if warning != "" {
			return "取得失敗"
		}
		return unavailable
	}
	milliseconds := math.Round(float64(*value) / float64(time.Millisecond))
	if milliseconds < 0 {
		milliseconds = 0
	}
	return fmt.Sprintf("%.0f ms", milliseconds)
}

func formatPlayers(online, max *int) string {
	switch {
	case online != nil && max != nil:
		return fmt.Sprintf("%d / %d", *online, *max)
	case online != nil:
		return fmt.Sprintf("%d", *online)
	case max != nil:
		return fmt.Sprintf("最大 %d", *max)
	default:
		return unavailable
	}
}

func formatMod(result domain.Result) (string, bool) {
	if !result.ModInfoDetected {
		return "", false
	}
	loader := strings.TrimSpace(result.ModLoader)
	if loader == "" {
		loader = "Loader不明"
	}
	if result.ModCount != nil {
		return fmt.Sprintf("検出（%s、%d mods）", loader, *result.ModCount), true
	}
	return fmt.Sprintf("検出（%s）", loader), true
}

func formatCopyText(result domain.Result, view ResultView) string {
	if result.Target.Host == "" {
		return ""
	}
	lines := []string{
		"Target: " + result.Target.Address(),
		"Status: " + string(result.Status),
		"Checked: " + view.CheckedAt,
		"Ping: " + view.Ping,
		"Version: " + view.Version,
		"Players: " + view.Players,
		"MOTD: " + view.MOTD,
	}
	if view.ShowMod {
		lines = append(lines, "MOD: "+view.Mod)
	}
	if view.ShowMessage {
		lines = append(lines, "Note: "+view.Message)
	}
	lines = append(lines, "Developed by: @tikipiya (tikisan)")
	return strings.Join(lines, "\n")
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "不明"
	}
	return value
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return unavailable
	}
	return value
}
