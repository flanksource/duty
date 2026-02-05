-- A basic connection view free from any sensitive data.
DROP VIEW IF EXISTS connections_list;
CREATE OR REPLACE VIEW connections_list AS
  SELECT
    id,
    name,
    namespace,
    type,
    CASE
      WHEN (string_to_array(url, '://'))[1] IN ('bark', 'discord', 'smtp', 'gotify', 'googlechat', 'ifttt', 'join', 'mattermost', 'matrix', 'ntfy', 'opsgenie', 'pushbullet', 'pushover', 'rocketchat', 'slack', 'teams', 'telegram', 'zulip') THEN 'notification'
      ELSE ''
    END AS category,
    created_by,
    created_at,
    updated_at
  FROM
    connections
  WHERE
    deleted_at IS NULL
  ORDER BY
    created_at;

-- 
CREATE OR REPLACE FUNCTION mask_sensitive(field_value TEXT)
RETURNS TEXT AS $$
BEGIN
  RETURN CASE
    WHEN field_value LIKE 'secret://%' OR 
         field_value LIKE 'configmap://%' OR 
         field_value LIKE 'helm://%' OR 
         field_value LIKE 'serviceaccount://%' OR 
         field_value = '' THEN field_value
    ELSE '***'
  END;
END;
$$ LANGUAGE plpgsql;
-- 

-- A connection view that masks sensitive fields.
DROP VIEW IF EXISTS connection_details;
CREATE OR REPLACE VIEW connection_details AS
  SELECT
    id, name, namespace, type, source, properties, insecure_tls, created_by, created_at, updated_at,
    CASE
      WHEN (string_to_array(url, '://'))[1] IN ('bark', 'discord', 'smtp', 'gotify', 'googlechat', 'ifttt', 'join', 'mattermost', 'matrix', 'ntfy', 'opsgenie', 'pushbullet', 'pushover', 'rocketchat', 'slack', 'teams', 'telegram', 'zulip') THEN 'notification'
      ELSE ''
    END AS category,
    mask_sensitive(username) AS username,
    mask_sensitive(PASSWORD) AS PASSWORD,
    mask_sensitive(certificate) AS certificate
  FROM connections
  WHERE
    deleted_at IS NULL
  ORDER BY
    created_at;

--
CREATE OR REPLACE FUNCTION connection_before_update()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.username = '***' THEN
    NEW.username = OLD.username;
  END IF;

  IF NEW.password = '***' THEN
    NEW.password = OLD.password;
  END IF;

  IF NEW.certificate = '***' THEN
    NEW.certificate = OLD.certificate;
  END IF;

  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER connection_before_update
BEFORE UPDATE ON connections
FOR EACH ROW EXECUTE PROCEDURE connection_before_update();
