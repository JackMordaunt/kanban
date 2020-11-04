package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"unsafe"

	"git.sr.ht/~jackmordaunt/kanban"
	"github.com/asdine/storm/v3"

	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/icons"
	"git.sr.ht/~whereswaldon/materials"

	"gioui.org/app"
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
	ui := &UI{
		Window: app.NewWindow(app.Title("Kanban")),
		Th:     material.NewTheme(gofont.Collection()),
		Kanban: &kanban.Kanban{
			Store: db,
		},
		// TODO: render dynamically from storage.
		Panels: []Panel{
			{
				Label:     "Todo",
				Color:     color.RGBA{R: 200, G: 200, B: 200, A: 255},
				Thickness: unit.Dp(50),
			},
			{
				Label:     "In Progress",
				Color:     color.RGBA{G: 100, B: 200, A: 255},
				Thickness: unit.Dp(50),
			},
			{
				Label:     "Testing",
				Color:     color.RGBA{R: 200, G: 100, A: 255},
				Thickness: unit.Dp(50),
			},
			{
				Label:     "Done",
				Color:     color.RGBA{R: 50, G: 200, B: 100, A: 255},
				Thickness: unit.Dp(50),
			},
		},
	}
	go func() {
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
	Kanban       *kanban.Kanban
	Th           *material.Theme
	Panels       []Panel
	TicketStates Map
	Modal        layout.Widget
	TicketForm   TicketForm
}

func (ui *UI) Loop() error {
	var ops op.Ops
	for {
		switch event := (<-ui.Events()).(type) {
		case system.DestroyEvent:
			return event.Err
		case system.ClipboardEvent:
			fmt.Printf("clipboard: %v\n", event.Text)
		case *system.CommandEvent:
			// TODO: integrate with command events.
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, event)
			ui.Update(gtx)
			ui.Layout(gtx)
			event.Frame(gtx.Ops)
		}
	}
}

func (ui *UI) Update(gtx C) {
	for ii := range ui.Panels {
		panel := &ui.Panels[ii]
		if panel.CreateTicket.Clicked() {
			ui.Modal = func(gtx C) D {
				return ui.TicketForm.Layout(gtx, ui.Th, panel.Label)
			}
		}
	}
	for _, state := range ui.TicketStates.List() {
		state := (*Ticket)(state)
		if state.NextButton.Clicked() {
			if err := ui.Kanban.Progress(state.ID); err != nil {
				fmt.Printf("error: %s\n", err)
			}
		}
		if state.PrevButton.Clicked() {
			if err := ui.Kanban.Regress(state.ID); err != nil {
				fmt.Printf("error: %s\n", err)
			}
		}
	}
	if ui.TicketForm.Submit.Clicked() {
		ticket, err := ui.TicketForm.Validate()
		if err != nil {
			fmt.Printf("error: %s\n", err)
			return
		}
		if err := ui.Kanban.Assign(ui.TicketForm.Stage, ticket); err != nil {
			fmt.Printf("error: assigning ticket: %s\n", err)
			return
		}
		ui.TicketForm = TicketForm{}
		ui.Modal = nil
	}
	if ui.TicketForm.Cancel.Clicked() {
		ui.TicketForm = TicketForm{}
		ui.Modal = nil
	}
}

func (ui *UI) Layout(gtx C) D {
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
							t := (*Ticket)(ui.TicketStates.Next(id, unsafe.Pointer(&Ticket{})))
							t.Ticket = stage.Tickets[ii]
							t.Stage = stage.Name
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
			return Modal(gtx, ui.Th, "Add Ticket", func(gtx C) D {
				return ui.Modal(gtx)
			})
		}),
	)
}

// Modal renders content centered with decorations.
func Modal(gtx C, th *material.Theme, title string, content layout.Widget) D {
	return Centered(gtx, func(gtx C) D {
		gtx.Constraints.Max.X = int(float32(gtx.Constraints.Max.X) * 0.8)
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
									return material.H6(th, title).Layout(gtx)
								})
							}),
						)
					}),
					layout.Rigid(func(gtx C) D {
						return inset.Layout(gtx, func(gtx C) D {
							return content(gtx)
						})
					}),
				)
			}),
		)
	})
}

func Centered(gtx C, content layout.Widget) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			return Rect{
				Size:  layout.FPt(gtx.Constraints.Max),
				Color: color.RGBA{A: 200},
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
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
							return content(gtx)
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
		}),
	)
}

// TicketForm renders the form for ticket information.
type TicketForm struct {
	Stage    string
	Title    materials.TextField
	Category materials.TextField
	Summary  materials.TextField
	Details  materials.TextField
	Submit   widget.Clickable
	Cancel   widget.Clickable
}

func (form TicketForm) Validate() (kanban.Ticket, error) {
	ticket := kanban.Ticket{
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
						return material.Button(th, &form.Cancel, "Cancel").Layout(gtx)
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

// Panel can hold cards.
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
									gtx.Constraints.Max = image.Point{
										X: gtx.Px(unit.Dp(10)),
										Y: gtx.Px(unit.Dp(10)),
									}
									return material.IconButton(th, &p.CreateTicket, icons.ContentAdd).Layout(gtx)
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
								gtx.Constraints.Max.Y = gtx.Px(unit.Dp(100))
								return widget.Border{
									Width: unit.Dp(0.5),
									Color: color.RGBA{A: 200},
								}.Layout(gtx, func(gtx C) D {
									return tickets[ii](gtx, ii)
								})
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
	Stage      string
	NextButton widget.Clickable
	PrevButton widget.Clickable
}

func (t *Ticket) Layout(gtx C, th *material.Theme) D {
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			// Side bar with controls
			return Rect{
				Color: color.RGBA{G: 100, B: 200, A: 255},
				Size: f32.Point{
					X: float32(gtx.Px(unit.Dp(15))),
					Y: float32(gtx.Constraints.Max.Y),
				},
			}.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(
				gtx,
				layout.Flexed(1, func(gtx C) D {
					// content
					return layout.Inset{
						Top:   unit.Dp(5),
						Left:  unit.Dp(10),
						Right: unit.Dp(10),
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
				}),
				layout.Rigid(func(gtx C) D {
					// bottom controls
					gtx.Constraints.Max = image.Point{
						X: gtx.Constraints.Max.X,
						Y: gtx.Px(unit.Dp(20)),
					}
					gtx.Constraints.Min = image.Point{
						Y: gtx.Px(unit.Dp(20)),
					}
					return layout.Stack{}.Layout(
						gtx,
						layout.Expanded(func(gtx C) D {
							return Rect{
								Color: color.RGBA{A: 100},
								Size: f32.Point{
									X: layout.FPt(gtx.Constraints.Max).X,
									Y: layout.FPt(gtx.Constraints.Min).Y,
								},
							}.Layout(gtx)
						}),
						layout.Stacked(func(gtx C) D {
							return layout.UniformInset(unit.Dp(2.5)).Layout(gtx, func(gtx C) D {
								inset := layout.Inset{Left: unit.Dp(2), Right: unit.Dp(2)}
								return layout.Flex{
									Axis: layout.Horizontal,
								}.Layout(
									gtx,
									layout.Flexed(1, func(gtx C) D {
										return D{Size: gtx.Constraints.Max}
									}),
									layout.Flexed(1, func(gtx C) D {
										return inset.Layout(gtx, func(gtx C) D {
											btn := material.IconButton(th, &t.PrevButton, icons.BackIcon)
											btn.Size = unit.Dp(15)
											gtx.Constraints.Max.X = gtx.Px(unit.Dp(15))
											return btn.Layout(gtx)
										})
									}),
									layout.Flexed(1, func(gtx C) D {
										return inset.Layout(gtx, func(gtx C) D {
											btn := material.IconButton(th, &t.NextButton, icons.ForwardIcon)
											btn.Size = unit.Dp(15)
											gtx.Constraints.Max.X = gtx.Px(unit.Dp(15))
											return btn.Layout(gtx)
										})
									}),
								)
							})
						}),
					)
				}),
			)
		}),
	)
}

// Map of arbitrary data.
type Map struct {
	data map[string]unsafe.Pointer
}

func (m *Map) Begin() {
	if m.data == nil {
		m.data = make(map[string]unsafe.Pointer)
	}
}

func (m *Map) Next(k string, v unsafe.Pointer) unsafe.Pointer {
	if _, ok := m.data[k]; !ok {
		m.data[k] = v
	}
	return m.data[k]
}

func (m *Map) List() []unsafe.Pointer {
	list := []unsafe.Pointer{}
	for _, v := range m.data {
		if v != nil {
			list = append(list, v)
		}
	}
	return list
}
