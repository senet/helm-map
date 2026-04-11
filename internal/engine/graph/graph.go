// Package graph provides the in-memory directed acyclic graph (DAG) model
// used to represent Helm chart dependencies, releases, and Kubernetes resources.
package graph

import "time"

// NodeKind represents the type of a node in the graph.
type NodeKind string

const (
	NodeChart       NodeKind = "Chart"
	NodeRelease     NodeKind = "Release"
	NodeK8sResource NodeKind = "K8sResource"
	NodeImage       NodeKind = "Image"
)

// Node represents a single vertex in the dependency/resource graph.
type Node struct {
	ID          string            `json:"id"`
	Kind        NodeKind          `json:"kind"`
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Repository  string            `json:"repository,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Optional    bool              `json:"optional"`
	Condition   string            `json:"condition,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// EdgeKind describes the semantic relationship between two nodes.
type EdgeKind string

const (
	EdgeDependsOn  EdgeKind = "DependsOn"  // chart → subchart
	EdgeOwns       EdgeKind = "Owns"       // release → k8s resource
	EdgeDeployedAs EdgeKind = "DeployedAs" // chart → release
	EdgeUses       EdgeKind = "Uses"       // resource → image
)

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From      string   `json:"from"`
	To        string   `json:"to"`
	Rel       EdgeKind `json:"rel"`
	Condition string   `json:"condition,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// GraphMeta contains metadata about how and when the graph was generated.
type GraphMeta struct {
	GeneratedAt   time.Time `json:"generatedAt"`
	HelmVersion   string    `json:"helmVersion,omitempty"`
	KubeContext   string    `json:"kubeContext,omitempty"`
	PluginVersion string    `json:"pluginVersion"`
}

// Graph is the top-level container for the dependency/resource DAG.
type Graph struct {
	Nodes []Node    `json:"nodes"`
	Edges []Edge    `json:"edges"`
	Meta  GraphMeta `json:"meta"`

	// Internal indexes built lazily.
	nodeIndex    map[string]*Node
	childIndex   map[string][]string
	parentIndex  map[string][]string
	indexed      bool
}

// NodeByID returns the node with the given ID, or nil if not found.
func (g *Graph) NodeByID(id string) *Node {
	g.ensureIndex()
	return g.nodeIndex[id]
}

// ChartNodeID returns a stable node ID for a chart dependency.
func ChartNodeID(name, version string) string {
	return "chart:" + name + ":" + version
}

// ReleaseNodeID returns a stable node ID for a Helm release.
func ReleaseNodeID(namespace, name string) string {
	return "release:" + namespace + "/" + name
}

// K8sNodeID returns a stable node ID for a Kubernetes resource.
func K8sNodeID(namespace, kind, name string) string {
	return "k8s:" + namespace + "/" + kind + "/" + name
}

// ImageNodeID returns a stable node ID for a container image.
func ImageNodeID(ref string) string {
	return "image:" + ref
}

func (g *Graph) ensureIndex() {
	if g.indexed {
		return
	}
	g.nodeIndex = make(map[string]*Node, len(g.Nodes))
	g.childIndex = make(map[string][]string)
	g.parentIndex = make(map[string][]string)
	for i := range g.Nodes {
		g.nodeIndex[g.Nodes[i].ID] = &g.Nodes[i]
	}
	for _, e := range g.Edges {
		g.childIndex[e.From] = append(g.childIndex[e.From], e.To)
		g.parentIndex[e.To] = append(g.parentIndex[e.To], e.From)
	}
	g.indexed = true
}
