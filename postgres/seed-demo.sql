-- Synthetic local demo data, not real player statistics.
-- Re-running this seed preserves existing rows, including edited demo players.
INSERT INTO batting (
    name, team, year, at_bat, hit, double, triple, home_run,
    ball_on_base, hit_by_pitch
)
VALUES
    ('Demo Batter 01', 'Demo Comets (Synthetic)', 2024, 100, 32, 6, 1, 4, 12, 2),
    ('Demo Batter 02', 'Demo Comets (Synthetic)', 2024, 100, 28, 5, 2, 3, 10, 1),
    ('Demo Batter 03', 'Demo Comets (Synthetic)', 2024, 100, 35, 8, 0, 7, 15, 2),
    ('Demo Batter 04', 'Demo Comets (Synthetic)', 2024, 100, 30, 6, 1, 5, 11, 1),
    ('Demo Batter 05', 'Demo Comets (Synthetic)', 2024, 100, 26, 4, 2, 2, 14, 3),
    ('Demo Batter 06', 'Demo Comets (Synthetic)', 2024, 100, 33, 7, 1, 6, 9, 2),
    ('Demo Batter 07', 'Demo Comets (Synthetic)', 2024, 100, 29, 5, 0, 4, 13, 1),
    ('Demo Batter 08', 'Demo Comets (Synthetic)', 2024, 100, 31, 6, 2, 3, 10, 2),
    ('Demo Batter 09', 'Demo Comets (Synthetic)', 2024, 100, 27, 4, 1, 5, 12, 1),
    ('Demo Batter 10', 'Demo Comets (Synthetic)', 2024, 100, 34, 8, 1, 4, 8, 2),
    ('Demo Batter 11', 'Demo Comets (Synthetic)', 2024, 100, 25, 5, 0, 2, 16, 1),
    ('Demo Batter 12', 'Demo Comets (Synthetic)', 2024, 100, 30, 7, 1, 3, 11, 3)
ON CONFLICT (name, team, year) DO NOTHING;
