-- Gate root -> www redirects until the canonical endpoint has trusted TLS.
ALTER TABLE proxy_hosts
    ADD COLUMN IF NOT EXISTS canonical_www_enabled BOOLEAN NOT NULL DEFAULT FALSE;

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
