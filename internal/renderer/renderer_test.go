package renderer_test

import (
	"strings"
	"testing"

	"github.com/senet/helm-map/internal/engine/graph"
	"github.com/senet/helm-map/internal/engine/resolver"
	"github.com/senet/helm-map/internal/renderer"
)

func buildTestGraph() *graph.Graph {
	deps := []resolver.ResolvedDep{
		{
			Name:       "backend",
			Version:    "2.0.0",
			Repository: "https://example.com/charts",
			Children: []resolver.ResolvedDep{
				{Name: "postgresql", Version: "12.0.0", Condition: "postgresql.enabled"},
			},
		},
		{Name: "frontend", Version: "1.5.0", Repository: "https://example.com/charts"},
		{Name: "redis", Version: "17.3.7", Condition: "redis.enabled", Tags: []string{"cache"}},
	}
	return graph.BuildFromDeps(deps, "my-app", "1.0.0", graph.BuildConfig{PluginVersion: "test"})
}

func TestTerminalRenderer_ContainsNodeNames(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatTerminal, renderer.Options{NoColor: true})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	for _, name := range []string{"my-app", "backend", "frontend", "redis", "postgresql"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected output to contain %q", name)
		}
	}
}

func TestTerminalRenderer_ShowsCondition(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatTerminal, renderer.Options{NoColor: true})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "[redis.enabled]") {
		t.Error("expected output to contain condition annotation [redis.enabled]")
	}
	if !strings.Contains(output, "[tags: cache]") {
		t.Error("expected output to contain tag annotation [tags: cache]")
	}
}

func TestTerminalRenderer_TreeStructure(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatTerminal, renderer.Options{NoColor: true})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	// Should have tree connectors.
	if !strings.Contains(output, "├──") && !strings.Contains(output, "└──") {
		t.Error("expected tree connectors in output")
	}
}

func TestJSONRenderer_ValidJSON(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatJSON, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, `"version": "1"`) {
		t.Error("expected JSON output to contain version field")
	}
	if !strings.Contains(output, `"kind": "Chart"`) {
		t.Error("expected JSON output to contain Chart node kind")
	}
}

func TestTerminalRenderer_EmptyGraph(t *testing.T) {
	g := graph.BuildFromDeps(nil, "empty", "1.0.0", graph.BuildConfig{PluginVersion: "test"})
	r := renderer.New(renderer.FormatTerminal, renderer.Options{NoColor: true})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "empty") {
		t.Error("expected output to contain root chart name 'empty'")
	}
}
