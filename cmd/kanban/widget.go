package main

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

// Rect creates a rectangle of the provided background color with
// Dimensions specified by size and a corner radius (on all corners)
// specified by radii.
type Rect struct {
	Color color.RGBA
	Size  f32.Point
	Radii float32
}

// Layout renders the Rect into the provided context
func (r Rect) Layout(gtx C) D {
	paint.FillShape(gtx.Ops, clip.UniformRRect(f32.Rectangle{Max: r.Size}, r.Radii).Op(gtx.Ops), r.Color)
	return layout.Dimensions{Size: image.Pt(int(r.Size.X), int(r.Size.Y))}
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

func WithIconColor(c color.RGBA) ButtonOption {
	return func(btn *ButtonStyle) {
		btn.IconButtonStyle.Color = c
		btn.ButtonStyle.Color = c
	}
}

func WithBgColor(c color.RGBA) ButtonOption {
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

func WithText(txt string) ButtonOption {
	return func(btn *ButtonStyle) {
		btn.Text = txt
	}
}

func WithInset(inset layout.Inset) ButtonOption {
	return func(btn *ButtonStyle) {
		btn.IconButtonStyle.Inset = inset
		btn.ButtonStyle.Inset = inset
	}
}
