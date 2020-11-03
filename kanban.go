package kanban

// Kanban engine that drives the model.
type Engine struct {
	stages []Stage
}

// Stages returns a list of stages.
func (eng Engine) Stages() ([]Stage, error) {
	return eng.stages, nil
}

// Stage returns a stage by the given name.
// Creates an empty stage if it doesn't exist.
func (eng *Engine) Stage(name string) (Stage, error) {
	for _, stage := range eng.stages {
		if stage.Name == name {
			return stage, nil
		}
	}
	eng.stages = append(eng.stages, Stage{Name: name})
	return eng.stages[len(eng.stages)-1], nil
}

func (eng *Engine) Move(stage string, ticket ID) error {
	return nil
}

func (eng *Engine) Assign(stage string, ticket Ticket) error {
	ticket.ID = eng.nextID()
	for ii := range eng.stages {
		s := &eng.stages[ii]
		if s.Name == stage {
			s.Tickets = append(s.Tickets, ticket)
			return nil
		}
	}
	eng.stages = append(eng.stages, Stage{
		Name:    stage,
		Tickets: []Ticket{ticket},
	})
	return nil
}

func (eng *Engine) Delete(ticket ID) error {
	for _, s := range eng.stages {
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
	for _, s := range eng.stages {
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
	for _, stage := range eng.stages {
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
