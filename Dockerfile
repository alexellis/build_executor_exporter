FROM golang:1.7.5

WORKDIR /go/src/github.com/alexellis/build_executor_exporter
COPY app.go /go/src/github.com/alexellis/build_executor_exporter

RUN go get -d -v

RUN go build -o build_executor_exporter

EXPOSE 9001

ENTRYPOINT ["./build_executor_exporter"]

