FROM alpine:3.21

USER root
RUN apk add bcc-tools bcc-dev bcc-doc linux-headers build-base go clang libbpf-dev gcc musl-dev openssl

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN clang -O2 -g -target bpf -D__TARGET_ARCH_x86 -c ssl_read.c -o ssl_read.o
RUN CGO_ENABLED=1 go build -o probe -ldflags="-linkmode=external" .

ENTRYPOINT ["./probe"]

# To build and run me, do:
# docker build . -t libssl && docker run --rm -it --privileged libssl
# Then in another terminal do:
# docker exec -it <container_id> wget -qO- https://www.google.com
# You should see logs about SSL connections in the first terminal.
