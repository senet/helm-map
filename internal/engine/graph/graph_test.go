package graph_test

import (
	"testing"

	"github.com/senet/helm-map/internal/engine/graph"
	"github.com/senet/helm-map/internal/engine/resolver"
)

func TestBuildFromDeps_Flat(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{Name: "nginx", Version: "13.2.0", Repository: "https://charts.bitnami.com/bitnami"},
		{Name: "memcached", Version: "6.6.0", Repository: "https://charts.bitnami.com/bitnami"},
	}

	g := graph.BuildFromDeps(deps, "my-chart", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	if len(g.Nodes) != 3 {
		t.Errorf("expected 3 nodes (root + 2 deps), got %d", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(g.Edges))
	}
}

func TestBuildFromDeps_Nested(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{
			Name:    "backend",
			Version: "2.0.0",
			Children: []resolver.ResolvedDep{
				{Name: "postgresql", Version: "12.0.0"},
			},
		},
		{Name: "frontend", Version: "1.5.0"},
	}

	g := graph.BuildFromDeps(deps, "app", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	if len(g.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(g.Edges))
	}
}

func TestRoots(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{Name: "sub", Version: "1.0.0"},
	}
	g := graph.BuildFromDeps(deps, "root", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	roots := g.Roots()
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].Name != "root" {
		t.Errorf("expected root node 'root', got %q", roots[0].Name)
	}
}

func TestChildren(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{Name: "a", Version: "1.0.0"},
		{Name: "b", Version: "2.0.0"},
	}
	g := graph.BuildFromDeps(deps, "root", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	rootID := graph.ChartNodeID("root", "1.0.0")
	children := g.Children(rootID)
	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
}

func TestMaxDepth_Flat(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{Name: "a", Version: "1.0.0"},
	}
	g := graph.BuildFromDeps(deps, "root", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	if d := g.MaxDepth(); d != 1 {
		t.Errorf("expected max depth 1, got %d", d)
	}
}

func TestMaxDepth_Nested(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{
			Name: "a", Version: "1.0.0",
			Children: []resolver.ResolvedDep{
				{
					Name: "b", Version: "1.0.0",
					Children: []resolver.ResolvedDep{
						{Name: "c", Version: "1.0.0"},
					},
				},
			},
		},
	}
	g := graph.BuildFromDeps(deps, "root", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	if d := g.MaxDepth(); d != 3 {
		t.Errorf("expected max depth 3, got %d", d)
	}
}

func TestTopoSort_NoCycle(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{
			Name: "a", Version: "1.0.0",
			Children: []resolver.ResolvedDep{
				{Name: "b", Version: "1.0.0"},
			},
		},
		{Name: "c", Version: "1.0.0"},
	}
	g := graph.BuildFromDeps(deps, "root", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	sorted, err := g.TopoSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 4 {
		t.Errorf("expected 4 nodes in sorted list, got %d", len(sorted))
	}
	// Root should come first.
	if sorted[0].Name != "root" {
		t.Errorf("expected root first in topo sort, got %q", sorted[0].Name)
	}
}

func TestTopoSort_Empty(t *testing.T) {
	g := graph.BuildFromDeps(nil, "root", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	sorted, err := g.TopoSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 1 {
		t.Errorf("expected 1 node (root only), got %d", len(sorted))
	}
}

func TestHasCycle_False(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{Name: "a", Version: "1.0.0"},
	}
	g := graph.BuildFromDeps(deps, "root", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	if g.HasCycle() {
		t.Error("expected no cycle")
	}
}

func TestConditionalDeps(t *testing.T) {
	deps := []resolver.ResolvedDep{
		{
			Name:      "redis",
			Version:   "17.0.0",
			Condition: "redis.enabled",
			Tags:      []string{"cache"},
		},
	}
	g := graph.BuildFromDeps(deps, "app", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	redisID := graph.ChartNodeID("redis", "17.0.0")
	n := g.NodeByID(redisID)
	if n == nil {
		t.Fatal("redis node not found")
	}
	if !n.Optional {
		t.Error("redis should be marked as optional")
	}
	if n.Condition != "redis.enabled" {
		t.Errorf("expected condition 'redis.enabled', got %q", n.Condition)
	}
	if len(n.Tags) != 1 || n.Tags[0] != "cache" {
		t.Errorf("expected tags [cache], got %v", n.Tags)
	}
}

func TestNodeIDFunctions(t *testing.T) {
	tests := []struct {
		fn       func() string
		expected string
	}{
		{func() string { return graph.ChartNodeID("nginx", "1.0.0") }, "chart:nginx:1.0.0"},
		{func() string { return graph.ReleaseNodeID("default", "my-release") }, "release:default/my-release"},
		{func() string { return graph.K8sNodeID("default", "Deployment", "app") }, "k8s:default/Deployment/app"},
		{func() string { return graph.ImageNodeID("nginx:1.25") }, "image:nginx:1.25"},
	}
	for _, tt := range tests {
		got := tt.fn()
		if got != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, got)
		}
	}
}

func TestDeduplication(t *testing.T) {
	// Same dependency referenced from two parents should appear only once as a node.
	deps := []resolver.ResolvedDep{
		{
			Name: "a", Version: "1.0.0",
			Children: []resolver.ResolvedDep{
				{Name: "shared", Version: "1.0.0"},
			},
		},
		{
			Name: "b", Version: "1.0.0",
			Children: []resolver.ResolvedDep{
				{Name: "shared", Version: "1.0.0"},
			},
		},
	}
	g := graph.BuildFromDeps(deps, "root", "1.0.0", graph.BuildConfig{PluginVersion: "test"})

	// root + a + b + shared = 4 unique nodes
	if len(g.Nodes) != 4 {
		t.Errorf("expected 4 unique nodes, got %d", len(g.Nodes))
	}
	// root→a, root→b, a→shared, b→shared = 4 edges
	if len(g.Edges) != 4 {
		t.Errorf("expected 4 edges, got %d", len(g.Edges))
	}
}
