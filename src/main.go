package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"time"

	firetail "github.com/FireTail-io/firetail-go-lib/middlewares/http"
)

func main() {
	devEnabled, _ := strconv.ParseBool(os.Getenv("FIRETAIL_KUBERNETES_SENSOR_DEV_MODE"))
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
