package util

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Rect creates a rectangle of the provided background color with
// Dimensions specified by size and a corner radius (on all corners)
// specified by radii.
type Rect struct {
	Color color.NRGBA
	Size  f32.Point
	Radii float32
}

// Layout renders the Rect into the provided context
func (r Rect) Layout(gtx C) D {
	paint.FillShape(
		gtx.Ops,
		r.Color,
		clip.UniformRRect(
			f32.Rectangle{Max: r.Size},
			r.Radii,
		).Op(gtx.Ops),
	)
	return layout.Dimensions{
		Size: image.Pt(int(r.Size.X), int(r.Size.Y)),
	}
}

// Button renders a clickable button.
func Button(
	state *widget.Clickable,
	opt ...ButtonOption,
) ButtonStyle {
	btn := ButtonStyle{}
	btn.IconButtonStyle.Button = state
	btn.ButtonStyle.Button = state
	for _, opt := range opt {
		opt(&btn)
	}
	return btn
}

// ButtonStyle provides a unified api for both icon buttons and text buttons.
type ButtonStyle struct {
	material.IconButtonStyle
	material.ButtonStyle
}

func (btn ButtonStyle) Layout(gtx C) D {
	if btn.Icon != nil {
		return btn.IconButtonStyle.Layout(gtx)
	}
	return btn.ButtonStyle.Layout(gtx)
}

type ButtonOption func(*ButtonStyle)

func WithSize(sz unit.Value) ButtonOption {
	return func(btn *ButtonStyle) {
		btn.Size = sz
	}
}

func WithIconColor(c color.NRGBA) ButtonOption {
	return func(btn *ButtonStyle) {
		btn.IconButtonStyle.Color = c
		btn.ButtonStyle.Color = c
	}
}

func WithBgColor(c color.NRGBA) ButtonOption {
	return func(btn *ButtonStyle) {
		btn.IconButtonStyle.Background = c
		btn.ButtonStyle.Background = c
	}
}

func WithIcon(icon *widget.Icon) ButtonOption {
	return func(btn *ButtonStyle) {
		btn.Icon = icon
	}
}

// func WithText(txt string) ButtonOption {
// 	return func(btn *ButtonStyle) {
// 		btn.Text = txt
// 	}
// }

func WithInset(inset layout.Inset) ButtonOption {
	return func(btn *ButtonStyle) {
		btn.IconButtonStyle.Inset = inset
		btn.ButtonStyle.Inset = inset
	}
}

// Div is a visual divider: a colored line with a thickness.
type Div struct {
	Thickness unit.Value
	Length    unit.Value
	Axis      layout.Axis
	Color     color.NRGBA
}

func (d Div) Layout(gtx C) D {
	// Draw a line as a very thin rectangle.
	var sz image.Point
	switch d.Axis {
	case layout.Horizontal:
		sz = image.Point{
			X: gtx.Px(d.Length),
			Y: gtx.Px(d.Thickness),
		}
	case layout.Vertical:
		sz = image.Point{
			X: gtx.Px(d.Thickness),
			Y: gtx.Px(d.Length),
		}
	}
	return Rect{
		Color: d.Color,
		Size:  layout.FPt(sz),
		Radii: 0,
	}.Layout(gtx)
}
