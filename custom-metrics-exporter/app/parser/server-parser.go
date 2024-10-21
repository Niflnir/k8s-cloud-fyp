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
	AvailableWorkers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "available_workers",
		Help: "Number of speech workers currently available",
	})
	ServerWorkerLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "latency_milliseconds",
		Help:    "Request latency between server and worker",
		Buckets: []float64{30, 60, 90, 120, 150, 180, 210, 240, 270, 300, float64(math.Inf(1))},
	})
	OpenConnectionTimestampMap   = map[string]string{}
	PauseInstanceTimestampMap    = map[string]string{}
	ForwardingClientTimestampMap = map[string]string{}
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

	fmt.Println("Worker count: ", workerCount)
	AvailableWorkers.Set(float64(workerCount))
	return true
}

func parseSendingEventToClientLog(logLine string) bool {
	sendingEventRegex := regexp.MustCompile(`INFO.* Sending event \{'status': (\d+).*result`)
	match := sendingEventRegex.FindStringSubmatch(logLine)
	if match == nil {
		return false
	}

	statusCode := match[1]

	/*
		Status code (integer):
		0: Success. Recognition is successful and results sent.
		1: No speech. The server sends a 'status':1 when it detects more than 10s of audio without a speaker, and ends the session.
		2: Aborted. Recognition was aborted.
		9: Not available. All recognizer processes are currently in use and recognition cannot be performed.
	*/
	if statusCode == "0" {
		DecodingRequestsSuccessful.Inc()
	} else {
		DecodingRequestsFailed.Inc()
	}

	return true
}

func parseForwardingClientLog(logLine string) bool {
	forwardingClientRegex := regexp.MustCompile(`.*(\d{2}:\d{2}:\d{2}).* (\w+-\w+-\w+-\w+-\w+): Forwarding client message.* to worker`)
	match := forwardingClientRegex.FindStringSubmatch(logLine)
	if match == nil {
		return false
	}

	forwardingClientTimestamp, requestId := match[1], match[2]
	ForwardingClientTimestampMap[requestId] = forwardingClientTimestamp

	return true
}

func updateMetricsOnConnectionCloseLog(logLine string) {
	connectionCloseRegex := regexp.MustCompile(`INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+): Handling on_connection_close()`)
	match := connectionCloseRegex.FindStringSubmatch(logLine)
	if match == nil {
		return
	}

	_, requestId := match[1], match[2]

	pauseInstanceTimestamp, pauseInstanceTimestampExists := PauseInstanceTimestampMap[requestId]
	fowardingClientTimestamp, forwardingClientTimestampExists := ForwardingClientTimestampMap[requestId]

	// Latency between server and worker = Forwarding client message timestamp (server) - Pause instance timestamp (worker)
	if forwardingClientTimestampExists && pauseInstanceTimestampExists {
		latencyDuration := utils.CalculateDurationMilliseconds(fowardingClientTimestamp, pauseInstanceTimestamp)
		fmt.Println("Latency Duration: ", latencyDuration)
		ServerWorkerLatency.Observe(float64(latencyDuration))
	}

	// Clear request ids from maps
	delete(OpenConnectionTimestampMap, requestId)
	delete(PauseInstanceTimestampMap, requestId)
	delete(ForwardingClientTimestampMap, requestId)
}

func ParseServerLog(logLine string) {
	if parseAvailableWorkersLog(logLine) {
		return
	} else if parseSendingEventToClientLog(logLine) {
		return
	} else if parseOpenConnectionLog(logLine) {
		return
	} else if parseForwardingClientLog(logLine) {
		return
	}
	updateMetricsOnConnectionCloseLog(logLine)
}
