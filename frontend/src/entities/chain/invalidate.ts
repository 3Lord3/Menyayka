import type { QueryClient } from '@tanstack/react-query';

// a request changes status together with its chain, so keys are invalidated in one block —
// otherwise screens drift apart in data freshness
export function invalidateChainQueries(queryClient: QueryClient): void {
  queryClient.invalidateQueries({ queryKey: ['chains'] });
  queryClient.invalidateQueries({ queryKey: ['exchange-options'] });
  queryClient.invalidateQueries({ queryKey: ['exchange-requests'] });
}
