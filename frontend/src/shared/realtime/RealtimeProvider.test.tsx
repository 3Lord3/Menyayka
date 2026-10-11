import { QueryClientProvider } from '@tanstack/react-query';
import { act, render } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { createTestQueryClient } from '@shared/testing/renderWithProviders';

import { RealtimeProvider } from './RealtimeProvider';

const mocks = vi.hoisted(() => ({
  realtimeEnabled: true,
  token: 'jwt' as string | null,
  client: {
    connect: vi.fn(),
    disconnect: vi.fn(),
    clearSubscriptions: vi.fn(),
    refreshSubscriptions: vi.fn(),
    onEvent: vi.fn(),
    setOnReconnected: vi.fn(),
  },
  invalidateForEvent: vi.fn(),
  invalidateAllRealtime: vi.fn(),
}));

vi.mock('@shared/config/env', () => ({
  env: {
    apiBaseUrl: '',
    get realtimeEnabled() {
      return mocks.realtimeEnabled;
    },
  },
}));

// the provider reads the session token through react-redux; the selector is stubbed so the
// shared-layer test does not have to reach into the app store (FSD boundary)
vi.mock('react-redux', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-redux')>();
  return { ...actual, useSelector: () => mocks.token };
});

vi.mock('./client', () => ({
  RealtimeClient: vi.fn(function MockRealtimeClient() {
    return mocks.client;
  }),
}));
vi.mock('./invalidate', () => ({
  invalidateForEvent: mocks.invalidateForEvent,
  invalidateAllRealtime: mocks.invalidateAllRealtime,
}));

let eventListener: ((event: unknown) => void) | undefined;
let queryClient = createTestQueryClient();

function renderProvider() {
  return render(
    <QueryClientProvider client={queryClient}>
      <RealtimeProvider>
        <div />
      </RealtimeProvider>
    </QueryClientProvider>,
  );
}

describe('RealtimeProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.realtimeEnabled = true;
    mocks.token = 'jwt';
    queryClient = createTestQueryClient();
    eventListener = undefined;
    mocks.client.onEvent.mockImplementation((listener: (event: unknown) => void) => {
      eventListener = listener;
      return () => {};
    });
  });

  it('connects while a session token is present and disconnects on unmount', () => {
    const { unmount } = renderProvider();

    expect(mocks.client.connect).toHaveBeenCalledTimes(1);
    expect(mocks.client.disconnect).not.toHaveBeenCalled();

    unmount();
    expect(mocks.client.disconnect).toHaveBeenCalled();
  });

  it('stays idle and drops subscriptions once the token is gone', () => {
    mocks.token = null;

    renderProvider();

    expect(mocks.client.connect).not.toHaveBeenCalled();
    expect(mocks.client.disconnect).toHaveBeenCalled();
    expect(mocks.client.clearSubscriptions).toHaveBeenCalled();
  });

  it('stays idle when realtime is disabled', () => {
    mocks.realtimeEnabled = false;

    renderProvider();

    expect(mocks.client.connect).not.toHaveBeenCalled();
  });

  it('invalidates queries for incoming events', () => {
    renderProvider();

    act(() => eventListener?.({ type: 'chain.created' }));

    expect(mocks.invalidateForEvent).toHaveBeenCalledWith(queryClient, { type: 'chain.created' });
  });

  it('invalidates everything realtime-aware after a reconnect', () => {
    renderProvider();

    mocks.client.setOnReconnected.mock.calls[0][0]();

    expect(mocks.invalidateAllRealtime).toHaveBeenCalledWith(queryClient);
  });

  it('replays subscriptions when the tab becomes visible again', () => {
    renderProvider();

    act(() => {
      document.dispatchEvent(new Event('visibilitychange'));
    });

    expect(mocks.client.refreshSubscriptions).toHaveBeenCalled();
  });
});
