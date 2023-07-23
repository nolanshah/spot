package application

import (
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

func WatchInputDirectory(config Config) error {
	// Set up signal handling to stop the watcher gracefully
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Create a new file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file watcher.")
		return err
	}
	defer watcher.Close()

	// Add the input directory to the watcher
	watcherDirs := []string{config.ContentPath, config.StaticPath, config.TemplatesPath}
	for _, apexDir := range watcherDirs {
		err := filepath.Walk(apexDir, func(watcherDir string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			err = watcher.Add(watcherDir)
			if err != nil {
				log.Error().Str("dir", watcherDir).Err(err).Msg("Failed to add dir to watcher.")
				return err
			}
			return nil
		})
		if err != nil {
			log.Error().Str("apexDir", apexDir).Err(err).Msg("Failed to walk dirs underneath apexDir.")
			return err
		}
	}

	// Initial conversion of files
	err = ProcessFiles(config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process.")
	}

	log.Info().Msg("Watching input directory for changes.")

	// Start watching for file events
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("file watcher closed unexpectedly")
			}

			// Only trigger conversion on file modifications or creations
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Info().Str("changedFile", event.Name).Msg("File change detected, reloading.")

				err = ProcessFiles(config)
				if err != nil {
					log.Error().Err(err).Msg("Failed to refresh.")
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return errors.New("file watcher closed unexpectedly")
			}

			log.Error().Err(err).Msg("File watcher error.")
		case <-stop:
			log.Info().Msg("Stopping file watcher.")
			return nil
		}
	}
}
