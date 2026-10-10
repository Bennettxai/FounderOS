#!/usr/bin/env python3
"""Benchmark Optimal Engine answers against JSONL question sets.

The harness intentionally uses only the Python standard library so it can run
in fresh CI and local shells without dependency setup.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.parse
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


VALID_LABELS = {"correct", "partial", "incorrect"}


def decode_retry_after_ms(detail: str, default_ms: int) -> int:
    try:
        data = json.loads(detail)
        value = int(data.get("retry_after_ms", default_ms))
        return max(value, default_ms)
    except Exception:
        return default_ms


def read_json_response(
    request: urllib.request.Request, url: str, retries: int, retry_sleep_ms: int
) -> dict[str, Any]:
    for attempt in range(retries + 1):
        try:
            with urllib.request.urlopen(request, timeout=120) as response:
                raw = response.read().decode("utf-8")
                return json.loads(raw) if raw else {}
        except urllib.error.HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")
            if error.code == 429 and attempt < retries:
                time.sleep(decode_retry_after_ms(detail, retry_sleep_ms) / 1000)
                continue
            raise RuntimeError(f"HTTP {error.code} from {url}: {detail}") from error

    raise RuntimeError(f"exhausted retries for {url}")


def post_json(
    url: str,
    payload: dict[str, Any],
    headers: dict[str, str] | None = None,
    retries: int = 0,
    retry_sleep_ms: int = 250,
) -> dict[str, Any]:
    body = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        url,
        data=body,
        headers={"Content-Type": "application/json", **(headers or {})},
        method="POST",
    )

    return read_json_response(request, url, retries, retry_sleep_ms)


def get_json(
    url: str,
    params: dict[str, Any],
    retries: int = 0,
    retry_sleep_ms: int = 250,
) -> dict[str, Any]:
    encoded = urllib.parse.urlencode(params)
    full_url = f"{url}?{encoded}"
    request = urllib.request.Request(full_url, method="GET")

    return read_json_response(request, full_url, retries, retry_sleep_ms)


def load_questions(path: Path, limit: int | None) -> list[dict[str, Any]]:
    questions: list[dict[str, Any]] = []

    with path.open("r", encoding="utf-8") as handle:
        for line_no, line in enumerate(handle, start=1):
            stripped = line.strip()
            if not stripped or stripped.startswith("#"):
                continue

            row = json.loads(stripped)
            for field in ("id", "question", "gold"):
                if not row.get(field):
                    raise ValueError(f"{path}:{line_no} missing required field {field!r}")

            questions.append(row)
            if limit and len(questions) >= limit:
                break

    return questions


def ask_engine(args: argparse.Namespace, query: str, workspace: str) -> tuple[str, dict[str, Any], int]:
    started = time.monotonic()
    engine_url = args.engine_url.rstrip("/")

    if args.engine_mode == "rag":
        payload: dict[str, Any] = {
            "query": query,
            "workspace": workspace,
            "format": "markdown",
        }

        if args.rag_skip_intent:
            payload["skip_intent"] = True
        if args.rag_skip_wiki:
            payload["skip_wiki"] = True
        if args.hybrid_limit is not None:
            payload["hybrid_limit"] = args.hybrid_limit
        if args.memory_limit is not None:
            payload["memory_limit"] = args.memory_limit

        response = post_json(
            f"{engine_url}/api/rag",
            payload,
            retries=args.http_retries,
            retry_sleep_ms=args.retry_sleep_ms,
        )
        envelope = response.get("envelope") or {}
        answer = (
            envelope.get("body")
            or response.get("answer")
            or response.get("context")
            or response.get("body")
            or ""
        )
    elif args.engine_mode == "memory-search":
        response = get_json(
            f"{engine_url}/api/memory",
            {
                "workspace": workspace,
                "q": query,
                "limit": args.retrieval_limit,
            },
            retries=args.http_retries,
            retry_sleep_ms=args.retry_sleep_ms,
        )
        memories = response.get("memories") or []
        answer = "\n\n".join(
            f"[{index + 1}] {memory.get('content', '')}"
            for index, memory in enumerate(memories)
        )
        response = {
            "source": "memory",
            "trace": {
                "n_candidates": len(memories),
                "n_delivered": len(memories),
                "wiki_hit?": False,
                "memory_search?": True,
            },
            "raw": response,
        }
    else:
        raise ValueError(f"unsupported engine mode {args.engine_mode}")

    elapsed_ms = int((time.monotonic() - started) * 1000)

    return str(answer), response, elapsed_ms


def judge_prompt(question: str, gold: str, answer: str) -> str:
    return f"""You are grading a memory retrieval benchmark.

Return only compact JSON with these keys:
- label: one of "correct", "partial", "incorrect"
- reason: one short sentence

Grade by semantic factual match, not wording. Use:
- correct: the answer contains the key facts needed by the gold answer.
- partial: the answer has some relevant facts but misses an important fact or is ambiguous.
- incorrect: the answer is wrong, unsupported, or does not answer the question.

Question:
{question}

Gold answer:
{gold}

Candidate answer:
{answer}
"""


def parse_judge_json(text: str) -> dict[str, str]:
    cleaned = text.strip()

    if cleaned.startswith("```"):
        cleaned = cleaned.strip("`")
        if cleaned.startswith("json"):
            cleaned = cleaned[4:].strip()

    start = cleaned.find("{")
    end = cleaned.rfind("}")
    if start != -1 and end != -1 and end > start:
        cleaned = cleaned[start : end + 1]

    data = json.loads(cleaned)
    label = str(data.get("label", "")).strip().lower()
    if label not in VALID_LABELS:
        raise ValueError(f"invalid judge label {label!r} in {text!r}")

    return {
        "label": label,
        "reason": str(data.get("reason", "")).strip(),
    }


def judge_with_ollama(args: argparse.Namespace, question: str, gold: str, answer: str) -> dict[str, str]:
    api_key = os.environ.get("OLLAMA_API_KEY")
    if not api_key and args.judge_base_url.startswith("https://ollama.com"):
        raise RuntimeError("OLLAMA_API_KEY is required for Ollama Cloud judging")

    headers = {}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    response = post_json(
        f"{args.judge_base_url.rstrip('/')}/chat",
        {
            "model": args.judge_model,
            "messages": [{"role": "user", "content": judge_prompt(question, gold, answer)}],
            "stream": False,
            "format": "json",
            "options": {"temperature": args.judge_temperature},
        },
        headers=headers,
        retries=args.http_retries,
        retry_sleep_ms=args.retry_sleep_ms,
    )

    content = ((response.get("message") or {}).get("content") or "").strip()
    return parse_judge_json(content)


def judge_with_openai_compatible(
    args: argparse.Namespace, question: str, gold: str, answer: str
) -> dict[str, str]:
    api_key = os.environ.get("JUDGE_API_KEY") or os.environ.get("OPENAI_API_KEY")
    if not api_key:
        raise RuntimeError("JUDGE_API_KEY or OPENAI_API_KEY is required for openai-compatible judging")

    response = post_json(
        f"{args.judge_base_url.rstrip('/')}/chat/completions",
        {
            "model": args.judge_model,
            "temperature": args.judge_temperature,
            "response_format": {"type": "json_object"},
            "messages": [{"role": "user", "content": judge_prompt(question, gold, answer)}],
        },
        headers={"Authorization": f"Bearer {api_key}"},
        retries=args.http_retries,
        retry_sleep_ms=args.retry_sleep_ms,
    )

    content = response["choices"][0]["message"]["content"]
    return parse_judge_json(content)


def majority_label(votes: list[dict[str, str]]) -> str:
    if not votes:
        return "unjudged"

    counts = {label: 0 for label in VALID_LABELS}
    for vote in votes:
        counts[vote["label"]] += 1

    # Tie-break conservatively.
    return max(("incorrect", "partial", "correct"), key=lambda label: (counts[label], label))


def judge_answer(args: argparse.Namespace, question: str, gold: str, answer: str) -> list[dict[str, str]]:
    if args.judge == "none":
        return []

    votes = []
    for _ in range(args.judge_votes):
        if args.judge == "ollama":
            votes.append(judge_with_ollama(args, question, gold, answer))
        elif args.judge == "openai-compatible":
            votes.append(judge_with_openai_compatible(args, question, gold, answer))
        else:
            raise ValueError(f"unsupported judge {args.judge}")
    return votes


def run(args: argparse.Namespace) -> int:
    questions = load_questions(Path(args.questions), args.limit)
    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    totals = {"correct": 0, "partial": 0, "incorrect": 0, "unjudged": 0}

    with out_path.open("w", encoding="utf-8") as output:
        for index, row in enumerate(questions, start=1):
            workspace = row.get("workspace") or args.workspace
            engine_query = str(row.get(args.query_field) or row["question"])
            error = None
            try:
                answer, raw_response, elapsed_ms = ask_engine(args, engine_query, workspace)
            except Exception as exc:
                answer = ""
                raw_response = {}
                elapsed_ms = 0
                votes = []
                label = "incorrect"
                error = f"engine: {exc}"
            else:
                try:
                    votes = judge_answer(args, row["question"], row["gold"], answer)
                    label = majority_label(votes)
                except Exception as exc:
                    label = "unjudged"
                    error = f"judge: {exc}"

            totals[label] += 1

            result = {
                "id": row["id"],
                "category": row.get("category"),
                "workspace": workspace,
                "question": row["question"],
                "engine_query": engine_query,
                "gold": row["gold"],
                "answer": answer,
                "label": label,
                "votes": votes,
                "engine_latency_ms": elapsed_ms,
                "engine_trace": raw_response.get("trace"),
                "engine_source": raw_response.get("source"),
                "error": error,
            }

            output.write(json.dumps(result, ensure_ascii=False) + "\n")
            output.flush()
            print(f"[{index}/{len(questions)}] {row['id']} {label} {elapsed_ms}ms", file=sys.stderr)
            if args.request_sleep_ms > 0:
                time.sleep(args.request_sleep_ms / 1000)

    judged = totals["correct"] + totals["partial"] + totals["incorrect"]
    accuracy = totals["correct"] / judged if judged else None
    summary = {"questions": len(questions), "totals": totals, "accuracy": accuracy, "out": str(out_path)}
    print(json.dumps(summary, indent=2))

    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--questions", required=True, help="JSONL question file")
    parser.add_argument("--out", required=True, help="JSONL result file")
    parser.add_argument("--engine-url", default="http://127.0.0.1:4200")
    parser.add_argument("--engine-mode", choices=["rag", "memory-search"], default="rag")
    parser.add_argument("--query-field", default="question")
    parser.add_argument("--retrieval-limit", type=int, default=5)
    parser.add_argument("--rag-skip-intent", action="store_true")
    parser.add_argument("--rag-skip-wiki", action="store_true")
    parser.add_argument("--hybrid-limit", type=int)
    parser.add_argument("--memory-limit", type=int)
    parser.add_argument("--http-retries", type=int, default=3)
    parser.add_argument("--retry-sleep-ms", type=int, default=250)
    parser.add_argument("--request-sleep-ms", type=int, default=0)
    parser.add_argument("--workspace", default="default")
    parser.add_argument("--limit", type=int)
    parser.add_argument("--judge", choices=["none", "ollama", "openai-compatible"], default="none")
    parser.add_argument("--judge-model", default="gpt-oss:120b")
    parser.add_argument("--judge-base-url", default="https://ollama.com/api")
    parser.add_argument("--judge-votes", type=int, default=3)
    parser.add_argument("--judge-temperature", type=float, default=0)
    args = parser.parse_args()

    if args.judge_votes < 1:
        parser.error("--judge-votes must be at least 1")

    return run(args)


if __name__ == "__main__":
    raise SystemExit(main())
