// Package kanban implements Kanban logic.
//
// Kanban is Project oriented, where a Project holds the context for given set
// of Stages and Tickets.
//
// Projects are independent of each other.
//
// Notes:
//
// Project
// - represents some project that can be broken down into to discrete tasks, described by a name
// - each project has it's own arbitrary pipeline of stages with which tickets move through left-to-right
// - contains an ordered list of stages
// - stages are re-orderable
// - can be renamed
// - can be deleted
//
// Stage
// - represents an important part in the lifecycle of a task, described by a name
// - contains an ordered list of tickets
// - tickets are re-orderable
// - tickets can advance back and forth between stages, typically linearly
// - can be renamed
// - can be deleted
//
// Ticket
// - contains information about a task for a project
// - is unique to a Project and sits within one of it's stages
// - cannot occupy more than one stage
// - can be edited
// - can be deleted
package kanban

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Project is a context for a given set of tickets.
type Project struct {
	ID uuid.UUID
	// Name of the project.
	Name string
	// Stages is the list of stages owned by the project.
	Stages Stages
	// Finalized is a psuedo stage that contains all finalized tickets.
	Finalized []Ticket
}

// MakeStage assigns a ticket to the given stage.
func (p *Project) MakeStage(name string) {
	p.Stages = append(p.Stages, Stage{
		Name: name,
	})
}

func (p *Project) ListStages() []Stage {
	return p.Stages
}

func (p *Project) MoveStage(name string, dir Direction) bool {
	return p.Stages.Swap(name, dir)
}

// AssignTicket assigns a ticket to the given stage.
func (p *Project) AssignTicket(stage string, ticket Ticket) error {
	return p.Stages.Find(stage).Assign(ticket)
}

// Update an existing ticket.
// It is an error to attempt to update a ticket that does not exist.
func (p *Project) UpdateTicket(ticket Ticket) error {
	for _, s := range p.Stages {
		if s.Update(ticket) {
			return nil
		}
	}
	return fmt.Errorf("ticket does not exist: %v", ticket)
}

// ProgressTicket moves a ticket to the "next" stage.
func (p *Project) ProgressTicket(ticket Ticket) {
	for ii, s := range p.Stages {
		if s.Contains(ticket) {
			if ii < len(p.Stages)-1 {
				_ = p.Stages[ii+1].Assign(p.Stages[ii].Take(ticket))
			}
			break
		}
	}
}

// RegressTicket moves a ticket to the "previous" stage.
func (p *Project) RegressTicket(ticket Ticket) {
	for ii, s := range p.Stages {
		if s.Contains(ticket) {
			if ii > 0 {
				_ = p.Stages[ii-1].Assign(p.Stages[ii].Take(ticket))
			}
			break
		}
	}
}

// MoveTicket within a stage.
func (p *Project) MoveTicket(ticket Ticket, dir Direction) bool {
	// @implement
	return false
}

func (p *Project) ListTickets(stage string) []Ticket {
	return p.Stages.Find(stage).Tickets
}

// StageForTicket returns the stage containing the specified ticket.
func (p *Project) StageForTicket(ticket Ticket) *Stage {
	for ii, s := range p.Stages {
		if s.Contains(ticket) {
			return &p.Stages[ii]
		}
	}
	return &Stage{}
}

// FinalizeTicket renders the ticket "complete" and moves it into an archive.
func (p *Project) FinalizeTicket(t Ticket) {
	for ii, s := range p.Stages {
		if s.Contains(t) {
			p.Stages[ii].UnAssign(t)
			p.Finalized = append(p.Finalized, t)
			break
		}
	}
}

// Stage in the kanban pipeline, can hold a number of tickets.
type Stage struct {
	Name    string
	Tickets []Ticket
}

// Assign appends a ticket to the stage with a unique ID.
// Existing tickets will be duplicated, but with different IDs.
func (s *Stage) Assign(ticket Ticket) error {
	if ticket.ID == uuid.Nil {
		id, err := uuid.NewUUID()
		if err != nil {
			return fmt.Errorf("generating ID: %v", err)
		}
		ticket.ID = id
		ticket.Created = time.Now()
	}
	s.Tickets = append(s.Tickets, ticket)
	return nil
}

// UnAssign removes a ticket from the stage.
func (s *Stage) UnAssign(ticket Ticket) {
	for ii, t := range s.Tickets {
		if t == ticket {
			if len(s.Tickets) == 1 {
				s.Tickets = []Ticket{}
			} else {
				s.Tickets = append(s.Tickets[:ii], s.Tickets[ii+1:]...)
			}
		}
	}
}

// Stages is a list of Stage.
type Stages []Stage

// Swap the specified stage in the given direction.
// Returns false when at a boundary, and therefore no swap can occur.
func (stages *Stages) Swap(stage string, dir Direction) bool {
	ii, ok := stages.Index(stage)
	if !ok {
		return false
	}
	if bounds := ii + dir.Next(); bounds < 0 || bounds > len(*stages)-1 {
		return false
	}
	(*stages)[ii], (*stages)[ii+dir.Next()] = (*stages)[ii+dir.Next()], (*stages)[ii]
	return true
}

// Find stage by name.
func (stages *Stages) Find(name string) *Stage {
	for ii, s := range *stages {
		if s.Name == name {
			return &(*stages)[ii]
		}
	}
	return &Stage{}
}

// Index returns the index postition for the stage, false if no stage exists.
func (stages *Stages) Index(name string) (int, bool) {
	for ii, s := range *stages {
		if s.Name == name {
			return ii, true
		}
	}
	return 0, false
}

// Take the specified ticket, if it exists.
// Removes it from the stage.
func (s *Stage) Take(ticket Ticket) Ticket {
	s.UnAssign(ticket)
	return ticket
}

// Contains returns true if the specified ticket exists in the stage.
func (s *Stage) Contains(ticket Ticket) bool {
	for _, t := range s.Tickets {
		if t == ticket {
			return true
		}
	}
	return false
}

// Update a ticket, returning a bool to indicate success.
// False means ticket does not exist and therefore nothing was updated.
func (s *Stage) Update(ticket Ticket) bool {
	for ii, t := range s.Tickets {
		if t.ID == ticket.ID {
			s.Tickets[ii] = ticket
			return true
		}
	}
	return false
}

// Ticket in a stage.
type Ticket struct {
	ID uuid.UUID
	// Title of the ticket.
	Title string
	// Summary contains short and concise overview of the ticket.
	Summary string
	// Details contains the full details of the ticket.
	Details string
	// Created when the ticket was created.
	Created time.Time
}

// Direction encodes mutually exclusive directions.
type Direction int8

const (
	Forward Direction = iota
	Backward
)

// Next returns the direction as a signed integer, where positive is forward.
func (dir Direction) Next() int {
	switch dir {
	case Forward:
		return 1
	case Backward:
		return -1
	}
	return 0
}

// Invert returns the inverse of dir.
func (dir Direction) Invert() Direction {
	switch dir {
	case Forward:
		return Backward
	case Backward:
		return Forward
	}
	return dir
}

func (p *Project) String() string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", *p)
}

// Clone a project ensuring all data is copied.
func (p Project) Clone() Project {
	var (
		stages    = make([]Stage, len(p.Stages))
		finalized = make([]Ticket, len(p.Finalized))
	)
	copy(finalized, p.Finalized)
	for ii, s := range p.Stages {
		tickets := make([]Ticket, len(s.Tickets))
		copy(tickets, s.Tickets)
		stages[ii] = Stage{
			Name:    s.Name,
			Tickets: tickets,
		}
	}
	return Project{
		ID:        p.ID,
		Name:      p.Name,
		Stages:    stages,
		Finalized: finalized,
	}
}

func (p *Project) Eq(other *Project) bool {
	return p.ID == other.ID &&
		p.Name == other.Name &&
		p.Stages.Eq(other.Stages)
}

func (s Stages) Eq(other Stages) bool {
	for ii := range s {
		if !s[ii].Eq(other[ii]) {
			return false
		}
	}
	return true
}

func (s Stage) Eq(other Stage) bool {
	if len(s.Tickets) != len(other.Tickets) {
		return false
	}
	for ii, t := range s.Tickets {
		if t != other.Tickets[ii] {
			return false
		}
	}
	return s.Name == other.Name
}
