import { tick } from "svelte";
import { editor, type BlockType } from "$lib/stores/editor";
import { contexts } from "$lib/stores/contexts";

/**
 * Handles block type selection from the slash menu, including sub-page creation.
 */
export async function selectBlockType(
  blockId: string,
  type: BlockType,
  blockElement: HTMLElement | null,
  parentContextId: string | undefined,
  properties?: Record<string, unknown>,
): Promise<void> {
  try {
    if (type === "page") {
      await createSubPage(
        blockId,
        blockElement,
        parentContextId,
        properties?.icon as string | undefined,
      );
      return;
    }

    editor.changeBlockType(blockId, type);
    editor.updateBlock(blockId, "");

    await tick();
    if (blockElement) {
      blockElement.innerText = "";
      blockElement.focus();
    }
  } catch (e) {
    console.error("Error in selectBlockType:", e);
  }
}

/**
 * Creates a sub-page context under the given parent, updating the block to a page reference.
 */
export async function createSubPage(
  blockId: string,
  blockElement: HTMLElement | null,
  parentContextId: string | undefined,
  icon?: string,
): Promise<void> {
  editor.updateBlock(blockId, "");
  editor.hideSlashMenu();
  await tick();
  if (blockElement) {
    blockElement.innerText = "";
  }

  if (!parentContextId) {
    editor.changeBlockType(blockId, "page");
    editor.updateBlock(blockId, "New page", { icon: icon || "document" });
    await tick();
    if (blockElement) {
      blockElement.innerText = "New page";
    }
    return;
  }

  try {
    const newContext = await contexts.createContext({
      name: "New page",
      type: "document",
      parent_id: parentContextId,
      blocks: [],
      icon: icon || "document",
    });

    editor.updateBlock(blockId, "New page", {
      pageId: newContext.id,
      icon: icon || "document",
    });
    editor.changeBlockType(blockId, "page");

    await contexts.loadContexts();

    const { kbPreferences } = await import("$lib/stores/kb-preferences");
    kbPreferences.expandPage(parentContextId);

    await tick();
    if (blockElement) {
      blockElement.innerText = "New page";
    }
  } catch (e) {
    console.error("Failed to create sub-page:", e);
  }
}
