// Package renderer provides output formatters for the helm-map graph.
package renderer

import "github.com/senet/helm-map/internal/engine/graph"

// Format specifies the output format for rendering.
type Format string

const (
	FormatTerminal Format = "terminal"
	FormatDOT      Format = "dot"
	FormatSVG      Format = "svg"
	FormatJSON     Format = "json"
)

// Options configures renderer behaviour.
type Options struct {
	// WithImages includes image nodes in the output.
	WithImages bool
	// NoColor disables ANSI colour codes (auto-detected for non-TTY).
	NoColor bool
}

// Renderer converts a graph into a byte representation.
type Renderer interface {
	Render(g *graph.Graph) ([]byte, error)
}

// New creates a renderer for the given format.
func New(f Format, opts Options) Renderer {
	switch f {
	case FormatTerminal:
		return &TerminalRenderer{opts: opts}
	case FormatJSON:
		return &JSONRenderer{}
	case FormatDOT:
		return &DotRenderer{}
	case FormatSVG:
		return &SvgRenderer{}
	default:
		return &TerminalRenderer{opts: opts}
	}
}
