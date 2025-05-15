package main

import (
	"log"
	"log/slog"
	"net/http"
	"sync"
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
	maxBodySize               int64
}

func (s *httpRequestAndResponseStreamer) getHandleAndPacketsChannel() (*pcap.Handle, <-chan gopacket.Packet) {
	handle, err := pcap.OpenLive("any", 1600, true, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	err = handle.SetBPFFilter(s.bpfExpression)
	if err != nil {
		log.Fatal(err)
	}
	packetsChannel := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	return handle, packetsChannel
}

func (s *httpRequestAndResponseStreamer) start() {
	assembler := tcpassembly.NewAssembler(
		tcpassembly.NewStreamPool(
			&bidirectionalStreamFactory{
				conns:                     &sync.Map{},
				requestAndResponseChannel: s.requestAndResponseChannel,
				maxBodySize:               s.maxBodySize,
			},
		),
	)

	handler, packetsChannel := s.getHandleAndPacketsChannel()

	ticker := time.Tick(time.Minute)
	for {
		select {
		case packet, ok := <-packetsChannel:
			if !ok {
				slog.Warn("Packet channel closed. Reinitializing...")
				handler.Close()
				handler, packetsChannel = s.getHandleAndPacketsChannel()
			}
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
			slog.Debug(
				"Captured packet:",
				"Src", src,
				"Dst", dst,
				"SrcPort", tcp.SrcPort.String(),
				"DstPort", tcp.DstPort.String(),
			)
			assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)
		case <-ticker:
			assembler.FlushOlderThan(time.Now().Add(-2 * time.Minute))
		default:
		}
	}
}
