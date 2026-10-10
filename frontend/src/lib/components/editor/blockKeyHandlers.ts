import { tick } from "svelte";
import { editor, type EditorBlock, type BlockType } from "$lib/stores/editor";

/**
 * Handles keydown events for contenteditable blocks.
 * Manages Enter (new block), Backspace (delete empty block), and arrow navigation.
 */
export function handleBlockKeydown(
  e: KeyboardEvent,
  block: EditorBlock,
  index: number,
  blockElement: HTMLElement,
  getBlocks: () => EditorBlock[],
  showSlashMenu: boolean,
): void {
  if (showSlashMenu && e.key === "Escape") {
    e.preventDefault();
    editor.hideSlashMenu();
    return;
  }

  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault();

    const currentContent = blockElement.innerText || "";
    editor.updateBlock(block.id, currentContent);

    if (
      currentContent === "" &&
      ["bulletList", "numberedList", "todo"].includes(block.type)
    ) {
      editor.changeBlockType(block.id, "paragraph");
      return;
    }

    const newType: BlockType = ["bulletList", "numberedList", "todo"].includes(
      block.type,
    )
      ? block.type
      : "paragraph";
    const newBlockId = editor.addBlockAfter(block.id, newType);

    tick().then(() => {
      const newBlockEl = document.querySelector(
        `[data-block-id="${newBlockId}"]`,
      ) as HTMLElement;
      newBlockEl?.focus();
    });
    return;
  }

  if (e.key === "Backspace") {
    const currentContent = blockElement.innerText || "";

    if (currentContent === "" && index > 0) {
      e.preventDefault();
      editor.deleteBlock(block.id);
      tick().then(() => {
        const blocks = getBlocks();
        const prevBlock = blocks[Math.max(0, index - 1)];
        if (prevBlock) {
          const prevEl = document.querySelector(
            `[data-block-id="${prevBlock.id}"]`,
          ) as HTMLElement;
          prevEl?.focus();
          const range = document.createRange();
          const sel = window.getSelection();
          if (prevEl.childNodes.length > 0) {
            range.setStartAfter(prevEl.lastChild!);
          } else {
            range.setStart(prevEl, 0);
          }
          range.collapse(true);
          sel?.removeAllRanges();
          sel?.addRange(range);
        }
      });
    }
    return;
  }

  if (e.key === "ArrowUp" && index > 0) {
    const selection = window.getSelection();
    if (selection && selection.anchorOffset === 0) {
      e.preventDefault();
      editor.updateBlock(block.id, blockElement.innerText || "");
      editor.focusPreviousBlock();
      tick().then(() => {
        const blocks = getBlocks();
        const prevBlock = blocks[index - 1];
        if (prevBlock) {
          const prevEl = document.querySelector(
            `[data-block-id="${prevBlock.id}"]`,
          ) as HTMLElement;
          prevEl?.focus();
        }
      });
    }
    return;
  }

  if (e.key === "ArrowDown" && index < getBlocks().length - 1) {
    const selection = window.getSelection();
    const contentLength = (blockElement.innerText || "").length;
    if (selection && selection.anchorOffset === contentLength) {
      e.preventDefault();
      editor.updateBlock(block.id, blockElement.innerText || "");
      editor.focusNextBlock();
      tick().then(() => {
        const blocks = getBlocks();
        const nextBlock = blocks[index + 1];
        if (nextBlock) {
          const nextEl = document.querySelector(
            `[data-block-id="${nextBlock.id}"]`,
          ) as HTMLElement;
          nextEl?.focus();
        }
      });
    }
  }
}

/**
 * Handles keydown events for divider blocks (non-contenteditable).
 */
export function handleDividerKeydown(
  e: KeyboardEvent,
  block: EditorBlock,
  index: number,
  getBlocks: () => EditorBlock[],
): void {
  if (e.key === "Enter") {
    e.preventDefault();
    const newBlockId = editor.addBlockAfter(block.id, "paragraph");
    tick().then(() => {
      const newBlockEl = document.querySelector(
        `[data-block-id="${newBlockId}"]`,
      ) as HTMLElement;
      newBlockEl?.focus();
    });
    return;
  }

  if (e.key === "Backspace") {
    e.preventDefault();
    if (index > 0) {
      editor.deleteBlock(block.id);
      tick().then(() => {
        const blocks = getBlocks();
        const prevBlock = blocks[Math.max(0, index - 1)];
        if (prevBlock) {
          const prevEl = document.querySelector(
            `[data-block-id="${prevBlock.id}"]`,
          ) as HTMLElement;
          prevEl?.focus();
        }
      });
    }
    return;
  }

  if (e.key === "ArrowUp" && index > 0) {
    e.preventDefault();
    editor.focusPreviousBlock();
    tick().then(() => {
      const blocks = getBlocks();
      const prevBlock = blocks[index - 1];
      if (prevBlock) {
        const prevEl = document.querySelector(
          `[data-block-id="${prevBlock.id}"]`,
        ) as HTMLElement;
        prevEl?.focus();
      }
    });
    return;
  }

  if (e.key === "ArrowDown" && index < getBlocks().length - 1) {
    e.preventDefault();
    editor.focusNextBlock();
    tick().then(() => {
      const blocks = getBlocks();
      const nextBlock = blocks[index + 1];
      if (nextBlock) {
        const nextEl = document.querySelector(
          `[data-block-id="${nextBlock.id}"]`,
        ) as HTMLElement;
        nextEl?.focus();
      }
    });
  }
}
