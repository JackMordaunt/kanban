package bolt

import (
	"encoding/json"
	"fmt"

	"git.sr.ht/~jackmordaunt/kanban"
	"git.sr.ht/~jackmordaunt/kanban/storage"
	"github.com/boltdb/bolt"
	"github.com/google/uuid"
)

type Bucket []byte

func (b Bucket) String() string {
	return string(b)
}

var (
	BucketProject Bucket = Bucket("Project")
)

var _ storage.Storer = (*Storer)(nil)

type Storer struct {
	*bolt.DB
}

func Open(path string) (*Storer, error) {
	db, err := bolt.Open(path, 0660, nil)
	if err != nil {
		return nil, fmt.Errorf("opening database file: %w", err)
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(BucketProject); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("initializing buckets: %w", err)
	}
	return &Storer{DB: db}, nil
}

func (db *Storer) Create(p kanban.Project) error {
	id, err := p.ID.MarshalBinary()
	if err != nil {
		return fmt.Errorf("serializing project ID: %w", err)
	}
	v, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("serializing project: %w", err)
	}
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(BucketProject)
		if b == nil {
			return fmt.Errorf("bucket not initialized: %s", BucketProject)
		}
		if b.Get(id) != nil {
			return fmt.Errorf("project already exists for ID %q", p.ID)
		}
		return b.Put(id, v)
	})
}

func (db *Storer) Save(projects ...kanban.Project) error {
	return db.Update(func(tx *bolt.Tx) error {
		for _, p := range projects {
			if p.ID == uuid.Nil {
				continue
			}
			id, err := p.ID.MarshalBinary()
			if err != nil {
				return fmt.Errorf("serializing project ID: %w", err)
			}
			v, err := json.Marshal(p)
			if err != nil {
				return fmt.Errorf("serializing project: %w", err)
			}
			if err := tx.Bucket(BucketProject).Put(id, v); err != nil {
				return fmt.Errorf("updating project: %w", err)
			}
		}
		return nil
	})
}

func (db *Storer) Lookup(name string) (p kanban.Project, ok bool, err error) {
	// @enhance: index by name
	return p, ok, db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(BucketProject).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if err := json.Unmarshal(v, &p); err != nil {
				return fmt.Errorf("deserializing project: %w", err)
			}
			if p.Name == name {
				ok = true
				break
			}
		}
		return nil
	})
}

func (db *Storer) Find(id uuid.UUID) (p kanban.Project, ok bool, err error) {
	key, err := id.MarshalBinary()
	if err != nil {
		return p, false, fmt.Errorf("serializing id: %w", err)
	}
	return p, ok, db.View(func(tx *bolt.Tx) error {
		if err := json.Unmarshal(tx.Bucket(BucketProject).Get(key), &p); err != nil {
			return fmt.Errorf("deserializing project: %w", err)
		}
		ok = true
		return nil
	})
}

func (db *Storer) List() (list []kanban.Project, err error) {
	return list, db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(BucketProject).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var p kanban.Project
			if err := json.Unmarshal(v, &p); err != nil {
				return fmt.Errorf("deserializing project: %w", err)
			}
			list = append(list, p)
		}
		return nil
	})
}

func (db *Storer) Load(projects []kanban.Project) error {
	if len(projects) > 0 {
		if projects[0].ID == uuid.Nil {
			list, err := db.List()
			if err != nil {
				return err
			}
			copy(projects, list)
			return nil
		}
	}
	return db.View(func(tx *bolt.Tx) error {
		for ii, p := range projects {
			id, err := p.ID.MarshalBinary()
			if err != nil {
				return fmt.Errorf("serializing project ID: %w", err)
			}
			if err := json.Unmarshal(tx.Bucket(BucketProject).Get(id), &projects[ii]); err != nil {
				return fmt.Errorf("deserializing project: %w", err)
			}
		}
		return nil
	})
}

func (db *Storer) Count() (count int, err error) {
	return count, db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(BucketProject).Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			count++
		}
		return nil
	})
}
