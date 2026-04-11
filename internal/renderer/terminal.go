package renderer

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/senet/helm-map/internal/engine/graph"
	"golang.org/x/term"
)

// TerminalRenderer outputs an ANSI-coloured tree to stdout.
type TerminalRenderer struct {
	opts Options
}

// ANSI colour codes.
const (
	colorCyan    = "\033[36m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorMagenta = "\033[35m"
	colorDim     = "\033[2m"
	colorReset   = "\033[0m"
	colorBold    = "\033[1m"
)

func (r *TerminalRenderer) isTTY() bool {
	if r.opts.NoColor {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func (r *TerminalRenderer) color(code, text string) string {
	if !r.isTTY() {
		return text
	}
	return code + text + colorReset
}

// Render produces a tree-formatted byte output of the graph.
func (r *TerminalRenderer) Render(g *graph.Graph) ([]byte, error) {
	var buf bytes.Buffer
	roots := g.Roots()
	for _, root := range roots {
		r.renderNode(&buf, g, root, "", true, true)
	}
	return buf.Bytes(), nil
}

func (r *TerminalRenderer) renderNode(buf *bytes.Buffer, g *graph.Graph, n graph.Node, prefix string, isLast bool, isRoot bool) {
	label := r.formatNode(n)
	annotation := r.formatAnnotation(n)

	if isRoot {
		fmt.Fprintf(buf, "%s%s\n", label, annotation)
	} else {
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		fmt.Fprintf(buf, "%s%s%s%s\n", prefix, connector, label, annotation)
	}

	children := g.Children(n.ID)
	var childPrefix string
	if isRoot {
		childPrefix = ""
	} else if isLast {
		childPrefix = prefix + "    "
	} else {
		childPrefix = prefix + "│   "
	}

	for i, child := range children {
		r.renderNode(buf, g, child, childPrefix, i == len(children)-1, false)
	}
}

func (r *TerminalRenderer) formatNode(n graph.Node) string {
	switch n.Kind {
	case graph.NodeChart:
		label := fmt.Sprintf("%s (Chart %s)", n.Name, n.Version)
		if n.Optional {
			return r.color(colorDim, label)
		}
		return r.color(colorCyan, label)
	case graph.NodeRelease:
		label := fmt.Sprintf("%s (Release, %s)", n.Name, n.Namespace)
		return r.color(colorGreen, label)
	case graph.NodeK8sResource:
		label := n.Name
		return r.color(colorYellow, label)
	case graph.NodeImage:
		return r.color(colorMagenta, n.Name)
	default:
		return n.Name
	}
}

func (r *TerminalRenderer) formatAnnotation(n graph.Node) string {
	var parts []string
	if n.Condition != "" {
		parts = append(parts, fmt.Sprintf("[%s]", n.Condition))
	}
	if len(n.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("[tags: %s]", strings.Join(n.Tags, ", ")))
	}
	if len(parts) == 0 {
		return ""
	}
	ann := "  " + strings.Join(parts, " ")
	if r.isTTY() {
		return colorDim + ann + colorReset
	}
	return ann
}
