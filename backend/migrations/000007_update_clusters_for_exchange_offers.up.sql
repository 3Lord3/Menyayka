-- The composite primary key already forbids duplicating one
-- "cluster — offer" pair. A separate constraint guarantees that an offer
-- belongs to at most one cluster.
--
-- On existing databases the constraint may already have been added manually
-- before this migration. In that case keep the existing uniqueness.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'cluster_members'::regclass
          AND conname = 'cluster_members_request_id_key'
    )
    AND to_regclass('cluster_members_request_id_key') IS NULL THEN
        ALTER TABLE cluster_members
            ADD CONSTRAINT cluster_members_request_id_key UNIQUE (request_id);
    END IF;
END $$;
