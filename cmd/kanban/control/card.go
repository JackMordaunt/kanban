package control

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/util"
)

// Card implements "https://material.io/components/cards".
type Card struct {
	// Media    image.Image
	Title    string
	Subtitle string
	Body     layout.Widget
	Actions  []Action
}

type Action struct {
	*widget.Clickable
	Label string
	Fg    color.NRGBA
	Bg    color.NRGBA
}

func (c Card) Layout(gtx C, th *material.Theme) D {
	// @cleanup: spacing strategy is adhoc.
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
			return layout.Inset{
				Bottom: unit.Dp(20),
				Left:   unit.Dp(15),
				Right:  unit.Dp(15),
			}.Layout(gtx, func(gtx C) D {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(
					gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Inset{
							Top:    unit.Dp(20),
							Bottom: unit.Dp(20),
						}.Layout(gtx, func(gtx C) D {
							return layout.Flex{
								Axis: layout.Vertical,
							}.Layout(
								gtx,
								layout.Rigid(func(gtx C) D {
									return material.H5(th, c.Title).Layout(gtx)
								}),
								layout.Rigid(func(gtx C) D {
									if c.Subtitle == "" {
										return D{}
									}
									return D{Size: image.Point{Y: gtx.Px(unit.Dp(10))}}
								}),
								layout.Rigid(func(gtx C) D {
									if c.Subtitle == "" {
										return D{}
									}
									return material.Body1(th, c.Subtitle).Layout(gtx)
								}),
							)
						})
					}),
					layout.Rigid(func(gtx C) D {
						if c.Body == nil {
							return D{}
						}
						return c.Body(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return D{Size: image.Point{Y: gtx.Px(unit.Dp(20))}}
					}),
					layout.Rigid(func(gtx C) D {
						if len(c.Actions) < 1 {
							return D{}
						}
						return layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(
							gtx,
							func() (actions []layout.FlexChild) {
								for ii := range c.Actions {
									action := &c.Actions[ii]
									actions = append(actions, layout.Rigid(func(gtx C) D {
										btn := material.Button(th, action.Clickable, action.Label)
										btn.Color = action.Fg
										btn.Background = action.Bg
										return btn.Layout(gtx)
									}))
								}
								return actions
							}()...,
						)
					}),
				)
			})
		}),
	)
}
