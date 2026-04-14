import { decode, encode } from '@msgpack/msgpack';
import type { WireEnvelope, WireStateUpdate, WireWelcome, WireError, WireBehaviorEvent } from '../types';
import { MsgType } from '../types';
import { useGameStore } from '../store/gameStore';
import { getToken } from './api';

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let selectedHull: string | null = null;

function buildWsUrl(): string {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
  const params = new URLSearchParams();
  const token = getToken();
  if (token) {
    params.set('token', token);
  }
  if (selectedHull) {
    params.set('hull', selectedHull);
  }
  const qs = params.toString();
  return `${proto}://${window.location.host}/ws${qs ? '?' + qs : ''}`;
}

function onMessage(event: MessageEvent) {
  const data = event.data;
  if (!(data instanceof ArrayBuffer)) return;

  const envelope = decode(new Uint8Array(data)) as WireEnvelope;

  // envelope.p is already decoded (Go's RawMessage embeds bytes inline),
  // so we cast directly instead of calling decode() again.
  switch (envelope.t) {
    case MsgType.Welcome: {
      const welcome = envelope.p as WireWelcome;
      useGameStore.getState().applyWelcome(welcome);
      console.log('[ws] welcome — player:', welcome.pid);
      break;
    }
    case MsgType.StateUpdate: {
      const update = envelope.p as WireStateUpdate;
      useGameStore.getState().applyStateUpdate(update);
      break;
    }
    case MsgType.Event: {
      const event = envelope.p as WireBehaviorEvent;
      useGameStore.getState().applyBehaviorEvent(event);
      console.log('[ws] behavior applied:', event.m);
      break;
    }
    case MsgType.Error: {
      const error = envelope.p as WireError;
      useGameStore.getState().setError(error.msg, error.cd);
      console.warn('[ws] server error:', error.msg);
      break;
    }
  }
}

/** Send a prompt command to the server. */
export function sendPrompt(text: string) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  // Don't pre-encode the inner payload — let the outer encode() serialize
  // { text } as a msgpack map.  Go's RawMessage captures the raw bytes and
  // Unmarshal decodes them as PromptPayload.  Pre-encoding would produce
  // a Uint8Array that the outer encoder writes as bin format (code c4),
  // which the server can't unmarshal as a map.
  const envelope = { t: MsgType.Prompt, p: { text } };
  ws.send(encode(envelope));
}

export function connect(hullId?: string) {
  if (ws && ws.readyState === WebSocket.OPEN) return;
  if (hullId) selectedHull = hullId;

  ws = new WebSocket(buildWsUrl());
  ws.binaryType = 'arraybuffer';

  ws.onopen = () => {
    console.log('[ws] connected');
    useGameStore.getState().setConnected(true);
  };

  ws.onmessage = onMessage;

  ws.onclose = () => {
    console.log('[ws] disconnected');
    useGameStore.getState().setConnected(false);
    ws = null;
    if (!reconnectTimer) {
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
      }, 3000);
    }
  };

  ws.onerror = (err) => {
    console.error('[ws] error:', err);
    ws?.close();
  };
}

export function disconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  ws?.close();
  ws = null;
}
