package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"golang.org/x/sync/semaphore"
)

type bidirectionalStreamFactory struct {
	conns                     *sync.Map
	requestAndResponseChannel *chan httpRequestAndResponse
	maxBodySize               int64
}

func (f *bidirectionalStreamFactory) New(netFlow, tcpFlow gopacket.Flow) tcpassembly.Stream {
	key := netFlow.FastHash() ^ tcpFlow.FastHash()

	// The second time we see the same connection, it will be from the server to the client
	if conn, ok := f.conns.LoadAndDelete(fmt.Sprint(key)); ok {
		return &conn.(*bidirectionalStream).serverToClient
	}

	s := &bidirectionalStream{
		net:                       netFlow,
		transport:                 tcpFlow,
		clientToServer:            tcpreader.NewReaderStream(),
		serverToClient:            tcpreader.NewReaderStream(),
		requestAndResponseChannel: f.requestAndResponseChannel,
		closeCallback: func() {
			f.conns.Delete(fmt.Sprint(key))
		},
		maxBodySize: f.maxBodySize,
	}
	f.conns.Store(fmt.Sprint(key), s)
	go s.run()

	// The first time we see the connection, it will be from the client to the server
	return &s.clientToServer
}

type bidirectionalStream struct {
	net, transport            gopacket.Flow
	clientToServer            tcpreader.ReaderStream
	serverToClient            tcpreader.ReaderStream
	requestAndResponseChannel *chan httpRequestAndResponse
	closeCallback             func()
	maxBodySize               int64
}

func (s *bidirectionalStream) run() {
	defer s.closeCallback()
	defer s.clientToServer.Close()
	defer s.serverToClient.Close()

	sem := semaphore.NewWeighted(2)

	requestChannel := make(chan *http.Request, 1)
	responseChannel := make(chan *http.Response, 1)

	err := sem.Acquire(context.Background(), 1)
	if err != nil {
		slog.Error("Failed to acquire semaphore for clientToServer reader:", "Err", err.Error())
		return
	}
	go func() {
		defer sem.Release(1)
		defer close(requestChannel)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Recovered from panic in clientToServer reader:", "Err", r)
			}
		}()
		requestBytes := make([]byte, s.maxBodySize)
		bytesRead, err := io.ReadFull(&s.clientToServer, requestBytes)
		if err != nil && err != io.ErrUnexpectedEOF {
			slog.Debug("Failed to read request bytes from stream:", "Err", err.Error(), "BytesRead", bytesRead)
			return
		}
		request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(requestBytes[:bytesRead])))
		if err != nil {
			slog.Debug("Failed to read request bytes:", "Err", err.Error())
			return
		}
		// RemoteAddr is not filled in by ReadRequest so we have to populate it ourselves
		request.RemoteAddr = fmt.Sprintf("%s:%s", s.net.Src().String(), s.transport.Src().String())
		requestChannel <- request
	}()

	err = sem.Acquire(context.Background(), 1)
	if err != nil {
		slog.Error("Failed to acquire semaphore for serverToClient reader:", "Err", err.Error())
		return
	}
	go func() {
		defer sem.Release(1)
		defer close(responseChannel)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Recovered from panic in serverToClient reader:", "Err", r)
			}
		}()
		responseBytes := make([]byte, s.maxBodySize)
		bytesRead, err := io.ReadFull(&s.serverToClient, responseBytes)
		if err != nil && err != io.ErrUnexpectedEOF {
			slog.Debug("Failed to read response bytes from stream:", "Err", err.Error(), "BytesRead", bytesRead)
			return
		}
		response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(responseBytes[:bytesRead])), nil)
		if err != nil {
			slog.Debug("Failed to read response bytes:", "Err", err.Error())
			return
		}
		responseChannel <- response
	}()

	// Wait for both goroutines to finish with timeout of 2 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := sem.Acquire(ctx, 2); err != nil {
		if err != context.DeadlineExceeded {
			slog.Error("Failed to acquire semaphore for both readers:", "Err", err.Error())
		}
		return
	}

	var capturedRequest *http.Request
	var capturedResponse *http.Response

	select {
	case capturedRequest = <-requestChannel:
	default:
	}

	select {
	case capturedResponse = <-responseChannel:
	default:
	}

	if capturedRequest == nil && capturedResponse == nil {
		slog.Debug(
			"No request or response captured from stream",
			"Src", s.net.Src().String(),
			"Dst", s.net.Dst().String(),
			"SrcPort", s.transport.Src().String(),
			"DstPort", s.transport.Dst().String(),
		)
	} else if capturedRequest == nil {
		slog.Warn(
			"Captured response but no request from stream",
			"Src", s.net.Src().String(),
			"Dst", s.net.Dst().String(),
			"SrcPort", s.transport.Src().String(),
			"DstPort", s.transport.Dst().String(),
		)
	} else if capturedResponse == nil {
		slog.Warn(
			"Captured request but no response from stream",
			"Src", s.net.Src().String(),
			"Dst", s.net.Dst().String(),
			"SrcPort", s.transport.Src().String(),
			"DstPort", s.transport.Dst().String(),
		)
	}

	if capturedRequest == nil || capturedResponse == nil {
		return
	}

	*s.requestAndResponseChannel <- httpRequestAndResponse{
		request:  capturedRequest,
		response: capturedResponse,
		src:      s.net.Src().String(),
		dst:      s.net.Dst().String(),
		srcPort:  s.transport.Src().String(),
		dstPort:  s.transport.Dst().String(),
	}
}
