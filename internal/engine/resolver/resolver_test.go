package resolver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/senet/helm-map/internal/engine/resolver"
)

func testdataDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..", "..", "testdata")
}

func TestResolve_FlatChart(t *testing.T) {
	deps, meta, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "flat-chart"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "flat-chart" {
		t.Errorf("expected chart name 'flat-chart', got %q", meta.Name)
	}
	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}
	if deps[0].Name != "nginx" {
		t.Errorf("expected first dep 'nginx', got %q", deps[0].Name)
	}
	if deps[1].Name != "memcached" {
		t.Errorf("expected second dep 'memcached', got %q", deps[1].Name)
	}
}

func TestResolve_NoDeps(t *testing.T) {
	deps, meta, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "no-deps-chart"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "no-deps-chart" {
		t.Errorf("expected chart name 'no-deps-chart', got %q", meta.Name)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestResolve_MultiLevel(t *testing.T) {
	deps, meta, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "multi-level-chart"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "multi-level-chart" {
		t.Errorf("expected chart name 'multi-level-chart', got %q", meta.Name)
	}
	if len(deps) != 3 {
		t.Fatalf("expected 3 top-level deps, got %d", len(deps))
	}

	// backend should have children (postgresql).
	var backend *resolver.ResolvedDep
	for i, d := range deps {
		if d.Name == "backend" {
			backend = &deps[i]
			break
		}
	}
	if backend == nil {
		t.Fatal("backend dep not found")
	}
	if len(backend.Children) != 1 {
		t.Fatalf("expected backend to have 1 child, got %d", len(backend.Children))
	}
	if backend.Children[0].Name != "postgresql" {
		t.Errorf("expected backend child 'postgresql', got %q", backend.Children[0].Name)
	}
}

func TestResolve_LockFileVersions(t *testing.T) {
	deps, _, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "multi-level-chart"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// redis should be resolved to lock file version 17.3.7, not constraint >=17.0.0.
	var redis *resolver.ResolvedDep
	for i, d := range deps {
		if d.Name == "redis" {
			redis = &deps[i]
			break
		}
	}
	if redis == nil {
		t.Fatal("redis dep not found")
	}
	if redis.Version != "17.3.7" {
		t.Errorf("expected redis version '17.3.7' from lock file, got %q", redis.Version)
	}
}

func TestResolve_ConditionalDeps(t *testing.T) {
	deps, _, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "multi-level-chart"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var redis *resolver.ResolvedDep
	for i, d := range deps {
		if d.Name == "redis" {
			redis = &deps[i]
			break
		}
	}
	if redis == nil {
		t.Fatal("redis dep not found")
	}
	if redis.Condition != "redis.enabled" {
		t.Errorf("expected condition 'redis.enabled', got %q", redis.Condition)
	}
	if len(redis.Tags) != 1 || redis.Tags[0] != "cache" {
		t.Errorf("expected tags [cache], got %v", redis.Tags)
	}
}

func TestResolve_FileLocalDep(t *testing.T) {
	deps, meta, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "file-dep-chart"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "file-dep-chart" {
		t.Errorf("expected chart name 'file-dep-chart', got %q", meta.Name)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(deps))
	}
	if deps[0].Name != "local-lib" {
		t.Errorf("expected dep 'local-lib', got %q", deps[0].Name)
	}
}

func TestResolve_LegacyRequirements(t *testing.T) {
	deps, meta, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "legacy-chart"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "legacy-chart" {
		t.Errorf("expected chart name 'legacy-chart', got %q", meta.Name)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep from requirements.yaml, got %d", len(deps))
	}
	if deps[0].Name != "mysql" {
		t.Errorf("expected dep 'mysql', got %q", deps[0].Name)
	}
}

func TestResolve_DepthLimit(t *testing.T) {
	deps, _, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "multi-level-chart"),
		MaxDepth:  1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At depth 1, backend should NOT have children resolved.
	for _, d := range deps {
		if d.Name == "backend" && len(d.Children) > 0 {
			t.Error("at depth 1, backend should not have resolved children")
		}
	}
}

func TestResolve_MissingChart(t *testing.T) {
	_, _, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: filepath.Join(testdataDir(), "nonexistent-chart"),
	})
	if err == nil {
		t.Error("expected error for nonexistent chart, got nil")
	}
}
