# Eigen — Improvement Backlog

Items derived from a full project evaluation against the README manifest. Grouped by priority. Each item includes enough context for a spec-agent to write change files and a compile-agent to implement.

---

## P0 — Foundation (unblocks everything else)

### ~~1. Unit tests for projection engine~~ ✅ Done

**Module**: `projection-engine`
**What**: Add Go table-driven tests for `internal/spec/projection.go` and `internal/spec/validation.go`. These are the most critical code paths in the project — the "compiler" for specs — and currently have zero test coverage.

**Projection tests should cover**:
- Empty change list produces empty spec
- Single initial change sets all fields
- Multiple changes: last-write-wins for scalar fields (title, owner, status, description, behavior)
- Acceptance criteria merged by `id` — new ids added, existing ids updated
- Acceptance criteria with `removed: true` are dropped from the projection
- Dependencies and technology are replaced wholesale (not merged)
- Metadata: `last_change` reflects most recent change timestamp, `changes_count` is correct
- Sequence ordering is respected regardless of filesystem order

**Validation tests should cover**:
- `Validate()`: missing required fields (id, domain, module, title, description, behavior, acceptance_criteria) produce errors
- `Validate()`: missing AC sub-fields (given, when, then) produce errors
- `Validate()`: broken dependency path produces error; valid path passes
- `ValidateChanges()`: no-op field (identical to current projection) is flagged
- `ValidateChanges()`: AC with same id and identical content is flagged as no-op
- `LintChangeFile()`: unquoted colons in scalars are caught
- `LintChangeFile()`: content inside block scalars (`|`, `>`) is not linted

**Files to create**: `eigen/internal/spec/projection_test.go`, `eigen/internal/spec/validation_test.go`
**Dependencies**: None

---

### ~~2. Unit tests for storage layer~~ ✅ Done

**Module**: `projection-engine` (or new module `spec-cli`)
**What**: Add Go tests for `internal/storage/storage.go`. Use `t.TempDir()` to create isolated filesystem fixtures.

**Tests should cover**:
- `ReadSpec()`: reads and unmarshals a valid spec.yaml
- `ReadChanges()`: reads all .yaml files from changes/ dir, returns sorted by sequence
- `ReadChanges()`: empty changes/ dir returns empty slice
- `WriteSpec()`: marshals and writes spec.yaml, readable back via `ReadSpec()`
- `NextSequence()`: returns 1 for empty dir, max+1 for populated dir
- `WalkModules()`: finds all modules (dirs containing changes/ subdir), respects prefix filter, returns sorted
- `SetChangeStatus()`: updates status field in-place, preserves all other content

**Files to create**: `eigen/internal/storage/storage_test.go`
**Dependencies**: None

---

### ~~3. Unit tests for HTTP API and review handlers~~ ✅ Done

**Module**: `spec-navigator/api`, `spec-navigator/change-review`
**What**: Add Go tests using `httptest` for all API handlers in `internal/server/api.go` and `internal/server/reviews.go`.

**API tests should cover**:
- `GET /api/modules` returns JSON list of all modules
- `GET /api/modules/<path>` returns module detail with spec content
- `GET /api/modules/<path>/changes` returns change list for module
- 404 for non-existent module path

**Review tests should cover**:
- `POST /api/reviews` creates session, returns session_id
- `GET /api/reviews/pending` returns session_id when one exists, 204 when none
- `GET /api/reviews/<id>` returns full session
- `POST /api/reviews/<id>/submit` with decision=approved sets status to submitted
- `POST /api/reviews/<id>/submit` with decision=rejected records comments
- Non-existent session returns 404

**Files to create**: `eigen/internal/server/api_test.go`, `eigen/internal/server/reviews_test.go`
**Dependencies**: None

---

### ~~4. GitHub Actions CI~~ ✅ Done

**What**: Add a CI workflow that runs on every push and PR. Minimum viable pipeline:
1. `go build ./...` in `eigen/` — verify the binary compiles
2. `go test ./...` in `eigen/` — run all unit tests
3. Build the binary, then run `eigen spec validate` against `specs/` — verify all specs are valid

**File to create**: `.github/workflows/ci.yml`
**Dependencies**: Items 1-3 (tests should exist before CI runs them, but CI can be set up first with just build + validate)

---

## P1 — Integrity & Robustness

### ~~5. Replace review session abstraction with change file status~~ ✅ Done

**Module**: `spec-navigator/change-review`
**What**: Review sessions are an unnecessary indirection. The change files already are the persistence layer — they survive restarts, they're on disk, and they have a `status` field. Replace the in-memory `reviewStore` and session API with direct status mutations on change files. The UI reads draft changes from the existing module API and writes decisions back via new status and comment endpoints.

**Behavior**:
- Remove `POST /api/reviews`, `GET /api/reviews/pending`, `GET /api/reviews/<id>`, `POST /api/reviews/<id>/submit` and the entire `ReviewSession` / `reviewStore` abstraction
- Add `POST /api/modules/<path>/changes/<filename>/approve` — sets `status: approved` on the change file (calls `eigen spec change-status` equivalent internally)
- Add `POST /api/modules/<path>/changes/<filename>/reject` — leaves status as `draft` and writes a `review_comment` field to the change file
- Add `review_comment` field to `spec.Change` struct and storage (written/read by the CLI too)
- Add `eigen spec change-comment <module-path> <filename> <comment>` CLI command so the agent never has to manually edit YAML — it uses the CLI to record rejection feedback, maintaining canonical formatting
- UI pending review panel: query existing `GET /api/modules/<path>/changes` and filter for `status: draft`; approve/reject buttons call the new endpoints above
- Agent polling: poll `GET /api/modules/<path>/changes` and watch for status transitions from `draft` → `approved` instead of polling `/api/reviews/pending`

**Files to modify**: `eigen/internal/server/reviews.go` (gut or remove), `eigen/internal/server/api.go` (add approve/reject endpoints), `eigen/internal/spec/types.go` (add `ReviewComment` field), `eigen/internal/storage/storage.go` (add `SetChangeComment`), `eigen/cmd/` (add `spec_change_comment.go`), `eigen/internal/server/ui/app.js`
**Dependencies**: None

---

### ~~6. Spec format version field~~ ✅ Done

**Module**: `projection-engine`
**What**: Add a `format` field to spec.yaml and change files (e.g., `format: eigen/v1`). This enables future format migrations without silent breakage.

**Behavior**:
- `eigen spec new` writes `format: eigen/v1` as the first field in the initial change
- `eigen spec project` writes `format: eigen/v1` as the first field in spec.yaml
- `eigen spec validate` warns if `format` is missing (for backward compat, don't error on existing specs)
- Projection logic ignores format field (it's metadata, not projected content)
- CLI does not enforce format version yet — this just establishes the convention

**Files to modify**: `eigen/internal/spec/types.go` (add Format field), `eigen/internal/spec/projection.go`, `eigen/internal/spec/validation.go`, `eigen/cmd/spec_new.go`
**Dependencies**: None

---

### ~~7. Pre-commit spec validation hook~~ ✅ Done

**Module**: `spec-cli/cmd-scaffold`
**What**: `eigen scaffold` should optionally install a git pre-commit hook that runs `eigen spec validate` before allowing a commit. This prevents invalid specs from entering the repository.

**Behavior**:
- `eigen scaffold` adds a `.git/hooks/pre-commit` script (or appends to existing) that runs `eigen spec validate`
- If validation fails, the commit is rejected with a clear error message
- `eigen scaffold --no-hooks` skips hook installation
- Hook only runs if spec files (anything under specs/) are staged — skip validation for code-only commits

**Files to modify**: `eigen/cmd/scaffold.go`
**Dependencies**: None

---

### ~~8. UI error handling and loading states~~ ✅ Done

**Module**: `spec-navigator/ui`
**What**: The web UI silently swallows all fetch errors and has no loading indicators. Add both.

**Error handling**:
- Wrap all `fetch()` calls in try/catch
- On error, show a dismissible toast/banner at the top of the content area with the error message
- On non-2xx response, extract error message from JSON body if available

**Loading states**:
- Show a spinner or "Loading..." text in the tree pane while `/api/modules` is loading
- Show a spinner in the detail pane while module detail is loading
- Disable approve/reject buttons while review submission is in flight

**Files to modify**: `eigen/internal/server/ui/app.js`, `eigen/internal/server/ui/style.css`
**Dependencies**: None

---

## P2 — Scale & Team Readiness

### ~~9. Compiled change audit trail (link spec history to code history)~~ ✅ Done

**Module**: `projection-engine`
**What**: When a change is marked `compiled`, record the git commit hash that implements it. This links spec history to code history — a property the README promises but doesn't implement.

**Behavior**:
- `eigen spec change-status <path> <file> compiled` accepts an optional `--commit <hash>` flag
- If `--commit` is provided, write a `compiled_commit` field into the change file alongside `status: compiled`
- If `--commit` is omitted and the CWD is a git repo, auto-detect HEAD as the commit hash
- The compile-agent skill should pass the commit hash when marking changes compiled
- `eigen serve` UI shows the commit hash (as a short hash) in the change timeline when present

**Files to modify**: `eigen/cmd/spec_change_status.go`, `eigen/internal/spec/types.go`, `eigen/internal/storage/storage.go`, `eigen/internal/server/ui/app.js`
**Dependencies**: None

---

### ~~10. Conflict detection for overlapping pending changes~~ ✅ Done

**Module**: New module — `projection-engine/conflict-detection` or extend `projection-engine`
**What**: When two pending (draft or approved) changes touch the same fields in the same module, flag it as a conflict before projection.

**Behavior**:
- New command: `eigen spec conflicts [path]` — scan all modules (or a specific one) for field-level overlaps between pending changes
- Two changes "conflict" if they both set the same scalar field, or both modify an AC with the same id
- Output: list of conflicting change pairs with the conflicting fields
- `eigen spec validate` should also report conflicts as warnings (not errors — conflicts are resolvable, not invalid)
- Exit code 0 if no conflicts, 1 if conflicts found

**Files to create**: Command file `eigen/cmd/spec_conflicts.go`, detection logic in `eigen/internal/spec/conflicts.go`
**Dependencies**: None

---

### ~~11. Keyboard navigation and accessibility in the UI~~ ✅ Done

**Module**: `spec-navigator/ui`
**What**: The tree view is mouse-only. Add keyboard navigation and basic ARIA attributes.

**Behavior**:
- Arrow up/down moves focus between visible tree nodes
- Arrow right expands a collapsed node; arrow left collapses an expanded node
- Enter selects the focused node (loads detail)
- Tree nodes have `role="treeitem"`, tree container has `role="tree"`
- `aria-expanded` on collapsible nodes
- `tabindex="0"` on the focused node, `tabindex="-1"` on others (roving tabindex)

**Files to modify**: `eigen/internal/server/ui/app.js`, `eigen/internal/server/ui/style.css`
**Dependencies**: None

---

### ~~12. Configurable review polling with backoff~~ ✅ Done

**Module**: `spec-navigator/change-review`
**What**: The review panel polls `GET /api/reviews/pending` every 3 seconds with a hardcoded `setInterval` and no cleanup. Improve this.

**Behavior**:
- Default poll interval: 5 seconds (reduced frequency)
- If no pending review is found for 5 consecutive polls, back off to 15 seconds
- When a pending review appears, reset to 5-second interval
- When the review panel is open (user is actively reviewing), stop polling
- Use `setTimeout` chain instead of `setInterval` so a slow response doesn't stack requests

**Files to modify**: `eigen/internal/server/ui/app.js`
**Dependencies**: None

---

## P3 — Strategic / Future

### ~~13. Module deprecation and removal lifecycle~~ ✅ Done

**Module**: `projection-engine`
**What**: Currently there's no way to deprecate or remove a spec module — only individual acceptance criteria can be marked `removed: true`. Add a lifecycle for modules themselves.

**Behavior**:
- Status field gains `deprecated` and `removed` values (currently: draft, approved, compiled)
- A change can set `status: deprecated` with a reason explaining what replaces it
- A deprecated module shows a visual indicator in `eigen serve` and `eigen spec list`
- A removed module is hidden from default views but preserved in history
- `eigen spec validate` warns if a non-deprecated module depends on a deprecated one

**Open questions**: Whether removed modules should be physically deleted or just hidden. This item needs more design.
