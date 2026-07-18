# helm-map Plugin — TODO & Implementation Roadmap

> Tracked items are organised by phase. Each phase is independently shippable.
> Phases build on each other but v3 users are never broken by v4 work.

---

## Legend

- `[ ]` Not started
- `[~]` In progress
- `[x]` Done
- `[!]` Blocked / needs decision
- `[s]` Stretch goal (nice to have, not required for phase completion)

---

## Phase 1 — Helm v3 CLI Plugin (Chart dependency tree, terminal output)

**Goal:** `helm plugin install https://github.com/your-org/helm-map` works. Developers can run `helm map chart ./my-chart` and see a dependency tree in the terminal. No cluster access required.

### Project scaffolding

- [x] Initialise Go module (`go mod init github.com/senet/helm-map`)
- [x] Set up `cmd/helm-map/main.go` with Cobra root command
- [x] Add sub-commands: `chart`, `deps`, `release`, `live`, `push`, `version`
- [x] Wire `--output`, `--depth`, `--with-images`, `--dry-run`, `--namespace`, `--kubeconfig`, `--kube-context` global flags
- [x] Bind flags to environment variables via Viper (`HELM_MAP_*` prefix)
- [x] Read config from `$HELM_DATA_HOME/helm-map/config.yaml` with Viper
- [x] Set up `Makefile` targets: `build`, `test`, `lint`, `release`, `install-local`
- [x] Set up `goreleaser` for cross-platform builds (linux/amd64, darwin/amd64, darwin/arm64, windows/amd64)
- [x] Add `.pre-commit-config.yaml` for pre-commit validation (fmt, lint, test, security)

### Plugin manifest & install

- [x] Write `plugin.yaml` with `hooks`, `platformCommand` entries
- [x] Write `scripts/install.sh` — downloads binary or builds from source, verifies SHA256
- [x] Write `scripts/verify.sh` — standalone checksum verification helper
- [x] Publish SHA256 checksums alongside GitHub Release assets (via goreleaser)
- [x] Test `helm plugin install` end-to-end on linux/amd64 (Helm v3 + v4)
- [ ] Test `helm plugin install` end-to-end on darwin/arm64
- [ ] Test `helm plugin install` end-to-end on windows/amd64
- [x] Verify `helm map --help` and `helm map version` work after install

### Dependency Resolver (`internal/engine/resolver`)

- [x] Parse `Chart.yaml` dependencies block (Helm v3/v4 format)
- [x] Parse `requirements.yaml` for legacy Helm v2 charts (read-only support)
- [x] Read pinned versions from `Chart.lock`
- [x] Resolve version constraints against repository index (`index.yaml`) when no lock file
- [x] Resolve `repository: "file://..."` local chart dependencies
- [x] Resolve `repository: "oci://..."` dependencies (OCI tag listing)
- [x] Implement remote chart downloading logic (fetch via `helm pull` or SDK into temp dir if argument is remote `repo/name:version`)
- [x] Handle `condition` field on dependencies (mark edge as optional + store condition string)
- [x] Handle `tags` field on dependencies
- [x] Implement `--depth` flag limiting for recursive resolution
- [x] Write unit tests covering: flat deps, nested deps, conditional deps, file:// deps, missing lock file, missing repo (9 tests)
- [x] Write integration test with a real multi-level chart fixture (testdata/multi-level-chart)

### Graph Builder (`internal/engine/graph`)

- [x] Define `Node`, `Edge`, `Graph`, `GraphMeta` types
- [x] Implement stable node ID generation (`chart:name:version`, etc.)
- [x] Implement `Build()` from `[]ResolvedDep` (chart-only path, Phase 1)
- [x] Implement `Roots()` — nodes with no incoming DependsOn edges
- [x] Implement `Children(id)` — direct children of a node
- [x] Implement `MaxDepth()` — longest path in the DAG
- [x] Implement `TopoSort()` using Kahn's algorithm (stable, deterministic order)
- [x] Write unit tests for graph construction, root detection, cycle detection (12 tests)

### Terminal Renderer (`internal/renderer/terminal.go`)

- [x] Implement ANSI tree rendering (custom implementation using golang.org/x/term)
- [x] Colour scheme: Chart=cyan, Release=green, K8sResource=yellow, Image=magenta
- [x] Show optional/conditional deps as dimmed with `[condition]` annotation
- [x] Show `[tags: x, y]` annotation for tagged deps
- [x] Detect non-TTY (piped output) and emit plain-text tree without ANSI codes
- [x] Write unit tests for tree output (5 tests)

### `helm map chart` command

- [x] Accept local path (`./my-chart`)
- [x] Fetch remote chart tarball if reference is `repo/name:version`
- [x] Wire to `resolver.Resolve` + `graph.Build` + `renderer.New(terminal)`
- [x] Write integration tests using fixture charts (6 testdata directories)

### `helm map deps` command

- [x] Flat list output: `name | version | repository | condition | tags`
- [x] Honour `--depth` flag
- [x] Machine-readable: `--output json` emits a JSON array of dep objects

### CI / Quality

- [x] Add GitHub Actions workflow: lint (`golangci-lint`), test (`go test ./...`), build
- [x] Enforce `go vet` in CI
- [x] Add test coverage reporting (target: >80% on `internal/engine/`)
- [x] Add `CONTRIBUTING.md`
- [x] Add `LICENSE` (Apache 2.0)
- [x] Write `README.md` with install instructions, usage examples
- [x] Add GitHub Actions release workflow with GPG signing (goreleaser + `.prov` files)

### Repo Fortification (Public Release)

- [x] Simplify `install.sh` — remove auth/private-repo code, direct URL download + SHA-256 checksum + source fallback
- [x] Pin all GitHub Actions to commit SHAs (supply-chain security)
- [x] Add `.github/CODEOWNERS` — all files require owner review
- [x] Add `.github/SECURITY.md` — responsible vulnerability disclosure policy
- [x] Add `.github/dependabot.yml` — weekly Go modules + GitHub Actions updates
- [x] Add PR template + issue templates (bug report, feature request)
- [x] Update `README.md` — CI/Release/License/Go Report badges, simplified install instructions
- [x] Update `CONTRIBUTING.md` — branch protection notes, CODEOWNERS reference, security guidance
- [x] Create `main` branch ruleset — require PR + 1 approval + CODEOWNERS review + CI checks (lint/test/build) + linear history + block force push/delete
- [x] Create `v*` tag protection ruleset — block deletion + force push
- [x] Make repo public
- [x] Re-release v0.1.0 with all hardened changes

---

## Phase 2 — Release & Live Resource Mapping

**Goal:** `helm map release my-release` and `helm map live` work. The graph includes Kubernetes resource nodes owned by releases.

### Release Inspector (`internal/engine/inspector`)

- [ ] Implement `Inspect()` using Helm SDK `action.Get` (no `kubectl` subprocess)
- [ ] Extract release metadata: name, namespace, chart name+version, revision, status, timestamp
- [ ] Parse rendered manifest YAML into `[]K8sResource` (GVK, name, namespace, labels, annotations)
- [ ] Filter resources to those owned by the release (via `app.kubernetes.io/managed-by: Helm` label)
- [ ] Implement `--with-images` flag: extract container image refs from Pod specs in Deployments, StatefulSets, DaemonSets, CronJobs, Jobs
- [ ] Write unit tests with mock Helm action responses
- [ ] Write integration test against a real kind/k3d cluster (CI optional, local required)

### Graph Builder — release path

- [ ] Extend `Build()` to accept `[]InspectedRelease` alongside `[]ResolvedDep`
- [ ] Generate `Release` nodes and `DeployedAs` edges (chart → release)
- [ ] Generate `K8sResource` nodes and `Owns` edges (release → resource)
- [ ] Generate `Image` nodes and `Uses` edges (resource → image) when `WithImages=true`
- [ ] Write unit tests for mixed chart+release graphs

### `helm map release` command

- [ ] Accept release name as positional arg
- [ ] Accept `--namespace` flag (default from `HELM_NAMESPACE`)
- [ ] Wire to `inspector.Inspect` + `graph.Build` + renderer
- [ ] Show resource count summary in terminal output header
- [ ] Write integration tests

### `helm map live` command

- [ ] List all releases in namespace via Helm SDK `action.List`
- [ ] Inspect each release in parallel (bounded goroutine pool, `--concurrency` flag, default 4)
- [ ] Merge into single graph, deduplicate shared chart dependencies
- [ ] Write integration tests

### JSON Renderer (`internal/renderer/json.go`)

- [x] Implement JSON rendering matching the `helm-map.com` schema v1
- [x] Include all `Node` and `Edge` fields
- [x] Include `meta` block (generatedAt, pluginVersion)
- [ ] Validate output against the JSON Schema before writing (`pkg/schema/v1`)
- [x] Write unit tests (JSON validity test in renderer_test.go)

### `helm map push` command

- [ ] Accept `--url` flag (default from config / env `HELM_MAP_PUSH_URL`)
- [ ] Accept `--token` flag (default from env `HELM_MAP_PUSH_TOKEN`)
- [ ] POST JSON graph to helm-map.com API endpoint
- [ ] Print returned graph URL to stdout on success
- [ ] Handle HTTP errors with clear user messages
- [ ] Write unit tests with mock HTTP server

---

## Phase 3 — SVG / DOT Rendering + OCI Distribution

**Goal:** `--output dot` and `--output svg` work. Plugin binary published as an OCI artifact alongside GitHub Release.

### DOT Renderer (`internal/renderer/dot.go`)

- [x] Emit valid Graphviz DOT syntax
- [x] Node styling: shape, colour, and label based on `NodeKind`
- [x] Edge styling: solid for `DependsOn`/`Owns`, dashed for optional deps, labelled with `EdgeKind`
- [x] Emit `subgraph cluster_*` for namespace grouping when graph includes release nodes
- [x] Write unit tests (DOT syntax validation via `dot -Tsvg` in CI)

### SVG Renderer (`internal/renderer/svg.go`)

- [x] Render graph to SVG via bundled Graphviz WASM engine (github.com/goccy/go-graphviz — pure Go, no system install)
- [x] Node styling inherited from DOT renderer (colour per NodeKind)
- [x] Edge styling inherited from DOT renderer (dashed for optional/conditional deps)
- [x] Self-contained: no external fonts, images, or scripts (Graphviz WASM embedded)
- [x] Write unit tests (SVG well-formedness, node count validation)

### OCI distribution

- [x] Publish Go binary as tarball to GitHub Releases via goreleaser
- [x] Publish SHA256 `checksums.txt` alongside release assets
- [ ] Publish OCI artifact of the binary (for Helm v4 OCI-install path) via `oras push`
- [x] Set up GitHub Actions release workflow triggered on `git tag v*`
- [x] Sign GitHub Release assets with GPG (`.prov` files via goreleaser)
- [ ] Test OCI install from `ghcr.io` registry

### Documentation

- [x] Add `--output dot` and `--output svg` examples to `README.md`
- [x] Add "Piping to Graphviz" section to docs
- [ ] Add rendered example SVG to `README.md`
- [ ] Write `CHANGELOG.md` for v0.3.0

---

## Phase 4 — Helm v4 WASM Plugin

**Goal:** `helm plugin install oci://ghcr.io/your-org/helm-map` works on Helm v4. The WASM module passes Helm v4's plugin verification. Falls back to binary on Helm v3 or environments without WASM runtime.

### WASM module (`cmd/wasm/`)

- [ ] Set up `cmd/wasm/main.go` with Extism Go PDK entrypoint (`//export map_run`)
- [ ] Implement `hostFunctions()` wrapper mapping Extism host calls to the core engine's `HostFunctions` interface
- [ ] Verify all core engine code compiles with TinyGo (resolve `reflect`, `fmt`, `os` compatibility)
- [ ] Replace `os.Exec` and direct filesystem calls in resolver/inspector with host function equivalents
- [ ] Compile to WASM: `tinygo build -o helm-map.wasm -target wasm ./cmd/wasm/`
- [ ] Verify WASM binary size is under 10 MB (TinyGo target; optimise if needed with `-no-debug`)
- [ ] Write smoke test: load WASM binary with Extism Go runtime, call `map_run`, verify output

### Host function interface (`internal/engine/host.go`)

- [ ] Define `HostFunctions` interface with all host function signatures
- [ ] Implement `DefaultHostFunctions` using Helm SDK + client-go (used by v3 binary)
- [ ] Implement `WasmHostFunctions` using Extism `pdk.Call` host function invocations (used by WASM)
- [ ] Write unit tests for both implementations with mock backends

### plugin.yaml v4 additions

- [ ] Add `wasm.oci`, `wasm.digest`, `wasm.type` fields to `plugin.yaml`
- [ ] Add `fallback.command` pointing to v3 binary (backwards compat)
- [ ] Test Helm v4 plugin install from OCI registry
- [ ] Test Helm v4 plugin invocation (`helm map chart ./fixture --output terminal`)
- [ ] Test fallback to v3 binary when WASM runtime is unavailable
- [ ] Test that Helm v3 still installs and runs correctly from the same `plugin.yaml`

### OCI publishing (WASM)

- [ ] Publish WASM binary to `ghcr.io/your-org/helm-map-wasm:<version>` via `oras push`
- [ ] Sign WASM OCI artifact with cosign (`cosign sign`)
- [ ] Pin digest in `plugin.yaml`: `wasm.digest: sha256:...`
- [ ] Add `WASM_PUBLISH=true` step to GitHub Actions release workflow
- [ ] Set up cosign key management (keyless via GitHub OIDC recommended)
- [ ] Document OCI install steps in `README.md` under "Helm v4" section

### Helm v4 compatibility testing

- [ ] Set up Helm v4 in CI (`helm/helm` releases, v4.x branch)
- [ ] Run full `helm map chart`, `helm map release`, `helm map live` test suite against Helm v4
- [ ] Test `--output json | helm map push` end-to-end on Helm v4
- [ ] Test that post-renderer plugin type is NOT needed for `helm-map` (it is CLI only)

---

## Phase 5 — helm-map.com Integration

**Goal:** `helm map push` delivers a graph to helm-map.com and returns a shareable URL. The web UI renders the graph interactively.

### API client (`internal/apiclient/`)

- [ ] Define `helm-map.com` API v1 interface (POST `/v1/graphs`, GET `/v1/graphs/{id}`)
- [ ] Implement `Client.Push(graph)` — POST JSON, return graph URL
- [ ] Implement authentication: Bearer token via `--token` flag or `HELM_MAP_PUSH_TOKEN` env var
- [ ] Implement retry with exponential backoff (3 retries, max 30s)
- [ ] Implement request timeout (default 30s, configurable via `--push-timeout`)
- [ ] Write unit tests with mock HTTP server

### `helm map push` polish

- [ ] Print shareable URL to stdout on success: `Graph published: https://helm-map.com/g/abc123`
- [ ] Support `--open` flag to open URL in default browser after push
- [ ] Support `--stdin` flag to read JSON graph from stdin (compose with `helm map chart … | helm map push --stdin`)
- [ ] Add `--private` flag to mark graphs as private (not publicly listed)
- [ ] Handle API errors gracefully with actionable messages

### Schema publishing

- [ ] Publish `pkg/schema/v1` as a Go module (`github.com/your-org/helm-map/pkg/schema/v1`) for import by helm-map.com backend
- [ ] Publish the JSON Schema file to `https://helm-map.com/schema/graph/v1.json`
- [ ] Version the schema: add `$schema` version field; bump to v2 only for breaking changes

---

## Ongoing / Cross-Cutting

### Testing infrastructure

- [ ] Set up kind/k3d cluster in GitHub Actions for integration tests
- [ ] Create fixture charts for testing: `fixtures/simple/`, `fixtures/nested/`, `fixtures/conditional/`, `fixtures/oci/`
- [ ] Add `make test-integration` target (requires cluster)
- [ ] Add `make test-unit` target (no cluster required, fast)
- [ ] Target: >80% unit test coverage on all `internal/` packages
- [ ] Target: integration tests cover all CLI commands on both Helm v3 and v4

### Documentation

- [ ] `README.md` — install, quickstart, all commands with examples, output format reference
- [ ] `ARCHITECTURE.md` — this file (keep up to date with code)
- [ ] `TODO.md` — this file (check off items as completed)
- [ ] `CONTRIBUTING.md` — dev setup, test commands, PR process
- [ ] `CHANGELOG.md` — per-release changes
- [ ] `docs/schema-v1.md` — JSON schema reference
- [ ] `docs/helm-v4-wasm.md` — WASM plugin deep dive for contributors
- [ ] Add inline GoDoc comments to all exported types and functions in `pkg/schema/v1`

### Performance

- [ ] Profile memory usage on large charts (50+ transitive deps)
- [ ] Implement bounded concurrency for parallel release inspection (`--concurrency` flag)
- [ ] Benchmark `TopoSort()` and `Build()` on graphs with 1000+ nodes
- [ ] Implement chart metadata cache with TTL to avoid re-fetching repository indexes on repeat runs
- [ ] [s] Implement incremental graph updates — only re-resolve changed charts

### Observability

- [ ] Structured logging via `go.uber.org/zap` — `--debug` flag enables verbose log output
- [ ] Log all Helm SDK and Kubernetes API calls at debug level
- [ ] Log cache hit/miss for chart resolution at debug level
- [ ] Emit timing summary at debug level: "Resolved 23 charts in 1.2s, inspected 3 releases in 0.8s"

### Security hardening

- [ ] Verify SHA256 of all downloaded chart tarballs (not just plugin binary)
- [ ] Reject `Chart.yaml` files with dependency `repository` values pointing to `file://` paths outside the chart directory (path traversal guard)
- [ ] Sanitise all node IDs before embedding in SVG/DOT output (XSS/injection guard)
- [ ] Rotate cosign signing key annually; document rotation process

### Stretch goals

- [ ] [s] `helm map diff <release-a> <release-b>` — show graph diff between two releases or revisions
- [ ] [s] `helm map watch` — live terminal UI that refreshes the graph on release changes
- [ ] [s] `--output mermaid` — emit Mermaid.js flowchart syntax for embedding in Markdown
- [ ] [s] `--output cytoscape` — emit Cytoscape.js JSON for advanced web graph rendering
- [ ] [s] Multi-cluster support: `--kube-contexts ctx1,ctx2` to merge graphs across clusters
- [ ] [s] ArgoCD integration: annotate release nodes with ArgoCD Application sync status
- [ ] [s] Flux integration: annotate chart nodes with HelmRelease/HelmRepository CR references
- [ ] [s] `helm map lint` — detect dependency graph issues: cycles, missing conditions, version conflicts
- [ ] [s] Neovim / VS Code extension that renders the graph inline in the editor sidebar
- [ ] [s] GitHub Action: post graph as PR comment with SVG diff when `Chart.yaml` changes

---

## Release Checklist (per version)

- [ ] All phase items for this version are checked off
- [ ] `go test ./...` passes with zero failures
- [ ] `golangci-lint run` passes with zero errors
- [ ] Integration tests pass on kind cluster with Helm v3 and Helm v4
- [ ] `CHANGELOG.md` updated
- [ ] `plugin.yaml` version bumped
- [ ] `go.mod` / `go.sum` tidy (`go mod tidy`)
- [ ] goreleaser dry-run passes: `goreleaser release --snapshot --clean`
- [ ] Tag pushed: `git tag v0.x.0 && git push origin v0.x.0`
- [ ] GitHub Release created with assets and checksums
- [ ] OCI artifact published and signed (Phase 3+)
- [ ] `README.md` install instructions tested from a clean environment
- [ ] Announcement in Helm Slack `#helm-users` channel
