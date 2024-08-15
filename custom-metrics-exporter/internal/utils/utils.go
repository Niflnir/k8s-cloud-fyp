package utils

import (
	"strconv"
	"strings"
)

func calculateDurationMilliseconds(startTime string, endTime string) int {
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
