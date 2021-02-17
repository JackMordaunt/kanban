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

// Load/Save to an in-memory cache.
// Write through to disk when data has changed, and refresh the cache.
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
//
// @bug the old data loaded from cache is always up to date with the new
// data, why?
// We need to have an old one and a new one in order to detect changes.
func (s *Storer) Save(p kanban.Project) error {
	old, ok, err := s.Cache.Load(p.Name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("project does not exist: %q", p.Name)
	}
	fmt.Printf("old: %v, p: %v\n", old, p)
	if !reflect.DeepEqual(p, old) {
		fmt.Printf("saving project to disk: %v\n", p)
		if err := s.Disk.Save(p); err != nil {
			return fmt.Errorf("saving to disk: %v", err)
		}
		return s.Refresh(p.Name)
	}
	return nil
}

// Load a project by name.
// Bool indicates whether a project exists for that name.
func (s *Storer) Load(name string) (kanban.Project, bool, error) {
	return s.Cache.Load(name)
}

// List projects.
func (s *Storer) List() ([]kanban.Project, error) {
	return s.Cache.List()
}

// Refresh a project entity by loading from disk.
func (s *Storer) Refresh(name string) error {
	p, ok, err := s.Disk.Load(name)
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

func (s *Storer) Close() error {
	return s.Disk.DB.Close()
}
