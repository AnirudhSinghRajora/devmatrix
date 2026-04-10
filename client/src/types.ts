// Wire protocol types matching server/internal/network/messages.go

export const MsgType = {
  StateUpdate: 1,
  Prompt: 2,
  Error: 3,
} as const;

export interface Envelope {
  t: number;
  p: Uint8Array;
}

export interface EntityState {
  id: number;
  pos: [number, number, number];
  rot: [number, number, number, number]; // quaternion xyzw
}

export interface StateUpdatePayload {
  tick: number;
  entities: EntityState[];
}
