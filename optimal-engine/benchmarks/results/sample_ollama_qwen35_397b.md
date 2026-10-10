# Optimal Engine Sample Memory Results

## Results

| Metric | Value |
|---|---:|
| Questions | 3 |
| Correct | 3 |
| Partial | 0 |
| Incorrect | 0 |
| Unjudged | 0 |
| Accuracy | 100.0% |
| Correct-or-partial | 100.0% |
| Judge votes correct | 9 |
| Judge votes partial | 0 |
| Judge votes incorrect | 0 |
| Engine errors | 0 |
| Wiki hits | 3/3 |
| Avg latency | 5849 ms |
| Median latency | 5518 ms |
| Min latency | 5440 ms |
| Max latency | 6589 ms |

## Evaluation Config

| Parameter | Value |
|---|---|
| Dataset | Sample pricing memory, 3 Qs |
| Dataset size | 3 questions |
| Answer system | Optimal Engine /api/rag, wiki-first |
| Answer temperature | n/a |
| Judge | qwen3.5:397b via Ollama Cloud |
| Judge voting | 3x majority |
| Retrieval top-k | engine default; returned 1 |
| Compute | local engine + Ollama Cloud judge |
| Result file | `benchmarks/results/sample_ollama_qwen35_397b.jsonl` |

## Breakdown

| Group | Count |
|---|---:|
| category:open-question | 1 |
| category:single-hop | 1 |
| category:why | 1 |
| source:wiki | 3 |

## Per-Question

| ID | Category | Label | Vote split | Latency | Source | Candidates |
|---|---|---|---:|---:|---|---:|
| sample-pricing-1 | single-hop | correct | 3/0/0 | 5440 ms | wiki | 1 |
| sample-pricing-2 | why | correct | 3/0/0 | 5518 ms | wiki | 1 |
| sample-pricing-3 | open-question | correct | 3/0/0 | 6589 ms | wiki | 1 |
