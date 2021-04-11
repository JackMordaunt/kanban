package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/profile"
	"github.com/spf13/pflag"

	"gioui.org/font/gofont"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~jackmordaunt/kanban/storage"
	"git.sr.ht/~jackmordaunt/kanban/storage/bolt"

	"gioui.org/app"
)

var (
	MemStorage bool
	ProfileOpt string
)

func init() {
	pflag.BoolVar(&MemStorage, "mem-storage", false, "store entities in memory")
	pflag.StringVar(&ProfileOpt, "profile", "", fmt.Sprintf("record runtime performance statistics %s", profiles))
	pflag.Parse()
}

func main() {
	if stopper := Profile(ProfileOpt).Start(); stopper != nil {
		defer stopper.Stop()
	}
	storage, err := func() (storage.Storer, error) {
		data, err := app.DataDir()
		if err != nil {
			return nil, fmt.Errorf("data dir: %v", err)
		}
		db := filepath.Join(data, "kanban.db")
		fmt.Printf("%s\n", db)
		return bolt.Open(db)
	}()
	if err != nil {
		log.Fatalf("storage driver: %v\n", err)
	}
	if closer, ok := storage.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	go func() {
		ui := UI{
			Window:  app.NewWindow(app.Title("Kanban"), app.MinSize(unit.Dp(700), unit.Dp(250))),
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

// Profile starts a profiler based on the provided option.
type Profile string

var (
	CPU   Profile = "cpu"
	Mem   Profile = "mem"
	Block Profile = "block"
	Trace Profile = "trace"

	profiles = Profiles{
		CPU,
		Mem,
		Block,
		Trace,
	}
)

func (p Profile) Start() interface{ Stop() } {
	switch p {
	case "cpu":
		return profile.Start(profile.CPUProfile)
	case "mem":
		return profile.Start(profile.MemProfile)
	}
	return nil
}

type Profiles []Profile

func (p Profiles) String() string {
	var s strings.Builder
	s.WriteByte('[')
	for ii, opt := range p {
		s.WriteString(string(opt))
		if ii != len(p)-1 {
			s.WriteString(", ")
		}
	}
	s.WriteByte(']')
	return s.String()
}
