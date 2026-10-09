import { act, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { RealtimeClient, RealtimeStatus } from './client';
import { RealtimeContext } from './context';
import { RealtimeStatusBanner } from './RealtimeStatusBanner';
import { RealtimeStatusToast } from './RealtimeStatusToast';

// A minimal stand-in for RealtimeClient that only exposes the status channel the feedback
// components subscribe to; wiring it through the context keeps the tests free of a real
// socket.
function createFakeClient(initial: RealtimeStatus) {
  const listeners = new Set<(status: RealtimeStatus) => void>();
  const client = {
    connectionStatus: initial,
    nextRetryInMs: null as number | null,
    retryNow: vi.fn(),
    onStatusChange(listener: (status: RealtimeStatus) => void) {
      listeners.add(listener);
      listener(client.connectionStatus);
      return () => listeners.delete(listener);
    },
    emit(status: RealtimeStatus) {
      client.connectionStatus = status;
      listeners.forEach((listener) => listener(status));
    },
  };
  return client;
}

function renderFeedback(status: RealtimeStatus) {
  const client = createFakeClient(status);
  const view = render(
    <RealtimeContext.Provider value={client as unknown as RealtimeClient}>
      <RealtimeStatusBanner />
      <RealtimeStatusToast />
    </RealtimeContext.Provider>,
  );
  return { client, ...view };
}

describe('realtime status feedback', () => {
  it('warns the user while the connection is being restored', () => {
    renderFeedback('reconnecting');

    const notices = screen.getAllByRole('status');
    expect(notices).toHaveLength(2);
    expect(notices[0]).toHaveTextContent('Связь потеряна. Переподключение…');
    expect(notices[1]).toHaveTextContent('Связь потеряна');
    expect(notices[1]).toHaveTextContent('Переподключение…');
  });

  it('counts the seconds down to the scheduled reconnect attempt', () => {
    vi.useFakeTimers();
    try {
      const { client } = renderFeedback('reconnecting');
      // no pending attempt means no countdown, text has no time
      expect(screen.getAllByRole('status')[0]).toHaveTextContent('Переподключение…');

      act(() => {
        client.nextRetryInMs = 8000;
        vi.advanceTimersByTime(1000);
      });
      expect(screen.getAllByRole('status')[0]).toHaveTextContent(
        'Связь потеряна. Переподключение через 8 с',
      );

      act(() => {
        client.nextRetryInMs = 3000;
        vi.advanceTimersByTime(1000);
      });
      expect(screen.getAllByRole('status')[0]).toHaveTextContent(
        'Связь потеряна. Переподключение через 3 с',
      );
    } finally {
      vi.useRealTimers();
    }
  });

  it('retries the connection when the user presses "Повторить"', async () => {
    const { client } = renderFeedback('reconnecting');

    screen.getByRole('button', { name: 'Повторить' }).click();

    expect(client.retryNow).toHaveBeenCalledTimes(1);
  });

  it('stays silent while the socket is live', () => {
    renderFeedback('connected');

    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('stays silent when realtime is idle (logged out / disabled)', () => {
    renderFeedback('disconnected');

    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('confirms the recovery, then fades out', async () => {
    vi.useFakeTimers();
    try {
      const { client } = renderFeedback('connected');

      act(() => client.emit('reconnecting'));
      expect(screen.getAllByRole('status')).toHaveLength(2);

      act(() => client.emit('connected'));
      const notices = screen.getAllByRole('status');
      expect(notices[0]).toHaveTextContent('Связь восстановлена. Данные обновлены');
      expect(notices[1]).toHaveTextContent('Связь восстановлена');
      expect(notices[1]).toHaveTextContent('Данные обновлены');

      await act(async () => {
        await vi.advanceTimersByTimeAsync(3000);
      });
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('does not show the recovery notice on the initial connection', () => {
    const { client } = renderFeedback('connecting');

    act(() => client.emit('connected'));

    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });
});
