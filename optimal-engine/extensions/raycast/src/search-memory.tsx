import {
  Action,
  ActionPanel,
  Color,
  Detail,
  Icon,
  List,
  Toast,
  showToast,
} from "@raycast/api";
import React, { useEffect, useRef, useState } from "react";
import { searchMemory } from "./lib/client";
import { getPreferences } from "./lib/preferences";
import type { SearchResult } from "./lib/types";

// ---------------------------------------------------------------------------
// Detail panel — shown in the right pane for the selected result
// ---------------------------------------------------------------------------

function MemoryDetail({
  result,
}: {
  result: SearchResult;
}): React.ReactElement {
  const metaEntries = Object.entries(result.metadata ?? {});

  const markdown = [
    result.content,
    "",
    "---",
    `**Slug:** \`${result.slug}\``,
    `**Audience:** ${result.audience}`,
    `**Score:** ${result.score.toFixed(4)}`,
    `**Workspace:** ${result.workspace}`,
    result.citation_uri ? `**Source:** ${result.citation_uri}` : null,
    metaEntries.length > 0
      ? `**Metadata:**\n\`\`\`json\n${JSON.stringify(result.metadata, null, 2)}\n\`\`\``
      : null,
  ]
    .filter(Boolean)
    .join("\n");

  return <Detail markdown={markdown} />;
}

// ---------------------------------------------------------------------------
// Main command
// ---------------------------------------------------------------------------

export default function SearchMemory(): React.ReactElement {
  const { workspace } = getPreferences();

  const [query, setQuery] = useState<string>("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Debounce: fire search 200 ms after the user stops typing
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (debounceRef.current !== null) {
      clearTimeout(debounceRef.current);
    }

    // Empty query — clear results immediately
    if (!query.trim()) {
      setResults([]);
      setIsLoading(false);
      setErrorMessage(null);
      return;
    }

    setIsLoading(true);

    debounceRef.current = setTimeout(() => {
      void (async () => {
        const result = await searchMemory(query.trim(), workspace);

        if (!result.ok) {
          setErrorMessage(result.error);
          setResults([]);
          await showToast({
            style: Toast.Style.Failure,
            title: "Search failed",
            message: result.error,
          });
        } else {
          setErrorMessage(null);
          setResults(result.data.results);
        }

        setIsLoading(false);
      })();
    }, 200);

    return () => {
      if (debounceRef.current !== null) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [query, workspace]);

  return (
    <List
      isLoading={isLoading}
      searchBarPlaceholder="Search your memory…"
      onSearchTextChange={setQuery}
      throttle={false}
      isShowingDetail={results.length > 0}
    >
      {errorMessage ? (
        <List.EmptyView
          icon={{ source: Icon.ExclamationMark, tintColor: Color.Red }}
          title="Search failed"
          description={errorMessage}
        />
      ) : !query.trim() ? (
        <List.EmptyView
          icon={Icon.MagnifyingGlass}
          title="Search Optimal Engine"
          description="Type to search across your workspace context"
        />
      ) : results.length === 0 && !isLoading ? (
        <List.EmptyView
          icon={Icon.QuestionMark}
          title="No results found"
          description={`No memories match "${query}" in workspace "${workspace}"`}
        />
      ) : (
        results.map((r) => (
          <List.Item
            key={r.id}
            icon={Icon.Dot}
            title={r.slug}
            subtitle={r.audience}
            accessories={[
              {
                tag: {
                  value: r.score.toFixed(2),
                  color:
                    r.score > 0.8
                      ? Color.Green
                      : r.score > 0.5
                        ? Color.Yellow
                        : Color.SecondaryText,
                },
              },
            ]}
            detail={<MemoryDetail result={r} />}
            actions={
              <ActionPanel>
                <ActionPanel.Section title="Memory">
                  <Action.CopyToClipboard
                    title="Copy Content"
                    content={r.content}
                    icon={Icon.Clipboard}
                  />
                  <Action.CopyToClipboard
                    title="Copy Slug"
                    content={r.slug}
                    icon={Icon.Tag}
                    shortcut={{ modifiers: ["cmd"], key: "s" }}
                  />
                  {r.citation_uri && (
                    <Action.OpenInBrowser
                      title="Open Source URL"
                      url={r.citation_uri}
                      icon={Icon.Globe}
                      shortcut={{ modifiers: ["cmd"], key: "o" }}
                    />
                  )}
                </ActionPanel.Section>
              </ActionPanel>
            }
          />
        ))
      )}
    </List>
  );
}
