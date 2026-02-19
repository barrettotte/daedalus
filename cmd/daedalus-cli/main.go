package main

import (
	"fmt"
	"os"
	"strings"
)

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: daedalus-cli --board <path> <command> [args...]

Commands:
  board                  Show board summary (title, list/card counts)
  lists                  List all lists with card counts
  cards <list>           List cards in a list
  card-get <id>          Show full card details
  card-create <list> <title>  Create a new card
  card-delete <id>       Delete a card by ID
  list-create <name>     Create a new list
  list-delete <name>     Delete a list and its cards
  export-json <path>     Export board to JSON file
  export-zip <path>      Export board to ZIP archive`)
}

func main() {
	args := os.Args[1:]

	// Parse --board flag
	boardPath := ""
	var rest []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--board" {
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --board requires a path argument")
				os.Exit(1)
			}
			boardPath = args[i+1]
			i++ // skip value
		} else if strings.HasPrefix(args[i], "--board=") {
			boardPath = strings.TrimPrefix(args[i], "--board=")
		} else {
			rest = append(rest, args[i])
		}
	}

	if boardPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --board flag is required")
		usage()
		os.Exit(1)
	}

	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no command specified")
		usage()
		os.Exit(1)
	}

	command := rest[0]
	cmdArgs := rest[1:]

	var err error
	switch command {
	case "board":
		err = cmdBoard(boardPath)
	case "lists":
		err = cmdLists(boardPath)
	case "list-create":
		err = cmdListCreate(boardPath, cmdArgs)
	case "list-delete":
		err = cmdListDelete(boardPath, cmdArgs)
	case "cards":
		err = cmdCards(boardPath, cmdArgs)
	case "card-create":
		err = cmdCardCreate(boardPath, cmdArgs)
	case "card-delete":
		err = cmdCardDelete(boardPath, cmdArgs)
	case "card-get":
		err = cmdCardGet(boardPath, cmdArgs)
	case "export-json":
		err = cmdExportJSON(boardPath, cmdArgs)
	case "export-zip":
		err = cmdExportZip(boardPath, cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n", command)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
