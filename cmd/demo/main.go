// demo launches a policy-server, N home clients, and the fleet dashboard
// in one process for easy local demonstration.
//
//	go run ./cmd/demo                      # defaults (50 homes, predictive, :8080)
//	go run ./cmd/demo --homes 20 --strategy greedy --dash-port 8080
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/amoghmanral/electrotech/internal/dashboard"
	"github.com/amoghmanral/electrotech/internal/home"
	"github.com/amoghmanral/electrotech/internal/policyserver"
)

func main() {
	nHomes := flag.Int("homes", 1000, "number of home clients")
	strategy := flag.String("strategy", "predictive", "greedy | reactive | predictive")
	serverPort := flag.Int("server-port", 9001, "policy gRPC port")
	dashPort := flag.Int("dash-port", 8080, "dashboard HTTP port")
	speed := flag.Float64("speed", 1.0, "initial tick-speed multiplier")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ---- policy server ----
	svc, err := policyserver.NewService(*strategy)
	if err != nil {
		log.Fatal(err)
	}
	grpcSrv := policyserver.NewGRPCServer(svc)
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *serverPort))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	go func() {
		log.Printf("[policy-server] strategy=%s listening on %s", svc.Strategy(), lis.Addr())
		if err := grpcSrv.Serve(lis); err != nil {
			log.Printf("[policy-server] serve: %v", err)
		}
	}()

	// Wait briefly for the server to bind, then create a shared client conn.
	time.Sleep(100 * time.Millisecond)
	conn, err := grpc.NewClient(
		fmt.Sprintf("127.0.0.1:%d", *serverPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	// ---- homes ----
	homes := make([]*home.Home, *nHomes)
	for i := 0; i < *nHomes; i++ {
		id := fmt.Sprintf("h-%02d", i+1)
		seed := int64(1000 + i*7919)
		homes[i] = home.New(id, seed, conn, nil)
	}
	fleet := home.NewFleet(homes)
	fleet.WireRPCSink()
	fleet.SetSpeed(*speed)
	go fleet.Run(ctx)

	// ---- dashboard ----
	dash := dashboard.New(fleet, func() any { return svc.Stats().Snapshot() }, svc.SetStrategy)
	dash.StartBroadcast(ctx, 250*time.Millisecond)
	go func() {
		addr := fmt.Sprintf(":%d", *dashPort)
		log.Printf("[dashboard] http://localhost%s", addr)
		if err := http.ListenAndServe(addr, dash.Router()); err != nil {
			log.Printf("[dashboard] %v", err)
		}
	}()

	fmt.Println()
	fmt.Println("──────────────────────────────────────────────")
	fmt.Printf("  ElectroTech demo\n")
	fmt.Printf("  Policy server: :%d  (%s)\n", *serverPort, *strategy)
	fmt.Printf("  Home clients:  %d\n", *nHomes)
	fmt.Printf("  Dashboard:     http://localhost:%d\n", *dashPort)
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println("  Ctrl-C to stop.")
	fmt.Println()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
	grpcSrv.GracefulStop()
	time.Sleep(100 * time.Millisecond)
}
