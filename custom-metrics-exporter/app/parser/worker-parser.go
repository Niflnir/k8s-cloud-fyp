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

// func parseAudioStartLog(logLine string) {
// 	audioStartRegex := regexp.MustCompile(`.*(\d{2}:\d{2}:\d{2},\d{3}).* Get Audio File Sample Rate from Header.*`)
// 	match := audioStartRegex.FindStringSubmatch(logLine)
// 	if match == nil {
// 		return
// 	}
//
// 	AudioStartTimestamp = match[1]
// 	fmt.Println("Audio start timestamp: ", AudioStartTimestamp)
// }

// func parseAudioEndLog(logLine string) {
// 	audioEndRegex := regexp.MustCompile(`.*(\d{2}:\d{2}:\d{2},\d{3}).* received EOS.*`)
// 	match := audioEndRegex.FindStringSubmatch(logLine)
// 	if match == nil {
// 		return
// 	}
//
// 	AudioEndTimestamp = match[1]
// 	fmt.Println("Audio end timestamp: ", AudioEndTimestamp)
// }

func ParseWorkerLog(logLine string) {
	parsePauseInstanceLog(logLine)
	// parseAudioStartLog(logLine)
	// parseAudioEndLog(logLine)
}
