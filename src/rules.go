package main

var rules map[string]string = map[string]string{
	"int10": `^[0-9]$`,
	"int":   `^[0-9]+$`,
	"time":  `^([0-1][0-9]|2[0-3]):[0-5][0-9]$`,
}
