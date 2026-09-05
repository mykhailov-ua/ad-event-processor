-- campaign_display_id_sort_key: numeric order for admin list ID column.
-- Must match internal/campaign/display_id.go CampaignDisplayIDSortKey (base-31 over uuid bytes).
CREATE OR REPLACE FUNCTION campaign_display_id_sort_key(campaign_id uuid)
RETURNS bigint
LANGUAGE plpgsql
IMMUTABLE
PARALLEL SAFE
STRICT
AS $$
DECLARE
  payload bytea := uuid_send(campaign_id);
  hash numeric := 0;
  i int;
BEGIN
  FOR i IN 0..15 LOOP
    hash := mod(hash * 31 + get_byte(payload, i), 18446744073709551616::numeric);
  END LOOP;
  RETURN (10000000 + mod(hash, 90000000))::bigint;
END;
$$;
