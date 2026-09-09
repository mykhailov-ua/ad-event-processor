ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS timezone_attestation_mode TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN campaigns.timezone_attestation_mode IS
    'Safe-page timezone attestation policy: off, ip_country, campaign_target, strict. Empty uses ip_country for legacy campaigns.';
