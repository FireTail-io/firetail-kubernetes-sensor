FROM golang:1.24-bullseye
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends libpcap-dev
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY main.go .
RUN go build -o /dist/main .
RUN rm -rf /src/*
RUN chmod +x /dist/main
CMD ["/dist/main"]
