package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/senet/helm-map/internal/config"
	"github.com/senet/helm-map/internal/engine/graph"
	"github.com/senet/helm-map/internal/engine/resolver"
	"github.com/senet/helm-map/internal/renderer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version information — injected at build time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	config.InitViper()

	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "helm-map",
		Short: "Visualize Helm chart dependency and release resource maps",
		Long: `helm-map generates dependency graphs for charts and resource maps
for live releases. Output formats: terminal tree, DOT, SVG, JSON.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags.
	pf := root.PersistentFlags()
	pf.StringP("output", "o", "terminal", "Output format: terminal, dot, svg, json")
	pf.Int("depth", 0, "Max dependency depth (0 = unlimited)")
	pf.Bool("with-images", false, "Include container images in the graph (Phase 2 Preview)")
	_ = pf.MarkHidden("with-images")
	pf.Bool("dry-run", false, "Resolve deps without hitting cluster")
	pf.StringP("namespace", "n", "", "Override namespace")
	pf.String("kubeconfig", "", "Override kubeconfig path")
	pf.String("kube-context", "", "Override Kubernetes context")

	// Bind flags to Viper.
	_ = viper.BindPFlag("output", pf.Lookup("output"))
	_ = viper.BindPFlag("depth", pf.Lookup("depth"))
	_ = viper.BindPFlag("withImages", pf.Lookup("with-images"))
	_ = viper.BindPFlag("dryRun", pf.Lookup("dry-run"))
	_ = viper.BindPFlag("namespace", pf.Lookup("namespace"))
	_ = viper.BindPFlag("kubeconfig", pf.Lookup("kubeconfig"))
	_ = viper.BindPFlag("kubeContext", pf.Lookup("kube-context"))

	// Register sub-commands.
	root.AddCommand(
		newChartCmd(),
		newDepsCmd(),
		newReleaseCmd(),
		newLiveCmd(),
		newPushCmd(),
		newVersionCmd(),
	)

	return root
}

// ── helm map chart ──────────────────────────────────────────────────────────

func newChartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chart <path|repo/name:version>",
		Short: "Dependency graph of a chart (local or remote)",
		Args:  cobra.ExactArgs(1),
		RunE:  runChart,
	}
}

func runChart(cmd *cobra.Command, args []string) error {
	chartPath := args[0]
	cfg := config.Load()

	deps, meta, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: chartPath,
		MaxDepth:  cfg.Depth,
	})
	if err != nil {
		return fmt.Errorf("resolving dependencies: %w", err)
	}

	g := graph.BuildFromDeps(deps, meta.Name, meta.Version, graph.BuildConfig{
		PluginVersion: version,
	})

	r := renderer.New(renderer.Format(cfg.Output), renderer.Options{
		WithImages: cfg.WithImages,
	})
	out, err := r.Render(g)
	if err != nil {
		return fmt.Errorf("rendering: %w", err)
	}

	fmt.Fprint(os.Stdout, string(out))
	return nil
}

// ── helm map deps ───────────────────────────────────────────────────────────

func newDepsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deps <path|repo/name:version>",
		Short: "Flat dependency list with resolved versions",
		Args:  cobra.ExactArgs(1),
		RunE:  runDeps,
	}
}

func runDeps(cmd *cobra.Command, args []string) error {
	chartPath := args[0]
	cfg := config.Load()

	deps, _, err := resolver.Resolve(resolver.ResolveConfig{
		ChartPath: chartPath,
		MaxDepth:  cfg.Depth,
	})
	if err != nil {
		return fmt.Errorf("resolving dependencies: %w", err)
	}

	if cfg.Output == "json" {
		flat := flattenDeps(deps)
		data, err := json.MarshalIndent(flat, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}

	// Table output.
	flat := flattenDeps(deps)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tREPOSITORY\tCONDITION\tTAGS")
	for _, d := range flat {
		tags := ""
		if len(d.Tags) > 0 {
			tags = fmt.Sprintf("%v", d.Tags)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", d.Name, d.Version, d.Repository, d.Condition, tags)
	}
	return w.Flush()
}

type flatDep struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Repository string   `json:"repository"`
	Condition  string   `json:"condition,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

func flattenDeps(deps []resolver.ResolvedDep) []flatDep {
	var result []flatDep
	var walk func([]resolver.ResolvedDep)
	walk = func(ds []resolver.ResolvedDep) {
		for _, d := range ds {
			result = append(result, flatDep{
				Name:       d.Name,
				Version:    d.Version,
				Repository: d.Repository,
				Condition:  d.Condition,
				Tags:       d.Tags,
			})
			walk(d.Children)
		}
	}
	walk(deps)
	return result
}

// ── helm map release (stub) ─────────────────────────────────────────────────

func newReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "release <release-name>",
		Annotations: map[string]string{"phase": "2"},
		Short:       "Resource map of a live release (Phase 2)",
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("'helm map release' is not yet implemented — coming in Phase 2")
		},
	}
}

// ── helm map live (stub) ────────────────────────────────────────────────────

func newLiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "live",
		Annotations: map[string]string{"phase": "2"},
		Short:       "Map of all releases in a namespace (Phase 2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("'helm map live' is not yet implemented — coming in Phase 2")
		},
	}
}

// ── helm map push (stub) ────────────────────────────────────────────────────

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "push",
		Annotations: map[string]string{"phase": "2"},
		Short:       "Push JSON graph to helm-map.com API (Phase 2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("'helm map push' is not yet implemented — coming in Phase 2")
		},
	}
}

// ── helm map version ────────────────────────────────────────────────────────

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print plugin version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("helm-map %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}
}
