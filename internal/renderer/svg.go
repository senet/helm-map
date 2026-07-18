package renderer

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-graphviz"
	"github.com/senet/helm-map/internal/engine/graph"
)

// SvgRenderer renders the graph as a standalone SVG using Graphviz layout.
// It reuses DotRenderer to produce the intermediate DOT representation and
// delegates layout and rasterisation to the bundled Graphviz WASM engine
// (github.com/goccy/go-graphviz), which requires no system-level installation.
type SvgRenderer struct{}

// Render produces an SVG byte slice for the graph.
func (r *SvgRenderer) Render(g *graph.Graph) ([]byte, error) {
	// Step 1: produce DOT intermediate representation.
	dot := &DotRenderer{}
	dotBytes, err := dot.Render(g)
	if err != nil {
		return nil, fmt.Errorf("generating DOT intermediate: %w", err)
	}

	// Step 2: initialise the Graphviz WASM engine.
	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialising graphviz engine: %w", err)
	}
	defer gv.Close()

	// Step 3: parse the DOT source.
	parsed, err := graphviz.ParseBytes(dotBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing DOT source: %w", err)
	}
	defer parsed.Close()

	// Step 4: render to SVG.
	var buf bytes.Buffer
	if err := gv.Render(ctx, parsed, graphviz.SVG, &buf); err != nil {
		return nil, fmt.Errorf("rendering SVG: %w", err)
	}

	return buf.Bytes(), nil
}
