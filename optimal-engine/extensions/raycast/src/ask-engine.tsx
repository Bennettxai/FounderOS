import {
  Action,
  ActionPanel,
  Color,
  Detail,
  Icon,
  List,
  Toast,
  showToast,
  useNavigation,
} from "@raycast/api";
import React, { useState } from "react";
import { askEngine } from "./lib/client";
import { getPreferences } from "./lib/preferences";
import type { RagEnvelope, RagSource } from "./lib/types";

// ---------------------------------------------------------------------------
// Result detail view
// ---------------------------------------------------------------------------

interface ResultDetailProps {
  envelope: RagEnvelope;
  onFollowUp: (query: string) => void;
}

function ResultDetail({
  envelope,
  onFollowUp,
}: ResultDetailProps): React.ReactElement {
  const sourcesSection =
    envelope.sources.length > 0
      ? [
          "",
          "---",
          "## Sources",
          ...envelope.sources.map((s, i) => {
            const link = s.citation_uri ? ` — [source](${s.citation_uri})` : "";
            return `${i + 1}. **${s.slug}** (score: ${s.score.toFixed(3)})${link}\n   > ${s.snippet}`;
          }),
        ]
      : [];

  const markdown = [envelope.body, ...sourcesSection].join("\n");

  const sourcesText = envelope.sources
    .map(
      (s: RagSource) =>
        `${s.slug}${s.citation_uri ? ` — ${s.citation_uri}` : ""}`,
    )
    .join("\n");

  return (
    <Detail
      markdown={markdown}
      navigationTitle={`Answer: ${envelope.query}`}
      metadata={
        <Detail.Metadata>
          <Detail.Metadata.Label title="Query" text={envelope.query} />
          <Detail.Metadata.Label title="Workspace" text={envelope.workspace} />
          <Detail.Metadata.Separator />
          <Detail.Metadata.Label
            title="Sources"
            text={String(envelope.sources.length)}
          />
        </Detail.Metadata>
      }
      actions={
        <ActionPanel>
          <ActionPanel.Section title="Answer">
            <Action.CopyToClipboard
              title="Copy Answer"
              content={envelope.body}
              icon={Icon.Clipboard}
            />
            {envelope.sources.length > 0 && (
              <Action.CopyToClipboard
                title="Copy Sources"
                content={sourcesText}
                icon={Icon.Link}
                shortcut={{ modifiers: ["cmd"], key: "c" }}
              />
            )}
          </ActionPanel.Section>
          <ActionPanel.Section title="Follow-up">
            <Action
              title="Ask Follow-up"
              icon={Icon.ArrowRight}
              shortcut={{ modifiers: ["cmd"], key: "f" }}
              onAction={() => onFollowUp(envelope.query)}
            />
          </ActionPanel.Section>
          <ActionPanel.Section title="Sources">
            {envelope.sources
              .filter((s) => s.citation_uri !== null)
              .map((s) => (
                <Action.OpenInBrowser
                  key={s.id}
                  title={`Open: ${s.slug}`}
                  url={s.citation_uri as string}
                  icon={Icon.Globe}
                />
              ))}
          </ActionPanel.Section>
        </ActionPanel>
      }
    />
  );
}

// ---------------------------------------------------------------------------
// Main command
// ---------------------------------------------------------------------------

export default function AskEngine(): React.ReactElement {
  const { workspace } = getPreferences();
  const { push } = useNavigation();

  const [query, setQuery] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [history, setHistory] = useState<string[]>([]);

  async function handleSearch(q: string): Promise<void> {
    const trimmed = q.trim();
    if (!trimmed) return;

    setIsLoading(true);

    const loadingToast = await showToast({
      style: Toast.Style.Animated,
      title: "Asking engine…",
      message: trimmed,
    });

    const result = await askEngine({
      query: trimmed,
      workspace,
      format: "markdown",
      bandwidth: "medium",
    });

    setIsLoading(false);
    await loadingToast.hide();

    if (!result.ok) {
      await showToast({
        style: Toast.Style.Failure,
        title: "Engine query failed",
        message: result.error,
      });
      return;
    }

    setHistory((prev) => [trimmed, ...prev.slice(0, 9)]);

    push(
      <ResultDetail
        envelope={result.data}
        onFollowUp={(prev) => {
          setQuery(`${prev} → `);
        }}
      />,
    );
  }

  return (
    <List
      isLoading={isLoading}
      searchBarPlaceholder="Ask Optimal Engine anything…"
      searchText={query}
      onSearchTextChange={setQuery}
      actions={
        <ActionPanel>
          <Action
            title="Ask"
            icon={Icon.MagnifyingGlass}
            onAction={() => void handleSearch(query)}
          />
        </ActionPanel>
      }
    >
      {history.length === 0 && !isLoading ? (
        <List.EmptyView
          icon={Icon.SpeechBubble}
          title="Ask the engine"
          description="Type a question and press Enter — the engine will answer using workspace context"
        />
      ) : (
        <List.Section title="Recent Questions">
          {history.map((q, i) => (
            <List.Item
              key={`${i}-${q}`}
              icon={{ source: Icon.Clock, tintColor: Color.SecondaryText }}
              title={q}
              actions={
                <ActionPanel>
                  <Action
                    title="Ask Again"
                    icon={Icon.ArrowClockwise}
                    onAction={() => void handleSearch(q)}
                  />
                  <Action
                    title="Edit Query"
                    icon={Icon.Pencil}
                    onAction={() => setQuery(q)}
                    shortcut={{ modifiers: ["cmd"], key: "e" }}
                  />
                </ActionPanel>
              }
            />
          ))}
        </List.Section>
      )}
    </List>
  );
}
