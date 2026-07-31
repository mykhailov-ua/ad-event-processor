import type { AppMetaLicense } from "../api";

type Props = {
  license?: AppMetaLicense;
};

export function LicenseBanner({ license }: Props) {
  if (!license || !license.banner_severity || license.banner_severity === "none") {
    return null;
  }

  const days =
    license.renew_days !== undefined ? `${license.renew_days} day(s)` : "soon";

  return (
    <div className={`license-banner severity-${license.banner_severity}`}>
      <strong>License {license.state}</strong>
      <span>
        Renew in {days}
        {license.valid_until ? ` (valid until ${license.valid_until})` : ""}
      </span>
    </div>
  );
}
