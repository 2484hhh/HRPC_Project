package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type ServerItem struct {
	Addr  string
	start time.Time
}

type HRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}

const (
	defaultPath    = "/_hrpc_/registry"
	defaultTimeout = time.Minute * 5
)

func New(timeout time.Duration) *HRegistry {
	return &HRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultHRegister = New(defaultTimeout)

func (hr *HRegistry) addService(addr string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	s := hr.servers[addr]
	if s == nil {
		hr.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		s.start = time.Now()
	}
}

func (hr *HRegistry) getAllService() []string {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	var result []string
	for addr, service := range hr.servers {
		if hr.timeout == 0 || service.start.Add(hr.timeout).After(time.Now()) {
			result = append(result, addr)
		} else {
			delete(hr.servers, addr)
		}
	}
	sort.Strings(result)
	return result
}

func (hr *HRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("X-HRPC-Servers", strings.Join(hr.getAllService(), ","))
	case "POST":
		addr := req.Header.Get("X-HRPC-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		hr.addService(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (hr *HRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, hr)
	log.Println("rpc registry path: ", registryPath)
}
func HandleHTTP() {
	DefaultHRegister.HandleHTTP(defaultPath)
}

func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry string, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-HRPC-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err: ", err)
		return err
	}
	return nil
}
