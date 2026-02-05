DO $$
BEGIN
  IF EXISTS (
      SELECT 1
      FROM pg_tables
      WHERE schemaname = 'public'
      AND tablename = 'playbook_run_actions'
  ) THEN
      -- Remove the check constraint.
      -- This has also been remove from schema (HCL), but the migration only runs after this.
      ALTER TABLE playbook_run_actions 
      DROP CONSTRAINT IF EXISTS playbook_action_not_null_run_id;

      -- Remove agent_id on playbook actions in agents to satisfy the foreign key.
      UPDATE playbook_run_actions SET agent_id = NULL WHERE agent_id NOT IN (
        SELECT id FROM agents
      );
  END IF;
END $$;