-- 002_seed_items.sql

INSERT INTO items (id, name, category, stats, price, tier_required, description) VALUES
-- Lasers (hitscan)
('wpn_laser_1', 'Basic Laser',     'weapon', '{"type":"laser","damage":8,"cooldown":0.5,"range":200,"speed":0,"spread":2}',     0, 1, 'Standard issue laser. Reliable but weak.'),
('wpn_laser_2', 'Focused Laser',   'weapon', '{"type":"laser","damage":12,"cooldown":0.4,"range":250,"speed":0,"spread":1}',  500, 2, 'Tighter beam, more damage.'),
('wpn_laser_3', 'Phase Laser',     'weapon', '{"type":"laser","damage":18,"cooldown":0.35,"range":300,"speed":0,"spread":0.5}', 1500, 3, 'Military-grade precision laser.'),
-- Plasma (projectile)
('wpn_plasma_1', 'Plasma Blaster', 'weapon', '{"type":"plasma","damage":25,"cooldown":1.5,"range":300,"speed":250,"spread":0}', 300, 1, 'Slow but packs a punch.'),
('wpn_plasma_2', 'Heavy Plasma',   'weapon', '{"type":"plasma","damage":40,"cooldown":1.8,"range":350,"speed":300,"spread":0}', 1200, 3, 'Heavier payload, longer reach.'),
-- Railgun
('wpn_railgun_1', 'Railgun',       'weapon', '{"type":"laser","damage":60,"cooldown":4.0,"range":400,"speed":0,"spread":0}',  2500, 4, 'One shot. Make it count.')
ON CONFLICT (id) DO NOTHING;

INSERT INTO items (id, name, category, stats, price, tier_required, description) VALUES
-- Shields
('shld_basic',    'Basic Shield',     'shield', '{"max_shield":50,"regen":5,"delay":3}',    0, 1, 'Minimal protection.'),
('shld_regen',    'Regen Shield',     'shield', '{"max_shield":50,"regen":10,"delay":2}',  400, 2, 'Faster recharge cycle.'),
('shld_heavy',    'Heavy Shield',     'shield', '{"max_shield":100,"regen":5,"delay":3}',  800, 2, 'Double capacity, same regen.'),
('shld_advanced', 'Advanced Shield',  'shield', '{"max_shield":100,"regen":12,"delay":1.5}', 2000, 4, 'Superior protection all around.')
ON CONFLICT (id) DO NOTHING;

INSERT INTO items (id, name, category, stats, price, tier_required, description) VALUES
-- Hulls
('hull_basic',   'Scout Hull',   'hull', '{"max_health":100,"max_speed":50,"thrust":40,"collision_radius":2}',     0, 1, 'Light and nimble.'),
('hull_medium',  'Cruiser Hull', 'hull', '{"max_health":150,"max_speed":35,"thrust":30,"collision_radius":2.5}',  600, 2, 'Tougher but slower.'),
('hull_heavy',   'Titan Hull',   'hull', '{"max_health":250,"max_speed":25,"thrust":20,"collision_radius":3.5}', 1500, 3, 'A fortress in space.'),
('hull_stealth', 'Phantom Hull', 'hull', '{"max_health":80,"max_speed":60,"thrust":50,"collision_radius":1.5}',  1800, 4, 'Fragile but incredibly fast.')
ON CONFLICT (id) DO NOTHING;

INSERT INTO items (id, name, category, stats, price, tier_required, description) VALUES
-- AI Cores
('ai_core_1', 'Basic Processor',    'ai_core', '{"ai_tier":1}',     0, 1, 'Limited command vocabulary.'),
('ai_core_2', 'Enhanced Processor', 'ai_core', '{"ai_tier":2}',   800, 1, 'Unlocks conditional behaviors.'),
('ai_core_3', 'Tactical Processor', 'ai_core', '{"ai_tier":3}',  2000, 1, 'Battlefield awareness enabled.'),
('ai_core_4', 'Strategic Mind',     'ai_core', '{"ai_tier":4}',  5000, 1, 'Complex multi-condition chains.'),
('ai_core_5', 'Quantum Brain',      'ai_core', '{"ai_tier":5}', 12000, 1, 'Full tactical AI. Maximum capability.')
ON CONFLICT (id) DO NOTHING;
