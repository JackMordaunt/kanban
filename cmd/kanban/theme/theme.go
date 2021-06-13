// Package theme specifies themed widget types.
package theme

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget/material"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Theme acts as a widget factory and captures all styling concerns.
type Theme struct {
	// Inherit from material theme because Gio widgets are coupled to it.
	material.Theme
	// Palette contains the semantic colors for this theme.
	Palette
}

// Palette specifies semantic colors.
type Palette struct {
	// Primary color displayed most frequently across screens and components.
	// Up to two optional variants.
	Primary  color.NRGBA
	Primary2 color.NRGBA
	Primary3 color.NRGBA

	// Secondary color used sparingly to accent ui elements.
	// Up to two optional variants.
	Secondary  color.NRGBA
	Secondary2 color.NRGBA
	Secondary3 color.NRGBA

	// Surface affects surfaces of components such as cards, sheets and menus.
	Surface color.NRGBA

	// Background appears behind scrollable content.
	Background color.NRGBA

	// Error indicates errors in components.
	Error color.NRGBA

	// On colors appear "on top" of the base color.
	// Choose contrasting colors.
	OnPrimary    color.NRGBA
	OnSecondary  color.NRGBA
	OnBackground color.NRGBA
	OnSurface    color.NRGBA
	OnError      color.NRGBA
}

// NewTheme allocates a theme factory using the given fonts and palette.
func NewTheme(fonts []text.FontFace, p Palette) *Theme {
	return &Theme{
		Theme:   *material.NewTheme(fonts),
		Palette: p,
	}
}

// // BootstrapPalette specifies the standard bootstrap 4 colors.
// var BootstrapPalette Palette = Palette{
// 	Primary: rgb(0x0275d8),
// 	Success: rgb(0x5cb85c),
// 	Info:    rgb(0x5bc0de),
// 	Warning: rgb(0xf0ad4e),
// 	Danger:  rgb(0xd9534f),
// 	Inverse: rgb(0x292b2c),
// 	BgLight: rgb(0xf8f9fa),
// 	BgDark:  rgb(0x343a40),
// 	Text:    color.NRGBA{R: 83, G: 215, B: 202, A: 255},
// }

// MaterialDesignBaseline is the baseline palette for material design.
// https://material.io/design/color/the-color-system.html#color-theme-creation
var MaterialDesignBaseline Palette = Palette{
	Primary:      rgb(0x6200EE),
	Primary2:     rgb(0x3700B3),
	Secondary:    rgb(0x03DAC6),
	Secondary2:   rgb(0x018786),
	Background:   rgb(0xFFFFFF),
	Surface:      rgb(0xFFFFFF),
	Error:        rgb(0xB00020),
	OnPrimary:    rgb(0xFFFFFF),
	OnSecondary:  rgb(0x000000),
	OnBackground: rgb(0x000000),
	OnSurface:    rgb(0x000000),
	OnError:      rgb(0xFFFFFF),
}

func rgb(c uint32) color.NRGBA {
	return argb(0xff000000 | c)
}

func argb(c uint32) color.NRGBA {
	return color.NRGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}
