package manifest

import (
	"fmt"
	"os"
	"path/filepath"
)

// Loader loads manifests from file paths.
type Loader struct {
	basePath string
}

// NewLoader creates a loader with the given base path.
func NewLoader(basePath string) *Loader {
	return &Loader{basePath: basePath}
}

// Load reads a manifest from a file path.
func (l *Loader) Load(path string) (*Manifest, error) {
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(l.basePath, path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", fullPath, err)
	}

	manifest, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", fullPath, err)
	}

	// Store the manifest path for resolving relative paths
	manifest.ManifestPath = fullPath

	return manifest, nil
}

// LoadAll loads multiple manifests from paths.
func (l *Loader) LoadAll(paths []string) ([]Manifest, error) {
	var manifests []Manifest
	for _, path := range paths {
		m, err := l.Load(path)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, *m)
	}
	return manifests, nil
}
