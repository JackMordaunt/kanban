package control

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/util"
	"git.sr.ht/~jackmordaunt/kanban/icons"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Panel can hold cards.
// One panel per stage in the kanban pipeline.
// Has a title and action bar.
type Panel struct {
	Label        string
	Color        color.NRGBA
	Thickness    unit.Value
	CreateTicket widget.Clickable

	layout.List
}

func (p *Panel) Layout(gtx C, th *material.Theme, tickets ...layout.ListElement) D {
	return widget.Border{
		Color: color.NRGBA{A: 200},
		Width: unit.Dp(0.5),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(
			gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Stack{}.Layout(
					gtx,
					layout.Expanded(func(gtx C) D {
						return util.Rect{
							Size: f32.Point{
								X: layout.FPt(gtx.Constraints.Max).X,
								Y: float32(gtx.Px(p.Thickness)),
							},
							Color: p.Color,
						}.Layout(gtx)
					}),
					layout.Stacked(func(gtx C) D {
						return layout.Inset{
							Left:  unit.Dp(10),
							Right: unit.Dp(15),
							Top:   unit.Dp(12),
						}.Layout(gtx, func(gtx C) D {
							return layout.Flex{
								Axis:      layout.Horizontal,
								Alignment: layout.Middle,
							}.Layout(
								gtx,
								layout.Rigid(func(gtx C) D {
									return material.H6(th, p.Label).Layout(gtx)
								}),
								layout.Flexed(1, func(gtx C) D {
									return D{Size: gtx.Constraints.Min}
								}),
								layout.Rigid(func(gtx C) D {
									return util.Button(
										&p.CreateTicket,
										util.WithIcon(icons.ContentAdd),
										util.WithSize(unit.Dp(15)),
										util.WithInset(layout.UniformInset(unit.Dp(6))),
										util.WithBgColor(color.NRGBA{}),
										util.WithIconColor(th.Fg),
									).Layout(gtx)
								}),
							)
						})
					}),
				)
			}),
			layout.Flexed(1, func(gtx C) D {
				return layout.Stack{}.Layout(
					gtx,
					layout.Expanded(func(gtx C) D {
						return util.Rect{
							Color: color.NRGBA{R: 240, G: 240, B: 240, A: 255},
							Size:  layout.FPt(gtx.Constraints.Max),
						}.Layout(gtx)
					}),
					layout.Stacked(func(gtx C) D {
						p.List.Axis = layout.Vertical
						return p.List.Layout(gtx, len(tickets), func(gtx C, ii int) D {
							return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx C) D {
								return tickets[ii](gtx, ii)
							})

						})
					}),
				)
			}),
		)
	})
}
