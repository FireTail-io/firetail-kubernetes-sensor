FROM golang:1.24-bullseye
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .
RUN chmod +x main
CMD ["./main"]
