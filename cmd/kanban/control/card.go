package control

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/util"
)

// Card lays the content out with a title for context.
type Card struct {
	Title string
}

func (c Card) Layout(gtx C, th *material.Theme, w layout.Widget) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			return util.Rect{
				Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
				Size:  layout.FPt(gtx.Constraints.Min),
				Radii: 4,
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			inset := layout.UniformInset(unit.Dp(10))
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Stack{}.Layout(
						gtx,
						layout.Expanded(func(gtx C) D {
							return util.Rect{
								Color: color.NRGBA{A: 100},
								Size: f32.Point{
									X: float32(gtx.Constraints.Max.X),
									Y: float32(gtx.Constraints.Min.Y),
								},
							}.Layout(gtx)
						}),
						layout.Stacked(func(gtx C) D {
							return inset.Layout(gtx, func(gtx C) D {
								return material.H6(th, c.Title).Layout(gtx)
							})
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return inset.Layout(gtx, func(gtx C) D {
						return w(gtx)
					})
				}),
			)
		}),
	)
}
