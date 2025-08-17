package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/charmbracelet/huh"
)

var repoPath string  // local repo path
var force bool       // force pull and push operations
var interactive bool // interactive mode

const (
	DatabasePath   = ".wsync/database.json"
	TokenPath      = ".wsync/token"
	WacceptedMajor = 3
	WminMinor      = 12
)

// ___________________________ INTERFACE ___________________________

func menu() {
	interactive = true

	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What to do ?").
				Options(
					huh.NewOption("Status", "status"),
					huh.NewOption("Sync", "sync"),
					huh.NewOption("Push", "push"),
					huh.NewOption("Pull", "pull"),
					huh.NewOption("List", "list"),
					huh.NewOption("Init", "init"),
					huh.NewOption("nothing", "nothing"),
				).
				Value(&action),
		),
	)
	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	switch action {
	case "init":
		Init(nil)
	case "status":
		Status()
	case "list":
		List()
	case "sync":
		Sync(nil)
	case "push":
		Push(nil)
	case "pull":
		Pull(nil)
	default:
		fmt.Println("bye bye 👋")
	}
}

func main() {
	log.SetFlags(0)

	flag.StringVar(&repoPath, "C", ".", "set the working directory")
	flag.BoolVar(&force, "F", false, "force push or pull")
	flag.BoolVar(&interactive, "i", false, "enable interactive mode")
	flag.Parse()

	args := flag.Args()
	if len(args) >= 1 {
		switch args[0] {
		case "init":
			Init(args[1:])
		case "sync":
			Sync(args[1:])
		case "pull":
			Pull(args[1:])
		case "push":
			Push(args[1:])
		case "remove":
			Remove(args[1:])
		case "add":
			Add(args[1:])
		case "list":
			List()
		case "status":
			Status()
		case "version":
			Version()
		default:
			log.Fatalln("invalid sub command")
		}
	} else {
		menu()
	}

}
