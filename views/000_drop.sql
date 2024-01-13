DROP TRIGGER if exists check_statuses_change_to_event_queue on check_statuses;

CREATE OR REPLACE FUNCTION drop_push_queue_triggers () returns void as $$
DECLARE
  triggerName TEXT;
  tableName TEXT;
BEGIN
  FOR tableName, triggerName IN
    SELECT event_object_table, trigger_name
    FROM information_schema.triggers
    WHERE trigger_name like '%s_change_to_event_queue'
    GROUP BY event_object_table, trigger_name

  LOOP
    EXECUTE format('
      DROP TRIGGER IF EXISTS %s ON %s',
      triggerName, tableName
    );
  END LOOP;
END;
$$ LANGUAGE 'plpgsql' SECURITY DEFINER;
