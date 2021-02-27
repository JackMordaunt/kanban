package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~jackmordaunt/kanban"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/control"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/util"
	"git.sr.ht/~jackmordaunt/kanban/icons"
	"github.com/google/uuid"
)

// TicketForm renders the form for ticket information.
//
// @Todo use form pattern from avisha.
type TicketForm struct {
	kanban.Ticket
	Stage     string
	Title     component.TextField
	Summary   component.TextField
	Details   component.TextField
	SubmitBtn widget.Clickable
	CancelBtn widget.Clickable
}

func (f *TicketForm) Set(t kanban.Ticket) {
	f.Ticket = t
	f.Title.SetText(t.Title)
	f.Summary.SetText(t.Summary)
	f.Details.SetText(t.Details)
}

// Submit uses form data to create a Ticket.
func (f TicketForm) Submit() kanban.Ticket {
	defer func() {
		f.Ticket = kanban.Ticket{}
	}()
	return kanban.Ticket{
		ID:      f.ID,
		Title:   strings.TrimSpace(f.Title.Text()),
		Summary: f.Summary.Text(),
		Details: f.Details.Text(),
	}
}

func (form *TicketForm) Layout(gtx C, th *material.Theme, stage string) D {
	form.Stage = stage
	form.Title.SingleLine = true
	return control.Card{
		Title: func() string {
			if form.Ticket.ID == uuid.Nil {
				return "Add Ticket"
			}
			return "Edit Ticket"
		}(),
		Body: func(gtx C) D {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return form.Title.Layout(gtx, th, "Title")
				}),
				layout.Rigid(func(gtx C) D {
					return form.Summary.Layout(gtx, th, "Summary")
				}),
				layout.Rigid(func(gtx C) D {
					return form.Details.Layout(gtx, th, "Details")
				}),
			)
		},
		Actions: []control.Action{
			{
				Clickable: &form.SubmitBtn,
				Label:     "Submit",
				Fg:        th.ContrastFg,
				Bg:        th.ContrastBg,
			},
			{
				Clickable: &form.CancelBtn,
				Label:     "Cancel",
				Fg:        th.Fg,
				Bg:        th.Bg,
			},
		},
	}.Layout(gtx, th)
}

// ProjectForm renders a form for manipulating projects.
type ProjectForm struct {
	Name   component.TextField
	Submit widget.Clickable
	Cancel widget.Clickable
}

func (form *ProjectForm) Layout(gtx C, th *material.Theme) D {
	return control.Card{
		Title: "Create a new Project",
		Body: func(gtx C) D {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return form.Name.Layout(gtx, th, "Project Name")
				}),
			)
		},
		Actions: []control.Action{
			{
				Clickable: &form.Submit,
				Label:     "Submit",
				Fg:        th.ContrastFg,
				Bg:        th.ContrastBg,
			},
			{
				Clickable: &form.Cancel,
				Label:     "Cancel",
				Fg:        th.Fg,
				Bg:        th.Bg,
			},
		},
	}.Layout(gtx, th)
}

// DeleteDialog prompts the user with an option to delete a ticket.
type DeleteDialog struct {
	kanban.Ticket
	Ok     widget.Clickable
	Cancel widget.Clickable
}

func (d *DeleteDialog) Layout(gtx C, th *material.Theme) D {
	return control.Card{
		Title: "Are you sure?",
		Body: func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Vertical,
				Alignment: layout.Middle,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return material.Body1(
						th,
						fmt.Sprintf("Delete ticket %q?", d.Title),
					).Layout(gtx)
				}),
			)
		},
		Actions: []control.Action{
			{
				Clickable: &d.Ok,
				Label:     "Delete",
				Fg:        th.ContrastFg,
				Bg:        color.NRGBA{R: 200, A: 255},
			},
			{
				Clickable: &d.Cancel,
				Label:     "Cancel",
				Fg:        th.Fg,
				Bg:        th.Bg,
			},
		},
	}.Layout(gtx, th)
}

// Ticket renders a ticket control.
type Ticket struct {
	kanban.Ticket
	Stage        string
	NextButton   widget.Clickable
	PrevButton   widget.Clickable
	EditButton   widget.Clickable
	DeleteButton widget.Clickable
	Content      widget.Clickable
}

// Layout the ticket card.
//
// The layouting here was actually quite tricky because `layout.List` simulates
// an infinite Y axis. That means you can't just specify a max Y constraint.
// This makes expanding stacked content vertically impossible with a naive use
// of `layout.Stack`.
//
// To get around this I used a macro and manually stacked things sized exactly
// to the content, rather than the maximum Y.
func (t *Ticket) Layout(gtx C, th *material.Theme, focused bool) D {
	var (
		barThickness   = unit.Dp(25)
		sideBarColor   = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
		bottomBarColor = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
		minContentSize = gtx.Px(unit.Dp(150))
	)
	if focused {
		return widget.Border{
			Color: color.NRGBA{B: 200, A: 200},
			Width: unit.Dp(2),
		}.Layout(gtx, func(gtx C) D {
			return t.Layout(gtx, th, false)
		})
	}
	return widget.Border{
		Width: unit.Dp(0.5),
		Color: color.NRGBA{A: 200},
	}.Layout(gtx, func(gtx C) D {
		dims := layout.Inset{
			Left: unit.Dp(25),
		}.Layout(gtx, func(gtx C) D {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Min.Y = minContentSize
					return t.content(gtx, th)
				}),
				layout.Rigid(func(gtx C) D {
					return t.bottomBar(
						gtx,
						th,
						image.Point{
							X: gtx.Constraints.Max.X,
							Y: gtx.Px(barThickness),
						},
						bottomBarColor,
					)
				}),
			)
		})
		t.sideBar(
			gtx,
			image.Point{
				X: gtx.Px(barThickness),
				Y: dims.Size.Y,
			},
			sideBarColor,
		)
		return dims
	})
}

func (t *Ticket) content(gtx C, th *material.Theme) D {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top:    unit.Dp(5),
		Bottom: unit.Dp(5),
		Left:   unit.Dp(10),
		Right:  unit.Dp(10),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(
			gtx,
			layout.Rigid(func(gtx C) D {
				return material.Label(th, unit.Dp(20), t.Title).Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
					l := material.Body1(th, t.Summary)
					l.Color = component.WithAlpha(l.Color, 200)
					return l.Layout(gtx)
				})
			}),
		)
	})
	call := macro.Stop()
	layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			return util.Rect{
				Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
				Size: layout.FPt(image.Point{
					X: gtx.Constraints.Max.X,
					Y: dims.Size.Y,
				}),
			}.Layout(gtx)

		}),
		layout.Expanded(func(gtx C) D {
			return t.Content.Layout(gtx)
		}),
	)
	call.Add(gtx.Ops)
	return dims
}

func (t *Ticket) bottomBar(gtx C, th *material.Theme, sz image.Point, c color.NRGBA) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			return util.Rect{
				Color: c,
				Size:  layout.FPt(sz),
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Inset{
						Left: unit.Px(10),
					}.Layout(gtx, func(gtx C) D {
						return material.Label(th, unit.Dp(10), func() string {
							d := time.Since(t.Created)
							d = d.Round(time.Minute)
							h := d / time.Hour
							d -= h * time.Hour
							m := d / time.Minute
							return fmt.Sprintf("%02d:%02d", h, m)
						}()).Layout(gtx)
					})
				}),
				layout.Flexed(1, func(gtx C) D {
					return D{Size: gtx.Constraints.Min}
				}),
				layout.Rigid(func(gtx C) D {
					return util.Button(
						&t.PrevButton,
						util.WithIcon(icons.BackIcon),
						util.WithSize(unit.Dp(12)),
						util.WithInset(layout.UniformInset(unit.Dp(6))),
						util.WithIconColor(color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
						util.WithBgColor(c),
					).Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return util.Button(
						&t.NextButton,
						util.WithIcon(icons.ForwardIcon),
						util.WithSize(unit.Dp(12)),
						util.WithInset(layout.UniformInset(unit.Dp(6))),
						util.WithIconColor(color.NRGBA{R: 0, G: 0, B: 0, A: 255}),
						util.WithBgColor(c),
					).Layout(gtx)
				}),
			)
		}),
	)
}

func (t *Ticket) sideBar(gtx C, sz image.Point, c color.NRGBA) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			util.Rect{
				Color: c,
				Size:  layout.FPt(sz),
			}.Layout(gtx)
			return D{}
		}),
		layout.Stacked(func(gtx C) D {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(
					gtx,
					layout.Rigid(func(gtx C) D {
						return util.Button(
							&t.EditButton,
							util.WithIcon(icons.ContentEdit),
							util.WithSize(unit.Dp(16)),
							util.WithInset(layout.UniformInset(unit.Dp(2))),
							util.WithIconColor(color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
							util.WithBgColor(c),
						).Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
							return util.Button(
								&t.DeleteButton,
								util.WithIcon(icons.ContentDelete),
								util.WithSize(unit.Dp(16)),
								util.WithInset(layout.UniformInset(unit.Dp(2))),
								util.WithIconColor(color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
								util.WithBgColor(c),
							).Layout(gtx)
						})
					}),
				)
			})
		}),
	)
}

// TicketDetails renders the read-only long form details of a ticket.
type TicketDetails struct {
	kanban.Ticket
	Edit   widget.Clickable
	Cancel widget.Clickable
}

func (t *TicketDetails) Layout(gtx C, th *material.Theme) D {
	return control.Card{
		Title:    t.Title,
		Subtitle: t.Summary,
		Body: func(gtx C) D {
			return material.Body1(th, t.Details).Layout(gtx)
		},
		Actions: []control.Action{
			{
				Clickable: &t.Edit,
				Label:     "Edit",
				Fg:        th.ContrastFg,
				Bg:        th.ContrastBg,
			},
			{
				Clickable: &t.Cancel,
				Label:     "Cancel",
				Fg:        th.Fg,
				Bg:        th.Bg,
			},
		},
	}.Layout(gtx, th)
}
