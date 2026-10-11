import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

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
}

beforeEach(() => {
  FakeWebSocket.instances = [];
  vi.resetModules();
  vi.stubGlobal('WebSocket', FakeWebSocket);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
});

// the env module reads import.meta.env at evaluation time, so it is imported fresh per case
async function loadClient(environment: Record<string, string>) {
  Object.entries(environment).forEach(([key, value]) => vi.stubEnv(key, value));
  const { apiClient } = await import('@shared/api');
  const get = vi.spyOn(apiClient, 'get').mockResolvedValue({ data: { ticket: 't' } } as never);
  const { RealtimeClient } = await import('./client');
  return { client: new RealtimeClient(), get };
}

describe('RealtimeClient environment handling', () => {
  it('derives a wss endpoint from an explicit API base URL', async () => {
    const { client } = await loadClient({
      VITE_REALTIME_ENABLED: 'true',
      VITE_API_BASE_URL: 'https://api.example.com',
    });

    client.connect();
    await vi.waitFor(() => expect(FakeWebSocket.instances).toHaveLength(1));

    expect(FakeWebSocket.instances[0].url).toBe('wss://api.example.com/api/v1/ws');
    client.disconnect();
  });

  it('neither requests a ticket nor opens a socket when realtime is disabled', async () => {
    const { client, get } = await loadClient({
      VITE_REALTIME_ENABLED: 'false',
      VITE_API_BASE_URL: '',
    });

    client.connect();
    await Promise.resolve();

    expect(get).not.toHaveBeenCalled();
    expect(FakeWebSocket.instances).toHaveLength(0);
    expect(client.connectionStatus).toBe('disconnected');
  });
});
