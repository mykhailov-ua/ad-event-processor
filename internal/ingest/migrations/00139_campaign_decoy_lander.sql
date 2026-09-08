ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS decoy_lander_id UUID NULL;

COMMENT ON COLUMN campaigns.decoy_lander_id IS
    'Optional hosted lander UUID for sandbox decoy shell (/lp/{id}/). When null, derive from safe_page_url or use static default.';
