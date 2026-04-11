import { useEffect, useState } from 'react';
import type { ShopItem, PlayerProfile } from '../types';
import { getShopItems, getProfile, buyItem, equipItem } from '../network/api';

export default function Shop({ onClose }: { onClose: () => void }) {
  const [items, setItems] = useState<ShopItem[]>([]);
  const [profile, setProfile] = useState<PlayerProfile | null>(null);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<string>('weapon');

  useEffect(() => {
    getShopItems().then(setItems).catch(() => setError('Failed to load shop'));
    getProfile().then(setProfile).catch(() => {});
  }, []);

  const handleBuy = async (itemId: string) => {
    setError('');
    try {
      const resp = await buyItem(itemId);
      if (profile) setProfile({ ...profile, coins: resp.coins, inventory: [...profile.inventory, itemId] });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Purchase failed');
    }
  };

  const handleEquip = async (itemId: string, slot: string) => {
    setError('');
    try {
      await equipItem(itemId, slot);
      if (profile) {
        const loadout = { ...profile.loadout, [slot]: itemId };
        setProfile({ ...profile, loadout });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Equip failed');
    }
  };

  const categories = ['weapon', 'shield', 'hull', 'ai_core'];
  const filtered = items.filter((i) => i.category === tab);

  return (
    <div style={{
      position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.85)',
      display: 'flex', justifyContent: 'center', alignItems: 'center',
      fontFamily: 'monospace', color: '#ddd', zIndex: 100,
    }}>
      <div style={{
        background: '#111', border: '1px solid #333', borderRadius: 8,
        padding: 24, width: 500, maxHeight: '80vh', overflowY: 'auto',
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
          <h2 style={{ margin: 0, color: '#0f0' }}>SHOP</h2>
          <div style={{ color: '#ff0' }}>
            {profile ? `${profile.coins} coins | Tier ${profile.ai_tier}` : '...'}
          </div>
          <button onClick={onClose} style={closeBtnStyle}>X</button>
        </div>

        <div style={{ display: 'flex', gap: 4, marginBottom: 16 }}>
          {categories.map((c) => (
            <button key={c} onClick={() => setTab(c)} style={{
              ...tabStyle, background: tab === c ? '#0a0' : '#222', color: tab === c ? '#000' : '#aaa',
            }}>
              {c.replace('_', ' ').toUpperCase()}
            </button>
          ))}
        </div>

        {error && <div style={{ color: '#f44', fontSize: 12, marginBottom: 8 }}>{error}</div>}

        {filtered.map((item) => {
          const owned = profile?.inventory.includes(item.id);
          const equipped = profile && (
            profile.loadout.hull === item.id ||
            profile.loadout.primary_weapon === item.id ||
            profile.loadout.secondary_weapon === item.id ||
            profile.loadout.shield === item.id
          );
          const slots: Record<string, string> = {
            weapon: 'primary_weapon', shield: 'shield', hull: 'hull',
          };

          return (
            <div key={item.id} style={{
              background: '#1a1a1a', borderRadius: 4, padding: '10px 12px',
              marginBottom: 8, border: equipped ? '1px solid #0f0' : '1px solid #333',
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <span style={{ color: '#fff', fontWeight: 'bold' }}>{item.name}</span>
                <span style={{ color: '#ff0', fontSize: 12 }}>
                  {item.price === 0 ? 'FREE' : `${item.price} coins`}
                  {item.tier_required > 1 && ` | T${item.tier_required}`}
                </span>
              </div>
              {item.description && (
                <div style={{ fontSize: 11, color: '#888', marginTop: 4 }}>{item.description}</div>
              )}
              <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                {!owned && (
                  <button onClick={() => handleBuy(item.id)} style={actionBtnStyle}>BUY</button>
                )}
                {owned && !equipped && slots[item.category] && (
                  <button onClick={() => handleEquip(item.id, slots[item.category])} style={actionBtnStyle}>
                    EQUIP
                  </button>
                )}
                {equipped && <span style={{ color: '#0f0', fontSize: 12 }}>EQUIPPED</span>}
                {owned && !equipped && !slots[item.category] && (
                  <span style={{ color: '#aaa', fontSize: 12 }}>OWNED</span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

const closeBtnStyle: React.CSSProperties = {
  background: 'none', border: '1px solid #555', color: '#aaa', cursor: 'pointer',
  fontFamily: 'monospace', width: 28, height: 28, borderRadius: 4,
};

const tabStyle: React.CSSProperties = {
  border: 'none', padding: '6px 12px', borderRadius: 4, cursor: 'pointer',
  fontFamily: 'monospace', fontSize: 11, fontWeight: 'bold',
};

const actionBtnStyle: React.CSSProperties = {
  background: '#0a0', color: '#000', border: 'none', padding: '4px 12px',
  borderRadius: 3, cursor: 'pointer', fontFamily: 'monospace', fontWeight: 'bold', fontSize: 11,
};
