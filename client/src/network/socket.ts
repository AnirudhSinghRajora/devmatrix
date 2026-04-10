import { decode } from '@msgpack/msgpack';
import type { Envelope, StateUpdatePayload } from '../types';
import { MsgType } from '../types';
import { useGameStore } from '../store/gameStore';

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

const WS_URL = `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/ws`;

function onMessage(event: MessageEvent) {
  const data = event.data;
  if (!(data instanceof ArrayBuffer)) return;

  const envelope = decode(new Uint8Array(data)) as Envelope;

  switch (envelope.t) {
    case MsgType.StateUpdate: {
      const payload = decode(envelope.p) as StateUpdatePayload;
      useGameStore.getState().updateEntities(payload);
      break;
    }
    case MsgType.Error: {
      console.error('[ws] server error:', decode(envelope.p));
      break;
    }
  }
}

export function connect() {
  if (ws && ws.readyState === WebSocket.OPEN) return;

  ws = new WebSocket(WS_URL);
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
    // Auto-reconnect after 2s
    if (!reconnectTimer) {
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
      }, 2000);
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
