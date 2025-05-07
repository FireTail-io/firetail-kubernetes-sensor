package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
)

type httpRequestAndResponse struct {
	request  *http.Request
	response *http.Response
}

type httpRequestAndResponseStreamer struct {
	bpfExpression             string
	requestAndResponseChannel *chan httpRequestAndResponse
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
			assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)
		case <-ticker:
			assembler.FlushOlderThan(time.Now().Add(-2 * time.Minute))
		default:
		}
	}
}

type bidirectionalStreamFactory struct {
	conns                     map[string]*bidirectionalStream
	requestAndResponseChannel *chan httpRequestAndResponse
}

func (f *bidirectionalStreamFactory) New(netFlow, tcpFlow gopacket.Flow) tcpassembly.Stream {
	key := netFlow.FastHash() ^ tcpFlow.FastHash()

	// The second time we see the same connection, it will be from the server to the client
	if conn, ok := f.conns[fmt.Sprint(key)]; ok {
		return &conn.serverToClient
	}

	s := &bidirectionalStream{
		net:                       netFlow,
		transport:                 tcpFlow,
		clientToServer:            tcpreader.NewReaderStream(),
		serverToClient:            tcpreader.NewReaderStream(),
		requestAndResponseChannel: f.requestAndResponseChannel,
	}
	f.conns[fmt.Sprint(key)] = s
	go s.run()

	// The first time we see the connection, it will be from the client to the server
	return &s.clientToServer
}

type bidirectionalStream struct {
	net, transport            gopacket.Flow
	clientToServer            tcpreader.ReaderStream
	serverToClient            tcpreader.ReaderStream
	requestAndResponseChannel *chan httpRequestAndResponse
}

func (s *bidirectionalStream) run() {
	wg := sync.WaitGroup{}
	wg.Add(2)

	requestChannel := make(chan *http.Request, 1)
	responseChannel := make(chan *http.Response, 1)

	go func() {
		reader := bufio.NewReader(&s.clientToServer)
		for {
			request, err := http.ReadRequest(reader)
			if err == io.EOF {
				wg.Done()
				return
			} else if err != nil {
				continue
			}
			responseBody := make([]byte, request.ContentLength)
			if request.ContentLength > 0 {
				io.ReadFull(request.Body, responseBody)
			}
			request.Body.Close()
			request.Body = io.NopCloser(bytes.NewReader(responseBody))
			requestChannel <- request

		}
	}()

	go func() {
		reader := bufio.NewReader(&s.serverToClient)
		for {
			response, err := http.ReadResponse(reader, nil)
			if err == io.ErrUnexpectedEOF {
				wg.Done()
				return
			} else if err != nil {
				continue
			}
			responseBody := make([]byte, response.ContentLength)
			if response.ContentLength > 0 {
				io.ReadFull(response.Body, responseBody)
			}
			response.Body.Close()
			response.Body = io.NopCloser(bytes.NewReader(responseBody))
			responseChannel <- response
		}
	}()

	wg.Wait()

	capturedRequest := <-requestChannel
	capturedResponse := <-responseChannel
	close(requestChannel)
	close(responseChannel)

	*s.requestAndResponseChannel <- httpRequestAndResponse{
		request:  capturedRequest,
		response: capturedResponse,
	}
}

func main() {
	log.Println("🔍 Starting local HTTP server...")
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
		})
		log.Fatal(http.ListenAndServe(":80", nil))
	}()

	log.Println("🔍 Starting HTTP request streamer...")
	requestAndResponseChannel := make(chan httpRequestAndResponse, 1)
	httpRequestStreamer := &httpRequestAndResponseStreamer{
		bpfExpression:             "tcp and (port 80 or port 443)",
		requestAndResponseChannel: &requestAndResponseChannel,
	}
	go httpRequestStreamer.start()

	log.Println("🔍 Starting HTTP request & response logger...")
	for {
		select {
		case requestAndResponse := <-requestAndResponseChannel:
			capturedRequestBody, err := io.ReadAll(requestAndResponse.request.Body)
			if err != nil {
				log.Println("Error reading request body:", err.Error())
				return
			}
			capturedResponseBody, err := io.ReadAll(requestAndResponse.response.Body)
			if err != nil {
				log.Println("Error reading request body:", err.Error())
				return
			}
			log.Println(
				"📡 Captured HTTP request & response:",
				"\n\tRequest:", requestAndResponse.request.Method, requestAndResponse.request.URL,
				"\n\tResponse:", requestAndResponse.response.Status,
				"\n\tHost:", requestAndResponse.request.Host,
				"\n\tRequest Body:", string(capturedRequestBody),
				"\n\tResponse Body:", string(capturedResponseBody),
				"\n\tRequest Headers:", requestAndResponse.request.Header,
				"\n\tResponse Headers:", requestAndResponse.response.Header,
			)
		default:
		}
	}
}
