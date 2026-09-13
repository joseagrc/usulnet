DO $$
BEGIN
    IF (SELECT relkind FROM pg_class WHERE oid = to_regclass('proxy_hosts')) = 'v' THEN
        DROP VIEW proxy_hosts;
        ALTER TABLE proxy_hosts_v47 DROP COLUMN IF EXISTS canonical_www_enabled;
        CREATE VIEW proxy_hosts AS SELECT * FROM proxy_hosts_v47;
    ELSE
        ALTER TABLE proxy_hosts DROP COLUMN IF EXISTS canonical_www_enabled;
    END IF;
END $$;
