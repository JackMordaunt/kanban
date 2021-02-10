package main

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/util"
)

// Modal renders content centered on a translucent scrim.
func Modal(gtx C, w layout.Widget) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			return util.Rect{
				Size:  layout.FPt(gtx.Constraints.Max),
				Color: color.NRGBA{A: 200},
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			return Centered(gtx, func(gtx C) D {
				gtx.Constraints.Max.X = int(float32(gtx.Constraints.Max.X) * 0.8)
				if gtx.Constraints.Max.X > gtx.Px(unit.Dp(600)) {
					gtx.Constraints.Max.X = gtx.Px(unit.Dp(600))
				}
				return w(gtx)
			})
		}),
	)
}

// Centered places the widget in the center of the container.
func Centered(gtx C, w layout.Widget) D {
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(
		gtx,
		layout.Flexed(1, func(gtx C) D {
			return D{Size: gtx.Constraints.Min}
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(
				gtx,
				layout.Flexed(1, func(gtx C) D {
					return D{Size: gtx.Constraints.Min}
				}),
				layout.Rigid(func(gtx C) D {
					return w(gtx)
				}),
				layout.Flexed(1, func(gtx C) D {
					return D{Size: gtx.Constraints.Min}
				}),
			)
		}),
		layout.Flexed(1, func(gtx C) D {
			return D{Size: gtx.Constraints.Min}
		}),
	)
}
