-- Report schedule M3: destination, Sheets owner, last run status.
ALTER TABLE report_schedules DROP CONSTRAINT IF EXISTS report_schedules_format_check;
ALTER TABLE report_schedules ADD CONSTRAINT report_schedules_format_check
    CHECK (format IN ('csv', 'json', 'xlsx', 'zip'));

ALTER TABLE report_schedules ADD COLUMN IF NOT EXISTS destination TEXT NOT NULL DEFAULT 'download';
ALTER TABLE report_schedules DROP CONSTRAINT IF EXISTS report_schedules_destination_check;
ALTER TABLE report_schedules ADD CONSTRAINT report_schedules_destination_check
    CHECK (destination IN ('download', 'google_sheet'));

ALTER TABLE report_schedules ADD COLUMN IF NOT EXISTS owner_user_id UUID;
ALTER TABLE report_schedules ADD COLUMN IF NOT EXISTS last_run_status TEXT;
ALTER TABLE report_schedules ADD COLUMN IF NOT EXISTS last_run_error_public TEXT;
