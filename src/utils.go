package src

import (
	"github.com/lucasb-eyer/go-colorful"
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

func getColorMap(rangeMap map[string][]float64, data EntryData, metric int) map[string][]string {
	colorGrd := map[string][]string{}
	for index, _ := range rangeMap {
		x0y0, _ := colorful.Hex(data.Metrics[metric][2])
		x1y0, _ := colorful.Hex(data.Metrics[metric][3])
		x0 := make([]string, len(rangeMap[index]))
		for i := range x0 {
			x0[i] = x0y0.BlendLuv(x1y0, rangeMap[index][i]).Hex()
		}
		// if all values are equal x0 will be #000000
		if equalValues(x0) {
			for i := 0; i < len(x0); i++ {
				x0[i] = x0y0.Hex()
			}
		}
		colorGrd[index] = x0
	}
	return colorGrd
}
func equalValues(a []string) bool {
	for i := 1; i < len(a); i++ {
		if a[i] != a[0] {
			return false
		}
	}
	return true
}
