INSERT INTO support.feedback (
    id,
    feedback_type,
    contact_email,
    message,
    deployment_id,
    binary_version,
    sku,
    attach_bundle,
    bundle_gzip,
    submitter_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);
