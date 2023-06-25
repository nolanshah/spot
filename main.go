package main

import (
	"encoding/binary"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func serveOutputDirectory(outputDir string, addr string, wg *sync.WaitGroup) error {
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

func watchInputDirectory(inputDir, outputDir string) error {
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
	err = watcher.Add(inputDir)
	if err != nil {
		log.Error().Err(err).Msg("Failed to watch input directory")
		return err
	}

	// Initial conversion of files
	err = processFiles(inputDir, outputDir)
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

				err = processFiles(inputDir, outputDir)
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

func processFiles(inputDir, outputDir string) error {
	err := filepath.Walk(inputDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Error accessing file")
			return err
		}

		if info.IsDir() {
			// Skip directories
			return nil
		}

		// Get the relative path of the input file
		relativePath, err := filepath.Rel(inputDir, filePath)
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to get relative path")
			return err
		}

		// Create the output directory structure
		outputPath := filepath.Join(outputDir, filepath.Dir(relativePath))
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to create output directory structure")
			return err
		}

		extension := filepath.Ext(info.Name())
		fileName := strings.TrimSuffix(info.Name(), extension)

		if extension == ".docx" || extension == ".md" || extension == ".txt" || extension == ".ipynb" {
			err = convertFileToHTML(inputDir, relativePath, outputDir, fileName)
			if err != nil {
				return err
			}
		} else if extension == ".webloc" {
			link, err := extractLinkFromWebloc(inputDir, relativePath)
			if err != nil {
				return err
			}
			log.Info().Str("file", relativePath).Str("link", link).Msg("Found a webloc link, not doing anything it with.")
		} else if extension == ".lnk" {
			link, err := extractLinkFromWebloc(inputDir, relativePath)
			if err != nil {
				return err
			}
			log.Info().Str("file", relativePath).Str("link", link).Msg("Found a webloc link, not doing anything it with.")
		} else {
			log.Info().Str("extension", extension).Str("file", relativePath).Msg("Skipping file since extension is not supported")
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Error walking through input directory")
		return err
	}

	return nil
}

type Webloc struct {
	XMLName xml.Name `xml:"plist"`
	Dict    struct {
		Key    string `xml:"key"`
		String string `xml:"string"`
	} `xml:"dict"`
}

func extractLinkFromWebloc(inputDir string, inputFileRelPath string) (string, error) {
	inputDirAbs, err := filepath.Abs(inputDir)
	if err != nil {
		log.Error().Err(err).Str("inputDirAbs", inputDir).Msg("Failed to get input absolute path")
		return "", err
	}
	inputFileAbsPath := filepath.Join(inputDirAbs, inputFileRelPath)

	// Read the contents of the webloc file
	data, err := ioutil.ReadFile(inputFileAbsPath)
	if err != nil {
		return "", err
	}

	// Unmarshal the XML data into a Webloc struct
	var webloc Webloc
	err = xml.Unmarshal(data, &webloc)
	if err != nil {
		return "", err
	}

	// Extract and return the URL
	return webloc.Dict.String, nil
}

type ShortcutHeader struct {
	HeaderSize     uint32
	LinkCLSID      [16]byte
	LinkFlags      uint32
	FileAttributes uint32
	CreationTime   uint64
	AccessTime     uint64
	WriteTime      uint64
	FileSize       uint32
	IconIndex      uint32
	ShowCommand    uint32
	HotKey         uint16
	Reserved1      uint16
	Reserved2      uint32
	Reserved3      uint32
}

func extractLinkFromShortcut(inputDir string, inputFileRelPath string) (string, error) {
	inputDirAbs, err := filepath.Abs(inputDir)
	if err != nil {
		log.Error().Err(err).Str("inputDirAbs", inputDir).Msg("Failed to get input absolute path")
		return "", err
	}
	inputFileAbsPath := filepath.Join(inputDirAbs, inputFileRelPath)

	// Open the shortcut file
	file, err := os.Open(inputFileAbsPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the shortcut header
	var header ShortcutHeader
	err = binary.Read(file, binary.LittleEndian, &header)
	if err != nil {
		return "", err
	}

	// Check if the file is a valid Windows shortcut
	if string(header.LinkCLSID[:]) != "{00021401-0000-0000-C000-000000000046}" {
		return "", errors.New("not a valid Windows shortcut file")
	}

	// Read the remaining data to extract the URL
	remainingDataSize := header.HeaderSize - 76
	remainingData := make([]byte, remainingDataSize)
	_, err = io.ReadFull(file, remainingData)
	if err != nil {
		return "", err
	}

	// Find the URL prefix
	urlPrefix := []byte("URL")
	index := bytesIndex(remainingData, urlPrefix)
	if index == -1 {
		return "", errors.New("no URL found in the shortcut file")
	}

	// Extract the URL
	url := string(remainingData[index+4:])

	return url, nil
}

func bytesIndex(data []byte, substr []byte) int {
	n := len(data)
	m := len(substr)
	for i := 0; i < n-m+1; i++ {
		if bytesEqual(data[i:i+m], substr) {
			return i
		}
	}
	return -1
}

func bytesEqual(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func convertFileToHTML(inputDir string, inputFileRelPath string, outputDir string, outputFileName string) error {
	outputFileName = outputFileName + ".html"

	// Get absolute path of input directory
	inputDirAbs, err := filepath.Abs(inputDir)
	if err != nil {
		log.Error().Err(err).Str("inputDirAbs", inputDir).Msg("Failed to get input absolute path")
		return err
	}

	// Get absolute path of output directory
	outputDirAbs, err := filepath.Abs(outputDir)
	if err != nil {
		log.Error().Err(err).Str("outputDirAbs", outputDir).Msg("Failed to get output absolute path")
		return err
	}

	// Create the output directory structure
	outputPath := filepath.Join(outputDir, filepath.Dir(inputFileRelPath))
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		log.Error().Err(err).Str("outputPath", outputPath).Msg("Failed to create output path")
		return err
	}

	assetsPath := filepath.Join(outputDirAbs, filepath.Dir("_assets"))
	if err := os.MkdirAll(assetsPath, 0755); err != nil {
		log.Error().Err(err).Str("assetsPath", assetsPath).Msg("Failed to create assets path")
		return err
	}

	// Construct the output file path
	inputFileAbsPath := filepath.Join(inputDirAbs, inputFileRelPath)
	outputFileRelPath := filepath.Join(filepath.Dir(inputFileRelPath), outputFileName)

	// Run the pandoc command to convert the file to HTML
	cmd := exec.Command("pandoc", inputFileAbsPath, "-o", outputFileName, "--standalone", "--extract-media=_assets")
	cmd.Dir = filepath.Join(outputDir, filepath.Dir(inputFileRelPath))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("input", inputFileAbsPath).Str("output", outputFileRelPath).Bytes("stdout/stderr", out).Msg("Failed to convert file to HTML with Pandoc")
		return err
	}

	log.Info().Str("input", inputFileAbsPath).Str("output", outputFileRelPath).Msg("Converted file to HTML")

	return nil
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := &cli.App{
		Name:  "bloop",
		Usage: "build static websites from unstructured docs",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "input",
				Usage:    "input directory",
				Required: true,
			},
			&cli.PathFlag{
				Name:     "config",
				Usage:    "config directory",
				Required: false,
			},
			&cli.PathFlag{
				Name:     "output",
				Usage:    "output directory",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "watch",
				Value: false,
			},
			&cli.StringFlag{
				Name:     "addr",
				Usage:    "Address to serve, defaults to `:8080`",
				Value:    ":8080",
				Required: false,
			},
			&cli.BoolFlag{
				Name:   "debug",
				Value:  false,
				Hidden: true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.Bool("debug") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Debug().Msg("Debug logging enabled.")
			}

			inputDir := cCtx.Path("input")
			outputDir := cCtx.Path("output")

			if cCtx.Bool("watch") {

				wg := sync.WaitGroup{}
				wg.Add(1) // Add the server to wait group

				// Run the file watcher in a separate goroutine
				go func() {
					err := watchInputDirectory(inputDir, outputDir)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							log.Fatal().Err(err).Msg("Insufficient permissions")
						} else {
							log.Fatal().Err(err).Msg("Error while watching input directory")
						}
					}
				}()

				addr := cCtx.String("addr")

				err := serveOutputDirectory(outputDir, addr, &wg)
				if err != nil {
					if errors.Is(err, os.ErrPermission) {
						log.Fatal().Err(err).Msg("Insufficient permissions")
					} else {
						log.Fatal().Err(err).Msg("Failed to serve output directory")
					}
				}

				wg.Wait()
			} else {
				err := processFiles(inputDir, outputDir)
				if err != nil {
					if errors.Is(err, os.ErrPermission) {
						log.Fatal().Err(err).Msg("Insufficient permissions")
					} else {
						log.Fatal().Err(err).Msg("Conversion failed")
					}
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err)
	}
}
