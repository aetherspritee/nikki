package src

import (
	"regexp"
	"strconv"
	"time"
)

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func reformatData(data string, MetricInfo string) int {
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
		// use the time.AddDate() method to check whether previous or following date exists in list
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
