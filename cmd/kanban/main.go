package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"unsafe"

	"git.sr.ht/~jackmordaunt/kanban"
	"github.com/asdine/storm/v3"

	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/icons"
	"git.sr.ht/~whereswaldon/materials"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
)

func main() {
	db, err := func() (*storm.DB, error) {
		path := filepath.Join(os.TempDir(), "kanban.db")
		fmt.Printf("%s\n", path)
		var init = false
		if _, err := os.Stat(path); os.IsNotExist(err) {
			init = true
		}
		db, err := storm.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening data file: %w", err)
		}
		if err := db.Init(&kanban.Stage{}); err != nil {
			return nil, err
		}
		if err := db.ReIndex(&kanban.Stage{}); err != nil {
			return nil, err
		}
		if init {
			for ii, stage := range []string{"Todo", "In Progress", "Testing", "Done"} {
				if err := db.Save(&kanban.Stage{ID: ii + 1, Name: stage}); err != nil {
					return nil, fmt.Errorf("creating default stages: %w", err)
				}
			}
		}
		return db, nil
	}()
	if err != nil {
		log.Fatalf("error: initializing data: %v", err)
	}
	defer db.Close()
	go func() {
		ui := UI{
			Window: app.NewWindow(app.Title("Kanban")),
			Th:     material.NewTheme(gofont.Collection()),
			Kanban: &kanban.Kanban{
				Store: db,
			},
			// TODO: render dynamically from storage.
			Panels: []Panel{
				{
					Label:     "Todo",
					Color:     color.RGBA{R: 0x91, G: 0x81, B: 0x8a, A: 220},
					Thickness: unit.Dp(50),
				},
				{
					Label:     "In Progress",
					Color:     color.RGBA{R: 0, G: 100, B: 200, A: 220},
					Thickness: unit.Dp(50),
				},
				{
					Label:     "Testing",
					Color:     color.RGBA{R: 200, G: 100, B: 0, A: 220},
					Thickness: unit.Dp(50),
				},
				{
					Label:     "Done",
					Color:     color.RGBA{R: 50, G: 200, B: 100, A: 220},
					Thickness: unit.Dp(50),
				},
			},
		}
		if err := ui.Loop(); err != nil {
			log.Fatalf("error: %v", err)
		}
		os.Exit(0)
	}()
	app.Main()
}

type (
	C = layout.Context
	D = layout.Dimensions
)

// UI is the high level object that contains all global state.
// Anything that needs to integrate with the external system is allocated on
// this object.
type UI struct {
	*app.Window
	Kanban        *kanban.Kanban
	Th            *material.Theme
	Panels        []Panel
	TicketStates  Map
	Modal         layout.Widget
	TicketForm    TicketForm
	TicketDetails TicketDetails
	DeleteDialog  DeleteDialog
	FocusedTicket struct {
		ID    int
		Index int
		Stage int
	}
}

func (ui *UI) Loop() error {
	var (
		ops    op.Ops
		events = ui.Events()
	)
	for event := range events {
		switch event := (event).(type) {
		case system.DestroyEvent:
			return event.Err
		case system.ClipboardEvent:
			fmt.Printf("clipboard: %v\n", event.Text)
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, event)
			ui.Update(gtx)
			ui.Layout(gtx)
			event.Frame(gtx.Ops)
		}
	}
	return nil
}

// TODO: investigate best way to handle errors. Some errors are for the user
// and others are for the devs.
// Currently errors are just printed; not great for windowed applications.
func (ui *UI) Update(gtx C) {
	for _, event := range gtx.Events(ui) {
		if k, ok := event.(key.Event); ok {
			switch k.Name {
			case key.NameEscape:
				ui.Clear()
			case key.NameEnter, key.NameReturn:
				// TODO: query for a single Ticket by ID, quickly.
				tickets, err := ui.Kanban.Tickets()
				if err != nil {
					fmt.Printf("error: %v", err)
					break
				}
				for _, t := range tickets {
					t := t
					if t.ID == ui.FocusedTicket.ID {
						ui.InspectTicket(t)
						break
					}
				}
			case key.NameDownArrow:
				ui.Refocus(NextTicket)
			case key.NameUpArrow:
				ui.Refocus(PreviousTicket)
			case key.NameRightArrow:
				ui.Refocus(NextStage)
			case key.NameLeftArrow:
				ui.Refocus(PreviousStage)
			}
		}
	}
	for ii := range ui.Panels {
		panel := &ui.Panels[ii]
		if panel.CreateTicket.Clicked() {
			ui.Modal = func(gtx C) D {
				return Card{
					Title: "Add Ticket",
				}.Layout(gtx, ui.Th, func(gtx C) D {
					return ui.TicketForm.Layout(gtx, ui.Th, panel.Label)
				})
			}
		}
	}
	for s, ok := ui.TicketStates.Next(); ok; s, ok = ui.TicketStates.Next() {
		t := (*Ticket)(s)
		if ui.Modal != nil {
			continue
		}
		if t.NextButton.Clicked() {
			if err := ui.Kanban.Progress(t.ID); err != nil {
				fmt.Printf("error: %s\n", err)
			}
		}
		if t.PrevButton.Clicked() {
			if err := ui.Kanban.Regress(t.ID); err != nil {
				fmt.Printf("error: %s\n", err)
			}
		}
		if t.EditButton.Clicked() {
			ui.EditTicket(t.Ticket)
		}
		if t.DeleteButton.Clicked() {
			ui.DeleteTicket(t.Ticket)
		}
		if t.Content.Clicked() {
			ui.InspectTicket(t.Ticket)
		}
	}
	if ui.TicketForm.Submit.Clicked() {
		ticket, err := ui.TicketForm.Validate()
		if err != nil {
			fmt.Printf("error: %s\n", err)
			return
		}
		if assign := ui.TicketForm.Stage != ""; assign {
			if err := ui.Kanban.Assign(ui.TicketForm.Stage, ticket); err != nil {
				fmt.Printf("error: assigning ticket: %s\n", err)
				return
			}
		} else {
			if err := ui.Kanban.Update(ticket); err != nil {
				fmt.Printf("error: updating ticket: %s\n", err)
				return
			}
		}
		ui.Clear()
	}
	if ui.TicketForm.Cancel.Clicked() {
		ui.Clear()
	}
	if ui.DeleteDialog.Ok.Clicked() {
		if err := ui.Kanban.Delete(ui.DeleteDialog.ID); err != nil {
			fmt.Printf("error: %s\n", err)
		}
		ui.Clear()
	}
	if ui.DeleteDialog.Cancel.Clicked() {
		ui.Clear()
	}
	if ui.TicketDetails.Edit.Clicked() {
		ui.EditTicket(ui.TicketDetails.Ticket)
	}
	if ui.TicketDetails.Cancel.Clicked() {
		ui.Clear()
	}
}

func (ui *UI) Layout(gtx C) D {
	key.InputOp{Tag: ui}.Add(gtx.Ops)
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			ui.TicketStates.Begin()
			var panels = make([]layout.FlexChild, len(ui.Panels))
			for kk := range ui.Panels {
				panel := &ui.Panels[kk]
				panels[kk] = layout.Flexed(1, func(gtx C) D {
					stage, _ := ui.Kanban.Stage(panel.Label)
					var cards = make([]layout.ListElement, len(stage.Tickets))
					for ii, ticket := range stage.Tickets {
						id := strconv.Itoa(ticket.ID)
						cards[ii] = func(gtx C, ii int) D {
							t := (*Ticket)(ui.TicketStates.New(id, unsafe.Pointer(&Ticket{})))
							t.Ticket = stage.Tickets[ii]
							t.Stage = stage.Name
							if ui.FocusedTicket.ID == t.ID {
								return widget.Border{
									Color: color.RGBA{B: 200, A: 200},
									Width: unit.Dp(2),
								}.Layout(gtx, func(gtx C) D {
									return t.Layout(gtx, ui.Th)
								})
							}
							return t.Layout(gtx, ui.Th)
						}
					}
					return panel.Layout(gtx, ui.Th, cards...)
				})
			}
			return layout.Flex{
				Axis:    layout.Horizontal,
				Spacing: layout.SpaceEvenly,
			}.Layout(
				gtx,
				panels...,
			)
		}),
		layout.Expanded(func(gtx C) D {
			if ui.Modal == nil {
				return D{}
			}
			return Modal(gtx, func(gtx C) D {
				return ui.Modal(gtx)
			})
		}),
	)
}

type Direction uint8

const (
	NextTicket Direction = iota
	PreviousTicket
	NextStage
	PreviousStage
)

// Refocus to the ticket in the given direction.
// Allows movement between tickets and stages in sequential order.
func (ui *UI) Refocus(d Direction) {
	stages, err := ui.Kanban.ListStages()
	if err != nil {
		fmt.Printf("error: querying stages: %v", err)
		return
	}
	for {
		switch d {
		case NextTicket:
			ui.FocusedTicket.Index++
			if ui.FocusedTicket.Index > len(stages[ui.FocusedTicket.Stage].Tickets) {
				ui.FocusedTicket.Index = 1
				ui.FocusedTicket.Stage++
				if ui.FocusedTicket.Stage > len(stages)-1 {
					ui.FocusedTicket.Stage = 0
				}
			}
		case PreviousTicket:
			ui.FocusedTicket.Index--
			if ui.FocusedTicket.Index < 1 {
				ui.FocusedTicket.Stage--
				if ui.FocusedTicket.Stage < 0 {
					ui.FocusedTicket.Stage = len(stages) - 1
				}
				ui.FocusedTicket.Index = len(stages[ui.FocusedTicket.Stage].Tickets)
			}
		case NextStage:
			ui.FocusedTicket.Index = 1
			ui.FocusedTicket.Stage++
			if ui.FocusedTicket.Stage > len(stages)-1 {
				ui.FocusedTicket.Stage = 0
			}
		case PreviousStage:
			ui.FocusedTicket.Index = 1
			ui.FocusedTicket.Stage--
			if ui.FocusedTicket.Stage < 0 {
				ui.FocusedTicket.Stage = len(stages) - 1
			}
		}
		if stage := stages[ui.FocusedTicket.Stage]; !stage.Empty() {
			break
		}
	}
	ui.FocusedTicket.ID = stages[ui.FocusedTicket.Stage].Tickets[ui.FocusedTicket.Index-1].ID
}

// Clear resets navigational state.
func (ui *UI) Clear() {
	ui.Modal = nil
	ui.TicketForm = TicketForm{}
	ui.FocusedTicket = struct {
		ID    int
		Index int
		Stage int
	}{}
}

// InspectTicket opens the ticket details card for the given ticket.
func (ui *UI) InspectTicket(t kanban.Ticket) {
	ui.TicketDetails.Ticket = t
	ui.Modal = func(gtx C) D {
		return Card{
			Title: fmt.Sprintf("%q", t.Title),
		}.Layout(gtx, ui.Th, func(gtx C) D {
			return ui.TicketDetails.Layout(gtx, ui.Th)
		})
	}
}

// EditTicket opens the ticket form for editing ticket data.
func (ui *UI) EditTicket(t kanban.Ticket) {
	ui.TicketForm.Set(t)
	ui.Modal = func(gtx C) D {
		return Card{
			Title: "Edit Ticket",
		}.Layout(gtx, ui.Th, func(gtx C) D {
			return ui.TicketForm.Layout(gtx, ui.Th, "")
		})
	}
}

// DeleteTickets opens the confirmation dialog for deleting a ticket.
func (ui *UI) DeleteTicket(t kanban.Ticket) {
	ui.DeleteDialog.Ticket = t
	ui.Modal = func(gtx C) D {
		return Card{
			Title: "Delete Ticket",
		}.Layout(gtx, ui.Th, func(gtx C) D {
			return ui.DeleteDialog.Layout(gtx, ui.Th)
		})
	}
}

// TicketForm renders the form for ticket information.
//
// TODO: tab navigation through form fields.
type TicketForm struct {
	Stage    string
	Data     kanban.Ticket
	Title    materials.TextField
	Category materials.TextField
	Summary  materials.TextField
	Details  materials.TextField
	Submit   widget.Clickable
	Cancel   widget.Clickable
}

func (form *TicketForm) Set(t kanban.Ticket) {
	form.Data = t
	form.Title.SetText(t.Title)
	form.Category.SetText(t.Category)
	form.Summary.SetText(t.Summary)
	form.Details.SetText(t.Details)
	// form.References.SetText(t.References)
}

// Validate the inputs.
// Note: No actual validation is done yet.
func (form TicketForm) Validate() (kanban.Ticket, error) {
	ticket := kanban.Ticket{
		ID:       form.Data.ID,
		Created:  form.Data.Created,
		Title:    form.Title.Text(),
		Details:  form.Details.Text(),
		Summary:  form.Summary.Text(),
		Category: form.Category.Text(),
	}
	return ticket, nil
}

func (form *TicketForm) Layout(gtx C, th *material.Theme, stage string) D {
	form.Stage = stage
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			return form.Title.Layout(gtx, th, "Title")
		}),
		layout.Rigid(func(gtx C) D {
			return form.Category.Layout(gtx, th, "Category")
		}),
		layout.Rigid(func(gtx C) D {
			return form.Summary.Layout(gtx, th, "Summary")
		}),
		layout.Rigid(func(gtx C) D {
			return form.Details.Layout(gtx, th, "Details")
		}),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Inset{
				Top: unit.Dp(10),
			}.Layout(gtx, func(gtx C) D {
				return layout.Flex{
					Axis: layout.Horizontal,
				}.Layout(
					gtx,
					layout.Flexed(1, func(gtx C) D {
						return D{Size: gtx.Constraints.Min}
					}),
					layout.Rigid(func(gtx C) D {
						btn := material.Button(th, &form.Cancel, "Cancel")
						btn.Color = th.Color.Primary
						btn.Background = color.RGBA{}
						return btn.Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return D{Size: image.Point{X: gtx.Px(unit.Dp(10))}}
					}),
					layout.Rigid(func(gtx C) D {
						return material.Button(th, &form.Submit, "Submit").Layout(gtx)
					}),
				)
			})
		}),
	)
}

// DeleteDialog prompts the user with an option to delete a ticket.
type DeleteDialog struct {
	kanban.Ticket
	Ok     widget.Clickable
	Cancel widget.Clickable
}

func (d *DeleteDialog) Layout(gtx C, th *material.Theme) D {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Center.Layout(gtx, func(gtx C) D {
				return material.Body1(
					th,
					fmt.Sprintf("Are you sure you want to delete ticket %q?", d.Title),
				).Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Inset{
				Top: unit.Dp(10),
			}.Layout(gtx, func(gtx C) D {
				return layout.Flex{
					Axis: layout.Horizontal,
				}.Layout(
					gtx,
					layout.Flexed(1, func(gtx C) D {
						return D{Size: gtx.Constraints.Min}
					}),
					layout.Rigid(func(gtx C) D {
						btn := material.Button(th, &d.Cancel, "Cancel")
						btn.Color = th.Color.Primary
						btn.Background = color.RGBA{}
						return btn.Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return D{Size: image.Point{X: gtx.Px(unit.Dp(10))}}
					}),
					layout.Rigid(func(gtx C) D {
						btn := material.Button(th, &d.Ok, "Delete")
						btn.Background = color.RGBA{R: 200, A: 255}
						return btn.Layout(gtx)
					}),
				)
			})
		}),
	)
}

// Panel can hold cards.
// One panel per stage in the kanban pipeline.
// Has a title and action bar.
type Panel struct {
	Label        string
	Color        color.RGBA
	Thickness    unit.Value
	CreateTicket widget.Clickable

	layout.List
}

func (p *Panel) Layout(gtx C, th *material.Theme, tickets ...layout.ListElement) D {
	return widget.Border{
		Color: color.RGBA{A: 200},
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
						return Rect{
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
									return Button(
										&p.CreateTicket,
										WithIcon(icons.ContentAdd),
										WithSize(unit.Dp(15)),
										WithInset(layout.UniformInset(unit.Dp(6))),
										WithBgColor(color.RGBA{}),
										WithIconColor(th.Color.Text),
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
						return Rect{
							Color: color.RGBA{R: 240, G: 240, B: 240, A: 255},
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
// an infinite Y axis. That means you can just specify a max Y constraint. This
// makes expanding stacked content vertically impossible with a naive use of
// `layout.Stack`.
//
// To get around this I used a macro and manually stacked things sized exactly
// to the content, rather than the maximum Y.
func (t *Ticket) Layout(gtx C, th *material.Theme) D {
	var (
		barThickness   = unit.Dp(25)
		sideBarColor   = color.RGBA{R: 50, G: 50, B: 50, A: 255}
		bottomBarColor = color.RGBA{R: 220, G: 220, B: 220, A: 255}
		minContentSize = gtx.Px(unit.Dp(150))
	)
	return widget.Border{
		Width: unit.Dp(0.5),
		Color: color.RGBA{A: 200},
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
				return layout.Inset{
					Top: unit.Dp(2),
				}.Layout(gtx, func(gtx C) D {
					th := *th
					th.Color.Text = materials.AlphaMultiply(th.Color.Text, 200)
					return material.Label(&th, unit.Dp(14), t.Category).Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
					return material.Body1(th, t.Summary).Layout(gtx)
				})
			}),
		)
	})
	call := macro.Stop()
	layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			return Rect{
				Color: color.RGBA{R: 255, G: 255, B: 255, A: 255},
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

func (t *Ticket) bottomBar(gtx C, th *material.Theme, sz image.Point, c color.RGBA) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			return Rect{
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
					return Button(
						&t.PrevButton,
						WithIcon(icons.BackIcon),
						WithSize(unit.Dp(12)),
						WithInset(layout.UniformInset(unit.Dp(6))),
						WithIconColor(color.RGBA{R: 0, G: 0, B: 0, A: 255}),
						WithBgColor(c),
					).Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return Button(
						&t.NextButton,
						WithIcon(icons.ForwardIcon),
						WithSize(unit.Dp(12)),
						WithInset(layout.UniformInset(unit.Dp(6))),
						WithIconColor(color.RGBA{R: 0, G: 0, B: 0, A: 255}),
						WithBgColor(c),
					).Layout(gtx)
				}),
			)
		}),
	)
}

func (t *Ticket) sideBar(gtx C, sz image.Point, c color.RGBA) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			Rect{
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
						return Button(
							&t.EditButton,
							WithIcon(icons.ContentEdit),
							WithSize(unit.Dp(16)),
							WithInset(layout.UniformInset(unit.Dp(2))),
							WithIconColor(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
							WithBgColor(c),
						).Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
							return Button(
								&t.DeleteButton,
								WithIcon(icons.ContentDelete),
								WithSize(unit.Dp(16)),
								WithInset(layout.UniformInset(unit.Dp(2))),
								WithIconColor(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
								WithBgColor(c),
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
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			return material.Body1(th, t.Details).Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Inset{
				Top: unit.Dp(10),
			}.Layout(gtx, func(gtx C) D {
				return layout.Flex{
					Axis: layout.Horizontal,
				}.Layout(
					gtx,
					layout.Flexed(1, func(gtx C) D {
						return D{Size: gtx.Constraints.Min}
					}),
					layout.Rigid(func(gtx C) D {
						btn := material.Button(th, &t.Cancel, "Cancel")
						btn.Color = th.Color.Primary
						btn.Background = color.RGBA{}
						return btn.Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return D{Size: image.Point{X: gtx.Px(unit.Dp(10))}}
					}),
					layout.Rigid(func(gtx C) D {
						return material.Button(th, &t.Edit, "Edit").Layout(gtx)
					}),
				)
			})
		}),
	)
}
