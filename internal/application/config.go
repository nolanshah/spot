package application

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
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
	InputPath   string            `yaml:"input_path"`
	OutputPath  string            `yaml:"output_path"`
	Template    string            `yaml:"template"`
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	CreatedAt   time.Time         `yaml:"created_at"`
	Tags        []string          `yaml:"tags"`
	Metadata    map[string]string `yaml:"metadata"`
}

type FrontMatterEntry struct {
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	CreatedAt   time.Time         `yaml:"created_at"`
	Tags        []string          `yaml:"tags"`
	Metadata    map[string]string `yaml:"metadata"`
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
		if config.Content[i].OutputPath != "" {
			config.Content[i].OutputPath = filepath.Join(config.BuildPath, config.Content[i].OutputPath)
		}
		config.Content[i].Template = filepath.Join(config.TemplatesPath, config.Content[i].Template)
		contentTrie.insert(config.Content[i].InputPath, &config.Content[i])
	}
	config.contentTrie = *contentTrie

	return config, nil
}

func MatchContentEntry(config Config, inputPath string, parseFrontMatter bool) ContentEntry {
	matchedEntry := config.contentTrie.search(inputPath)
	retEntry := ContentEntry{}
	if matchedEntry == nil && config.DefaultTemplate != "" {
		// Create a default ContentEntry
		outputPath := filepath.Join(config.BuildPath, getOutputPath(inputPath, config.ContentPath))
		retEntry = ContentEntry{
			InputPath:  inputPath,
			OutputPath: outputPath,
			Template:   config.DefaultTemplate,
		}
		log.Trace().Any("generatedEntry", matchedEntry).Str("outputPath", outputPath).Str("buildPath", config.BuildPath).Msg("Generated default content entry.")
	} else if matchedEntry != nil {
		retEntry = *matchedEntry
		retEntry.InputPath = inputPath
		resolvedOutputPath := filepath.Join(config.BuildPath, getOutputPath(inputPath, config.ContentPath))
		if retEntry.OutputPath == "" {
			// Fill the output path using default logic
			retEntry.OutputPath = resolvedOutputPath
		} else if retEntry.OutputPath == resolvedOutputPath {
			// no-op
		} else {
			log.Fatal().Any("entry", matchedEntry).Str("inputPath", inputPath).Msg("Remapping the output path isn't supported yet. Output path must be empty for a content entry that applies to multiple children.")
		}
	}

	if parseFrontMatter {
		var fme FrontMatterEntry

		file, err := os.Open(inputPath)
		if err != nil {
			log.Err(err).Str("file", inputPath).Msg("Failed to read file to extract front matter.")
		} else {
			defer file.Close()
			_, err := frontmatter.Parse(file, &fme)
			if err != nil {
				log.Err(err).Str("file", inputPath).Msg("Failed to read front matter from file.")
			} else {
				retEntry.Title = fme.Title
				retEntry.Description = fme.Description
				retEntry.CreatedAt = fme.CreatedAt
				retEntry.Tags = fme.Tags
				retEntry.Metadata = fme.Metadata
			}
		}
	}

	return retEntry
}

func getOutputPath(inputPath string, baseInputPath string) string {
	// Get the relative path of the input file
	relInputPath, err := filepath.Rel(baseInputPath, inputPath)
	if err != nil {
		log.Error().Err(err).Str("file", inputPath).Msg("Failed to get relative path.")
		panic(err) // if we reach this without erroring, we screwed up somewhere upstream
	}

	// Modify input path to have .html extension
	ext := filepath.Ext(relInputPath)
	base := strings.TrimSuffix(relInputPath, ext)
	return base + ".html"
}
