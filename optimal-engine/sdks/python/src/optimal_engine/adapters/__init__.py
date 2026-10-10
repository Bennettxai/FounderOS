"""Optional adapters for popular agent frameworks.

The adapter modules import their respective frameworks lazily so the
core SDK has zero hard dependencies on OpenAI Agents or LangChain. Install
the corresponding extras to use them::

    pip install "optimal-engine[openai]"
    pip install "optimal-engine[langchain]"
"""

from __future__ import annotations

__all__: list[str] = []
