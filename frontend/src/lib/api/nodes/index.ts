export * from './types';
export * from './nodes';

import * as nodesApi from './nodes';

export const api = {
  getNodes: nodesApi.getNodes,
  getNodeTree: nodesApi.getNodeTree,
  getActiveNode: nodesApi.getActiveNode,
  getNode: nodesApi.getNode,
  createNode: nodesApi.createNode,
  updateNode: nodesApi.updateNode,
  activateNode: nodesApi.activateNode,
  deactivateNode: nodesApi.deactivateNode,
  deleteNode: nodesApi.deleteNode,
  getNodeChildren: nodesApi.getNodeChildren,
  reorderNode: nodesApi.reorderNode,
};

export default api;
