export * from './types';
export * from './deals';

import * as dealsApi from './deals';

export const api = {
  getAllDeals: dealsApi.getAllDeals,
  updateDealStage: dealsApi.updateDealStage,
};

export default api;
