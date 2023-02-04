package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"regexp"
	// "sort"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/indent"
	toml "github.com/naoina/toml"
	"strconv"
	"strings"
	"time"
	// "github.com/peterrk/slices"
	"golang.org/x/term"
)

type Position float64

func (p Position) value() float64 {
	return math.Min(1, math.Max(0, float64(p)))
}

// Position aliases.
const (
	Top    Position = 0.0
	Bottom Position = 1.0
	Center Position = 0.5
	Left   Position = 0.0
	Right  Position = 1.0
)

func getLines(s string) (lines []string, widest int) {
	lines = strings.Split(s, "\n")

	for _, l := range lines {
		w := ansi.PrintableRuneWidth(l)
		if widest < w {
			widest = w
		}
	}

	return lines, widest
}

const (
	// In real life situations we'd adjust the document to fit the width we've
	// detected. In the case of this example we're hardcoding the width, and
	// later using the detected width only to truncate in order to avoid jaggy
	// wrapping.
	width = 96

	columnWidth = 30
)

// Style definitions.
var (

	// General.

	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(subtle).
		String()

	url = lipgloss.NewStyle().Foreground(special).Render

	// Tabs.

	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	// Title.

	titleStyle = lipgloss.NewStyle().
			MarginLeft(1).
			MarginRight(5).
			Padding(0, 1).
			Italic(true).
			Foreground(lipgloss.Color("#FFF7DB")).
			SetString("Lip Gloss")

	descStyle = lipgloss.NewStyle().MarginTop(1)

	infoStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle)

	// Dialog.

	// List.

	list = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(subtle).
		MarginRight(2).
		Height(8).
		Width(columnWidth + 1)

	listHeader = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(subtle).
			MarginRight(2).
			Render

	listItem = lipgloss.NewStyle().PaddingLeft(2).Render

	checkMark = lipgloss.NewStyle().SetString("✓").
			Foreground(special).
			PaddingRight(1).
			String()

	listDone = func(s string) string {
		return checkMark + lipgloss.NewStyle().
			Strikethrough(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#969B86", Dark: "#696969"}).
			Render(s)
	}

	// Paragraphs/History.

	historyStyle = lipgloss.NewStyle().
			Align(lipgloss.Left).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(highlight).
			Margin(1, 3, 0, 0).
			Padding(1, 2).
			Height(19).
			Width(columnWidth)

	// Status Bar.

	statusNugget = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})

	statusStyle = lipgloss.NewStyle().
			Inherit(statusBarStyle).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1).
			MarginRight(1)

	encodingStyle = statusNugget.Copy().
			Background(lipgloss.Color("#A550DF")).
			Align(lipgloss.Right)

	statusText = lipgloss.NewStyle().Inherit(statusBarStyle)

	fishCakeStyle = statusNugget.Copy().Background(lipgloss.Color("#6124DF"))

	// Page.

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)
)

type model struct {
	choices       []string
	metrics       []string // metrics to show
	cursor1       int      // which metric is currently shown
	cursor2       int
	chosen        bool
	data          EntryData // data lül
	quitting      bool
	inputs        []textinput.Model
	cursorMode    textinput.CursorMode
	focusIndex    int
	wrongInput    bool
	wrongIndex    int
	generalConfig General
}

func initialModel() model {
	cfg := readConfig()
	//fmt.Printf("metrics: %v\n", cfg.Metrics)
	metrics := []string{}
	for _, element := range cfg.Metrics {
		metrics = append(metrics, element.Name)
	}
	// TODO: first check whether metrics have changed
	// Handle: only removed metrics, only added metrics,
	// added and removed metrics

	data := loadJSON()
	//update the decoded data based on changes in config!
	updatedMetrics := make(map[int][]string, len(cfg.Metrics))
	for index, element := range cfg.Metrics {
		updatedMetrics[index] = []string{element.Name, element.Rule, element.Color1, element.Color2}
	}
	// rearrange data to fit metrics and add new
	// element for new metrics
	var newData []MetricData
	newData2 := make([]MetricData, len(metrics))
	// FIXME: if onlyAdded
	if len(metrics)-len(data.Data) > 0 {
		// if metrics were added
		newDataR := data.Data
		for dontCare := 0; dontCare < (len(metrics) - len(data.Data)); dontCare++ {
			newDataR = append(newDataR, MetricData{})
		}
		newData = newDataR
	} else {
		// FIXME: if metrics were removed
		// find names of old metrics
		oldMetrics := make([]string, len(data.Metrics))
		for ind, _ := range data.Metrics {
			oldMetrics[ind] = data.Metrics[ind][0]
		}
		missing := []int{}
		// check where one misses
		for idx, element := range oldMetrics {
			miss := false
			for _, element2 := range metrics {
				if element == element2 {
					miss = true
				}
			}
			if miss == false {
				missing = append(missing, idx)
			}
		}
		// remove corresponding entry
		// TODO: Handle deletion+addition of metrics
		newDataR := make([]MetricData, len(metrics))
		for idx, element := range data.Data {
			isMissing := false
			for _, element2 := range missing {
				if idx == element2 {
					isMissing = true
				}
			}
			if isMissing == false {
				newDataR = append(newDataR, element)
			}
		}
		newData = newDataR
	}
	// TODO: if metrics were added and removed
	// how to do this tho
	data.Metrics = updatedMetrics
	for idx, ele := range newData {
		foundCorrectIdx := false
		// check if metric name in data is in same order as metrics in Metrics field
		// such that change of order doesnt break everything
		for sIdx := 0; sIdx < len(metrics); sIdx++ {
			if newData[idx].Name == metrics[sIdx] {
				newData2[sIdx] = ele
				newData2[sIdx].Name = metrics[sIdx]
				foundCorrectIdx = true
			}
		}
		if foundCorrectIdx == false {
			newData2[idx] = MetricData{
				Name:   metrics[idx],
				Date:   []time.Time{},
				Value:  []string{},
				Color1: data.Metrics[idx][2],
				Color2: data.Metrics[idx][3],
			}
		}

	}
	data.Data = newData2
	// fmt.Printf("The reformatted data: %v\n", newData2)

	m := model{
		metrics:       metrics,
		data:          data,
		choices:       []string{"view calendar", "add entry"},
		chosen:        false,
		quitting:      false,
		inputs:        make([]textinput.Model, len(metrics)),
		wrongInput:    false,
		generalConfig: cfg.General,
	}

	var (
		focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(m.generalConfig.ActiveButtonColor))
		cursorStyle  = focusedStyle.Copy()
	)
	var t textinput.Model
	for i := range m.metrics {
		t = textinput.New()
		t.CursorStyle = cursorStyle
		switch i {
		case 0:
			t.Placeholder = m.metrics[i]
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		default:
			t.Placeholder = m.metrics[i]

		}
		m.inputs[i] = t
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure these keys always quit
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		if k == "q" || k == "esc" || k == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Hand off the message and model to the appropriate update function for the
	// appropriate view based on the current state.
	if !m.chosen {
		return updateMenu(m, msg)
	}
	return updateChosen(m, msg)
}

func (m model) View() string {
	var s string
	if m.quitting {
		return "\n  See you later!\n\n"
	}
	if !m.chosen {
		s = menuView(m)
	} else {
		s = chosenView(m)
	}
	return indent.String("\n"+s+"\n\n", 2)
}

func menuView(m model) string {
	var (
		buttonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color(m.generalConfig.ButtonColor)).
				Padding(0, 3).
				MarginTop(1)
		activeButtonStyle = buttonStyle.Copy().
					Foreground(lipgloss.Color("#FFF7DB")).
					Background(lipgloss.Color(m.generalConfig.ActiveButtonColor)).
					MarginRight(2).
					Underline(true)
		dialogBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(m.generalConfig.BorderColor)).
				Padding(1, 0).
				BorderTop(true).
				BorderLeft(true).
				BorderRight(true).
				BorderBottom(true)
	)
	// Initial menu view
	var s string
	// construct just like menu view
	candElements := []string{}
	for idx, element := range m.choices {
		if idx == m.cursor1 {
			candElements = append(candElements, activeButtonStyle.Render(element))
		} else {
			candElements = append(candElements, buttonStyle.Render(element))
		}
	}
	options := MyJoinVertical(Top, candElements)
	dialog := lipgloss.Place(width, 9,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(options),
		lipgloss.WithWhitespaceForeground(subtle),
	)
	s += dialog
	// The footer
	s += "\n\nPress q to quit.\n"

	return s
}

func chosenView(m model) string {
	var s string
	switch m.cursor1 {
	case 0:
		s = calendarView(m)
	case 1:
		s = newEntryView(m)
	}
	return s
}

func calendarView(m model) string {

	var (
		tab = lipgloss.NewStyle().
			Border(tabBorder, true).
			BorderForeground(lipgloss.Color(m.generalConfig.BorderColor)).
			Padding(0, 1)

		activeTab = tab.Copy().Border(activeTabBorder, true)

		tabGap = tab.Copy().
			BorderTop(false).
			BorderLeft(false).
			BorderRight(false)

		dialogBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(m.generalConfig.BorderColor)).
				Padding(1, 0).
				BorderTop(true).
				BorderLeft(true).
				BorderRight(true).
				BorderBottom(true)
	)
	var s string
	metricCands := []string{}
	for i, choice := range m.metrics {
		if i == m.cursor2 {
			metricCands = append(metricCands, activeTab.Render(choice))
		} else {
			metricCands = append(metricCands, tab.Render(choice))
		}
	}
	row := MyJoinHorizontal(Top, metricCands)
	gap := tabGap.Render(strings.Repeat(" ", max(0, width-lipgloss.Width(row)-2)))
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
	s += row
	s += "\n\n"
	if len(m.data.Data[m.cursor2].Value) >= 1 {
		zeGrid := createGrid(m.data, "year", time.Now(), m.cursor2)
		s += prerenderGrid(zeGrid)

		// min, avg and max value
		mmin, mmax, mavg := getMinMaxAvg(m.data, m.cursor2)
		mmin = "Minumum:  " + mmin + " || "
		mavg = "Average:  " + mavg + " || "
		mmax = "Maximum:  " + mmax
		minMaxAvgString := lipgloss.JoinHorizontal(lipgloss.Center, mmin, mavg, mmax)
		question := lipgloss.NewStyle().Width(70).Align(lipgloss.Center).Render(minMaxAvgString)
		currStreak, LongestStreak := streakChecker(m.data, m.cursor2)
		cStreak := "Current Streak:  " + strconv.Itoa(currStreak) + " || "
		lStreak := "Longest Streak:  " + strconv.Itoa(LongestStreak)
		streakString := lipgloss.JoinHorizontal(lipgloss.Center, cStreak, lStreak)
		streak_render := lipgloss.NewStyle().Width(70).Align(lipgloss.Center).Render(streakString)
		ui := lipgloss.JoinVertical(lipgloss.Center, question, streak_render)
		dialog := lipgloss.Place(width, 9,
			lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Render(ui),
			lipgloss.WithWhitespaceForeground(subtle),
		)
		s += dialog

		// streakUI := lipgloss.JoinVertical(lipgloss.Center, streak_render)
		// streakDialog := lipgloss.Place(width, 9,
		// 	lipgloss.Center, lipgloss.Center,
		// 	dialogBoxStyle.Render(streakUI),
		// 	lipgloss.WithWhitespaceForeground(subtle),
		// )
		// s += streakDialog

	} else {
		// render a text box saying "no entries yet"
		// add it to the s string
		// add it to the s string
		disclaimer := lipgloss.NewStyle().Width(70).Align(lipgloss.Center).Render("No entries yet!")
		ui := lipgloss.JoinVertical(lipgloss.Center, disclaimer)
		dialog := lipgloss.Place(width, 9,
			lipgloss.Center, lipgloss.Center,
			dialogBoxStyle.Render(ui),
			lipgloss.WithWhitespaceForeground(subtle),
		)
		s += dialog
	}

	// The footer
	s += "\n\nPress b to return to the menu.\nPress q to quit."
	return s
}

func newEntryView(m model) string {
	var (
		focusedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(m.generalConfig.ActiveButtonColor))
		blurredStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(m.generalConfig.ActiveButtonColor))
		focusedButton = focusedStyle.Copy().Render("[ Submit ]")
		blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
		helpStyle     = blurredStyle.Copy()
	)
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}
	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	if m.wrongInput == true {
		b.WriteString(helpStyle.Render("Wrong input for field "))
		b.WriteString(helpStyle.Render(m.metrics[m.wrongIndex]))
	}
	return b.String()
}

func updateMenu(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	// update the menu
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {
		case "j", "down":
			if m.cursor1 == len(m.choices)-1 {
				m.cursor1 = 0
			} else {
				m.cursor1++
			}
		case "k", "up":
			if m.cursor1 == 0 {
				m.cursor1 = len(m.choices) - 1
			} else {
				m.cursor1--
			}
		case "enter", "l":
			// open chosenView
			m.chosen = true
		}
	}
	return m, nil
}

func updateChosen(m model, msg tea.Msg) (tea.Model, tea.Cmd) {

	switch m.cursor1 {
	case 0:
		m, cmd := updateCalendar(m, msg)
		return m, cmd
	case 1:
		m, cmd := updateEntry(m, msg)
		return m, cmd
	}
	return m, nil
}

func updateCalendar(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	// update chosen view

	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "right" and "l" keys move the cursor to the right
		case "right", "l":
			if m.cursor2 < len(m.metrics)-1 {
				m.cursor2++
			} else {
				m.cursor2 = 0
			}

		// The "left" and "h" keys move the cursor to the left
		case "left", "h":
			if m.cursor2 > 0 {
				m.cursor2--
			} else {
				m.cursor2 = len(m.metrics) - 1
			}
		case "b":
			m.chosen = false
		case "1":
			m.cursor2 = 0
		case "2":
			if len(m.metrics) > 1 {
				m.cursor2 = 1
			}
		case "3":
			if len(m.metrics) > 2 {
				m.cursor2 = 2
			}
		case "4":
			if len(m.metrics) > 3 {
				m.cursor2 = 3
			}
		case "5":
			if len(m.metrics) > 4 {
				m.cursor2 = 4
			}
		case "6":
			if len(m.metrics) > 5 {
				m.cursor2 = 5
			}
		case "7":
			if len(m.metrics) > 6 {
				m.cursor2 = 6
			}
		case "8":
			if len(m.metrics) > 7 {
				m.cursor2 = 7
			}
		case "9":
			if len(m.metrics) > 8 {
				m.cursor2 = 8
			}
		}
	}
	return m, nil
}

func updateEntry(m model, msg tea.Msg) (tea.Model, tea.Cmd) {

	var (
		focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(m.generalConfig.ActiveButtonColor))
		noStyle      = lipgloss.NewStyle()
	)
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit
		case "b":
			m.chosen = false
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, exit.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				// Validate inputs
				inputValid := true
				for i, e := range m.inputs {
					rule := m.data.Metrics[i][1]
					if ruleChecker(e.Value(), rule) == false {
						inputValid = false
						m.wrongIndex = i
					}
				}
				if inputValid == true {
					for idx, ele := range m.inputs {
						// Check if entry for that day exists, replace it or just store
						if checkIfEntryExists(m.data, idx, time.Now()) == true {
							m.data.Data[idx].Date[len(m.data.Data[idx].Date)-1] = time.Now()
							m.data.Data[idx].Value[len(m.data.Data[idx].Value)-1] = ele.Value()

						} else {
							m.data.Data[idx].Date = append(m.data.Data[idx].Date, time.Now())
							m.data.Data[idx].Value = append(m.data.Data[idx].Value, ele.Value())
						}
					}
					m.wrongInput = false
					res := storeJSON(m.data)
					if res != 0 {
						panic("wwww")
					}
					// TODO: reset the complete view component

					// return to menu
					m.chosen = false
				} else {
					m.wrongInput = true
				}

				return m, nil
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}
	cmd := m.updateInputs(msg)

	return m, cmd
}

func checkIfEntryExists(data EntryData, metric int, t time.Time) bool {
	if len(data.Data[metric].Date) < 1 {
		return false
	}
	lastEntry := data.Data[metric].Date[len(data.Data[metric].Date)-1]
	lastDate := lastEntry.Format("02.01.2006")
	if t.Format("02.01.2006") == lastDate {
		return true
	} else {
		return false
	}
}

func MyJoinHorizontal(pos Position, strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	var (
		// Groups of strings broken into multiple lines
		blocks = make([][]string, len(strs))

		// Max line widths for the above text blocks
		maxWidths = make([]int, len(strs))

		// Height of the tallest block
		maxHeight int
	)

	// Break text blocks into lines and get max widths for each text block
	for i, str := range strs {
		blocks[i], maxWidths[i] = getLines(str)
		if len(blocks[i]) > maxHeight {
			maxHeight = len(blocks[i])
		}
	}

	// Add extra lines to make each side the same height
	for i := range blocks {
		if len(blocks[i]) >= maxHeight {
			continue
		}

		extraLines := make([]string, maxHeight-len(blocks[i]))

		switch pos {
		case Top:
			blocks[i] = append(blocks[i], extraLines...)

		case Bottom:
			blocks[i] = append(extraLines, blocks[i]...)

		default: // Somewhere in the middle
			n := len(extraLines)
			split := int(math.Round(float64(n) * pos.value()))
			top := n - split
			bottom := n - top

			blocks[i] = append(extraLines[top:], blocks[i]...)
			blocks[i] = append(blocks[i], extraLines[bottom:]...)
		}
	}

	// Merge lines
	var b strings.Builder
	for i := range blocks[0] { // remember, all blocks have the same number of members now
		for j, block := range blocks {
			b.WriteString(block[i])

			// Also make lines the same length
			b.WriteString(strings.Repeat(" ", maxWidths[j]-ansi.PrintableRuneWidth(block[i])))
		}
		if i < len(blocks[0])-1 {
			b.WriteRune('\n')
		}
	}

	return b.String()
}

func MyJoinVertical(pos Position, strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	var (
		blocks   = make([][]string, len(strs))
		maxWidth int
	)

	for i := range strs {
		var w int
		blocks[i], w = getLines(strs[i])
		if w > maxWidth {
			maxWidth = w
		}
	}

	var b strings.Builder
	for i, block := range blocks {
		for j, line := range block {
			w := maxWidth - ansi.PrintableRuneWidth(line)

			switch pos {
			case Left:
				b.WriteString(line)
				b.WriteString(strings.Repeat(" ", w))

			case Right:
				b.WriteString(strings.Repeat(" ", w))
				b.WriteString(line)

			default: // Somewhere in the middle
				if w < 1 {
					b.WriteString(line)
					break
				}

				split := int(math.Round(float64(w) * pos.value()))
				right := w - split
				left := w - right

				b.WriteString(strings.Repeat(" ", left))
				b.WriteString(line)
				b.WriteString(strings.Repeat(" ", right))
			}

			// Write a newline as long as we're not on the last line of the
			// last block.
			if !(i == len(blocks)-1 && j == len(block)-1) {
				b.WriteRune('\n')
			}
		}
	}

	return b.String()
}

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

type MetricData struct {
	Name   string
	Date   []time.Time
	Value  []string
	Color1 string
	Color2 string
}

type EntryData struct {
	Metrics map[int][]string
	Data    []MetricData
}

type TestData struct {
	Date  int
	Value string
}

func decodeJson() {

	var result EntryData
	jsonFile, err := ioutil.ReadFile("test.json")
	if err != nil {
		panic(nil)
	}

	json.Unmarshal(jsonFile, &result)

}

func encodeJson() {

	sleepData := MetricData{
		Name:  "Got up",
		Date:  []time.Time{time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		Value: []string{"06:00", "07:30", "08:20", "12:30", "06:00", "06:00", "06:00", "06:00", "08:45", "11:20"},
	}

	moodData := MetricData{
		Name:  "Mood",
		Date:  []time.Time{time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		Value: []string{"10", "6", "6", "6", "6", "8", "8", "7", "8", "8"},
	}

	bundledData := []MetricData{
		sleepData, moodData,
	}

	data := EntryData{
		Metrics: map[int][]string{0: []string{"Sleep", "time"}, 1: []string{"Mood", "int10"}},
		Data:    bundledData,
	}

	var jsonData []byte
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	}
	_ = ioutil.WriteFile("test.json", jsonData, 0644)
}

func colorGrid(xSteps, ySteps int) [][]string {
	x0y0, _ := colorful.Hex("#F25D94")
	x1y0, _ := colorful.Hex("#EDFF82")
	x0y1, _ := colorful.Hex("#643AFF")
	x1y1, _ := colorful.Hex("#14F9D5")

	x0 := make([]colorful.Color, ySteps)
	for i := range x0 {
		x0[i] = x0y0.BlendLuv(x0y1, float64(i)/float64(ySteps))
	}

	x1 := make([]colorful.Color, ySteps)
	for i := range x1 {
		x1[i] = x1y0.BlendLuv(x1y1, float64(i)/float64(ySteps))
	}

	grid := make([][]string, ySteps)
	for x := 0; x < ySteps; x++ {
		y0 := x0[x]
		grid[x] = make([]string, xSteps)
		for y := 0; y < xSteps; y++ {
			grid[x][y] = y0.BlendLuv(x1[x], float64(y)/float64(xSteps)).Hex()
		}
	}

	return grid
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ############################
// THIS IS WHERE THE FUN BEGINS
// ############################

// add func to add and store entries!
// add config via toml (need to find some parser for that)
// add mouse support show info for single day

var rules map[string]string = map[string]string{
	"int10": `^[0-9]$`,
	"int":   `^[0-9]+$`,
	"time":  `^([0-1][0-9]|2[0-3]):[0-5][0-9]$`,
}

// render grid as year view
func renderGrid(colorGrid [][]string) {
	var (
		tab = lipgloss.NewStyle().
			Border(tabBorder, true).
			BorderForeground(highlight).
			Padding(0, 1)

		activeTab = tab.Copy().Border(activeTabBorder, true)

		tabGap = tab.Copy().
			BorderTop(false).
			BorderLeft(false).
			BorderRight(false)
	)
	doc := strings.Builder{}
	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	{
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			tab.Render("Went to bed"),
			activeTab.Render("Got up"),
			tab.Render("Studied nihongo"),
			tab.Render("Worked out"),
			tab.Render("Mood"),
		)
		gap := tabGap.Render(strings.Repeat(" ", max(0, width-lipgloss.Width(row)-2)))
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
		doc.WriteString(row + "\n\n")
	}
	b := strings.Builder{}
	for _, x := range colorGrid {
		for _, y := range x {
			// s := lipgloss.NewStyle().SetString(" ").Background(lipgloss.Color(y))
			s := lipgloss.NewStyle().SetString("▄").Foreground(lipgloss.Color(y))
			b.WriteString(s.String())
			w := lipgloss.NewStyle().SetString(" ")
			b.WriteString(w.String())
		}
		// w2 := lipgloss.NewStyle().SetString("  ")
		// b.WriteRune('\n')
		b.WriteRune('\n')
		// b.WriteString(w2.String())
	}
	colors := b.String()

	doc.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, colors))
	if physicalWidth > 0 {
		docStyle = docStyle.MaxWidth(physicalWidth)
	}

	// Okay, let's print it
	fmt.Println(docStyle.Render(doc.String()))
}

func prerenderGrid(colorGrid [][]string) string {
	doc := strings.Builder{}
	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	b := strings.Builder{}
	for _, x := range colorGrid {
		for _, y := range x {
			// s := lipgloss.NewStyle().SetString(" ").Background(lipgloss.Color(y))
			s := lipgloss.NewStyle().SetString("").Foreground(lipgloss.Color(y))
			b.WriteString(s.String())
			w := lipgloss.NewStyle().SetString(" ")
			b.WriteString(w.String())
		}
		// w2 := lipgloss.NewStyle().SetString("  ")
		// b.WriteRune('\n')
		b.WriteRune('\n')
		// b.WriteString(w2.String())
	}
	colors := b.String()

	doc.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, colors))
	if physicalWidth > 0 {
		docStyle = docStyle.MaxWidth(physicalWidth)
	}
	return doc.String()
}

func createGrid(data EntryData, format string, startDate time.Time, metric int) [][]string {

	var firstDay time.Weekday
	var numOfDays int
	var sizeX int
	sizeY := 7
	if format == "month" {
		// get year
		year := startDate.Format("02-01-2006")
		year = year[len(year)-4 : len(year)]
		// check for leap year
		leapCheck := "29.02." + year
		leapYear := false
		_, err := time.Parse("02.01.2006", leapCheck)
		if err != nil {
			leapYear = true
		}

		// get month
		month := startDate.Format("02-01-2006")
		month = month[len(month)-8 : len(month)-6]
		// check if 30 or 31 days based on given start date
		if leapYear {
			// try parsing day 29 and 31
			_, err := time.Parse("02.01.2006", "29."+month+"."+year)
			if err != nil {
				_, err := time.Parse("02.01.2006", "31"+month+"."+year)
				if err != nil {
					numOfDays = 30
				} else {
					numOfDays = 31
				}
			} else {
				numOfDays = 29
			}
		} else {
			// try parsing day 28 and 31
			_, err := time.Parse("02.01.2006", "28."+month+"."+year)
			if err != nil {
				_, err := time.Parse("02.01.2006", "31"+month+"."+year)
				if err != nil {
					numOfDays = 30
				} else {
					numOfDays = 31
				}
			} else {
				numOfDays = 28
			}
		}
		// build string for first of month
		firstDayString := "01." + month + "." + year
		_, err2 := time.Parse("02.01.2006", firstDayString)
		if err2 != nil {
			panic(err)
		}
		sizeX = 5
	} else if format == "year" {
		// get year
		year := startDate.Format("02-01-2006")
		year = year[len(year)-4 : len(year)]
		// date for jan 1
		firstDayString := "01.01." + year
		_, err := time.Parse("02.01.2006", firstDayString)
		if err != nil {
			panic(err)
		}

		// check for leap year
		leapCheck := "29.02." + year
		numOfDays = 366
		_, err2 := time.Parse("02.01.2006", leapCheck)
		if err2 != nil {
			numOfDays = 365
		}
		sizeX = 53
	}
	// got first weekday and length of year/month
	// create the grid
	grid := make([]int, sizeY*sizeX)
	weekdays := make([]int, numOfDays)
	counter := (int(firstDay) + 6) % 7
	for index, _ := range weekdays {
		weekdays[index] = counter + 1
		counter = (counter + 1) % 7
	}
	// go through weekdays and grid, add values in appropriate locations
	toAddInFront := weekdays[0] - 1
	toAddInBack := 7 - weekdays[len(weekdays)-1]
	front := make([]int, toAddInFront)
	back := make([]int, toAddInBack)
	grid = append(front, weekdays...)
	grid = append(grid, back...)

	formattedGrid := make([][]int, sizeY)
	for ind, element := range grid {
		formattedGrid[ind%7] = append(formattedGrid[ind%7], element)
	}
	rangeMap := calcRangeMap(data)
	// fmt.Printf("ZE RANGEMAP:: %v\n", rangeMap)
	colorMap := getColorMap(rangeMap, data, metric)
	// fmt.Printf("ZE COLORMAP:: %v\n", colorMap)
	completeGrid := mapDataToGrid(data, formattedGrid, startDate, metric, colorMap)

	return completeGrid
}

func mapDataToGrid(data EntryData, grid [][]int, toShow time.Time, metric int, colorMap map[string][]string) [][]string {
	// create map for date to number
	dateMap := map[int]string{}
	year := toShow.Format("02-01-2006")
	year = year[len(year)-4 : len(year)]
	year_as_int, err := strconv.Atoi(year)
	if err != nil {
		panic(err)
	}
	leapCheck := "29.02." + year
	yearLength := 366
	_, err2 := time.Parse("02.01.2006", leapCheck)
	if err2 != nil {
		yearLength = 365
	}
	start := time.Date(year_as_int, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= yearLength; i++ {
		dateMap[i] = start.Format("02.01.2006")
		start = start.AddDate(0, 0, 1)
	}

	// use date map to match
	dateMask := map[int]string{}
	for _, element := range data.Data[metric].Date {
		element := element.Format("02.01.2006")
		for index, element2 := range dateMap {
			if element == element2 {
				dateMask[index] = element
			} else if dateMask[index] == "" {
				dateMask[index] = "-"
			}
		}
	}
	// loop over grid in overcomplicated way to make sure you understand
	// how that shit works :)
	coloredGrid := make([][]string, 7)
	counter := 1
	counter2 := 0
	for i := 0; i < len(grid[0]); i++ {
		for j := 0; j < 7; j++ {
			if grid[j][i] != 0 {
				if dateMask[counter] != "-" {
					grid[j][i] = 2
					// coloredGrid[j] = append(coloredGrid[j], data.Data[metric].Value[counter2])
					coloredGrid[j] = append(coloredGrid[j], colorMap[data.Metrics[metric][0]][counter2])
					counter2 += 1
				} else {
					coloredGrid[j] = append(coloredGrid[j], "#D9DCCF")
				}
				counter += 1
			} else {
				coloredGrid[j] = append(coloredGrid[j], "#383838")
			}
		}
	}

	return coloredGrid

}

func reformatData(data string, MetricInfo string) int {
	// fmt.Printf("ZE DATA: %v, ZE RULE: %v\n", data, MetricInfo)
	if MetricInfo == "int10" {
		formattedData, err := strconv.Atoi(data)
		if err != nil {
			panic(err)
		}
		return formattedData
	} else if MetricInfo == "int" {
		formattedData, err := strconv.Atoi(data)

		if err != nil {
			panic(err)
		}
		return formattedData
	} else if MetricInfo == "time" {
		data = data[0:2] + data[3:5]
		formattedData, err := strconv.Atoi(data)

		if err != nil {
			panic(err)
		}

		return formattedData
	} else {
		panic("YOOOOOOOOOOOOOOOOO")
	}
}

func formatData(data int, MetricInfo string) string {
	if MetricInfo == "int10" {
		formattedData := strconv.Itoa(data)
		return formattedData
	} else if MetricInfo == "int" {
		formattedData := strconv.Itoa(data)
		return formattedData
	} else if MetricInfo == "time" {
		formattedData := strconv.Itoa(data)
		var formattedTime string
		if len(formattedData) < 4 {
			formattedTime = "0" + formattedData[0:1] + ":" + formattedData[1:3]
		} else {
			formattedTime = formattedData[0:2] + ":" + formattedData[2:4]
		}
		return formattedTime
	} else {
		panic("eroooooooooooo")
	}
}

func calcRangeMap(data EntryData) map[string][]float64 {
	rangeMap := map[string][]float64{}
	for index, _ := range data.Metrics {
		// access data slice
		rule := data.Metrics[index][1]
		metric := data.Metrics[index][0]
		currData := data.Data[index].Value
		// fmt.Printf("ZE CURR DATA: %v\n", currData)
		// reformat data in there
		formattedData := []int{}
		for _, element := range currData {
			formattedData = append(formattedData, reformatData(element, rule))
		}
		// normalize to range {0,1}
		var normalizedData []float64
		max := 1e-20
		min := 1e20
		for _, element := range formattedData {
			if float64(element) > max {
				max = float64(element)
			}
			if float64(element) < min {
				min = float64(element)
			}
		}
		max = max - min
		for idx, element := range formattedData {
			formattedData[idx] = int(float64(element) - min)
		}
		for _, element := range formattedData {
			normalizedData = append(normalizedData, float64(element)/max)
		}
		rangeMap[metric] = normalizedData
	}
	return rangeMap
}

func getMinMaxAvg(data EntryData, metric int) (string, string, string) {
	rule := data.Metrics[metric][1]
	currMax := 0
	currMin := 10000000000000
	sumHours := 0
	sumMins := 0
	for _, element := range data.Data[metric].Value {
		formattedElement := reformatData(element, rule)
		if formattedElement > currMax {
			currMax = formattedElement
		}
		if formattedElement < int(currMin) {
			currMin = formattedElement
		}
		currHours := formattedElement / 100
		sumHours += currHours
		sumMins += formattedElement - (currHours * 100)
	}
	avgHours := sumHours / len(data.Data[metric].Value)
	avgMins := sumMins / len(data.Data[metric].Value)
	avg := avgHours*100 + avgMins
	minString := formatData(currMin, rule)
	maxString := formatData(currMax, rule)
	avgString := formatData(avg, rule)

	return minString, maxString, avgString
}

func getColorMap(rangeMap map[string][]float64, data EntryData, metric int) map[string][]string {
	colorGrd := map[string][]string{}
	for index, _ := range rangeMap {
		x0y0, _ := colorful.Hex(data.Metrics[metric][2])
		x1y0, _ := colorful.Hex(data.Metrics[metric][3])

		x0 := make([]string, len(rangeMap[index]))
		for i := range x0 {
			x0[i] = x0y0.BlendLuv(x1y0, rangeMap[index][i]).Hex()
		}
		colorGrd[index] = x0
	}
	return colorGrd
}

// writes entry data into JSON
func addEntry(file EntryData) EntryData {
	// need temporary storage for new entry
	tempStorage := EntryData{
		Metrics: file.Metrics,
		Data:    []MetricData{},
	}

	for metric, _ := range file.Metrics {

		inputCheck := true
		for inputCheck == true {
			var input string
			fmt.Printf("Please input todays data for metric %v\n", file.Metrics[metric][0])
			// get input
			fmt.Scan(&input)
			// check if input is valid
			ok := ruleChecker(input, file.Metrics[metric][1])
			if ok {
				// create new MetricData var
				tempData := MetricData{
					Date:  []time.Time{time.Now()},
					Value: []string{input},
				}
				// add it to temp storage
				tempStorage.Data = append(tempStorage.Data, tempData)
				inputCheck = false
			} else {
				// no bueno
				fmt.Println("Wrong format")
			}
		}
	}
	// append temp storage to actual data
	for index, _ := range file.Data {
		file.Data[index].Date = append(file.Data[index].Date, tempStorage.Data[index].Date...)
		file.Data[index].Value = append(file.Data[index].Value, tempStorage.Data[index].Value...)
	}
	return file
}

func ruleChecker(input string, rule string) bool {
	// fmt.Printf("input: %v, rule: %v\n", input, rule)
	val, ok := rules[rule]
	if ok {
		// match regex
		if rule == "int10" {
			i, err := strconv.Atoi(input)
			if err != nil {
				// wtf are you doing
				return false
			}
			input = strconv.Itoa(i - 1)
		}
		var validInput = regexp.MustCompile(val)
		if validInput.MatchString(input) {
			// youre good to go
			return true
		} else {
			// aw hell naw
			return false
		}
	} else {
		return false
	}
}

// TODO: create a function a see the longest streak of continous inputs
// also render them as under the calendar view
func streakChecker(data EntryData, metric int) (int, int) {
	streak := 1
	longestStreak := 0
	// streakOK := true
	dates := data.Data[metric].Date
	for idx, element := range dates {
		formDate := element.Format("02.01.2006")
		year, _ := strconv.Atoi(formDate[len(formDate)-4 : len(formDate)])
		prevMonth, _ := (strconv.Atoi(formDate[len(formDate)-7 : len(formDate)-5]))
		month := time.Month(prevMonth)
		day, _ := strconv.Atoi(formDate[len(formDate)-10 : len(formDate)-8])
		// TODO: use the time.AddDate() method to check whether previous or following date exists in list
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		nextDate := date.AddDate(0, 0, 1).Format("02.01.2006")
		if len(dates) <= idx+1 {
			// last element in the list

		} else {
			// next date must exist
			nextDateInList := dates[idx+1].Format("02.01.2006")
			if nextDateInList == nextDate {
				// streak still ok
				streak += 1
			} else {
				// streak broken
				streak = 0
			}
		}
		if streak > longestStreak {
			longestStreak = streak
		}
	}
	return streak, longestStreak
}

func newEntry() {
	file := loadJSON()
	// check if there is an entry for today already
	file = addEntry(file)
	storeJSON(file)
}

// stores JSON
func storeJSON(data EntryData) int {
	var jsonData []byte
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return 1
	}
	_ = ioutil.WriteFile("data.json", jsonData, 0644)
	return 0
}

// loads JSON
func loadJSON() EntryData {
	var result EntryData
	jsonFile, err := ioutil.ReadFile("data.json")
	if err != nil {
		panic(nil)
	}

	json.Unmarshal(jsonFile, &result)
	return result
}

type General struct {
	BorderColor       string
	ActiveButtonColor string
	ButtonColor       string
}

type Config struct {
	General General
	Metrics []struct {
		Name   string
		Color1 string
		Color2 string
		Rule   string
	}
}

func readConfig() Config {
	var cfg Config
	config, err := os.Open("config.toml")
	if err != nil {
		panic(err)
	}
	defer config.Close()
	if err := toml.NewDecoder(config).Decode(&cfg); err != nil {
		panic(err)
	}

	return cfg
}
