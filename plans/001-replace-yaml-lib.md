# Plan 001: Replace gopkg.in/yaml.v3 with github.com/goccy/go-yaml

## Goal
Swap the YAML backend while preserving serialized frontmatter semantics (quoting, indentation, anchors, dry-run output) and ensuring contributors know about the new dependency.

## Detailed Steps
1. **Inventory usage**
   - Run `rg -n "gopkg.in/yaml.v3" -g'*.go'` and `rg -n "gopkg.in/yaml.v3"` to capture all code/doc references; paste the file list into this plan for traceability.
     - Current hits: `main.go`, `README.adoc`, `go.mod`, `CHANGELOG.adoc`, `go.sum`, `plans/001-replace-yaml-lib.md` (self-reference)
   - Check `go.mod`/`go.sum` manually and note any indirect dependencies that might also fall away once the old module is removed.
   - Flag any helper functions or structs typed against `yaml.Node`, `yaml.Encoder`, etc., because they will need targeted rewrites. Currently only `serializeFrontmatter` and regex helpers in `main.go` depend on yaml.v3 types.

2. **Behavior capture**
   - Record the current `serializeFrontmatter` output for a diverse fixture set (basic strings, URLs, timestamps, anchors, multi-line text). Keep copies under `docs/fixtures/` if helpful.
     - Baseline samples captured in `docs/fixtures/serializer-baseline.txt` via `go run . set ... --dry-run`, covering simple scalars, URLs, timestamps, colon/hash characters, and multi-line text.
   - Note where we rely on `SetIndent(2)`, custom regex cleanup for quoted keys, or any other post-processing so we can validate parity. Current code depends on `yaml.NewEncoder().SetIndent(2)` plus `regexp.MustCompile("(?m)^(\\s*)\"([A-Za-z0-9_-]+)\":")` for stripping quotes around keys.
   - Capture how errors are wrapped (e.g., `fmt.Errorf` vs `ExitError`) to ensure we keep CLI messaging stable. Existing helpers wrap everything with `fmt.Errorf("context: %w", err)` except CLI-level not-found paths which return `&ExitError{Code:2}`.

3. **Assess go-yaml API**
   - Reviewed pkg.go.dev docs (v1.18.0) and README. Encoder options map cleanly to our needs: `yaml.NewEncoder(w, yaml.Indent(2), yaml.UseLiteralStyleIfMultiline(true))` etc., and we can still construct AST nodes via `yaml.ValueToNode`/`Encoder.EncodeToNode` to force `ast.StringNode` styles.
   - go-yaml preserves anchors/aliases via struct tags and offers `WithSmartAnchor` plus `MarshalAnchor` callbacks, so anchor fidelity should improve relative to manual regex cleanup. It already emits bare keys for simple scalars, so we expect to drop the regex hack once Node styles are enforced where necessary.
   - Errors now include positional metadata. We'll continue wrapping them with `fmt.Errorf("context: %w", err)` so CLI UX stays identical even though underlying error text becomes richer. No API gaps found; plan to stick with `yaml.MapSlice` when we need ordered output (not currently required).

4. **Dependency update**
   - Run `go get github.com/goccy/go-yaml@latest` to add the module, then remove `gopkg.in/yaml.v3` imports from `go.mod`.
   - Execute `go mod tidy`, `go mod download`, and `go mod verify` to align with CI expectations.
   - Inspect `go.sum` diff to ensure only the intended modules changed; document any surprising removals/additions.

5. **Code migration**
   - For each file from step 1, swap the import path and update types/functions to their go-yaml equivalents (e.g., `yaml.Node`, encoder helpers).
   - Update helper utilities to use go-yaml specific APIs (such as `yaml.NewEncoder` options) and remove obsolete regex-based quote stripping if redundant.
   - Ensure error wrapping remains identical; add comments where behavior differs intentionally.

6. **Quote control enhancements**
   - Refactor `serializeFrontmatter` to construct explicit `yaml.Node` trees, setting `Style = yaml.PlainStyle` for safe scalars while retaining double quotes for values that require them.
   - Leverage go-yaml hooks (e.g., `yaml.WithStringStyle`) if it simplifies enforcing plain style.
   - Add unit-level helpers that decide when to force quotes so future changes can tap into a single decision point.

7. **Testing**
   - Extend `main_test.go` with table-driven cases covering: timestamps vs strings, URLs, values containing `:` or `#`, anchors/aliases, and dry-run paths.
   - Add regression tests for any fixture captured in step 2, asserting byte-for-byte equality where feasible.
   - Consider property-style tests that reparse serialized YAML to ensure round-trip fidelity.

8. **Docs update**
   - Update README, CHANGELOG, AGENTS, and any release/playbook docs to mention go-yaml, including reasons for the swap (better quote control, performance, etc.).
   - Call out any new constraints (e.g., go-yaml minimum Go version) so contributors are aware.
   - If user-facing behavior changes (even subtly), document it in CHANGELOG under an "Unreleased" section.

9. **Verification**
   - Run `go build -v ./...` to ensure the project compiles without the old dependency.
   - Execute `go test ./...` for a quick signal, followed by the full `go test -v -race -coverprofile=coverage.out ./...` suite to mirror CI.
   - Capture command output (especially failures) in this plan or PR notes so reviewers know what was validated.

10. **Rollout and communication**
   - In the PR description, summarize observed risks (anchor behavior, marshaler differences) and how we mitigated them.
   - Outline a rollback path (e.g., keep a branch/tag before the dependency swap, note commands to revert go.mod/go.sum changes).
   - Flag downstream tooling owners if they rely on the old quoting rules, and suggest running their pipelines against the branch before merge.
