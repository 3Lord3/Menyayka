import type { ReactNode } from 'react';

import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { RealtimeClient, RealtimeStatus } from './client';
import { RealtimeContext } from './context';

// useRealtime* gate on the build flag, so it is stubbed before the module is imported.
vi.stubEnv('VITE_REALTIME_ENABLED', 'true');

function wrapperFor(client: unknown) {
  return ({ children }: { children: ReactNode }) => (
    <RealtimeContext.Provider value={client as RealtimeClient}>{children}</RealtimeContext.Provider>
  );
}

describe('useRealtimeChain', () => {
  it('subscribes to the chain and unsubscribes on unmount', async () => {
    const { useRealtimeChain } = await import('./useRealtime');
    const client = { subscribe: vi.fn(), unsubscribe: vi.fn() };

    const { unmount } = renderHook(() => useRealtimeChain(5), {
      wrapper: wrapperFor(client),
    });

    expect(client.subscribe).toHaveBeenCalledWith(5);

    unmount();
    expect(client.unsubscribe).toHaveBeenCalledWith(5);
  });

  it('resubscribes when the chain changes', async () => {
    const { useRealtimeChain } = await import('./useRealtime');
    const client = { subscribe: vi.fn(), unsubscribe: vi.fn() };

    const { rerender } = renderHook(({ chainId }) => useRealtimeChain(chainId), {
      initialProps: { chainId: 5 as number | undefined },
      wrapper: wrapperFor(client),
    });

    rerender({ chainId: 6 });

    expect(client.unsubscribe).toHaveBeenCalledWith(5);
    expect(client.subscribe).toHaveBeenCalledWith(6);
  });

  it('does not subscribe without a chain id', async () => {
    const { useRealtimeChain } = await import('./useRealtime');
    const client = { subscribe: vi.fn(), unsubscribe: vi.fn() };

    renderHook(() => useRealtimeChain(undefined), { wrapper: wrapperFor(client) });

    expect(client.subscribe).not.toHaveBeenCalled();
  });
});

describe('useRealtimeStatus', () => {
  it('tracks every status change', async () => {
    const { useRealtimeStatus } = await import('./useRealtime');
    const listeners = new Set<(status: RealtimeStatus) => void>();
    const client = {
      connectionStatus: 'connecting' as RealtimeStatus,
      onStatusChange(listener: (status: RealtimeStatus) => void) {
        listeners.add(listener);
        listener(client.connectionStatus);
        return () => listeners.delete(listener);
      },
    };

    const { result } = renderHook(() => useRealtimeStatus(), {
      wrapper: wrapperFor(client),
    });
    expect(result.current).toBe('connecting');

    act(() => {
      client.connectionStatus = 'reconnecting';
      listeners.forEach((listener) => listener('reconnecting'));
    });

    expect(result.current).toBe('reconnecting');
  });

  it('reports disconnected when there is no client', async () => {
    const { useRealtimeStatus } = await import('./useRealtime');

    const { result } = renderHook(() => useRealtimeStatus(), { wrapper: wrapperFor(null) });

    expect(result.current).toBe('disconnected');
  });
});

describe('useRealtimeConnected', () => {
  it('reports false without a client', async () => {
    const { useRealtimeConnected } = await import('./useRealtime');

    const { result } = renderHook(() => useRealtimeConnected(), { wrapper: wrapperFor(null) });

    expect(result.current).toBe(false);
  });

  it('tracks the live connection state', async () => {
    const { useRealtimeConnected } = await import('./useRealtime');
    const listeners = new Set<(connected: boolean) => void>();
    const client = {
      onConnectionChange(listener: (connected: boolean) => void) {
        listeners.add(listener);
        listener(false);
        return () => listeners.delete(listener);
      },
    };

    const { result } = renderHook(() => useRealtimeConnected(), { wrapper: wrapperFor(client) });
    expect(result.current).toBe(false);

    act(() => {
      listeners.forEach((listener) => listener(true));
    });

    expect(result.current).toBe(true);
  });
});
