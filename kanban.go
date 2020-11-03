package kanban

import (
	"fmt"
	"strconv"
)

// Kanban manipulates the model.
type Kanban struct {
	Stages []Stage
}

// ListStages returns a list of stages.
func (k Kanban) ListStages() ([]Stage, error) {
	return k.Stages, nil
}

// NextStage returns the stage that follows the specified one.
func (k Kanban) NextStage(current string) (string, error) {
	for ii, stage := range k.Stages {
		if stage.Name == current && ii < len(k.Stages)-1 {
			return k.Stages[ii+1].Name, nil
		}
	}
	return current, fmt.Errorf("no more stages after: %q", current)
}

// NextStage returns the stage that preceeds the specified one.
func (k Kanban) PreviousStage(current string) (string, error) {
	for ii, stage := range k.Stages {
		if stage.Name == current && ii > 0 {
			return k.Stages[ii-1].Name, nil
		}
	}
	return current, fmt.Errorf("no more stages before: %q", current)
}

// Stage returns a stage by the given name.
// Creates an empty stage if it doesn't exist.
func (k *Kanban) Stage(name string) (Stage, error) {
	for _, stage := range k.Stages {
		if stage.Name == name {
			return stage, nil
		}
	}
	k.Stages = append(k.Stages, Stage{Name: name})
	return k.Stages[len(k.Stages)-1], nil
}

func (k *Kanban) Move(stage string, ticket ID) error {
	var found *Ticket
	for ii := range k.Stages {
		stage := k.Stages[ii]
		for kk := range stage.Tickets {
			t := stage.Tickets[kk]
			if t.ID == ticket {
				found = &t
				if err := k.Delete(ticket); err != nil {
					return err
				}
				break
			}
		}
	}
	if found == nil {
		return fmt.Errorf("ticket %q does not exist", ticket)
	}
	return k.Assign(stage, *found)
}

// Progress a ticket to the next stage.
func (k *Kanban) Progress(ticket ID) error {
	stage, err := k.StageFor(ticket)
	if err != nil {
		return err
	}
	next, err := k.NextStage(stage)
	if err != nil {
		return err
	}
	return k.Move(next, ticket)
}

// Regress a ticket to the previous stage.
func (k *Kanban) Regress(ticket ID) error {
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

func (k *Kanban) StageFor(ticket ID) (string, error) {
	for _, s := range k.Stages {
		for _, t := range s.Tickets {
			if t.ID == ticket {
				return s.Name, nil
			}
		}
	}
	return "", fmt.Errorf("ticket %q does not exist", ticket)
}

func (k *Kanban) Assign(stage string, ticket Ticket) error {
	ticket.ID = k.nextID()
	for ii := range k.Stages {
		s := &k.Stages[ii]
		if s.Name == stage {
			s.Tickets = append(s.Tickets, ticket)
			return nil
		}
	}
	k.Stages = append(k.Stages, Stage{
		Name:    stage,
		Tickets: []Ticket{ticket},
	})
	return nil
}

func (k *Kanban) Delete(ticket ID) error {
	for kk := range k.Stages {
		s := &k.Stages[kk]
		for ii := range s.Tickets {
			if s.Tickets[ii].ID == ticket {
				s.Tickets = append(s.Tickets[:ii], s.Tickets[ii+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (k *Kanban) Update(ticket Ticket) error {
	for _, s := range k.Stages {
		for ii := range s.Tickets {
			if s.Tickets[ii].ID == ticket.ID {
				s.Tickets[ii] = ticket
				return nil
			}
		}
	}
	return nil
}

func (k *Kanban) nextID() ID {
	var max int
	for _, stage := range k.Stages {
		for _, t := range stage.Tickets {
			if int(t.ID) > max {
				max = int(t.ID)
			}
		}
	}
	return ID(max + 1)
}

type ID int

// Stage in the kanban pipeline, can hold a number of tickets.
type Stage struct {
	Name    string
	Tickets []Ticket
}

// Ticket in a stage.
type Ticket struct {
	ID         ID
	Title      string
	Category   string
	Summary    string
	Details    string
	References []ID
}

func (id ID) String() string {
	return strconv.Itoa(int(id))
}
