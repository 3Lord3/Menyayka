// Mirror of the backend event contract (internal/realtime/event.go). Keep in sync.

export type RealtimeEventType =
  | 'chain.created'
  | 'chain.status_changed'
  | 'chain.vote_changed'
  | 'chain.replacement_available'
  | 'chain.replacement_selected'
  | 'chain.expired'
  | 'chain.fulfillment_changed'
  | 'chain.removed'
  | 'request.status_changed'
  | 'request.updated'
  | 'item.status_changed';

export interface RealtimeEventPayload {
  chainId?: number;
  requestId?: number;
  itemId?: number;
  status?: string;
  targetRequestId?: number;
  replacementRequestId?: number;
  releasedRequestIds?: number[];
}

export interface RealtimeEvent {
  type: RealtimeEventType;
  chainId?: number;
  payload?: RealtimeEventPayload;
  at?: string;
}

export type RealtimeAction = 'subscribed' | 'unsubscribed' | 'pong' | 'error';

export interface RealtimeControl {
  action: RealtimeAction;
  chainId?: number;
  reason?: string;
  code?: string;
  message?: string;
}

export function isRealtimeControl(value: unknown): value is RealtimeControl {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as { action?: unknown }).action === 'string'
  );
}
