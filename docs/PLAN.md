# Development Rules

- **Testing Runner (`gotest`):** For Go tests, ALWAYS use the globally installed `gotest` CLI command.
- **`gotest` in Agent Plans:** You MUST include `go install github.com/tinywasm/devflow/cmd/gotest@latest` as the first prerequisite in your plan.
- **Strict File Structure:** Flat Hierarchy for Go libraries. Files exceeding 500 lines MUST be subdivided.
- **Standard Library Only:** NEVER use external assertion libraries (e.g. `testify`). Use `testing`, `net/http/httptest`, and `reflect`.
- **Mocking:** Tests MUST use Mocks for all external interfaces. Keep tests fast and side-effect free.
- **Publishing:** ALWAYS use `gopush 'commit message'` CLI command to deploy, NEVER `git commit`/`git push`.

---

# ORM Pending Fixes Plan

The previous plan was mostly executed correctly by the external agent. However, some minor details in the documentation (`SKILL.md`) were missed and left outdated regarding the pluralization removal and terminology. Additionally, a test assertion in `tests/ormc_test.go` was failing due to the pluralization change (this test has already been fixed, but documentation updates remain).

## 1. Prerequisites
Before executing code logic, install the test runner globally:
```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## 2. Update Outdated References in `docs/SKILL.md`
**Goal:** Align the documentation precisely with the new logic changes.
* **File:** `docs/SKILL.md`
* **Action 1 (Pluralization):** Around line 131, the documentation still claims that `ormc` generates table names using pluralization.
  * From: `If absent, ormc generates it as the snake_case plural of the struct name.`
  * To: `If absent, ormc generates it as the snake_case of the struct name.`
* **Action 2 (Terminology):** Around line 145, the comments still use the word "Meta" instead of "_" descriptors.
  * From: `// 1. Where clauses use generated Meta descriptors (no magic strings)`
  * To: `// 1. Where clauses use generated _ descriptors (no magic strings)`

## 3. Verify Tests and Deploy
* Run `gotest` in the `tinywasm/orm` root to ensure tests remain green. 
*(Note: the failing string match assertion `Expected Ref=parent` in `ormc_test.go` was already patched).*
* Deploy the final documentation fixes utilizing the `gopush "docs: update SKILL.md pluralization and comments"` command.
