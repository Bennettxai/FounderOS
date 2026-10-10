export * from './types';
export * from './calendar';

import * as calendarApi from './calendar';

export const api = {
  getCalendarEvents: calendarApi.getCalendarEvents,
  getCalendarEvent: calendarApi.getCalendarEvent,
  createCalendarEvent: calendarApi.createCalendarEvent,
  updateCalendarEvent: calendarApi.updateCalendarEvent,
  deleteCalendarEvent: calendarApi.deleteCalendarEvent,
  syncCalendar: calendarApi.syncCalendar,
  getTodayEvents: calendarApi.getTodayEvents,
  getUpcomingEvents: calendarApi.getUpcomingEvents,
  // Calendar-specific OAuth (isolated scopes - calendar only)
  getCalendarConnectionStatus: calendarApi.getCalendarConnectionStatus,
  getCalendarAuthUrl: calendarApi.getCalendarAuthUrl,
  disconnectCalendar: calendarApi.disconnectCalendar,
  getTaskSuggestions: calendarApi.getTaskSuggestions,
};

export default api;
