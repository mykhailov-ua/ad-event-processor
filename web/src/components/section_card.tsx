import type { ReactNode } from 'react';
import { Icon } from './icon.js';

export type SectionCardProps = {
  title?: string;
  desc?: string;
  icon?: string;
  urgent?: 'normal' | 'warning' | 'danger';
  className?: string;
  children?: ReactNode;
};

export function SectionCard({ title, desc, icon, urgent, className, children }: SectionCardProps) {
  const urgentClass = urgent ? ` settings-panel--urgent-${urgent}` : '';
  return (
    <section className={`settings-panel${urgentClass} ${className ?? ''}`.trim()}>
      {title || desc ? (
        <div className="settings-panel__header">
          {title ? (
            <div className="settings-panel__title-row">
              {icon ? <Icon name={icon} size={18} className="settings-panel__icon" /> : null}
              <h2 className="settings-panel__title">{title}</h2>
            </div>
          ) : null}
          {desc ? <p className="settings-panel__desc">{desc}</p> : null}
        </div>
      ) : null}
      <div className="settings-panel__body">{children}</div>
    </section>
  );
}
