import { useSyncExternalStore } from 'react';

import { formatRemaining, isFrozenReplacement, needsMyAction, type Chain } from '@entities/chain';

// tick every 30 s — minute resolution is enough ("47 h 58 min"), and the server refetch updates
// the deadline itself; without a local tick the string would freeze
const TICK_MS = 30_000;

// all deadlines live in freezeDeadlineAt and differ only by chain status/reason,
// so the choice stays with the calling component
export type DeadlinePurpose = 'response' | 'ship' | 'replacement';

// the replacement deadline is shared, but for an invited candidate it counts their answer time

export function chainDeadlinePurpose(chain: Chain): DeadlinePurpose | undefined {
  if (!isFrozenReplacement(chain)) return undefined;
  return needsMyAction(chain) ? 'response' : 'replacement';
}

// an external store, not useState with an interval: Date.now() can't be read directly in
// render, and the snapshot refreshes also on subscription — a deadline arriving after the card
// mounted is counted from the current time. One interval per page.
let snapshot = Date.now();
let interval: ReturnType<typeof setInterval> | undefined;
const listeners = new Set<() => void>();

function subscribeToClock(listener: () => void): () => void {
  snapshot = Date.now();
  listeners.add(listener);
  if (interval === undefined) {
    interval = setInterval(() => {
      snapshot = Date.now();
      listeners.forEach((notify) => notify());
    }, TICK_MS);
  }
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) {
      clearInterval(interval);
      interval = undefined;
    }
  };
}

function subscribeToNothing(): () => void {
  return () => {};
}

function readClock(): number {
  return snapshot;
}

export function useDeadlineLabel(
  purpose: DeadlinePurpose | null,
  deadlineAt: string | null | undefined,
): string | null {
  const visible = purpose !== null && Boolean(deadlineAt);
  const now = useSyncExternalStore(visible ? subscribeToClock : subscribeToNothing, readClock);

  return visible ? formatRemaining(deadlineAt, now) : null;
}
