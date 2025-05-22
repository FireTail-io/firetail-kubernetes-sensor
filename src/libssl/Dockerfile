FROM alpine:3.21

USER root
RUN apk add bcc-tools bcc-dev bcc-doc linux-headers build-base go clang libbpf-dev gcc musl-dev openssl bpftool libbpf
RUN apk add llvm

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

# Compile the BPF program using Clang
# -O2 is the optimisation level required for the bytecode to pass the verifier
# We need to specify the target architecture so the compiler knows which  pt_regs structure definition to use
RUN clang -target bpf -D__TARGET_ARCH_arm64 -O2 -g -c ssl_read.c -o ssl_read.o

# Strip DWARF debugging information from the object file as we don't need it
RUN llvm-strip -g ssl_read.o

# Build the Go program
RUN CGO_ENABLED=1 go build -o probe -ldflags="-linkmode=external" .

ENTRYPOINT ["./probe"]


# On a mac, install lima, start an ubuntu VM, and go inside it:
#   brew install lima
#   limactl start --name=default --vm-type=vz --rosetta template://ubuntu-24.10
#   lima
# TODO: Automate installing Docker inside the VM by creating our own lima template
# Then install Docker inside the VM: https://docs.docker.com/engine/install/ubuntu/
# You can then build this image and run it:
#   sudo docker build . -t libssl && sudo docker run --rm -it --privileged --name libssl libssl
# Open another terminal, go inside the ubuntu VM, get the ID of the running container, go inside of it and run wget:
#   lima
#   sudo docker ps
#   sudo docker exec -it libssl wget -qO- https://www.google.com
