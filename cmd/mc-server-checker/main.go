package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"mc-server-checker/assets"
	appsvc "mc-server-checker/internal/app"
	"mc-server-checker/internal/platform"
	"mc-server-checker/internal/protocol/java"
	"mc-server-checker/internal/storage"
	"mc-server-checker/internal/ui"
)

func main() {
	app.SetMetadata(fyne.AppMetadata{
		ID:         "com.tikipiya.mcserverchecker",
		Name:       "Minecraft Server Checker",
		Version:    platform.Version,
		Build:      1,
		Icon:       assets.Icon(),
		Migrations: map[string]bool{"fyneDo": true},
	})
	application := app.NewWithID("com.tikipiya.mcserverchecker")
	application.SetIcon(assets.Icon())
	client := java.NewClient(java.DefaultProtocolVersion)
	service := appsvc.NewCheckService(client, appsvc.DefaultTimeout)
	preferences := storage.NewPreferences(application.Preferences())
	build := platform.CurrentBuildInfo()
	if metadata := application.Metadata(); build.Version == "dev" && metadata.Version != "" {
		build.Version = metadata.Version
	}
	window := ui.NewMainWindow(application, service, preferences, build)
	window.ShowAndRun()
}
