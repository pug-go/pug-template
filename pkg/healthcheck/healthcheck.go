package healthcheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

const (
	CheckResultSuccess = "OK"

	CheckHandlerPathLiveness  = "/live"
	CheckHandlerPathReadiness = "/ready"
)

type Handler interface {
	// AddLivenessCheck adds a check that indicates that this instance of the
	// application should be destroyed or restarted. A failed liveness check
	// indicates that this instance is unhealthy, not some upstream dependency.
	// Every liveness check is also included as a readiness check.
	AddLivenessCheck(name string, check Check)

	// AddReadinessCheck adds a check that indicates that this instance of the
	// application is currently unable to serve requests because of an upstream
	// or some transient failure. If a readiness check fails, this instance
	// should no longer receiver requests, but should not be restarted or
	// destroyed.
	AddReadinessCheck(name string, check Check)

	// LiveEndpointHandlerFunc is the HTTP handler for just the /live endpoint, which is
	// useful if you need to attach it into your own HTTP handler tree.
	LiveEndpointHandlerFunc(http.ResponseWriter, *http.Request)

	// ReadyEndpointHandlerFunc is the HTTP handler for just the /ready endpoint, which is
	// useful if you need to attach it into your own HTTP handler tree.
	ReadyEndpointHandlerFunc(http.ResponseWriter, *http.Request)
}

// Check is a health/readiness check.
type Check func() error

// NewHandler creates a new basic Handler
func NewHandler() Handler {
	return &basicHandler{
		livenessChecks:  make(map[string]Check),
		readinessChecks: make(map[string]Check),
	}
}

// basicHandler is a basic Handler implementation.
type basicHandler struct {
	checksMutex     sync.RWMutex
	livenessChecks  map[string]Check
	readinessChecks map[string]Check
}

func (s *basicHandler) AddLivenessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.livenessChecks[name] = check
}

func (s *basicHandler) AddReadinessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.readinessChecks[name] = check
}

func (s *basicHandler) LiveEndpointHandlerFunc(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.livenessChecks)
}

func (s *basicHandler) ReadyEndpointHandlerFunc(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.readinessChecks, s.livenessChecks)
}

type result struct {
	name   string
	result string
}

func (s *basicHandler) handle(w http.ResponseWriter, r *http.Request, checkGroups ...map[string]Check) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checkResults := make(map[string]string)
	status := http.StatusOK
	for _, checkGroup := range checkGroups {
		if st := s.collectChecks(checkGroup, checkResults); st != http.StatusOK {
			status = st
		}
	}

	// write out the response code and content type header
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	w.WriteHeader(status)

	// unless ?full=1, return an empty body. Kubernetes only cares about the
	// HTTP status code, so we won't waste bytes on the full body.
	if r.URL.Query().Get("full") != "1" {
		_, _ = w.Write([]byte("{}\n"))
		return
	}

	// otherwise, write the JSON body ignoring any encoding errors (which
	// shouldn't really be possible since we're encoding a map[string]string).
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	_ = encoder.Encode(checkResults)
}

func (s *basicHandler) collectChecks(checks map[string]Check, resultsOut map[string]string) (status int) {
	s.checksMutex.RLock()
	defer s.checksMutex.RUnlock()

	status = http.StatusOK

	checkersQty := len(checks)
	if checkersQty == 0 {
		return
	}

	results := make(chan result, checkersQty)

	// run checks functions and send it to conveyor
	for name, check := range checks {
		go func(name string, check Check) {
			defer func() {
				if r := recover(); r != nil {
					results <- result{
						name:   name,
						result: fmt.Sprintf("checker panic recovered: %v", r),
					}
				}
			}()

			var val = CheckResultSuccess
			if err := check(); err != nil {
				val = err.Error()
			}

			results <- result{
				name:   name,
				result: val,
			}
		}(name, check)
	}

	// handle results like conveyor
	for res := range results {
		resultsOut[res.name] = res.result

		if res.result != CheckResultSuccess {
			status = http.StatusServiceUnavailable
		}

		checkersQty--
		if checkersQty == 0 {
			close(results)
		}
	}

	return status
}
