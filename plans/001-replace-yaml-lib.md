# Plan 001: Replace gopkg.in/yaml.v3 with github.com/goccy/go-yaml

1. **Inventory usage**: list every import of `gopkg.in/yaml.v3` in code, docs, and go.mod/go.sum to know the touch points.
2. **Behavior capture**: record current quoting quirks, SetIndent usage, and regex-based key cleanup so we can verify parity after the swap.
3. **Assess go-yaml API**: confirm Marshal/Unmarshal compatibility, note AST/node APIs, StringStyle controls, and encoder options relevant to plain scalars.
4. **Dependency update**: add `github.com/goccy/go-yaml` (latest stable) to go.mod, drop the old module, and run `go mod tidy` plus `go mod verify`.
5. **Code migration**: replace imports, adjust helper functions to use go-yaml types, and delete the manual regex if the new encoder already emits bare keys.
6. **Quote control**: refactor `serializeFrontmatter` to build `yaml.Node` trees and set `Style = yaml.PlainStyle` where safe, keeping quotes for bodies that need them.
7. **Testing**: extend `main_test.go` with cases covering timestamps, URLs, strings requiring quotes, anchors, and dry-run flows to guard behavior.
8. **Docs update**: mention go-yaml in README, CHANGELOG, AGENTS, and any release/checklist docs so future contributors know about the dependency swap.
9. **Verification**: run `go build -v ./...`, `go test ./...`, then the full `go test -v -race -coverprofile=coverage.out ./...` to mirror CI.
10. **Rollout**: highlight risks (anchor behavior, marshaler differences) in the PR description and outline rollback steps if downstream tooling depends on old quoting.
