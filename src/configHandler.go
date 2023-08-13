package src

import (
	toml "github.com/naoina/toml"
	"os"
)

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

func ReadConfig() Config {
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

func checkConfig() (EntryData, []string, map[int][]string, []string) {
	cfg := ReadConfig()
	//fmt.Printf("metrics: %v\n", cfg.Metrics)
	metrics := []string{}
	data := loadJSON()
	for _, element := range data.Data {
		metrics = append(metrics, element.Name)
	}
	// first check whether metrics have changed
	// Handle: only removed metrics, only added metrics,
	// added and removed metrics

	//update the decoded data based on changes in config!
	updatedMetrics := make(map[int][]string, len(cfg.Metrics))
	updatedMetricsNames := []string{}
	newMetricNames := []string{}
	for index, element := range cfg.Metrics {
		updatedMetrics[index] = []string{element.Name, element.Rule, element.Color1, element.Color2}
		updatedMetricsNames = append(updatedMetricsNames, element.Name)
		if !contains(metrics, element.Name) {
			newMetricNames = append(newMetricNames, element.Name)
		}
	}
	for _, element := range metrics {
		if !contains(updatedMetricsNames, element) {
			newMetricNames = append(newMetricNames, element)
		}
	}

	return data, metrics, updatedMetrics, updatedMetricsNames
}

func checkForConfigChanges(updatedMetrics []string, metrics []string) bool {
	if len(updatedMetrics) != len(metrics) {
		return true
	}
	for idx, _ := range updatedMetrics {
		if updatedMetrics[idx] != metrics[idx] {
			return true
		}
	}
	return false
}

func checkForDeletedMetrics(metrics []string, newMetrics []string) (bool, []string) {
	change := false
	deletedMetrics := []string{}
	for _, metric := range metrics {
		if !contains(newMetrics, metric) {
			change = true
			deletedMetrics = append(deletedMetrics, metric)
		}
	}
	return change, deletedMetrics
}

func checkForAddedMetrics(metrics []string, newMetrics []string) (bool, []string) {
	change := false
	addedMetrics := []string{}
	for _, metric := range newMetrics {
		found := false
		for _, metric2 := range metrics {
			if metric2 == metric {
				found = true
			}
		}
		if !found {
			change = true
			addedMetrics = append(addedMetrics, metric)
		}
	}
	return change, addedMetrics
}
