package theme

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

// Action is a customizable interactive.
type Action struct {
	*widget.Clickable
	Text  string
	Color color.NRGBA
	Bg    color.NRGBA
}

// DialogStyle describes a dialog widget.
type DialogStyle struct {
	Title    layout.Widget
	SubTitle layout.Widget
	Body     layout.Widget
	Actions  []layout.Widget

	Background color.NRGBA
}

// Dialog constructs a card widget.
func (t *Theme) Dialog(header, subheader string, body layout.Widget, actions ...Action) DialogStyle {
	var acts []layout.Widget
	for ii := range actions {
		a := actions[ii]
		acts = append(acts, func(gtx C) D {
			btn := material.Button(&t.Theme, a.Clickable, a.Text)
			btn.Text = strings.ToUpper(btn.Text)
			btn.Color = a.Color
			btn.Background = a.Bg
			return btn.Layout(gtx)
		})
	}
	return DialogStyle{
		Title:      material.Label(&t.Theme, unit.Dp(24), header).Layout,
		SubTitle:   material.Label(&t.Theme, unit.Dp(14), subheader).Layout,
		Body:       body,
		Background: t.Background,
		Actions:    acts,
	}
}

func (c DialogStyle) Layout(gtx C) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			return component.Rect{
				Color: c.Background,
				Size:  gtx.Constraints.Min,
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(
					gtx,
					layout.Rigid(func(gtx C) D {
						return c.Title(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return c.SubTitle(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx C) D {
						return c.Body(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx C) D {
						var actions []layout.FlexChild
						actions = append(actions, layout.Flexed(1, func(gtx C) D {
							return D{Size: image.Point{X: gtx.Constraints.Min.X}}
						}))
						for ii := range c.Actions {
							ii := ii
							actions = append(actions, layout.Rigid(c.Actions[ii]))
						}
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Middle,
						}.Layout(
							gtx,
							actions...,
						)
					}),
				)
			})
		}),
	)
}
