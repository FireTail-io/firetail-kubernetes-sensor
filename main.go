package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"

	"syscall"
	"unsafe"
)

// Adapted from https://www.bytesizego.com/blog/golang-byte-to-string

// Filter represents a classic BPF filter program that can be applied to a socket
type Filter []bpf.Instruction

// ApplyTo applies the current filter onto the provided file descriptor
func (filter Filter) ApplyTo(fd int) (err error) {
	var assembled []bpf.RawInstruction
	if assembled, err = bpf.Assemble(filter); err != nil {
		return err
	}

	var program = unix.SockFprog{
		Len:    uint16(len(assembled)),
		Filter: (*unix.SockFilter)(unsafe.Pointer(&assembled[0])),
	}
	var b = (*[unix.SizeofSockFprog]byte)(unsafe.Pointer(&program))[:unix.SizeofSockFprog]

	if _, _, errno := syscall.Syscall6(syscall.SYS_SETSOCKOPT,
		uintptr(fd), uintptr(syscall.SOL_SOCKET), uintptr(syscall.SO_ATTACH_FILTER),
		uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 0); errno != 0 {
		return errno
	}

	return nil
}

func parsePacket(data []byte) (*layers.IPv4, *layers.TCP, error) {
	packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
	if errLayer := packet.ErrorLayer(); errLayer != nil {
		return nil, nil, fmt.Errorf("error decoding packet: %w", errLayer.Error())
	}

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return nil, nil, fmt.Errorf("no IPv4 layer found")
	}

	ip, _ := ipLayer.(*layers.IPv4)
	if ip == nil {
		return nil, nil, fmt.Errorf("failed to parse IPv4 layer")
	}

	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return nil, nil, fmt.Errorf("no TCP layer found")
	}

	tcp, _ := tcpLayer.(*layers.TCP)
	if tcp == nil {
		return nil, nil, fmt.Errorf("failed to parse TCP layer")
	}
	if len(tcp.Payload) == 0 {
		return nil, nil, fmt.Errorf("no payload found in TCP layer")
	}

	return ip, tcp, nil
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}

func main() {
	const TARGET_PORT = 80

	// Start a simple HTTP server so we can send some requests to it
	http.HandleFunc("/", HelloServer)
	go http.ListenAndServe(":80", nil)

	// Create a raw socket to listen for TCP packets
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		log.Println("Failed to open raw socket", err.Error())
		return
	}

	// Apply a BPF filter to the socket
	err = Filter{
		bpf.LoadAbsolute{Off: 22, Size: 2},         // load the destination port
		bpf.JumpIf{Val: TARGET_PORT, SkipFalse: 1}, // if Val != TARGET_PORT skip next instruction
		bpf.RetConstant{Val: 0xffff},               // return 0xffff bytes (or less) from packet
		bpf.RetConstant{Val: 0x0},                  // return 0 bytes, effectively ignore this packet
	}.ApplyTo(fd)
	if err != nil {
		log.Println("Failed to apply filter", err.Error())
		return
	}

	// Receive packets in a loop
	log.Printf("🧐 Listening for packets on port %v...\n", TARGET_PORT)
	for {
		var buf [4096]byte
		n, _, err := syscall.Recvfrom(fd, buf[:], 0)
		if err != nil {
			log.Println("Failed to receive packet", err.Error())
			continue
		}

		ip, tcp, err := parsePacket(buf[:n])
		if err != nil {
			log.Println("😭 Failed to parse packet", err.Error())
			continue
		}

		log.Printf(
			"✅ Received packet from %s:%d to %s:%d with payload:\n----------START----------\n%s\n-----------END-----------\n",
			ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort, string(tcp.Payload),
		)
	}
}
