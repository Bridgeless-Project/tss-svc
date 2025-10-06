package health

import (
	"fmt"
	"sync"

	"github.com/Bridgeless-Project/tss-svc/internal/bridge/chain"
	"golang.org/x/sync/singleflight"
)

type Checkable interface {
	HealthCheck() error
}

type Status struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type Response struct {
	Ok bool `json:"ok"`
	// Map of service/component name to its status
	Statuses map[string]Status `json:"statuses"`
}

type Map struct {
	statuses  map[string]Status
	overallOk bool
	mu        *sync.Mutex
}

func NewMap() *Map {
	return &Map{
		statuses:  make(map[string]Status),
		overallOk: true,
		mu:        &sync.Mutex{},
	}
}

func (s *Map) Set(name string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err == nil {
		s.statuses[name] = Status{Ok: true}
		return
	}

	s.statuses[name] = Status{Ok: false, Error: err.Error()}
	s.overallOk = false
	return
}

func (s *Map) Collect() Response {
	s.mu.Lock()
	defer s.mu.Unlock()

	copied := make(map[string]Status, len(s.statuses))
	for k, v := range s.statuses {
		copied[k] = v
	}

	return Response{
		Ok:       s.overallOk,
		Statuses: copied,
	}
}

type Checker struct {
	coreConnector Checkable
	clientsRepo   chain.Repository
	rg            *singleflight.Group
}

func NewChecker(coreConnector Checkable, clientsRepo chain.Repository) *Checker {
	return &Checker{
		coreConnector: coreConnector,
		clientsRepo:   clientsRepo,
		rg:            &singleflight.Group{},
	}
}

func (h *Checker) Check() Response {
	response, _, _ := h.rg.Do("health_check", func() (interface{}, error) {
		return h.check(), nil
	})

	return response.(Response)
}

func (h *Checker) check() Response {
	var (
		wg          sync.WaitGroup
		statusesMap = NewMap()
		clients     = h.clientsRepo.Clients()
	)

	fmt.Println("Checking health of core connector and", len(clients), "chain clients")

	wg.Add(len(clients) + 1)
	go func() {
		defer wg.Done()
		statusesMap.Set("core_connector", h.coreConnector.HealthCheck())
	}()
	for chainId, cl := range clients {
		go func(id string, client Checkable) {
			defer wg.Done()
			statusesMap.Set(fmt.Sprintf("chain_%s", chainId), client.HealthCheck())
		}(chainId, cl)
	}
	wg.Wait()

	return statusesMap.Collect()
}
