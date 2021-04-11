// Package lazy implements a lazy storage that only touches the
// disk when necessary.
package lazy

import (
	"fmt"

	"git.sr.ht/~jackmordaunt/kanban"
	"git.sr.ht/~jackmordaunt/kanban/storage"
	"git.sr.ht/~jackmordaunt/kanban/storage/bolt"
	"git.sr.ht/~jackmordaunt/kanban/storage/mem"
	"github.com/google/uuid"
)

var _ storage.Storer = (*Storer)(nil)

// Storer writes to disk when a change has been detected.
//
// Reads come directly from disk because it is memory mapped.
// Cache simply holds old values so that we can detect for changes
// to know when to write data.
//
// Writes get flushed to disk, hence that is the work being minimized.
type Storer struct {
	Cache *mem.Storer
	*bolt.Storer
}

// Open a lazy storer, initializing the underlying database at the path
// specified.
func Open(path string) (*Storer, error) {
	disk, err := bolt.Open(path)
	if err != nil {
		return nil, err
	}
	s := Storer{
		Cache:  mem.New(),
		Storer: disk,
	}
	if err := s.Populate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save a project. Only saves to disk if changed.
func (s *Storer) Save(projects ...kanban.Project) error {
	var save []kanban.Project
	for _, p := range projects {
		old, ok, err := s.Cache.Find(p.ID)
		if err != nil {
			return err
		}
		if !ok {
			s.Cache.Active.Add(p)
			continue
		}
		if !p.Eq(&old) {
			save = append(save, p)
		}
	}
	if len(save) > 0 {
		if err := s.Storer.Save(save...); err != nil {
			return fmt.Errorf("saving to disk: %w", err)
		}
		return s.Populate()
	}
	return nil
}

// Refresh a project entity by loading from disk.
func (s *Storer) Refresh(id uuid.UUID) error {
	p, ok, err := s.Storer.Find(id)
	if err != nil {
		return fmt.Errorf("loading from disk: %w", err)
	}
	if !ok {
		return fmt.Errorf("project does not exist: %v", id)
	}
	s.Cache.Active.Add(p)
	return nil
}

// Populate cache from disk.
func (s *Storer) Populate() error {
	s.Cache.Clear()
	projects, err := s.Storer.List()
	if err != nil {
		return fmt.Errorf("loading projects from disk: %w", err)
	}
	for _, p := range projects {
		s.Cache.Active.Add(p)
	}
	archived, err := s.Storer.ListArchived()
	if err != nil {
		return fmt.Errorf("loading archived projects from disk: %w", err)
	}
	for _, p := range archived {
		s.Cache.Archived.Add(p)
	}
	return nil
}
