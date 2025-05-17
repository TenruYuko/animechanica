//go:build (linux || darwin) && !windows

package server

import (
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

func StartServer(webFS embed.FS, embeddedLogo []byte) {
	// Start the backend
	app, flags, selfupdater := startApp(embeddedLogo)

	// Get the backend host and port from the app config
	backendHost := app.Config.Server.Host
	backendPort := app.Config.Server.Port

	// If the backend host is 0.0.0.0 or empty, we'll use 127.0.0.1 for the frontend to connect to
	connectHost := backendHost
	if backendHost == "0.0.0.0" || backendHost == "" {
		// Try to get the IP address from environment variables for containers/VMs
		if os.Getenv("SERVER_IP") != "" {
			connectHost = os.Getenv("SERVER_IP")
		} else {
			// Default to localhost if no specific IP is provided
			connectHost = "127.0.0.1"
		}
	}

	// Start the frontend server in a goroutine
	go func() {
		// Wait a moment to let the backend initialize first
		time.Sleep(1 * time.Second)
		
		err := StartFrontendServer(connectHost, backendPort)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start frontend server")
		} else {
			log.Info().Msg(fmt.Sprintf("Frontend available at http://0.0.0.0:43210 (connecting to backend at http://%s:%d)", connectHost, backendPort))
		}
	}()

	// Start the backend server loop
	startAppLoop(&webFS, app, flags, selfupdater)
}
