/**
 * @param {string} tag
 * @param {Record<string, unknown>|null} [props]
 * @param {...(Node|string|number|null|undefined|false)} children
 * @returns {HTMLElement}
 */
export function el(tag, props, ...children) {
  const node = document.createElement(tag);
  if (props) {
    for (const [key, value] of Object.entries(props)) {
      if (value === undefined || value === false) continue;
      if (key === 'className') node.className = String(value);
      else if (key === 'dataset') Object.assign(node.dataset, value);
      else if (key === 'style' && typeof value === 'object') Object.assign(node.style, value);
      else if (key.startsWith('on') && typeof value === 'function') {
        node.addEventListener(key.slice(2).toLowerCase(), value);
      } else if (key === 'htmlFor') node.htmlFor = String(value);
      else if (key === 'textContent') node.textContent = String(value);
      else if (key === 'checked' || key === 'disabled') node[key] = Boolean(value);
      else node.setAttribute(key, String(value));
    }
  }
  appendChildren(node, children);
  return node;
}

/**
 * @param {HTMLElement} parent
 * @param {...(Node|string|number|null|undefined|false|Array<unknown>)} children
 */
export function appendChildren(parent, children) {
  for (const child of children) {
    if (child == null || child === false) continue;
    if (Array.isArray(child)) appendChildren(parent, child);
    else if (typeof child === 'string' || typeof child === 'number') {
      parent.appendChild(document.createTextNode(String(child)));
    } else if (child instanceof Node) parent.appendChild(child);
  }
}

/**
 * @param {HTMLElement} node
 * @param {...(Node|string|number|null|undefined|false)} children
 */
export function replaceChildren(node, ...children) {
  node.replaceChildren();
  appendChildren(node, children);
}
