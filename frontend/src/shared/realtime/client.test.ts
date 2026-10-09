import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// The env module reads import.meta.env at evaluation time, so the flag is stubbed
// before the client is imported dynamically.
vi.stubEnv('VITE_REALTIME_ENABLED', 'true');

class FakeWebSocket {
  static readonly OPEN = 1;
  static instances: FakeWebSocket[] = [];

  readonly url: string;
  readonly protocols: string[];
  readyState = 0;
  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn();

  constructor(url: string, protocols: string[] = []) {
    this.url = url;
    this.protocols = protocols;
    FakeWebSocket.instances.push(this);
  }

  simulateOpen() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.();
  }

  simulateMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }
}

async function importClient() {
  const { apiClient } = await import('@shared/api');
  vi.spyOn(apiClient, 'get').mockResolvedValue({ data: { ticket: 'ticket-1' } } as never);
  const { RealtimeClient } = await import('./client');
  return { RealtimeClient, apiClient };
}

beforeEach(() => {
  FakeWebSocket.instances = [];
  vi.stubGlobal('WebSocket', FakeWebSocket);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('RealtimeClient', () => {
  it('connects with the one-time ticket as a subprotocol and replays subscriptions', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();
    const events: unknown[] = [];
    client.onEvent((event) => events.push(event));

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));

    const socket = FakeWebSocket.instances[0];
    socket.simulateOpen();
    expect(socket.url).toContain('/api/v1/ws');
    expect(socket.protocols).toEqual(['bearer', 'ticket-1']);

    client.subscribe(42);
    expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ action: 'subscribe', chainId: 42 }));

    socket.simulateMessage({ type: 'chain.vote_changed', chainId: 42 });
    expect(events).toHaveLength(1);

    client.disconnect();
    expect(socket.close).toHaveBeenCalled();
  });

  it('resubscribes when the server drops a subscription by TTL', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    const socket = FakeWebSocket.instances[0];
    socket.simulateOpen();

    client.subscribe(7);
    socket.send.mockClear();

    socket.simulateMessage({ action: 'unsubscribed', chainId: 7, reason: 'ttl' });

    expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ action: 'subscribe', chainId: 7 }));
  });

  // StrictMode setup->cleanup->setup: an in-flight ticket request must not strand the client.
  it('connects again when disconnect+connect happen while the ticket is in flight', async () => {
    const { apiClient } = await import('@shared/api');
    let resolveTicket: ((value: unknown) => void) | undefined;
    vi.spyOn(apiClient, 'get').mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveTicket = resolve;
        }) as never,
    );
    const { RealtimeClient } = await import('./client');
    const client = new RealtimeClient();

    client.connect();
    client.disconnect();
    client.connect();
    resolveTicket?.({ data: { ticket: 'ticket-2' } });

    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    expect(FakeWebSocket.instances[0].protocols).toEqual(['bearer', 'ticket-2']);
  });

  it('reports the live connection state', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();
    const states: boolean[] = [];
    client.onConnectionChange((connected) => states.push(connected));
    expect(client.connected).toBe(false);

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    FakeWebSocket.instances[0].simulateOpen();
    expect(client.connected).toBe(true);

    client.disconnect();
    expect(client.connected).toBe(false);
    // the initial false is pushed by onConnectionChange itself
    expect(states).toEqual([false, true, false]);
  });

  it('does not replay cleared subscriptions after a re-login', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    FakeWebSocket.instances[0].simulateOpen();
    client.subscribe(7);

    // logout: the socket dies and the desired subscriptions go with the session
    client.disconnect();
    client.clearSubscriptions();

    // another account connects on the same browser
    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(2));
    const next = FakeWebSocket.instances[1];
    next.simulateOpen();

    expect(next.send).not.toHaveBeenCalledWith(JSON.stringify({ action: 'subscribe', chainId: 7 }));
  });

  it('reports the connection status transitions', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();
    const statuses: string[] = [];
    client.onStatusChange((status) => statuses.push(status));
    expect(client.connectionStatus).toBe('disconnected');

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    expect(client.connectionStatus).toBe('connecting');

    FakeWebSocket.instances[0].simulateOpen();
    expect(client.connectionStatus).toBe('connected');

    // an unexpected close flips to reconnecting until the retry
    FakeWebSocket.instances[0].onclose?.();
    expect(client.connectionStatus).toBe('reconnecting');

    client.disconnect();
    expect(client.connectionStatus).toBe('disconnected');
    expect(statuses).toEqual([
      'disconnected',
      'connecting',
      'connected',
      'reconnecting',
      'disconnected',
    ]);
  });

  it('invalidates realtime state after a reconnect', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();
    const onReconnected = vi.fn();
    client.setOnReconnected(onReconnected);

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    FakeWebSocket.instances[0].simulateOpen();
    expect(onReconnected).not.toHaveBeenCalled();

    FakeWebSocket.instances[0].onclose?.();
    // the scheduled retry opens a second socket
    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(2));
    FakeWebSocket.instances[1].simulateOpen();
    expect(onReconnected).toHaveBeenCalledTimes(1);

    client.disconnect();
  });

  it('retries on demand instead of waiting out the backoff delay', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    FakeWebSocket.instances[0].simulateOpen();

    // nothing to retry while the socket is live
    client.retryNow();
    expect(FakeWebSocket.instances).toHaveLength(1);

    FakeWebSocket.instances[0].onclose?.();
    expect(client.connectionStatus).toBe('reconnecting');

    // the backoff pause is skipped; the state stays "reconnecting"
    client.retryNow();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(2));
    expect(client.connectionStatus).toBe('reconnecting');

    client.disconnect();
  });

  it('reports the time left before the scheduled retry', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    FakeWebSocket.instances[0].simulateOpen();
    expect(client.nextRetryInMs).toBeNull();

    FakeWebSocket.instances[0].onclose?.();
    // first step is the base delay with jitter: half at the least
    expect(client.nextRetryInMs).toBeGreaterThan(0);
    expect(client.nextRetryInMs).toBeLessThanOrEqual(1000);

    // an on-demand retry cancels the pending attempt
    client.retryNow();
    expect(client.nextRetryInMs).toBeNull();

    client.disconnect();
  });

  it('sends an app-level ping while the socket is open', async () => {
    vi.useFakeTimers();
    try {
      const { RealtimeClient } = await importClient();
      const client = new RealtimeClient();
      client.connect();
      await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
      const socket = FakeWebSocket.instances[0];
      socket.simulateOpen();
      socket.send.mockClear();

      await vi.advanceTimersByTimeAsync(9 * 60 * 1000);

      expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ action: 'ping' }));
      client.disconnect();
    } finally {
      vi.useRealTimers();
    }
  });

  it('sends unsubscribe and replays only the remaining subscriptions', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    const socket = FakeWebSocket.instances[0];
    socket.simulateOpen();

    client.subscribe(1);
    client.subscribe(2);
    socket.send.mockClear();

    client.unsubscribe(1);
    client.refreshSubscriptions();

    expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ action: 'unsubscribe', chainId: 1 }));
    expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ action: 'subscribe', chainId: 2 }));
    // the dropped room must not be resurrected by the replay
    expect(socket.send).not.toHaveBeenCalledWith(
      JSON.stringify({ action: 'subscribe', chainId: 1 }),
    );
  });

  it('forwards control messages and ignores malformed payloads', async () => {
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();
    const controls: unknown[] = [];
    const events: unknown[] = [];
    client.onControl((control) => controls.push(control));
    client.onEvent((event) => events.push(event));

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));
    const socket = FakeWebSocket.instances[0];
    socket.simulateOpen();

    socket.simulateMessage({ action: 'error', code: 'rate_limited', message: 'slow down' });
    expect(controls).toEqual([{ action: 'error', code: 'rate_limited', message: 'slow down' }]);

    // a TTL drop for a room we never subscribed to must not trigger a resubscribe
    socket.send.mockClear();
    socket.simulateMessage({ action: 'unsubscribed', chainId: 999, reason: 'ttl' });
    expect(socket.send).not.toHaveBeenCalled();

    socket.onmessage?.({ data: '{not json' } as MessageEvent);
    expect(events).toHaveLength(0);

    socket.simulateMessage({ type: 'chain.vote_changed', chainId: 1 });
    expect(events).toHaveLength(1);
  });

  // a hanging/failed ticket request must not strand the client in `connecting`
  it('retries with a fresh ticket after a failed ticket request', async () => {
    vi.useFakeTimers();
    try {
      const { apiClient } = await import('@shared/api');
      const get = vi.spyOn(apiClient, 'get');
      get.mockRejectedValueOnce(new Error('ticket unavailable'));
      get.mockResolvedValue({ data: { ticket: 'ticket-retry' } } as never);
      const { RealtimeClient } = await import('./client');
      const client = new RealtimeClient();

      client.connect();
      await vi.advanceTimersByTimeAsync(0);
      expect(client.connectionStatus).toBe('reconnecting');

      // the backoff retry opens a new socket
      await vi.advanceTimersByTimeAsync(1_000);
      await vi.advanceTimersByTimeAsync(0);
      expect(FakeWebSocket.instances).toHaveLength(1);
      expect(FakeWebSocket.instances[0].protocols).toEqual(['bearer', 'ticket-retry']);

      client.disconnect();
    } finally {
      vi.useRealTimers();
    }
  });

  it('never opens a socket when the environment has no WebSocket', async () => {
    vi.stubGlobal('WebSocket', undefined);
    const { RealtimeClient } = await importClient();
    const client = new RealtimeClient();

    client.connect();
    await Promise.resolve();

    expect(FakeWebSocket.instances).toHaveLength(0);
    expect(client.connectionStatus).toBe('disconnected');
  });
});
