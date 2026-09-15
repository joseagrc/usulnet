-- 060_proxy_container_metadata.up.sql
--
-- Imported proxy definitions often have an upstream Docker name but omit the
-- optional container metadata used by the proxy-host list.  Backfill only an
-- exact host-scoped name match, so IP and external-DNS upstreams remain
-- intentionally unlinked.

UPDATE proxy_hosts AS proxy
SET
    container_id = container.id,
    container_name = container.name,
    updated_at = NOW()
FROM containers AS container
WHERE proxy.host_id = container.host_id
  AND proxy.upstream_host = container.name
  AND COALESCE(proxy.container_id, '') = '';
