export * from "./types";
export * from "./compliance";

import * as complianceApi from "./compliance";

export const api = {
  // DSAR
  exportDSAR: complianceApi.exportDSAR,
  // Erasure
  previewErasure: complianceApi.previewErasure,
  eraseData: complianceApi.eraseData,
  // Legal Holds
  listLegalHolds: complianceApi.listLegalHolds,
  placeLegalHold: complianceApi.placeLegalHold,
  releaseLegalHold: complianceApi.releaseLegalHold,
  // Retention Policies
  listRetentionPolicies: complianceApi.listRetentionPolicies,
  createRetentionPolicy: complianceApi.createRetentionPolicy,
  deleteRetentionPolicy: complianceApi.deleteRetentionPolicy,
  runRetentionSweep: complianceApi.runRetentionSweep,
  // Audit Log
  queryAuditEvents: complianceApi.queryAuditEvents,
};

export default api;
