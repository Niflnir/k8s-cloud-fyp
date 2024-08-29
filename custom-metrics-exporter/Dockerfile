FROM golang:alpine3.20

WORKDIR /app

COPY . .

RUN go build -o /app/custom-metrics-exporter .

EXPOSE 8080

CMD ["/app/custom-metrics-exporter"]
