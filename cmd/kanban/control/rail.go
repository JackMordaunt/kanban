package control

import (
	"image/color"
	"unsafe"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/state"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/util"
)

// Rail is an interactive side rail with a list of widget items that can be
// selected.
//
// Typically used as navigation or contextual actions.
//
// Rail is stateful.
type Rail struct {
	layout.List
	Map state.Map
}

// RailChild is an item that renders in a rail.
type RailChild struct {
	Name string
	W    layout.Widget
}

// Destination is a rail item that represents a navigatable object.
// Destinations are pab+dded by default.
func Destination(name string, w layout.Widget) RailChild {
	return RailChild{
		Name: name,
		W:    w,
	}
}

func (r *Rail) next(key string) *widget.Clickable {
	return (*widget.Clickable)(r.Map.New(key, unsafe.Pointer(&widget.Clickable{})))
}

// Selected reports which rail child was selected, if any.
// Reports the first click encountered.
func (r *Rail) Selected() (string, bool) {
	for r.Map.More() {
		k, v := r.Map.Next()
		if (*widget.Clickable)(v).Clicked() {
			return k, true
		}
	}
	return "", false
}

// Layout the rail with the given items.
func (r *Rail) Layout(gtx C, action layout.Widget, items ...RailChild) D {
	r.List.Axis = layout.Vertical
	r.List.Alignment = layout.Middle
	r.Map.Begin()
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			return action(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx C) D {
				return util.Div{
					Color:     color.NRGBA{R: 220, B: 220, G: 220, A: 255},
					Length:    unit.Px(float32(gtx.Constraints.Max.X)),
					Thickness: unit.Dp(1),
					Axis:      layout.Horizontal,
				}.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx C) D {
			return r.List.Layout(gtx, len(items), func(gtx C, ii int) D {
				rc := items[ii]
				return material.Clickable(gtx, r.next(rc.Name), func(gtx C) D {
					return rc.W(gtx)
				})
			})
		}),
	)
}
