package ui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/indent"
	"golang.org/x/term"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
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

// render grid as year view
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
