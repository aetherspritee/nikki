package src

import (
	"encoding/json"
	"fmt"
	"github.com/aetherspritee/nikki/src/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"log"
	"os"
	"time"
)

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

// ##################################
// ### DATA STORAGE FUNCTIONALITY ###
// ##################################

func decodeJson() {

	var result EntryData
	jsonFile, err := os.ReadFile("test.json")
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
	_ = os.WriteFile("test.json", jsonData, 0644)
}

// stores JSON
func storeJSON(data EntryData) int {
	var jsonData []byte
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return 1
	}
	fmt.Println("Saving data")
	_ = os.WriteFile("data.json", jsonData, 0644)
	return 0
}

// loads JSON
func loadJSON() EntryData {
	var result EntryData
	jsonFile, err := os.ReadFile("data.json")
	if err != nil {
		panic(nil)
	}

	json.Unmarshal(jsonFile, &result)
	return result
}

// #######################################
// ### DATA MANIPULATION FUNCTIONALITY ###
// #######################################

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

func newEntry() {
	file := loadJSON()
	// check if there is an entry for today already
	file = addEntry(file)
	storeJSON(file)
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
