// Package lazy implements a lazy storage that only touches the
// disk when necessary.
package lazy

import (
	"fmt"
	"reflect"

	"git.sr.ht/~jackmordaunt/kanban"
	"git.sr.ht/~jackmordaunt/kanban/storage"
	"git.sr.ht/~jackmordaunt/kanban/storage/bolt"
	"git.sr.ht/~jackmordaunt/kanban/storage/mem"
	"github.com/google/uuid"
)

var _ storage.Storer = (*Storer)(nil)

// Storer writes to disk when a change has been detected.
type Storer struct {
	Cache *mem.Storer
	Disk  *bolt.Storer
}

// Open a lazy storer, initializing the underlying database at the path
// specified.
func Open(path string) (*Storer, error) {
	disk, err := bolt.Open(path)
	if err != nil {
		return nil, err
	}
	s := Storer{
		Cache: mem.New(),
		Disk:  disk,
	}
	return &s, s.Populate()
}

// Create a project. Saves to disk.
func (s *Storer) Create(p kanban.Project) error {
	if err := s.Cache.Create(p); err != nil {
		return fmt.Errorf("creating on disk: %w", err)
	}
	if err := s.Disk.Create(p); err != nil {
		return fmt.Errorf("creating on disk: %w", err)
	}
	return nil
}

// Save a project. Only saves to disk if changed.
func (s *Storer) Save(projects ...kanban.Project) error {
	for _, p := range projects {
		old, ok, err := s.Cache.Find(p.ID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("project does not exist: %q", p.Name)
		}
		if !reflect.DeepEqual(p, old) {
			if err := s.Disk.Save(p); err != nil {
				return fmt.Errorf("saving to disk: %w", err)
			}
			return s.Refresh(p.ID)
		}
	}
	return nil
}

// Load a project by ID.
// Bool indicates whether a project exists for that ID.
func (s *Storer) Find(id uuid.UUID) (kanban.Project, bool, error) {
	return s.Cache.Find(id)
}

// List projects.
func (s *Storer) List() ([]kanban.Project, error) {
	return s.Cache.List()
}

// Refresh a project entity by loading from disk.
func (s *Storer) Refresh(id uuid.UUID) error {
	p, ok, err := s.Disk.Find(id)
	if err != nil {
		return fmt.Errorf("loading from disk: %w", err)
	}
	if !ok {
		return fmt.Errorf("project does not exist: %v", id)
	}
	return s.Cache.Save(p)
}

// Populate cache from disk.
func (s *Storer) Populate() error {
	projects, err := s.Disk.List()
	if err != nil {
		return fmt.Errorf("loading projects from disk: %w", err)
	}
	for _, p := range projects {
		if err := s.Cache.Create(p); err != nil {
			return fmt.Errorf("saving project to cache: %w", err)
		}
	}
	archived, err := s.Disk.ListArchived()
	if err != nil {
		return fmt.Errorf("loading archived projects from disk: %w", err)
	}
	for _, p := range archived {
		if err := func() error {
			if err := s.Cache.Create(p); err != nil {
				return err
			}
			if err := s.Cache.Archive(p.ID); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("saving archived projects to cache: %w", err)
		}
	}
	return nil
}

func (s *Storer) Load(projects []kanban.Project) error {
	return s.Cache.Load(projects)
}

func (s *Storer) Close() error {
	return s.Disk.DB.Close()
}

func (s *Storer) Count() (int, error) {
	return s.Cache.Count()
}

func (s *Storer) Archive(id uuid.UUID) error {
	if err := s.Disk.Archive(id); err != nil {
		return err
	}
	return s.Populate()
}

func (s *Storer) Restore(id uuid.UUID) error {
	if err := s.Disk.Restore(id); err != nil {
		return err
	}
	return s.Populate()
}

func (s *Storer) ListArchived() ([]kanban.Project, error) {
	return s.Cache.ListArchived()
}
