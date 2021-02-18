package storm

import (
	"errors"
	"fmt"
	"log"

	"git.sr.ht/~jackmordaunt/kanban"
	"git.sr.ht/~jackmordaunt/kanban/storage"

	"github.com/asdine/storm/v3"
)

var _ storage.Storer = (*Storer)(nil)

// Storer implements Project storage using storm db.
type Storer struct {
	DB *storm.DB
}

// Schema is a database representation of a kanban.Project.
type Schema struct {
	ID      string `storm:"id"`
	Project kanban.Project
}

// Open a database handle using the file specified by path.
func Open(path string) (*Storer, error) {
	db, err := storm.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening data file: %w", err)
	}
	if err := db.Init(&Schema{}); err != nil {
		return nil, fmt.Errorf("initialising schema: %v", err)
	}
	if err != nil {
		log.Fatalf("error: initializing data: %v", err)
	}
	return &Storer{DB: db}, nil

}

func (s *Storer) Create(p kanban.Project) error {
	return s.DB.Save(&Schema{ID: p.Name, Project: p})
}

func (s *Storer) Save(projects ...kanban.Project) error {
	for _, p := range projects {
		p := p
		if err := s.DB.Update(&Schema{ID: p.Name, Project: p}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Storer) Lookup(name string) (kanban.Project, bool, error) {
	var schema Schema
	if err := s.DB.One("ID", name, &schema); err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return schema.Project, false, nil
		} else {
			return schema.Project, false, err
		}
	}
	return schema.Project, true, nil
}

func (s *Storer) List() (list []kanban.Project, err error) {
	var (
		projects []Schema
	)
	if err := s.DB.All(&projects); err != nil {
		return nil, fmt.Errorf("loading projects: %v", err)
	}
	for _, p := range projects {
		p := p
		list = append(list, p.Project)
	}
	return list, nil
}

func (s *Storer) Load(projects []kanban.Project) error {
	for ii := range projects {
		var (
			name   = projects[ii].Name
			schema = Schema{
				ID:      name,
				Project: projects[ii],
			}
		)
		if err := s.DB.One("ID", name, &schema); err != nil {
			return err
		}
		projects[ii] = schema.Project
	}
	return nil
}
