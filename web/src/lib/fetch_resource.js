import { api } from '../helpers/api_client.js';
import { to } from './to.js';

/**
 * @typedef {Object} ResourceState
 * @property {unknown|null} data
 * @property {boolean} loading
 * @property {unknown|null} error
 */

/**
 * @param {() => string|null|undefined} getUrl
 * @param {{ skip?: () => boolean, onUpdate: (state: ResourceState) => void }} opts
 */
export function createResource(getUrl, opts) {
  const skip = opts.skip ?? (() => false);
  let abort = null;
  let gen = 0;
  let destroyed = false;

  async function load() {
    if (destroyed) return;
    const url = getUrl();
    if (!url || skip()) {
      opts.onUpdate({ data: null, loading: false, error: null });
      return;
    }
    if (abort) abort.abort();
    const ctrl = new AbortController();
    abort = ctrl;
    const id = ++gen;
    opts.onUpdate({ data: null, loading: true, error: null });
    const [result, err] = await to(api(url, { signal: ctrl.signal }));
    if (destroyed || id !== gen) return;
    if (err) {
      if (err.name === 'AbortError') return;
      opts.onUpdate({ data: null, loading: false, error: err });
      return;
    }
    opts.onUpdate({ data: result.data, loading: false, error: null });
  }

  load();

  return {
    reload: load,
    destroy() {
      destroyed = true;
      abort?.abort();
      gen += 1;
    },
  };
}
