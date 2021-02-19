package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"unsafe"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/control"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/state"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/util"
	"git.sr.ht/~jackmordaunt/kanban/icons"
	"git.sr.ht/~jackmordaunt/kanban/storage"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// UI is the high level object that contains UI-global state.
//
// Anything that needs to integrate with the external system is allocated on
// this object.
//
// UI has three primary methods "Loop", "Update" and "Layout".
// Loop starts the event loop and runs until the program terminates.
// Update changes state based on events.
// Layout takes the UI state and renders using Gio primitives.
type UI struct {
	// Window is a reference to the window handle.
	*app.Window

	// Th contains theme data application wide.
	Th *material.Theme

	// Storage driver responsible for allocating Project objects.
	Storage storage.Storer

	// Projects is an in-memory list of the projects.
	// Refreshed from Storage before every frame.
	// Save to Storage after every frame.
	Projects Projects

	// Project is the currently active kanban Project.
	// Contains the state and methods for kanban operations.
	// Points to memory allocated by the storage implementation.
	// nil value implies no active project.
	Project *kanban.Project

	// Panels render the active Project stages.
	// Shares the same lifetime as the active project.
	Panels []*control.Panel

	// Rail allows intra-project navigation as a side bar.
	// When a Project item is clicked, that Project is loaded from storage and
	// becomes the active Project.
	Rail control.Rail

	// TicketStates allocates memory for the Project's tickets and assocated
	// UI state.
	TicketStates state.Map

	// Modal is rendered atop the main content when not nil.
	Modal layout.Widget

	// Form state.
	TicketForm    TicketForm
	TicketDetails TicketDetails
	DeleteDialog  DeleteDialog
	ProjectForm   ProjectForm

	// FocusedTicket struct {
	// 	ID    kanban.ID
	// 	Index int
	// 	Stage kanban.ID
	// }

	CreateProjectButton widget.Clickable
}

// Loop runs the event loop until terminated.
func (ui *UI) Loop() error {
	var (
		ops    op.Ops
		events = ui.Window.Events()
	)
	projects, err := ui.Storage.List()
	if err != nil {
		return fmt.Errorf("loading projects: %v", err)
	}
	ui.Projects = projects
	for event := range events {
		switch event := (event).(type) {
		case system.DestroyEvent:
			return event.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, event)
			ui.Load()
			ui.Update(gtx)
			ui.Layout(gtx)
			ui.Save()
			event.Frame(gtx.Ops)
		}
	}
	return ui.Shutdown()
}

// Shutdown does cleanup.
func (ui *UI) Shutdown() error {
	// if err := ui.Storage.Save(*ui.Project); err != nil {
	// 	return fmt.Errorf("saving project: %v", err)
	// }
	return nil
}

// Update state based on events.
//
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
				// @Cleanup
				// on "enter" we want to launch edit form for the focused ticket.
				//
				// var (
				// 	t kanban.Ticket
				// )
				// if err := ui.Project.Find("ID", ui.FocusedTicket, &t); err != nil {
				// 	fmt.Printf("error: %v\n", err)
				// } else {
				// 	ui.InspectTicket(t)
				// }
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
	if ui.ProjectForm.Submit.Clicked() {
		p := kanban.Project{
			Name: ui.ProjectForm.Name.Text(),
			Stages: []kanban.Stage{
				{Name: "Todo"},
				{Name: "In Progress"},
				{Name: "Testing"},
				{Name: "Done"},
			},
		}
		if err := ui.Storage.Create(p); err != nil {
			log.Printf("creating new project: %v", err)
		} else {
			// Note: the Storer interface only updates the projects
			// in the slice given to it. Therefore we add the Project
			// to the slice here.  @cleanup
			ui.Projects = append(ui.Projects, p)
		}
		ui.Clear()
	}
	if p, ok := ui.Rail.Selected(); ok {
		if ui.Project == nil || ui.Project.Name != p {
			project, ok := ui.Projects.Find(p)
			if ok {
				ui.Clear()
				ui.Project = project
				ui.Panels = func() (panels []*control.Panel) {
					for _, s := range project.Stages {
						panels = append(panels, &control.Panel{
							Label:     s.Name,
							Color:     color.NRGBA{R: 100, B: 100, G: 200, A: 255},
							Thickness: unit.Dp(50),
						})
					}
					return panels
				}()
			}
		}
	}
	if ui.ProjectForm.Cancel.Clicked() {
		ui.Clear()
	}
	for ii := range ui.Panels {
		panel := ui.Panels[ii]
		if panel.CreateTicket.Clicked() {
			ui.AddTicket(panel.Label)
		}
	}
	for ui.TicketStates.More() {
		_, v := ui.TicketStates.Next()
		t := (*Ticket)(v)
		if ui.Modal != nil {
			continue
		}
		if t.NextButton.Clicked() {
			ui.Project.ProgressTicket(t.Ticket)
		}
		if t.PrevButton.Clicked() {
			ui.Project.RegressTicket(t.Ticket)
		}
		if t.EditButton.Clicked() {
			ui.EditTicket(&t.Ticket)
		}
		if t.DeleteButton.Clicked() {
			ui.DeleteTicket(t.Ticket)
		}
		if t.Content.Clicked() {
			ui.InspectTicket(t.Ticket)
		}
	}
	if ui.TicketForm.SubmitBtn.Clicked() {
		// @todo handle create/update ambiguity.
		_ = ui.TicketForm.Submit()
		ui.Project.AssignTicket(ui.TicketForm.Stage, *ui.TicketForm.Ticket)
		ui.Clear()
	}
	if ui.TicketForm.CancelBtn.Clicked() {
		ui.Clear()
	}
	if ui.DeleteDialog.Ok.Clicked() {
		ui.Project.FinalizeTicket(ui.DeleteDialog.Ticket)
		ui.Clear()
	}
	if ui.DeleteDialog.Cancel.Clicked() {
		ui.Clear()
	}
	if ui.TicketDetails.Edit.Clicked() {
		ui.EditTicket(&ui.TicketDetails.Ticket)
	}
	if ui.TicketDetails.Cancel.Clicked() {
		ui.Clear()
	}
	if ui.CreateProjectButton.Clicked() {
		ui.CreateProject()
	}
}

// Layout UI.
func (ui *UI) Layout(gtx C) D {
	key.InputOp{Tag: ui}.Add(gtx.Ops)
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			gtx.Constraints.Max.X = gtx.Px(unit.Dp(80))
			gtx.Constraints.Min.X = 0
			return ui.layoutRail(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return ui.layoutContent(gtx)
		}),
	)
}

func (ui *UI) layoutRail(gtx C) D {
	var (
		rc []control.RailChild
	)
	projects, err := ui.Storage.List()
	if err != nil {
		log.Printf("error: loading projects: %v", err)
	}
	for _, p := range projects {
		p := p
		rc = append(rc, control.Destination(p.Name, func(gtx C) D {
			return layout.Stack{
				Alignment: layout.Center,
			}.Layout(
				gtx,
				layout.Stacked(func(gtx C) D {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx C) D {
						return material.Label(ui.Th, unit.Dp(16), p.Name).Layout(gtx)
					})
				}),
				layout.Expanded(func(gtx C) D {
					cs := gtx.Constraints
					if ui.Project != nil && ui.Project.Name == p.Name {
						return util.Rect{
							Color: color.NRGBA{A: 100},
							Size:  f32.Pt(float32(cs.Max.X), float32(cs.Min.Y)),
						}.Layout(gtx)
					}
					return D{Size: image.Point{X: cs.Max.X, Y: cs.Min.Y}}
				}),
			)
		}))
	}
	return ui.Rail.Layout(
		gtx,
		func(gtx C) D {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
				btn := material.IconButton(ui.Th, &ui.CreateProjectButton, icons.ContentAdd)
				btn.Size = unit.Dp(20)
				btn.Inset = layout.UniformInset(unit.Dp(8))
				return btn.Layout(gtx)
			})
		},
		rc...,
	)
}

func (ui *UI) layoutContent(gtx C) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Stacked(func(gtx C) D {
			if ui.Project == nil {
				return D{}
			}
			ui.TicketStates.Begin()
			return layout.Flex{
				Axis:    layout.Horizontal,
				Spacing: layout.SpaceEvenly,
			}.Layout(
				gtx,
				func() (panels []layout.FlexChild) {
					// @decouple this iteration relies on the coincidence that panels are ordered the same.
					for ii, stage := range ui.Project.Stages {
						stage := stage
						panel := ui.Panels[ii]
						panels = append(panels, layout.Flexed(1, func(gtx C) D {
							return panel.Layout(gtx, ui.Th, func() (tickets []layout.ListElement) {
								for _, ticket := range stage.Tickets {
									ticket := ticket
									t := (*Ticket)(ui.TicketStates.New(ticket.Title, unsafe.Pointer(&Ticket{})))
									t.Ticket = ticket
									t.Stage = stage.Name
									tickets = append(tickets, func(gtx C, index int) D {
										return t.Layout(gtx, ui.Th, false)
									})
								}
								return tickets
							}()...)
						}))
					}
					return panels
				}()...,
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
	// var (
	// 	project kanban.Project
	// 	stage kanban.Stage
	// )
	// if err := ui.Project.Find("ID", ui.ActiveProject, &project); err != nil {
	// 	log.Printf("error: %v", err)
	// 	return
	// }
	// if err := ui.Project.Find("ID", projet.St)

	// for {
	// 	switch d {
	// 	case NextTicket:
	// 		ui.FocusedTicket.Index++
	// 		if ui.FocusedTicket.Index > len(stages[ui.FocusedTicket.Stage].Tickets) {
	// 			ui.FocusedTicket.Index = 1
	// 			ui.FocusedTicket.Stage++
	// 			if ui.FocusedTicket.Stage > len(stages)-1 {
	// 				ui.FocusedTicket.Stage = 0
	// 			}
	// 		}
	// 	case PreviousTicket:
	// 		ui.FocusedTicket.Index--
	// 		if ui.FocusedTicket.Index < 1 {
	// 			ui.FocusedTicket.Stage--
	// 			if ui.FocusedTicket.Stage < 0 {
	// 				ui.FocusedTicket.Stage = len(stages) - 1
	// 			}
	// 			ui.FocusedTicket.Index = len(stages[ui.FocusedTicket.Stage].Tickets)
	// 		}
	// 	case NextStage:
	// 		ui.FocusedTicket.Index = 1
	// 		ui.FocusedTicket.Stage++
	// 		if ui.FocusedTicket.Stage > len(stages)-1 {
	// 			ui.FocusedTicket.Stage = 0
	// 		}
	// 	case PreviousStage:
	// 		ui.FocusedTicket.Index = 1
	// 		ui.FocusedTicket.Stage--
	// 		if ui.FocusedTicket.Stage < 0 {
	// 			ui.FocusedTicket.Stage = len(stages) - 1
	// 		}
	// 	}
	// 	if stage := stages[ui.FocusedTicket.Stage]; !stage.Empty() {
	// 		break
	// 	}
	// }
	// ui.FocusedTicket.ID = stages[ui.FocusedTicket.Stage].Tickets[ui.FocusedTicket.Index-1].ID
}

// Clear resets navigational state.
func (ui *UI) Clear() {
	ui.Modal = nil
	ui.TicketForm = TicketForm{}
	ui.ProjectForm = ProjectForm{}
	ui.DeleteDialog = DeleteDialog{}
	// @cleanup
	// ui.FocusedTicket = struct {
	// 	ID    kanban.ID
	// 	Index int
	// 	Stage kanban.ID
	// }{}
}

// InspectTicket opens the ticket details card for the given ticket.
func (ui *UI) InspectTicket(t kanban.Ticket) {
	ui.TicketDetails.Ticket = t
	ui.Modal = func(gtx C) D {
		return control.Card{
			Title: fmt.Sprintf("%q", t.Title),
		}.Layout(gtx, ui.Th, func(gtx C) D {
			return ui.TicketDetails.Layout(gtx, ui.Th)
		})
	}
}

// EditTicket opens the ticket form for editing ticket data.
func (ui *UI) EditTicket(t *kanban.Ticket) {
	ui.TicketForm.Set(t)
	ui.Modal = func(gtx C) D {
		return control.Card{
			Title: "Edit Ticket",
		}.Layout(gtx, ui.Th, func(gtx C) D {
			return ui.TicketForm.Layout(gtx, ui.Th, "")
		})
	}
}

// AddTicket opens the ticket form for creating ticket data.
func (ui *UI) AddTicket(stage string) {
	ui.TicketForm.Set(&kanban.Ticket{})
	ui.Modal = func(gtx C) D {
		return control.Card{
			Title: "Add Ticket",
		}.Layout(gtx, ui.Th, func(gtx C) D {
			return ui.TicketForm.Layout(gtx, ui.Th, stage)
		})
	}
}

// CreatTicket opens the project creation dialog.
func (ui *UI) CreateProject() {
	ui.Modal = func(gtx C) D {
		return control.Card{
			Title: "Create a new Project",
		}.Layout(gtx, ui.Th, func(gtx C) D {
			return ui.ProjectForm.Layout(gtx, ui.Th)
		})
	}
}

// DeleteTickets opens the confirmation dialog for deleting a ticket.
func (ui *UI) DeleteTicket(t kanban.Ticket) {
	ui.DeleteDialog.Ticket = t
	ui.Modal = func(gtx C) D {
		return control.Card{
			Title: "Delete Ticket",
		}.Layout(gtx, ui.Th, func(gtx C) D {
			return ui.DeleteDialog.Layout(gtx, ui.Th)
		})
	}
}

// Projects is a list of Project entities with added behaviours.
type Projects []kanban.Project

// Find and return project by name.
// Boolean indicates whether the project exists.
func (plist Projects) Find(name string) (*kanban.Project, bool) {
	for _, p := range plist {
		p := p
		if p.Name == name {
			return &p, true
		}
	}
	return nil, false
}

// Load entities from storage.
func (ui *UI) Load() {
	if err := ui.Storage.Load(ui.Projects); err != nil {
		log.Printf("error: loading projects: %v", err)
	}
}

// Save entities to storage.
func (ui *UI) Save() {
	if err := ui.Storage.Save(ui.Projects...); err != nil {
		log.Printf("error: saving projects: %v", err)
	}
}
