package main

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
)

func updateSpeechWorkerCountMetric(logLine string) error {
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

func parseServerLog(logLine string) {
	updateSpeechWorkerCountMetric(logLine)
}

func parseWorkerLog(logLine string) {
}

func getContainerLogs(clientset *kubernetes.Clientset, namespace string, podName string) {
	podLogOpts := v1.PodLogOptions{
		Follow: true,
	}
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error streaming pod logs:", err)
	}
	defer podLogs.Close()

	scanner := bufio.NewScanner(podLogs)
	for scanner.Scan() {
		logLine := scanner.Text()
		if strings.Contains(podName, "server") {
			parseServerLog(logLine)
		} else if strings.Contains(podName, "worker") {
			parseWorkerLog(logLine)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading standard input:", err)
	}
}

func getPodName(clientset *kubernetes.Clientset, namespace string, labelSelector string) string {
	podName := ""
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		panic(err)
	}

	// Extract pod names
	if len(pods.Items) > 0 {
		podName = pods.Items[0].Name
	} else {
		panic("No pods found with the given label selector")
	}

	return podName
}

func main() {
	// Load kubeconfig
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		panic(err)
	}

	// Get server and worker pod names
	namespace := "decoding-sdk"
	decodingSdkServerPodName := getPodName(clientset, namespace, "app=decoding-sdk-server")
	decodingSdkWorkerPodName := getPodName(clientset, namespace, "app=decoding-sdk-worker")

	// Spawn go routines to watch and parse server and worker pod logs
	go getContainerLogs(clientset, namespace, decodingSdkServerPodName)
	go getContainerLogs(clientset, namespace, decodingSdkWorkerPodName)

	// Let main routine run indefinitely
	select {}
}
