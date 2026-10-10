import {
  Action,
  ActionPanel,
  Form,
  Icon,
  Toast,
  popToRoot,
  showToast,
  useNavigation,
} from "@raycast/api";
import React, { useState } from "react";
import { addMemory } from "./lib/client";
import { getPreferences } from "./lib/preferences";

// ---------------------------------------------------------------------------
// Audience options
// ---------------------------------------------------------------------------

const AUDIENCE_OPTIONS = [
  { value: "general", title: "General" },
  { value: "technical", title: "Technical" },
  { value: "executive", title: "Executive" },
  { value: "internal", title: "Internal" },
] as const;

type AudienceValue = (typeof AUDIENCE_OPTIONS)[number]["value"];

// ---------------------------------------------------------------------------
// Form values
// ---------------------------------------------------------------------------

interface FormValues {
  content: string;
  is_static: boolean;
  audience: AudienceValue;
  citation_uri: string;
}

// ---------------------------------------------------------------------------
// Main command
// ---------------------------------------------------------------------------

export default function AddMemory(): React.ReactElement {
  const { workspace } = getPreferences();
  const { pop } = useNavigation();

  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [contentError, setContentError] = useState<string | undefined>();

  async function handleSubmit(values: FormValues): Promise<void> {
    const content = values.content.trim();

    if (!content) {
      setContentError("Content is required");
      return;
    }
    setContentError(undefined);
    setIsSubmitting(true);

    const loadingToast = await showToast({
      style: Toast.Style.Animated,
      title: "Saving memory…",
    });

    const trimmedCitationUri = values.citation_uri.trim();
    const result = await addMemory({
      content,
      workspace,
      is_static: values.is_static,
      audience: values.audience,
      // exactOptionalPropertyTypes: only include the key when it has a value
      ...(trimmedCitationUri ? { citation_uri: trimmedCitationUri } : {}),
    });

    setIsSubmitting(false);

    if (!result.ok) {
      await loadingToast.hide();
      await showToast({
        style: Toast.Style.Failure,
        title: "Failed to save memory",
        message: result.error,
      });
      return;
    }

    await loadingToast.hide();
    await showToast({
      style: Toast.Style.Success,
      title: "Memory saved",
      message: `ID: ${result.data.id}  ·  slug: ${result.data.slug}`,
    });

    // Auto-close — go back to root (or previous view if navigated in)
    try {
      pop();
    } catch {
      await popToRoot();
    }
  }

  return (
    <Form
      isLoading={isSubmitting}
      actions={
        <ActionPanel>
          <Action.SubmitForm
            title="Save Memory"
            icon={Icon.Plus}
            onSubmit={handleSubmit}
          />
        </ActionPanel>
      }
    >
      <Form.TextArea
        id="content"
        title="Content"
        placeholder="Enter a fact, decision, or observation…"
        // Only pass error when it has a value — exactOptionalPropertyTypes requires this
        {...(contentError !== undefined ? { error: contentError } : {})}
        onChange={() => setContentError(undefined)}
        autoFocus
      />

      <Form.Separator />

      <Form.Dropdown id="audience" title="Audience" defaultValue="general">
        {AUDIENCE_OPTIONS.map((opt) => (
          <Form.Dropdown.Item
            key={opt.value}
            value={opt.value}
            title={opt.title}
          />
        ))}
      </Form.Dropdown>

      <Form.Checkbox
        id="is_static"
        title="Static"
        label="Pin this memory (never evicted)"
        defaultValue={false}
      />

      <Form.TextField
        id="citation_uri"
        title="Source URL"
        placeholder="https://… (optional)"
        defaultValue=""
      />

      <Form.Description title="Workspace" text={workspace} />
    </Form>
  );
}
