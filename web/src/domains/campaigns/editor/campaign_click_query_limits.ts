/**
 * Mirrors internal/campaign/click_query_helpers.go (jsonb column stores map[string]string).
 * Postgres jsonb has no stricter app limit; these bounds match server ValidateClickQueryParams.
 */
export const CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS = 40;
export const CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN = 512;

const CLICK_QUERY_LONGEST_KEY_LEN = 14; // ad_campaign_id

/** Worst-case pretty-printed JSON: 40 keys, 512-char values, 2-space indent. */
export const CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS =
  2 +
  CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS *
    (4 + CLICK_QUERY_LONGEST_KEY_LEN + 4 + CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN + 2);

/** Cap manual vertical resize (~20 lines at text-sm). */
export const CAMPAIGN_CLICK_QUERY_PARAMS_TEXTAREA_MAX_HEIGHT_CLASS = 'max-h-80';

export const CAMPAIGN_EDITOR_CODE_EXTRALIGHT_CLASS = 'ui-editor-code-extralight';

/** @deprecated Use CAMPAIGN_EDITOR_CODE_EXTRALIGHT_CLASS. */
export const CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS = CAMPAIGN_EDITOR_CODE_EXTRALIGHT_CLASS;

/** Inter wght 200 for campaign editor code/UUID/URL fields. */
export const CAMPAIGN_CLICK_QUERY_PARAMS_TEXTAREA_MONO_CLASS =
  CAMPAIGN_EDITOR_CODE_EXTRALIGHT_CLASS;

export const CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT = `Whole JSON up to ${CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS.toLocaleString()} characters. Up to ${CAMPAIGN_CLICK_QUERY_PARAM_MAX_KEYS} keys; each value max ${CAMPAIGN_CLICK_QUERY_PARAM_MAX_VALUE_LEN} characters.`;
