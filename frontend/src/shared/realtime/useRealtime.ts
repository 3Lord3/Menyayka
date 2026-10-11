import { useEffect, useState } from 'react';

import { env } from '@shared/config/env';

import type { RealtimeStatus } from './client';
import { useRealtimeClient } from './context';

// useRealtimeChain subscribes to a single chain room while a detail/deal/replacement
// screen is mounted and unsubscribes on unmount or chainId change. Chain-wide events
// are also delivered to the user room, but the explicit subscription makes delivery on
// intense single-chain screens faster and gives an ack.
export function useRealtimeChain(chainId?: number): void {
  const client = useRealtimeClient();
  useEffect(() => {
    if (!env.realtimeEnabled || !client || !chainId) {
      return;
    }
    client.subscribe(chainId);
    return () => client.unsubscribe(chainId);
  }, [client, chainId]);
}

// useRealtimeConnected reports whether the push channel is actually live right now.
// Polling fallbacks key off this instead of a build-time flag: a build-time flag says
// nothing about a socket that a proxy, an exhausted connection limit or a fail-closed
// origin check just rejected: exactly the case where polling has to stay fast.
export function useRealtimeConnected(): boolean {
  const client = useRealtimeClient();
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!client) {
      return undefined;
    }
    return client.onConnectionChange(setConnected);
  }, [client]);

  return client ? connected : false;
}

// useRealtimeStatus exposes the full connection lifecycle so the UI can tell a user when
// the live channel is lost and being restored.
export function useRealtimeStatus(): RealtimeStatus {
  const client = useRealtimeClient();
  const [status, setStatus] = useState<RealtimeStatus>('disconnected');

  useEffect(() => {
    if (!client) {
      return undefined;
    }
    return client.onStatusChange(setStatus);
  }, [client]);

  return client ? status : 'disconnected';
}
