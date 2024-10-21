package parser

import (
	"regexp"
)

func parsePauseInstanceLog(logLine string) {
	pauseInstanceRegex := regexp.MustCompile(`.*(\d{2}:\d{2}:\d{2},\d{3}).* (\w+-\w+-\w+-\w+-\w+): Pause the instance.*`)
	match := pauseInstanceRegex.FindStringSubmatch(logLine)
	if match == nil {
		return
	}

	pauseInstanceTimestamp, requestId := match[1], match[2]
	PauseInstanceTimestampMap[requestId] = pauseInstanceTimestamp
}

func ParseWorkerLog(logLine string) {
	parsePauseInstanceLog(logLine)
}
