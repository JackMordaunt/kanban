package kanban

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/asdine/storm/v3"
)

// Kanban manipulates the model.
type Kanban struct {
	// Data access layer for querying and mutating data.
	Store *storm.DB
}

// ID is a unique identifier encoded as an integer.
type ID int

// Entity is unique schema object that changes over time.
type Entity struct {
	ID      ID `storm:"id,index,increment"`
	Created time.Time
}

// Project is a context for a given set of tickets.
type Project struct {
	Entity `storm:"inline"`
	Name   string `storm:"unique"`
	// Stages lists stage IDs in order.
	Stages Stages
}

type Stages []ID

func (stages *Stages) Swap(id ID, dir Direction) {
	for ii := range *stages {
		if (*stages)[ii] == id {
			if bounds := ii + dir.Next(); bounds < 0 || bounds > len(*stages)-1 {
				return
			}
			(*stages)[ii], (*stages)[ii+dir.Next()] = (*stages)[ii+dir.Next()], (*stages)[ii]
		}
	}
}

// Stage in the kanban pipeline, can hold a number of tickets.
type Stage struct {
	Entity `storm:"inline"`
	Name   string
	// Tickest lists ticket IDs in order.
	Tickets []ID // @Todo abstract into "reorderable list", to use with project stage list as well.
}

func (s *Stage) Assign(ticket ID) {
	for _, t := range s.Tickets {
		if t == ticket {
			return
		}
	}
	s.Tickets = append(s.Tickets, ticket)
}

func (s *Stage) UnAssign(ticket ID) {
	for ii, t := range s.Tickets {
		if t == ticket {
			s.Tickets = append(s.Tickets[:ii], s.Tickets[ii+1:]...)
		}
	}
}

// FinalisedTicket is an inactive ticket kept for analytic purposes.
type FinalisedTicket = Ticket

// Ticket in a stage.
type Ticket struct {
	Entity  `storm:"inline"`
	Project ID
	Stage   ID

	// Title of the ticket.
	Title string
	// Summary contains short and concise overview of the ticket.
	Summary string
	// Details contains the full details of the ticket.
	Details string
}

// ListStages returns a list of stages.
func (k Kanban) ListStages(projectID ID) (stages []Stage, err error) {
	var (
		project Project
	)
	if err := k.Store.Find("ID", projectID, &project); err != nil {
		return nil, fmt.Errorf("loading project: %v", err)
	}
	for _, stageID := range project.Stages {
		var (
			stage Stage
		)
		if err := k.Store.Find("ID", stageID, &stage); err != nil {
			return stages, fmt.Errorf("loading stage: %v", err)
		}
		stages = append(stages, stage)
	}
	return stages, nil
}

// NextStage returns the stage that follows the specified one.
func (k Kanban) NextStage(projectID ID, current ID) (string, error) {
	stage, err := k.NextStageForDirection(projectID, current, Forward)
	return stage.Name, err
}

// NextStage returns the stage that preceeds the specified one.
func (k Kanban) PreviousStage(projectID ID, current ID) (string, error) {
	stage, err := k.NextStageForDirection(projectID, current, Backward)
	return stage.Name, err
}

// NextStageForDirection gets the next stage in the given direction.
func (k Kanban) NextStageForDirection(projectID ID, current ID, dir Direction) (next Stage, err error) {
	var (
		project Project
	)
	if err := k.Store.Find("ID", projectID, &project); err != nil {
		return Stage{}, fmt.Errorf("finding project: %v", err)
	}
	for ii, stage := range project.Stages {
		if stage == current {
			// @Todo bounds check.
			return next, k.Store.Find("ID", project.Stages[ii+dir.Next()], &next)
		}
	}
	return next, err
}

// MoveStage moves a stage one place in the given direction.
func (k Kanban) MoveStage(projectID ID, id ID, dir Direction) error {
	var (
		project Project
	)
	if err := k.Store.Find("ID", projectID, &project); err != nil {
		return fmt.Errorf("finding project: %v", err)
	}
	project.Stages.Swap(id, dir)
	if err := k.Store.Save(&project); err != nil {
		return fmt.Errorf("saving project: %v", err)
	}
	return nil
}

// Stage returns a stage by the given name.
// Creates an empty stage if it doesn't exist.
func (k *Kanban) Stage(name string) (stage Stage, err error) {
	err = k.Store.Find("Name", name, &stage)
	if errors.Is(err, storm.ErrNotFound) {
		if err := k.Store.Save(&stage); err != nil {
			return stage, err
		}
		return k.Stage(name)
	}
	return stage, err
}

// Move a ticket to the specified stage.
// Assigns to the bottom of the target stage.
func (k *Kanban) Move(stageID ID, ticketID ID) error {
	var (
		ticket       Ticket
		currentStage Stage
		targetStage  Stage
	)
	if err := k.Store.Find("ID", ticketID, &ticket); err != nil {
		return err
	}
	if err := k.Store.Find("ID", ticket.Stage, &currentStage); err != nil {
		return err
	}
	if err := k.Store.Find("ID", stageID, &targetStage); err != nil {
		return err
	}
	ticket.Stage = targetStage.ID
	currentStage.UnAssign(ticketID)
	targetStage.Assign(ticketID)
	if err := k.Store.Save(&ticket); err != nil {
		return err
	}
	if err := k.Store.Save(&currentStage); err != nil {
		return err
	}
	if err := k.Store.Save(&targetStage); err != nil {
		return err
	}
	return nil
}

// Progress a ticket to the next stage.
func (k *Kanban) Progress(ticketID ID) error {
	var (
		ticket  Ticket
		project Project
		stageID ID
	)
	if err := k.Store.Find("ID", ticketID, &ticket); err != nil {
		return err
	}
	if err := k.Store.Find("ID", ticket.Project, &project); err != nil {
		return err
	}
	for ii, id := range project.Stages {
		if id == ticket.Stage {
			stageID = project.Stages[ii+1]
		}
	}
	return k.Move(stageID, ticket.ID)
}

// Regress a ticket to the previous stage.
func (k *Kanban) Regress(ticketID ID) error {
	var (
		ticket  Ticket
		project Project
		stageID ID
	)
	if err := k.Store.Find("ID", ticketID, &ticket); err != nil {
		return err
	}
	if err := k.Store.Find("ID", ticket.Project, &project); err != nil {
		return err
	}
	for ii, id := range project.Stages {
		if id == ticket.Stage {
			stageID = project.Stages[ii-1]
		}
	}
	return k.Move(stageID, ticket.ID)
}

// Assign a ticket to a stage.
func (k *Kanban) Assign(name string, ticket Ticket) error {
	var (
		stage Stage
	)
	if err := k.Store.Find("Name", name, &stage); err != nil {
		return fmt.Errorf("finding stage %q: %v", name, err)
	}
	stage.Assign(ticket.ID)
	ticket.Stage = stage.ID
	if err := k.Store.Save(&ticket); err != nil {
		return fmt.Errorf("saving ticket: %v", err)
	}
	if err := k.Store.Update(&stage); err != nil {
		return fmt.Errorf("saving stage: %v", err)
	}
	return nil
}

// Finalize a ticket.
// Either the ticket was completed, made irrelevant, or faulty in some manner.
func (k *Kanban) Finalize(ticketID ID) error {
	var (
		ticket Ticket
	)
	if err := k.Store.Find("ID", ticketID, &ticket); err != nil {
		return fmt.Errorf("ticket not exist: %v", err)
	}
	if err := k.Store.DeleteStruct(&ticket); err != nil {
		return fmt.Errorf("deleting active ticket: %v", err)
	}
	return k.Store.Save(FinalisedTicket(ticket))
}

func (k *Kanban) Update(ticket Ticket) error {
	return k.Store.Update(&ticket)
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

// None reports whether the ID represents a valid entity or is a zero value.
func (id ID) None() bool {
	return id < 1
}

func (id ID) String() string {
	return strconv.Itoa(int(id))
}
