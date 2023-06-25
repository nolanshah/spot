package application

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"
)

func ServeOutputDirectory(outputDir string, addr string, wg *sync.WaitGroup) error {
	// Set up file server to serve the output directory
	fs := http.FileServer(http.Dir(outputDir))
	http.Handle("/", fs)

	server := &http.Server{Addr: addr}

	log.Info().Str("addr", addr).Msg("Serving files...")

	// Create a channel to listen for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a goroutine to handle server shutdown
	go func() {
		<-sigChan // Wait for the signal
		log.Info().Msg("Shutting down HTTP server...")

		// Shutdown the server gracefully with a timeout of 5 seconds
		err := server.Shutdown(nil)
		if err != nil {
			log.Error().Err(err).Msg("Error during server shutdown")
		}

		wg.Done() // Notify wait group that shutdown is complete
	}()

	// Start the HTTP server
	err := server.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to start HTTP server")
		return err
	}

	return nil
}
