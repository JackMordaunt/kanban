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
	"gioui.org/x/eventx"
	"git.sr.ht/~jackmordaunt/kanban"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/control"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/state"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/util"
	"git.sr.ht/~jackmordaunt/kanban/icons"
	"git.sr.ht/~jackmordaunt/kanban/storage"
	"github.com/google/uuid"
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
	// Points into Projects slice.
	// nil value implies no active project.
	Project *kanban.Project

	// previous is used to detect when the active project has change in order to
	// run init code like allocating panels.
	previous *kanban.Project

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
	TicketForm                 TicketForm
	TicketDetails              TicketDetails
	DeleteDialog               DeleteDialog
	ProjectForm                ProjectForm
	ArchiveProjectConfirmation ArchiveProjectConfirmation

	// Focus tracks the focused ticket for keyboard navigation.
	Focus struct {
		Stage  int
		Ticket int
		T      *kanban.Ticket
	}

	CreateProjectBtn widget.Clickable
	EditProjectBtn   widget.Clickable
}

// Loop runs the event loop until terminated.
func (ui *UI) Loop() error {
	count, err := ui.Storage.Count()
	if err != nil {
		return fmt.Errorf("counting projects: %w", err)
	}
	ui.Projects = make(Projects, count)
	if err := ui.Storage.Load(ui.Projects); err != nil {
		return fmt.Errorf("loading projects: %w", err)
	}
	if len(ui.Projects) > 0 {
		ui.Project = &ui.Projects[0]
	}
	var (
		ops    op.Ops
		events = ui.Window.Events()
	)
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
	return nil
}

// Update state based on events.
//
// @TODO(jfm): investigate best way to handle errors. Some errors are for the user
// and others are for the devs.
// Currently errors are just printed; not great for windowed applications.
//
// @BUG(jfm): ui not recieving keyboard events.
func (ui *UI) Update(gtx C) {
	if ui.Project != ui.previous {
		ui.sync()
	}
	for _, event := range gtx.Events(ui) {
		if k, ok := event.(key.Event); ok {
			if k.State == key.Press {
				switch k.Name {
				case key.NameEscape:
					ui.Clear()
				case key.NameEnter, key.NameReturn:
					if t := ui.Focus.T; t != nil {
						ui.EditTicket(*t)
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
	}
	if ui.ProjectForm.SubmitBtn.Clicked() {
		if ui.ProjectForm.Mode() == ModeEdit {
			ui.ProjectForm.Submit()
		}
		if ui.ProjectForm.Mode() == ModeCreate {
			if err := ui.Storage.Create(kanban.Project{
				ID:   uuid.New(),
				Name: ui.ProjectForm.Name.Text(),
				Stages: []kanban.Stage{
					{Name: "Todo"},
					{Name: "In Progress"},
					{Name: "Testing"},
					{Name: "Done"},
				},
			}); err != nil {
				log.Printf("creating new project: %v", err)
			} else {
				if projects, err := ui.Storage.List(); err == nil {
					ui.Projects = projects
				} else {
					log.Printf("listing projects: %v", err)
				}
			}
		}
		ui.Clear()
	}
	// @CLEANUP(jfm): Unclear code. If no active project or the selected
	// project does not match active project, make the selected project the
	// active project.
	// ID goes through a string representation bc no generics, probably a better
	// way to structure that.
	if id, ok := ui.Rail.Selected(); ok {
		if ui.Project == nil || ui.Project.ID.String() != id {
			if id, err := uuid.Parse(id); err == nil {
				project, ok := ui.Projects.Find(id)
				if ok {
					ui.Project = project
				}
			}
		}
	}
	if ui.ProjectForm.CancelBtn.Clicked() {
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
			ui.EditTicket(t.Ticket)
		}
		if t.DeleteButton.Clicked() {
			ui.DeleteTicket(t.Ticket)
		}
		if t.Content.Clicked() {
			ui.InspectTicket(t.Ticket)
		}
	}
	if ui.TicketForm.SubmitBtn.Clicked() {
		t := ui.TicketForm.Submit()
		if t.ID == uuid.Nil {
			if err := ui.Project.AssignTicket(ui.TicketForm.Stage, t); err != nil {
				log.Printf("assigning ticket: %v", err)
			}
		} else {
			if err := ui.Project.UpdateTicket(t); err != nil {
				log.Printf("updating ticket: %v", err)
			}
		}
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
		ui.EditTicket(ui.TicketDetails.Ticket)
	}
	if ui.TicketDetails.Cancel.Clicked() {
		ui.Clear()
	}
	if ui.CreateProjectBtn.Clicked() {
		ui.CreateProject()
	}
	if ui.EditProjectBtn.Clicked() {
		ui.EditProject()
	}
	if ui.ProjectForm.Delete.Button.Clicked() {
		ui.ShowArchiveProjectConfirmation()
	}
	if ui.ArchiveProjectConfirmation.SubmitBtn.Clicked() {
		if ui.ArchiveProjectConfirmation.Confirmation.Text() == ui.Project.Name {
			if err := ui.Storage.Archive(ui.Project.ID); err != nil {
				log.Printf("error: archiving project: %v", err)
			}
			ui.Save()
			if len(ui.Projects) > 0 {
				ui.Project = &ui.Projects[0]
			}
			ui.Clear()
		}
	}
	if ui.ArchiveProjectConfirmation.CancelBtn.Clicked() {
		ui.Clear()
	}
}

// Layout UI.
func (ui *UI) Layout(gtx C) D {
	// NOTE(jfm): an active modal implies a more specific key focus: to that
	// modal widget.
	key.InputOp{Tag: ui}.Add(gtx.Ops)
	if ui.Modal == nil {
		key.FocusOp{Tag: ui}.Add(gtx.Ops)
	}
	// Spy on the events so that we can do some global actions regardless of
	// focus.
	spy, gtx := eventx.Enspy(gtx)
	defer func() {
		for _, group := range spy.AllEvents() {
			for _, event := range group.Items {
				if k, ok := event.(key.Event); ok {
					if k.State == key.Press && k.Name == key.NameEscape {
						ui.Clear()
						return
					}
				}
			}
		}
	}()
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			gtx.Constraints.Max.X = gtx.Px(unit.Dp(80))
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
	for _, p := range ui.Projects {
		p := p
		rc = append(rc, control.Destination(p.ID.String(), func(gtx C) D {
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
					if isActive := ui.Project != nil && ui.Project.ID == p.ID; isActive {
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
				btn := material.IconButton(ui.Th, &ui.CreateProjectBtn, icons.ContentAdd)
				btn.Size = unit.Dp(20)
				btn.Inset = layout.UniformInset(unit.Dp(8))
				return btn.Layout(gtx)
			})
		},
		rc...,
	)
}

func (ui *UI) layoutContent(gtx C) D {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(
		gtx,
		// @todo streamline into app bar.
		layout.Rigid(func(gtx C) D {
			if ui.Project == nil {
				return D{}
			}
			return layout.Stack{}.Layout(
				gtx,
				layout.Expanded(func(gtx C) D {
					return util.Rect{
						Color: color.NRGBA{A: 255},
						Size: f32.Point{
							X: float32(gtx.Constraints.Max.X),
							Y: float32(gtx.Constraints.Min.Y),
						},
					}.Layout(gtx)
				}),
				layout.Stacked(func(gtx C) D {
					return layout.Inset{
						Left:  unit.Dp(10),
						Right: unit.Dp(10),
					}.Layout(gtx, func(gtx C) D {
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Middle,
						}.Layout(
							gtx,
							layout.Rigid(func(gtx C) D {
								l := material.H5(ui.Th, ui.Project.Name)
								l.Color = ui.Th.ContrastFg
								return l.Layout(gtx)
							}),
							layout.Flexed(1, func(gtx C) D {
								return D{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Min.Y}}
							}),
							layout.Rigid(func(gtx C) D {
								btn := material.IconButton(ui.Th, &ui.EditProjectBtn, icons.Configuration)
								btn.Background = color.NRGBA{}
								btn.Inset = layout.UniformInset(unit.Dp(5))
								return btn.Layout(gtx)
							}),
						)
					})
				}),
			)
		}),
		layout.Flexed(1, func(gtx C) D {
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
							if len(ui.Panels) == 0 {
								return panels
							}
							// @decouple this iteration relies on the coincidence that panels are ordered the same.
							for ii, stage := range ui.Project.Stages {
								stage := stage
								panel := ui.Panels[ii]
								panels = append(panels, layout.Flexed(1, func(gtx C) D {
									return panel.Layout(gtx, ui.Th, func() (tickets []layout.ListElement) {
										for ii := range stage.Tickets {
											ticket := stage.Tickets[ii]
											t := (*Ticket)(ui.TicketStates.New(ticket.ID.String(), unsafe.Pointer(&Ticket{})))
											t.Ticket = ticket
											t.Stage = stage.Name
											tickets = append(tickets, func(gtx C, index int) D {
												var focused bool
												if ui.Focus.T != nil && ui.Focus.T.ID == t.ID {
													focused = true
												}
												return t.Layout(gtx, ui.Th, focused)
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
//
// Stages and Tickets form a 2D array, so we can simply iterate over that with
// an index for each dimension.
func (ui *UI) Refocus(d Direction) {
	if len(ui.Project.Stages) == 0 {
		return
	}
	if ui.Focus.T == nil {
		if stage := &ui.Project.Stages[ui.Focus.Stage]; len(stage.Tickets) > 0 {
			ui.Focus.T = &stage.Tickets[ui.Focus.Ticket]
		}
		return
	}
	switch d {
	case NextTicket:
		ui.Focus.Ticket++
		if ui.Focus.Ticket > len(ui.Project.Stages[ui.Focus.Stage].Tickets)-1 {
			ui.Focus.Ticket = 0
		}
	case PreviousTicket:
		ui.Focus.Ticket--
		if ui.Focus.Ticket < 0 {
			ui.Focus.Ticket = len(ui.Project.Stages[ui.Focus.Stage].Tickets) - 1
		}
	}
	if len(ui.Project.Stages) > 1 {
		switch d {
		case NextStage:
			ui.Focus.Stage++
			ui.Focus.Ticket = 0
			if ui.Focus.Stage > len(ui.Project.Stages)-1 {
				ui.Focus.Stage = 0
			}
		case PreviousStage:
			ui.Focus.Stage--
			ui.Focus.Ticket = 0
			if ui.Focus.Stage < 0 {
				ui.Focus.Stage = len(ui.Project.Stages) - 1
			}
		}
	}
	if stage := &ui.Project.Stages[ui.Focus.Stage]; len(stage.Tickets) == 0 {
		ui.Refocus(d)
	} else {
		ui.Focus.T = &stage.Tickets[ui.Focus.Ticket]
	}
}

// Clear resets navigational state.
func (ui *UI) Clear() {
	ui.Modal = nil
	ui.TicketForm = TicketForm{}
	ui.ProjectForm = ProjectForm{}
	ui.DeleteDialog = DeleteDialog{}
	ui.ArchiveProjectConfirmation = ArchiveProjectConfirmation{}
}

// InspectTicket opens the ticket details card for the given ticket.
func (ui *UI) InspectTicket(t kanban.Ticket) {
	ui.TicketDetails.Ticket = t
	ui.Modal = func(gtx C) D {
		return ui.TicketDetails.Layout(gtx, ui.Th)
	}
}

// EditTicket opens the ticket form for editing ticket data.
func (ui *UI) EditTicket(t kanban.Ticket) {
	ui.TicketForm.Edit(t)
	ui.Modal = func(gtx C) D {
		return ui.TicketForm.Layout(gtx, ui.Th, "")
	}
}

// AddTicket opens the ticket form for creating ticket data.
func (ui *UI) AddTicket(stage string) {
	ui.TicketForm.Title.Focus()
	ui.Modal = func(gtx C) D {
		return ui.TicketForm.Layout(gtx, ui.Th, stage)
	}
}

// DeleteTickets opens the confirmation dialog for deleting a ticket.
func (ui *UI) DeleteTicket(t kanban.Ticket) {
	ui.DeleteDialog.Ticket = t
	ui.Modal = func(gtx C) D {
		return ui.DeleteDialog.Layout(gtx, ui.Th)
	}
}

// CreateProject opens the project creation dialog.
func (ui *UI) CreateProject() {
	ui.ProjectForm.Name.Focus()
	ui.Modal = func(gtx C) D {
		return ui.ProjectForm.Layout(gtx, ui.Th)
	}
}

// EditProject opens the project edit form.
func (ui *UI) EditProject() {
	if ui.Project == nil {
		return
	}
	ui.ProjectForm.Edit(ui.Project)
	ui.ProjectForm.Name.Focus()
	ui.Modal = func(gtx C) D {
		return ui.ProjectForm.Layout(gtx, ui.Th)
	}
}

func (ui *UI) ShowArchiveProjectConfirmation() {
	if ui.Project == nil {
		return
	}
	ui.Modal = func(gtx C) D {
		return ui.ArchiveProjectConfirmation.Layout(gtx, ui.Th)
	}
	ui.ArchiveProjectConfirmation.Confirmation.Focus()
}

// Projects is a list of Project entities with added behaviours.
type Projects []kanban.Project

// Find and return project by name.
// Boolean indicates whether the project exists.
func (plist Projects) Find(id uuid.UUID) (*kanban.Project, bool) {
	for ii := range plist {
		if plist[ii].ID == id {
			return &plist[ii], true
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
	// Remove any zeroed out projects because they don't exist anymore.
	for ii, p := range ui.Projects {
		if p.ID == uuid.Nil {
			if len(ui.Projects) > 1 {
				ui.Projects = append(ui.Projects[:ii], ui.Projects[ii+1:]...)
			}
		}
	}
}

// sync any project dependent state when a project has changed.
func (ui *UI) sync() {
	ui.Clear()
	ui.previous = ui.Project
	// Allocate one panel per stage.
	ui.Panels = func() (panels []*control.Panel) {
		for ii, s := range ui.Project.Stages {
			panels = append(panels, &control.Panel{
				Label: s.Name,
				// First 4 panel colors are hardcoded.
				// Where to store UI state? Ideally not alongside the stage, since it's
				// purely a UI concern.
				// If we have more than four stages, just wrap the colors.
				// @improve
				Color: []color.NRGBA{
					{R: 100, B: 100, G: 200, A: 255},
					{R: 100, B: 200, G: 100, A: 255},
					{R: 200, B: 100, G: 100, A: 255},
					{R: 200, B: 200, G: 100, A: 255},
				}[ii%4],
				Thickness: unit.Dp(50),
			})
		}
		return panels
	}()
}
