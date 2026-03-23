# Blueprint Research Gate

Blueprint work should not default to deep external research.

Klein-Harness keeps a small gated pre-blueprint stage:

- `researchMode: none`
- `researchMode: targeted`
- `researchMode: deep`

## Flow

```text
design question
  -> repo-local scan
  -> research gate
  -> research memo
  -> research summary
  -> draft blueprint
  -> conflict review
  -> final blueprint
```

## Artifacts

- memo: `.harness/research/<slug>.md`
- optional machine memo: `.harness/research/<slug>.json`
- index: `.harness/state/research-index.json`
- bounded hot summary: `.harness/state/research-summary.json`

Blueprint generation should consume repo-local scan output, the memo/summary, and conflict analysis instead of raw long external material when possible.

## Trigger guidance

Use `targeted` or `deep` when:

- external framework or protocol behavior matters
- upstream or official behavior may have changed
- repository context is insufficient
- multiple architecture options need comparison
- migration or rollout risk is material

Prefer `none` when repo-local scan is already enough to produce a reliable blueprint.
