/**
 * WebSocket type definitions for the socket plugin.
 * Used by the useWebSocket hook across all platforms.
 */

export interface WSMessage {
  type: string;
  channel: string;
  payload: unknown;
  timestamp: string;
}

export interface WSOptions {
  url?: string;
  reconnect?: boolean;
  maxRetries?: number;
  channels?: string[];
}

export type WSStatus = 'connecting' | 'connected' | 'disconnected' | 'error';
