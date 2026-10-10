#!/usr/bin/env python3
"""Render a compact benchmark card from Optimal Engine JSONL results."""

from __future__ import annotations

import argparse
import json
import statistics
from collections import Counter
from pathlib import Path
from typing import Any


def load_jsonl(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            stripped = line.strip()
            if stripped:
                rows.append(json.loads(stripped))
    return rows


def pct(numerator: int, denominator: int) -> str:
    if denominator == 0:
        return "n/a"
    return f"{(numerator / denominator) * 100:.1f}%"


def vote_counts(rows: list[dict[str, Any]]) -> Counter[str]:
    counts: Counter[str] = Counter()
    for row in rows:
        for vote in row.get("votes") or []:
            counts[str(vote.get("label", "unknown"))] += 1
    return counts


def latency_summary(rows: list[dict[str, Any]]) -> dict[str, int | str]:
    values = [
        row.get("engine_latency_ms")
        for row in rows
        if isinstance(row.get("engine_latency_ms"), (int, float)) and row.get("engine_latency_ms") > 0
    ]

    if not values:
        return {"avg": "n/a", "median": "n/a", "min": "n/a", "max": "n/a"}

    return {
        "avg": int(round(statistics.mean(values))),
        "median": int(round(statistics.median(values))),
        "min": int(min(values)),
        "max": int(max(values)),
    }


def numeric_trace_summary(rows: list[dict[str, Any]], key: str) -> str:
    values = [
        (row.get("engine_trace") or {}).get(key)
        for row in rows
        if isinstance((row.get("engine_trace") or {}).get(key), (int, float))
    ]

    if not values:
        return "n/a"

    return f"{statistics.mean(values):.2f}"


def render_markdown(args: argparse.Namespace, rows: list[dict[str, Any]]) -> str:
    labels = Counter(str(row.get("label", "unknown")) for row in rows)
    votes = vote_counts(rows)
    judged = labels["correct"] + labels["partial"] + labels["incorrect"]
    latencies = latency_summary(rows)
    sources = Counter(str(row.get("engine_source", "unknown")) for row in rows)
    categories = Counter(str(row.get("category", "unknown")) for row in rows)
    wiki_hits = sum(1 for row in rows if (row.get("engine_trace") or {}).get("wiki_hit?") is True)
    errors = sum(1 for row in rows if row.get("error"))

    lines = [
        f"# {args.title}",
        "",
        "## Results",
        "",
        "| Metric | Value |",
        "|---|---:|",
        f"| Questions | {len(rows)} |",
        f"| Correct | {labels['correct']} |",
        f"| Partial | {labels['partial']} |",
        f"| Incorrect | {labels['incorrect']} |",
        f"| Unjudged | {labels['unjudged']} |",
        f"| Accuracy | {pct(labels['correct'], judged)} |",
        f"| Correct-or-partial | {pct(labels['correct'] + labels['partial'], judged)} |",
        f"| Judge votes correct | {votes['correct']} |",
        f"| Judge votes partial | {votes['partial']} |",
        f"| Judge votes incorrect | {votes['incorrect']} |",
        f"| Engine errors | {errors} |",
        f"| Wiki hits | {wiki_hits}/{len(rows)} |",
        f"| Avg candidates | {numeric_trace_summary(rows, 'n_candidates')} |",
        f"| Avg delivered | {numeric_trace_summary(rows, 'n_delivered')} |",
        f"| Avg latency | {latencies['avg']} ms |",
        f"| Median latency | {latencies['median']} ms |",
        f"| Min latency | {latencies['min']} ms |",
        f"| Max latency | {latencies['max']} ms |",
        "",
        "## Evaluation Config",
        "",
        "| Parameter | Value |",
        "|---|---|",
        f"| Dataset | {args.dataset} |",
        f"| Dataset size | {len(rows)} questions |",
        f"| Answer system | {args.answer_system} |",
        f"| Answer temperature | {args.answer_temperature} |",
        f"| Judge | {args.judge} |",
        f"| Judge voting | {args.judge_votes}x majority |",
        f"| Retrieval top-k | {args.retrieval_top_k} |",
        f"| Compute | {args.compute} |",
        f"| Result file | `{args.results}` |",
        "",
        "## Breakdown",
        "",
        "| Group | Count |",
        "|---|---:|",
    ]

    for category, count in sorted(categories.items()):
        lines.append(f"| category:{category} | {count} |")
    for source, count in sorted(sources.items()):
        lines.append(f"| source:{source} | {count} |")

    lines.extend(
        [
            "",
            "## Per-Question",
            "",
            "| ID | Category | Label | Vote split | Latency | Source | Candidates |",
            "|---|---|---|---:|---:|---|---:|",
        ]
    )

    for row in rows:
        row_votes = Counter(str(vote.get("label", "unknown")) for vote in row.get("votes") or [])
        trace = row.get("engine_trace") or {}
        split = f"{row_votes['correct']}/{row_votes['partial']}/{row_votes['incorrect']}"
        lines.append(
            "| {id} | {category} | {label} | {split} | {latency} ms | {source} | {candidates} |".format(
                id=row.get("id", ""),
                category=row.get("category", ""),
                label=row.get("label", ""),
                split=split,
                latency=row.get("engine_latency_ms", ""),
                source=row.get("engine_source", ""),
                candidates=trace.get("n_candidates", ""),
            )
        )

    lines.append("")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--results", required=True, help="Path to result JSONL")
    parser.add_argument("--out", required=True, help="Path to markdown report")
    parser.add_argument("--title", default="Optimal Engine Benchmark Results")
    parser.add_argument("--dataset", default="local sample")
    parser.add_argument("--answer-system", default="Optimal Engine /api/rag")
    parser.add_argument("--answer-temperature", default="n/a")
    parser.add_argument("--judge", default="none")
    parser.add_argument("--judge-votes", default="0")
    parser.add_argument("--retrieval-top-k", default="engine default")
    parser.add_argument("--compute", default="local")
    args = parser.parse_args()

    rows = load_jsonl(Path(args.results))
    report = render_markdown(args, rows)
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(report, encoding="utf-8")
    print(report)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
