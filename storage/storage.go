// Package storage specifies a storage interface for Kanban Projects.
// Sub packages implement the interface providing different storage strategies.
package storage

import (
	"git.sr.ht/~jackmordaunt/kanban"
)

// Storer persists Project entities.
type Storer interface {
	// Create a new Project.
	Create(kanban.Project) error
	// Save one or more existing Projects, updating the storage device.
	Save(...kanban.Project) error
	// Load updates the Projects using data from the storage device.
	// Allows caller to allocate and control memory.
	// Avoids copyig.
	Load([]kanban.Project) error
	// Lookup a Project by name.
	Lookup(name string) (kanban.Project, bool, error)
	// List all existing Projects.
	List() ([]kanban.Project, error)
}
