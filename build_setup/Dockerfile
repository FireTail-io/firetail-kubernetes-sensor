FROM golang:1.24-bullseye
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends libpcap-dev
COPY ./src/go.* ./
RUN go mod download
COPY ./src/ ./
RUN go build -o /dist/main .
RUN rm -rf /src/*
RUN chmod +x /dist/main
CMD ["/dist/main"]
