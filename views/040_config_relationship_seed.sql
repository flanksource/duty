-- DO NOT MODIFY THIS FILE
-- THIS SCRIPT IS SUPPOSED TO RUN ONLY ONCE

DO $$
DECLARE
    v_config_item_id UUID;
    v_count INTEGER := 0;
    v_error_count INTEGER := 0;
BEGIN
    -- Loop through all non-deleted config_items
    FOR v_config_item_id IN
        SELECT id
        FROM config_items
        WHERE deleted_at IS NULL
        ORDER BY created_at
    LOOP
        BEGIN
            -- Call the function for each config_item
            PERFORM create_alias_config_relationships_for_config_item(v_config_item_id);
            v_count := v_count + 1;

        EXCEPTION
            WHEN OTHERS THEN
                -- Log error but continue processing
                v_error_count := v_error_count + 1;
                RAISE NOTICE 'Error processing config_item %: %', v_config_item_id, SQLERRM;
        END;
    END LOOP;

    RAISE NOTICE 'Completed creating alias config_relationships. Processed % config items with % errors', v_count, v_error_count;
END $$;
