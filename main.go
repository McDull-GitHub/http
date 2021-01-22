package main

import (
	"io"
	"net"
	"net/http"
	"golang.org/x/net/proxy"
)

var (
	socks5Proxy proxy.Dialer
	client      *http.Client
)

func newClient(dialer proxy.Dialer) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: func(net, addr string) (net.Conn, error) {
				return socks5Proxy.Dial(net, addr)
			},
		},
	}
}

func main() {
	var err error
	socks5Proxy, err = proxy.SOCKS5("tcp", ":9909", nil, proxy.Direct)
	if err != nil {
		return
	}
	client = newClient(socks5Proxy)
	httpHandler := http.HandlerFunc(func(c http.ResponseWriter, req *http.Request) {
		if req.Method == "CONNECT" {
			serverConnection, err := socks5Proxy.Dial("tcp", req.Host)
			if err != nil {
				return
			}
			hijacker, ok := c.(http.Hijacker)
			if !ok {
				serverConnection.Close()
				return
			}
			c.WriteHeader(200)
			_, data, err := hijacker.Hijack()
			if err != nil {
				serverConnection.Close()
				return
			}
			go io.Copy(serverConnection, data)
			go io.Copy(data, serverConnection)
		} else {
			req.RequestURI = ""
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			header := c.Header()
			for a, b := range resp.Header {
				header[a] = b
			}
			c.WriteHeader(resp.StatusCode)
			io.Copy(c, resp.Body)
		}
	})
	http.ListenAndServe(":9910", httpHandler)
}
