-- name: SearchCommandPaletteCampaigns :many
SELECT id, name, status, updated_at
FROM campaigns
WHERE customer_id = sqlc.arg(customer_id)
  AND deleted_at IS NULL
  AND name ILIKE '%' || sqlc.arg(query) || '%'
ORDER BY
  (lower(name) LIKE lower(sqlc.arg(query)) || '%') DESC,
  updated_at DESC,
  name ASC
LIMIT sqlc.arg(result_limit);

-- name: SearchCommandPaletteFlows :many
SELECT f.id, f.name, f.created_at
FROM flows f
WHERE f.name ILIKE '%' || sqlc.arg(query) || '%'
  AND EXISTS (
    SELECT 1
    FROM campaigns c
    WHERE c.customer_id = sqlc.arg(customer_id)
      AND c.flow_id = f.id
      AND c.deleted_at IS NULL
  )
ORDER BY
  (lower(f.name) LIKE lower(sqlc.arg(query)) || '%') DESC,
  f.created_at DESC,
  f.name ASC
LIMIT sqlc.arg(result_limit);

-- name: SearchCommandPaletteLanders :many
SELECT id, name, created_at
FROM landers
WHERE name ILIKE '%' || sqlc.arg(query) || '%'
ORDER BY
  (lower(name) LIKE lower(sqlc.arg(query)) || '%') DESC,
  created_at DESC,
  name ASC
LIMIT sqlc.arg(result_limit);

-- name: SearchCommandPaletteOffers :many
SELECT id, name, created_at
FROM offers
WHERE name ILIKE '%' || sqlc.arg(query) || '%'
ORDER BY
  (lower(name) LIKE lower(sqlc.arg(query)) || '%') DESC,
  created_at DESC,
  name ASC
LIMIT sqlc.arg(result_limit);

-- name: ListCommandPaletteRecents :many
SELECT item_id, kind, label, href, meta, "group", accessed_at
FROM command_palette_recents
WHERE customer_id = sqlc.arg(customer_id)
  AND user_id = sqlc.arg(user_id)
ORDER BY accessed_at DESC
LIMIT 20;

-- name: UpsertCommandPaletteRecent :exec
INSERT INTO command_palette_recents (
    customer_id,
    user_id,
    item_id,
    kind,
    label,
    href,
    meta,
    "group",
    accessed_at
) VALUES (
    sqlc.arg(customer_id),
    sqlc.arg(user_id),
    sqlc.arg(item_id),
    sqlc.arg(kind),
    sqlc.arg(label),
    sqlc.arg(href),
    sqlc.arg(meta),
    sqlc.arg(group_name),
    now()
)
ON CONFLICT (customer_id, user_id, item_id, kind)
DO UPDATE SET
    label = EXCLUDED.label,
    href = EXCLUDED.href,
    meta = EXCLUDED.meta,
    "group" = EXCLUDED."group",
    accessed_at = now();

-- name: PruneCommandPaletteRecents :exec
DELETE FROM command_palette_recents AS outer_recents
WHERE outer_recents.id IN (
    SELECT inner_recents.id
    FROM command_palette_recents AS inner_recents
    WHERE inner_recents.customer_id = sqlc.arg(customer_id)
      AND inner_recents.user_id = sqlc.arg(user_id)
    ORDER BY inner_recents.accessed_at DESC
    OFFSET sqlc.arg(keep_max)
);
