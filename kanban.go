package kanban

import (
	"fmt"
	"strconv"
)

// Kanban engine that drives the model.
type Engine struct {
	Stages []Stage
}

// ListStages returns a list of stages.
func (eng Engine) ListStages() ([]Stage, error) {
	return eng.Stages, nil
}

// NextStage returns the stage that follows the specified one.
func (eng Engine) NextStage(current string) (string, error) {
	for ii, stage := range eng.Stages {
		if stage.Name == current && ii < len(eng.Stages)-1 {
			return eng.Stages[ii+1].Name, nil
		}
	}
	return current, fmt.Errorf("no more stages after: %q", current)
}

// NextStage returns the stage that preceeds the specified one.
func (eng Engine) PreviousStage(current string) (string, error) {
	for ii, stage := range eng.Stages {
		if stage.Name == current && ii > 0 {
			return eng.Stages[ii-1].Name, nil
		}
	}
	return current, fmt.Errorf("no more stages before: %q", current)
}

// Stage returns a stage by the given name.
// Creates an empty stage if it doesn't exist.
func (eng *Engine) Stage(name string) (Stage, error) {
	for _, stage := range eng.Stages {
		if stage.Name == name {
			return stage, nil
		}
	}
	eng.Stages = append(eng.Stages, Stage{Name: name})
	return eng.Stages[len(eng.Stages)-1], nil
}

func (eng *Engine) Move(stage string, ticket ID) error {
	var found *Ticket
	for ii := range eng.Stages {
		stage := eng.Stages[ii]
		for kk := range stage.Tickets {
			t := stage.Tickets[kk]
			if t.ID == ticket {
				found = &t
				if err := eng.Delete(ticket); err != nil {
					return err
				}
				break
			}
		}
	}
	if found == nil {
		return fmt.Errorf("ticket %q does not exist", ticket)
	}
	return eng.Assign(stage, *found)
}

func (eng *Engine) Assign(stage string, ticket Ticket) error {
	ticket.ID = eng.nextID()
	for ii := range eng.Stages {
		s := &eng.Stages[ii]
		if s.Name == stage {
			s.Tickets = append(s.Tickets, ticket)
			return nil
		}
	}
	eng.Stages = append(eng.Stages, Stage{
		Name:    stage,
		Tickets: []Ticket{ticket},
	})
	return nil
}

func (eng *Engine) Delete(ticket ID) error {
	for kk := range eng.Stages {
		s := &eng.Stages[kk]
		for ii := range s.Tickets {
			if s.Tickets[ii].ID == ticket {
				s.Tickets = append(s.Tickets[:ii], s.Tickets[ii+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (eng *Engine) Update(ticket Ticket) error {
	for _, s := range eng.Stages {
		for ii := range s.Tickets {
			if s.Tickets[ii].ID == ticket.ID {
				s.Tickets[ii] = ticket
				return nil
			}
		}
	}
	return nil
}

func (eng *Engine) nextID() ID {
	var max int
	for _, stage := range eng.Stages {
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
