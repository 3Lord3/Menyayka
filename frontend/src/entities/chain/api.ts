import { apiClient } from '@shared/api';

import type {
  Chain,
  ChainVoteResult,
  ConfirmResult,
  DeclineResult,
  ExchangeOptions,
  FulfillmentResult,
  ReplacementOption,
  SelectReplacementResult,
  VotePayload,
} from './model';

export async function fetchChain(chainId: number): Promise<Chain> {
  const { data } = await apiClient.get<Chain>(`/chains/${chainId}`);
  return data;
}

export async function fetchExchangeOptions(offerId: number): Promise<ExchangeOptions[]> {
  const { data } = await apiClient.get<ExchangeOptions[]>(
    `/exchange-offers/${offerId}/exchange-options`,
  );
  return data;
}

// a directed response to a specific next-link request; the backend closes the response ring
export async function voteForRequest(
  chainId: number,
  payload: VotePayload,
): Promise<ChainVoteResult> {
  const { data } = await apiClient.put<ChainVoteResult>(`/chains/${chainId}/votes`, payload);
  return data;
}

export async function withdrawVote(chainId: number, payload: VotePayload): Promise<void> {
  await apiClient.delete(`/chains/${chainId}/votes`, { params: payload });
}

// the candidate pool for replacing a departed participant; the reply is a flat array, not {data: [...]}
export async function fetchReplacements(chainId: number): Promise<ReplacementOption[]> {
  const { data } = await apiClient.get<ReplacementOption[]>(`/chains/${chainId}/replacements`);
  return data;
}

// invites a candidate to a freed position; not idempotent — a repeated PUT with the same
// requestId the server rejects (422)
export async function selectReplacement(
  chainId: number,
  requestId: number,
): Promise<SelectReplacementResult> {
  const { data } = await apiClient.put<SelectReplacementResult>(`/chains/${chainId}/replacement`, {
    requestId,
  });
  return data;
}

// re-confirms participation in an assembled chain (second round)
export async function confirmChain(chainId: number): Promise<ConfirmResult> {
  const { data } = await apiClient.post<ConfirmResult>(`/chains/${chainId}/confirm`);
  return data;
}

// declining participation in an assembled chain (second round)
export async function declineChain(chainId: number): Promise<DeclineResult> {
  const { data } = await apiClient.post<DeclineResult>(`/chains/${chainId}/decline`);
  return data;
}

// "I shipped the item" — simulates pickup-point confirmation: requestId is my
// request (chain.currentRequestId), it goes LOCKED → IN_PROGRESS, the chain FROZEN → IN_PROGRESS
export async function confirmHandoff(
  chainId: number,
  requestId: number,
): Promise<FulfillmentResult> {
  const { data } = await apiClient.post<FulfillmentResult>('/integrations/avito/handoffs', {
    chainId,
    requestId,
  });
  return data;
}

// "I picked it up" — delivery confirmation: requestId is the source link's request
// (receivesFromPosition), it goes IN_PROGRESS → DONE; once all requests are DONE — chain COMPLETED
export async function confirmReceipt(
  chainId: number,
  requestId: number,
): Promise<FulfillmentResult> {
  const { data } = await apiClient.post<FulfillmentResult>(`/chains/${chainId}/receipt`, {
    requestId,
  });
  return data;
}
