// Package lazy implements a lazy storage that only touches the
// disk when necessary.
package lazy

import (
	"fmt"
	"reflect"

	"git.sr.ht/~jackmordaunt/kanban"
	"git.sr.ht/~jackmordaunt/kanban/storage"
	"git.sr.ht/~jackmordaunt/kanban/storage/mem"
	"git.sr.ht/~jackmordaunt/kanban/storage/storm"
)

var _ storage.Storer = (*Storer)(nil)

// Storer writes to disk when a change has been detected.
type Storer struct {
	Cache *mem.Storer
	Disk  *storm.Storer
}

// Open a lazy storer, initializing the underlying database at the path
// specified.
func Open(path string) (*Storer, error) {
	disk, err := storm.Open(path)
	if err != nil {
		return nil, err
	}
	s := &Storer{
		Cache: mem.New(),
		Disk:  disk,
	}
	return s, s.Populate()
}

// Create a project. Saves to disk.
func (s *Storer) Create(p kanban.Project) error {
	if err := s.Cache.Create(p); err != nil {
		return fmt.Errorf("creating on disk: %v", err)
	}
	if err := s.Disk.Create(p); err != nil {
		return fmt.Errorf("creating on disk: %v", err)
	}
	return nil
}

// Save a project. Only saves to disk if changed.
func (s *Storer) Save(projects ...kanban.Project) error {
	for _, p := range projects {
		old, ok, err := s.Cache.Lookup(p.Name)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("project does not exist: %q", p.Name)
		}
		if !reflect.DeepEqual(p, old) {
			if err := s.Disk.Save(p); err != nil {
				return fmt.Errorf("saving to disk: %v", err)
			}
			return s.Refresh(p.Name)
		}
	}
	return nil
}

// Load a project by name.
// Bool indicates whether a project exists for that name.
func (s *Storer) Lookup(name string) (kanban.Project, bool, error) {
	return s.Cache.Lookup(name)
}

// List projects.
func (s *Storer) List() ([]kanban.Project, error) {
	return s.Disk.List()
}

// Refresh a project entity by loading from disk.
func (s *Storer) Refresh(name string) error {
	p, ok, err := s.Disk.Lookup(name)
	if err != nil {
		return fmt.Errorf("loading from disk: %v", err)
	}
	if !ok {
		return fmt.Errorf("project does not exist: %v", name)
	}
	return s.Cache.Save(p)
}

// Populate cache from disk.
func (s *Storer) Populate() error {
	projects, err := s.Disk.List()
	if err != nil {
		return fmt.Errorf("loading projects from disk: %v", err)
	}
	for _, p := range projects {
		if err := s.Cache.Create(p); err != nil {
			return fmt.Errorf("saving project to cache: %v", err)
		}
	}
	return nil
}

func (s *Storer) Load(projects []kanban.Project) error {
	return s.Disk.Load(projects)
}

func (s *Storer) Close() error {
	return s.Disk.DB.Close()
}
