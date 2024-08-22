DO $$
BEGIN
    -- Check if the column "count" exists in the "config_changes" table
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'config_changes'
        AND column_name = 'count'
    ) THEN
        -- Update existing NULL values in the "count" column
        UPDATE config_changes
        SET count = 1
        WHERE count IS NULL;
    END IF;
END $$;

DO $$
BEGIN
    -- Check if the column "first_observed" exists in the "config_changes" table
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'config_changes'
        AND column_name = 'first_observed'
    ) THEN
        -- Update existing NULL values in the "first_observed" column
        UPDATE config_changes
        SET first_observed = created_at
        WHERE first_observed IS NULL;
    END IF;
END $$;

DO $$
BEGIN
    -- Check if the column "category" exists in the "playbooks" table
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'playbooks'
        AND column_name = 'category'
    ) THEN
        -- Update existing NULL values in the "category" column
        UPDATE playbooks
        SET category = ''
        WHERE category IS NULL;
    END IF;
END $$;
