# AGENTS Guide
Repo: github.com/marad/frontmatter (Go 1.24); no Cursor/Copilot rules.
Build: use `go build -v ./...` (mirrors CI matrix).
Release binaries: `go build -o frontmatter main.go` before packaging/tests expect binary (see main_test).
Deps: `go mod download` + `go mod verify` before builds to match CI cache.
Full test: `go test -v -race -coverprofile=coverage.out ./...`.
Quick test: `go test ./...` for fast iteration when race/cover not needed.
Single test: `go test -run TestSetSingleField ./...` (replace pattern).
Static checks: `go vet ./...`; add other linters if needed pre-submit.
Formatting: always run `gofmt` (or goimports) on touched Go files; no custom formatter.
Imports: standard lib, third-party, module-local in separate blocks; keep alphabetical within block.
Types: prefer concrete structs with exported names only when part of API (see FrontmatterInfo, ExitError).
Interfaces: keep small/behavioral; rely on std lib types where possible.
Naming: CamelCase exported, lowerCamel internal; keep ExitError-style suffixes conveying intent.
Errors: wrap with fmt.Errorf("context: %w", err); use custom ExitError only for CLI exit codes.
Nil/zero handling: favor map initializations with make before nested writes (see setValueByPath).
YAML handling: use yaml.v3 encoder with SetIndent(2); keep frontmatter separators as constants.
Testing: maintain helper assertions in main_test.go; prefer table tests when expanding coverage.
IO: always close files via defer; use bufio.Reader for multi-pass parsing as done in run path.
Dry-run semantics: keep write paths printing to stdout without touching disk (see writeFileContentForDryRun).
