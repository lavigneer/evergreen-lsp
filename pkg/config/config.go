package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/a-h/templ/lsp/protocol"
	"github.com/goccy/go-yaml"
	"github.com/lavigneer/evergreen-lsp/pkg/project"
)

type Config struct {
	Projects []*project.Project `yaml:"projects"`
	Lint     Lint               `yaml:"lint"`
}

type ProjDocResult struct {
	Project  *project.Project
	Document *project.Document
}

func (c *Config) FindProjDoc(docURI protocol.DocumentURI) (*ProjDocResult, bool) {
	for _, p := range c.Projects {
		if d, ok := p.TextDocuments[docURI]; ok {
			return &ProjDocResult{
				Project:  p,
				Document: d,
			}, true
		}
	}
	return nil, false
}

type Lint struct {
	EnforceTags     bool `yaml:"enforce_tags"`
	NoInlineScripts bool `yaml:"no_inline_scripts"`
}

const (
	ConfigFileName         = "evergreenlsp.config.yaml"
	DefaultEnforceTags     = true
	DefaultNoInlineScripts = true
)

func NewWithDefaults(ctx context.Context, workspacePath string) (*Config, error) {
	f, err := os.ReadFile(filepath.Join(workspacePath, ConfigFileName))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	config := Config{
		Projects: []*project.Project{project.New("evergreen.yml")},
		Lint: Lint{
			EnforceTags:     DefaultEnforceTags,
			NoInlineScripts: DefaultNoInlineScripts,
		},
	}
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		return nil, err
	}

	for _, p := range config.Projects {
		p.SetRoot(workspacePath)
		err := p.Init(ctx)
		if err != nil {
			return nil, err
		}
	}

	return &config, nil
}

var (
	rootIdentifiers       = []string{ConfigFileName, ".git"}
	ErrIdentifierNotFound = errors.New("workspace identifier not found")
	ErrRootNotFound       = errors.New("workspace root not found")
)

func FindWorkspaceRoot(currentPath string) (string, error) {
	for _, id := range rootIdentifiers {
		path, err := findRootIDDir(currentPath, id)
		if errors.Is(err, ErrIdentifierNotFound) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrRootNotFound, err)
		}
		return path, nil
	}
	return "", ErrRootNotFound
}

func findRootIDDir(currentPath string, identifier string) (string, error) {
	dirEntries, err := os.ReadDir(currentPath)
	if err != nil {
		return "", err
	}
	found := slices.ContainsFunc(dirEntries, func(entry os.DirEntry) bool {
		return entry.Name() == identifier
	})
	if !found {
		parentDir := filepath.Dir(currentPath)
		if parentDir == currentPath {
			return "", ErrIdentifierNotFound
		}
		return findRootIDDir(parentDir, identifier)
	}
	return currentPath, nil
}
