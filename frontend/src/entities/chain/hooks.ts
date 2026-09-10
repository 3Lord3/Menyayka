import { useQueries, useQuery } from '@tanstack/react-query';

import { fetchChain, fetchExchangeOptions, fetchReplacements } from './api';
import type { Chain } from './model';

// live statuses are polled: otherwise the FROZEN transition after another's confirmation and the
// neighbors' shipping on the deal screen show only after a manual refresh
const ACTUALIZATION_MS = 30_000;

function isPollableStatus(status: string | undefined): boolean {
  return status === 'PROPOSED' || status === 'FROZEN' || status === 'IN_PROGRESS';
}

export function chainQueryOptions(chainId?: number) {
  return {
    queryKey: ['chains', chainId],
    queryFn: () => fetchChain(chainId as number),
    enabled: Boolean(chainId),
    refetchInterval: (query: { state: { data?: { status: string } } }) =>
      isPollableStatus(query.state.data?.status) ? ACTUALIZATION_MS : false,
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
  const base = chainQueryOptions(chainId);
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
  return useQueries({ queries: chainIds.map((chainId) => chainQueryOptions(chainId)) });
}

export function useExchangeOptions(offerId?: number) {
  return useQuery({
    queryKey: ['exchange-options', offerId],
    queryFn: () => fetchExchangeOptions(offerId as number),
    enabled: Boolean(offerId),
    refetchInterval: (query) =>
      (query.state.data ?? []).some((entry) => entry.status === 'PROPOSED')
        ? ACTUALIZATION_MS
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
