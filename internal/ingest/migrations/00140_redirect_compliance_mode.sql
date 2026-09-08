ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS redirect_compliance_mode TEXT NOT NULL DEFAULT 'strict';

COMMENT ON COLUMN campaigns.redirect_compliance_mode IS
    'strict: 302 redirect only; legacy_dmr: allow meta-refresh DMR when dmr_enabled or dmr=1 on /click.';

UPDATE campaigns
SET redirect_compliance_mode = 'legacy_dmr'
WHERE dmr_enabled = true
  AND redirect_compliance_mode = 'strict';
