FROM golang:1.24-bullseye
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN GOARCH=amd64 GOOS=linux go build -o /dist/main .
RUN chmod +x /dist/main
CMD ["/dist/main"]
