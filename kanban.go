package kanban

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/asdine/storm/v3"
)

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

type IProject interface {
	MakeStage(stage string)
	ListStages() []Stage
	MoveStage(stage string, dir Direction) bool
	AssignTicket(stage string, t Ticket)
	ProgressTicket(t Ticket)
	RegressTicket(t Ticket)
	ListTickets(stage string) []Ticket
	MoveTicket(t Ticket, dir Direction) bool
	FinalizeTicket(t Ticket)
}

// Storage handles serialization of Project entities.
type Storer interface {
	Create(p *Project) error
	Save(p *Project) error
	Load(name string) (*Project, bool, error)
	List() ([]*Project, error)
}

// Project is a context for a given set of tickets.
type Project struct {
	// Name of project, must be unique.
	Name string
	// Stages owned by this project.
	Stages    Stages
	Finalized []Ticket
}

var _ IProject = (*Project)(nil)

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
func (p *Project) AssignTicket(stage string, ticket Ticket) {
	p.Stages.Find(stage).Assign(ticket)
}

// ProgressTicket moves a ticket to the "next" stage.
func (p *Project) ProgressTicket(ticket Ticket) {
	for ii, s := range p.Stages {
		if s.Contains(ticket) {
			if ii < len(p.Stages)-1 {
				p.Stages[ii+1].Assign(p.Stages[ii].Take(ticket))
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
				p.Stages[ii-1].Assign(p.Stages[ii].Take(ticket))
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

// FinalizeTicket renders the ticket "complete" ad moves it into an archive.
func (p *Project) FinalizeTicket(t Ticket) {
	for _, s := range p.Stages {
		if s.Contains(t) {
			s.UnAssign(t)
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

// Assign appends a ticket id to the stage.
func (s *Stage) Assign(ticket Ticket) {
	for _, t := range s.Tickets {
		if t == ticket {
			return
		}
	}
	s.Tickets = append(s.Tickets, ticket)
}

// UnAssign removes a ticket id from the stage.
func (s *Stage) UnAssign(ticket Ticket) {
	for ii, t := range s.Tickets {
		if t == ticket {
			s.Tickets = append(s.Tickets[:ii], s.Tickets[ii+1:]...)
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

// Ticket in a stage.
type Ticket struct {
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

// MapStorer implements in-memory storage for Projects.
type MapStorer struct {
	Data  map[string]Project
	Order []string
	Err   error
}

var _ Storer = (*MapStorer)(nil)

func (s MapStorer) New() *MapStorer {
	return &MapStorer{
		Data: make(map[string]Project),
	}
}

func (s *MapStorer) Create(p *Project) error {
	if len(strings.TrimSpace(p.Name)) == 0 {
		return fmt.Errorf("project name required")
	}
	if _, ok := s.Data[p.Name]; ok {
		return fmt.Errorf("project %q exists", p.Name)
	}
	s.Data[p.Name] = *p
	s.Order = append(s.Order, p.Name)
	return nil
}

func (s *MapStorer) Save(p *Project) error {
	if _, ok := s.Data[p.Name]; ok {
		s.Data[p.Name] = *p
	} else {
		return fmt.Errorf("project %q does not exist", p.Name)
	}
	return nil
}

func (s *MapStorer) Load(name string) (*Project, bool, error) {
	if p, ok := s.Data[name]; ok {
		return &p, ok, nil
	}
	return nil, false, nil
}

func (s *MapStorer) List() (list []*Project, err error) {
	for _, name := range s.Order {
		if p, ok := s.Data[name]; ok {
			list = append(list, &p)
		}
	}
	return list, nil
}

// @todo move storer impl into package.

// StormStorer implements Project storage using storm db.
type StormStorer struct {
	DB *storm.DB
}

// ProjectSchema is a schema representation of a project.
type ProjectSchema struct {
	Name string `storm:"id"`
	Project
}

var _ Storer = (*StormStorer)(nil)

func (s *StormStorer) Create(p *Project) error {
	if len(strings.TrimSpace(p.Name)) == 0 {
		return fmt.Errorf("project name required")
	}
	fmt.Printf("creating project: %v\n ", p)
	return s.DB.Save(&ProjectSchema{Name: p.Name, Project: *p})
}

func (s *StormStorer) Save(p *Project) error {
	if len(strings.TrimSpace(p.Name)) == 0 {
		return fmt.Errorf("project name required")
	}
	return s.DB.Update(&ProjectSchema{Name: p.Name, Project: *p})
}

func (s *StormStorer) Load(name string) (*Project, bool, error) {
	var p ProjectSchema
	if err := s.DB.Find("Name", name, &p); err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return &p.Project, false, nil
		} else {
			return &p.Project, false, err
		}
	}
	return &p.Project, true, nil
}

func (s *StormStorer) List() (list []*Project, err error) {
	var (
		projects []*ProjectSchema
	)
	if err := s.DB.All(&projects); err != nil {
		return nil, fmt.Errorf("loading projects: %v", err)
	}
	for _, p := range projects {
		list = append(list, &p.Project)
	}
	return list, nil
}
