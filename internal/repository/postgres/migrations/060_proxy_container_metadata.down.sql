-- 060_proxy_container_metadata.down.sql
--
-- This data-only migration intentionally has no destructive rollback.  A
-- container association can also be supplied manually, so clearing it during
-- a rollback could discard operator-provided metadata.

SELECT 1;
