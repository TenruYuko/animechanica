//go:build windows && nosystray

package server

import (
	"context"
	"embed"
)

func StartServer(webFS embed.FS, embeddedLogo []byte) {
	// Create root context for Windows nosystray (no signal handling needed)
	ctx := context.Background()

	app, flags, selfupdater := startApp(ctx, embeddedLogo)

	startAppLoop(ctx, &webFS, app, flags, selfupdater)
}
