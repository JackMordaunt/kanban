// Package storage specifies a storage interface for Kanban Projects.
// Sub packages implement the interface providing different storage strategies.
package storage

import (
	"git.sr.ht/~jackmordaunt/kanban"
)

// Storer persists Project entities.
type Storer interface {
	Create(p kanban.Project) error
	Save(p *kanban.Project) error
	Load(name string) (*kanban.Project, bool, error)
	List() ([]*kanban.Project, error)
}
