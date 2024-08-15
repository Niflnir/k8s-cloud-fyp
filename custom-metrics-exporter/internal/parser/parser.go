package parser

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	SERVER_REQUESTS = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "server_requests",
		Help: "Number of requests received by the server",
	})
	DECODING_REQUESTS_FAILED = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "decoding_requests_failed",
		Help: "Number of failed decoding requests",
	})
	DECODING_REQUESTS_SUCCESSFUL = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "decoding_requests_successful",
		Help: "Number of successful decoding requests",
	})
	SPEECH_WORKER_COUNT = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "speech_worker_count",
		Help: "Number of speech workers currently available",
	})
	REQUEST_DURATION = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "request_duration_milliseconds",
		Help: "Time taken to complete a request",
	})
	LATENCY = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "latency_milliseconds",
		Help:    "Request latency",
		Buckets: []float64{30, 60, 90, 120, 150, 180, 210, 240, 270, 300, float64(math.Inf(1))},
	})
	REAL_TIME_FACTOR = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "real_time_factor",
		Help:    "Real Time Factor of decoding request",
		Buckets: []float64{1, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, float64(math.Inf(1))},
	})
	requestStartString string
	requestEndString   string
)

func parseRequestStart(logLine string) {
	requestStartRegex := regexp.MustCompile(`INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+): OPEN`)
	match := requestStartRegex.FindStringSubmatch(logLine)

	if match == nil {
		return
	}

	requestStartString = match[1]
	fmt.Println(requestStartString)
}

func parseRequestEnd(logLine string) {
	requestEndRegex := regexp.MustCompile(`INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+): Sending event.*result`)
	match := requestEndRegex.FindStringSubmatch(logLine)

	if match == nil {
		return
	}

	requestEndString = match[1]
	duration := calculateDurationMilliseconds(requestStartString, requestEndString)
	REQUEST_DURATION.Observe(float64(duration))
}

func parseSpeechWorkerCount(logLine string) error {
	speechWorkerCountRegex := regexp.MustCompile(`INFO.* Number of worker available (\d+)`)
	match := speechWorkerCountRegex.FindStringSubmatch(logLine)

	if match == nil {
		return nil
	}

	workerCount, err := strconv.Atoi(match[1])
	if err != nil {
		return fmt.Errorf("error converting worker count to integer: %w", err)
	}

	fmt.Println("match found: ", workerCount)
	SPEECH_WORKER_COUNT.Set(float64(workerCount))
	return nil
}

func parseDecodingRequests(logLine string) {
	sendingEventRegex := regexp.MustCompile(`INFO.* (\w+-\w+-\w+-\w+-\w+).* Sending event \{'status': (\d+).*result`)

	match := sendingEventRegex.FindStringSubmatch(logLine)

	if match == nil {
		return
	}

	requestId, statusCode := match[1], match[2]
	/*
		Status code (integer):
		0: Success. Recognition is successful and results sent.
		1: No speech. The server sends a 'status':1 when it detects more than 10s of audio without a speaker, and ends the session.
		2: Aborted. Recognition was aborted.
		9: Not available. All recognizer processes are currently in use and recognition cannot be performed.
	*/
	if statusCode == "0" {
		fmt.Println("Decoding request successful (requestId:", requestId, ")")
	} else {
		fmt.Println("Decoding request failed (requestId:", requestId, ", status:", statusCode, ")")
	}
}
