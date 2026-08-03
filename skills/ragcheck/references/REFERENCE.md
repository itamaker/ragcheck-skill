# ragcheck reference

Full CLI, JSON schema, and metric reference for the `ragcheck` binary. Load this only when you need field-level or formula-level detail beyond what `SKILL.md` covers.

## CLI

### `ragcheck score`

```
ragcheck score -qrels <path> -run <path> [-k N] [-json]
```

| Flag | Required | Default | Meaning |
| --- | --- | --- | --- |
| `-qrels` | yes | — | Path to qrels JSON |
| `-run` | yes | — | Path to retrieval run JSON |
| `-k` | no | `5` | Top-k cutoff for all metrics |
| `-json` | no | `false` | Emit machine-readable JSON instead of text |

Text output prints, in order: `Queries`, `Missing runs`, `Precision@k`, `Recall@k`, `HitRate@k`, `MRR@k`, `MAP@k`, `nDCG@k`. `-json` emits the same fields as a `Metrics` object: `queries`, `k`, `missing_runs`, `precision_at_k`, `recall_at_k`, `hit_rate_at_k`, `mrr_at_k`, `map_at_k`, `ndcg_at_k`.

### `ragcheck judge`

```
ragcheck judge -input <path> [-json]
```

| Flag | Required | Default | Meaning |
| --- | --- | --- | --- |
| `-input` | yes | — | Path to a judge input JSON file |
| `-json` | no | `false` | Emit machine-readable JSON instead of text |

Text output prints averaged `Queries`, `Answer relevance`, `Context relevance`, `Groundedness`, `Reference coverage`, then one line per query with per-query relevance/groundedness and any unsupported tokens. `-json` emits a `JudgeReport`: `queries`, `answer_relevance`, `context_relevance`, `groundedness`, `reference_coverage`, and a `results[]` array of per-query `JudgeResult` objects (`query_id`, `answer_relevance`, `context_relevance`, `groundedness`, `reference_coverage`, `unsupported_tokens[]`, `missing_context_hints[]`).

### No-argument / TUI

`ragcheck` (no args), `ragcheck tui`, and `ragcheck interactive` all launch the same Bubble Tea interactive UI. Any other unrecognized first argument prints usage to stderr and exits `2`.

### Exit codes

- `0`: success.
- `1`: runtime error (e.g. file not found, invalid JSON) — message printed to stderr.
- `2`: usage error (missing required flag, unknown subcommand, flag parse failure) — message printed to stderr.

## JSON schemas

### qrels file (`-qrels`)

Array of query relevance judgments. Each entry supports either binary relevance (`relevant`) or graded relevance (`grades`) — if both are omitted the query has no relevant documents.

```json
[
  { "query_id": "q1", "relevant": ["doc-1", "doc-3"] },
  { "query_id": "q2", "grades": { "doc-4": 2, "doc-9": 1 } }
]
```

- `query_id` (string, required): joins against the run file's `query_id`.
- `relevant` (string[], optional): document IDs treated as relevance grade `1`.
- `grades` (object, optional): document ID → numeric grade (used directly for graded nDCG; any grade `> 0` counts as relevant for Precision/Recall/HitRate/MRR/MAP). Takes precedence over `relevant` when both are present.

### run file (`-run`)

Array of retrieval results, one per query, ranked best-first.

```json
[
  { "query_id": "q1", "results": ["doc-1", "doc-2", "doc-9"] }
]
```

- `query_id` (string, required): should match a qrels entry; queries in `-run` with no corresponding qrels entry are ignored, and qrels entries with no matching run count toward `missing_runs`.
- `results` (string[], required): ranked document IDs, best match first. Only the first `k` are considered.

### judge input file (`-input`)

Array of RAG records to judge.

```json
[
  {
    "query_id": "q1",
    "question": "What caused the retrieval latency spike?",
    "answer": "The latency spike came from cache misses and slower vector search.",
    "reference": "Cache misses and slow vector search caused the retrieval latency spike.",
    "contexts": [
      "The incident review says cache misses increased tail latency during the rollout.",
      "A slower vector search path also contributed to the retrieval latency spike."
    ]
  }
]
```

- `query_id` (string, required)
- `question` (string, required)
- `answer` (string, required): the generated answer being judged.
- `reference` (string, optional): a gold/reference answer, used to compute `reference_coverage`. If omitted, `reference_coverage` is `0`.
- `contexts` (string[], required): retrieved passages the answer should be grounded in.

## Metric definitions

### Retrieval metrics (`score`), computed per query then averaged

All metrics only look at the top `k` entries of `results`.

- **Precision@k**: `hits / k`, where `hits` = count of top-k results with grade `> 0`.
- **Recall@k**: `hits / relevantCount`, where `relevantCount` = total documents with grade `> 0` in qrels (0 if none — query excluded from the average's numerator contribution).
- **HitRate@k**: `1` if `hits > 0` else `0`, averaged across queries.
- **MRR@k**: `1 / rank` of the first relevant hit within top-k (0 if no hit).
- **MAP@k**: average precision within top-k — sum of `(running hit count / rank)` at each relevant hit, divided by `min(relevantCount, k)`.
- **nDCG@k**: `DCG@k / IDCG@k` using gain `2^grade - 1` and discount `log2(rank + 1)`; `IDCG@k` uses qrels grades sorted descending as the ideal ranking.
- **missing_runs**: count of qrels entries whose `query_id` has no corresponding entry in the run file (their `hits` are treated as 0 for all metrics above).

### Judge metrics (`judge`), computed per record then averaged

All four scores are token-overlap ratios: tokenize both sides (lowercase, letters/digits only, tokens shorter than 3 chars dropped, a small stopword list removed), then `matched-unique-source-tokens / unique-source-tokens`.

- **answer_relevance**: overlap of `answer` tokens against `question` tokens.
- **context_relevance**: overlap of joined `contexts` tokens against `question` tokens.
- **groundedness**: overlap of `answer` tokens against joined `contexts` tokens.
- **reference_coverage**: overlap of `answer` tokens against `reference` tokens (`0` if `reference` is empty/omitted).
- **unsupported_tokens**: up to 5 sorted `answer` tokens not found in `contexts` tokens (answer content the contexts don't back up).
- **missing_context_hints**: up to 5 sorted `reference` tokens not found in `contexts` tokens (reference content the contexts don't cover).

This is a lightweight, dependency-free heuristic judge (no LLM call) — treat its scores as a fast offline signal, not a substitute for human or LLM-graded evaluation.
