import type { EngineContext } from "../engine.js";
import type { WikiArticle } from "@optimal-engine/client";

export function useWiki(engine: EngineContext) {
  let pages = $state<WikiArticle[]>([]);
  let selected = $state<WikiArticle | null>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function loadPages() {
    loading = true;
    error = null;
    try {
      const ws = engine.getWorkspace();
      pages = await engine.client.wiki.list({ workspace: ws });
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function openPage(slug: string, audience?: string) {
    loading = true;
    error = null;
    try {
      const ws = engine.getWorkspace();
      const { asWikiSlug } = await import("@optimal-engine/client");
      selected = await engine.client.wiki.get(asWikiSlug(slug), {
        workspace: ws,
        audience,
      });
    } catch (e) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  // Reload on workspace change
  $effect(() => {
    engine.workspace.subscribe(() => void loadPages());
  });

  return {
    get pages() {
      return pages;
    },
    get selected() {
      return selected;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    openPage,
    refresh: loadPages,
  };
}
