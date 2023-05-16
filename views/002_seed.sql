DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM people WHERE name = 'System') THEN
        INSERT INTO people (name) VALUES ('System');
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM agents WHERE name = 'local') THEN
        INSERT INTO agents (id, name) VALUES ('00000000-0000-0000-0000-000000000000', 'local');
    END IF;
END $$;

DO $$
BEGIN
   IF NOT EXISTS (SELECT FROM severities ) THEN
        INSERT INTO severities (id, name, icon, aliases)
        VALUES
            (1, 'Critical', 'error',ARRAY ['P1']),
            (2, 'Blocker', 'error', ARRAY['P2']),
            (3, 'High', 'warning',ARRAY ['P3']),
            (4, 'Medium', 'info',ARRAY ['P4']),
            (5, 'Low', 'info', ARRAY['P4']);
   END IF;
END $$;
