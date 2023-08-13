package src

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"time"
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
	cursorMode    cursor.CursorMode
	focusIndex    int
	wrongInput    bool
	wrongIndex    int
	generalConfig General
}

func InitialModel() model {
	data, metrics, updatedMetrics, newMetricNames := checkConfig()

	// Check if config has changed
	configChanged := checkForConfigChanges(newMetricNames, metrics)

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
