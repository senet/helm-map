package graph

import (
	"fmt"
	"sort"
	"time"

	"github.com/senet/helm-map/internal/engine/resolver"
)

// BuildConfig holds options for graph construction.
type BuildConfig struct {
	PluginVersion string
	HelmVersion   string
	KubeContext   string
}

// BuildFromDeps constructs a Graph from resolved chart dependencies.
func BuildFromDeps(deps []resolver.ResolvedDep, rootName, rootVersion string, cfg BuildConfig) *Graph {
	g := &Graph{
		Meta: GraphMeta{
			GeneratedAt:   time.Now().UTC(),
			PluginVersion: cfg.PluginVersion,
			HelmVersion:   cfg.HelmVersion,
			KubeContext:   cfg.KubeContext,
		},
	}

	seen := make(map[string]bool)

	// Add the root chart node.
	rootID := ChartNodeID(rootName, rootVersion)
	g.addNode(Node{
		ID:      rootID,
		Kind:    NodeChart,
		Name:    rootName,
		Version: rootVersion,
	}, seen)

	// Recursively add all dependency nodes and edges.
	g.addDepsRecursive(rootID, deps, seen)

	return g
}

func (g *Graph) addDepsRecursive(parentID string, deps []resolver.ResolvedDep, seen map[string]bool) {
	for _, dep := range deps {
		childID := ChartNodeID(dep.Name, dep.Version)
		optional := dep.Condition != "" || len(dep.Tags) > 0
		g.addNode(Node{
			ID:         childID,
			Kind:       NodeChart,
			Name:       dep.Name,
			Version:    dep.Version,
			Repository: dep.Repository,
			Optional:   optional,
			Condition:  dep.Condition,
			Tags:       dep.Tags,
		}, seen)

		g.Edges = append(g.Edges, Edge{
			From:      parentID,
			To:        childID,
			Rel:       EdgeDependsOn,
			Condition: dep.Condition,
			Tags:      dep.Tags,
		})

		if len(dep.Children) > 0 {
			g.addDepsRecursive(childID, dep.Children, seen)
		}
	}
}

func (g *Graph) addNode(n Node, seen map[string]bool) {
	if seen[n.ID] {
		return
	}
	seen[n.ID] = true
	g.Nodes = append(g.Nodes, n)
}

// Roots returns all nodes that have no incoming DependsOn edges.
func (g *Graph) Roots() []Node {
	g.ensureIndex()
	var roots []Node
	for _, n := range g.Nodes {
		if len(g.parentIndex[n.ID]) == 0 {
			roots = append(roots, n)
		}
	}
	return roots
}

// Children returns the direct children of the node with the given ID.
func (g *Graph) Children(id string) []Node {
	g.ensureIndex()
	var children []Node
	for _, childID := range g.childIndex[id] {
		if n := g.nodeIndex[childID]; n != nil {
			children = append(children, *n)
		}
	}
	return children
}

// MaxDepth returns the length of the longest path in the DAG.
func (g *Graph) MaxDepth() int {
	g.ensureIndex()
	memo := make(map[string]int)
	maxD := 0
	for _, n := range g.Nodes {
		d := g.depthFrom(n.ID, memo)
		if d > maxD {
			maxD = d
		}
	}
	return maxD
}

func (g *Graph) depthFrom(id string, memo map[string]int) int {
	if d, ok := memo[id]; ok {
		return d
	}
	maxChild := 0
	for _, childID := range g.childIndex[id] {
		d := g.depthFrom(childID, memo)
		if d > maxChild {
			maxChild = d
		}
	}
	result := 0
	if len(g.childIndex[id]) > 0 {
		result = 1 + maxChild
	}
	memo[id] = result
	return result
}

// TopoSort returns nodes in topological order using Kahn's algorithm.
// The order is deterministic — ties are broken by node ID (lexicographic).
// Returns an error if the graph contains a cycle.
func (g *Graph) TopoSort() ([]Node, error) {
	g.ensureIndex()

	inDegree := make(map[string]int)
	for _, n := range g.Nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range g.Edges {
		inDegree[e.To]++
	}

	// Seed queue with nodes that have no incoming edges.
	var queue []string
	for _, n := range g.Nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}
	sort.Strings(queue) // deterministic start order

	var result []Node
	for len(queue) > 0 {
		// Pop first (smallest ID).
		id := queue[0]
		queue = queue[1:]

		node := g.nodeIndex[id]
		if node != nil {
			result = append(result, *node)
		}

		// Process children.
		children := g.childIndex[id]
		for _, childID := range children {
			inDegree[childID]--
			if inDegree[childID] == 0 {
				queue = append(queue, childID)
			}
		}
		// Re-sort to keep deterministic order.
		sort.Strings(queue)
	}

	if len(result) != len(g.Nodes) {
		return nil, fmt.Errorf("graph contains a cycle: sorted %d of %d nodes", len(result), len(g.Nodes))
	}

	return result, nil
}

// HasCycle returns true if the graph contains a cycle.
func (g *Graph) HasCycle() bool {
	_, err := g.TopoSort()
	return err != nil
}
