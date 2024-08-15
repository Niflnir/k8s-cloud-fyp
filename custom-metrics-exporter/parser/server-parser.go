package parser

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"custom-metrics-exporter/utils"
)

var (
	DecodingRequestsFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "decoding_requests_failed",
		Help: "Number of failed decoding requests",
	})
	DecodingRequestsSuccessful = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "decoding_requests_successful",
		Help: "Number of successful decoding requests",
	})
	SpeechWorkerCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "speech_worker_count",
		Help: "Number of speech workers currently available",
	})
	RequestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "request_duration_milliseconds",
		Help: "Time taken to complete a request",
	})
	ServerWorkerLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "latency_milliseconds",
		Help:    "Request latency between server and worker",
		Buckets: []float64{30, 60, 90, 120, 150, 180, 210, 240, 270, 300, float64(math.Inf(1))},
	})
	RealTimeFactor = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "real_time_factor",
		Help:    "Real Time Factor of decoding request",
		Buckets: []float64{1, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, float64(math.Inf(1))},
	})
	OpenConnectionTimestampMap = map[string]string{}
	PauseInstanceTimestampMap  = map[string]string{}
	AudioStartTimestamp        string
	// AudioEndTimestamp          string
	SendingEventTimestampMap = map[string]string{}
)

func parseOpenConnectionLog(logLine string) bool {
	requestStartRegex := regexp.MustCompile(`INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+): OPEN`)
	match := requestStartRegex.FindStringSubmatch(logLine)
	if match == nil {
		return false
	}

	openConnectionTimestamp, requestId := match[1], match[2]
	OpenConnectionTimestampMap[requestId] = openConnectionTimestamp

	return true
}

func parseAvailableWorkersLog(logLine string) bool {
	speechWorkerCountRegex := regexp.MustCompile(`INFO.* Number of worker available (\d+)`)
	match := speechWorkerCountRegex.FindStringSubmatch(logLine)
	if match == nil {
		return false
	}

	workerCount, err := strconv.Atoi(match[1])
	if err != nil {
		return false
	}

	fmt.Println("match found: ", workerCount)
	SpeechWorkerCount.Set(float64(workerCount))
	return true
}

func parseSendingEventLog(logLine string) bool {
	sendingEventRegex := regexp.MustCompile(`INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+).* Sending event \{'status': (\d+).*result`)
	match := sendingEventRegex.FindStringSubmatch(logLine)
	if match == nil {
		return false
	}

	sendingEventTimestamp, requestId, statusCode := match[1], match[2], match[3]
	SendingEventTimestampMap[requestId] = sendingEventTimestamp

	/*
		Status code (integer):
		0: Success. Recognition is successful and results sent.
		1: No speech. The server sends a 'status':1 when it detects more than 10s of audio without a speaker, and ends the session.
		2: Aborted. Recognition was aborted.
		9: Not available. All recognizer processes are currently in use and recognition cannot be performed.
	*/
	if statusCode == "0" {
		// fmt.Println("Decoding request successful (requestId:", requestId, ")")
		DecodingRequestsSuccessful.Inc()
	} else {
		// fmt.Println("Decoding request failed (requestId:", requestId, ", status:", statusCode, ")")
		DecodingRequestsFailed.Inc()
	}

	return true
}

func updateMetricsOnConnectionCloseLog(logLine string) {
	connectionCloseRegex := regexp.MustCompile(`INFO.* (\w+-\w+-\w+-\w+-\w+): Handling on_connection_close()`)
	match := connectionCloseRegex.FindStringSubmatch(logLine)
	if match == nil {
		return
	}

	requestId := match[1]
	fmt.Println("Request id: ", requestId)
	openConnectionTimestamp, openConnectionTimestampExists := OpenConnectionTimestampMap[requestId]
	pauseInstanceTimestamp, pauseInstanceTimestampExists := PauseInstanceTimestampMap[requestId]
	sendingEventTimestamp, sendingEventTimestampExists := SendingEventTimestampMap[requestId]

	// Request duration = Sending event timestamp - Open connection timestamp
	if sendingEventTimestampExists && openConnectionTimestampExists {
		requestDuration := utils.CalculateDurationMilliseconds(openConnectionTimestamp, sendingEventTimestamp)
		fmt.Println("Request Duration: ", requestDuration)
		RequestDuration.Observe(float64(requestDuration))
	}

	// Latency between server and worker = Sending event timestamp (server) - Pause instance timestamp (worker)
	if sendingEventTimestampExists && pauseInstanceTimestampExists {
		latencyDuration := utils.CalculateDurationMilliseconds(pauseInstanceTimestamp, sendingEventTimestamp)
		fmt.Println("Latency Duration: ", latencyDuration)
		ServerWorkerLatency.Observe(float64(latencyDuration))
	}

	// Real Time Factor = Request duration / Audio duration
	// if sendingEventTimestampExists && openConnectionTimestampExists && pauseInstanceTimestampExists {
	// 	requestDuration := utils.CalculateDurationMilliseconds(openConnectionTimestamp, sendingEventTimestamp)
	// 	audioDuration := utils.CalculateDurationMilliseconds(AudioStartTimestamp, pauseInstanceTimestamp)
	// 	RealTimeFactor.Observe(float64(requestDuration) / float64(audioDuration))
	// 	fmt.Println("Request duration: ", requestDuration)
	// 	fmt.Println("Audio duration: ", audioDuration)
	// 	fmt.Println("Real time factor: ", float64(requestDuration)/float64(audioDuration))
	// }

	// Clear request ids from maps
	delete(OpenConnectionTimestampMap, requestId)
	delete(PauseInstanceTimestampMap, requestId)
	delete(SendingEventTimestampMap, requestId)
}

func ParseServerLog(logLine string) {
	if parseAvailableWorkersLog(logLine) {
		return
	} else if parseSendingEventLog(logLine) {
		return
	} else if parseOpenConnectionLog(logLine) {
		return
	}
	updateMetricsOnConnectionCloseLog(logLine)
}
