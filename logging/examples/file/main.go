//go:build example

package main

import (
	"os"
	"path/filepath"

	"github.com/jasoet/pkg/v2/logging"
	"github.com/rs/zerolog/log"
)

func main() {
	// Create temp directory for logs
	tempDir, err := os.MkdirTemp("", "logging-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "app.log")

	// Initialize with file output only
	logging.InitializeWithFile("file-example", false,
		logging.OutputFile,
		&logging.FileConfig{
			Path: logFile,
		})

	// Log messages (appear in file only, not console)
	log.Info().Msg("Application started")
	log.Info().
		Str("user_id", "67890").
		Str("action", "login").
		Msg("User action")

	log.Warn().Str("resource", "cache").Msg("Resource unavailable")
	log.Error().Str("operation", "save").Msg("Operation failed")

	// Read and display the log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		panic(err)
	}

	println("\n=== Log File Content ===")
	println(string(content))
	println("\n=== End of Log File ===")
	println("\nLog file location:", logFile)
	println("(File will be deleted after example exits)")
}
