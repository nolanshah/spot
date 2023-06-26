package application

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ConfigPath      string         `yaml:"-"`
	ContentPath     string         `yaml:"content_path"`
	StaticPath      string         `yaml:"static_path"`
	TemplatesPath   string         `yaml:"templates_path"`
	BuildPath       string         `yaml:"build_path"`
	DefaultTemplate string         `yaml:"default_template"`
	Content         []ContentEntry `yaml:"content"`

	contentTrie pathTrie `yaml:"-"`
}

type ContentEntry struct {
	InputPath  string `yaml:"input_path"`
	OutputPath string `yaml:"output_path"`
	Template   string `yaml:"template"`
}

type trieNode struct {
	isEnd    bool
	children map[string]*trieNode
	entry    *ContentEntry
}

type pathTrie struct {
	root *trieNode
}

func newTrie() *pathTrie {
	return &pathTrie{
		root: &trieNode{
			isEnd:    false,
			children: make(map[string]*trieNode),
			entry:    nil,
		},
	}
}

func (t *pathTrie) insert(path string, entry *ContentEntry) {
	node := t.root
	segments := strings.Split(path, "/")
	for _, segment := range segments {
		if node.children[segment] == nil {
			node.children[segment] = &trieNode{
				isEnd:    false,
				children: make(map[string]*trieNode),
				entry:    nil,
			}
		}
		node = node.children[segment]
	}
	node.isEnd = true
	node.entry = entry
}

func (t *pathTrie) search(path string) *ContentEntry {
	node := t.root
	segments := strings.Split(path, "/")
	for _, segment := range segments {
		if node.children[segment] == nil {
			break
		}
		node = node.children[segment]
	}
	if node.isEnd {
		return node.entry
	}
	return nil
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
	if len(config.DefaultTemplate) > 0 {
		config.DefaultTemplate = filepath.Join(config.TemplatesPath, config.DefaultTemplate)
	}

	// Build the path trie for content entries
	contentTrie := newTrie()
	for i := range config.Content {
		config.Content[i].InputPath = filepath.Join(config.ContentPath, config.Content[i].InputPath)
		config.Content[i].OutputPath = filepath.Join(config.BuildPath, config.Content[i].OutputPath)
		config.Content[i].Template = filepath.Join(config.TemplatesPath, config.Content[i].Template)
		contentTrie.insert(config.Content[i].InputPath, &config.Content[i])
	}
	config.contentTrie = *contentTrie

	return config, nil
}

func MatchContentEntry(config Config, inputPath string) *ContentEntry {
	return config.contentTrie.search(inputPath)
}
