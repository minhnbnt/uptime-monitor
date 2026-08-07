package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type HealthResponse struct {
	Status    string `json:"status"`
	From      string `json:"from"`
	Timestamp string `json:"timestamp"`
}

var serverMap = sync.Map{}

func runWebServer(ctx context.Context, port int) {

	addr := fmt.Sprintf(":%d", port)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {

		value, has := serverMap.Load(port)
		now := time.Now()
		if has {
			elapsed := now.Sub(value.(time.Time))
			fmt.Println(addr, "called after", elapsed)
		} else {
			fmt.Println(addr, "called first time")
		}

		serverMap.Store(port, now)

		w.Header().Set("Content-Type", "application/json")

		response := HealthResponse{
			Status:    "OK",
			From:      addr,
			Timestamp: time.Now().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("failed to encode json: %s", err.Error())
		}
	})

	server := http.Server{Addr: addr, Handler: mux}

	go func() {

		<-ctx.Done()
		if err := server.Shutdown(ctx); err != nil {
			log.Panic(err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Panic(err)
	}
}

func main() {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	waitgroup := sync.WaitGroup{}

	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		waitgroup.Wait()
		ticker.Stop()
	}()

	for port := 10000; port < 20000; port++ {
		waitgroup.Go(func() { runWebServer(ctx, port) })
	}
}
