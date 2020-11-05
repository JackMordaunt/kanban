package kanban

import (
	"fmt"
	"time"

	"github.com/asdine/storm/v3"
)

// Kanban manipulates the model.
type Kanban struct {
	Store *storm.DB
}

// Stage in the kanban pipeline, can hold a number of tickets.
type Stage struct {
	ID      int `storm:"id,index"`
	Name    string
	Tickets []Ticket
}

// Ticket in a stage.
type Ticket struct {
	ID         int
	Title      string
	Category   string
	Summary    string
	Details    string
	References []int
	Created    time.Time
}

// ListStages returns a list of stages.
func (k Kanban) ListStages() ([]Stage, error) {
	var stages []Stage
	if err := k.Store.AllByIndex("ID", &stages); err != nil {
		return nil, fmt.Errorf("collecting stage: %w", err)
	}
	return stages, nil
}

// NextStage returns the stage that follows the specified one.
func (k Kanban) NextStage(current string) (string, error) {
	stages, err := k.ListStages()
	if err != nil {
		return "", err
	}
	for ii, stage := range stages {
		if stage.Name == current && ii < len(stages)-1 {
			return stages[ii+1].Name, nil
		}
	}
	return current, fmt.Errorf("no more stages after: %q", current)
}

// NextStage returns the stage that preceeds the specified one.
func (k Kanban) PreviousStage(current string) (string, error) {
	stages, err := k.ListStages()
	if err != nil {
		return "", err
	}
	for ii, stage := range stages {
		if stage.Name == current && ii > 0 {
			return stages[ii-1].Name, nil
		}
	}
	return current, fmt.Errorf("no more stages before: %q", current)
}

// Stage returns a stage by the given name.
// Creates an empty stage if it doesn't exist.
func (k *Kanban) Stage(name string) (Stage, error) {
	stages, err := k.ListStages()
	if err != nil {
		return Stage{}, err
	}
	for _, stage := range stages {
		if stage.Name == name {
			return stage, nil
		}
	}
	id, err := k.nextID()
	if err != nil {
		return Stage{}, err
	}
	stage := Stage{ID: id, Name: name}
	if err := k.Store.Save(&stage); err != nil {
		return Stage{}, err
	}
	return stage, nil
}

func (k *Kanban) Move(stage string, ticket int) error {
	stages, err := k.ListStages()
	if err != nil {
		return err
	}
	for ii := range stages {
		s := stages[ii]
		for kk := range s.Tickets {
			t := s.Tickets[kk]
			if t.ID == ticket {
				if err := k.Delete(ticket); err != nil {
					return fmt.Errorf("deleting: %w", err)
				}
				return k.Assign(stage, t)
			}
		}
	}
	return fmt.Errorf("ticket %q does not exist", ticket)
}

// Progress a ticket to the next stage.
func (k *Kanban) Progress(ticket int) error {
	stage, err := k.StageFor(ticket)
	if err != nil {
		return fmt.Errorf("finding stage: %w", err)
	}
	next, err := k.NextStage(stage)
	if err != nil {
		return fmt.Errorf("loading stage: %w", err)
	}
	if err := k.Move(next, ticket); err != nil {
		return fmt.Errorf("moving ticket: %w", err)
	}
	return nil
}

// Regress a ticket to the previous stage.
func (k *Kanban) Regress(ticket int) error {
	stage, err := k.StageFor(ticket)
	if err != nil {
		return err
	}
	next, err := k.PreviousStage(stage)
	if err != nil {
		return err
	}
	return k.Move(next, ticket)
}

func (k *Kanban) StageFor(ticket int) (string, error) {
	stages, err := k.ListStages()
	if err != nil {
		return "", err
	}
	for _, s := range stages {
		for _, t := range s.Tickets {
			if t.ID == ticket {
				return s.Name, nil
			}
		}
	}
	return "", fmt.Errorf("ticket %q does not exist", ticket)
}

func (k *Kanban) Assign(name string, ticket Ticket) error {
	if ticket.ID == 0 {
		id, err := k.nextID()
		if err != nil {
			return fmt.Errorf("generating ID: %w", err)
		}
		ticket.ID = id
		ticket.Created = time.Now()
	}
	stage, err := k.Stage(name)
	if err != nil {
		return fmt.Errorf("finding stage: %w", err)
	}
	stage.Tickets = append(stage.Tickets, ticket)
	return k.Store.Update(&stage)
}

func (k *Kanban) Delete(ticket int) error {
	stages, err := k.ListStages()
	if err != nil {
		return err
	}
	for kk := range stages {
		s := &stages[kk]
		for ii := range s.Tickets {
			if s.Tickets[ii].ID == ticket {
				s.Tickets = append(s.Tickets[:ii], s.Tickets[ii+1:]...)
				return k.Store.Update(s)
			}
		}
	}
	return fmt.Errorf("ticket %q does not exist", ticket)
}

func (k *Kanban) Update(ticket Ticket) error {
	stages, err := k.ListStages()
	if err != nil {
		return err
	}
	for _, s := range stages {
		for ii := range s.Tickets {
			if s.Tickets[ii].ID == ticket.ID {
				s.Tickets[ii] = ticket
				return k.Store.Update(&s)
			}
		}
	}
	return fmt.Errorf("ticket %q does not exist", ticket)
}

func (k *Kanban) nextID() (int, error) {
	stages, err := k.ListStages()
	if err != nil {
		return -1, err
	}
	var max int
	for _, stage := range stages {
		for _, t := range stage.Tickets {
			if int(t.ID) > max {
				max = int(t.ID)
			}
		}
	}
	return max + 1, nil
}
