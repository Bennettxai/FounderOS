// CRM API Module
// Re-exports all types and functions

export * from './types';
export * from './crm';

// Local API object for direct imports
import * as crmApi from './crm';
export const api = crmApi;
