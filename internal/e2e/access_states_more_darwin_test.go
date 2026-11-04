//go:build darwin

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestAccessRoleDescriptionPopupSelected reads, as VoiceOver does, a
// carousel's role description, a button opening a list box, and a
// grid's day chosen without painting it.
func TestAccessRoleDescriptionPopupSelected(t *testing.T) {
	var frames atomic.Int32
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			ui.Column(c).Role(ui.RoleGroup).Label("Highlights").RoleDescription("carousel")
			ui.Button(c, "Fruit").HasPopup(ui.PopupListBox)
			ui.Column(c).Role(ui.RoleTable).Label("November").Children(func() {
				ui.Row(c).Role(ui.RoleRow).Gap(8).Children(func() {
					ui.Text(c, "1").Role(ui.RoleCell).Checked(false)
					ui.Text(c, "2").Role(ui.RoleCell).Checked(true)
				})
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "States", Width: 400, Height: 300, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	for _, c := range []struct{ label, attr, want string }{
		{"Highlights", "AXRoleDescription", "carousel"},
		{"Fruit", "AXPopupValue", "listbox"},
		{"2", "AXSelected", "1"},
		{"1", "AXSelected", "0"},
	} {
		eventually(t, c.label+"'s "+c.attr, func() bool {
			v, _, _, ok := axAttribute(w, c.label, c.attr)
			return ok && v == c.want
		})
	}
}
