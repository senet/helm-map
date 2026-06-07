package renderer

import (
	"encoding/json"

	"github.com/senet/helm-map/internal/engine/graph"
)

// JSONRenderer outputs the graph as structured JSON.
type JSONRenderer struct{}

type jsonOutput struct {
	Version string          `json:"version"`
	Meta    graph.GraphMeta `json:"meta"`
	Nodes   []graph.Node    `json:"nodes"`
	Edges   []graph.Edge    `json:"edges"`
}

// Render produces JSON output matching the helm-map.com schema v1.
func (r *JSONRenderer) Render(g *graph.Graph) ([]byte, error) {
	out := jsonOutput{
		Version: "1",
		Meta:    g.Meta,
		Nodes:   g.Nodes,
		Edges:   g.Edges,
	}
	if out.Nodes == nil {
		out.Nodes = []graph.Node{}
	}
	if out.Edges == nil {
		out.Edges = []graph.Edge{}
	}
	return json.MarshalIndent(out, "", "  ")
}
