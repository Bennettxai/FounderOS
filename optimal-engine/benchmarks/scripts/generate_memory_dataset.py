#!/usr/bin/env python3
"""Generate deterministic large-scale memory benchmark fixtures.

The generated data is synthetic but intentionally shaped like institutional
memory: decisions, reasons, owners, open questions, and supersessions spread
across many conversation-scoped records.
"""

from __future__ import annotations

import argparse
import json
import random
from pathlib import Path
from typing import Any


PROJECTS = [
    "Atlas", "Beacon", "Cipher", "Delta", "Evergreen", "Falcon", "Harbor", "Ion",
    "Juniper", "Keystone", "Lumen", "Meridian", "Nimbus", "Orchid", "Prairie",
    "Quartz", "Ranger", "Solstice", "Tundra", "Vector", "Willow", "Zenith",
]

DOMAINS = [
    "customer portal pricing", "identity migration", "partner onboarding", "billing retry",
    "clinical export", "data retention", "support escalation", "microVM rollout",
    "renewal pipeline", "analytics warehouse", "security review", "workflow capture",
]

OWNERS = [
    "Alice", "Bob", "Chandra", "Devon", "Eli", "Fatima", "Grace", "Hiro",
    "Iris", "Jules", "Kai", "Lina", "Maya", "Noah", "Omar", "Priya",
]

DECISIONS = [
    "standardize on annual price protection",
    "route enterprise exceptions through the platform review",
    "require source-linked evidence before promotion",
    "ship the migration in two phases",
    "keep the workflow in planning mode until validation passes",
    "use the wiki-first retrieval path for stable decisions",
    "treat unresolved observations as pending claims",
    "refresh stale context before agent execution",
]

REASONS = [
    "the previous process created stale handoffs",
    "the customer needed an auditable evidence trail",
    "support tickets showed repeated confusion",
    "the cost model stayed inside the approved margin",
    "security wanted partition checks before summarization",
    "the team needed a rollback path for unsafe changes",
    "the pilot showed lower latency with scoped retrieval",
    "the reviewer required bitemporal validity before reuse",
]

OPEN_QUESTIONS = [
    "whether strategic accounts get a deeper discount floor",
    "which reviewer owns exception approval",
    "when stale observations should expire from the active pool",
    "how much recursive detail agents may load by default",
    "whether local-node sync should require human review",
    "which workflow traces are strong enough for skill promotion",
]


def token_filler(rng: random.Random, approx_tokens: int) -> str:
    if approx_tokens <= 0:
        return ""

    words = [
        "context", "source", "claim", "fact", "memory", "workflow", "review",
        "evidence", "validity", "retrieval", "policy", "audit", "owner", "scope",
    ]
    return " ".join(rng.choice(words) for _ in range(approx_tokens))


def conversation_record(index: int, args: argparse.Namespace, rng: random.Random) -> dict[str, Any]:
    project = PROJECTS[index % len(PROJECTS)]
    domain = DOMAINS[index % len(DOMAINS)]
    owner = OWNERS[index % len(OWNERS)]
    backup_owner = OWNERS[(index + 5) % len(OWNERS)]
    decision = DECISIONS[index % len(DECISIONS)]
    reason = REASONS[index % len(REASONS)]
    open_question = OPEN_QUESTIONS[index % len(OPEN_QUESTIONS)]
    price = 1200 + (index % 9) * 175
    year = 2026 + (index % 3)
    quarter = f"Q{(index % 4) + 1}"
    slug = f"bench-{args.name}-conv-{index:04d}"
    conv_id = f"{args.name}-conv-{index:04d}"

    summary = (
        f"Conversation {conv_id}: Project {project} handled {domain}. "
        f"Decision: {decision}. Owner: {owner}. Backup owner: {backup_owner}. "
        f"Budget anchor: ${price} per account for {quarter} {year}. "
        f"Reason: {reason}. Open question: {open_question}. "
        f"Validity: current for {quarter} {year} unless superseded. "
        f"Source package: synthetic transcript {conv_id}. "
        f"Claim status: reviewed. Fact precision: project-level and quarter-level."
    )

    filler = token_filler(rng, args.filler_tokens)
    content = summary if not filler else f"{summary}\n\nBackground transcript filler:\n{filler}"

    return {
        "conversation_id": conv_id,
        "workspace": args.workspace,
        "slug": slug,
        "project": project,
        "domain": domain,
        "owner": owner,
        "backup_owner": backup_owner,
        "decision": decision,
        "reason": reason,
        "open_question": open_question,
        "price": price,
        "quarter": quarter,
        "year": year,
        "content": content,
    }


def questions_for(record: dict[str, Any], per_conversation: int) -> list[dict[str, Any]]:
    conv = record["conversation_id"]
    project = record["project"]
    domain = record["domain"]
    base = [
        {
            "id": f"{conv}-decision",
            "category": "decision",
            "question": f"What decision did Project {project} make for {domain}?",
            "gold": f"Project {project} decided to {record['decision']}.",
        },
        {
            "id": f"{conv}-owner",
            "category": "owner",
            "question": f"Who owns Project {project}'s {domain} work?",
            "gold": f"{record['owner']} owns it, with {record['backup_owner']} as backup owner.",
        },
        {
            "id": f"{conv}-reason",
            "category": "why",
            "question": f"Why did Project {project} make the {domain} decision?",
            "gold": f"The reason was that {record['reason']}.",
        },
        {
            "id": f"{conv}-budget",
            "category": "numeric",
            "question": f"What budget anchor did Project {project} record for {domain}?",
            "gold": (
                f"The budget anchor was ${record['price']} per account for "
                f"{record['quarter']} {record['year']}."
            ),
        },
        {
            "id": f"{conv}-open-question",
            "category": "open-question",
            "question": f"What open question remains for Project {project}'s {domain} work?",
            "gold": f"The open question is {record['open_question']}.",
        },
    ]

    variants: list[dict[str, Any]] = []
    variant_index = 0

    while len(variants) < per_conversation:
        for question in base:
            if len(variants) >= per_conversation:
                break

            cloned = dict(question)
            if variant_index > 0:
                cloned["id"] = f"{cloned['id']}-v{variant_index + 1}"
                cloned["question"] = alternate_question(cloned["category"], record, variant_index)
            variants.append(cloned)
        variant_index += 1

    selected = variants
    for question in selected:
        question["workspace"] = record["workspace"]
        question["conversation_id"] = conv
        question["slug"] = record["slug"]
        question["retrieval_query"] = f"Project {project} {domain}"
    return selected


def alternate_question(category: str, record: dict[str, Any], variant_index: int) -> str:
    project = record["project"]
    domain = record["domain"]
    variants = {
        "decision": [
            f"Recall the decision for Project {project} in {domain}.",
            f"What was settled for {domain} on Project {project}?",
            f"Which decision should I remember about Project {project}'s {domain} work?",
        ],
        "owner": [
            f"Who is responsible for the {domain} memory in Project {project}?",
            f"Name the owner and backup for Project {project}'s {domain} work.",
            f"Which people are attached to Project {project}'s {domain} ownership?",
        ],
        "why": [
            f"What rationale supports Project {project}'s {domain} decision?",
            f"Explain why Project {project} handled {domain} that way.",
            f"What reason was recorded for the {domain} decision in Project {project}?",
        ],
        "numeric": [
            f"Which dollar anchor is recorded for Project {project}'s {domain} plan?",
            f"Give the budget number and time period for Project {project}'s {domain}.",
            f"What price-like numeric fact belongs to Project {project}'s {domain} memory?",
        ],
        "open-question": [
            f"What unresolved issue remains for Project {project}'s {domain} work?",
            f"Which question is still pending in Project {project}'s {domain} memory?",
            f"What follow-up question should reviewers ask about Project {project}'s {domain}?",
        ],
    }

    choices = variants.get(category, [f"What should I remember about Project {project}'s {domain} work?"])
    return choices[(variant_index - 1) % len(choices)]


def write_jsonl(path: Path, rows: list[dict[str, Any]]) -> None:
    with path.open("w", encoding="utf-8") as handle:
        for row in rows:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--name", default="synthetic-500")
    parser.add_argument("--out-dir", required=True)
    parser.add_argument("--workspace", default="bench-large")
    parser.add_argument("--conversations", type=int, default=100)
    parser.add_argument("--questions-per-conversation", type=int, default=5)
    parser.add_argument("--filler-tokens", type=int, default=0)
    parser.add_argument("--seed", type=int, default=20260602)
    args = parser.parse_args()

    if args.questions_per_conversation < 1:
        raise ValueError("--questions-per-conversation must be positive")

    rng = random.Random(args.seed)
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    corpus = [conversation_record(index, args, rng) for index in range(args.conversations)]
    questions = [
        question
        for record in corpus
        for question in questions_for(record, args.questions_per_conversation)
    ]

    write_jsonl(out_dir / "corpus.jsonl", corpus)
    write_jsonl(out_dir / "questions.jsonl", questions)

    manifest = {
        "name": args.name,
        "workspace": args.workspace,
        "conversations": len(corpus),
        "questions": len(questions),
        "questions_per_conversation": args.questions_per_conversation,
        "filler_tokens_per_conversation": args.filler_tokens,
        "approx_total_filler_tokens": args.filler_tokens * len(corpus),
        "corpus": "corpus.jsonl",
        "questions_file": "questions.jsonl",
        "categories": sorted({question["category"] for question in questions}),
    }

    (out_dir / "manifest.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
    print(json.dumps(manifest, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
