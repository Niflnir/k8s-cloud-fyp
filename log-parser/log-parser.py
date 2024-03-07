from prometheus_client import start_http_server, Summary, Counter
import time


http_requests_total = Counter('http_requests_total', 'Total number of http requests received by the server')

# Path to the log file
log_file_path = '/opt/decoding-sdk-logs/master_server.log'


def parse_log(line):
    if "OPEN" in line:
        http_requests_total.inc()


def main():
    # Start up the server to expose the metrics.
    start_http_server(8080)
    with open(log_file_path, 'r') as log_file:
        while True:
            line = log_file.readline()
            if not line:
                time.sleep(1)
                continue
            parse_log(line)
            # metrics = parse_log(line)
            # # Assuming metrics is a dictionary with metric names and values
            # for metric_name, value in metrics.items():
            #     LOG_PARSE_TIME.labels(metric_name=metric_name).observe(value)


if __name__ == "__main__":
    main()

