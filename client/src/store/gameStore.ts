import { create } from 'zustand';
import type { EntityState, StateUpdatePayload } from '../types';

interface GameState {
  connected: boolean;
  tick: number;
  entities: Map<number, EntityState>;
  setConnected: (connected: boolean) => void;
  updateEntities: (payload: StateUpdatePayload) => void;
}

export const useGameStore = create<GameState>((set) => ({
  connected: false,
  tick: 0,
  entities: new Map(),

  setConnected: (connected) => set({ connected }),

  updateEntities: (payload) =>
    set(() => {
      const entities = new Map<number, EntityState>();
      for (const e of payload.entities) {
        entities.set(e.id, e);
      }
      return { tick: payload.tick, entities };
    }),
}));
