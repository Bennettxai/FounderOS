export * from './types';
export * from './dashboard';

import * as dashboardApi from './dashboard';

export const api = {
  getDashboardSummary: dashboardApi.getDashboardSummary,
  getFocusItems: dashboardApi.getFocusItems,
  createFocusItem: dashboardApi.createFocusItem,
  updateFocusItem: dashboardApi.updateFocusItem,
  deleteFocusItem: dashboardApi.deleteFocusItem,
  getTasks: dashboardApi.getTasks,
  createTask: dashboardApi.createTask,
  updateTask: dashboardApi.updateTask,
  toggleTask: dashboardApi.toggleTask,
  deleteTask: dashboardApi.deleteTask,
};

export default api;
