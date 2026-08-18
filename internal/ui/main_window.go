package ui

import (
	"context"
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	appsvc "mc-server-checker/internal/app"
	"mc-server-checker/internal/domain"
	"mc-server-checker/internal/platform"
	"mc-server-checker/internal/storage"
)

type MainWindow struct {
	window       fyne.Window
	clipboard    fyne.Clipboard
	service      *appsvc.CheckService
	preferences  *storage.Preferences
	rootCtx      context.Context
	rootCancel   context.CancelFunc
	checkCancel  context.CancelFunc
	dispatch     func(func())
	afterOutcome func()

	addressEntry *widget.Entry
	portEntry    *widget.Entry
	checkButton  *widget.Button
	copyButton   *widget.Button
	activity     *widget.Activity
	statusLabel  *widget.Label
	checkedLabel *widget.Label
	pingLabel    *widget.Label
	versionLabel *widget.Label
	playersLabel *widget.Label
	motdLabel    *widget.Label
	modTitle     *widget.Label
	modLabel     *widget.Label
	modRow       *fyne.Container
	messageLabel *widget.Label

	currentRequestID uint64
	checking         bool
	view             ResultView
}

func NewMainWindow(application fyne.App, service *appsvc.CheckService, preferences *storage.Preferences, build platform.BuildInfo) *MainWindow {
	window := application.NewWindow("Minecraft Server Checker")
	rootCtx, rootCancel := context.WithCancel(context.Background())
	m := &MainWindow{
		window:      window,
		clipboard:   application.Clipboard(),
		service:     service,
		preferences: preferences,
		rootCtx:     rootCtx,
		rootCancel:  rootCancel,
		dispatch:    fyne.Do,
		view:        InitialResultView(),
	}
	m.buildContent(build)
	m.installCloseHandler()
	window.Resize(fyne.NewSize(560, 430))
	return m
}

func (m *MainWindow) ShowAndRun() {
	m.window.ShowAndRun()
}

func (m *MainWindow) Window() fyne.Window {
	return m.window
}

func (m *MainWindow) buildContent(build platform.BuildInfo) {
	address, port := m.preferences.Load()
	m.addressEntry = widget.NewEntry()
	m.addressEntry.SetPlaceHolder("play.example.com")
	m.addressEntry.SetText(address)
	m.portEntry = widget.NewEntry()
	m.portEntry.SetPlaceHolder("25565")
	m.portEntry.SetText(port)

	m.checkButton = widget.NewButton("確認", m.startCheck)
	m.checkButton.Importance = widget.HighImportance
	m.copyButton = widget.NewButtonWithIcon("結果をコピー", theme.ContentCopyIcon(), m.copyResult)
	m.copyButton.Disable()
	m.activity = widget.NewActivity()
	m.activity.Hide()

	m.statusLabel = widget.NewLabelWithStyle(m.view.StatusText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	m.checkedLabel = widget.NewLabel(m.view.CheckedAt)
	m.pingLabel = widget.NewLabel(m.view.Ping)
	m.versionLabel = widget.NewLabel(m.view.Version)
	m.playersLabel = widget.NewLabel(m.view.Players)
	m.motdLabel = widget.NewLabel(m.view.MOTD)
	m.motdLabel.Wrapping = fyne.TextWrapWord
	m.modLabel = widget.NewLabel("")
	m.messageLabel = widget.NewLabel("")
	m.messageLabel.Wrapping = fyne.TextWrapWord
	m.messageLabel.Importance = widget.WarningImportance
	m.messageLabel.Hide()

	addressRow := container.NewBorder(nil, nil, widget.NewLabel("Address"), nil, m.addressEntry)
	portRow := container.NewBorder(nil, nil, widget.NewLabel("Port"), m.checkButton, m.portEntry)
	input := container.NewVBox(addressRow, portRow)

	statusRow := container.NewHBox(m.statusLabel, m.activity)
	checkedRow := container.NewBorder(nil, nil, widget.NewLabel("最終確認"), nil, m.checkedLabel)
	detailRow := func(label *widget.Label, value fyne.CanvasObject) *fyne.Container {
		labelWidth := canvas.NewRectangle(color.Transparent)
		labelWidth.SetMinSize(fyne.NewSize(72, label.MinSize().Height))
		return container.NewBorder(nil, nil, container.NewStack(labelWidth, label), nil, value)
	}
	m.modTitle = widget.NewLabel("")
	m.modRow = detailRow(m.modTitle, m.modLabel)
	resultGrid := container.NewVBox(
		detailRow(widget.NewLabel("Ping"), m.pingLabel),
		detailRow(widget.NewLabel("Version"), m.versionLabel),
		detailRow(widget.NewLabel("Players"), m.playersLabel),
		detailRow(widget.NewLabel("MOTD"), m.motdLabel),
		m.modRow,
	)
	result := container.NewVBox(statusRow, checkedRow, widget.NewSeparator(), resultGrid, m.messageLabel)

	version := build.Version
	if version == "" {
		version = "dev"
	}
	footer := container.NewBorder(nil, nil, m.copyButton, widget.NewLabel("v"+version))
	content := container.NewBorder(input, footer, nil, nil, container.NewVScroll(result))
	minimum := canvas.NewRectangle(color.Transparent)
	minimum.SetMinSize(fyne.NewSize(520, 380))
	m.window.SetContent(container.NewStack(minimum, container.NewPadded(content)))

	m.addressEntry.OnSubmitted = func(string) { m.startCheck() }
	m.portEntry.OnSubmitted = func(string) { m.startCheck() }
	m.addressEntry.OnChanged = func(string) { m.inputChanged() }
	m.portEntry.OnChanged = func(string) { m.inputChanged() }
	m.updateCheckButton()
}

func (m *MainWindow) installCloseHandler() {
	m.window.SetCloseIntercept(func() {
		m.rootCancel()
		if m.checkCancel != nil {
			m.checkCancel()
		}
		m.window.SetCloseIntercept(nil)
		m.window.Close()
	})
}

func (m *MainWindow) startCheck() {
	if m.checking {
		return
	}
	input := appsvc.CheckInput{Address: m.addressEntry.Text, Port: m.portEntry.Text}
	if _, err := appsvc.NormalizeTarget(input.Address, input.Port); err != nil {
		m.applyView(CompletedResultView(domain.Result{Status: domain.StatusError}, err))
		return
	}

	checkCtx, cancel := context.WithCancel(m.rootCtx)
	m.checkCancel = cancel
	m.checking = true
	m.view = CheckingResultView(m.view)
	m.applyView(m.view)
	m.checkButton.SetText("確認中…")
	m.checkButton.Disable()
	m.activity.Show()
	m.activity.Start()

	requestID, outcomes := m.service.CheckAsync(checkCtx, input)
	m.currentRequestID = requestID
	go func() {
		outcome, ok := <-outcomes
		if !ok {
			return
		}
		m.dispatch(func() { m.applyOutcome(outcome) })
	}()
}

func (m *MainWindow) applyOutcome(outcome appsvc.CheckOutcome) {
	if m.afterOutcome != nil {
		defer m.afterOutcome()
	}
	if outcome.RequestID != m.currentRequestID {
		return
	}
	m.checking = false
	m.currentRequestID = 0
	if m.checkCancel != nil {
		m.checkCancel()
		m.checkCancel = nil
	}
	m.activity.Stop()
	m.activity.Hide()
	m.checkButton.SetText("確認")
	m.updateCheckButton()

	result := outcome.Result
	if result.CheckedAt.IsZero() && result.Target.Host != "" {
		result.CheckedAt = time.Now()
	}
	m.view = CompletedResultView(result, outcome.Error)
	m.applyView(m.view)
	if outcome.Error == nil && result.Status == domain.StatusOnline {
		m.preferences.Save(result.Target)
	}
}

func (m *MainWindow) inputChanged() {
	if m.checking {
		if m.checkCancel != nil {
			m.checkCancel()
			m.checkCancel = nil
		}
		m.currentRequestID = 0
		m.checking = false
		m.activity.Stop()
		m.activity.Hide()
		m.checkButton.SetText("確認")
	}
	m.view = PreviousResultView(m.view)
	m.applyView(m.view)
	m.updateCheckButton()
}

func (m *MainWindow) updateCheckButton() {
	if m.checking {
		m.checkButton.Disable()
		return
	}
	_, err := appsvc.NormalizeTarget(m.addressEntry.Text, m.portEntry.Text)
	if err != nil {
		m.checkButton.Disable()
	} else {
		m.checkButton.Enable()
	}
}

func (m *MainWindow) applyView(view ResultView) {
	m.statusLabel.SetText(view.StatusText)
	switch view.Status {
	case domain.StatusOnline:
		m.statusLabel.Importance = widget.SuccessImportance
	case domain.StatusOffline:
		m.statusLabel.Importance = widget.DangerImportance
	case domain.StatusError:
		m.statusLabel.Importance = widget.WarningImportance
	default:
		m.statusLabel.Importance = widget.MediumImportance
	}
	m.statusLabel.Refresh()
	m.checkedLabel.SetText(view.CheckedAt)
	m.pingLabel.SetText(view.Ping)
	m.versionLabel.SetText(view.Version)
	m.playersLabel.SetText(view.Players)
	m.motdLabel.SetText(view.MOTD)
	if view.ShowMod {
		m.modTitle.SetText("MOD")
		m.modLabel.SetText(view.Mod)
	} else {
		m.modTitle.SetText("")
		m.modLabel.SetText("")
	}
	if view.ShowMessage {
		m.messageLabel.SetText(view.Message)
		m.messageLabel.Show()
	} else {
		m.messageLabel.Hide()
	}
	if view.CopyText == "" {
		m.copyButton.Disable()
	} else {
		m.copyButton.Enable()
	}
}

func (m *MainWindow) copyResult() {
	if m.view.CopyText == "" {
		return
	}
	m.clipboard.SetContent(m.view.CopyText)
	m.messageLabel.SetText("結果をクリップボードへコピーしました。")
	m.messageLabel.Show()
}

func (m *MainWindow) String() string {
	return fmt.Sprintf("Minecraft Server Checker (%s)", m.view.StatusText)
}
