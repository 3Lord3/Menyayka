import type { RequestStatus } from '@entities/exchangeRequest';

import type { StatusTone } from '@shared/ui';

export type ChainStatus =
  'CANDIDATE' | 'PROPOSED' | 'FROZEN' | 'IN_PROGRESS' | 'COMPLETED' | 'BROKEN';

// the client never sets 'thinking' — the server returns such a vote from old data
export type VoteValue = 'pending' | 'approved' | 'rejected' | 'thinking';

// participants is a candidate pool, not participants: at CANDIDATE each ring position is
// expanded into all its cluster's requests. vote is tied to the target: position p's decision
// lives in the next position's vote.
// freezeDeadlineAt on PROPOSED — the answer deadline (or quick-replacement pickup when
// invalidReason = 'frozen_replacement'); on FROZEN — the shipping deadline
export interface ChainParticipant {
  clusterId: number;
  requestId: number;
  position: number;
  isCurrentUser: boolean;
  offeredItemId: number;
  offeredItemTitle: string;
  offeredItemDescription: string;
  wantedDescription: string;
  // the photo may be absent entirely (omitempty), not only null
  imageUrl?: string | null;
  // LOCKED — not shipped, IN_PROGRESS — shipped, DONE — receiver confirmed
  requestStatus: RequestStatus;
  vote?: VoteValue;
}

export interface Chain {
  id: number;
  status: ChainStatus;
  score: number;
  length: number;
  version: number;
  currentRequestId: number;
  currentPosition: number;
  givesToPosition: number;
  receivesFromPosition: number;
  freezeDeadlineAt?: string | null;
  invalidReason?: string | null;
  createdAt: string;
  updatedAt: string;
  participants: ChainParticipant[];
}

export interface ExchangeOption {
  clusterId: number;
  requestId: number;
  itemId: number;
  title: string;
  description: string;
  wantedDescription: string;
  imageUrl?: string | null;
  vote?: VoteValue;
}

export interface ExchangeOptions {
  chainId: number;
  status: ChainStatus;
  score: number;
  length: number;
  currentRequestId: number;
  currentPosition: number;
  givesToPosition: number;
  receivesFromPosition: number;
  currentOffer: ExchangeOption;
  receiveOptions: ExchangeOption[];
}

export interface VotePayload {
  requestId: number;
  targetRequestId: number;
}

export interface ChainVoteResult {
  chainId: number;
  requestId: number;
  targetRequestId: number;
  vote: VoteValue;
  votedAt: string;
  chainStatus: ChainStatus;
}

// the pool is already filtered and sorted by the server — the client never assembles the list
export interface ReplacementOption {
  requestId: number;
  offeredItemId: number;
  title: string;
  description: string;
  wantedDescription: string;
  // the photo may be absent entirely (omitempty), not only null
  imageUrl?: string | null;
  reliability: number;
  respondedAt: string;
}

export interface SelectReplacementResult {
  chainId: number;
  requestId: number;
  status: ChainStatus;
}

export interface ConfirmResult {
  chainId: number;
  status: ChainStatus;
}

// replacementAvailable refers to the previous participant around the ring, not the decliner
export interface DeclineResult {
  chainId: number;
  status: ChainStatus;
  replacementAvailable: boolean;
}

export interface ChainLink {
  position: number;
  candidates: ChainParticipant[];
}

export interface FulfillmentResult {
  chainId: number;
  requestId: number;
  status: ChainStatus;
}

// thinking is a second-round value; it has no place in the first-round dictionary
export const VOTE_META: Partial<Record<VoteValue, ConfirmVoteMeta>> = {
  pending: { label: 'Ожидаем', tone: 'warning' },
  approved: { label: 'Отклик принят', tone: 'success' },
  rejected: { label: 'Отклик отклонён', tone: 'error' },
};

export interface ConfirmVoteMeta {
  label: string;
  tone: StatusTone;
}

export const CONFIRM_VOTE_META: Record<VoteValue, ConfirmVoteMeta> = {
  approved: { label: 'Согласился', tone: 'success' },
  thinking: { label: 'Думает', tone: 'warning' },
  pending: { label: 'Ожидает ответа', tone: 'warning' },
  rejected: { label: 'Отказался', tone: 'error' },
};

// an empty vote at position (p + 1) % length means a vacancy at position p
export const VACANCY_META: ConfirmVoteMeta = {
  label: 'Место освободилось',
  tone: 'neutral',
};

export const HARD_LOCK_MESSAGE = 'Товар жёстко заблокирован: изменить или удалить заявку нельзя';

export function isHardLocked(status: ChainStatus): boolean {
  return status === 'FROZEN' || status === 'IN_PROGRESS';
}

// at COMPLETED there's no hard lock anymore, but the deal still needs to open
export function hasDeal(status: ChainStatus): boolean {
  return isHardLocked(status) || status === 'COMPLETED';
}

// the chain stays FROZEN until the first handoff
export function needsShipment(status: ChainStatus): boolean {
  return status === 'FROZEN';
}

// as early as PROPOSED the chain occupies the request — other options of it are dimmed
export function isAssembled(status: ChainStatus): boolean {
  return status === 'PROPOSED' || isHardLocked(status);
}

// we show the probability while it still affects the decision: on CANDIDATE (choosing a
// variant), PROPOSED (confirmation), and FROZEN (item not shipped); after that the deal is
// done or fell apart — badge is empty
export function showsScore(status: ChainStatus): boolean {
  return status === 'CANDIDATE' || status === 'PROPOSED' || status === 'FROZEN';
}

// a quick replacement after a decline on FROZEN: the chain is PROPOSED again, but its deadline
// is finding a replacement, not responding (see PROJECT.md §4.8)
export function isFrozenReplacement(chain: Chain): boolean {
  return chain.status === 'PROPOSED' && chain.invalidReason === 'frozen_replacement';
}

export function myParticipant(chain: Chain): ChainParticipant | null {
  return chain.participants.find((p) => p.isCurrentUser) ?? null;
}

export function chainLinks(chain: Chain): ChainLink[] {
  const byPosition = new Map<number, ChainParticipant[]>();
  for (const participant of chain.participants) {
    const candidates = byPosition.get(participant.position);
    if (candidates) candidates.push(participant);
    else byPosition.set(participant.position, [participant]);
  }
  return [...byPosition.entries()]
    .sort(([a], [b]) => a - b)
    .map(([position, candidates]) => ({ position, candidates }));
}

// a cluster's requests belong to different users: without scoping by requestId the screen
// would show their foreign items under one participant, and an unknown requestId returns the
// whole pool, not an empty one
export function receivesItem(chain: Chain, requestId?: number): ChainParticipant[] {
  const sources = chain.participants.filter((p) => p.position === chain.receivesFromPosition);
  if (requestId === undefined) return sources;
  const selected = sources.find((p) => p.requestId === requestId);
  return selected ? [selected] : sources;
}

// on an assembled chain exactly one participant maps to a position
export function sourceParticipant(chain: Chain): ChainParticipant | null {
  const sources = receivesItem(chain);
  return sources.length === 1 ? sources[0] : null;
}

// on CANDIDATE there may be fewer positions than length — can't use (p + 1) % length
export function nextInRing(chain: Chain, position: number): number | null {
  const positions = [...new Set(chain.participants.map((p) => p.position))].sort((a, b) => a - b);
  const index = positions.indexOf(position);
  if (positions.length === 0 || index === -1) return null;
  return positions[(index + 1) % positions.length];
}

// an empty vote at the next position is a vacancy, not pending
export function confirmVoteAt(chain: Chain, position: number): VoteValue | null {
  const next = nextInRing(chain, position);
  if (next === null) return null;
  return chain.participants.find((p) => p.position === next)?.vote ?? null;
}

// on the options card the same value arrives as receiveOptions[0].vote
export function myConfirmVote(chain: Chain): VoteValue | null {
  return confirmVoteAt(chain, chain.currentPosition);
}

// the "voter ↔ target" relation is bijective, so shifting votes doesn't affect the count
export function approvedVotes(chain: Chain): number {
  if (!isAssembled(chain.status)) return 0;
  return chain.participants.reduce((count, p) => count + (p.vote === 'approved' ? 1 : 0), 0);
}

export function needsMyAction(chain: Chain): boolean {
  return chain.status === 'PROPOSED' && myConfirmVote(chain) !== 'approved';
}

export type DealState =
  | { status: 'ship'; deadlineAt: string | null }
  | { status: 'shipped-waiting'; shipped: number; total: number }
  | { status: 'in-transit'; shipped: number; total: number }
  | { status: 'received-waiting' }
  | { status: 'completed' }
  | { status: 'unavailable' };

export function dealState(chain: Chain): DealState {
  if (chain.status === 'COMPLETED') return { status: 'completed' };
  if (chain.status !== 'FROZEN' && chain.status !== 'IN_PROGRESS') {
    return { status: 'unavailable' };
  }
  const me = myParticipant(chain);
  if (!me) return { status: 'unavailable' };
  const total = chain.length;
  const shipped = chain.participants.filter((p) => p.requestStatus !== 'LOCKED').length;
  // the shipping screen is visible even on IN_PROGRESS: a neighbor may have shipped first
  if (me.requestStatus === 'LOCKED') {
    return { status: 'ship', deadlineAt: chain.freezeDeadlineAt ?? null };
  }
  if (sourceParticipant(chain)?.requestStatus === 'DONE') return { status: 'received-waiting' };
  if (shipped < total) return { status: 'shipped-waiting', shipped, total };
  return { status: 'in-transit', shipped, total };
}
