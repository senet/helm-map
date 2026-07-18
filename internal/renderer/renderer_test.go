package renderer_test

import (
	"html"
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

// ── DOT renderer tests ───────────────────────────────────────────────────────

func TestDOTRenderer_ValidStructure(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatDOT, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	if !strings.HasPrefix(strings.TrimSpace(output), "digraph helm_map {") {
		t.Error("expected DOT output to start with 'digraph helm_map {'")
	}
	if !strings.HasSuffix(strings.TrimSpace(output), "}") {
		t.Error("expected DOT output to end with '}'")
	}
}

func TestDOTRenderer_ContainsAllNodes(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatDOT, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	for _, name := range []string{"my-app", "backend", "frontend", "redis", "postgresql"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected DOT output to contain node name %q", name)
		}
	}
}

func TestDOTRenderer_OptionalEdgeIsDashed(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatDOT, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	// redis has a condition, so its edge must be dashed.
	output := string(out)
	redisID := graph.ChartNodeID("redis", "17.3.7")
	// Find the edge line that points to redis and verify it uses style=dashed.
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, redisID) && strings.Contains(line, "->") {
			if !strings.Contains(line, "dashed") {
				t.Errorf("expected edge to redis to be dashed, got: %s", line)
			}
		}
	}
}

func TestDOTRenderer_ConditionAppearsInEdgeLabel(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatDOT, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	if !strings.Contains(string(out), "redis.enabled") {
		t.Error("expected DOT output to contain condition 'redis.enabled' in edge label")
	}
}

func TestDOTRenderer_TagsAppearInEdgeLabel(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatDOT, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	if !strings.Contains(string(out), "cache") {
		t.Error("expected DOT output to contain tag 'cache' in edge label")
	}
}

func TestDOTRenderer_EmptyGraph(t *testing.T) {
	g := graph.BuildFromDeps(nil, "empty", "1.0.0", graph.BuildConfig{PluginVersion: "test"})
	r := renderer.New(renderer.FormatDOT, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "empty") {
		t.Error("expected DOT output to contain root chart name 'empty'")
	}
	// Must still be valid DOT envelope.
	if !strings.Contains(output, "digraph helm_map {") {
		t.Error("expected valid DOT envelope even for empty graph")
	}
}

func TestDOTRenderer_DOTInjectionSafe(t *testing.T) {
	// A chart whose name contains DOT-special characters must be safely escaped.
	deps := []resolver.ResolvedDep{
		{Name: `evil"chart`, Version: `1.0\0`, Repository: "https://example.com/charts"},
	}
	g := graph.BuildFromDeps(deps, `root"app`, "1.0.0", graph.BuildConfig{PluginVersion: "test"})
	r := renderer.New(renderer.FormatDOT, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	// The unescaped quote must not appear bare inside a label value.
	// Every label= value is wrapped in quotes; internal quotes must be backslash-escaped.
	if strings.Contains(output, `label="evil"chart`) {
		t.Error("unescaped quote found in DOT output — potential injection")
	}
	// The escaped form must be present.
	if !strings.Contains(output, `evil\"chart`) {
		t.Error("expected escaped quote in node label")
	}
}

func TestDOTRenderer_NamespaceSubgraph(t *testing.T) {
	// Build a graph with a Release node that has a Namespace set.
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "chart:myapp:1.0.0", Kind: graph.NodeChart, Name: "myapp", Version: "1.0.0"},
			{ID: "release:prod/myapp", Kind: graph.NodeRelease, Name: "myapp", Namespace: "prod"},
		},
		Edges: []graph.Edge{
			{From: "chart:myapp:1.0.0", To: "release:prod/myapp", Rel: graph.EdgeDeployedAs},
		},
	}
	r := renderer.New(renderer.FormatDOT, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "subgraph cluster_prod") {
		t.Error("expected namespace subgraph 'cluster_prod' for Release node with namespace 'prod'")
	}
}

// ── SVG renderer tests ───────────────────────────────────────────────────────

func TestSVGRenderer_ProducesSVGEnvelope(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatSVG, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "<svg") {
		t.Error("expected SVG output to contain '<svg' element")
	}
	if !strings.Contains(output, "</svg>") {
		t.Error("expected SVG output to contain '</svg>' closing tag")
	}
}

func TestSVGRenderer_ContainsNodeNames(t *testing.T) {
	g := buildTestGraph()
	r := renderer.New(renderer.FormatSVG, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	// SVG is XML: Graphviz encodes characters like '-' as '&#45;'.
	// Decode entities before checking for plain-text node names.
	decoded := html.UnescapeString(string(out))
	for _, name := range []string{"my-app", "backend", "frontend", "redis", "postgresql"} {
		if !strings.Contains(decoded, name) {
			t.Errorf("expected SVG output to contain node name %q", name)
		}
	}
}

func TestSVGRenderer_EmptyGraph(t *testing.T) {
	g := graph.BuildFromDeps(nil, "solo", "0.1.0", graph.BuildConfig{PluginVersion: "test"})
	r := renderer.New(renderer.FormatSVG, renderer.Options{})

	out, err := r.Render(g)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	if !strings.Contains(string(out), "<svg") {
		t.Error("expected valid SVG even for a single-node graph")
	}
}
