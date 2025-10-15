package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/fasthttp/websocket"
)

func isWebSocket(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, provider *Provider) {
	// Parse the remote WebSocket URL from the provider
	remoteURL, err := url.Parse(provider.WsURL)
	if err != nil {
		log.Printf("ERROR: Could not parse WebSocket provider URL for %s: %v", provider.Name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Dial the remote WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(remoteURL.String(), r.Header)
	if err != nil {
		log.Printf("ERROR: Could not connect to upstream WebSocket %s: %v", provider.Name, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer conn.Close()

	// Upgrade the client's connection to a WebSocket
	upgrader := websocket.Upgrader{}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ERROR: Failed to upgrade client connection to WebSocket: %v", err)
		return
	}
	defer clientConn.Close()

	log.Printf("Successfully established WebSocket proxy to %s", provider.Name)

	// Continuously shuttle messages between the client and the server
	errc := make(chan error, 2)

	// Goroutine to copy messages from client to server
	go func() {
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err = conn.WriteMessage(msgType, msg); err != nil {
				errc <- err
				return
			}
		}
	}()

	// Goroutine to copy messages from server to client
	go func() {
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err = clientConn.WriteMessage(msgType, msg); err != nil {
				errc <- err
				return
			}
		}
	}()

	// Wait for the first error from either direction, then close
	<-errc
	log.Println("WebSocket proxy connection closed.")
}

func NewServer(addr string, balancer *Balancer) *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Select the best provider based on current health and latency
		provider := balancer.SelectBestProvider()
		if provider == nil {
			http.Error(w, "No healthy upstream providers are available.", http.StatusServiceUnavailable)
			return
		}

		// Check if the request is for a WebSocket and handle it accordingly.
		if isWebSocket(r) {
			log.Printf("Incoming WebSocket request, selecting provider: %s", provider.Name)
			handleWebSocket(w, r, provider)
			return
		}

		// If it's not a WebSocket, handle it as a standard HTTP request.
		log.Printf("Incoming HTTP request, selecting provider: %s (Latency: %v)", provider.Name, provider.Latency)

		remote, err := url.Parse(provider.HttpURL)
		if err != nil {
			log.Printf("ERROR: Could not parse provider URL for %s: %v", provider.Name, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = r.URL.Path
			req.URL.RawQuery = r.URL.RawQuery
			req.Host = remote.Host
			req.Header = r.Header
		}

		proxy.ServeHTTP(w, r)
	})

	return &http.Server{
		Addr:    addr,
		Handler: handler,
	}
}

