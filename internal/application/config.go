package application

import (
	"io/ioutil"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ConfigPath    string         `yaml:"-"`
	ContentPath   string         `yaml:"content_path"`
	StaticPath    string         `yaml:"static_path"`
	TemplatesPath string         `yaml:"templates_path"`
	BuildPath     string         `yaml:"build_path"`
	Content       []ContentEntry `yaml:"content"`
}

type ContentEntry struct {
	InputPath  string      `yaml:"input_path"`
	OutputPath string      `yaml:"output_path"`
	Converter  interface{} `yaml:"converter"`
	Template   string      `yaml:"template"`
}

func ParseConfig(configPath string) (Config, error) {
	var config Config

	// Check if configPath is an absolute path
	if !filepath.IsAbs(configPath) {
		// Convert to absolute path
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get absolute path for config.")
			return config, err
		}
		configPath = absPath
	}

	// Read YAML file
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read yaml config.")
		return config, err
	}

	// Parse YAML into Config struct
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse yaml config.")
		return config, err
	}

	// Populate configPath
	config.ConfigPath = configPath

	// Make paths absolute
	basePath := filepath.Dir(configPath)
	config.ContentPath = filepath.Join(basePath, config.ContentPath)
	config.StaticPath = filepath.Join(basePath, config.StaticPath)
	config.TemplatesPath = filepath.Join(basePath, config.TemplatesPath)
	config.BuildPath = filepath.Join(basePath, config.BuildPath)

	for i := range config.Content {
		config.Content[i].InputPath = filepath.Join(config.ContentPath, config.Content[i].InputPath)
		config.Content[i].OutputPath = filepath.Join(config.BuildPath, config.Content[i].OutputPath)
	}

	return config, nil

}

func MatchContentEntry(config Config, inputPath string) *ContentEntry {
	// Iterate over each content entry
	for _, entry := range config.Content {
		// Check if the input path matches the content entry's input path
		if entry.InputPath == inputPath {
			return &entry
		}
	}
	return nil // No match found
}
