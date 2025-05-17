package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// StartFrontendServer starts the Next.js frontend server on port 43210
// It configures the environment to connect to the backend at the specified host and port
func StartFrontendServer(backendHost string, backendPort int) error {
	// Determine the frontend directory path (relative to the binary location)
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)
	frontendDir := filepath.Join(exeDir, "seanime-web")

	// Check if frontend directory exists
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		// If the directory doesn't exist, try going up one level (for development environments)
		frontendDir = filepath.Join(exeDir, "..", "seanime-web")
		if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
			return fmt.Errorf("frontend directory not found at %s or its parent directory", filepath.Join(exeDir, "seanime-web"))
		}
	}

	// Set up the command to start the Next.js server
	var cmd *exec.Cmd
	backendURL := fmt.Sprintf("http://%s:%d", backendHost, backendPort)

	// Set up the environment for the frontend
	env := os.Environ()
	env = append(env, "NEXT_PUBLIC_PLATFORM=mirror")
	env = append(env, fmt.Sprintf("NEXT_PUBLIC_BACKEND_URL=%s", backendURL))

	// Explicitly set the host for internal connection
	// This is critical for video streaming to work properly
	env = append(env, fmt.Sprintf("NEXT_PUBLIC_BACKEND_HOST=%s", backendHost))
	env = append(env, fmt.Sprintf("NEXT_PUBLIC_BACKEND_PORT=%d", backendPort))

	// Using npx to ensure we use the locally installed next
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "npx", "next", "dev", "--hostname=0.0.0.0", "--port=43210", "--turbo")
	} else {
		cmd = exec.Command("npx", "next", "dev", "--hostname=0.0.0.0", "--port=43210", "--turbo")
	}

	cmd.Dir = frontendDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the frontend server
	log.Info().Msg(fmt.Sprintf("Starting frontend server on port 43210, connecting to backend at %s", backendURL))
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start frontend server: %w", err)
	}

	// Check if npm is installed
	npmCmd := exec.Command("npm", "--version")
	npmOutput, npmErr := npmCmd.CombinedOutput()

	if npmErr != nil {
		log.Warn().Msg("npm is not installed or not found in PATH. Frontend server might not start properly.")
	} else {
		log.Info().Msg(fmt.Sprintf("Found npm version: %s", strings.TrimSpace(string(npmOutput))))

		// Install dependencies if needed
		log.Info().Msg("Installing frontend dependencies...")
		installCmd := exec.Command("npm", "install", "--no-fund", "--no-audit")
		installCmd.Dir = frontendDir
		installOutput, installErr := installCmd.CombinedOutput()

		if installErr != nil {
			log.Warn().Msg(fmt.Sprintf("Error installing dependencies: %v\n%s", installErr, string(installOutput)))
		} else {
			log.Info().Msg("Frontend dependencies installed successfully")
		}
	}

	// Give the frontend server some time to start
	log.Info().Msg("Waiting for frontend server to start...")
	time.Sleep(2 * time.Second)

	return nil
}
