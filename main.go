package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ncarlier/webhookd/pkg/api"
	"github.com/ncarlier/webhookd/pkg/config"
	"github.com/ncarlier/webhookd/pkg/logger"
	"github.com/ncarlier/webhookd/pkg/worker"
	"github.com/mholt/certmagic"

)

func main() {
	flag.Parse()

	if *version {
		printVersion()
		return
	}

	conf := config.Get()

	level := "info"
	if *conf.Debug {
		level = "debug"
	}
	logger.Init(level)

	logger.Debug.Println("Starting webhookd server...")
	
	// certmagic
        certmagic.Agreed = true
        certmagic.Email = "mail@mail.com"
        certmagic.CA = certmagic.LetsEncryptProductionCA


	server := &http.Server{
		Addr:     "443",
		Handler:  api.NewRouter(config.Get()),
		ErrorLog: logger.Error,
	}

	// Start the dispatcher.
	logger.Debug.Printf("Starting the dispatcher (%d workers)...\n", *conf.NbWorkers)
	worker.StartDispatcher(*conf.NbWorkers)

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		logger.Debug.Println("Server is shutting down...")
		api.Shutdown()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Error.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Info.Println("Server is ready to handle requests at", *conf.ListenAddr)
	api.Start()
        if err := certmagic.HTTPS([]string{"www.example.com"}, server.Handler); err != nil {
                logger.Error.Fatalf("Could not listen on %s: %v\n", *conf.ListenAddr, err)
        }


	<-done
	logger.Debug.Println("Server stopped")
}
