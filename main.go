package main

import (
	"os"

	"github.com/itamaker/ragcheck/internal/app"
)

// version is set at release time (GoReleaser injects main.version).
var version = "dev"

func main() {
	app.Version = version
	os.Exit(app.Run(os.Args[1:]))
}
