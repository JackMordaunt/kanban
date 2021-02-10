// @Note data lifecycle idea: frame by frame sync
// 1. load data at start of frame; pass it in to the ui context
// 2. save data at end of frame, every frame
// 3. mutate data with plain methods knowing that mutations will be saved at a known point
//
// let data be heirarchical eg projects -> stages -> tickets
//
// How are we accessing the data mostly?
// Active Project gets loaded every frame.
// All stages and tickets for the active project need to be read every frame.
// Mutations occur async.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"git.sr.ht/~jackmordaunt/kanban"
	"github.com/asdine/storm/v3"

	"gioui.org/font/gofont"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/control"
	"git.sr.ht/~jackmordaunt/kanban/cmd/kanban/state"
	"git.sr.ht/~jackmordaunt/kanban/icons"

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
		db, err := storm.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening data file: %w", err)
		}
		if err := db.Init(&kanban.Ticket{}); err != nil {
			return nil, err
		}
		if err := db.Init(&kanban.Project{}); err != nil {
			return nil, err
		}
		return db, nil
	}()
	if err != nil {
		log.Fatalf("error: initializing data: %v", err)
	}
	defer db.Close()
	go func() {
		ui := UI{
			Window:  app.NewWindow(app.Title("Kanban")),
			Th:      material.NewTheme(gofont.Collection()),
			Storage: &kanban.StormStorer{DB: db},
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
	Storage kanban.Storer
	Th      *material.Theme
	Project *kanban.Project

	// @Todo panels shouldn't be stateful.
	Panels        []control.Panel
	Rail          control.Rail
	TicketStates  state.Map
	Modal         layout.Widget
	TicketForm    TicketForm
	TicketDetails TicketDetails
	DeleteDialog  DeleteDialog

	// FocusedTicket struct {
	// 	ID    kanban.ID
	// 	Index int
	// 	Stage kanban.ID
	// }

	CreateProjectButton widget.Clickable
	ProjectForm         ProjectForm
}

func (ui *UI) Loop() error {
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
			ui.Update(gtx)
			ui.Layout(gtx)
			event.Frame(gtx.Ops)
		}
	}
	return ui.Shutdown()
}

// Shutdown does cleanup.
func (ui UI) Shutdown() error {
	if err := ui.Storage.Save(ui.Project); err != nil {
		return fmt.Errorf("saving project: %v", err)
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
				// @Cleanup
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
	for ii := range ui.Panels {
		panel := &ui.Panels[ii]
		if panel.CreateTicket.Clicked() {
			ui.AddTicket(panel.Label)
		}
	}
	for _, s := ui.TicketStates.Next(); ui.TicketStates.More(); _, s = ui.TicketStates.Next() {
		t := (*Ticket)(s)
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
		// if err != nil {
		// 	fmt.Printf("error: %s\n", err)
		// } else {
		// 	if assign := ui.TicketForm.Stage != ""; assign {
		// 		ui.Project.AssignTicket(ui.TicketForm.Stage, ticket)
		// 	} else {
		// 		ui.Project.Update(ticket)
		// 	}
		// }
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
	if ui.ProjectForm.Cancel.Clicked() {
		ui.Clear()
	}
	if ui.ProjectForm.Submit.Clicked() {
		if err := ui.Storage.Create(&kanban.Project{
			Name: ui.ProjectForm.Name.Text(),
		}); err != nil {
			log.Printf("creating new project: %v", err)
		}
		ui.Clear()
	}
	if p, ok := ui.Rail.Selected(); ok {
		project, ok, err := ui.Storage.Load(p)
		if err != nil {
			log.Printf("loading project %q: %v", p, err)
		}
		if ok {
			ui.Project = project
		}

	}
}

func (ui *UI) Layout(gtx C) D {
	key.InputOp{Tag: ui}.Add(gtx.Ops)
	return layout.Flex{Axis: layout.Horizontal}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			gtx.Constraints.Max.X = gtx.Px(unit.Dp(80))
			gtx.Constraints.Min.X = 0
			var (
				rc []control.RailChild
			)
			// @cleanup
			// if err := ui.Project.AllByIndex("ID", &projects); err != nil {
			// 	log.Printf("error: loading projects: %v", err)
			// }
			// for _, p := range projects {
			// 	p := p
			// 	rc = append(rc, Destination(p.ID.String(), func(gtx C) D {
			// 		return layout.Stack{
			// 			Alignment: layout.Center,
			// 		}.Layout(
			// 			gtx,
			// 			layout.Stacked(func(gtx C) D {
			// 				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx C) D {
			// 					return material.Label(ui.Th, unit.Dp(16), p.Name).Layout(gtx)
			// 				})
			// 			}),
			// 			layout.Expanded(func(gtx C) D {
			// 				cs := gtx.Constraints
			// 				if p.ID == ui.ActiveProject {
			// 					return util.Rect{
			// 						Color: color.NRGBA{A: 100},
			// 						Size:  f32.Pt(float32(cs.Max.X), float32(cs.Min.Y)),
			// 					}.Layout(gtx)
			// 				}
			// 				return D{Size: image.Point{X: cs.Max.X, Y: cs.Min.Y}}
			// 			}),
			// 		)
			// 	}))
			// }
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
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Stack{}.Layout(
				gtx,
				layout.Stacked(func(gtx C) D {
					if ui.Project == nil {
						return D{}
					}
					ui.TicketStates.Begin()
					var panels []layout.FlexChild
					// @cleanup
					// var (
					// 	project kanban.Project
					// 	stage   kanban.Stage
					// 	ticket  kanban.Ticket
					// 	t       *Ticket
					// )
					// // @fixme show project creation hint when there are no projects.
					// if err := ui.Project.One("ID", ui.ActiveProject, &project); err != nil {
					// 	log.Printf("error: project %v", err)
					// }
					// for _, id := range project.Stages {
					// 	if err := ui.Project.One("ID", id, &stage); err != nil {
					// 		log.Printf("error: stage %v", err)
					// 	}
					// 	// render the stage panel.
					// 	for _, id := range stage.Tickets {
					// 		if err := ui.Project.One("ID", id, &ticket); err != nil {
					// 			log.Printf("error: ticket %v", err)
					// 		}
					// 		t = (*Ticket)(ui.TicketStates.New(strconv.Itoa(int(id)), unsafe.Pointer(&Ticket{})))
					// 		t.Ticket = ticket
					// 		t.Stage = stage.Name
					// 		panels = append(panels, layout.Flexed(1, func(gtx C) D {
					// 			if ui.FocusedTicket.ID == id {
					// 				return widget.Border{
					// 					Color: color.NRGBA{B: 200, A: 200},
					// 					Width: unit.Dp(2),
					// 				}.Layout(gtx, func(gtx C) D {
					// 					return t.Layout(gtx, ui.Th)
					// 				})
					// 			}
					// 			return t.Layout(gtx, ui.Th)
					// 		}))
					// 	}
					// }
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
