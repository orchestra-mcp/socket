/**
 * WebSocket client hook for the socket plugin.
 * Singleton connection with auto-reconnect, offline queue, and channel pub/sub.
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import type { WSMessage, WSOptions, WSStatus } from '../types/websocket';

// ── Singleton State ───────────────────────────────────────────────

type Listener = (message: WSMessage) => void;

let socket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let retryCount = 0;

const listeners = new Map<string, Set<Listener>>();
const offlineQueue: string[] = [];

const DEFAULT_URL = 'ws://localhost:8080/ws';
const MAX_RETRY_DELAY = 30_000;

// ── Helpers ───────────────────────────────────────────────────────

function getRetryDelay(attempt: number): number {
  return Math.min(1000 * Math.pow(2, attempt), MAX_RETRY_DELAY);
}

function flushQueue(): void {
  while (offlineQueue.length > 0 && socket?.readyState === WebSocket.OPEN) {
    const msg = offlineQueue.shift();
    if (msg) socket.send(msg);
  }
}

function dispatch(message: WSMessage): void {
  const channelListeners = listeners.get(message.channel);
  if (channelListeners) {
    channelListeners.forEach((fn) => fn(message));
  }
  const wildcardListeners = listeners.get('*');
  if (wildcardListeners) {
    wildcardListeners.forEach((fn) => fn(message));
  }
}

// ── Connection Manager ────────────────────────────────────────────

function connect(
  url: string,
  maxRetries: number,
  onStatus: (status: WSStatus) => void,
): void {
  if (socket?.readyState === WebSocket.OPEN) return;

  onStatus('connecting');
  socket = new WebSocket(url);

  socket.onopen = () => {
    retryCount = 0;
    onStatus('connected');
    flushQueue();
  };

  socket.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data) as WSMessage;
      dispatch(message);
    } catch {
      // Ignore malformed messages
    }
  };

  socket.onclose = () => {
    onStatus('disconnected');
    socket = null;
    if (retryCount < maxRetries) {
      const delay = getRetryDelay(retryCount);
      retryCount++;
      reconnectTimer = setTimeout(() => connect(url, maxRetries, onStatus), delay);
    }
  };

  socket.onerror = () => {
    onStatus('error');
  };
}

function disconnect(): void {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (socket) {
    socket.onclose = null;
    socket.close();
    socket = null;
  }
  retryCount = 0;
}

// ── Hook ──────────────────────────────────────────────────────────

export function useWebSocket(options?: WSOptions) {
  const {
    url = DEFAULT_URL,
    reconnect = true,
    maxRetries = 10,
    channels = [],
  } = options ?? {};

  const [status, setStatus] = useState<WSStatus>('disconnected');
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);
  const channelsRef = useRef(channels);
  channelsRef.current = channels;

  // Connect on mount, disconnect when last consumer unmounts
  useEffect(() => {
    if (!reconnect && socket) return;
    connect(url, maxRetries, setStatus);
    // We do not disconnect on unmount because the socket is shared.
    // Disconnection is handled explicitly by the app.
  }, [url, maxRetries, reconnect]);

  // Track last message for subscribed channels
  useEffect(() => {
    const handler: Listener = (msg) => {
      if (
        channelsRef.current.length === 0 ||
        channelsRef.current.includes(msg.channel)
      ) {
        setLastMessage(msg);
      }
    };
    const wildcard = listeners.get('*') ?? new Set();
    wildcard.add(handler);
    listeners.set('*', wildcard);

    return () => {
      wildcard.delete(handler);
      if (wildcard.size === 0) listeners.delete('*');
    };
  }, []);

  const send = useCallback((message: WSMessage) => {
    const serialized = JSON.stringify(message);
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(serialized);
    } else {
      offlineQueue.push(serialized);
    }
  }, []);

  const subscribe = useCallback((channel: string, listener: Listener) => {
    const set = listeners.get(channel) ?? new Set();
    set.add(listener);
    listeners.set(channel, set);
    return () => {
      set.delete(listener);
      if (set.size === 0) listeners.delete(channel);
    };
  }, []);

  const unsubscribe = useCallback((channel: string, listener: Listener) => {
    const set = listeners.get(channel);
    if (set) {
      set.delete(listener);
      if (set.size === 0) listeners.delete(channel);
    }
  }, []);

  return { status, send, subscribe, unsubscribe, lastMessage, disconnect };
}
