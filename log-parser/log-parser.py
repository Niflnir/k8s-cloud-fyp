from prometheus_client import start_http_server, Summary, Counter, Gauge
import time
import re


server_http_requests_total = Counter('server_http_requests_total', 'Total number of http requests received by the server')
worker_count_metric = Gauge('worker_count', 'Number of speech workers available')

# Path to the log file
log_file_path = '/opt/decoding-sdk-logs/master_server.log'


def extract_worker_count(log_line):
    pattern = r'Number of worker available (\d+)'
    match = re.search(pattern, log_line)
    if match:
        worker_count = int(match.group(1))
        return worker_count
    else:
        return 0


def parse_log(log_line):
    if "OPEN" in log_line:
        server_http_requests_total.inc()

    worker_count = extract_worker_count(log_line)
    worker_count_metric.set(worker_count)


def main():
    start_http_server(8080)
    with open(log_file_path, 'r') as log_file:
        while True:
            line = log_file.readline()
            if not line:
                time.sleep(1)
                continue
            parse_log(line)


if __name__ == "__main__":
    main()

