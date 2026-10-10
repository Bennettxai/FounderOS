export type ContentThemeDefinition = {
  purpose: string;
  sublanes: string[];
};

export const OWNED_CONTENT_PROFILES: string[] = [];
export const CLIENT_CONTENT_PROFILES: string[] = [];
export const OWNED_CONTENT_WORKSTREAMS = ["Organic Content"];
export const CONTENT_THEMES: string[] = [];
export const CONTENT_THEME_COLORS: Record<string, string> = {
  Uncategorized: "#737373",
};
export const CONTENT_THEME_DEFINITIONS: Record<string, ContentThemeDefinition> =
  {};
