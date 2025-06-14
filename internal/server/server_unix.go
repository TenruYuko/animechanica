//go:build (linux || darwin) && !windows

package server

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func StartServer(webFS embed.FS, embeddedLogo []byte) {
	// Create root context that can be cancelled on shutdown signals
	ctx, cancel := context.WithCancel(context.Background())

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info().Msgf("Received signal %v, initiating graceful shutdown...", sig)
		cancel() // Cancel context to stop all background goroutines
		
		// Give some time for graceful shutdown, then force exit
		time.Sleep(5 * time.Second)
		log.Warn().Msg("Force exit after graceful shutdown timeout")
		os.Exit(1)
	}()

	app, flags, selfupdater := startApp(ctx, embeddedLogo)

	startAppLoop(ctx, &webFS, app, flags, selfupdater)
}
