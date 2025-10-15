package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// Holds the live state of an upstream RPC provider
type Provider struct {
	Name      string
	HttpURL   string
	WsURL     string
	Latency   time.Duration
	IsHealthy bool
}

// Manages the pool of providers
type Balancer struct {
	providers []*Provider
	mu        sync.RWMutex // A mutex to safely handle concurrent reads/writes to the provider list
}

func NewBalancer(configs []ProviderConfig) *Balancer {
	var providers []*Provider
	for _, p := range configs {
		providers = append(providers, &Provider{
			Name:      p.Name,
			HttpURL:   p.HTTPURL,
			WsURL:     p.WsURL,
			IsHealthy: true, // Assume all providers are healthy at startup
		})
	}
	return &Balancer{providers: providers}
}

// Runs a background loop to periodically check provider health.
func (b *Balancer) StartHealthChecks() {
	// Change this to make a real RPC call, like `getHealth`, in the future
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	b.runChecks()

	for range ticker.C {
		b.runChecks()
	}
}

func (b *Balancer) runChecks() {
	log.Println("Running health checks on all providers...")
	for _, p := range b.providers {
		go b.checkProviderHealth(p)
	}
}

func (b *Balancer) checkProviderHealth(p *Provider) {
	start := time.Now()
	// Just tests if the base URL is reachable.
	resp, err := http.Get(p.HttpURL)
	latency := time.Since(start)

	b.mu.Lock() // Lock for writing
	defer b.mu.Unlock()

	if err != nil || (resp != nil && resp.StatusCode >= 400) {
		if p.IsHealthy {
			log.Printf("Provider %s has gone UNHEALTHY (Latency: %v, Error: %v)", p.Name, latency, err)
		}
		p.IsHealthy = false
	} else {
		if !p.IsHealthy {
			log.Printf("Provider %s has become HEALTHY again (Latency: %v)", p.Name, latency)
		}
		p.IsHealthy = true
		p.Latency = latency
	}
	if resp != nil {
		resp.Body.Close()
	}
}

// Finds the currently healthiest provider with the lowest latency.
func (b *Balancer) SelectBestProvider() *Provider {
	b.mu.RLock() // Lock for reading
	defer b.mu.RUnlock()

	var bestProvider *Provider
	// Set initial minimum latency to a very high value
	minLatency := time.Hour

	for _, p := range b.providers {
		if p.IsHealthy && p.Latency < minLatency {
			minLatency = p.Latency
			bestProvider = p
		}
	}

	return bestProvider
}
