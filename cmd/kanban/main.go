package main

import (
	"log"
	"os"
	"path/filepath"

	"git.sr.ht/~jackmordaunt/kanban/storage/storm"

	"github.com/spf13/pflag"

	"gioui.org/font/gofont"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/storage"
	"git.sr.ht/~jackmordaunt/kanban/storage/mem"

	"gioui.org/app"
)

var (
	MemStorage bool
)

func init() {
	pflag.BoolVar(&MemStorage, "mem-storage", false, "store entities in memory")
	pflag.Parse()
}

func main() {
	storage, err := func() (storage.Storer, error) {
		if MemStorage {
			return mem.New(), nil
		} else {
			return storm.Open(filepath.Join(os.TempDir(), "kanban.db"))
		}
	}()
	if err != nil {
		log.Fatalf("storage driver: %v\n", err)
	}
	if closer, ok := storage.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	go func() {
		ui := UI{
			Window:  app.NewWindow(app.Title("Kanban")),
			Th:      material.NewTheme(gofont.Collection()),
			Storage: storage,
		}
		if err := ui.Loop(); err != nil {
			log.Fatalf("error: %v", err)
		}
		os.Exit(0)
	}()
	app.Main()
}
