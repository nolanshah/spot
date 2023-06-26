package application

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

func WatchInputDirectory(config Config, processFiles func(config Config) error) error {
	// Set up signal handling to stop the watcher gracefully
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Create a new file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file watcher")
		return err
	}
	defer watcher.Close()

	// Add the input directory to the watcher
	err = watcher.Add(config.ContentPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to watch input directory")
		return err
	}

	// Initial conversion of files
	err = processFiles(config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert initial files to HTML")
		return err
	}

	log.Info().Msg("Watching input directory for changes...")

	// Start watching for file events
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("file watcher closed unexpectedly")
			}

			// Only trigger conversion on file modifications or creations
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Info().Str("file", event.Name).Msg("File change detected")

				err = processFiles(config)
				if err != nil {
					log.Error().Err(err).Msg("Failed to convert files to HTML")
					return err
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return errors.New("file watcher closed unexpectedly")
			}

			log.Error().Err(err).Msg("File watcher error")
		case <-stop:
			log.Info().Msg("Stopping file watcher")
			return nil
		}
	}
}
