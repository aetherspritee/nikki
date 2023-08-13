package src

import (
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/reflow/indent"
	"golang.org/x/term"
	"os"
	"strconv"
	"strings"
	"time"
)

// ##################################
// ## MODEL SETUP & INITIALIZATION ##
// ##################################

type model struct {
	choices       []string
	metrics       []string // metrics to show
	cursor1       int      // which metric is currently shown
	cursor2       int
	chosen        bool
	data          EntryData // data lül
	quitting      bool
	inputs        []textinput.Model
	cursorMode    cursor.Mode
	focusIndex    int
	wrongInput    bool
	wrongIndex    int
	generalConfig General
}

func InitialModel() model {
	data, metrics, updatedMetrics, newMetricNames := checkConfig()

	// Check if config has changed
	configChanged := checkForConfigChanges(newMetricNames, metrics)

	// TODO: break up into own method under metric handling
	if configChanged {
		// Check if elements were deleted, delete corresponding entries if so
		metricDeleted, deletedMetrics := checkForDeletedMetrics(metrics, newMetricNames)
		if metricDeleted {
			data.Data, metrics = deleteMetrics(data.Data, deletedMetrics)
		}
		// Check if elements were added, add new entries if so
		metricAdded, addedMetrics := checkForAddedMetrics(metrics, newMetricNames)
		if metricAdded {
			data.Data = addMetrics(addedMetrics, data.Data)
		}
		// Check if elements were rearranged, properly order them
		// use correct metrics
		data.Metrics = updatedMetrics
		data = updateMetrics(data, metrics, newMetricNames)
	}

	storeJSON(data)

	cfg := ReadConfig()

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

// ##################################
// ## METRIC MAINTENANCE & UPDATES ##
// ##################################
// TODO: create a method here that combines all metric checks and updates

func addMetrics(addedMetrics []string, data []MetricData) []MetricData {
	newData := data
	for dontCare := 0; dontCare < len(addedMetrics); dontCare++ {
		newData = append(newData, MetricData{})
	}
	if len(addedMetrics) > 0 {
		counter := 0
		for idx, _ := range newData {
			if newData[idx].Name == "" {
				newData[idx].Name = addedMetrics[counter]
				counter += 1
			}
		}
	}
	return newData
}

func updateMetrics(data EntryData, metrics []string, newMetricNames []string) EntryData {
	metrics = newMetricNames
	newData := make([]MetricData, len(metrics))
	for idx, ele := range data.Data {
		foundCorrectIdx := false
		for sIdx := 0; sIdx < len(metrics); sIdx++ {
			if ele.Name == metrics[sIdx] {
				newData[sIdx] = ele
				newData[sIdx].Name = metrics[sIdx]
				foundCorrectIdx = true
				break
				// fmt.Printf("newData2: %v\n", newData)
			}
		}
		if foundCorrectIdx == false {
			newData[idx] = MetricData{
				Name:   metrics[idx],
				Date:   []time.Time{},
				Value:  []string{},
				Color1: data.Metrics[idx][2],
				Color2: data.Metrics[idx][3],
			}
		}
		data.Data = newData
	}
	return data
}

func deleteMetrics(data []MetricData, deletedMetrics []string) ([]MetricData, []string) {
	var newData []MetricData
	newMetrics := []string{}

	for _, currData := range data {
		// if metric is not in deletedMetrics, add data to new data
		if !contains(deletedMetrics, currData.Name) {
			newData = append(newData, currData)
			newMetrics = append(newMetrics, currData.Name)
		}
	}
	// fmt.Printf("newData: %v\n", newData)
	return newData, newMetrics
}

// ##########################
// ## VISUAL UI COMPONENTS ##
// ##########################

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

// ###################
// ## VIEW UPDATING ##
// ###################

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
	// TODO: break this up, put handling of key input into ui file
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

// ####################
// ## GRID RENDERING ##
// ####################

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

func createGrid(data EntryData, format string, startDate time.Time, metric int) [][]string {

	var firstDay time.Weekday
	var numOfDays int
	var sizeX int
	sizeY := 7
	if format == "month" {
		sizeX, numOfDays = prepareMonthView(numOfDays, sizeX, startDate)
	} else if format == "year" {
		sizeX, numOfDays = prepareYearView(numOfDays, sizeX, startDate)
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

func prepareMonthView(numOfDays int, sizeX int, startDate time.Time) (int, int) {
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
	return sizeX, numOfDays
}

func prepareYearView(numOfDays int, sizeX int, startDate time.Time) (int, int) {
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

	return sizeX, numOfDays
}
