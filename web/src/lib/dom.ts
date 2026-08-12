/**
 * DOM helpers for the admin SPA (TypeScript — §12 migration seed).
 */

type Child = Node | string | number | null | undefined | false | Child[];

type ElProps = Record<string, unknown> & {
  className?: string;
  dataset?: Record<string, string>;
  style?: Partial<CSSStyleDeclaration> | Record<string, string>;
  htmlFor?: string;
  textContent?: string;
  checked?: boolean;
  disabled?: boolean;
};

/**
 * Create an HTMLElement with optional props and children.
 */
export function el(tag: string, props?: ElProps | null, ...children: Child[]): HTMLElement {
  const node = document.createElement(tag);
  if (props) {
    for (const [key, value] of Object.entries(props)) {
      if (value === undefined || value === false) continue;
      if (key === 'className') node.className = String(value);
      else if (key === 'dataset' && value && typeof value === 'object') {
        Object.assign(node.dataset, value as Record<string, string>);
      } else if (key === 'style' && value && typeof value === 'object') {
        Object.assign(node.style, value);
      } else if (key.startsWith('on') && typeof value === 'function') {
        node.addEventListener(key.slice(2).toLowerCase(), value as EventListener);
      } else if (key === 'htmlFor') (node as HTMLLabelElement).htmlFor = String(value);
      else if (key === 'textContent') node.textContent = String(value);
      else if (key === 'checked' || key === 'disabled' || key === 'hidden') {
        (node as HTMLInputElement)[key] = Boolean(value);
      } else node.setAttribute(key, String(value));
    }
  }
  appendChildren(node, children);
  return node;
}

/**
 * Append mixed children to a parent node.
 */
export function appendChildren(parent: HTMLElement, children: Child[] | Child): void {
  const list = Array.isArray(children) ? children : [children];
  for (const child of list) {
    if (child == null || child === false) continue;
    if (Array.isArray(child)) appendChildren(parent, child);
    else if (typeof child === 'string' || typeof child === 'number') {
      parent.appendChild(document.createTextNode(String(child)));
    } else if (child instanceof Node) parent.appendChild(child);
  }
}

/**
 * Replace all children of a node.
 */
export function replaceChildren(node: HTMLElement, ...children: Child[]): void {
  node.replaceChildren();
  appendChildren(node, children);
}

/**
 * Span with monospace class.
 */
export function monoEl(text: string, props: ElProps = {}): HTMLElement {
  return el('span', { className: 'font-mono', ...props }, text);
}

/**
 * Return the string value of an input, textarea, or select event target.
 */
export function eventTargetValue(e: Event): string {
  const t = e.target;
  if (t instanceof HTMLInputElement || t instanceof HTMLTextAreaElement || t instanceof HTMLSelectElement) {
    return t.value;
  }
  return '';
}

/**
 * Return whether a checkbox/radio event target is checked.
 */
export function eventTargetChecked(e: Event): boolean {
  const t = e.target;
  return t instanceof HTMLInputElement ? t.checked : false;
}
