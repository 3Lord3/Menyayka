import { apiClient } from '@shared/api';
import { env } from '@shared/config/env';

import { isRealtimeControl, type RealtimeControl, type RealtimeEvent } from './events';

type EventListener = (event: RealtimeEvent) => void;
type ControlListener = (control: RealtimeControl) => void;
type ConnectionListener = (connected: boolean) => void;
type StatusListener = (status: RealtimeStatus) => void;

// RealtimeStatus is the user-visible connection lifecycle. `reconnecting` means an
// established connection dropped (or a first attempt failed) and the client keeps
// retrying: the state the UI has to surface, because the data is now stale.
export type RealtimeStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

// The server expires idle chain subscriptions after WS_SUB_TTL (30m by default); the
// app-level ping interval must stay well below it. Server Ping does not renew the TTL.
const APP_PING_MS = 9 * 60 * 1000;
const RECONNECT_BASE_MS = 1000;
const RECONNECT_MAX_MS = 30_000;
// A hanging ticket request must not strand the client in the `connecting` state forever.
const TICKET_TIMEOUT_MS = 10_000;
// WebSocket.OPEN per the WHATWG spec; the global constructor may not exist (node/tests).
const WS_OPEN = 1;

// wsBase derives the WebSocket origin: explicit VITE_API_BASE_URL wins, otherwise the
// connection is same-origin (dev proxy / prod reverse proxy).
function wsBase(): string {
  if (env.apiBaseUrl) {
    return env.apiBaseUrl.replace(/^http/, 'ws');
  }
  const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
  return `${scheme}://${window.location.host}`;
}

function hasWebSocket(): boolean {
  return typeof WebSocket !== 'undefined';
}

export class RealtimeClient {
  private socket: WebSocket | null = null;
  private readonly eventListeners = new Set<EventListener>();
  private readonly controlListeners = new Set<ControlListener>();
  private readonly connectionListeners = new Set<ConnectionListener>();
  private readonly statusListeners = new Set<StatusListener>();
  private status: RealtimeStatus = 'disconnected';
  // desired subscriptions survive reconnects and are replayed on open
  private readonly subscriptions = new Set<number>();
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAt: number | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private connecting = false;
  private manuallyClosed = false;
  private opened = false;
  private onReconnected: (() => void) | null = null;
  // A generation guards connect()/disconnect(): a stale open() awaiting its ticket must
  // not open a socket for a finished lifecycle.
  private generation = 0;

  /** Whether a socket is currently open. */
  get connected(): boolean {
    return this.socket !== null && this.socket.readyState === WS_OPEN;
  }

  /** Connection lifecycle state (drives the UI banner). */
  get connectionStatus(): RealtimeStatus {
    return this.status;
  }

  /** Milliseconds left before the next reconnect attempt, or null. */
  get nextRetryInMs(): number | null {
    return this.reconnectAt === null ? null : Math.max(0, this.reconnectAt - Date.now());
  }

  onEvent(listener: EventListener): () => void {
    this.eventListeners.add(listener);
    return () => this.eventListeners.delete(listener);
  }

  onControl(listener: ControlListener): () => void {
    this.controlListeners.add(listener);
    return () => this.controlListeners.delete(listener);
  }

  onConnectionChange(listener: ConnectionListener): () => void {
    this.connectionListeners.add(listener);
    listener(this.connected);
    return () => {
      this.connectionListeners.delete(listener);
    };
  }

  onStatusChange(listener: StatusListener): () => void {
    this.statusListeners.add(listener);
    listener(this.status);
    return () => {
      this.statusListeners.delete(listener);
    };
  }

  setOnReconnected(callback: () => void): void {
    this.onReconnected = callback;
  }

  connect(): void {
    if (!env.realtimeEnabled || !hasWebSocket()) return;
    // Latest call wins: connect() right after disconnect() must not be swallowed by a
    // still-pending open().
    this.manuallyClosed = false;
    if (this.connecting || this.socket) return;
    this.connecting = true;
    this.generation += 1;
    // A reconnect keeps the "reconnecting" state set on drop.
    if (this.reconnectAttempts === 0) {
      this.setStatus('connecting');
    }
    void this.open(this.generation);
  }

  disconnect(): void {
    this.manuallyClosed = true;
    this.generation += 1;
    this.connecting = false;
    this.clearReconnectTimer();
    this.stopPing();
    const socket = this.socket;
    this.socket = null;
    if (socket) {
      socket.onopen = null;
      socket.onmessage = null;
      socket.onclose = null;
      socket.onerror = null;
      socket.close();
    }
    this.reconnectAttempts = 0;
    this.opened = false;
    this.setStatus('disconnected');
    this.notifyConnection();
  }

  /**
   * Skips the remaining backoff and retries immediately (user pressed retry).
   */
  retryNow(): void {
    if (this.status !== 'reconnecting') return;
    this.clearReconnectTimer();
    this.connect();
  }

  /**
   * Drops desired subscriptions (called on logout: the client is app-lifetime).
   */
  clearSubscriptions(): void {
    this.subscriptions.clear();
  }

  subscribe(chainId: number): void {
    this.subscriptions.add(chainId);
    this.send({ action: 'subscribe', chainId });
  }

  unsubscribe(chainId: number): void {
    this.subscriptions.delete(chainId);
    this.send({ action: 'unsubscribe', chainId });
  }

  // refreshSubscriptions replays subscriptions dropped by TTL while the tab was hidden.
  refreshSubscriptions(): void {
    this.subscriptions.forEach((chainId) => this.send({ action: 'subscribe', chainId }));
  }

  private async open(generation: number): Promise<void> {
    try {
      const ticket = await this.fetchTicket();
      if (this.manuallyClosed || generation !== this.generation) return;

      const socket = new WebSocket(`${wsBase()}/api/v1/ws`, ['bearer', ticket]);
      if (generation !== this.generation) {
        socket.close();
        return;
      }
      this.socket = socket;
      socket.onopen = () => this.handleOpen(socket);
      socket.onmessage = (message) => this.handleMessage(socket, message);
      socket.onclose = () => this.handleClose(socket);
      socket.onerror = () => socket.close();
    } catch {
      if (generation === this.generation) {
        this.socket = null;
        this.setStatus('reconnecting');
        this.scheduleReconnect();
      }
    } finally {
      if (generation === this.generation) {
        this.connecting = false;
      }
    }
  }

  private async fetchTicket(): Promise<string> {
    const response = await apiClient.get<{ ticket: string }>('/ws/ticket', {
      timeout: TICKET_TIMEOUT_MS,
    });
    return response.data.ticket;
  }

  private handleOpen(socket: WebSocket): void {
    if (this.socket !== socket) return;
    this.reconnectAttempts = 0;
    this.startPing();
    // replay desired subscriptions
    this.subscriptions.forEach((chainId) => this.send({ action: 'subscribe', chainId }));
    // a reconnect may have missed events; invalidate everything realtime-aware
    if (this.opened) {
      this.onReconnected?.();
    }
    this.opened = true;
    this.setStatus('connected');
    this.notifyConnection();
  }

  private handleMessage(socket: WebSocket, message: MessageEvent): void {
    if (this.socket !== socket) return;
    let parsed: unknown;
    try {
      parsed = JSON.parse(message.data as string);
    } catch {
      return;
    }
    if (isRealtimeControl(parsed)) {
      this.handleControl(parsed);
      return;
    }
    const event = parsed as RealtimeEvent;
    this.eventListeners.forEach((listener) => listener(event));
  }

  private handleControl(control: RealtimeControl): void {
    // TTL-dropped subscription on an open screen: resubscribe transparently
    if (
      control.action === 'unsubscribed' &&
      control.reason === 'ttl' &&
      control.chainId !== undefined &&
      this.subscriptions.has(control.chainId)
    ) {
      this.send({ action: 'subscribe', chainId: control.chainId });
    }
    this.controlListeners.forEach((listener) => listener(control));
  }

  private handleClose(socket: WebSocket): void {
    if (this.socket !== socket) return;
    this.socket = null;
    this.stopPing();
    this.notifyConnection();
    if (!this.manuallyClosed) {
      this.setStatus('reconnecting');
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer || this.manuallyClosed) return;
    const exponential = Math.min(RECONNECT_MAX_MS, RECONNECT_BASE_MS * 2 ** this.reconnectAttempts);
    // jitter avoids a thundering herd
    const delay = exponential / 2 + Math.random() * (exponential / 2);
    this.reconnectAttempts += 1;
    this.reconnectAt = Date.now() + delay;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.reconnectAt = null;
      this.connect();
    }, delay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.reconnectAt = null;
  }

  private startPing(): void {
    this.stopPing();
    this.pingTimer = setInterval(() => this.send({ action: 'ping' }), APP_PING_MS);
  }

  private stopPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }

  private notifyConnection(): void {
    const connected = this.connected;
    this.connectionListeners.forEach((listener) => listener(connected));
  }

  private setStatus(status: RealtimeStatus): void {
    if (this.status === status) return;
    this.status = status;
    this.statusListeners.forEach((listener) => listener(status));
  }

  private send(payload: unknown): void {
    if (this.socket && this.socket.readyState === WS_OPEN) {
      this.socket.send(JSON.stringify(payload));
    }
  }
}
