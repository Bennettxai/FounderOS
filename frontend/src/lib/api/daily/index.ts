export * from './types';
export * from './daily';

import * as dailyApi from './daily';

export const api = {
  getDailyLogs: dailyApi.getDailyLogs,
  getTodayLog: dailyApi.getTodayLog,
  getDailyLogByDate: dailyApi.getDailyLogByDate,
  saveDailyLog: dailyApi.saveDailyLog,
  updateDailyLog: dailyApi.updateDailyLog,
  deleteDailyLog: dailyApi.deleteDailyLog,
};

export default api;
