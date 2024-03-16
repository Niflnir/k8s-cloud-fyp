import re
from kubernetes import client, config, watch
from prometheus_client import start_http_server, disable_created_metrics, Counter, Gauge, Histogram

SERVER_REQUESTS = Counter('server_requests', 'Number of requests received by the server')
DECODING_REQUESTS_FAILED = Counter('decoding_requests_failed', 'Number of failed decoding requests')
DECODING_REQUESTS_SUCCESSFUL = Counter('decoding_requests_successful', 'Number of successful decoding requests')
SPEECH_WORKER_COUNT = Gauge('speech_worker_count', 'Number of speech workers currently available')
REQUEST_DURATION = Histogram('request_duration_milliseconds', 'Time taken to complete a request', ['request_id'])
REQUEST_LATENCY = Histogram('request_latency_milliseconds', 'Request latency in milliseconds')


def calculate_duration_milliseconds(start_time, end_time):
    start_time_parts = start_time.split(',')
    end_time_parts = end_time.split(',')
    start_seconds = int(start_time_parts[0].replace(':', ''))
    end_seconds = int(end_time_parts[0].replace(':', ''))
    return (end_seconds - start_seconds) * 1000 + (int(end_time_parts[1]) - int(start_time_parts[1]))


def update_latency_metric(log_line):
    pattern = r".*101 GET.*?(\d+\.\d+)ms"
    match = re.search(pattern, log_line)
    if match:
        latency_milliseconds = float(match.group(1))
        REQUEST_LATENCY.observe(latency_milliseconds)


def update_request_count_and_duration_metrics(log_line):
    global start_time
    global end_time

    request_start_pattern = re.compile(r'INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+): OPEN')
    request_end_pattern = re.compile(r'INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+): Sending event')

    request_start_match = request_start_pattern.search(log_line)
    if request_start_match:
        start_time, _ = request_start_match.groups()
        SERVER_REQUESTS.inc()
        return

    request_end_match = request_end_pattern.search(log_line)
    if request_end_match:
        end_time, end_request_id = request_end_match.groups()

        duration_milliseconds = calculate_duration_milliseconds(start_time, end_time)
        REQUEST_DURATION.labels(request_id=end_request_id).observe(duration_milliseconds)


def update_decoding_requests_metrics(log_line):
    pattern = r"INFO.* Sending event \{'status': (\d+).*"
    match = re.search(pattern, log_line)
    if match:
        status = int(match.group(1))
        # Status code (integer):
        # 0: Success. Recognition is successful and results sent.
        # 1: No speech. The server sends a 'status':1 when it detects more than 10s of audio without a speaker, and ends the session.
        # 2: Aborted. Recognition was aborted.
        # 9: Not available. All recognizer processes are currently in use and recognition cannot be performed.
        if status == 0: 
            DECODING_REQUESTS_SUCCESSFUL.inc()
        else:
            DECODING_REQUESTS_FAILED.inc()


def update_speech_worker_count_metric(log_line):
    pattern = r'INFO.* Number of worker available (\d+)'
    match = re.search(pattern, log_line)
    if match:
        SPEECH_WORKER_COUNT.set(int(match.group(1)))


def parse_log(log_line):
    update_speech_worker_count_metric(log_line)
    update_decoding_requests_metrics(log_line)
    update_request_count_and_duration_metrics(log_line)
    update_latency_metric(log_line)


def main():
    disable_created_metrics()
    start_http_server(8080)

    config.load_incluster_config()
    v1 = client.CoreV1Api()

    label_selector = 'app=decoding-sdk-server'
    namespace = 'monitoring'
    container = 'decoding-sdk-server'

    ret = v1.list_namespaced_pod(namespace=namespace, label_selector=label_selector)

    pod = ret.items[0]
    pod_name = pod.metadata.name
    w = watch.Watch()
    for log_line in w.stream(v1.read_namespaced_pod_log, name=pod_name, namespace=namespace, container=container, tail_lines=1, follow=True):
        parse_log(log_line)


if __name__ == "__main__":
    main()
