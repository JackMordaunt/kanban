package mem

import (
	"fmt"

	"git.sr.ht/~jackmordaunt/kanban/storage"

	"git.sr.ht/~jackmordaunt/kanban"
)

var _ storage.Storer = (*Storer)(nil)

// Storer implements in-memory storage for Projects.
type Storer struct {
	Data  map[string]kanban.Project
	Order []string
	Err   error
}

func New() *Storer {
	return &Storer{
		Data: make(map[string]kanban.Project),
	}
}

func (s *Storer) Create(p kanban.Project) error {
	if _, ok := s.Data[p.Name]; ok {
		return fmt.Errorf("project %q exists", p.Name)
	}
	s.Data[p.Name] = p
	s.Order = append(s.Order, p.Name)
	return nil
}

func (s *Storer) Save(p kanban.Project) error {
	if _, ok := s.Data[p.Name]; ok {
		s.Data[p.Name] = p
	} else {
		return fmt.Errorf("project %q does not exist", p.Name)
	}
	return nil
}

func (s *Storer) Load(name string) (kanban.Project, bool, error) {
	if p, ok := s.Data[name]; ok {
		return p, ok, nil
	}
	return kanban.Project{}, false, nil
}

func (s *Storer) List() (list []kanban.Project, err error) {
	for _, name := range s.Order {
		if p, ok := s.Data[name]; ok {
			list = append(list, p)
		}
	}
	return list, nil
}
