package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/cilium/ebpf"
	"golang.org/x/sys/unix"
)

//go:embed ssl_read.o
var bpfBytecode []byte

type sslEvent struct {
	PidTgid uint64
	SslPtr  uint64
	Buffer  uint64
	Num     int32
	_       [4]byte // padding
}

func main() {
	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(bpfBytecode))
	if err != nil {
		log.Fatalf("loading spec: %v", err)
	}

	objs := struct {
		SslReadEnterV3 *ebpf.Program `ebpf:"ssl_read_enter_v3"`
		Events         *ebpf.Map     `ebpf:"events"`
	}{}

	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		log.Fatalf("loading objects: %v", err)
	}
	defer objs.SslReadEnterV3.Close()
	defer objs.Events.Close()

	// Change this path based on your system's OpenSSL path
	libssl := "/usr/lib/x86_64-linux-gnu/libssl.so.1.1"

	up, err := link.OpenExecutable(libssl)
	if err != nil {
		log.Fatalf("open executable: %v", err)
	}

	// Attach uprobe to SSL_read
	uprober, err := up.Uprobe("SSL_read", objs.SslReadEnterV3, nil)
	if err != nil {
		log.Fatalf("attach uprobe: %v", err)
	}
	defer uprober.Close()

	// Read events
	rd, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		log.Fatalf("create ringbuf reader: %v", err)
	}
	defer rd.Close()

	log.Println("Waiting for SSL_read calls...")

	for {
		record, err := rd.Read()
		if err != nil {
			log.Fatalf("read ringbuf: %v", err)
		}

		var evt sslEvent
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &evt); err != nil {
			log.Printf("decode event: %v", err)
			continue
		}

		pid := evt.PidTgid >> 32
		fmt.Printf("SSL_read: pid=%d ssl=0x%x buf=0x%x num=%d\n", pid, evt.SslPtr, evt.Buffer, evt.Num)
	}
}