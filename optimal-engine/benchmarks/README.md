# Optimal Engine Benchmarks

This directory contains lightweight benchmark harnesses for measuring Optimal
Engine retrieval quality against question-answer datasets.

The first harness is model-provider agnostic: it calls the local Optimal Engine
API for answers, then optionally asks a separate judge model to grade the answer
against a gold answer.

## Local Smoke Test

Start the engine first:

```bash
iex -S mix
```

Then run the sample questions without a model judge:

```bash
python3 benchmarks/scripts/optimal_eval.py \
  --questions benchmarks/sample_questions.jsonl \
  --judge none \
  --out benchmarks/results/sample_unjudged.jsonl
```

## Ollama Cloud Judge

Ollama Cloud direct API access uses `https://ollama.com/api` and bearer-token
auth via `OLLAMA_API_KEY`.

```bash
export OLLAMA_API_KEY=...

python3 benchmarks/scripts/optimal_eval.py \
  --questions benchmarks/sample_questions.jsonl \
  --judge ollama \
  --judge-model gpt-oss:120b \
  --judge-votes 3 \
  --out benchmarks/results/sample_ollama.jsonl
```

Useful options:

- `--engine-url http://127.0.0.1:4200`
- `--workspace default`
- `--limit 20`
- `--judge-temperature 0`
- `--judge-base-url https://ollama.com/api`

## OpenAI-Compatible Judge

For any provider exposing `/v1/chat/completions`:

```bash
export JUDGE_API_KEY=...

python3 benchmarks/scripts/optimal_eval.py \
  --questions benchmarks/sample_questions.jsonl \
  --judge openai-compatible \
  --judge-base-url https://example.com/v1 \
  --judge-model some-model \
  --out benchmarks/results/sample_openai_compat.jsonl
```

## Question Format

Each line is JSON:

```json
{"id":"q1","workspace":"default","question":"...","gold":"...","category":"single-hop"}
```

`workspace` is optional; `--workspace` is used as the default.

## Output Format

Each result line includes:

- question metadata
- Optimal Engine answer and trace
- latency in milliseconds
- judge votes
- final label: `correct`, `partial`, `incorrect`, or `unjudged`

Never put provider API keys in question files or result files.

## Benchmark Card

Render a benchmark-card style summary from any result JSONL:

```bash
python3 benchmarks/scripts/summarize_results.py \
  --results benchmarks/results/sample_ollama_qwen35_397b.jsonl \
  --out benchmarks/results/sample_ollama_qwen35_397b.md \
  --title "Optimal Engine Sample Memory Results" \
  --dataset "Sample pricing memory, 3 Qs" \
  --answer-system "Optimal Engine /api/rag, wiki-first" \
  --judge "qwen3.5:397b via Ollama Cloud" \
  --judge-votes 3 \
  --retrieval-top-k "engine default; returned 1" \
  --compute "local engine + Ollama Cloud judge"
```

This produces a small public-facing report with evaluation config, aggregate
score, latency, retrieval source mix, and per-question vote splits.

## Large-Scale Memory Benchmarks

Generate a 500-question institutional-memory dataset:

```bash
python3 benchmarks/scripts/generate_memory_dataset.py \
  --name synthetic-500 \
  --out-dir benchmarks/generated/synthetic-500 \
  --workspace bench-synthetic-500 \
  --conversations 100 \
  --questions-per-conversation 5
```

Seed it into the running engine and promote each conversation to a wiki page:

```bash
python3 benchmarks/scripts/seed_memory_corpus.py \
  --corpus benchmarks/generated/synthetic-500/corpus.jsonl \
  --out benchmarks/generated/synthetic-500/seeded.jsonl \
  --workspace bench-synthetic-500 \
  --benchmark synthetic-500 \
  --promote
```

Run the benchmark:

```bash
python3 benchmarks/scripts/optimal_eval.py \
  --questions benchmarks/generated/synthetic-500/questions.jsonl \
  --workspace bench-synthetic-500 \
  --engine-mode memory-search \
  --query-field retrieval_query \
  --retrieval-limit 3 \
  --judge ollama \
  --judge-model qwen3.5:397b \
  --judge-votes 3 \
  --http-retries 6 \
  --retry-sleep-ms 300 \
  --out benchmarks/results/synthetic_500_memory_ollama_qwen35_397b.jsonl
```

Run the same questions through natural-language `/api/rag` while bypassing the
intent model for benchmark throughput:

```bash
python3 benchmarks/scripts/optimal_eval.py \
  --questions benchmarks/generated/synthetic-500/questions.jsonl \
  --workspace bench-synthetic-500 \
  --engine-mode rag \
  --rag-skip-intent \
  --memory-limit 5 \
  --judge none \
  --http-retries 6 \
  --retry-sleep-ms 300 \
  --out benchmarks/results/synthetic_500_rag_skip_intent_unjudged.jsonl
```

Then render the public card:

```bash
python3 benchmarks/scripts/summarize_results.py \
  --results benchmarks/results/synthetic_500_memory_ollama_qwen35_397b.jsonl \
  --out benchmarks/results/synthetic_500_memory_ollama_qwen35_397b.md \
  --title "Optimal Engine Synthetic-500 Memory Results" \
  --dataset "100 conversations, 500 questions" \
  --answer-system "Optimal Engine /api/memory FTS" \
  --judge "qwen3.5:397b via Ollama Cloud" \
  --judge-votes 3 \
  --retrieval-top-k 3 \
  --compute "local engine + Ollama Cloud judge"
```

The generated question rows include both a natural-language `question` and a
focused `retrieval_query`. Use `--query-field retrieval_query` when the goal is
to measure scoped memory retrieval. Use the default `/api/rag` mode separately
when testing natural-language routing and wiki-first recall. Use
`--rag-skip-intent` for large controlled runs where the question text already
contains the needed retrieval cues and the benchmark should measure retrieval
rather than intent-model latency.

For BEAM-like question volume, generate 35 conversations and 700 paraphrased
questions:

```bash
python3 benchmarks/scripts/generate_memory_dataset.py \
  --name synthetic-700 \
  --out-dir benchmarks/generated/synthetic-700 \
  --workspace bench-synthetic-700 \
  --conversations 35 \
  --questions-per-conversation 20
```

Long-context stress runs can add filler tokens per conversation. Treat these as
dedicated runs because corpus size, indexing time, latency, and judge cost grow
quickly:

```bash
python3 benchmarks/scripts/generate_memory_dataset.py \
  --name synthetic-1m-context \
  --out-dir benchmarks/generated/synthetic-1m-context \
  --workspace bench-synthetic-1m \
  --conversations 35 \
  --questions-per-conversation 20 \
  --filler-tokens 1000000
```
