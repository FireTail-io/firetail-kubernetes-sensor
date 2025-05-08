FROM golang:1.24-bullseye
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends libpcap-dev
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN GOARCH=amd64 GOOS=linux go build -o /dist/main .
RUN chmod +x /dist/main
CMD ["/dist/main"]
