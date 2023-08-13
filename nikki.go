package main

import (
	"fmt"
	"github.com/aetherspritee/nikki/src"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

// TODOS
// TODO: calendar colors are off for some metrics
// TODO: last open restructuring todo
// TODO: fully support monthly view
// TODO: build and refactor rules
// TODO: better data storage solution
// TODO: add mouse support show info for single day (idek if thats possible)

// ############################################
// ##  Welcome to 日記, a TUI habit tracker! ##
// ############################################

func main() {
	// encodeJson()
	// decodeJson()
	src.ReadConfig()

	p := tea.NewProgram(src.InitialModel())
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
