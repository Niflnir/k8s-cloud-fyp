package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func CalculateDurationMilliseconds(startTime string, endTime string) int {
	startTime = convertToMilliseconds(startTime)
	endTime = convertToMilliseconds(endTime)
	fmt.Println("---")
	fmt.Println(startTime)
	fmt.Println(endTime)
	fmt.Println("---")

	startParts := strings.Split(startTime, ",")
	endParts := strings.Split(endTime, ",")

	startSeconds, _ := strconv.Atoi(strings.ReplaceAll(startParts[0], ":", ""))
	endSeconds, _ := strconv.Atoi(strings.ReplaceAll(endParts[0], ":", ""))

	startMillis, _ := strconv.Atoi(startParts[1])
	endMillis, _ := strconv.Atoi(endParts[1])

	secondsDiff := endSeconds - startSeconds
	millisDiff := endMillis - startMillis

	return (secondsDiff * 1000) + millisDiff
}

func convertToMilliseconds(time string) string {
	if !strings.Contains(time, ",") {
		time += ",000"
	}
	return time
}
