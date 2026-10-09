import type { QueryClient } from '@tanstack/react-query';

import type { RealtimeEvent } from './events';

// Event -> query invalidation. The frontend never stores server state from events: it
// only invalidates existing query keys so the body is refetched over REST. TanStack
// matches by key prefix, so invalidating ['chains'] also refetches ['chains', id] and
// ['chains', id, 'replacements']: no scoped duplicate calls are needed.

const CHAINS_KEY = ['chains'];
const OPTIONS_KEY = ['exchange-options'];
const REQUESTS_KEY = ['exchange-requests'];
const ITEMS_KEY = ['items'];

function invalidateKeys(queryClient: QueryClient, keys: readonly unknown[][]): void {
  keys.forEach((queryKey) => queryClient.invalidateQueries({ queryKey: [...queryKey] }));
}

export function invalidateForEvent(queryClient: QueryClient, event: RealtimeEvent): void {
  switch (event.type) {
    case 'chain.created':
      // a new chain has no subscribers yet; refreshing the lists is the whole effect
      invalidateKeys(queryClient, [CHAINS_KEY, OPTIONS_KEY]);
      return;
    case 'request.status_changed':
    case 'request.updated':
      invalidateKeys(queryClient, [REQUESTS_KEY]);
      return;
    case 'item.status_changed':
      invalidateKeys(queryClient, [ITEMS_KEY]);
      return;
    default:
      // any chain.* event may indirectly change items too (freeze/expiry reserve or
      // release them)
      invalidateKeys(queryClient, [CHAINS_KEY, OPTIONS_KEY, REQUESTS_KEY, ITEMS_KEY]);
  }
}

// After a reconnect we may have missed events: refresh every realtime-affected key.
export function invalidateAllRealtime(queryClient: QueryClient): void {
  invalidateKeys(queryClient, [CHAINS_KEY, OPTIONS_KEY, REQUESTS_KEY, ITEMS_KEY]);
}
