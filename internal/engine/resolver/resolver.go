// Package resolver implements Helm chart dependency resolution by parsing
// Chart.yaml, Chart.lock, and requirements.yaml files.
package resolver

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ResolveConfig holds configuration for dependency resolution.
type ResolveConfig struct {
	ChartPath       string
	MaxDepth        int  // 0 = unlimited
	IncludeOptional bool // resolve conditional deps even if condition would disable them
}

// ResolvedDep represents a single resolved chart dependency.
type ResolvedDep struct {
	Name       string        `json:"name"`
	Version    string        `json:"version"`
	Repository string        `json:"repository"`
	Condition  string        `json:"condition,omitempty"`
	Tags       []string      `json:"tags,omitempty"`
	Children   []ResolvedDep `json:"children,omitempty"`
}

// ChartMetadata is a minimal representation of Chart.yaml.
type ChartMetadata struct {
	APIVersion   string       `yaml:"apiVersion"`
	Name         string       `yaml:"name"`
	Version      string       `yaml:"version"`
	Dependencies []Dependency `yaml:"dependencies"`
}

// Dependency represents a single entry in the Chart.yaml dependencies block.
type Dependency struct {
	Name       string   `yaml:"name"`
	Version    string   `yaml:"version"`
	Repository string   `yaml:"repository"`
	Condition  string   `yaml:"condition"`
	Tags       []string `yaml:"tags"`
	Alias      string   `yaml:"alias"`
}

// LockFile represents Chart.lock.
type LockFile struct {
	Dependencies []LockDep `yaml:"dependencies"`
	Digest       string    `yaml:"digest"`
	Generated    string    `yaml:"generated"`
}

// LockDep represents a single locked dependency entry.
type LockDep struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
}

// Resolve recursively resolves chart dependencies starting from cfg.ChartPath.
func Resolve(cfg ResolveConfig) ([]ResolvedDep, *ChartMetadata, error) {
	return resolveAt(cfg.ChartPath, cfg.MaxDepth, 0)
}

func resolveAt(chartPath string, maxDepth, currentDepth int) ([]ResolvedDep, *ChartMetadata, error) {
	meta, err := readChartMetadata(chartPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading chart metadata at %s: %w", chartPath, err)
	}

	deps := meta.Dependencies
	if len(deps) == 0 {
		// Try legacy requirements.yaml.
		deps, _ = readRequirements(chartPath)
	}

	if len(deps) == 0 {
		return nil, meta, nil
	}

	// Try to read lock file for pinned versions.
	lock, _ := readLockFile(chartPath)
	lockMap := buildLockMap(lock)

	var resolved []ResolvedDep
	for _, dep := range deps {
		rd := ResolvedDep{
			Name:       dep.Name,
			Version:    resolveVersion(dep, lockMap),
			Repository: dep.Repository,
			Condition:  dep.Condition,
			Tags:       dep.Tags,
		}

		// Recurse into children if depth allows.
		if maxDepth == 0 || currentDepth+1 < maxDepth {
			childPath := findChildChartPath(chartPath, dep)
			if childPath != "" {
				children, _, err := resolveAt(childPath, maxDepth, currentDepth+1)
				if err == nil {
					rd.Children = children
				}
			}
		}

		resolved = append(resolved, rd)
	}

	return resolved, meta, nil
}

func readChartMetadata(chartPath string) (*ChartMetadata, error) {
	data, err := os.ReadFile(filepath.Join(chartPath, "Chart.yaml"))
	if err != nil {
		return nil, err
	}
	var meta ChartMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing Chart.yaml: %w", err)
	}
	return &meta, nil
}

func readRequirements(chartPath string) ([]Dependency, error) {
	data, err := os.ReadFile(filepath.Join(chartPath, "requirements.yaml"))
	if err != nil {
		return nil, err
	}
	var reqs struct {
		Dependencies []Dependency `yaml:"dependencies"`
	}
	if err := yaml.Unmarshal(data, &reqs); err != nil {
		return nil, err
	}
	return reqs.Dependencies, nil
}

func readLockFile(chartPath string) (*LockFile, error) {
	data, err := os.ReadFile(filepath.Join(chartPath, "Chart.lock"))
	if err != nil {
		return nil, err
	}
	var lock LockFile
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	return &lock, nil
}

func buildLockMap(lock *LockFile) map[string]LockDep {
	m := make(map[string]LockDep)
	if lock == nil {
		return m
	}
	for _, d := range lock.Dependencies {
		m[d.Name] = d
	}
	return m
}

func resolveVersion(dep Dependency, lockMap map[string]LockDep) string {
	if locked, ok := lockMap[dep.Name]; ok {
		return locked.Version
	}
	// Fall back to the constraint as the "resolved" version.
	return dep.Version
}

func findChildChartPath(parentPath string, dep Dependency) string {
	// Check for file:// local dependency.
	if len(dep.Repository) > 7 && dep.Repository[:7] == "file://" {
		rel := dep.Repository[7:]
		abs := filepath.Join(parentPath, rel)
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs
		}
	}

	// Check for unpacked subchart in charts/ directory.
	subPath := filepath.Join(parentPath, "charts", dep.Name)
	if info, err := os.Stat(subPath); err == nil && info.IsDir() {
		return subPath
	}

	return ""
}
