FROM golang:1.24-bullseye
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends libpcap-dev
COPY ./src/ ./
RUN go mod download
COPY . .
RUN go build -o /dist/main .
RUN chmod +x /dist/main
CMD ["/dist/main"]
