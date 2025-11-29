package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/a2y-d5l/ci/wip/examples/clients/web/backend"
)

//go:embed all:frontend
var filesystem embed.FS

func main() {
	index, err := filesystem.ReadFile("frontend/index.html")
	if err != nil {
		log.Fatalf("Failed to read index.html: %v", err)
	}

	static, err := fs.Sub(filesystem, "frontend")
	if err != nil {
		log.Fatalf("Failed to create frontend sub-filesystem: %v", err)
	}

	backend.Run(index, static)
}
