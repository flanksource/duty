-- Cleanup old trigger function that used to add 'push_queue.delete' events.
-- This will remove the trigger as well.
DROP FUNCTION IF EXISTS push_deletes_to_event_queue CASCADE;

-- Cleanup old trigger that used to add 'push_queue.create' events.
-- This will remove the trigger as well.
DROP FUNCTION IF EXISTS push_changes_to_event_queue CASCADE;