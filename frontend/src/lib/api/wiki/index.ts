export * from "./types";
export * from "./wiki";

import * as wikiApi from "./wiki";

export const api = {
  listWikiPages: wikiApi.listWikiPages,
  getWikiPage: wikiApi.getWikiPage,
  curateWikiPage: wikiApi.curateWikiPage,
  renderWikiPage: wikiApi.renderWikiPage,
  verifyWikiPage: wikiApi.verifyWikiPage,
};

export default api;
