package utils

import (
	"bufio"
	"os"
	"strings"
)

func ArrayContains(array []string, check string) bool {
	for _, ele := range array {
		if ele == check {
			return true
		}
	}
	return false
}

func ArrayContainsNoCase(array []string, check string) bool {
	for _, ele := range array {
		if strings.ToLower(ele) == strings.ToLower(check) {
			return true
		}
	}
	return false
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.ToLower(scanner.Text()))
	}
	return lines, scanner.Err()
}
