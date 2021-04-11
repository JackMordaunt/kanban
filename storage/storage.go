// Package storage specifies a storage interface for Kanban Projects.
// Sub packages implement the interface providing different storage strategies.
package storage

import (
	"git.sr.ht/~jackmordaunt/kanban"
	"github.com/google/uuid"
)

// Storer persists Project entities.
type Storer interface {
	// Create a new Project.
	Create(kanban.Project) error
	// Save one or more existing Projects, updating the storage device.
	Save(...kanban.Project) error
	// Load updates the Projects using data from the storage device.
	// Allows caller to allocate and control memory.
	// Avoids copying.
	Load([]kanban.Project) error
	// Find a Project by ID.
	Find(id uuid.UUID) (kanban.Project, bool, error)
	// List all existing Projects.
	List() ([]kanban.Project, error)
	// Count returns the number of projects that exist in the store.
	Count() (int, error)
	// Archive a project.
	// An archived project will have it's data saved, but won't show up under
	// normal queries.
	Archive(uuid.UUID) error
	// ListArchived lists all archived projects.
	ListArchived() ([]kanban.Project, error)
	// Restore takes an archived project and makes it live again.
	Restore(uuid.UUID) error
}
