export * from './types';
export * from './usage';

import * as usageApi from './usage';

export const api = {
  getUsageSummary: usageApi.getUsageSummary,
  getUsageByProvider: usageApi.getUsageByProvider,
  getUsageByModel: usageApi.getUsageByModel,
  getUsageByAgent: usageApi.getUsageByAgent,
  getUsageTrend: usageApi.getUsageTrend,
  getMCPUsage: usageApi.getMCPUsage,
};

export default api;
