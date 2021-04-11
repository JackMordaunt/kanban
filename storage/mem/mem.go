package mem

import (
	"fmt"

	"github.com/google/uuid"

	"git.sr.ht/~jackmordaunt/kanban"
)

// var _ storage.Storer = (*Storer)(nil)

// Storer implements in-memory storage for Projects.
type Storer struct {
	Active   Bucket
	Archived Bucket
}

type Bucket struct {
	Data  map[uuid.UUID]kanban.Project
	Order []uuid.UUID
}

func New() *Storer {
	return &Storer{
		Active: Bucket{
			Data: make(map[uuid.UUID]kanban.Project),
		},
		Archived: Bucket{
			Data: make(map[uuid.UUID]kanban.Project),
		},
	}
}

func (s *Storer) Create(p kanban.Project) error {
	if _, ok := s.Active.Data[p.ID]; ok {
		return fmt.Errorf("project %q exists", p.Name)
	}
	s.Active.Add(p)
	return nil
}

func (s *Storer) Save(projects ...kanban.Project) error {
	for _, p := range projects {
		if _, ok := s.Active.Data[p.ID]; ok {
			s.Active.Data[p.ID] = p.Clone()
		} else {
			return fmt.Errorf("project %q does not exist", p.Name)
		}
	}
	return nil
}

func (s *Storer) Find(id uuid.UUID) (p kanban.Project, ok bool, err error) {
	p, ok = s.Active.Data[id]
	return p.Clone(), ok, nil
}

func (s *Storer) Count() (int, error) {
	return len(s.Active.Data), nil
}

func (s *Storer) List() (list []kanban.Project, err error) {
	return s.Active.List(), nil
}

func (s *Storer) Load(projects []kanban.Project) error {
	for ii := range projects {
		projects[ii] = s.Active.Data[projects[ii].ID].Clone()
	}
	return nil
}

func (s *Storer) Archive(id uuid.UUID) error {
	p, ok := s.Active.Data[id]
	if !ok {
		return nil
	}
	s.Active.Delete(id)
	s.Archived.Add(p)
	return nil
}

func (s *Storer) Restore(id uuid.UUID) error {
	p, ok := s.Archived.Data[id]
	if !ok {
		return nil
	}
	s.Archived.Delete(id)
	s.Active.Add(p)
	return nil
}

func (s *Storer) ListArchived() (list []kanban.Project, err error) {
	return s.Archived.List(), nil
}

func (s *Storer) Clear() {
	s.Active = Bucket{
		Data: make(map[uuid.UUID]kanban.Project),
	}
	s.Archived = Bucket{
		Data: make(map[uuid.UUID]kanban.Project),
	}
}

func (b *Bucket) Add(p kanban.Project) {
	b.Data[p.ID] = p
	b.Order = append(b.Order, p.ID)
}

func (b *Bucket) Delete(id uuid.UUID) {
	delete(b.Data, id)
	for ii := range b.Order {
		if b.Order[ii] == id {
			b.Order = append(b.Order[:ii], b.Order[ii+1:]...)
			break
		}
	}
}

func (b *Bucket) List() (list []kanban.Project) {
	for _, id := range b.Order {
		list = append(list, b.Data[id])
	}
	return list
}
