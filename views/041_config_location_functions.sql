---
CREATE INDEX IF NOT EXISTS config_locations_location_pattern_idx 
ON config_locations (location text_pattern_ops) INCLUDE (id);

-- Function to get children by location based on config external IDs
CREATE OR REPLACE FUNCTION get_children_id_by_location(
    config_id_param UUID, 
    alias_prefix TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID
) AS $$
DECLARE
    ext_id TEXT;
    filtered_external_ids TEXT[];
BEGIN
    -- Get the config item's external_id array
    SELECT external_id INTO filtered_external_ids 
    FROM config_items 
    WHERE config_items.id = config_id_param;
    
    -- If no external_id found, return empty result
    IF filtered_external_ids IS NULL THEN
        RETURN;
    END IF;
    
    -- Filter external_ids by prefix if provided
    IF alias_prefix IS NOT NULL AND alias_prefix <> '' THEN
        filtered_external_ids := ARRAY(
            SELECT ext_id_val
            FROM unnest(filtered_external_ids) AS ext_id_val
            WHERE ext_id_val LIKE alias_prefix || '%'
        );
    END IF;
    
    -- For each filtered external_id, find configs that have that prefix in their location
    FOREACH ext_id IN ARRAY filtered_external_ids
    LOOP
        RETURN QUERY
        SELECT cl.id
        FROM config_locations cl
        WHERE cl.location LIKE ext_id || '%' AND cl.id != config_id_param;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 
CREATE OR REPLACE FUNCTION get_children_by_location(
    config_id UUID, 
    alias_prefix TEXT DEFAULT NULL,
    include_deleted BOOLEAN DEFAULT FALSE
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    type TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT config_items.id, config_items.name, config_items.type FROM config_items 
    WHERE config_items.id IN (SELECT children_ids.id FROM get_children_id_by_location(config_id, alias_prefix) AS children_ids)
        AND (include_deleted OR deleted_at IS NULL);
END;
$$ LANGUAGE plpgsql;

-- Function to get parent IDs by location based on config external IDs
CREATE OR REPLACE FUNCTION get_parent_ids_by_location(
    config_id UUID, 
    alias_prefix TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID
) AS $$
DECLARE
    location_row RECORD;
BEGIN
    -- For each location of the config item, find all configs that have this location in their external_id
    FOR location_row IN 
        SELECT cl.location
        FROM config_locations cl
        WHERE cl.id = config_id
        AND (alias_prefix IS NULL OR alias_prefix = '' OR cl.location LIKE alias_prefix || '%')
    LOOP
        RETURN QUERY
        SELECT ci.id
        FROM config_items ci
        WHERE location_row.location = ANY(ci.external_id);
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 
CREATE OR REPLACE FUNCTION get_parents_by_location(
    config_id UUID, 
    alias_prefix TEXT DEFAULT NULL,
    include_deleted BOOLEAN DEFAULT FALSE
)
RETURNS TABLE (
    id UUID,
    name TEXT,
    type TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT config_items.id, config_items.name, config_items.type FROM config_items 
    WHERE config_items.id IN (SELECT parent_ids.id FROM get_parent_ids_by_location(config_id, alias_prefix) AS parent_ids)
        AND (include_deleted OR deleted_at IS NULL)
    ORDER BY
    CASE config_items.type
            -- Level 1: Top-level infrastructure
            WHEN 'AWS::::Account' THEN 1
            WHEN 'Kubernetes::Cluster' THEN 1
            
            -- Level 2: Regional/Zone resources (AWS) and Cluster-wide resources (K8s)
            WHEN 'AWS::Region' THEN 2
            WHEN 'Kubernetes::Node' THEN 3
            WHEN 'Kubernetes::StorageClass' THEN 3
            WHEN 'Kubernetes::IngressClass' THEN 3
            WHEN 'Kubernetes::ClusterRole' THEN 3
            WHEN 'Kubernetes::Role' THEN 3

            -- Level 3: Sub-regional resources (AWS) and Cluster-wide resources (K8s)
            WHEN 'AWS::AvailabilityZone' THEN 3
            WHEN 'AWS::AvailabilityZoneID' THEN 3
            WHEN 'AWS::IAM::Role' THEN 3
            WHEN 'AWS::IAM::User' THEN 3
            WHEN 'Kubernetes::ClusterRoleBinding' THEN 3
            WHEN 'Kubernetes::Namespace' THEN 3
            WHEN 'Kubernetes::RoleBinding' THEN 3
            WHEN 'Kubernetes::PersistentVolume' THEN 3
            
            -- Level 4:
            WHEN 'AWS::EC2::VPC' THEN 4
            WHEN 'AWS::S3::Bucket' THEN 4
            WHEN 'Kubernetes::HelmRelease' THEN 4
            WHEN 'Kubernetes::Kustomization' THEN 4
            
            -- Level 5:
            WHEN 'AWS::EC2::Subnet' THEN 5
            WHEN 'Kubernetes::Deployment' THEN 5
            WHEN 'Kubernetes::StatefulSet' THEN 5
            WHEN 'Kubernetes::DaemonSet' THEN 5
            WHEN 'Kubernetes::Job' THEN 5
            WHEN 'Kubernetes::CronJob' THEN 5
            WHEN 'Kubernetes::Ingress' THEN 5

            -- Level 6:
            WHEN 'Kubernetes::Service' THEN 6
            WHEN 'Kubernetes::Endpoints' THEN 6
            WHEN 'Kubernetes::ReplicaSet' THEN 6
            WHEN 'Kubernetes::PersistentVolumeClaim' THEN 6
        ELSE 99 -- Fallback for unknown types
    END,
    config_items.name; -- Secondary sort by name for items at the same level;
END;
$$ LANGUAGE plpgsql;