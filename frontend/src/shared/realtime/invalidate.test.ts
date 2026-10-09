import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';

import { invalidateAllRealtime, invalidateForEvent } from './invalidate';

function spyClient() {
  const queryClient = new QueryClient();
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
  return { queryClient, invalidate };
}

describe('invalidateForEvent', () => {
  it('invalidates chains, options and items for a chain event', () => {
    const { queryClient, invalidate } = spyClient();

    invalidateForEvent(queryClient, { type: 'chain.status_changed', chainId: 3 });

    const keys = invalidate.mock.calls.map((call) => call[0]?.queryKey);
    expect(keys).toContainEqual(['chains']);
    expect(keys).toContainEqual(['exchange-options']);
    expect(keys).toContainEqual(['exchange-requests']);
    expect(keys).toContainEqual(['items']);
  });

  it('does not touch items for chain.created but refreshes lists', () => {
    const { queryClient, invalidate } = spyClient();

    invalidateForEvent(queryClient, { type: 'chain.created' });

    const keys = invalidate.mock.calls.map((call) => call[0]?.queryKey);
    expect(keys).toContainEqual(['chains']);
    expect(keys).toContainEqual(['exchange-options']);
    expect(keys).not.toContainEqual(['items']);
  });

  it('invalidates the request list once — the key matches by prefix', () => {
    const { queryClient, invalidate } = spyClient();

    invalidateForEvent(queryClient, { type: 'request.status_changed', payload: { requestId: 11 } });

    expect(invalidate.mock.calls.map((call) => call[0]?.queryKey)).toEqual([['exchange-requests']]);
  });

  it('invalidates the item list once', () => {
    const { queryClient, invalidate } = spyClient();

    invalidateForEvent(queryClient, { type: 'item.status_changed', payload: { itemId: 5 } });

    expect(invalidate.mock.calls.map((call) => call[0]?.queryKey)).toEqual([['items']]);
  });

  it('treats a removed chain as a full chain-state change', () => {
    const { queryClient, invalidate } = spyClient();

    invalidateForEvent(queryClient, { type: 'chain.removed', chainId: 3 });

    const keys = invalidate.mock.calls.map((call) => call[0]?.queryKey);
    expect(keys).toContainEqual(['chains']);
    expect(keys).toContainEqual(['exchange-options']);
    expect(keys).toContainEqual(['exchange-requests']);
    expect(keys).toContainEqual(['items']);
  });
});

describe('invalidateAllRealtime', () => {
  it('invalidates every realtime key after reconnect', () => {
    const { queryClient, invalidate } = spyClient();

    invalidateAllRealtime(queryClient);

    const keys = invalidate.mock.calls.map((call) => call[0]?.queryKey);
    expect(keys).toEqual([['chains'], ['exchange-options'], ['exchange-requests'], ['items']]);
  });
});
