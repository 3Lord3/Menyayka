import { useQueries, useQuery } from '@tanstack/react-query';

import { useRealtimeConnected } from '@shared/realtime';

import { fetchChain, fetchExchangeOptions, fetchReplacements } from './api';
import type { Chain } from './model';

// live statuses are polled: otherwise the FROZEN transition after another's confirmation and the
// neighbors' shipping on the deal screen show only after a manual refresh. With a push channel
// this is only a slow safety net (events do the refreshing).
const ACTUALIZATION_MS = 30_000;
const REALTIME_FALLBACK_MS = 5 * 60_000;

// The slow interval is safe only while the socket is actually up.
function actualizationInterval(realtimeConnected: boolean): number {
  return realtimeConnected ? REALTIME_FALLBACK_MS : ACTUALIZATION_MS;
}

function isPollableStatus(status: string | undefined): boolean {
  return status === 'PROPOSED' || status === 'FROZEN' || status === 'IN_PROGRESS';
}

export function chainQueryOptions(chainId?: number, realtimeConnected = false) {
  return {
    queryKey: ['chains', chainId],
    queryFn: () => fetchChain(chainId as number),
    enabled: Boolean(chainId),
    refetchInterval: (query: { state: { data?: { status: string } } }) =>
      isPollableStatus(query.state.data?.status) ? actualizationInterval(realtimeConnected) : false,
    refetchOnWindowFocus: (query: { state: { data?: { status: string } } }) =>
      isPollableStatus(query.state.data?.status),
  };
}

export function useChain(
  chainId?: number,
  options: {
    refetchInterval?: number | false | ((chain: Chain | undefined) => number | false);
  } = {},
) {
  const { refetchInterval } = options;
  const realtimeConnected = useRealtimeConnected();
  const base = chainQueryOptions(chainId, realtimeConnected);
  return useQuery({
    ...base,
    refetchInterval:
      refetchInterval === undefined
        ? base.refetchInterval
        : typeof refetchInterval === 'function'
          ? (query) => refetchInterval(query.state.data)
          : refetchInterval,
  });
}

export function useChains(chainIds: number[]) {
  const realtimeConnected = useRealtimeConnected();
  return useQueries({
    queries: chainIds.map((chainId) => chainQueryOptions(chainId, realtimeConnected)),
  });
}

export function useExchangeOptions(offerId?: number) {
  const realtimeConnected = useRealtimeConnected();
  return useQuery({
    queryKey: ['exchange-options', offerId],
    queryFn: () => fetchExchangeOptions(offerId as number),
    enabled: Boolean(offerId),
    refetchInterval: (query) =>
      (query.state.data ?? []).some((entry) => entry.status === 'PROPOSED')
        ? actualizationInterval(realtimeConnected)
        : false,
    refetchOnWindowFocus: (query) =>
      (query.state.data ?? []).some((entry) => entry.status === 'PROPOSED'),
  });
}

// key deliberately nested under ['chains']: the replacement pool is volatile and must be re-read
// on any chain mutation, and invalidation goes by prefix
export function replacementQueryOptions(
  chainId?: number,
  options: { enabled?: boolean; refetchInterval?: number | false } = {},
) {
  return {
    queryKey: ['chains', chainId, 'replacements'],
    queryFn: () => fetchReplacements(chainId as number),
    enabled: Boolean(chainId) && options.enabled !== false,
    refetchInterval: options.refetchInterval ?? false,
  };
}

export function useReplacements(
  chainId?: number,
  options: { enabled?: boolean; refetchInterval?: number | false } = {},
) {
  return useQuery(replacementQueryOptions(chainId, options));
}

// the "Exchange options" list needs the replacement pool for all PROPOSED chains at once
export function useReplacementsForChains(chainIds: number[]) {
  return useQueries({ queries: chainIds.map((chainId) => replacementQueryOptions(chainId)) });
}
