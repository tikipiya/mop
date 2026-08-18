package assets

import (
	_ "embed"
	"sync"

	"fyne.io/fyne/v2"
)

var (
	//go:embed Icon.png
	iconData []byte

	iconOnce     sync.Once
	iconResource fyne.Resource
)

func Icon() fyne.Resource {
	iconOnce.Do(func() {
		iconResource = fyne.NewStaticResource("Icon.png", iconData)
	})
	return iconResource
}
