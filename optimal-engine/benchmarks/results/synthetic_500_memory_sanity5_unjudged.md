# Optimal Engine Synthetic-500 Memory-Search Sanity Run

## Results

| Metric | Value |
|---|---:|
| Questions | 5 |
| Correct | 0 |
| Partial | 0 |
| Incorrect | 0 |
| Unjudged | 5 |
| Accuracy | n/a |
| Correct-or-partial | n/a |
| Judge votes correct | 0 |
| Judge votes partial | 0 |
| Judge votes incorrect | 0 |
| Engine errors | 0 |
| Wiki hits | 0/5 |
| Avg latency | 4 ms |
| Median latency | 2 ms |
| Min latency | 1 ms |
| Max latency | 13 ms |

## Evaluation Config

| Parameter | Value |
|---|---|
| Dataset | 100 seeded conversations, first 5 of 500 questions |
| Dataset size | 5 questions |
| Answer system | Optimal Engine /api/memory FTS |
| Answer temperature | n/a |
| Judge | none |
| Judge voting | 0x majority |
| Retrieval top-k | 3 |
| Compute | local engine |
| Result file | `benchmarks/results/synthetic_500_memory_sanity5_unjudged.jsonl` |

## Breakdown

| Group | Count |
|---|---:|
| category:decision | 1 |
| category:numeric | 1 |
| category:open-question | 1 |
| category:owner | 1 |
| category:why | 1 |
| source:memory | 5 |

## Per-Question

| ID | Category | Label | Vote split | Latency | Source | Candidates |
|---|---|---|---:|---:|---|---:|
| synthetic-500-conv-0000-decision | decision | unjudged | 0/0/0 | 13 ms | memory | 1 |
| synthetic-500-conv-0000-owner | owner | unjudged | 0/0/0 | 2 ms | memory | 1 |
| synthetic-500-conv-0000-reason | why | unjudged | 0/0/0 | 1 ms | memory | 1 |
| synthetic-500-conv-0000-budget | numeric | unjudged | 0/0/0 | 1 ms | memory | 1 |
| synthetic-500-conv-0000-open-question | open-question | unjudged | 0/0/0 | 2 ms | memory | 1 |
