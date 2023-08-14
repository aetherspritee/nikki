package src

import (
	"regexp"
	"strconv"
)

var rules map[string]string = map[string]string{
	"int10": `^[0-9]$`,
	"int":   `^[0-9]+$`,
	"bool":  `^0|1$`,
	"goal":  `^goal (int|time)$`, // i want to have the option to provide a goal in different formats that is then compared with input values
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
