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

export function el(tag: string, props?: ElProps | null, ...children: Child[]): HTMLElement {
  const node = document.createElement(tag);
  let defaultValue: string | undefined;
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
      else if (key === 'defaultValue') defaultValue = String(value);
      else if (key === 'checked' || key === 'disabled' || key === 'hidden') {
        (node as HTMLInputElement)[key] = Boolean(value);
      } else node.setAttribute(key, String(value));
    }
  }
  appendChildren(node, children);
  if (defaultValue !== undefined) {
    if (
      node instanceof HTMLInputElement ||
      node instanceof HTMLTextAreaElement ||
      node instanceof HTMLSelectElement
    ) {
      node.value = defaultValue;
    }
  }
  return node;
}

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

export function replaceChildren(node: HTMLElement, ...children: Child[]): void {
  node.replaceChildren();
  appendChildren(node, children);
}

export function monoEl(text: string, props: ElProps = {}): HTMLElement {
  return el('span', { className: 'font-mono', ...props }, text);
}

export function eventTargetValue(e: Event): string {
  const t = e.target;
  if (
    t instanceof HTMLInputElement ||
    t instanceof HTMLTextAreaElement ||
    t instanceof HTMLSelectElement
  ) {
    return t.value;
  }
  return '';
}

export function eventTargetChecked(e: Event): boolean {
  const t = e.target;
  return t instanceof HTMLInputElement ? t.checked : false;
}
