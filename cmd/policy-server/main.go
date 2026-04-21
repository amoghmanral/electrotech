// policy-server is the standalone gRPC service that home clients call
// once per tick for strategic dispatch decisions.
//
//	policy-server --port 9001 --strategy predictive
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/amoghmanral/electrotech/internal/policyserver"
)

func main() {
	port := flag.Int("port", 9001, "gRPC listen port")
	httpPort := flag.Int("http-port", 0, "(optional) HTTP /stats port; 0 disables")
	strategy := flag.String("strategy", "predictive", "greedy | reactive | predictive")
	flag.Parse()

	svc, err := policyserver.NewService(*strategy)
	if err != nil {
		log.Fatal(err)
	}
	g := policyserver.NewGRPCServer(svc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	if *httpPort > 0 {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(svc.Stats().Snapshot())
			})
			addr := fmt.Sprintf(":%d", *httpPort)
			log.Printf("[policy-server] stats http at %s/stats", addr)
			_ = http.ListenAndServe(addr, mux)
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("[policy-server] shutting down")
		g.GracefulStop()
	}()

	log.Printf("[policy-server] strategy=%s listening on %s", svc.Strategy(), lis.Addr())
	if err := g.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
