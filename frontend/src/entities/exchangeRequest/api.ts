import { apiClient } from '@shared/api';

import type { CreateRequestPayload, ExchangeRequest, UpdateRequestPayload } from './model';

// Backend contract: the resource is called /exchange-offers, but inside the frontend the
// entity stays an "exchange request" (ExchangeRequest).
export async function fetchRequests(): Promise<ExchangeRequest[]> {
  const { data } = await apiClient.get<ExchangeRequest[]>('/exchange-offers');
  return data;
}

export async function fetchRequest(requestId: number): Promise<ExchangeRequest> {
  const { data } = await apiClient.get<ExchangeRequest>(`/exchange-offers/${requestId}`);
  return data;
}

export async function createRequest(payload: CreateRequestPayload): Promise<ExchangeRequest> {
  const { data } = await apiClient.post<ExchangeRequest>('/exchange-offers', payload);
  return data;
}

export async function updateRequest(
  requestId: number,
  payload: UpdateRequestPayload,
): Promise<ExchangeRequest> {
  const { data } = await apiClient.put<ExchangeRequest>(`/exchange-offers/${requestId}`, payload);
  return data;
}

export async function removeRequest(requestId: number, version: number): Promise<void> {
  await apiClient.delete(`/exchange-offers/${requestId}`, { params: { version } });
}
