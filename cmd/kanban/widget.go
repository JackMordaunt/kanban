package main

import (
	"image"
	"image/color"
	"unsafe"

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

// Rail is an interactive side rail with a list of widget items that can be
// selected.
//
// Typically used as navigation or contextual actions.
//
// Rail is stateful.
type Rail struct {
	layout.List
	Map Map
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
	for k, v := r.Map.Next(); r.Map.More(); k, v = r.Map.Next() {
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
				return Div{
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
