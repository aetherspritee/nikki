package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

// TODOS
// TODO: refactor code to be maintanable
// TODO: automatic resizing of ui
// TODO: build and refactor rules
// TODO: better data storage solution
// TODO: add mouse support show info for single day (idek if thats possible)

//////////////////////////////////////////
// Welcome to 日記, a TUI habit tracker! //
///////////////////////////// ////////////

func main() {

	// encodeJson()
	// decodeJson()
	readConfig()

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	// file := loadJSON()
	// file = addEntry(file)
	// zeGrid := createGrid(file, "year", time.Now(), 0)
	// renderGrid(zeGrid)
	// res := storeJSON(file)
	// if res != 0 {
	// }
}
