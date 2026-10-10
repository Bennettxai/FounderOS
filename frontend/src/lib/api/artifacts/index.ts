export * from './types';
export * from './artifacts';

import * as artifactsApi from './artifacts';

export const api = {
  getArtifacts: artifactsApi.getArtifacts,
  getArtifact: artifactsApi.getArtifact,
  createArtifact: artifactsApi.createArtifact,
  updateArtifact: artifactsApi.updateArtifact,
  deleteArtifact: artifactsApi.deleteArtifact,
  linkArtifact: artifactsApi.linkArtifact,
  getArtifactVersions: artifactsApi.getArtifactVersions,
  restoreArtifactVersion: artifactsApi.restoreArtifactVersion,
};

export default api;
