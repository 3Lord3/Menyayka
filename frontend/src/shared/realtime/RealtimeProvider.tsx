import { useEffect, useState, type ReactNode } from 'react';

import { useQueryClient } from '@tanstack/react-query';
import { useSelector } from 'react-redux';

import { env } from '@shared/config/env';

import { RealtimeClient } from './client';
import { RealtimeContext } from './context';
import { invalidateAllRealtime, invalidateForEvent } from './invalidate';

// The token slice shape is read through react-redux (not @app/*) so this shared
// provider stays inside the shared FSD layer.
function selectToken(state: { user: { token: string | null } }): string | null {
  return state.user.token;
}

// RealtimeProvider owns the socket lifecycle, bound to the session token. When
// VITE_REALTIME_ENABLED=false it never connects and the polling fallback stays active.
export function RealtimeProvider({ children }: { children: ReactNode }) {
  const [client] = useState(() => new RealtimeClient());
  const token = useSelector(selectToken);
  const queryClient = useQueryClient();

  // event -> invalidation mapping lives for the whole app lifetime
  useEffect(() => {
    return client.onEvent((event) => invalidateForEvent(queryClient, event));
  }, [client, queryClient]);

  useEffect(() => {
    client.setOnReconnected(() => invalidateAllRealtime(queryClient));
  }, [client, queryClient]);

  useEffect(() => {
    if (!env.realtimeEnabled || !token) {
      client.disconnect();
      // drop subscriptions at session end, else a later login keeps the previous account's rooms
      client.clearSubscriptions();
      return;
    }
    client.connect();
    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        client.refreshSubscriptions();
      }
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      document.removeEventListener('visibilitychange', onVisible);
      client.disconnect();
    };
  }, [client, token]);

  return <RealtimeContext.Provider value={client}>{children}</RealtimeContext.Provider>;
}
