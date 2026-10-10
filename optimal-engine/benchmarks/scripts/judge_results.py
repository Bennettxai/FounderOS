#!/usr/bin/env python3
"""Judge an existing Optimal Engine result JSONL without rerunning the engine."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any


def normalize(text: str) -> str:
    return re.sub(r"\s+", " ", text.lower()).strip()


def contains_phrase(answer: str, phrase: str) -> bool:
    return normalize(phrase) in normalize(answer)


def between(text: str, prefix: str, suffix: str = ".") -> str:
    start = text.find(prefix)
    if start == -1:
        return ""
    start += len(prefix)
    end = text.find(suffix, start)
    if end == -1:
        end = len(text)
    return text[start:end].strip()


def synthetic_exact_vote(row: dict[str, Any]) -> dict[str, str]:
    """Deterministic judge for generated synthetic memory datasets.

    The generated corpus stores each gold answer verbatim as structured memory
    fields. This judge checks those category-specific key facts rather than
    using fuzzy prose similarity.
    """

    category = row.get("category")
    gold = str(row.get("gold") or "")
    answer = str(row.get("answer") or "")

    if not answer or answer.startswith("_No results"):
        return {"label": "incorrect", "reason": "No answer was returned."}

    if category == "decision":
        expected = between(gold, "decided to ")
        ok = bool(expected) and contains_phrase(answer, expected)
    elif category == "owner":
        match = re.search(r"^(.+?) owns it, with (.+?) as backup owner\.", gold)
        ok = bool(match) and contains_phrase(answer, f"Owner: {match.group(1)}") and contains_phrase(
            answer, f"Backup owner: {match.group(2)}"
        )
    elif category == "why":
        expected = between(gold, "The reason was that ")
        ok = bool(expected) and contains_phrase(answer, expected)
    elif category == "numeric":
        match = re.search(r"(\$[0-9,]+ per account for Q[1-4] [0-9]{4})", gold)
        ok = bool(match) and contains_phrase(answer, match.group(1))
    elif category == "open-question":
        expected = between(gold, "The open question is ")
        ok = bool(expected) and contains_phrase(answer, expected)
    else:
        ok = contains_phrase(answer, gold)

    if ok:
        return {"label": "correct", "reason": "Answer contains the category-specific gold fact."}

    return {"label": "incorrect", "reason": "Answer is missing the category-specific gold fact."}


def majority_label(votes: list[dict[str, str]]) -> str:
    counts = {"correct": 0, "partial": 0, "incorrect": 0}
    for vote in votes:
        counts[vote["label"]] += 1
    return max(("incorrect", "partial", "correct"), key=lambda label: (counts[label], label))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--results", required=True, help="Existing result JSONL")
    parser.add_argument("--out", required=True, help="Judged result JSONL")
    parser.add_argument("--judge", choices=["synthetic-exact"], default="synthetic-exact")
    parser.add_argument("--judge-votes", type=int, default=1)
    args = parser.parse_args()

    if args.judge_votes < 1:
        parser.error("--judge-votes must be at least 1")

    in_path = Path(args.results)
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    totals = {"correct": 0, "partial": 0, "incorrect": 0, "unjudged": 0}
    count = 0

    with in_path.open("r", encoding="utf-8") as source, out_path.open("w", encoding="utf-8") as output:
        for line in source:
            if not line.strip():
                continue
            row = json.loads(line)
            votes = [synthetic_exact_vote(row) for _ in range(args.judge_votes)]
            label = majority_label(votes)
            row["votes"] = votes
            row["label"] = label
            row["error"] = row.get("error")
            output.write(json.dumps(row, ensure_ascii=False) + "\n")
            totals[label] += 1
            count += 1

    accuracy = totals["correct"] / (totals["correct"] + totals["partial"] + totals["incorrect"])
    print(json.dumps({"questions": count, "totals": totals, "accuracy": accuracy, "out": str(out_path)}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
