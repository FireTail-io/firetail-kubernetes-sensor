package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"time"

	firetail "github.com/FireTail-io/firetail-go-lib/middlewares/http"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
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
			// RemoteAddr is not filled in by ReadRequest so we have to populate it ourselves
			request.RemoteAddr = fmt.Sprintf("%s:%s", s.net.Src().String(), s.transport.Src().String())
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
		src:      s.net.Src().String(),
		dst:      s.net.Dst().String(),
		srcPort:  s.transport.Src().String(),
		dstPort:  s.transport.Dst().String(),
	}
}

func main() {
	devEnabled, err := strconv.ParseBool(os.Getenv("FIRETAIL_KUBERNETES_SENSOR_DEV_MODE"))
	if err != nil {
		devEnabled = false
	}

	if devEnabled {
		slog.Warn("🧰 Development mode enabled, setting log level to debug...")
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	logsApiToken, logsApiTokenSet := os.LookupEnv("FIRETAIL_API_TOKEN")
	if !logsApiTokenSet {
		log.Fatal("FIRETAIL_API_TOKEN environment variable not set")
	}

	var ipManager *serviceIpManager
	if disableServiceIpFilter, err := strconv.ParseBool(os.Getenv("DISABLE_SERVICE_IP_FILTERING")); !(err == nil && disableServiceIpFilter) {
		slog.Info(
			"Service IP filter enabled, monitoring service IPs...",
		)
		ipManager = newServiceIpManager()
	}

	bpfExpression, bpfExpressionSet := os.LookupEnv("BPF_EXPRESSION")
	if !bpfExpressionSet {
		slog.Info(
			"BPF_EXPRESSION environment variable not set, using default: tcp and (port 80 or port 443). See docs for " +
				"further info.",
		)
		bpfExpression = "tcp and (port 80 or port 443)"
	}

	devServerEnabled, err := strconv.ParseBool(os.Getenv("FIRETAIL_KUBERNETES_SENSOR_DEV_SERVER_ENABLED"))
	if err == nil && devServerEnabled {
		slog.Warn("🧰 Development server enabled, starting example HTTP server...")
		go func() {
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
			})
			log.Fatal(http.ListenAndServe(":80", nil))
		}()
	}

	requestAndResponseChannel := make(chan httpRequestAndResponse, 1)
	httpRequestStreamer := &httpRequestAndResponseStreamer{
		bpfExpression:             bpfExpression,
		requestAndResponseChannel: &requestAndResponseChannel,
		ipManager:                 ipManager,
	}
	go httpRequestStreamer.start()

	var maxLogAge time.Duration = 0
	if devEnabled {
		slog.Warn("🧰 Development mode enabled, setting max age of logs held by Firetail middleware to 1 second...")
		maxLogAge = time.Second
	}
	firetailMiddleware, err := firetail.GetMiddleware(
		&firetail.Options{
			LogsApiToken: logsApiToken,
			LogsApiUrl:   os.Getenv("FIRETAIL_API_URL"),
			MaxLogAge:    maxLogAge,
		},
	)
	if err != nil {
		log.Fatal("Failed to initialise Firetail middleware:", err.Error())
	}

	for {
		select {
		case requestAndResponse := <-requestAndResponseChannel:
			if !(ipManager == nil || ipManager.isServiceIP(requestAndResponse.dst)) {
				slog.Debug(
					"Ignoring request to non-service IP:",
					"Src", requestAndResponse.src,
					"Dst", requestAndResponse.dst,
					"SrcPort", requestAndResponse.srcPort,
					"DstPort", requestAndResponse.dstPort,
				)
				continue
			}
			slog.Debug(
				"Captured request and response:",
				"Method", requestAndResponse.request.Method,
				"URL", requestAndResponse.request.URL,
				"StatusCode", requestAndResponse.response.StatusCode,
				"Src", requestAndResponse.src,
				"Dst", requestAndResponse.dst,
				"SrcPort", requestAndResponse.srcPort,
				"DstPort", requestAndResponse.dstPort,
			)
			firetailMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(requestAndResponse.response.StatusCode)
				for key, values := range requestAndResponse.response.Header {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				capturedResponseBody, err := io.ReadAll(requestAndResponse.response.Body)
				if err != nil {
					slog.Error("Error reading request body:", "err", err.Error())
					return
				}
				w.Write(capturedResponseBody)
			})).ServeHTTP(
				httptest.NewRecorder(),
				requestAndResponse.request,
			)
		default:
		}
	}
}
