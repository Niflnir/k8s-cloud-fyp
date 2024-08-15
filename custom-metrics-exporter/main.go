package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"custom-metrics-exporter/parser"
)

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
			parser.ParseServerLog(logLine)
		} else if strings.Contains(podName, "worker") {
			parser.ParseWorkerLog(logLine)
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
