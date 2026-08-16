import { Icon } from './icon.js';

export type BreadcrumbItem = {
  label: string;
  href?: string;
};

export type BreadcrumbsProps = {
  items: BreadcrumbItem[];
};

/**
 * Breadcrumb trail for page headers.
 */
export function Breadcrumbs({ items }: BreadcrumbsProps) {
  if (items.length === 0) return null;
  return (
    <nav className="page-header__breadcrumbs" aria-label="Breadcrumb">
      {items.map((item, index) => (
        <span key={`${item.label}-${index}`} className="page-header__crumb-group">
          {index > 0 ? (
            <Icon name="chevron-right" size={12} className="page-header__crumb-sep" />
          ) : null}
          {item.href ? (
            <a href={item.href} className="page-header__crumb-link">{item.label}</a>
          ) : (
            <span className="page-header__crumb-current">{item.label}</span>
          )}
        </span>
      ))}
    </nav>
  );
}
