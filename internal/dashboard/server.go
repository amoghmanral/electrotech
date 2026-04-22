// Package dashboard serves the fleet web UI and JSON API.
package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/amoghmanral/electrotech/internal/home"
)

//go:embed static
var staticFS embed.FS

// ServerStatsProvider lets us plug in a callback that returns a JSON-
// serializable snapshot of the policy-server's own metrics.
type ServerStatsProvider func() any

// Called when dashboard requests a strategy change
type PolicyChanger func(strategy string) error

type Server struct {
	fleet         *home.Fleet
	statsFn       ServerStatsProvider
	policyChanger PolicyChanger

	upgrader websocket.Upgrader
	subs     map[chan []byte]struct{}
	subsMu   sync.Mutex
}

func New(f *home.Fleet, statsFn ServerStatsProvider, policyChanger PolicyChanger) *Server {
	return &Server{
		fleet:         f,
		statsFn:       statsFn,
		policyChanger: policyChanger,
		upgrader:      websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		subs:          make(map[chan []byte]struct{}),
	}
}

type combinedState struct {
	Fleet       home.FleetSnapshot `json:"fleet"`
	ServerStats any                `json:"server"`
}

func (s *Server) snapshot() combinedState {
	snap := s.fleet.Snapshot()
	var stats any
	if s.statsFn != nil {
		stats = s.statsFn()
	}
	return combinedState{Fleet: snap, ServerStats: stats}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/ws", s.handleWS)
	mux.HandleFunc("/api/speed", s.handleSpeed)
	mux.HandleFunc("/api/pause", s.handlePause)
	mux.HandleFunc("/api/resume", s.handleResume)
	mux.HandleFunc("/api/policy", s.handlePolicy)
	return mux
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.snapshot())
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	ch := make(chan []byte, 8)
	s.subsMu.Lock()
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()
	defer func() {
		s.subsMu.Lock()
		delete(s.subs, ch)
		s.subsMu.Unlock()
	}()
	// initial snapshot
	msg, _ := json.Marshal(s.snapshot())
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return
	}
	for m := range ch {
		if err := conn.WriteMessage(websocket.TextMessage, m); err != nil {
			return
		}
	}
}

// StartBroadcast starts a goroutine that snapshots + pushes state at the
// given interval to all WS subscribers.
func (s *Server) StartBroadcast(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				msg, err := json.Marshal(s.snapshot())
				if err != nil {
					continue
				}
				s.subsMu.Lock()
				for ch := range s.subs {
					select {
					case ch <- msg:
					default:
					}
				}
				s.subsMu.Unlock()
			}
		}
	}()
}

func (s *Server) handleSpeed(w http.ResponseWriter, r *http.Request) {
	x, err := strconv.ParseFloat(r.URL.Query().Get("x"), 64)
	if err != nil || x <= 0 {
		http.Error(w, "need ?x=<float>", 400)
		return
	}
	s.fleet.SetSpeed(x)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
	s.fleet.SetRunning(false)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	s.fleet.SetRunning(true)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePolicy(w http.ResponseWriter, r *http.Request) {
	strategy := r.URL.Query().Get("strategy")
	s.policyChanger(strategy)
	w.WriteHeader(http.StatusNoContent)
}
