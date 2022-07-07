package client

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type HRegistryDiscovery struct {
	*MultiServersDiscovery
	registry   string
	timeout    time.Duration
	lastUpdate time.Time
}

const defaultUpdateTimeout = time.Second * 10

func NewHRegistryDiscovery(registerAddr string, timeout time.Duration) *HRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	t := &HRegistryDiscovery{
		MultiServersDiscovery: NewMultiServersDiscovery(make([]string, 0)),
		registry:              registerAddr,
		timeout:               timeout,
	}
	return t
}

func (h *HRegistryDiscovery) Update(servers []string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.servers = servers
	h.lastUpdate = time.Now()
	return nil
}

func (h *HRegistryDiscovery) Refresh() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.lastUpdate.Add(h.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh servers from registry", h.registry)
	resp, err := http.Get(h.registry)
	if err != nil {
		log.Println("rpc registry refresh err:", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("X-HRPC-Servers"), ",")
	h.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			h.servers = append(h.servers, strings.TrimSpace(server))
		}
	}
	h.lastUpdate = time.Now()
	return nil
}

func (h *HRegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := h.Refresh(); err != nil {
		return "", err
	}
	return h.MultiServersDiscovery.Get(mode)
}

func (h *HRegistryDiscovery) GetAll() ([]string, error) {
	if err := h.Refresh(); err != nil {
		return nil, err
	}
	return h.MultiServersDiscovery.GetAll()
}
