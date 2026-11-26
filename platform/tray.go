package platform

// TrayItem is one entry in a tray menu. It is a seam-level type so the tray
// contract does not depend on the widget layer.
type TrayItem struct {
	Label     string
	ID        int
	Separator bool
	Checked   bool
	Disabled  bool
	Children  []TrayItem
}

// Tray is a system-tray icon. It is desktop-only: App.Tray reports false when
// the backend has no tray support, and mobile backends always report false.
//
// The callbacks are not optional decoration — without them the interface is
// unusable. An app could set a menu and never learn which entry was chosen, or
// that the icon was clicked at all, so a tray built on the earlier version could
// only ever be a picture. They mirror what the widget-layer implementation this
// was ported from already had.
type Tray interface {
	SetIconPNG(data []byte) error
	SetIconName(name string)
	SetTooltip(text string)
	SetTitle(title string)
	SetMenu(items []TrayItem)
	Show()
	Hide()
	Remove()

	// SetOnActivate fires when the icon itself is clicked.
	SetOnActivate(fn func())
	// SetOnSecondaryActivate fires on a middle/secondary click.
	SetOnSecondaryActivate(fn func())
	// SetOnMenu fires with the ID of the chosen item — TrayItem.ID, which is
	// the app's own identifier and is never rewritten by the backend.
	SetOnMenu(fn func(id int))
	// SetOnScroll fires with the wheel delta and "vertical" or "horizontal".
	SetOnScroll(fn func(delta int, orientation string))
}
