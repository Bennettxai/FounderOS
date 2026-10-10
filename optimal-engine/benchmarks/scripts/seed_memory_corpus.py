#!/usr/bin/env python3
"""Seed a generated benchmark corpus into Optimal Engine.

Each corpus row is inserted as a versioned memory and optionally promoted to a
wiki page. Promotion is useful because the current RAG path is wiki-first for
stable institutional-memory answers.
"""

from __future__ import annotations

import argparse
import json
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


def post_json(url: str, payload: dict[str, Any]) -> dict[str, Any]:
    body = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        url,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(request, timeout=120) as response:
            raw = response.read().decode("utf-8")
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as error:
        detail = error.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {error.code} from {url}: {detail}") from error


def load_jsonl(path: Path, limit: int | None) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            stripped = line.strip()
            if stripped:
                rows.append(json.loads(stripped))
                if limit and len(rows) >= limit:
                    break
    return rows


def seed_row(args: argparse.Namespace, row: dict[str, Any]) -> dict[str, Any]:
    workspace = row.get("workspace") or args.workspace
    metadata = {
        "benchmark": args.benchmark,
        "conversation_id": row["conversation_id"],
        "project": row["project"],
        "domain": row["domain"],
        "source": "benchmark_generator",
    }

    memory = post_json(
        f"{args.engine_url.rstrip('/')}/api/memory",
        {
            "workspace": workspace,
            "audience": args.audience,
            "content": row["content"],
            "metadata": metadata,
        },
    )

    memory_id = memory["id"]
    promoted = None

    if args.promote:
        promoted = post_json(
            f"{args.engine_url.rstrip('/')}/api/memory/{memory_id}/promote",
            {
                "workspace": workspace,
                "audience": args.audience,
                "slug": row["slug"],
            },
        )

    return {
        "conversation_id": row["conversation_id"],
        "workspace": workspace,
        "memory_id": memory_id,
        "slug": row["slug"],
        "promoted": bool(promoted),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--corpus", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--engine-url", default="http://127.0.0.1:4200")
    parser.add_argument("--workspace", default="bench-large")
    parser.add_argument("--audience", default="default")
    parser.add_argument("--benchmark", default="synthetic")
    parser.add_argument("--limit", type=int)
    parser.add_argument("--promote", action="store_true")
    parser.add_argument("--sleep-ms", type=int, default=0)
    args = parser.parse_args()

    rows = load_jsonl(Path(args.corpus), args.limit)
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    with out_path.open("w", encoding="utf-8") as output:
        for index, row in enumerate(rows, start=1):
            seeded = seed_row(args, row)
            output.write(json.dumps(seeded, ensure_ascii=False) + "\n")
            output.flush()
            print(f"[{index}/{len(rows)}] {seeded['conversation_id']} -> {seeded['memory_id']}")
            if args.sleep_ms > 0:
                time.sleep(args.sleep_ms / 1000)

    print(json.dumps({"seeded": len(rows), "out": str(out_path)}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
