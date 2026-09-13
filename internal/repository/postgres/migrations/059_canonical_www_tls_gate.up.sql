-- Gate root -> www redirects until the canonical endpoint has trusted TLS.
-- Some upgraded installations expose proxy_hosts as the compatibility view
-- backed by proxy_hosts_v47. Handle both that layout and a regular table.
DO $$
BEGIN
    IF (SELECT relkind FROM pg_class WHERE oid = to_regclass('proxy_hosts')) = 'v' THEN
        ALTER TABLE proxy_hosts_v47
            ADD COLUMN IF NOT EXISTS canonical_www_enabled BOOLEAN NOT NULL DEFAULT FALSE;
        CREATE OR REPLACE VIEW proxy_hosts AS SELECT * FROM proxy_hosts_v47;
    ELSE
        ALTER TABLE proxy_hosts
            ADD COLUMN IF NOT EXISTS canonical_www_enabled BOOLEAN NOT NULL DEFAULT FALSE;
    END IF;
END $$;

-- Preserve already-working canonical pairs during upgrade. Hosts previously
-- recovered to a single domain remain safely disabled.
UPDATE proxy_hosts AS host
SET canonical_www_enabled = TRUE
WHERE EXISTS (
    SELECT 1
    FROM unnest(host.domains) AS www_domain
    CROSS JOIN unnest(host.domains) AS root_domain
    WHERE lower(trim(www_domain)) = 'www.' || lower(trim(root_domain))
);
