import re
import concurrent.futures
from kubernetes import client, config, watch
from prometheus_client import start_http_server, disable_created_metrics, Counter, Gauge, Histogram, Summary

SERVER_REQUESTS = Counter('server_requests', 'Number of requests received by the server')
DECODING_REQUESTS_FAILED = Counter('decoding_requests_failed', 'Number of failed decoding requests')
DECODING_REQUESTS_SUCCESSFUL = Counter('decoding_requests_successful', 'Number of successful decoding requests')
SPEECH_WORKER_COUNT = Gauge('speech_worker_count', 'Number of speech workers currently available')
REQUEST_DURATION = Histogram('request_duration_milliseconds', 'Time taken to complete a request', buckets=[100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, float('inf')])
LATENCY = Histogram('latency_milliseconds', 'Latency in milliseconds', buckets=[30, 60, 90, 120, 150, 180, 210, 240, 270, 300, float('inf')])
AUDIO_LENGTH = Histogram('audio_length_milliseconds', 'Total length of the audio in milliseconds')
REAL_TIME_FACTOR = Histogram('real_time_factor', 'Real Time Factor of decoding request')

previous_log_line = ''
request_dict = {}
decoding_requests_dict = {}
real_time_factor_dict = {}


def calculate_duration_milliseconds(start_time, end_time):
    start_time_parts = start_time.split(',')
    end_time_parts = end_time.split(',')
    start_seconds = int(start_time_parts[0].replace(':', ''))
    end_seconds = int(end_time_parts[0].replace(':', ''))
    return (end_seconds - start_seconds) * 1000 + (int(end_time_parts[1]) - int(start_time_parts[1]))


def update_latency_metric_server(log_line):
    sending_event_pattern = re.compile(r'INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+): Sending event.*')
    connection_close_pattern = re.compile(r'INFO.* (\d{2}:\d{2}:\d{2},\d{3}) (\w+-\w+-\w+-\w+-\w+): Handling on_connection_close.*')

    sending_event_match = sending_event_pattern.search(log_line)
    if sending_event_match:
        sending_event_time, request_id = sending_event_match.groups()
        if request_id in request_dict:
            duration_milliseconds = calculate_duration_milliseconds(request_dict.pop(request_id, None), sending_event_time)
            LATENCY.observe(duration_milliseconds)
        return

    connection_close_match = connection_close_pattern.search(log_line)
    if connection_close_match:
        connection_close_time, request_id = connection_close_match.groups()
        if request_id in request_dict:
            duration_milliseconds = calculate_duration_milliseconds(request_dict.pop(request_id, None), connection_close_time)
            LATENCY.observe(duration_milliseconds)


def update_latency_metric_worker(log_line):
    pause_instance_pattern = re.compile(r'.*(\d{2}:\d{2}:\d{2},\d{3}).* (\w+-\w+-\w+-\w+-\w+): Pause the instance.*')
    pause_instance_match = pause_instance_pattern.search(log_line)

    if pause_instance_match:
        pause_instance_time, request_id = pause_instance_match.groups()
        request_dict[request_id] = pause_instance_time


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
        if end_request_id in real_time_factor_dict:
            REAL_TIME_FACTOR.observe(duration_milliseconds / real_time_factor_dict.pop(end_request_id, None))
        else:
            real_time_factor_dict[end_request_id] = duration_milliseconds

        REQUEST_DURATION.observe(duration_milliseconds)


def update_decoding_requests_metrics(log_line):
    global previous_log_line

    if (log_line == previous_log_line):
        return

    previous_log_line = log_line
    sending_event_pattern = re.compile(r"INFO.* (\w+-\w+-\w+-\w+-\w+).* Sending event \{'status': (\d+).*")
    sending_event_match = sending_event_pattern.search(log_line)
    if sending_event_match:
        request_id, status = sending_event_match.groups()
        # Status code (integer):
        # 0: Success. Recognition is successful and results sent.
        # 1: No speech. The server sends a 'status':1 when it detects more than 10s of audio without a speaker, and ends the session.
        # 2: Aborted. Recognition was aborted.
        # 9: Not available. All recognizer processes are currently in use and recognition cannot be performed.
        if request_id not in decoding_requests_dict:
            if status == '0':
                    decoding_requests_dict[request_id] = True
                    DECODING_REQUESTS_SUCCESSFUL.inc()
            else:
                    decoding_requests_dict[request_id] = True
                    DECODING_REQUESTS_FAILED.inc()


def update_speech_worker_count_metric(log_line):
    pattern = r'INFO.* Number of worker available (\d+)'
    match = re.search(pattern, log_line)
    if match:
        SPEECH_WORKER_COUNT.set(int(match.group(1)))


def update_audio_length_metric(log_line):
    global start_audio_time
    global end_audio_time

    request_start_pattern = re.compile(r'.*(\d{2}:\d{2}:\d{2},\d{3}).* Get Audio File Sample Rate from Header.*')
    request_start_match = request_start_pattern.search(log_line)
    if request_start_match:
        start_audio_time = request_start_match.group(1)
        return

    request_end_pattern = re.compile(r'.*(\d{2}:\d{2}:\d{2},\d{3}).* received EOS.*')
    request_end_match = request_end_pattern.search(log_line)
    if request_end_match:
        end_audio_time = request_end_match.group(1)
        return

    request_pause_pattern = re.compile(r'.*(\w+-\w+-\w+-\w+-\w+).* Pause the instance.*')
    request_pause_match = request_pause_pattern.search(log_line)
    if request_pause_match:
        pause_request_id = request_pause_match.group(1)
        duration_milliseconds = calculate_duration_milliseconds(start_audio_time, end_audio_time)
        if pause_request_id in real_time_factor_dict:
            REAL_TIME_FACTOR.observe(real_time_factor_dict.pop(pause_request_id, None), duration_milliseconds)
        else:
            real_time_factor_dict[pause_request_id] = duration_milliseconds

        AUDIO_LENGTH.observe(duration_milliseconds)


def parse_server_log(log_line):
    update_speech_worker_count_metric(log_line)
    update_decoding_requests_metrics(log_line)
    update_request_count_and_duration_metrics(log_line)
    update_latency_metric_server(log_line)


def parse_worker_log(log_line):
    update_audio_length_metric(log_line)
    update_latency_metric_worker(log_line)


def parse_logs():
    config.load_incluster_config()
    v1 = client.CoreV1Api()
    namespace = 'monitoring'

    def stream_logs(container):
        w = watch.Watch()

        if container == 'decoding-sdk-server':
            ret = v1.list_namespaced_pod(namespace=namespace, label_selector='app=decoding-sdk-server')
            pod = ret.items[0]
            pod_name = pod.metadata.name
            for log_line in w.stream(v1.read_namespaced_pod_log, name=pod_name, namespace=namespace, container=container, tail_lines=1, follow=True):
                parse_server_log(log_line)

        elif container == 'decoding-sdk-worker':
            ret = v1.list_namespaced_pod(namespace=namespace, label_selector='app=decoding-sdk-worker')
            pod = ret.items[0]
            pod_name = pod.metadata.name
            for log_line in w.stream(v1.read_namespaced_pod_log, name=pod_name, namespace=namespace, container=container, tail_lines=1, follow=True):
                parse_worker_log(log_line)

    with concurrent.futures.ThreadPoolExecutor() as executor:
        futures = []
        futures.append(executor.submit(stream_logs, 'decoding-sdk-server'))
        futures.append(executor.submit(stream_logs, 'decoding-sdk-worker'))
        concurrent.futures.wait(futures)


def main():
    disable_created_metrics()
    start_http_server(8080)
    parse_logs()


if __name__ == "__main__":
    main()
