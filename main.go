package main

import (
	"embed"
	"seanime/internal/server"
)

//go:embed seanime-web/out/*
var WebFS embed.FS

//go:embed internal/icon/logo.png
var embeddedLogo []byte

func main() {
	server.StartServer(WebFS, embeddedLogo)
}
