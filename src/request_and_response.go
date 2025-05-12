package main

import (
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
)

type httpRequestAndResponse struct {
	request  *http.Request
	response *http.Response
	src      string
	dst      string
	srcPort  string
	dstPort  string
}

type httpRequestAndResponseStreamer struct {
	bpfExpression             string
	requestAndResponseChannel *chan httpRequestAndResponse
	ipManager                 *serviceIpManager
}

func (s *httpRequestAndResponseStreamer) start() {
	handle, err := pcap.OpenLive("any", 1600, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	err = handle.SetBPFFilter(s.bpfExpression)
	if err != nil {
		log.Fatal(err)
	}

	assembler := tcpassembly.NewAssembler(
		tcpassembly.NewStreamPool(
			&bidirectionalStreamFactory{
				conns:                     make(map[string]*bidirectionalStream),
				requestAndResponseChannel: s.requestAndResponseChannel,
			},
		),
	)
	ticker := time.Tick(time.Minute)
	packetsChannel := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	for {
		select {
		case packet := <-packetsChannel:
			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil {
				continue
			}
			tcp, ok := packet.TransportLayer().(*layers.TCP)
			if !ok {
				continue
			}
			net, ok := packet.NetworkLayer().(*layers.IPv4)
			if !ok {
				continue
			}
			src := net.SrcIP.String()
			dst := net.DstIP.String()
			if s.ipManager != nil && !(s.ipManager.isServiceIP(dst) || s.ipManager.isServiceIP(src)) {
				slog.Debug(
					"Ignoring packet not destined for or originating from a service IP:",
					"Src", src,
					"Dst", dst,
					"SrcPort", tcp.SrcPort.String(),
					"DstPort", tcp.DstPort.String(),
				)
				continue
			}
			assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)
		case <-ticker:
			assembler.FlushOlderThan(time.Now().Add(-2 * time.Minute))
		default:
		}
	}
}
