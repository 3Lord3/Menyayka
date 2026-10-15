import { useEffect, useRef, useState } from 'react';

import type { RealtimeStatus } from './client';
import { useRealtimeStatus } from './useRealtime';

// How long the recovery notice stays on screen after the socket is back.
const RESTORED_MS = 3_000;

interface RealtimeFeedback {
  status: RealtimeStatus;
  /** True for a few seconds right after a reconnect succeeded. */
  restored: boolean;
}

// useRealtimeFeedback turns the raw socket lifecycle into what the UI shows: the current
// status plus a short-lived flag confirming the recovery. Without that flag a reconnected
// socket looks like nothing happened at all; the data simply stops being stale.
export function useRealtimeFeedback(): RealtimeFeedback {
  const status = useRealtimeStatus();
  const [restored, setRestored] = useState(false);
  const previous = useRef<RealtimeStatus>(status);

  useEffect(() => {
    const wasReconnecting = previous.current === 'reconnecting';
    previous.current = status;
    if (!wasReconnecting || status !== 'connected') {
      return undefined;
    }
    setRestored(true);
    const timer = setTimeout(() => setRestored(false), RESTORED_MS);
    return () => clearTimeout(timer);
  }, [status]);

  return { status, restored };
}
