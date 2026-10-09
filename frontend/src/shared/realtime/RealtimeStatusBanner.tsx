import { useEffect, useState } from 'react';

import { useRealtimeClient } from './context';
import { useRealtimeFeedback } from './useRealtimeFeedback';
import './RealtimeStatusBanner.scss';

// Seconds left before the reconnect attempt, ticked per second (the delay is randomized).
function useRetryCountdown(active: boolean): number | null {
  const client = useRealtimeClient();
  const [seconds, setSeconds] = useState<number | null>(null);

  useEffect(() => {
    if (!client || !active) {
      return undefined;
    }
    const tick = () => {
      const remaining = client.nextRetryInMs;
      setSeconds(remaining === null ? null : Math.max(1, Math.ceil(remaining / 1000)));
    };
    const initial = setTimeout(tick, 0);
    const timer = setInterval(tick, 1000);
    return () => {
      clearTimeout(initial);
      clearInterval(timer);
    };
  }, [client, active]);

  return active ? seconds : null;
}

// RealtimeStatusBanner is the desktop feedback of the push channel (mobile twin:
// RealtimeStatusToast). Warns while reconnecting and confirms recovery; informational,
// polling runs underneath either way.
export function RealtimeStatusBanner() {
  const client = useRealtimeClient();
  const { status, restored } = useRealtimeFeedback();
  const seconds = useRetryCountdown(status === 'reconnecting');

  if (status === 'reconnecting') {
    return (
      <div className="realtime-banner realtime-banner--offline" role="status" aria-live="polite">
        <span className="realtime-banner__text">
          Связь потеряна. Переподключение
          {seconds === null ? '…' : ` через ${seconds} с`}
        </span>
        <button
          type="button"
          className="realtime-banner__action"
          onClick={() => client?.retryNow()}
        >
          Повторить
        </button>
      </div>
    );
  }

  if (restored && status === 'connected') {
    return (
      <div className="realtime-banner realtime-banner--restored" role="status" aria-live="polite">
        <span className="realtime-banner__text">Связь восстановлена. Данные обновлены</span>
      </div>
    );
  }

  return null;
}
