import { useEffect, useRef } from 'react';

import { useQueryClient } from '@tanstack/react-query';

import type { ChainStatus } from '@entities/chain';

import { useRealtimeConnected } from '@shared/realtime';

// margin for client/server clock skew: without it freeze_deadline_at <= NOW()
// on the backend won't hold yet and the rollback waits until the next poll
const EXPIRY_SLACK_MS = 2_000;

export interface ProposalExpiryState {
  chainId: number;
  listStatus?: ChainStatus;
  detailStatus?: ChainStatus;
  deadlineAt?: string | null;
}

// the backend rolls back an overdue PROPOSED lazily and only in GET /chains/{id}, so the options
// list keeps returning PROPOSED with live second-round buttons, and confirming them hits a 410.
// The hook hits the detail at the deadline and invalidates the list after it moves on.
export function useProposalExpiry(states: ProposalExpiryState[]): void {
  const queryClient = useQueryClient();
  // with a push channel the worker emits chain.expired; the timer is the fallback while down
  const realtimeConnected = useRealtimeConnected();
  // the array is rebuilt every render: effects hook onto its value, not its reference
  const signature = states
    .map((state) =>
      [state.chainId, state.listStatus, state.detailStatus, state.deadlineAt].join(':'),
    )
    .join('|');
  // declared first: effects run in declaration order, so below the ref is already current
  const latest = useRef(states);
  useEffect(() => {
    latest.current = states;
  });

  // on the detail screen there's nothing to compare to: the only rollback sign is that the
  // detail just left PROPOSED
  const seenDetail = useRef(new Map<number, ChainStatus>());

  useEffect(() => {
    if (realtimeConnected) return;
    let stale = false;
    // not some(): it would stop the scan on the first stale chain and forget the other statuses
    latest.current.forEach((state) => {
      const previous = seenDetail.current.get(state.chainId);
      if (state.detailStatus === undefined) return;
      seenDetail.current.set(state.chainId, state.detailStatus);
      if (state.detailStatus === 'PROPOSED') return;
      if (state.listStatus === 'PROPOSED' || previous === 'PROPOSED') stale = true;
    });
    if (stale) void queryClient.invalidateQueries({ queryKey: ['exchange-options'] });
  }, [signature, queryClient, realtimeConnected]);

  useEffect(() => {
    if (realtimeConnected) return;
    const timers = latest.current
      .filter((state) => state.detailStatus === 'PROPOSED')
      .map((state) => {
        const deadline = Date.parse(state.deadlineAt ?? '');
        if (Number.isNaN(deadline)) return undefined;
        // the deadline may have passed before mounting (detail from cache) — Math.max avoids
        // waiting into negative time
        const delay = Math.max(0, deadline - Date.now()) + EXPIRY_SLACK_MS;
        return setTimeout(() => {
          void queryClient.invalidateQueries({ queryKey: ['chains', state.chainId] });
        }, delay);
      });
    return () => {
      timers.forEach((timer) => {
        if (timer !== undefined) clearTimeout(timer);
      });
    };
  }, [signature, queryClient, realtimeConnected]);
}
