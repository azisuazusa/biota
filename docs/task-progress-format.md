# <Feature Name> — TODO

Reference: `docs/YYYY-MM-DD-<slug>/prd.md`

<One-sentence summary of what's being built.>

---

## Phase 1 — Database <✅ DONE | 🚧 IN PROGRESS | (blank = not started)>

- [ ] `path/to/migration.up.sql`: <exact schema change>
- [ ] `path/to/migration.down.sql`: <reverse>
- [ ] `path/to/entity.go`: <fields, constants, helpers>

---

## Phase 2 — Repository
## Phase 2 — Repository

- [ ] `path/to/repo.go`: <columns/queries to update>

---

## Phase 3 — Business logic / Usecase

- [ ] `path/to/util.go`: <helper> + tests in `path/to/util_test.go`
- [ ] `path/to/usecase.go`: <validation / rule>
- [ ] `path/to/usecase.go`: <edge case behaviour>
- [ ] `path/to/usecase_test.go`: tests for <case A>, <case B>, <case C>

---

## Phase 4 — <Bug fix / secondary change>

- [ ] `path/to/file`: <remove/change X>
- [ ] `path/to/wiring`: <update DI>
- [ ] `path/to/test`: <assert new behaviour>

---

## Phase 5 — UI

- [ ] `path/to/controller`: <parse fields, error logging>
- [ ] `path/to/view`: <control + helper text>
- [ ] `path/to/list view`: <badge/action> — <note if an existing feature already covers it>

---

## Phase 6 — Public / downstream surfaces

- [ ] `path/to/page`: <change>
- [ ] Confirm no change needed in `<other app>` (<reason>)

---

## Phase N — QA

- [ ] `<test command>` passes
- [ ] Verified <scenario 1> (environment: scratch DB / staging / browser)
- [ ] Verified <scenario 2>
- [ ] <Edge case flow> → <expected result>

Conventions these files follow

- Location: each feature gets a dated folder, docs/YYYY-MM-DD-slug/, holding prd.md (what and why) and todo.md (how, by phase).
- PRD: "In Scope" records each decision and the reason for it, for example "keeps the suffix so already-shared links keep working". "Out of Scope" lists things that were deliberately left out, not only things nobody thought of.
- TODO tasks: every task starts with the exact file path, followed by a colon and the change.
- Progress:
  - Tick [x] on each task as it's finished.
  - Add ✅ DONE to a phase heading once every task in it is ticked.
  - If you change a plan while implementing, write a short note on that task instead of deleting it, e.g. "copy link already exists as … so no new button".
- Phase order: follows the dependency chain: DB → repo → logic → fix → UI → public surfaces → QA.
- QA phase: a list of behaviours you can check by hand, each mirroring a row in the PRD's success metrics or behaviour matrix.
