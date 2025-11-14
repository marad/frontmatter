# Plan 001: Replace gopkg.in/yaml.v3 with github.com/goccy/go-yaml

## Goal
Swap the YAML backend while preserving serialized frontmatter semantics (quoting, indentation, anchors, dry-run output) and ensuring contributors know about the new dependency.

## Detailed Steps
1. **Inventory usage**
   - Run `rg -n "gopkg.in/yaml.v3" -g'*.go'` and `rg -n "gopkg.in/yaml.v3"` to capture all code/doc references; paste the file list into this plan for traceability.
   - Check `go.mod`/`go.sum` manually and note any indirect dependencies that might also fall away once the old module is removed.
   - Flag any helper functions or structs typed against `yaml.Node`, `yaml.Encoder`, etc., because they will need targeted rewrites.

2. **Behavior capture**
   - Record the current `serializeFrontmatter` output for a diverse fixture set (basic strings, URLs, timestamps, anchors, multi-line text). Keep copies under `docs/fixtures/` if helpful.
   - Note where we rely on `SetIndent(2)`, custom regex cleanup for quoted keys, or any other post-processing so we can validate parity.
   - Capture how errors are wrapped (e.g., `fmt.Errorf` vs `ExitError`) to ensure we keep CLI messaging stable.

3. **Assess go-yaml API**
   - Review `github.com/goccy/go-yaml` docs/examples focusing on `yaml.Marshal`, `yaml.Node`, `Encoder#SetIndent`, and style control like `yaml.WithStringStyle` or direct node `Style` mutations.
   - Confirm whether go-yaml already emits bare keys/plain scalars, reducing the need for the regex cleanup step.
   - Identify any incompatibilities (e.g., different error types, missing features) and log mitigation ideas in this plan before coding.

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
