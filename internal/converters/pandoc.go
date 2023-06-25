package converters

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func ConvertFileToHTML(inputDir string, inputFileRelPath string, outputDir string, outputFileName string) error {
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
