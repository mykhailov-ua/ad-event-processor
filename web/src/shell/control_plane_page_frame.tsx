import type { ReactNode } from 'react';

export type ControlPlanePageFrameProps = {
  title: ReactNode;
  description?: ReactNode;
  toolbar?: ReactNode;
  filters?: ReactNode;
  aside?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
};

export function ControlPlanePageFrame({
  title,
  description,
  toolbar,
  filters,
  aside,
  footer,
  children,
}: ControlPlanePageFrameProps) {
  return (
    <article>
      <header>
        <h1>{title}</h1>
        {description ? <p>{description}</p> : null}
        {toolbar ? <div role="toolbar">{toolbar}</div> : null}
      </header>

      {filters ? <section aria-label="Filters">{filters}</section> : null}

      <div>
        <main>{children}</main>
        {aside ? <aside>{aside}</aside> : null}
      </div>

      {footer ? <footer>{footer}</footer> : null}
    </article>
  );
}

export type ControlPlaneDirectoryFrameProps = {
  title: ReactNode;
  toolbar?: ReactNode;
  filters?: ReactNode;
  table: ReactNode;
  footer?: ReactNode;
  error?: ReactNode;
  status?: ReactNode;
};

export function ControlPlaneDirectoryFrame({
  title,
  toolbar,
  filters,
  table,
  footer,
  error,
  status,
}: ControlPlaneDirectoryFrameProps) {
  return (
    <ControlPlanePageFrame filters={filters} footer={footer} title={title} toolbar={toolbar}>
      {status}
      {error}
      <section aria-label="Results">{table}</section>
    </ControlPlanePageFrame>
  );
}

export type ControlPlaneEditorFrameProps = {
  title: ReactNode;
  description?: ReactNode;
  toolbar?: ReactNode;
  tabs?: ReactNode;
  primary: ReactNode;
  secondary?: ReactNode;
  aside?: ReactNode;
};

export function ControlPlaneEditorFrame({
  title,
  description,
  toolbar,
  tabs,
  primary,
  secondary,
  aside,
}: ControlPlaneEditorFrameProps) {
  return (
    <article>
      <header>
        <h1>{title}</h1>
        {description ? <p>{description}</p> : null}
        {toolbar ? <div role="toolbar">{toolbar}</div> : null}
        {tabs ? <nav aria-label="Sections">{tabs}</nav> : null}
      </header>

      <div>
        <main>
          <section aria-label="Primary">{primary}</section>
          {secondary ? <section aria-label="Secondary">{secondary}</section> : null}
        </main>
        {aside ? <aside>{aside}</aside> : null}
      </div>
    </article>
  );
}
