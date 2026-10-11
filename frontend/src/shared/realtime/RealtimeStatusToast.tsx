import { useRealtimeFeedback } from './useRealtimeFeedback';
import './RealtimeStatusToast.scss';

// RealtimeStatusToast is the mobile feedback of the push channel (desktop twin:
// RealtimeStatusBanner): shows connection lost/restored as a short-lived toast above
// the bottom navigation.
export function RealtimeStatusToast() {
  const { status, restored } = useRealtimeFeedback();

  if (status === 'reconnecting') {
    return (
      <div className="realtime-toast" role="status" aria-live="polite">
        <span className="realtime-toast__dot realtime-toast__dot--warning" aria-hidden />
        <span className="realtime-toast__title">Связь потеряна</span>
        <span className="realtime-toast__subtitle">Переподключение…</span>
      </div>
    );
  }

  if (restored && status === 'connected') {
    return (
      <div className="realtime-toast" role="status" aria-live="polite">
        <span className="realtime-toast__dot realtime-toast__dot--success" aria-hidden />
        <span className="realtime-toast__title">Связь восстановлена</span>
        <span className="realtime-toast__subtitle">Данные обновлены</span>
      </div>
    );
  }

  return null;
}
