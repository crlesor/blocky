package resolver

import (
	"blocky/config"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

type metricsMock struct {
	mock.Mock
}

func (m *metricsMock) RecordStats(request *Request, response *Response) {
	m.Called(request, response)
}

func (m *metricsMock) Start() {
	m.Called()
}

type resolverMock struct {
	mock.Mock
	NextResolver
}

func (r *resolverMock) Configuration() (result []string) {
	return
}

func (r *resolverMock) Resolve(req *Request) (*Response, error) {
	args := r.Called(req)
	resp, ok := args.Get(0).(*Response)

	if ok {
		return resp, args.Error((1))
	}

	return nil, args.Error(1)
}

func TestDOHUpstream(fn func(request *dns.Msg) (response *dns.Msg),
	reqFn ...func(w http.ResponseWriter)) config.Upstream {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("here")

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal("can't read request: ", err)
		}

		msg := new(dns.Msg)
		err = msg.Unpack(body)
		if err != nil {
			log.Fatal("can't deserialize message: ", err)
		}
		response := fn(msg)
		response.SetReply(msg)

		b, err := response.Pack()
		if err != nil {
			log.Fatal("can't serialize message: ", err)
		}
		w.Header().Set("content-type", "application/dns-message")

		for _, f := range reqFn {
			f(w)
		}
		_, err = w.Write(b)
		if err != nil {
			log.Fatal("can't write response: ", err)
		}
	}))
	upstream, err := config.ParseUpstream(server.URL)

	if err != nil {
		log.Fatal("can't resolve address: ", err)
	}

	return upstream
}

func TestUDPUpstream(fn func(request *dns.Msg) (response *dns.Msg)) config.Upstream {
	a, err := net.ResolveUDPAddr("udp4", ":0")
	if err != nil {
		log.Fatal("can't resolve address: ", err)
	}

	ln, err := net.ListenUDP("udp4", a)
	if err != nil {
		log.Fatal("can't create connection: ", err)
	}

	ladr := ln.LocalAddr().String()
	host := strings.Split(ladr, ":")[0]
	p, err := strconv.Atoi(strings.Split(ladr, ":")[1])

	if err != nil {
		log.Fatal("can't convert port: ", err)
	}

	port := uint16(p)

	go func() {
		for {
			buffer := make([]byte, 1024)
			n, addr, err := ln.ReadFromUDP(buffer)

			if err != nil {
				log.Fatal("error on reading from udp: ", err)
			}

			msg := new(dns.Msg)
			err = msg.Unpack(buffer[0 : n-1])

			if err != nil {
				log.Fatal("can't deserialize message: ", err)
			}

			response := fn(msg)
			response.SetReply(msg)

			b, err := response.Pack()
			if err != nil {
				log.Fatal("can't serialize message: ", err)
			}

			_, err = ln.WriteToUDP(b, addr)
			if err != nil {
				log.Fatal("can't write to UDP: ", err)
			}
		}
	}()

	return config.Upstream{Net: "udp", Host: host, Port: port}
}
