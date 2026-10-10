export * from './types';
export * from './settings';

import * as settingsApi from './settings';

export const api = {
  getSettings: settingsApi.getSettings,
  updateSettings: settingsApi.updateSettings,
  getSystemInfo: settingsApi.getSystemInfo,
};

export default api;
