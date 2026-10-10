import { request } from '../base';
import type { DealResponse, DealStage } from './types';

// Standalone deals endpoints for pipeline view
// Note: Client-specific deal operations are in the clients module

export async function getAllDeals(stage?: DealStage) {
  const params = stage ? `?stage_filter=${stage}` : '';
  return request<DealResponse[]>(`/deals${params}`);
}

export async function updateDealStage(dealId: string, stage: DealStage) {
  return request<DealResponse>(`/deals/${dealId}/stage`, {
    method: 'PATCH',
    body: { stage }
  });
}
