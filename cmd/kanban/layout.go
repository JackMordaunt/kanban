package main

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Card lays the content out with a title for context.
type Card struct {
	Title string
}

func (c Card) Layout(gtx C, th *material.Theme, w layout.Widget) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			return Rect{
				Color: color.RGBA{R: 255, G: 255, B: 255, A: 255},
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
							return Rect{
								Color: color.RGBA{A: 100},
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

// Modal renders content centered on a translucent scrim.
func Modal(gtx C, w layout.Widget) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			return Rect{
				Size:  layout.FPt(gtx.Constraints.Max),
				Color: color.RGBA{A: 200},
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
