import type { ChainStatus } from '@entities/chain';

// a decline doesn't always break the chain: the server rolls it back to CANDIDATE, keeps
// PROPOSED with a vacancy for replacement, or dissolves it entirely — the outcome is visible
// only from the response
const DECLINE_MESSAGE: Partial<Record<ChainStatus, string>> = {
  BROKEN: 'Вы вышли из сделки. Цепочка распалась',
  CANDIDATE: 'Вы вышли из сделки. Цепочка вернулась к сбору откликов',
  PROPOSED: 'Вы вышли из сделки. Участники подбирают замену',
};

export function declineMessage(status: ChainStatus): string {
  return DECLINE_MESSAGE[status] ?? 'Вы вышли из сделки';
}
