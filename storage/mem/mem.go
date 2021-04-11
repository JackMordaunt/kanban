package mem

import (
	"fmt"

	"github.com/google/uuid"

	"git.sr.ht/~jackmordaunt/kanban"
)

// var _ storage.Storer = (*Storer)(nil)

// Storer implements in-memory storage for Projects.
type Storer struct {
	Data          map[uuid.UUID]kanban.Project
	Order         []uuid.UUID
	Archived      map[uuid.UUID]kanban.Project
	ArchivedOrder []uuid.UUID
}

func New() *Storer {
	return &Storer{
		Data:     make(map[uuid.UUID]kanban.Project),
		Archived: make(map[uuid.UUID]kanban.Project),
	}
}

func (s *Storer) Create(p kanban.Project) error {
	if _, ok := s.Data[p.ID]; ok {
		return fmt.Errorf("project %q exists", p.Name)
	}
	s.Data[p.ID] = p
	s.Order = append(s.Order, p.ID)
	return nil
}

func (s *Storer) Save(projects ...kanban.Project) error {
	for _, p := range projects {
		if _, ok := s.Data[p.ID]; ok {
			s.Data[p.ID] = p
		} else {
			return fmt.Errorf("project %q does not exist", p.Name)
		}
	}
	return nil
}

func (s *Storer) Find(id uuid.UUID) (kanban.Project, bool, error) {
	for _, p := range s.Data {
		if p.ID == id {
			return s.Data[p.ID], true, nil
		}
	}
	return kanban.Project{}, false, nil
}

func (s *Storer) Count() (int, error) {
	return len(s.Data), nil
}

func (s *Storer) List() (list []kanban.Project, err error) {
	for _, id := range s.Order {
		if p, ok := s.Data[id]; ok {
			list = append(list, p)
		}
	}
	return list, nil
}

func (s *Storer) Load(projects []kanban.Project) error {
	for ii := range projects {
		p := s.Data[projects[ii].ID]
		projects[ii] = p
	}
	return nil
}

func (s *Storer) Archive(id uuid.UUID) error {
	p, ok := s.Data[id]
	if !ok {
		return nil
	}
	delete(s.Data, id)
	s.Archived[id] = p
	return nil
}

func (s *Storer) Restore(id uuid.UUID) error {
	p, ok := s.Archived[id]
	if !ok {
		return nil
	}
	delete(s.Archived, id)
	s.Data[id] = p
	return nil
}

func (s *Storer) ListArchived() (list []kanban.Project, err error) {
	for _, id := range s.ArchivedOrder {
		if p, ok := s.Archived[id]; ok {
			list = append(list, p)
		}
	}
	return list, nil
}
