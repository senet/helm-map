// Package resolver implements Helm chart dependency resolution by parsing
// Chart.yaml, Chart.lock, and requirements.yaml files.
package resolver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
// If the path is not a local directory, it attempts to fetch it as a remote chart.
func Resolve(cfg ResolveConfig) ([]ResolvedDep, *ChartMetadata, error) {
	chartPath := cfg.ChartPath

	// Check if the path is a local directory
	info, err := os.Stat(chartPath)
	if err == nil && info.IsDir() {
		return resolveAt(chartPath, cfg.MaxDepth, 0)
	}

	tempDir, tmpErr := os.MkdirTemp("", "helm-map-*")
	if tmpErr != nil {
		return nil, nil, fmt.Errorf("creating temp dir: %w", tmpErr)
	}
	// Note: since we might resolve recursively, we need to carefully manage the temp dir.
	// But actually, resolveAt reads everything into memory (ResolvedDep, ChartMetadata), 
	// so the temp dir can be cleaned up after resolveAt returns.
	defer os.RemoveAll(tempDir)

	if err == nil && !info.IsDir() {
		// It's a local file, try to extract it
		cmd := exec.Command("tar", "-xzf", chartPath, "-C", tempDir)
		if output, tarErr := cmd.CombinedOutput(); tarErr != nil {
			return nil, nil, fmt.Errorf("untarring local chart archive %s: %w\n%s", chartPath, tarErr, string(output))
		}
		
		entries, dirErr := os.ReadDir(tempDir)
		if dirErr != nil || len(entries) == 0 {
			return nil, nil, fmt.Errorf("could not find chart directory in archive")
		}
		
		for _, entry := range entries {
			if entry.IsDir() {
				return resolveAt(filepath.Join(tempDir, entry.Name()), cfg.MaxDepth, 0)
			}
		}
		return nil, nil, fmt.Errorf("no directory found inside chart archive %s", chartPath)
	}

	// It's not a local path, attempt to pull it as a remote chart
	ref := chartPath
	version := ""
	if parts := strings.Split(ref, ":"); len(parts) == 2 {
		ref = parts[0]
		version = parts[1]
	}

	args := []string{"pull", ref, "--untar", "--untardir", tempDir}
	if version != "" {
		args = append(args, "--version", version)
	}

	cmd := exec.Command("helm", args...)
	if output, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
		return nil, nil, fmt.Errorf("pulling remote chart %s: %w\n%s", chartPath, cmdErr, string(output))
	}

	// The untarred chart will be in tempDir/<chart-name>
	nameParts := strings.Split(ref, "/")
	chartName := nameParts[len(nameParts)-1]
	
	return resolveAt(filepath.Join(tempDir, chartName), cfg.MaxDepth, 0)
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
			childPath := findChildChartPath(chartPath, dep, rd.Version)
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

func findChildChartPath(parentPath string, dep Dependency, lockedVersion string) string {
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

	// For remote repositories (oci:// or http(s)://), attempt to pull the chart to a temp directory
	if dep.Repository != "" && !strings.HasPrefix(dep.Repository, "file://") {
		tempDir, err := os.MkdirTemp("", "helm-map-dep-*")
		if err != nil {
			return "" // Silently fail to resolve remote children if temp dir fails
		}

		version := lockedVersion
		if version == "" {
			version = dep.Version
		}

		args := []string{"pull", dep.Name, "--repo", dep.Repository, "--untar", "--untardir", tempDir}

		// If it is an OCI registry, --repo is not used, the ref is the repository + name
		if strings.HasPrefix(dep.Repository, "oci://") {
			ref := dep.Repository + "/" + dep.Name
			args = []string{"pull", ref, "--untar", "--untardir", tempDir}
		}

		if version != "" {
			args = append(args, "--version", version)
		}

		cmd := exec.Command("helm", args...)
		if err := cmd.Run(); err == nil {
			return filepath.Join(tempDir, dep.Name)
		}
	}

	return ""
}
