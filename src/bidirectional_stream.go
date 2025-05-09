package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
)

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
