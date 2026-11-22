# PRD: <Feature Name>

**Date:** YYYY-MM-DD
**Status:** Draft | Approved | Implemented

---

## 1. Problem Statement

<What's broken or missing today, and why. Cite current behaviour concretely (columns, flags, file:line references).>

<Optional second paragraph: a related bug/issue being fixed as part of this work, and why it's bundled.>

---

## 2. Why This Matters

<Consequence of not doing this: workarounds users are forced into, data/money lost, etc.>

---

## 3. Who Is This For

**Primary:** <main user/role + where they act, e.g. route/screen>
**Secondary:** <other affected users>

---

## 4. What Success Looks Like

| Metric | Baseline | Target | Timeframe |
|--------|----------|--------|-----------|
| <observable outcome> | <today / N/A> | <target value> | At launch |
| <...> | <...> | <...> | <...> |

---

## 5. Scope

### In Scope

**<Sub-area 1>**
- <Data model change: new column/field, defaults, allowed values>
- <Behaviour matrix if multiple states interact:>

  | | <State A> | <State B> | <State C> |
  |---|---|---|---|
  | <Surface 1> | ... | ... | ... |
  | <Surface 2> | ... | ... | ... |

- <Access/permission model, and explicitly what it is NOT>
- <Which existing code paths change, and which need no change (and why)>

**<Sub-area 2>**
- <Rules, with the reasoning behind non-obvious choices>
- <Edge cases: edits, transitions back, collisions, retries>

**<UI / Admin>**
- <Form controls, badges, buttons; reuse of existing components/scripts>

**<Bug fix (if bundled)>**
- <Exact change + what stays unchanged>

### Out of Scope
- <Feature explicitly not built>
- <Backfill/migration of old data (decided: no/yes)>
- <Adjacent concepts that don't exist yet>

---

## 6. Constraints & Dependencies

- <Migration tooling / numbering conventions>
- <Existing checks that stay as-is and why the new feature passes them>
- <Validation rules that must still hold>
- <Architectural conventions to respect (interfaces, layering)>

---

## 7. Related Docs

| Document | Purpose |
|----------|---------|
| `docs/<...>/` | <why it's relevant> |

---

## 8. Open Questions

- <Question> — or "None at this time."
