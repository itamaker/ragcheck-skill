---
name: ragcheck
description: Evaluate retrieval and RAG (retrieval-augmented generation) runs offline with the ragcheck Go CLI. Use when scoring a retrieval run against qrels to get Precision@k, Recall@k, HitRate@k, MRR@k, MAP@k, and nDCG@k; when running the offline answer judge to check answer relevance, context relevance, groundedness, or reference coverage for RAG outputs; when comparing two or more retrieval configs by re-scoring different run files against the same qrels; or when building qrels/run/judge JSON input files from scratch. Trigger for requests like "score this retrieval run", "what's the Precision@5 and nDCG@5 here", "judge these RAG answers for groundedness", "check if this answer is grounded in its context", "compare these two retriever configs", or "evaluate retrieval quality offline without a Python notebook".
---

# ragcheck

`ragcheck` is a Go CLI that scores retrieval runs against qrels (query relevance judgments) and judges RAG answer quality offline — no notebooks, no external eval service, no network calls. It ships two direct subcommands (`score`, `judge`) plus an interactive Bubble Tea TUI when invoked with no arguments.

## Workflow

1. Confirm `ragcheck` is available. If not installed (`brew install itamaker/tap/ragcheck` or a GitHub release binary), and you're working inside this repo, build it locally with `go build -o ragcheck .` or run subcommands via `go run . <subcommand> ...`.

2. To score a retrieval run against ground-truth relevance judgments:
   - Ensure you have a qrels file (array of `{query_id, relevant[]}` or `{query_id, grades{doc_id: grade}}`) and a run file (array of `{query_id, results[]}`, ranked document IDs) that share `query_id` values.
   - Run `ragcheck score -qrels <qrels.json> -run <run.json> -k <N>` (`-k` defaults to 5).
   - Add `-json` for machine-readable output when you need to parse or diff the numbers programmatically.
   - Output includes: query count, missing-run count (qrels with no matching run), and Precision@k, Recall@k, HitRate@k, MRR@k, MAP@k, nDCG@k.

3. To judge RAG answer quality offline (no qrels needed):
   - Build a judge input file: an array of `{query_id, question, answer, contexts[], reference?}` records (`reference` is optional).
   - Run `ragcheck judge -input <judge.json>`, optionally with `-json`.
   - Output includes per-query and averaged answer relevance, context relevance, groundedness, and reference coverage, plus any unsupported answer tokens (answer content not backed by the retrieved contexts) and missing-context hints (reference content absent from contexts).

4. To compare retrieval configs (e.g. two chunking strategies or rerankers), run `ragcheck score` once per candidate `-run` file against the *same* `-qrels` file and the same `-k`, then diff the metrics — there is no built-in multi-run comparison mode, so run it multiple times and compare the printed/JSON numbers yourself.

5. Interactive/manual exploration: running `ragcheck` with no arguments (or `ragcheck tui` / `ragcheck interactive`) launches a Bubble Tea terminal UI with the same score/judge actions. This requires a real terminal — prefer the direct `score`/`judge` subcommands above when running from a non-interactive agent shell.

6. For the full qrels/run/judge JSON field reference, metric formulas, and CLI flag/exit-code details, see `references/REFERENCE.md`.

## Prompt Patterns

- "Score this retrieval run against the qrels at k=10."
- "What's the Precision@5 and nDCG@5 for this run?"
- "Judge these RAG answers for groundedness and relevance."
- "Compare these two retriever configs using the same qrels file."
- "Check whether this answer is grounded in its retrieved context."
- "Run ragcheck on my qrels.json and run.json."
- "Evaluate retrieval quality offline without spinning up Python eval tooling."

## Resources

- `examples/qrels.json`: sample qrels file (binary relevance via `relevant[]`).
- `examples/run.json`: sample retrieval run file matching the qrels' `query_id`s.
- `examples/judge.json`: sample judge input with question/answer/reference/contexts.
- `references/REFERENCE.md`: full JSON schemas, CLI flags, exit codes, and metric definitions.
