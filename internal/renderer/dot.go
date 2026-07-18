package renderer

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/senet/helm-map/internal/engine/graph"
)

// DotRenderer outputs the graph as Graphviz DOT format.
type DotRenderer struct{}

// dotNodeStyle holds the visual attributes for each NodeKind.
type dotNodeStyle struct {
	shape     string
	fillcolor string
}

// nodeStyles maps NodeKind to its Graphviz visual style.
var nodeStyles = map[graph.NodeKind]dotNodeStyle{
	graph.NodeChart:       {shape: "box", fillcolor: "#aed6f1"},     // blue
	graph.NodeRelease:     {shape: "box", fillcolor: "#a9dfbf"},     // green
	graph.NodeK8sResource: {shape: "box", fillcolor: "#fdebd0"},     // yellow/orange
	graph.NodeImage:       {shape: "ellipse", fillcolor: "#d7bde2"}, // magenta
}

// defaultNodeStyle is used when a NodeKind is not in nodeStyles.
var defaultNodeStyle = dotNodeStyle{shape: "box", fillcolor: "#eeeeee"}

// Render produces Graphviz DOT output for the graph.
func (r *DotRenderer) Render(g *graph.Graph) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("digraph helm_map {\n")
	buf.WriteString("  graph [rankdir=LR fontname=\"Helvetica\"]\n")
	buf.WriteString("  node [fontname=\"Helvetica\" fontsize=12]\n")
	buf.WriteString("  edge [fontname=\"Helvetica\" fontsize=10]\n")

	// Group Release nodes by namespace for subgraph clusters; all others go inline.
	type clusterEntry struct {
		ns    string
		nodes []graph.Node
	}
	clusterMap := make(map[string][]graph.Node)
	var inlineNodes []graph.Node

	for _, n := range g.Nodes {
		if n.Kind == graph.NodeRelease && n.Namespace != "" {
			clusterMap[n.Namespace] = append(clusterMap[n.Namespace], n)
		} else {
			inlineNodes = append(inlineNodes, n)
		}
	}

	// Emit inline nodes (Charts, Images, K8sResources, and Releases without namespace).
	if len(inlineNodes) > 0 {
		buf.WriteByte('\n')
		for _, n := range inlineNodes {
			writeNodeDOT(&buf, n, "  ")
		}
	}

	// Emit namespace subgraphs in deterministic order.
	if len(clusterMap) > 0 {
		namespaces := make([]string, 0, len(clusterMap))
		for ns := range clusterMap {
			namespaces = append(namespaces, ns)
		}
		sort.Strings(namespaces)

		for _, ns := range namespaces {
			buf.WriteByte('\n')
			fmt.Fprintf(&buf, "  subgraph cluster_%s {\n", dotSafeIdent(ns))
			fmt.Fprintf(&buf, "    label=%s\n", dotQuote(ns))
			fmt.Fprintf(&buf, "    style=dashed\n")
			for _, n := range clusterMap[ns] {
				writeNodeDOT(&buf, n, "    ")
			}
			buf.WriteString("  }\n")
		}
	}

	// Emit edges.
	if len(g.Edges) > 0 {
		buf.WriteByte('\n')
		for _, e := range g.Edges {
			writeEdgeDOT(&buf, e)
		}
	}

	buf.WriteString("}\n")
	return buf.Bytes(), nil
}

// writeNodeDOT writes a single DOT node statement to buf.
func writeNodeDOT(buf *bytes.Buffer, n graph.Node, indent string) {
	style, ok := nodeStyles[n.Kind]
	if !ok {
		style = defaultNodeStyle
	}

	label := dotNodeLabel(n)
	borderColor := "#666666"
	if n.Optional {
		borderColor = "#aaaaaa"
	}

	fmt.Fprintf(buf, "%s%s [label=%s shape=%s style=\"filled,rounded\" fillcolor=%s color=%s]\n",
		indent,
		dotQuote(n.ID),
		dotQuote(label),
		style.shape,
		dotQuote(style.fillcolor),
		dotQuote(borderColor),
	)
}

// writeEdgeDOT writes a single DOT edge statement to buf.
func writeEdgeDOT(buf *bytes.Buffer, e graph.Edge) {
	style := "solid"
	// Optional/conditional edges and non-DependsOn relationships use dashed lines.
	if e.Condition != "" || len(e.Tags) > 0 || e.Rel != graph.EdgeDependsOn {
		style = "dashed"
	}

	fmt.Fprintf(buf, "  %s -> %s [style=%s",
		dotQuote(e.From),
		dotQuote(e.To),
		style,
	)

	var labelParts []string
	if e.Condition != "" {
		labelParts = append(labelParts, e.Condition)
	}
	if len(e.Tags) > 0 {
		labelParts = append(labelParts, "tags: "+strings.Join(e.Tags, ", "))
	}
	if len(labelParts) > 0 {
		fmt.Fprintf(buf, " label=%s", dotQuote(strings.Join(labelParts, "\n")))
	}

	buf.WriteString("]\n")
}

// dotNodeLabel returns the display label for a node.
func dotNodeLabel(n graph.Node) string {
	switch n.Kind {
	case graph.NodeChart:
		if n.Version != "" {
			return n.Name + "\n" + n.Version
		}
		return n.Name
	case graph.NodeRelease:
		if n.Namespace != "" {
			return n.Name + "\n[Release:" + n.Namespace + "]"
		}
		return n.Name + "\n[Release]"
	case graph.NodeK8sResource:
		return n.Name
	case graph.NodeImage:
		return n.Name
	default:
		return n.Name
	}
}

// dotQuote wraps s in DOT double-quotes, escaping internal special characters.
// This prevents DOT injection from user-supplied chart names, versions, or
// repository URLs that contain backslashes, quotes, or newlines.
func dotQuote(s string) string {
	// Order matters: escape backslashes first to avoid double-escaping.
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	// Actual newline bytes become DOT's \n sequence (label line breaks).
	s = strings.ReplaceAll(s, "\n", `\n`)
	return `"` + s + `"`
}

// dotSafeIdent converts an arbitrary string to a safe DOT identifier
// (only alphanumerics and underscores) for use in subgraph cluster names.
func dotSafeIdent(s string) string {
	var b strings.Builder
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b.WriteRune(c)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
