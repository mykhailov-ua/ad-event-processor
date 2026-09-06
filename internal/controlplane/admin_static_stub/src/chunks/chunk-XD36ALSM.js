var w1 = Object.create;
var oc = Object.defineProperty;
var R1 = Object.getOwnPropertyDescriptor;
var _1 = Object.getOwnPropertyNames;
var I1 = Object.getPrototypeOf,
  k1 = Object.prototype.hasOwnProperty;
var pa = (e, t) => () => (t || e((t = { exports: {} }).exports, t), t.exports),
  nA = (e, t) => {
    for (var a in t) oc(e, a, { get: t[a], enumerable: !0 });
  },
  M1 = (e, t, a, r) => {
    if ((t && typeof t == 'object') || typeof t == 'function')
      for (let o of _1(t))
        !k1.call(e, o) &&
          o !== a &&
          oc(e, o, { get: () => t[o], enumerable: !(r = R1(t, o)) || r.enumerable });
    return e;
  };
var E = (e, t, a) => (
  (a = e != null ? w1(I1(e)) : {}),
  M1(t || !e || !e.__esModule ? oc(a, 'default', { value: e, enumerable: !0 }) : a, e)
);
var bh = pa((we) => {
  'use strict';
  function sc(e, t) {
    var a = e.length;
    e.push(t);
    e: for (; 0 < a; ) {
      var r = (a - 1) >>> 1,
        o = e[r];
      if (0 < _i(o, t)) (e[r] = t), (e[a] = o), (a = r);
      else break e;
    }
  }
  function ha(e) {
    return e.length === 0 ? null : e[0];
  }
  function ki(e) {
    if (e.length === 0) return null;
    var t = e[0],
      a = e.pop();
    if (a !== t) {
      e[0] = a;
      e: for (var r = 0, o = e.length, n = o >>> 1; r < n; ) {
        var l = 2 * (r + 1) - 1,
          i = e[l],
          s = l + 1,
          u = e[s];
        if (0 > _i(i, a))
          s < o && 0 > _i(u, i)
            ? ((e[r] = u), (e[s] = a), (r = s))
            : ((e[r] = i), (e[l] = a), (r = l));
        else if (s < o && 0 > _i(u, a)) (e[r] = u), (e[s] = a), (r = s);
        else break e;
      }
    }
    return t;
  }
  function _i(e, t) {
    var a = e.sortIndex - t.sortIndex;
    return a !== 0 ? a : e.id - t.id;
  }
  we.unstable_now = void 0;
  typeof performance == 'object' && typeof performance.now == 'function'
    ? ((ch = performance),
      (we.unstable_now = function () {
        return ch.now();
      }))
    : ((nc = Date),
      (dh = nc.now()),
      (we.unstable_now = function () {
        return nc.now() - dh;
      }));
  var ch,
    nc,
    dh,
    Aa = [],
    sr = [],
    E1 = 1,
    Vt = null,
    it = 3,
    uc = !1,
    rl = !1,
    ol = !1,
    cc = !1,
    ph = typeof setTimeout == 'function' ? setTimeout : null,
    hh = typeof clearTimeout == 'function' ? clearTimeout : null,
    fh = typeof setImmediate < 'u' ? setImmediate : null;
  function Ii(e) {
    for (var t = ha(sr); t !== null; ) {
      if (t.callback === null) ki(sr);
      else if (t.startTime <= e) ki(sr), (t.sortIndex = t.expirationTime), sc(Aa, t);
      else break;
      t = ha(sr);
    }
  }
  function dc(e) {
    if (((ol = !1), Ii(e), !rl))
      if (ha(Aa) !== null) (rl = !0), Do || ((Do = !0), To());
      else {
        var t = ha(sr);
        t !== null && fc(dc, t.startTime - e);
      }
  }
  var Do = !1,
    nl = -1,
    gh = 5,
    yh = -1;
  function vh() {
    return cc ? !0 : !(we.unstable_now() - yh < gh);
  }
  function lc() {
    if (((cc = !1), Do)) {
      var e = we.unstable_now();
      yh = e;
      var t = !0;
      try {
        e: {
          (rl = !1), ol && ((ol = !1), hh(nl), (nl = -1)), (uc = !0);
          var a = it;
          try {
            t: {
              for (Ii(e), Vt = ha(Aa); Vt !== null && !(Vt.expirationTime > e && vh()); ) {
                var r = Vt.callback;
                if (typeof r == 'function') {
                  (Vt.callback = null), (it = Vt.priorityLevel);
                  var o = r(Vt.expirationTime <= e);
                  if (((e = we.unstable_now()), typeof o == 'function')) {
                    (Vt.callback = o), Ii(e), (t = !0);
                    break t;
                  }
                  Vt === ha(Aa) && ki(Aa), Ii(e);
                } else ki(Aa);
                Vt = ha(Aa);
              }
              if (Vt !== null) t = !0;
              else {
                var n = ha(sr);
                n !== null && fc(dc, n.startTime - e), (t = !1);
              }
            }
            break e;
          } finally {
            (Vt = null), (it = a), (uc = !1);
          }
          t = void 0;
        }
      } finally {
        t ? To() : (Do = !1);
      }
    }
  }
  var To;
  typeof fh == 'function'
    ? (To = function () {
        fh(lc);
      })
    : typeof MessageChannel < 'u'
      ? ((ic = new MessageChannel()),
        (mh = ic.port2),
        (ic.port1.onmessage = lc),
        (To = function () {
          mh.postMessage(null);
        }))
      : (To = function () {
          ph(lc, 0);
        });
  var ic, mh;
  function fc(e, t) {
    nl = ph(function () {
      e(we.unstable_now());
    }, t);
  }
  we.unstable_IdlePriority = 5;
  we.unstable_ImmediatePriority = 1;
  we.unstable_LowPriority = 4;
  we.unstable_NormalPriority = 3;
  we.unstable_Profiling = null;
  we.unstable_UserBlockingPriority = 2;
  we.unstable_cancelCallback = function (e) {
    e.callback = null;
  };
  we.unstable_forceFrameRate = function (e) {
    0 > e || 125 < e
      ? console.error(
          'forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported'
        )
      : (gh = 0 < e ? Math.floor(1e3 / e) : 5);
  };
  we.unstable_getCurrentPriorityLevel = function () {
    return it;
  };
  we.unstable_next = function (e) {
    switch (it) {
      case 1:
      case 2:
      case 3:
        var t = 3;
        break;
      default:
        t = it;
    }
    var a = it;
    it = t;
    try {
      return e();
    } finally {
      it = a;
    }
  };
  we.unstable_requestPaint = function () {
    cc = !0;
  };
  we.unstable_runWithPriority = function (e, t) {
    switch (e) {
      case 1:
      case 2:
      case 3:
      case 4:
      case 5:
        break;
      default:
        e = 3;
    }
    var a = it;
    it = e;
    try {
      return t();
    } finally {
      it = a;
    }
  };
  we.unstable_scheduleCallback = function (e, t, a) {
    var r = we.unstable_now();
    switch (
      (typeof a == 'object' && a !== null
        ? ((a = a.delay), (a = typeof a == 'number' && 0 < a ? r + a : r))
        : (a = r),
      e)
    ) {
      case 1:
        var o = -1;
        break;
      case 2:
        o = 250;
        break;
      case 5:
        o = 1073741823;
        break;
      case 4:
        o = 1e4;
        break;
      default:
        o = 5e3;
    }
    return (
      (o = a + o),
      (e = {
        id: E1++,
        callback: t,
        priorityLevel: e,
        startTime: a,
        expirationTime: o,
        sortIndex: -1,
      }),
      a > r
        ? ((e.sortIndex = a),
          sc(sr, e),
          ha(Aa) === null && e === ha(sr) && (ol ? (hh(nl), (nl = -1)) : (ol = !0), fc(dc, a - r)))
        : ((e.sortIndex = o), sc(Aa, e), rl || uc || ((rl = !0), Do || ((Do = !0), To()))),
      e
    );
  };
  we.unstable_shouldYield = vh;
  we.unstable_wrapCallback = function (e) {
    var t = it;
    return function () {
      var a = it;
      it = t;
      try {
        return e.apply(this, arguments);
      } finally {
        it = a;
      }
    };
  };
});
var Sh = pa((sA, xh) => {
  'use strict';
  xh.exports = bh();
});
var Th = pa((F) => {
  'use strict';
  var hc = Symbol.for('react.transitional.element'),
    A1 = Symbol.for('react.portal'),
    T1 = Symbol.for('react.fragment'),
    D1 = Symbol.for('react.strict_mode'),
    O1 = Symbol.for('react.profiler'),
    P1 = Symbol.for('react.consumer'),
    B1 = Symbol.for('react.context'),
    U1 = Symbol.for('react.forward_ref'),
    N1 = Symbol.for('react.suspense'),
    H1 = Symbol.for('react.memo'),
    _h = Symbol.for('react.lazy'),
    z1 = Symbol.for('react.activity'),
    Lh = Symbol.iterator;
  function F1(e) {
    return e === null || typeof e != 'object'
      ? null
      : ((e = (Lh && e[Lh]) || e['@@iterator']), typeof e == 'function' ? e : null);
  }
  var Ih = {
      isMounted: function () {
        return !1;
      },
      enqueueForceUpdate: function () {},
      enqueueReplaceState: function () {},
      enqueueSetState: function () {},
    },
    kh = Object.assign,
    Mh = {};
  function Po(e, t, a) {
    (this.props = e), (this.context = t), (this.refs = Mh), (this.updater = a || Ih);
  }
  Po.prototype.isReactComponent = {};
  Po.prototype.setState = function (e, t) {
    if (typeof e != 'object' && typeof e != 'function' && e != null)
      throw Error(
        'takes an object of state variables to update or a function which returns an object of state variables.'
      );
    this.updater.enqueueSetState(this, e, t, 'setState');
  };
  Po.prototype.forceUpdate = function (e) {
    this.updater.enqueueForceUpdate(this, e, 'forceUpdate');
  };
  function Eh() {}
  Eh.prototype = Po.prototype;
  function gc(e, t, a) {
    (this.props = e), (this.context = t), (this.refs = Mh), (this.updater = a || Ih);
  }
  var yc = (gc.prototype = new Eh());
  yc.constructor = gc;
  kh(yc, Po.prototype);
  yc.isPureReactComponent = !0;
  var Ch = Array.isArray;
  function pc() {}
  var xe = { H: null, A: null, T: null, S: null },
    Ah = Object.prototype.hasOwnProperty;
  function vc(e, t, a) {
    var r = a.ref;
    return { $$typeof: hc, type: e, key: t, ref: r !== void 0 ? r : null, props: a };
  }
  function q1(e, t) {
    return vc(e.type, t, e.props);
  }
  function bc(e) {
    return typeof e == 'object' && e !== null && e.$$typeof === hc;
  }
  function G1(e) {
    var t = { '=': '=0', ':': '=2' };
    return (
      '$' +
      e.replace(/[=:]/g, function (a) {
        return t[a];
      })
    );
  }
  var wh = /\/+/g;
  function mc(e, t) {
    return typeof e == 'object' && e !== null && e.key != null ? G1('' + e.key) : t.toString(36);
  }
  function V1(e) {
    switch (e.status) {
      case 'fulfilled':
        return e.value;
      case 'rejected':
        throw e.reason;
      default:
        switch (
          (typeof e.status == 'string'
            ? e.then(pc, pc)
            : ((e.status = 'pending'),
              e.then(
                function (t) {
                  e.status === 'pending' && ((e.status = 'fulfilled'), (e.value = t));
                },
                function (t) {
                  e.status === 'pending' && ((e.status = 'rejected'), (e.reason = t));
                }
              )),
          e.status)
        ) {
          case 'fulfilled':
            return e.value;
          case 'rejected':
            throw e.reason;
        }
    }
    throw e;
  }
  function Oo(e, t, a, r, o) {
    var n = typeof e;
    (n === 'undefined' || n === 'boolean') && (e = null);
    var l = !1;
    if (e === null) l = !0;
    else
      switch (n) {
        case 'bigint':
        case 'string':
        case 'number':
          l = !0;
          break;
        case 'object':
          switch (e.$$typeof) {
            case hc:
            case A1:
              l = !0;
              break;
            case _h:
              return (l = e._init), Oo(l(e._payload), t, a, r, o);
          }
      }
    if (l)
      return (
        (o = o(e)),
        (l = r === '' ? '.' + mc(e, 0) : r),
        Ch(o)
          ? ((a = ''),
            l != null && (a = l.replace(wh, '$&/') + '/'),
            Oo(o, t, a, '', function (u) {
              return u;
            }))
          : o != null &&
            (bc(o) &&
              (o = q1(
                o,
                a +
                  (o.key == null || (e && e.key === o.key)
                    ? ''
                    : ('' + o.key).replace(wh, '$&/') + '/') +
                  l
              )),
            t.push(o)),
        1
      );
    l = 0;
    var i = r === '' ? '.' : r + ':';
    if (Ch(e))
      for (var s = 0; s < e.length; s++) (r = e[s]), (n = i + mc(r, s)), (l += Oo(r, t, a, n, o));
    else if (((s = F1(e)), typeof s == 'function'))
      for (e = s.call(e), s = 0; !(r = e.next()).done; )
        (r = r.value), (n = i + mc(r, s++)), (l += Oo(r, t, a, n, o));
    else if (n === 'object') {
      if (typeof e.then == 'function') return Oo(V1(e), t, a, r, o);
      throw (
        ((t = String(e)),
        Error(
          'Objects are not valid as a React child (found: ' +
            (t === '[object Object]' ? 'object with keys {' + Object.keys(e).join(', ') + '}' : t) +
            '). If you meant to render a collection of children, use an array instead.'
        ))
      );
    }
    return l;
  }
  function Mi(e, t, a) {
    if (e == null) return e;
    var r = [],
      o = 0;
    return (
      Oo(e, r, '', '', function (n) {
        return t.call(a, n, o++);
      }),
      r
    );
  }
  function j1(e) {
    if (e._status === -1) {
      var t = e._result;
      (t = t()),
        t.then(
          function (a) {
            (e._status === 0 || e._status === -1) && ((e._status = 1), (e._result = a));
          },
          function (a) {
            (e._status === 0 || e._status === -1) && ((e._status = 2), (e._result = a));
          }
        ),
        e._status === -1 && ((e._status = 0), (e._result = t));
    }
    if (e._status === 1) return e._result.default;
    throw e._result;
  }
  var Rh =
      typeof reportError == 'function'
        ? reportError
        : function (e) {
            if (typeof window == 'object' && typeof window.ErrorEvent == 'function') {
              var t = new window.ErrorEvent('error', {
                bubbles: !0,
                cancelable: !0,
                message:
                  typeof e == 'object' && e !== null && typeof e.message == 'string'
                    ? String(e.message)
                    : String(e),
                error: e,
              });
              if (!window.dispatchEvent(t)) return;
            } else if (typeof process == 'object' && typeof process.emit == 'function') {
              process.emit('uncaughtException', e);
              return;
            }
            console.error(e);
          },
    Y1 = {
      map: Mi,
      forEach: function (e, t, a) {
        Mi(
          e,
          function () {
            t.apply(this, arguments);
          },
          a
        );
      },
      count: function (e) {
        var t = 0;
        return (
          Mi(e, function () {
            t++;
          }),
          t
        );
      },
      toArray: function (e) {
        return (
          Mi(e, function (t) {
            return t;
          }) || []
        );
      },
      only: function (e) {
        if (!bc(e))
          throw Error('React.Children.only expected to receive a single React element child.');
        return e;
      },
    };
  F.Activity = z1;
  F.Children = Y1;
  F.Component = Po;
  F.Fragment = T1;
  F.Profiler = O1;
  F.PureComponent = gc;
  F.StrictMode = D1;
  F.Suspense = N1;
  F.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE = xe;
  F.__COMPILER_RUNTIME = {
    __proto__: null,
    c: function (e) {
      return xe.H.useMemoCache(e);
    },
  };
  F.cache = function (e) {
    return function () {
      return e.apply(null, arguments);
    };
  };
  F.cacheSignal = function () {
    return null;
  };
  F.cloneElement = function (e, t, a) {
    if (e == null) throw Error('The argument must be a React element, but you passed ' + e + '.');
    var r = kh({}, e.props),
      o = e.key;
    if (t != null)
      for (n in (t.key !== void 0 && (o = '' + t.key), t))
        !Ah.call(t, n) ||
          n === 'key' ||
          n === '__self' ||
          n === '__source' ||
          (n === 'ref' && t.ref === void 0) ||
          (r[n] = t[n]);
    var n = arguments.length - 2;
    if (n === 1) r.children = a;
    else if (1 < n) {
      for (var l = Array(n), i = 0; i < n; i++) l[i] = arguments[i + 2];
      r.children = l;
    }
    return vc(e.type, o, r);
  };
  F.createContext = function (e) {
    return (
      (e = {
        $$typeof: B1,
        _currentValue: e,
        _currentValue2: e,
        _threadCount: 0,
        Provider: null,
        Consumer: null,
      }),
      (e.Provider = e),
      (e.Consumer = { $$typeof: P1, _context: e }),
      e
    );
  };
  F.createElement = function (e, t, a) {
    var r,
      o = {},
      n = null;
    if (t != null)
      for (r in (t.key !== void 0 && (n = '' + t.key), t))
        Ah.call(t, r) && r !== 'key' && r !== '__self' && r !== '__source' && (o[r] = t[r]);
    var l = arguments.length - 2;
    if (l === 1) o.children = a;
    else if (1 < l) {
      for (var i = Array(l), s = 0; s < l; s++) i[s] = arguments[s + 2];
      o.children = i;
    }
    if (e && e.defaultProps) for (r in ((l = e.defaultProps), l)) o[r] === void 0 && (o[r] = l[r]);
    return vc(e, n, o);
  };
  F.createRef = function () {
    return { current: null };
  };
  F.forwardRef = function (e) {
    return { $$typeof: U1, render: e };
  };
  F.isValidElement = bc;
  F.lazy = function (e) {
    return { $$typeof: _h, _payload: { _status: -1, _result: e }, _init: j1 };
  };
  F.memo = function (e, t) {
    return { $$typeof: H1, type: e, compare: t === void 0 ? null : t };
  };
  F.startTransition = function (e) {
    var t = xe.T,
      a = {};
    xe.T = a;
    try {
      var r = e(),
        o = xe.S;
      o !== null && o(a, r),
        typeof r == 'object' && r !== null && typeof r.then == 'function' && r.then(pc, Rh);
    } catch (n) {
      Rh(n);
    } finally {
      t !== null && a.types !== null && (t.types = a.types), (xe.T = t);
    }
  };
  F.unstable_useCacheRefresh = function () {
    return xe.H.useCacheRefresh();
  };
  F.use = function (e) {
    return xe.H.use(e);
  };
  F.useActionState = function (e, t, a) {
    return xe.H.useActionState(e, t, a);
  };
  F.useCallback = function (e, t) {
    return xe.H.useCallback(e, t);
  };
  F.useContext = function (e) {
    return xe.H.useContext(e);
  };
  F.useDebugValue = function () {};
  F.useDeferredValue = function (e, t) {
    return xe.H.useDeferredValue(e, t);
  };
  F.useEffect = function (e, t) {
    return xe.H.useEffect(e, t);
  };
  F.useEffectEvent = function (e) {
    return xe.H.useEffectEvent(e);
  };
  F.useId = function () {
    return xe.H.useId();
  };
  F.useImperativeHandle = function (e, t, a) {
    return xe.H.useImperativeHandle(e, t, a);
  };
  F.useInsertionEffect = function (e, t) {
    return xe.H.useInsertionEffect(e, t);
  };
  F.useLayoutEffect = function (e, t) {
    return xe.H.useLayoutEffect(e, t);
  };
  F.useMemo = function (e, t) {
    return xe.H.useMemo(e, t);
  };
  F.useOptimistic = function (e, t) {
    return xe.H.useOptimistic(e, t);
  };
  F.useReducer = function (e, t, a) {
    return xe.H.useReducer(e, t, a);
  };
  F.useRef = function (e) {
    return xe.H.useRef(e);
  };
  F.useState = function (e) {
    return xe.H.useState(e);
  };
  F.useSyncExternalStore = function (e, t, a) {
    return xe.H.useSyncExternalStore(e, t, a);
  };
  F.useTransition = function () {
    return xe.H.useTransition();
  };
  F.version = '19.2.8';
});
var te = pa((cA, Dh) => {
  'use strict';
  Dh.exports = Th();
});
var Ph = pa((ft) => {
  'use strict';
  var X1 = te();
  function Oh(e) {
    var t = 'https://react.dev/errors/' + e;
    if (1 < arguments.length) {
      t += '?args[]=' + encodeURIComponent(arguments[1]);
      for (var a = 2; a < arguments.length; a++) t += '&args[]=' + encodeURIComponent(arguments[a]);
    }
    return (
      'Minified React error #' +
      e +
      '; visit ' +
      t +
      ' for the full message or use the non-minified dev environment for full errors and additional helpful warnings.'
    );
  }
  function ur() {}
  var dt = {
      d: {
        f: ur,
        r: function () {
          throw Error(Oh(522));
        },
        D: ur,
        C: ur,
        L: ur,
        m: ur,
        X: ur,
        S: ur,
        M: ur,
      },
      p: 0,
      findDOMNode: null,
    },
    W1 = Symbol.for('react.portal');
  function Q1(e, t, a) {
    var r = 3 < arguments.length && arguments[3] !== void 0 ? arguments[3] : null;
    return {
      $$typeof: W1,
      key: r == null ? null : '' + r,
      children: e,
      containerInfo: t,
      implementation: a,
    };
  }
  var ll = X1.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE;
  function Ei(e, t) {
    if (e === 'font') return '';
    if (typeof t == 'string') return t === 'use-credentials' ? t : '';
  }
  ft.__DOM_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE = dt;
  ft.createPortal = function (e, t) {
    var a = 2 < arguments.length && arguments[2] !== void 0 ? arguments[2] : null;
    if (!t || (t.nodeType !== 1 && t.nodeType !== 9 && t.nodeType !== 11)) throw Error(Oh(299));
    return Q1(e, t, null, a);
  };
  ft.flushSync = function (e) {
    var t = ll.T,
      a = dt.p;
    try {
      if (((ll.T = null), (dt.p = 2), e)) return e();
    } finally {
      (ll.T = t), (dt.p = a), dt.d.f();
    }
  };
  ft.preconnect = function (e, t) {
    typeof e == 'string' &&
      (t
        ? ((t = t.crossOrigin),
          (t = typeof t == 'string' ? (t === 'use-credentials' ? t : '') : void 0))
        : (t = null),
      dt.d.C(e, t));
  };
  ft.prefetchDNS = function (e) {
    typeof e == 'string' && dt.d.D(e);
  };
  ft.preinit = function (e, t) {
    if (typeof e == 'string' && t && typeof t.as == 'string') {
      var a = t.as,
        r = Ei(a, t.crossOrigin),
        o = typeof t.integrity == 'string' ? t.integrity : void 0,
        n = typeof t.fetchPriority == 'string' ? t.fetchPriority : void 0;
      a === 'style'
        ? dt.d.S(e, typeof t.precedence == 'string' ? t.precedence : void 0, {
            crossOrigin: r,
            integrity: o,
            fetchPriority: n,
          })
        : a === 'script' &&
          dt.d.X(e, {
            crossOrigin: r,
            integrity: o,
            fetchPriority: n,
            nonce: typeof t.nonce == 'string' ? t.nonce : void 0,
          });
    }
  };
  ft.preinitModule = function (e, t) {
    if (typeof e == 'string')
      if (typeof t == 'object' && t !== null) {
        if (t.as == null || t.as === 'script') {
          var a = Ei(t.as, t.crossOrigin);
          dt.d.M(e, {
            crossOrigin: a,
            integrity: typeof t.integrity == 'string' ? t.integrity : void 0,
            nonce: typeof t.nonce == 'string' ? t.nonce : void 0,
          });
        }
      } else t == null && dt.d.M(e);
  };
  ft.preload = function (e, t) {
    if (typeof e == 'string' && typeof t == 'object' && t !== null && typeof t.as == 'string') {
      var a = t.as,
        r = Ei(a, t.crossOrigin);
      dt.d.L(e, a, {
        crossOrigin: r,
        integrity: typeof t.integrity == 'string' ? t.integrity : void 0,
        nonce: typeof t.nonce == 'string' ? t.nonce : void 0,
        type: typeof t.type == 'string' ? t.type : void 0,
        fetchPriority: typeof t.fetchPriority == 'string' ? t.fetchPriority : void 0,
        referrerPolicy: typeof t.referrerPolicy == 'string' ? t.referrerPolicy : void 0,
        imageSrcSet: typeof t.imageSrcSet == 'string' ? t.imageSrcSet : void 0,
        imageSizes: typeof t.imageSizes == 'string' ? t.imageSizes : void 0,
        media: typeof t.media == 'string' ? t.media : void 0,
      });
    }
  };
  ft.preloadModule = function (e, t) {
    if (typeof e == 'string')
      if (t) {
        var a = Ei(t.as, t.crossOrigin);
        dt.d.m(e, {
          as: typeof t.as == 'string' && t.as !== 'script' ? t.as : void 0,
          crossOrigin: a,
          integrity: typeof t.integrity == 'string' ? t.integrity : void 0,
        });
      } else dt.d.m(e);
  };
  ft.requestFormReset = function (e) {
    dt.d.r(e);
  };
  ft.unstable_batchedUpdates = function (e, t) {
    return e(t);
  };
  ft.useFormState = function (e, t, a) {
    return ll.H.useFormState(e, t, a);
  };
  ft.useFormStatus = function () {
    return ll.H.useHostTransitionStatus();
  };
  ft.version = '19.2.8';
});
var xc = pa((fA, Uh) => {
  'use strict';
  function Bh() {
    if (
      !(
        typeof __REACT_DEVTOOLS_GLOBAL_HOOK__ > 'u' ||
        typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE != 'function'
      )
    )
      try {
        __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Bh);
      } catch (e) {
        console.error(e);
      }
  }
  Bh(), (Uh.exports = Ph());
});
var Qv = pa((tu) => {
  'use strict';
  var ze = Sh(),
    uy = te(),
    K1 = xc();
  function C(e) {
    var t = 'https://react.dev/errors/' + e;
    if (1 < arguments.length) {
      t += '?args[]=' + encodeURIComponent(arguments[1]);
      for (var a = 2; a < arguments.length; a++) t += '&args[]=' + encodeURIComponent(arguments[a]);
    }
    return (
      'Minified React error #' +
      e +
      '; visit ' +
      t +
      ' for the full message or use the non-minified dev environment for full errors and additional helpful warnings.'
    );
  }
  function cy(e) {
    return !(!e || (e.nodeType !== 1 && e.nodeType !== 9 && e.nodeType !== 11));
  }
  function Yl(e) {
    var t = e,
      a = e;
    if (e.alternate) for (; t.return; ) t = t.return;
    else {
      e = t;
      do (t = e), (t.flags & 4098) !== 0 && (a = t.return), (e = t.return);
      while (e);
    }
    return t.tag === 3 ? a : null;
  }
  function dy(e) {
    if (e.tag === 13) {
      var t = e.memoizedState;
      if ((t === null && ((e = e.alternate), e !== null && (t = e.memoizedState)), t !== null))
        return t.dehydrated;
    }
    return null;
  }
  function fy(e) {
    if (e.tag === 31) {
      var t = e.memoizedState;
      if ((t === null && ((e = e.alternate), e !== null && (t = e.memoizedState)), t !== null))
        return t.dehydrated;
    }
    return null;
  }
  function Nh(e) {
    if (Yl(e) !== e) throw Error(C(188));
  }
  function Z1(e) {
    var t = e.alternate;
    if (!t) {
      if (((t = Yl(e)), t === null)) throw Error(C(188));
      return t !== e ? null : e;
    }
    for (var a = e, r = t; ; ) {
      var o = a.return;
      if (o === null) break;
      var n = o.alternate;
      if (n === null) {
        if (((r = o.return), r !== null)) {
          a = r;
          continue;
        }
        break;
      }
      if (o.child === n.child) {
        for (n = o.child; n; ) {
          if (n === a) return Nh(o), e;
          if (n === r) return Nh(o), t;
          n = n.sibling;
        }
        throw Error(C(188));
      }
      if (a.return !== r.return) (a = o), (r = n);
      else {
        for (var l = !1, i = o.child; i; ) {
          if (i === a) {
            (l = !0), (a = o), (r = n);
            break;
          }
          if (i === r) {
            (l = !0), (r = o), (a = n);
            break;
          }
          i = i.sibling;
        }
        if (!l) {
          for (i = n.child; i; ) {
            if (i === a) {
              (l = !0), (a = n), (r = o);
              break;
            }
            if (i === r) {
              (l = !0), (r = n), (a = o);
              break;
            }
            i = i.sibling;
          }
          if (!l) throw Error(C(189));
        }
      }
      if (a.alternate !== r) throw Error(C(190));
    }
    if (a.tag !== 3) throw Error(C(188));
    return a.stateNode.current === a ? e : t;
  }
  function my(e) {
    var t = e.tag;
    if (t === 5 || t === 26 || t === 27 || t === 6) return e;
    for (e = e.child; e !== null; ) {
      if (((t = my(e)), t !== null)) return t;
      e = e.sibling;
    }
    return null;
  }
  var Ce = Object.assign,
    J1 = Symbol.for('react.element'),
    Ai = Symbol.for('react.transitional.element'),
    pl = Symbol.for('react.portal'),
    Fo = Symbol.for('react.fragment'),
    py = Symbol.for('react.strict_mode'),
    ed = Symbol.for('react.profiler'),
    hy = Symbol.for('react.consumer'),
    Ha = Symbol.for('react.context'),
    Qd = Symbol.for('react.forward_ref'),
    td = Symbol.for('react.suspense'),
    ad = Symbol.for('react.suspense_list'),
    Kd = Symbol.for('react.memo'),
    cr = Symbol.for('react.lazy');
  Symbol.for('react.scope');
  var rd = Symbol.for('react.activity');
  Symbol.for('react.legacy_hidden');
  Symbol.for('react.tracing_marker');
  var $1 = Symbol.for('react.memo_cache_sentinel');
  Symbol.for('react.view_transition');
  var Hh = Symbol.iterator;
  function il(e) {
    return e === null || typeof e != 'object'
      ? null
      : ((e = (Hh && e[Hh]) || e['@@iterator']), typeof e == 'function' ? e : null);
  }
  var eC = Symbol.for('react.client.reference');
  function od(e) {
    if (e == null) return null;
    if (typeof e == 'function') return e.$$typeof === eC ? null : e.displayName || e.name || null;
    if (typeof e == 'string') return e;
    switch (e) {
      case Fo:
        return 'Fragment';
      case ed:
        return 'Profiler';
      case py:
        return 'StrictMode';
      case td:
        return 'Suspense';
      case ad:
        return 'SuspenseList';
      case rd:
        return 'Activity';
    }
    if (typeof e == 'object')
      switch (e.$$typeof) {
        case pl:
          return 'Portal';
        case Ha:
          return e.displayName || 'Context';
        case hy:
          return (e._context.displayName || 'Context') + '.Consumer';
        case Qd:
          var t = e.render;
          return (
            (e = e.displayName),
            e ||
              ((e = t.displayName || t.name || ''),
              (e = e !== '' ? 'ForwardRef(' + e + ')' : 'ForwardRef')),
            e
          );
        case Kd:
          return (t = e.displayName || null), t !== null ? t : od(e.type) || 'Memo';
        case cr:
          (t = e._payload), (e = e._init);
          try {
            return od(e(t));
          } catch {}
      }
    return null;
  }
  var hl = Array.isArray,
    N = uy.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE,
    le = K1.__DOM_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE,
    Qr = { pending: !1, data: null, method: null, action: null },
    nd = [],
    qo = -1;
  function xa(e) {
    return { current: e };
  }
  function Ye(e) {
    0 > qo || ((e.current = nd[qo]), (nd[qo] = null), qo--);
  }
  function ye(e, t) {
    qo++, (nd[qo] = e.current), (e.current = t);
  }
  var ba = xa(null),
    Tl = xa(null),
    Sr = xa(null),
    cs = xa(null);
  function ds(e, t) {
    switch ((ye(Sr, t), ye(Tl, e), ye(ba, null), t.nodeType)) {
      case 9:
      case 11:
        e = (e = t.documentElement) && (e = e.namespaceURI) ? Yg(e) : 0;
        break;
      default:
        if (((e = t.tagName), (t = t.namespaceURI))) (t = Yg(t)), (e = Pv(t, e));
        else
          switch (e) {
            case 'svg':
              e = 1;
              break;
            case 'math':
              e = 2;
              break;
            default:
              e = 0;
          }
    }
    Ye(ba), ye(ba, e);
  }
  function ln() {
    Ye(ba), Ye(Tl), Ye(Sr);
  }
  function ld(e) {
    e.memoizedState !== null && ye(cs, e);
    var t = ba.current,
      a = Pv(t, e.type);
    t !== a && (ye(Tl, e), ye(ba, a));
  }
  function fs(e) {
    Tl.current === e && (Ye(ba), Ye(Tl)), cs.current === e && (Ye(cs), (Gl._currentValue = Qr));
  }
  var Sc, zh;
  function jr(e) {
    if (Sc === void 0)
      try {
        throw Error();
      } catch (a) {
        var t = a.stack.trim().match(/\n( *(at )?)/);
        (Sc = (t && t[1]) || ''),
          (zh =
            -1 <
            a.stack.indexOf(`
    at`)
              ? ' (<anonymous>)'
              : -1 < a.stack.indexOf('@')
                ? '@unknown:0:0'
                : '');
      }
    return (
      `
` +
      Sc +
      e +
      zh
    );
  }
  var Lc = !1;
  function Cc(e, t) {
    if (!e || Lc) return '';
    Lc = !0;
    var a = Error.prepareStackTrace;
    Error.prepareStackTrace = void 0;
    try {
      var r = {
        DetermineComponentFrameRoot: function () {
          try {
            if (t) {
              var c = function () {
                throw Error();
              };
              if (
                (Object.defineProperty(c.prototype, 'props', {
                  set: function () {
                    throw Error();
                  },
                }),
                typeof Reflect == 'object' && Reflect.construct)
              ) {
                try {
                  Reflect.construct(c, []);
                } catch (h) {
                  var f = h;
                }
                Reflect.construct(e, [], c);
              } else {
                try {
                  c.call();
                } catch (h) {
                  f = h;
                }
                e.call(c.prototype);
              }
            } else {
              try {
                throw Error();
              } catch (h) {
                f = h;
              }
              (c = e()) && typeof c.catch == 'function' && c.catch(function () {});
            }
          } catch (h) {
            if (h && f && typeof h.stack == 'string') return [h.stack, f.stack];
          }
          return [null, null];
        },
      };
      r.DetermineComponentFrameRoot.displayName = 'DetermineComponentFrameRoot';
      var o = Object.getOwnPropertyDescriptor(r.DetermineComponentFrameRoot, 'name');
      o &&
        o.configurable &&
        Object.defineProperty(r.DetermineComponentFrameRoot, 'name', {
          value: 'DetermineComponentFrameRoot',
        });
      var n = r.DetermineComponentFrameRoot(),
        l = n[0],
        i = n[1];
      if (l && i) {
        var s = l.split(`
`),
          u = i.split(`
`);
        for (o = r = 0; r < s.length && !s[r].includes('DetermineComponentFrameRoot'); ) r++;
        for (; o < u.length && !u[o].includes('DetermineComponentFrameRoot'); ) o++;
        if (r === s.length || o === u.length)
          for (r = s.length - 1, o = u.length - 1; 1 <= r && 0 <= o && s[r] !== u[o]; ) o--;
        for (; 1 <= r && 0 <= o; r--, o--)
          if (s[r] !== u[o]) {
            if (r !== 1 || o !== 1)
              do
                if ((r--, o--, 0 > o || s[r] !== u[o])) {
                  var d =
                    `
` + s[r].replace(' at new ', ' at ');
                  return (
                    e.displayName &&
                      d.includes('<anonymous>') &&
                      (d = d.replace('<anonymous>', e.displayName)),
                    d
                  );
                }
              while (1 <= r && 0 <= o);
            break;
          }
      }
    } finally {
      (Lc = !1), (Error.prepareStackTrace = a);
    }
    return (a = e ? e.displayName || e.name : '') ? jr(a) : '';
  }
  function tC(e, t) {
    switch (e.tag) {
      case 26:
      case 27:
      case 5:
        return jr(e.type);
      case 16:
        return jr('Lazy');
      case 13:
        return e.child !== t && t !== null ? jr('Suspense Fallback') : jr('Suspense');
      case 19:
        return jr('SuspenseList');
      case 0:
      case 15:
        return Cc(e.type, !1);
      case 11:
        return Cc(e.type.render, !1);
      case 1:
        return Cc(e.type, !0);
      case 31:
        return jr('Activity');
      default:
        return '';
    }
  }
  function Fh(e) {
    try {
      var t = '',
        a = null;
      do (t += tC(e, a)), (a = e), (e = e.return);
      while (e);
      return t;
    } catch (r) {
      return (
        `
Error generating stack: ` +
        r.message +
        `
` +
        r.stack
      );
    }
  }
  var id = Object.prototype.hasOwnProperty,
    Zd = ze.unstable_scheduleCallback,
    wc = ze.unstable_cancelCallback,
    aC = ze.unstable_shouldYield,
    rC = ze.unstable_requestPaint,
    Tt = ze.unstable_now,
    oC = ze.unstable_getCurrentPriorityLevel,
    gy = ze.unstable_ImmediatePriority,
    yy = ze.unstable_UserBlockingPriority,
    ms = ze.unstable_NormalPriority,
    nC = ze.unstable_LowPriority,
    vy = ze.unstable_IdlePriority,
    lC = ze.log,
    iC = ze.unstable_setDisableYieldValue,
    Xl = null,
    Dt = null;
  function gr(e) {
    if ((typeof lC == 'function' && iC(e), Dt && typeof Dt.setStrictMode == 'function'))
      try {
        Dt.setStrictMode(Xl, e);
      } catch {}
  }
  var Ot = Math.clz32 ? Math.clz32 : cC,
    sC = Math.log,
    uC = Math.LN2;
  function cC(e) {
    return (e >>>= 0), e === 0 ? 32 : (31 - ((sC(e) / uC) | 0)) | 0;
  }
  var Ti = 256,
    Di = 262144,
    Oi = 4194304;
  function Yr(e) {
    var t = e & 42;
    if (t !== 0) return t;
    switch (e & -e) {
      case 1:
        return 1;
      case 2:
        return 2;
      case 4:
        return 4;
      case 8:
        return 8;
      case 16:
        return 16;
      case 32:
        return 32;
      case 64:
        return 64;
      case 128:
        return 128;
      case 256:
      case 512:
      case 1024:
      case 2048:
      case 4096:
      case 8192:
      case 16384:
      case 32768:
      case 65536:
      case 131072:
        return e & 261888;
      case 262144:
      case 524288:
      case 1048576:
      case 2097152:
        return e & 3932160;
      case 4194304:
      case 8388608:
      case 16777216:
      case 33554432:
        return e & 62914560;
      case 67108864:
        return 67108864;
      case 134217728:
        return 134217728;
      case 268435456:
        return 268435456;
      case 536870912:
        return 536870912;
      case 1073741824:
        return 0;
      default:
        return e;
    }
  }
  function Hs(e, t, a) {
    var r = e.pendingLanes;
    if (r === 0) return 0;
    var o = 0,
      n = e.suspendedLanes,
      l = e.pingedLanes;
    e = e.warmLanes;
    var i = r & 134217727;
    return (
      i !== 0
        ? ((r = i & ~n),
          r !== 0
            ? (o = Yr(r))
            : ((l &= i), l !== 0 ? (o = Yr(l)) : a || ((a = i & ~e), a !== 0 && (o = Yr(a)))))
        : ((i = r & ~n),
          i !== 0
            ? (o = Yr(i))
            : l !== 0
              ? (o = Yr(l))
              : a || ((a = r & ~e), a !== 0 && (o = Yr(a)))),
      o === 0
        ? 0
        : t !== 0 &&
            t !== o &&
            (t & n) === 0 &&
            ((n = o & -o), (a = t & -t), n >= a || (n === 32 && (a & 4194048) !== 0))
          ? t
          : o
    );
  }
  function Wl(e, t) {
    return (e.pendingLanes & ~(e.suspendedLanes & ~e.pingedLanes) & t) === 0;
  }
  function dC(e, t) {
    switch (e) {
      case 1:
      case 2:
      case 4:
      case 8:
      case 64:
        return t + 250;
      case 16:
      case 32:
      case 128:
      case 256:
      case 512:
      case 1024:
      case 2048:
      case 4096:
      case 8192:
      case 16384:
      case 32768:
      case 65536:
      case 131072:
      case 262144:
      case 524288:
      case 1048576:
      case 2097152:
        return t + 5e3;
      case 4194304:
      case 8388608:
      case 16777216:
      case 33554432:
        return -1;
      case 67108864:
      case 134217728:
      case 268435456:
      case 536870912:
      case 1073741824:
        return -1;
      default:
        return -1;
    }
  }
  function by() {
    var e = Oi;
    return (Oi <<= 1), (Oi & 62914560) === 0 && (Oi = 4194304), e;
  }
  function Rc(e) {
    for (var t = [], a = 0; 31 > a; a++) t.push(e);
    return t;
  }
  function Ql(e, t) {
    (e.pendingLanes |= t),
      t !== 268435456 && ((e.suspendedLanes = 0), (e.pingedLanes = 0), (e.warmLanes = 0));
  }
  function fC(e, t, a, r, o, n) {
    var l = e.pendingLanes;
    (e.pendingLanes = a),
      (e.suspendedLanes = 0),
      (e.pingedLanes = 0),
      (e.warmLanes = 0),
      (e.expiredLanes &= a),
      (e.entangledLanes &= a),
      (e.errorRecoveryDisabledLanes &= a),
      (e.shellSuspendCounter = 0);
    var i = e.entanglements,
      s = e.expirationTimes,
      u = e.hiddenUpdates;
    for (a = l & ~a; 0 < a; ) {
      var d = 31 - Ot(a),
        c = 1 << d;
      (i[d] = 0), (s[d] = -1);
      var f = u[d];
      if (f !== null)
        for (u[d] = null, d = 0; d < f.length; d++) {
          var h = f[d];
          h !== null && (h.lane &= -536870913);
        }
      a &= ~c;
    }
    r !== 0 && xy(e, r, 0),
      n !== 0 && o === 0 && e.tag !== 0 && (e.suspendedLanes |= n & ~(l & ~t));
  }
  function xy(e, t, a) {
    (e.pendingLanes |= t), (e.suspendedLanes &= ~t);
    var r = 31 - Ot(t);
    (e.entangledLanes |= t), (e.entanglements[r] = e.entanglements[r] | 1073741824 | (a & 261930));
  }
  function Sy(e, t) {
    var a = (e.entangledLanes |= t);
    for (e = e.entanglements; a; ) {
      var r = 31 - Ot(a),
        o = 1 << r;
      (o & t) | (e[r] & t) && (e[r] |= t), (a &= ~o);
    }
  }
  function Ly(e, t) {
    var a = t & -t;
    return (a = (a & 42) !== 0 ? 1 : Jd(a)), (a & (e.suspendedLanes | t)) !== 0 ? 0 : a;
  }
  function Jd(e) {
    switch (e) {
      case 2:
        e = 1;
        break;
      case 8:
        e = 4;
        break;
      case 32:
        e = 16;
        break;
      case 256:
      case 512:
      case 1024:
      case 2048:
      case 4096:
      case 8192:
      case 16384:
      case 32768:
      case 65536:
      case 131072:
      case 262144:
      case 524288:
      case 1048576:
      case 2097152:
      case 4194304:
      case 8388608:
      case 16777216:
      case 33554432:
        e = 128;
        break;
      case 268435456:
        e = 134217728;
        break;
      default:
        e = 0;
    }
    return e;
  }
  function $d(e) {
    return (e &= -e), 2 < e ? (8 < e ? ((e & 134217727) !== 0 ? 32 : 268435456) : 8) : 2;
  }
  function Cy() {
    var e = le.p;
    return e !== 0 ? e : ((e = window.event), e === void 0 ? 32 : Yv(e.type));
  }
  function qh(e, t) {
    var a = le.p;
    try {
      return (le.p = e), t();
    } finally {
      le.p = a;
    }
  }
  var Or = Math.random().toString(36).slice(2),
    et = '__reactFiber$' + Or,
    St = '__reactProps$' + Or,
    vn = '__reactContainer$' + Or,
    sd = '__reactEvents$' + Or,
    mC = '__reactListeners$' + Or,
    pC = '__reactHandles$' + Or,
    Gh = '__reactResources$' + Or,
    Kl = '__reactMarker$' + Or;
  function ef(e) {
    delete e[et], delete e[St], delete e[sd], delete e[mC], delete e[pC];
  }
  function Go(e) {
    var t = e[et];
    if (t) return t;
    for (var a = e.parentNode; a; ) {
      if ((t = a[vn] || a[et])) {
        if (((a = t.alternate), t.child !== null || (a !== null && a.child !== null)))
          for (e = Zg(e); e !== null; ) {
            if ((a = e[et])) return a;
            e = Zg(e);
          }
        return t;
      }
      (e = a), (a = e.parentNode);
    }
    return null;
  }
  function bn(e) {
    if ((e = e[et] || e[vn])) {
      var t = e.tag;
      if (t === 5 || t === 6 || t === 13 || t === 31 || t === 26 || t === 27 || t === 3) return e;
    }
    return null;
  }
  function gl(e) {
    var t = e.tag;
    if (t === 5 || t === 26 || t === 27 || t === 6) return e.stateNode;
    throw Error(C(33));
  }
  function $o(e) {
    var t = e[Gh];
    return t || (t = e[Gh] = { hoistableStyles: new Map(), hoistableScripts: new Map() }), t;
  }
  function je(e) {
    e[Kl] = !0;
  }
  var wy = new Set(),
    Ry = {};
  function no(e, t) {
    sn(e, t), sn(e + 'Capture', t);
  }
  function sn(e, t) {
    for (Ry[e] = t, e = 0; e < t.length; e++) wy.add(t[e]);
  }
  var hC = RegExp(
      '^[:A-Z_a-z\\u00C0-\\u00D6\\u00D8-\\u00F6\\u00F8-\\u02FF\\u0370-\\u037D\\u037F-\\u1FFF\\u200C-\\u200D\\u2070-\\u218F\\u2C00-\\u2FEF\\u3001-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFFD][:A-Z_a-z\\u00C0-\\u00D6\\u00D8-\\u00F6\\u00F8-\\u02FF\\u0370-\\u037D\\u037F-\\u1FFF\\u200C-\\u200D\\u2070-\\u218F\\u2C00-\\u2FEF\\u3001-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFFD\\-.0-9\\u00B7\\u0300-\\u036F\\u203F-\\u2040]*$'
    ),
    Vh = {},
    jh = {};
  function gC(e) {
    return id.call(jh, e)
      ? !0
      : id.call(Vh, e)
        ? !1
        : hC.test(e)
          ? (jh[e] = !0)
          : ((Vh[e] = !0), !1);
  }
  function Qi(e, t, a) {
    if (gC(t))
      if (a === null) e.removeAttribute(t);
      else {
        switch (typeof a) {
          case 'undefined':
          case 'function':
          case 'symbol':
            e.removeAttribute(t);
            return;
          case 'boolean':
            var r = t.toLowerCase().slice(0, 5);
            if (r !== 'data-' && r !== 'aria-') {
              e.removeAttribute(t);
              return;
            }
        }
        e.setAttribute(t, '' + a);
      }
  }
  function Pi(e, t, a) {
    if (a === null) e.removeAttribute(t);
    else {
      switch (typeof a) {
        case 'undefined':
        case 'function':
        case 'symbol':
        case 'boolean':
          e.removeAttribute(t);
          return;
      }
      e.setAttribute(t, '' + a);
    }
  }
  function Ta(e, t, a, r) {
    if (r === null) e.removeAttribute(a);
    else {
      switch (typeof r) {
        case 'undefined':
        case 'function':
        case 'symbol':
        case 'boolean':
          e.removeAttribute(a);
          return;
      }
      e.setAttributeNS(t, a, '' + r);
    }
  }
  function Yt(e) {
    switch (typeof e) {
      case 'bigint':
      case 'boolean':
      case 'number':
      case 'string':
      case 'undefined':
        return e;
      case 'object':
        return e;
      default:
        return '';
    }
  }
  function _y(e) {
    var t = e.type;
    return (e = e.nodeName) && e.toLowerCase() === 'input' && (t === 'checkbox' || t === 'radio');
  }
  function yC(e, t, a) {
    var r = Object.getOwnPropertyDescriptor(e.constructor.prototype, t);
    if (
      !e.hasOwnProperty(t) &&
      typeof r < 'u' &&
      typeof r.get == 'function' &&
      typeof r.set == 'function'
    ) {
      var o = r.get,
        n = r.set;
      return (
        Object.defineProperty(e, t, {
          configurable: !0,
          get: function () {
            return o.call(this);
          },
          set: function (l) {
            (a = '' + l), n.call(this, l);
          },
        }),
        Object.defineProperty(e, t, { enumerable: r.enumerable }),
        {
          getValue: function () {
            return a;
          },
          setValue: function (l) {
            a = '' + l;
          },
          stopTracking: function () {
            (e._valueTracker = null), delete e[t];
          },
        }
      );
    }
  }
  function ud(e) {
    if (!e._valueTracker) {
      var t = _y(e) ? 'checked' : 'value';
      e._valueTracker = yC(e, t, '' + e[t]);
    }
  }
  function Iy(e) {
    if (!e) return !1;
    var t = e._valueTracker;
    if (!t) return !0;
    var a = t.getValue(),
      r = '';
    return (
      e && (r = _y(e) ? (e.checked ? 'true' : 'false') : e.value),
      (e = r),
      e !== a ? (t.setValue(e), !0) : !1
    );
  }
  function ps(e) {
    if (((e = e || (typeof document < 'u' ? document : void 0)), typeof e > 'u')) return null;
    try {
      return e.activeElement || e.body;
    } catch {
      return e.body;
    }
  }
  var vC = /[\n"\\]/g;
  function Qt(e) {
    return e.replace(vC, function (t) {
      return '\\' + t.charCodeAt(0).toString(16) + ' ';
    });
  }
  function cd(e, t, a, r, o, n, l, i) {
    (e.name = ''),
      l != null && typeof l != 'function' && typeof l != 'symbol' && typeof l != 'boolean'
        ? (e.type = l)
        : e.removeAttribute('type'),
      t != null
        ? l === 'number'
          ? ((t === 0 && e.value === '') || e.value != t) && (e.value = '' + Yt(t))
          : e.value !== '' + Yt(t) && (e.value = '' + Yt(t))
        : (l !== 'submit' && l !== 'reset') || e.removeAttribute('value'),
      t != null
        ? dd(e, l, Yt(t))
        : a != null
          ? dd(e, l, Yt(a))
          : r != null && e.removeAttribute('value'),
      o == null && n != null && (e.defaultChecked = !!n),
      o != null && (e.checked = o && typeof o != 'function' && typeof o != 'symbol'),
      i != null && typeof i != 'function' && typeof i != 'symbol' && typeof i != 'boolean'
        ? (e.name = '' + Yt(i))
        : e.removeAttribute('name');
  }
  function ky(e, t, a, r, o, n, l, i) {
    if (
      (n != null &&
        typeof n != 'function' &&
        typeof n != 'symbol' &&
        typeof n != 'boolean' &&
        (e.type = n),
      t != null || a != null)
    ) {
      if (!((n !== 'submit' && n !== 'reset') || t != null)) {
        ud(e);
        return;
      }
      (a = a != null ? '' + Yt(a) : ''),
        (t = t != null ? '' + Yt(t) : a),
        i || t === e.value || (e.value = t),
        (e.defaultValue = t);
    }
    (r = r ?? o),
      (r = typeof r != 'function' && typeof r != 'symbol' && !!r),
      (e.checked = i ? e.checked : !!r),
      (e.defaultChecked = !!r),
      l != null &&
        typeof l != 'function' &&
        typeof l != 'symbol' &&
        typeof l != 'boolean' &&
        (e.name = l),
      ud(e);
  }
  function dd(e, t, a) {
    (t === 'number' && ps(e.ownerDocument) === e) ||
      e.defaultValue === '' + a ||
      (e.defaultValue = '' + a);
  }
  function en(e, t, a, r) {
    if (((e = e.options), t)) {
      t = {};
      for (var o = 0; o < a.length; o++) t['$' + a[o]] = !0;
      for (a = 0; a < e.length; a++)
        (o = t.hasOwnProperty('$' + e[a].value)),
          e[a].selected !== o && (e[a].selected = o),
          o && r && (e[a].defaultSelected = !0);
    } else {
      for (a = '' + Yt(a), t = null, o = 0; o < e.length; o++) {
        if (e[o].value === a) {
          (e[o].selected = !0), r && (e[o].defaultSelected = !0);
          return;
        }
        t !== null || e[o].disabled || (t = e[o]);
      }
      t !== null && (t.selected = !0);
    }
  }
  function My(e, t, a) {
    if (t != null && ((t = '' + Yt(t)), t !== e.value && (e.value = t), a == null)) {
      e.defaultValue !== t && (e.defaultValue = t);
      return;
    }
    e.defaultValue = a != null ? '' + Yt(a) : '';
  }
  function Ey(e, t, a, r) {
    if (t == null) {
      if (r != null) {
        if (a != null) throw Error(C(92));
        if (hl(r)) {
          if (1 < r.length) throw Error(C(93));
          r = r[0];
        }
        a = r;
      }
      a == null && (a = ''), (t = a);
    }
    (a = Yt(t)),
      (e.defaultValue = a),
      (r = e.textContent),
      r === a && r !== '' && r !== null && (e.value = r),
      ud(e);
  }
  function un(e, t) {
    if (t) {
      var a = e.firstChild;
      if (a && a === e.lastChild && a.nodeType === 3) {
        a.nodeValue = t;
        return;
      }
    }
    e.textContent = t;
  }
  var bC = new Set(
    'animationIterationCount aspectRatio borderImageOutset borderImageSlice borderImageWidth boxFlex boxFlexGroup boxOrdinalGroup columnCount columns flex flexGrow flexPositive flexShrink flexNegative flexOrder gridArea gridRow gridRowEnd gridRowSpan gridRowStart gridColumn gridColumnEnd gridColumnSpan gridColumnStart fontWeight lineClamp lineHeight opacity order orphans scale tabSize widows zIndex zoom fillOpacity floodOpacity stopOpacity strokeDasharray strokeDashoffset strokeMiterlimit strokeOpacity strokeWidth MozAnimationIterationCount MozBoxFlex MozBoxFlexGroup MozLineClamp msAnimationIterationCount msFlex msZoom msFlexGrow msFlexNegative msFlexOrder msFlexPositive msFlexShrink msGridColumn msGridColumnSpan msGridRow msGridRowSpan WebkitAnimationIterationCount WebkitBoxFlex WebKitBoxFlexGroup WebkitBoxOrdinalGroup WebkitColumnCount WebkitColumns WebkitFlex WebkitFlexGrow WebkitFlexPositive WebkitFlexShrink WebkitLineClamp'.split(
      ' '
    )
  );
  function Yh(e, t, a) {
    var r = t.indexOf('--') === 0;
    a == null || typeof a == 'boolean' || a === ''
      ? r
        ? e.setProperty(t, '')
        : t === 'float'
          ? (e.cssFloat = '')
          : (e[t] = '')
      : r
        ? e.setProperty(t, a)
        : typeof a != 'number' || a === 0 || bC.has(t)
          ? t === 'float'
            ? (e.cssFloat = a)
            : (e[t] = ('' + a).trim())
          : (e[t] = a + 'px');
  }
  function Ay(e, t, a) {
    if (t != null && typeof t != 'object') throw Error(C(62));
    if (((e = e.style), a != null)) {
      for (var r in a)
        !a.hasOwnProperty(r) ||
          (t != null && t.hasOwnProperty(r)) ||
          (r.indexOf('--') === 0
            ? e.setProperty(r, '')
            : r === 'float'
              ? (e.cssFloat = '')
              : (e[r] = ''));
      for (var o in t) (r = t[o]), t.hasOwnProperty(o) && a[o] !== r && Yh(e, o, r);
    } else for (var n in t) t.hasOwnProperty(n) && Yh(e, n, t[n]);
  }
  function tf(e) {
    if (e.indexOf('-') === -1) return !1;
    switch (e) {
      case 'annotation-xml':
      case 'color-profile':
      case 'font-face':
      case 'font-face-src':
      case 'font-face-uri':
      case 'font-face-format':
      case 'font-face-name':
      case 'missing-glyph':
        return !1;
      default:
        return !0;
    }
  }
  var xC = new Map([
      ['acceptCharset', 'accept-charset'],
      ['htmlFor', 'for'],
      ['httpEquiv', 'http-equiv'],
      ['crossOrigin', 'crossorigin'],
      ['accentHeight', 'accent-height'],
      ['alignmentBaseline', 'alignment-baseline'],
      ['arabicForm', 'arabic-form'],
      ['baselineShift', 'baseline-shift'],
      ['capHeight', 'cap-height'],
      ['clipPath', 'clip-path'],
      ['clipRule', 'clip-rule'],
      ['colorInterpolation', 'color-interpolation'],
      ['colorInterpolationFilters', 'color-interpolation-filters'],
      ['colorProfile', 'color-profile'],
      ['colorRendering', 'color-rendering'],
      ['dominantBaseline', 'dominant-baseline'],
      ['enableBackground', 'enable-background'],
      ['fillOpacity', 'fill-opacity'],
      ['fillRule', 'fill-rule'],
      ['floodColor', 'flood-color'],
      ['floodOpacity', 'flood-opacity'],
      ['fontFamily', 'font-family'],
      ['fontSize', 'font-size'],
      ['fontSizeAdjust', 'font-size-adjust'],
      ['fontStretch', 'font-stretch'],
      ['fontStyle', 'font-style'],
      ['fontVariant', 'font-variant'],
      ['fontWeight', 'font-weight'],
      ['glyphName', 'glyph-name'],
      ['glyphOrientationHorizontal', 'glyph-orientation-horizontal'],
      ['glyphOrientationVertical', 'glyph-orientation-vertical'],
      ['horizAdvX', 'horiz-adv-x'],
      ['horizOriginX', 'horiz-origin-x'],
      ['imageRendering', 'image-rendering'],
      ['letterSpacing', 'letter-spacing'],
      ['lightingColor', 'lighting-color'],
      ['markerEnd', 'marker-end'],
      ['markerMid', 'marker-mid'],
      ['markerStart', 'marker-start'],
      ['overlinePosition', 'overline-position'],
      ['overlineThickness', 'overline-thickness'],
      ['paintOrder', 'paint-order'],
      ['panose-1', 'panose-1'],
      ['pointerEvents', 'pointer-events'],
      ['renderingIntent', 'rendering-intent'],
      ['shapeRendering', 'shape-rendering'],
      ['stopColor', 'stop-color'],
      ['stopOpacity', 'stop-opacity'],
      ['strikethroughPosition', 'strikethrough-position'],
      ['strikethroughThickness', 'strikethrough-thickness'],
      ['strokeDasharray', 'stroke-dasharray'],
      ['strokeDashoffset', 'stroke-dashoffset'],
      ['strokeLinecap', 'stroke-linecap'],
      ['strokeLinejoin', 'stroke-linejoin'],
      ['strokeMiterlimit', 'stroke-miterlimit'],
      ['strokeOpacity', 'stroke-opacity'],
      ['strokeWidth', 'stroke-width'],
      ['textAnchor', 'text-anchor'],
      ['textDecoration', 'text-decoration'],
      ['textRendering', 'text-rendering'],
      ['transformOrigin', 'transform-origin'],
      ['underlinePosition', 'underline-position'],
      ['underlineThickness', 'underline-thickness'],
      ['unicodeBidi', 'unicode-bidi'],
      ['unicodeRange', 'unicode-range'],
      ['unitsPerEm', 'units-per-em'],
      ['vAlphabetic', 'v-alphabetic'],
      ['vHanging', 'v-hanging'],
      ['vIdeographic', 'v-ideographic'],
      ['vMathematical', 'v-mathematical'],
      ['vectorEffect', 'vector-effect'],
      ['vertAdvY', 'vert-adv-y'],
      ['vertOriginX', 'vert-origin-x'],
      ['vertOriginY', 'vert-origin-y'],
      ['wordSpacing', 'word-spacing'],
      ['writingMode', 'writing-mode'],
      ['xmlnsXlink', 'xmlns:xlink'],
      ['xHeight', 'x-height'],
    ]),
    SC =
      /^[\u0000-\u001F ]*j[\r\n\t]*a[\r\n\t]*v[\r\n\t]*a[\r\n\t]*s[\r\n\t]*c[\r\n\t]*r[\r\n\t]*i[\r\n\t]*p[\r\n\t]*t[\r\n\t]*:/i;
  function Ki(e) {
    return SC.test('' + e)
      ? "javascript:throw new Error('React has blocked a javascript: URL as a security precaution.')"
      : e;
  }
  function za() {}
  var fd = null;
  function af(e) {
    return (
      (e = e.target || e.srcElement || window),
      e.correspondingUseElement && (e = e.correspondingUseElement),
      e.nodeType === 3 ? e.parentNode : e
    );
  }
  var Vo = null,
    tn = null;
  function Xh(e) {
    var t = bn(e);
    if (t && (e = t.stateNode)) {
      var a = e[St] || null;
      e: switch (((e = t.stateNode), t.type)) {
        case 'input':
          if (
            (cd(
              e,
              a.value,
              a.defaultValue,
              a.defaultValue,
              a.checked,
              a.defaultChecked,
              a.type,
              a.name
            ),
            (t = a.name),
            a.type === 'radio' && t != null)
          ) {
            for (a = e; a.parentNode; ) a = a.parentNode;
            for (
              a = a.querySelectorAll('input[name="' + Qt('' + t) + '"][type="radio"]'), t = 0;
              t < a.length;
              t++
            ) {
              var r = a[t];
              if (r !== e && r.form === e.form) {
                var o = r[St] || null;
                if (!o) throw Error(C(90));
                cd(
                  r,
                  o.value,
                  o.defaultValue,
                  o.defaultValue,
                  o.checked,
                  o.defaultChecked,
                  o.type,
                  o.name
                );
              }
            }
            for (t = 0; t < a.length; t++) (r = a[t]), r.form === e.form && Iy(r);
          }
          break e;
        case 'textarea':
          My(e, a.value, a.defaultValue);
          break e;
        case 'select':
          (t = a.value), t != null && en(e, !!a.multiple, t, !1);
      }
    }
  }
  var _c = !1;
  function Ty(e, t, a) {
    if (_c) return e(t, a);
    _c = !0;
    try {
      var r = e(t);
      return r;
    } finally {
      if (
        ((_c = !1),
        (Vo !== null || tn !== null) &&
          (Zs(), Vo && ((t = Vo), (e = tn), (tn = Vo = null), Xh(t), e)))
      )
        for (t = 0; t < e.length; t++) Xh(e[t]);
    }
  }
  function Dl(e, t) {
    var a = e.stateNode;
    if (a === null) return null;
    var r = a[St] || null;
    if (r === null) return null;
    a = r[t];
    e: switch (t) {
      case 'onClick':
      case 'onClickCapture':
      case 'onDoubleClick':
      case 'onDoubleClickCapture':
      case 'onMouseDown':
      case 'onMouseDownCapture':
      case 'onMouseMove':
      case 'onMouseMoveCapture':
      case 'onMouseUp':
      case 'onMouseUpCapture':
      case 'onMouseEnter':
        (r = !r.disabled) ||
          ((e = e.type),
          (r = !(e === 'button' || e === 'input' || e === 'select' || e === 'textarea'))),
          (e = !r);
        break e;
      default:
        e = !1;
    }
    if (e) return null;
    if (a && typeof a != 'function') throw Error(C(231, t, typeof a));
    return a;
  }
  var ja = !(
      typeof window > 'u' ||
      typeof window.document > 'u' ||
      typeof window.document.createElement > 'u'
    ),
    md = !1;
  if (ja)
    try {
      (Bo = {}),
        Object.defineProperty(Bo, 'passive', {
          get: function () {
            md = !0;
          },
        }),
        window.addEventListener('test', Bo, Bo),
        window.removeEventListener('test', Bo, Bo);
    } catch {
      md = !1;
    }
  var Bo,
    yr = null,
    rf = null,
    Zi = null;
  function Dy() {
    if (Zi) return Zi;
    var e,
      t = rf,
      a = t.length,
      r,
      o = 'value' in yr ? yr.value : yr.textContent,
      n = o.length;
    for (e = 0; e < a && t[e] === o[e]; e++);
    var l = a - e;
    for (r = 1; r <= l && t[a - r] === o[n - r]; r++);
    return (Zi = o.slice(e, 1 < r ? 1 - r : void 0));
  }
  function Ji(e) {
    var t = e.keyCode;
    return (
      'charCode' in e ? ((e = e.charCode), e === 0 && t === 13 && (e = 13)) : (e = t),
      e === 10 && (e = 13),
      32 <= e || e === 13 ? e : 0
    );
  }
  function Bi() {
    return !0;
  }
  function Wh() {
    return !1;
  }
  function Lt(e) {
    function t(a, r, o, n, l) {
      (this._reactName = a),
        (this._targetInst = o),
        (this.type = r),
        (this.nativeEvent = n),
        (this.target = l),
        (this.currentTarget = null);
      for (var i in e) e.hasOwnProperty(i) && ((a = e[i]), (this[i] = a ? a(n) : n[i]));
      return (
        (this.isDefaultPrevented = (
          n.defaultPrevented != null ? n.defaultPrevented : n.returnValue === !1
        )
          ? Bi
          : Wh),
        (this.isPropagationStopped = Wh),
        this
      );
    }
    return (
      Ce(t.prototype, {
        preventDefault: function () {
          this.defaultPrevented = !0;
          var a = this.nativeEvent;
          a &&
            (a.preventDefault
              ? a.preventDefault()
              : typeof a.returnValue != 'unknown' && (a.returnValue = !1),
            (this.isDefaultPrevented = Bi));
        },
        stopPropagation: function () {
          var a = this.nativeEvent;
          a &&
            (a.stopPropagation
              ? a.stopPropagation()
              : typeof a.cancelBubble != 'unknown' && (a.cancelBubble = !0),
            (this.isPropagationStopped = Bi));
        },
        persist: function () {},
        isPersistent: Bi,
      }),
      t
    );
  }
  var lo = {
      eventPhase: 0,
      bubbles: 0,
      cancelable: 0,
      timeStamp: function (e) {
        return e.timeStamp || Date.now();
      },
      defaultPrevented: 0,
      isTrusted: 0,
    },
    zs = Lt(lo),
    Zl = Ce({}, lo, { view: 0, detail: 0 }),
    LC = Lt(Zl),
    Ic,
    kc,
    sl,
    Fs = Ce({}, Zl, {
      screenX: 0,
      screenY: 0,
      clientX: 0,
      clientY: 0,
      pageX: 0,
      pageY: 0,
      ctrlKey: 0,
      shiftKey: 0,
      altKey: 0,
      metaKey: 0,
      getModifierState: of,
      button: 0,
      buttons: 0,
      relatedTarget: function (e) {
        return e.relatedTarget === void 0
          ? e.fromElement === e.srcElement
            ? e.toElement
            : e.fromElement
          : e.relatedTarget;
      },
      movementX: function (e) {
        return 'movementX' in e
          ? e.movementX
          : (e !== sl &&
              (sl && e.type === 'mousemove'
                ? ((Ic = e.screenX - sl.screenX), (kc = e.screenY - sl.screenY))
                : (kc = Ic = 0),
              (sl = e)),
            Ic);
      },
      movementY: function (e) {
        return 'movementY' in e ? e.movementY : kc;
      },
    }),
    Qh = Lt(Fs),
    CC = Ce({}, Fs, { dataTransfer: 0 }),
    wC = Lt(CC),
    RC = Ce({}, Zl, { relatedTarget: 0 }),
    Mc = Lt(RC),
    _C = Ce({}, lo, { animationName: 0, elapsedTime: 0, pseudoElement: 0 }),
    IC = Lt(_C),
    kC = Ce({}, lo, {
      clipboardData: function (e) {
        return 'clipboardData' in e ? e.clipboardData : window.clipboardData;
      },
    }),
    MC = Lt(kC),
    EC = Ce({}, lo, { data: 0 }),
    Kh = Lt(EC),
    AC = {
      Esc: 'Escape',
      Spacebar: ' ',
      Left: 'ArrowLeft',
      Up: 'ArrowUp',
      Right: 'ArrowRight',
      Down: 'ArrowDown',
      Del: 'Delete',
      Win: 'OS',
      Menu: 'ContextMenu',
      Apps: 'ContextMenu',
      Scroll: 'ScrollLock',
      MozPrintableKey: 'Unidentified',
    },
    TC = {
      8: 'Backspace',
      9: 'Tab',
      12: 'Clear',
      13: 'Enter',
      16: 'Shift',
      17: 'Control',
      18: 'Alt',
      19: 'Pause',
      20: 'CapsLock',
      27: 'Escape',
      32: ' ',
      33: 'PageUp',
      34: 'PageDown',
      35: 'End',
      36: 'Home',
      37: 'ArrowLeft',
      38: 'ArrowUp',
      39: 'ArrowRight',
      40: 'ArrowDown',
      45: 'Insert',
      46: 'Delete',
      112: 'F1',
      113: 'F2',
      114: 'F3',
      115: 'F4',
      116: 'F5',
      117: 'F6',
      118: 'F7',
      119: 'F8',
      120: 'F9',
      121: 'F10',
      122: 'F11',
      123: 'F12',
      144: 'NumLock',
      145: 'ScrollLock',
      224: 'Meta',
    },
    DC = { Alt: 'altKey', Control: 'ctrlKey', Meta: 'metaKey', Shift: 'shiftKey' };
  function OC(e) {
    var t = this.nativeEvent;
    return t.getModifierState ? t.getModifierState(e) : (e = DC[e]) ? !!t[e] : !1;
  }
  function of() {
    return OC;
  }
  var PC = Ce({}, Zl, {
      key: function (e) {
        if (e.key) {
          var t = AC[e.key] || e.key;
          if (t !== 'Unidentified') return t;
        }
        return e.type === 'keypress'
          ? ((e = Ji(e)), e === 13 ? 'Enter' : String.fromCharCode(e))
          : e.type === 'keydown' || e.type === 'keyup'
            ? TC[e.keyCode] || 'Unidentified'
            : '';
      },
      code: 0,
      location: 0,
      ctrlKey: 0,
      shiftKey: 0,
      altKey: 0,
      metaKey: 0,
      repeat: 0,
      locale: 0,
      getModifierState: of,
      charCode: function (e) {
        return e.type === 'keypress' ? Ji(e) : 0;
      },
      keyCode: function (e) {
        return e.type === 'keydown' || e.type === 'keyup' ? e.keyCode : 0;
      },
      which: function (e) {
        return e.type === 'keypress'
          ? Ji(e)
          : e.type === 'keydown' || e.type === 'keyup'
            ? e.keyCode
            : 0;
      },
    }),
    BC = Lt(PC),
    UC = Ce({}, Fs, {
      pointerId: 0,
      width: 0,
      height: 0,
      pressure: 0,
      tangentialPressure: 0,
      tiltX: 0,
      tiltY: 0,
      twist: 0,
      pointerType: 0,
      isPrimary: 0,
    }),
    Zh = Lt(UC),
    NC = Ce({}, Zl, {
      touches: 0,
      targetTouches: 0,
      changedTouches: 0,
      altKey: 0,
      metaKey: 0,
      ctrlKey: 0,
      shiftKey: 0,
      getModifierState: of,
    }),
    HC = Lt(NC),
    zC = Ce({}, lo, { propertyName: 0, elapsedTime: 0, pseudoElement: 0 }),
    FC = Lt(zC),
    qC = Ce({}, Fs, {
      deltaX: function (e) {
        return 'deltaX' in e ? e.deltaX : 'wheelDeltaX' in e ? -e.wheelDeltaX : 0;
      },
      deltaY: function (e) {
        return 'deltaY' in e
          ? e.deltaY
          : 'wheelDeltaY' in e
            ? -e.wheelDeltaY
            : 'wheelDelta' in e
              ? -e.wheelDelta
              : 0;
      },
      deltaZ: 0,
      deltaMode: 0,
    }),
    GC = Lt(qC),
    VC = Ce({}, lo, { newState: 0, oldState: 0 }),
    jC = Lt(VC),
    YC = [9, 13, 27, 32],
    nf = ja && 'CompositionEvent' in window,
    bl = null;
  ja && 'documentMode' in document && (bl = document.documentMode);
  var XC = ja && 'TextEvent' in window && !bl,
    Oy = ja && (!nf || (bl && 8 < bl && 11 >= bl)),
    Jh = ' ',
    $h = !1;
  function Py(e, t) {
    switch (e) {
      case 'keyup':
        return YC.indexOf(t.keyCode) !== -1;
      case 'keydown':
        return t.keyCode !== 229;
      case 'keypress':
      case 'mousedown':
      case 'focusout':
        return !0;
      default:
        return !1;
    }
  }
  function By(e) {
    return (e = e.detail), typeof e == 'object' && 'data' in e ? e.data : null;
  }
  var jo = !1;
  function WC(e, t) {
    switch (e) {
      case 'compositionend':
        return By(t);
      case 'keypress':
        return t.which !== 32 ? null : (($h = !0), Jh);
      case 'textInput':
        return (e = t.data), e === Jh && $h ? null : e;
      default:
        return null;
    }
  }
  function QC(e, t) {
    if (jo)
      return e === 'compositionend' || (!nf && Py(e, t))
        ? ((e = Dy()), (Zi = rf = yr = null), (jo = !1), e)
        : null;
    switch (e) {
      case 'paste':
        return null;
      case 'keypress':
        if (!(t.ctrlKey || t.altKey || t.metaKey) || (t.ctrlKey && t.altKey)) {
          if (t.char && 1 < t.char.length) return t.char;
          if (t.which) return String.fromCharCode(t.which);
        }
        return null;
      case 'compositionend':
        return Oy && t.locale !== 'ko' ? null : t.data;
      default:
        return null;
    }
  }
  var KC = {
    color: !0,
    date: !0,
    datetime: !0,
    'datetime-local': !0,
    email: !0,
    month: !0,
    number: !0,
    password: !0,
    range: !0,
    search: !0,
    tel: !0,
    text: !0,
    time: !0,
    url: !0,
    week: !0,
  };
  function eg(e) {
    var t = e && e.nodeName && e.nodeName.toLowerCase();
    return t === 'input' ? !!KC[e.type] : t === 'textarea';
  }
  function Uy(e, t, a, r) {
    Vo ? (tn ? tn.push(r) : (tn = [r])) : (Vo = r),
      (t = Ts(t, 'onChange')),
      0 < t.length &&
        ((a = new zs('onChange', 'change', null, a, r)), e.push({ event: a, listeners: t }));
  }
  var xl = null,
    Ol = null;
  function ZC(e) {
    Tv(e, 0);
  }
  function qs(e) {
    var t = gl(e);
    if (Iy(t)) return e;
  }
  function tg(e, t) {
    if (e === 'change') return t;
  }
  var Ny = !1;
  ja &&
    (ja
      ? ((Ni = 'oninput' in document),
        Ni ||
          ((Ec = document.createElement('div')),
          Ec.setAttribute('oninput', 'return;'),
          (Ni = typeof Ec.oninput == 'function')),
        (Ui = Ni))
      : (Ui = !1),
    (Ny = Ui && (!document.documentMode || 9 < document.documentMode)));
  var Ui, Ni, Ec;
  function ag() {
    xl && (xl.detachEvent('onpropertychange', Hy), (Ol = xl = null));
  }
  function Hy(e) {
    if (e.propertyName === 'value' && qs(Ol)) {
      var t = [];
      Uy(t, Ol, e, af(e)), Ty(ZC, t);
    }
  }
  function JC(e, t, a) {
    e === 'focusin'
      ? (ag(), (xl = t), (Ol = a), xl.attachEvent('onpropertychange', Hy))
      : e === 'focusout' && ag();
  }
  function $C(e) {
    if (e === 'selectionchange' || e === 'keyup' || e === 'keydown') return qs(Ol);
  }
  function ew(e, t) {
    if (e === 'click') return qs(t);
  }
  function tw(e, t) {
    if (e === 'input' || e === 'change') return qs(t);
  }
  function aw(e, t) {
    return (e === t && (e !== 0 || 1 / e === 1 / t)) || (e !== e && t !== t);
  }
  var Bt = typeof Object.is == 'function' ? Object.is : aw;
  function Pl(e, t) {
    if (Bt(e, t)) return !0;
    if (typeof e != 'object' || e === null || typeof t != 'object' || t === null) return !1;
    var a = Object.keys(e),
      r = Object.keys(t);
    if (a.length !== r.length) return !1;
    for (r = 0; r < a.length; r++) {
      var o = a[r];
      if (!id.call(t, o) || !Bt(e[o], t[o])) return !1;
    }
    return !0;
  }
  function rg(e) {
    for (; e && e.firstChild; ) e = e.firstChild;
    return e;
  }
  function og(e, t) {
    var a = rg(e);
    e = 0;
    for (var r; a; ) {
      if (a.nodeType === 3) {
        if (((r = e + a.textContent.length), e <= t && r >= t)) return { node: a, offset: t - e };
        e = r;
      }
      e: {
        for (; a; ) {
          if (a.nextSibling) {
            a = a.nextSibling;
            break e;
          }
          a = a.parentNode;
        }
        a = void 0;
      }
      a = rg(a);
    }
  }
  function zy(e, t) {
    return e && t
      ? e === t
        ? !0
        : e && e.nodeType === 3
          ? !1
          : t && t.nodeType === 3
            ? zy(e, t.parentNode)
            : 'contains' in e
              ? e.contains(t)
              : e.compareDocumentPosition
                ? !!(e.compareDocumentPosition(t) & 16)
                : !1
      : !1;
  }
  function Fy(e) {
    e =
      e != null && e.ownerDocument != null && e.ownerDocument.defaultView != null
        ? e.ownerDocument.defaultView
        : window;
    for (var t = ps(e.document); t instanceof e.HTMLIFrameElement; ) {
      try {
        var a = typeof t.contentWindow.location.href == 'string';
      } catch {
        a = !1;
      }
      if (a) e = t.contentWindow;
      else break;
      t = ps(e.document);
    }
    return t;
  }
  function lf(e) {
    var t = e && e.nodeName && e.nodeName.toLowerCase();
    return (
      t &&
      ((t === 'input' &&
        (e.type === 'text' ||
          e.type === 'search' ||
          e.type === 'tel' ||
          e.type === 'url' ||
          e.type === 'password')) ||
        t === 'textarea' ||
        e.contentEditable === 'true')
    );
  }
  var rw = ja && 'documentMode' in document && 11 >= document.documentMode,
    Yo = null,
    pd = null,
    Sl = null,
    hd = !1;
  function ng(e, t, a) {
    var r = a.window === a ? a.document : a.nodeType === 9 ? a : a.ownerDocument;
    hd ||
      Yo == null ||
      Yo !== ps(r) ||
      ((r = Yo),
      'selectionStart' in r && lf(r)
        ? (r = { start: r.selectionStart, end: r.selectionEnd })
        : ((r = ((r.ownerDocument && r.ownerDocument.defaultView) || window).getSelection()),
          (r = {
            anchorNode: r.anchorNode,
            anchorOffset: r.anchorOffset,
            focusNode: r.focusNode,
            focusOffset: r.focusOffset,
          })),
      (Sl && Pl(Sl, r)) ||
        ((Sl = r),
        (r = Ts(pd, 'onSelect')),
        0 < r.length &&
          ((t = new zs('onSelect', 'select', null, t, a)),
          e.push({ event: t, listeners: r }),
          (t.target = Yo))));
  }
  function Vr(e, t) {
    var a = {};
    return (
      (a[e.toLowerCase()] = t.toLowerCase()),
      (a['Webkit' + e] = 'webkit' + t),
      (a['Moz' + e] = 'moz' + t),
      a
    );
  }
  var Xo = {
      animationend: Vr('Animation', 'AnimationEnd'),
      animationiteration: Vr('Animation', 'AnimationIteration'),
      animationstart: Vr('Animation', 'AnimationStart'),
      transitionrun: Vr('Transition', 'TransitionRun'),
      transitionstart: Vr('Transition', 'TransitionStart'),
      transitioncancel: Vr('Transition', 'TransitionCancel'),
      transitionend: Vr('Transition', 'TransitionEnd'),
    },
    Ac = {},
    qy = {};
  ja &&
    ((qy = document.createElement('div').style),
    'AnimationEvent' in window ||
      (delete Xo.animationend.animation,
      delete Xo.animationiteration.animation,
      delete Xo.animationstart.animation),
    'TransitionEvent' in window || delete Xo.transitionend.transition);
  function io(e) {
    if (Ac[e]) return Ac[e];
    if (!Xo[e]) return e;
    var t = Xo[e],
      a;
    for (a in t) if (t.hasOwnProperty(a) && a in qy) return (Ac[e] = t[a]);
    return e;
  }
  var Gy = io('animationend'),
    Vy = io('animationiteration'),
    jy = io('animationstart'),
    ow = io('transitionrun'),
    nw = io('transitionstart'),
    lw = io('transitioncancel'),
    Yy = io('transitionend'),
    Xy = new Map(),
    gd =
      'abort auxClick beforeToggle cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel'.split(
        ' '
      );
  gd.push('scrollEnd');
  function ia(e, t) {
    Xy.set(e, t), no(t, [e]);
  }
  var hs =
      typeof reportError == 'function'
        ? reportError
        : function (e) {
            if (typeof window == 'object' && typeof window.ErrorEvent == 'function') {
              var t = new window.ErrorEvent('error', {
                bubbles: !0,
                cancelable: !0,
                message:
                  typeof e == 'object' && e !== null && typeof e.message == 'string'
                    ? String(e.message)
                    : String(e),
                error: e,
              });
              if (!window.dispatchEvent(t)) return;
            } else if (typeof process == 'object' && typeof process.emit == 'function') {
              process.emit('uncaughtException', e);
              return;
            }
            console.error(e);
          },
    jt = [],
    Wo = 0,
    sf = 0;
  function Gs() {
    for (var e = Wo, t = (sf = Wo = 0); t < e; ) {
      var a = jt[t];
      jt[t++] = null;
      var r = jt[t];
      jt[t++] = null;
      var o = jt[t];
      jt[t++] = null;
      var n = jt[t];
      if (((jt[t++] = null), r !== null && o !== null)) {
        var l = r.pending;
        l === null ? (o.next = o) : ((o.next = l.next), (l.next = o)), (r.pending = o);
      }
      n !== 0 && Wy(a, o, n);
    }
  }
  function Vs(e, t, a, r) {
    (jt[Wo++] = e),
      (jt[Wo++] = t),
      (jt[Wo++] = a),
      (jt[Wo++] = r),
      (sf |= r),
      (e.lanes |= r),
      (e = e.alternate),
      e !== null && (e.lanes |= r);
  }
  function uf(e, t, a, r) {
    return Vs(e, t, a, r), gs(e);
  }
  function so(e, t) {
    return Vs(e, null, null, t), gs(e);
  }
  function Wy(e, t, a) {
    e.lanes |= a;
    var r = e.alternate;
    r !== null && (r.lanes |= a);
    for (var o = !1, n = e.return; n !== null; )
      (n.childLanes |= a),
        (r = n.alternate),
        r !== null && (r.childLanes |= a),
        n.tag === 22 && ((e = n.stateNode), e === null || e._visibility & 1 || (o = !0)),
        (e = n),
        (n = n.return);
    return e.tag === 3
      ? ((n = e.stateNode),
        o &&
          t !== null &&
          ((o = 31 - Ot(a)),
          (e = n.hiddenUpdates),
          (r = e[o]),
          r === null ? (e[o] = [t]) : r.push(t),
          (t.lane = a | 536870912)),
        n)
      : null;
  }
  function gs(e) {
    if (50 < El) throw ((El = 0), (Ud = null), Error(C(185)));
    for (var t = e.return; t !== null; ) (e = t), (t = e.return);
    return e.tag === 3 ? e.stateNode : null;
  }
  var Qo = {};
  function iw(e, t, a, r) {
    (this.tag = e),
      (this.key = a),
      (this.sibling =
        this.child =
        this.return =
        this.stateNode =
        this.type =
        this.elementType =
          null),
      (this.index = 0),
      (this.refCleanup = this.ref = null),
      (this.pendingProps = t),
      (this.dependencies = this.memoizedState = this.updateQueue = this.memoizedProps = null),
      (this.mode = r),
      (this.subtreeFlags = this.flags = 0),
      (this.deletions = null),
      (this.childLanes = this.lanes = 0),
      (this.alternate = null);
  }
  function Et(e, t, a, r) {
    return new iw(e, t, a, r);
  }
  function cf(e) {
    return (e = e.prototype), !(!e || !e.isReactComponent);
  }
  function qa(e, t) {
    var a = e.alternate;
    return (
      a === null
        ? ((a = Et(e.tag, t, e.key, e.mode)),
          (a.elementType = e.elementType),
          (a.type = e.type),
          (a.stateNode = e.stateNode),
          (a.alternate = e),
          (e.alternate = a))
        : ((a.pendingProps = t),
          (a.type = e.type),
          (a.flags = 0),
          (a.subtreeFlags = 0),
          (a.deletions = null)),
      (a.flags = e.flags & 65011712),
      (a.childLanes = e.childLanes),
      (a.lanes = e.lanes),
      (a.child = e.child),
      (a.memoizedProps = e.memoizedProps),
      (a.memoizedState = e.memoizedState),
      (a.updateQueue = e.updateQueue),
      (t = e.dependencies),
      (a.dependencies = t === null ? null : { lanes: t.lanes, firstContext: t.firstContext }),
      (a.sibling = e.sibling),
      (a.index = e.index),
      (a.ref = e.ref),
      (a.refCleanup = e.refCleanup),
      a
    );
  }
  function Qy(e, t) {
    e.flags &= 65011714;
    var a = e.alternate;
    return (
      a === null
        ? ((e.childLanes = 0),
          (e.lanes = t),
          (e.child = null),
          (e.subtreeFlags = 0),
          (e.memoizedProps = null),
          (e.memoizedState = null),
          (e.updateQueue = null),
          (e.dependencies = null),
          (e.stateNode = null))
        : ((e.childLanes = a.childLanes),
          (e.lanes = a.lanes),
          (e.child = a.child),
          (e.subtreeFlags = 0),
          (e.deletions = null),
          (e.memoizedProps = a.memoizedProps),
          (e.memoizedState = a.memoizedState),
          (e.updateQueue = a.updateQueue),
          (e.type = a.type),
          (t = a.dependencies),
          (e.dependencies = t === null ? null : { lanes: t.lanes, firstContext: t.firstContext })),
      e
    );
  }
  function $i(e, t, a, r, o, n) {
    var l = 0;
    if (((r = e), typeof e == 'function')) cf(e) && (l = 1);
    else if (typeof e == 'string')
      l = cR(e, a, ba.current) ? 26 : e === 'html' || e === 'head' || e === 'body' ? 27 : 5;
    else
      e: switch (e) {
        case rd:
          return (e = Et(31, a, t, o)), (e.elementType = rd), (e.lanes = n), e;
        case Fo:
          return Kr(a.children, o, n, t);
        case py:
          (l = 8), (o |= 24);
          break;
        case ed:
          return (e = Et(12, a, t, o | 2)), (e.elementType = ed), (e.lanes = n), e;
        case td:
          return (e = Et(13, a, t, o)), (e.elementType = td), (e.lanes = n), e;
        case ad:
          return (e = Et(19, a, t, o)), (e.elementType = ad), (e.lanes = n), e;
        default:
          if (typeof e == 'object' && e !== null)
            switch (e.$$typeof) {
              case Ha:
                l = 10;
                break e;
              case hy:
                l = 9;
                break e;
              case Qd:
                l = 11;
                break e;
              case Kd:
                l = 14;
                break e;
              case cr:
                (l = 16), (r = null);
                break e;
            }
          (l = 29), (a = Error(C(130, e === null ? 'null' : typeof e, ''))), (r = null);
      }
    return (t = Et(l, a, t, o)), (t.elementType = e), (t.type = r), (t.lanes = n), t;
  }
  function Kr(e, t, a, r) {
    return (e = Et(7, e, r, t)), (e.lanes = a), e;
  }
  function Tc(e, t, a) {
    return (e = Et(6, e, null, t)), (e.lanes = a), e;
  }
  function Ky(e) {
    var t = Et(18, null, null, 0);
    return (t.stateNode = e), t;
  }
  function Dc(e, t, a) {
    return (
      (t = Et(4, e.children !== null ? e.children : [], e.key, t)),
      (t.lanes = a),
      (t.stateNode = {
        containerInfo: e.containerInfo,
        pendingChildren: null,
        implementation: e.implementation,
      }),
      t
    );
  }
  var lg = new WeakMap();
  function Kt(e, t) {
    if (typeof e == 'object' && e !== null) {
      var a = lg.get(e);
      return a !== void 0 ? a : ((t = { value: e, source: t, stack: Fh(t) }), lg.set(e, t), t);
    }
    return { value: e, source: t, stack: Fh(t) };
  }
  var Ko = [],
    Zo = 0,
    ys = null,
    Bl = 0,
    Xt = [],
    Wt = 0,
    Er = null,
    ga = 1,
    ya = '';
  function Ua(e, t) {
    (Ko[Zo++] = Bl), (Ko[Zo++] = ys), (ys = e), (Bl = t);
  }
  function Zy(e, t, a) {
    (Xt[Wt++] = ga), (Xt[Wt++] = ya), (Xt[Wt++] = Er), (Er = e);
    var r = ga;
    e = ya;
    var o = 32 - Ot(r) - 1;
    (r &= ~(1 << o)), (a += 1);
    var n = 32 - Ot(t) + o;
    if (30 < n) {
      var l = o - (o % 5);
      (n = (r & ((1 << l) - 1)).toString(32)),
        (r >>= l),
        (o -= l),
        (ga = (1 << (32 - Ot(t) + o)) | (a << o) | r),
        (ya = n + e);
    } else (ga = (1 << n) | (a << o) | r), (ya = e);
  }
  function df(e) {
    e.return !== null && (Ua(e, 1), Zy(e, 1, 0));
  }
  function ff(e) {
    for (; e === ys; ) (ys = Ko[--Zo]), (Ko[Zo] = null), (Bl = Ko[--Zo]), (Ko[Zo] = null);
    for (; e === Er; )
      (Er = Xt[--Wt]),
        (Xt[Wt] = null),
        (ya = Xt[--Wt]),
        (Xt[Wt] = null),
        (ga = Xt[--Wt]),
        (Xt[Wt] = null);
  }
  function Jy(e, t) {
    (Xt[Wt++] = ga), (Xt[Wt++] = ya), (Xt[Wt++] = Er), (ga = t.id), (ya = t.overflow), (Er = e);
  }
  var tt = null,
    Le = null,
    J = !1,
    Lr = null,
    Zt = !1,
    yd = Error(C(519));
  function Ar(e) {
    var t = Error(
      C(418, 1 < arguments.length && arguments[1] !== void 0 && arguments[1] ? 'text' : 'HTML', '')
    );
    throw (Ul(Kt(t, e)), yd);
  }
  function ig(e) {
    var t = e.stateNode,
      a = e.type,
      r = e.memoizedProps;
    switch (((t[et] = e), (t[St] = r), a)) {
      case 'dialog':
        W('cancel', t), W('close', t);
        break;
      case 'iframe':
      case 'object':
      case 'embed':
        W('load', t);
        break;
      case 'video':
      case 'audio':
        for (a = 0; a < Fl.length; a++) W(Fl[a], t);
        break;
      case 'source':
        W('error', t);
        break;
      case 'img':
      case 'image':
      case 'link':
        W('error', t), W('load', t);
        break;
      case 'details':
        W('toggle', t);
        break;
      case 'input':
        W('invalid', t),
          ky(t, r.value, r.defaultValue, r.checked, r.defaultChecked, r.type, r.name, !0);
        break;
      case 'select':
        W('invalid', t);
        break;
      case 'textarea':
        W('invalid', t), Ey(t, r.value, r.defaultValue, r.children);
    }
    (a = r.children),
      (typeof a != 'string' && typeof a != 'number' && typeof a != 'bigint') ||
      t.textContent === '' + a ||
      r.suppressHydrationWarning === !0 ||
      Ov(t.textContent, a)
        ? (r.popover != null && (W('beforetoggle', t), W('toggle', t)),
          r.onScroll != null && W('scroll', t),
          r.onScrollEnd != null && W('scrollend', t),
          r.onClick != null && (t.onclick = za),
          (t = !0))
        : (t = !1),
      t || Ar(e, !0);
  }
  function sg(e) {
    for (tt = e.return; tt; )
      switch (tt.tag) {
        case 5:
        case 31:
        case 13:
          Zt = !1;
          return;
        case 27:
        case 3:
          Zt = !0;
          return;
        default:
          tt = tt.return;
      }
  }
  function Uo(e) {
    if (e !== tt) return !1;
    if (!J) return sg(e), (J = !0), !1;
    var t = e.tag,
      a;
    if (
      ((a = t !== 3 && t !== 27) &&
        ((a = t === 5) &&
          ((a = e.type), (a = !(a !== 'form' && a !== 'button') || qd(e.type, e.memoizedProps))),
        (a = !a)),
      a && Le && Ar(e),
      sg(e),
      t === 13)
    ) {
      if (((e = e.memoizedState), (e = e !== null ? e.dehydrated : null), !e)) throw Error(C(317));
      Le = Kg(e);
    } else if (t === 31) {
      if (((e = e.memoizedState), (e = e !== null ? e.dehydrated : null), !e)) throw Error(C(317));
      Le = Kg(e);
    } else
      t === 27
        ? ((t = Le), Pr(e.type) ? ((e = Yd), (Yd = null), (Le = e)) : (Le = t))
        : (Le = tt ? $t(e.stateNode.nextSibling) : null);
    return !0;
  }
  function eo() {
    (Le = tt = null), (J = !1);
  }
  function Oc() {
    var e = Lr;
    return e !== null && (bt === null ? (bt = e) : bt.push.apply(bt, e), (Lr = null)), e;
  }
  function Ul(e) {
    Lr === null ? (Lr = [e]) : Lr.push(e);
  }
  var vd = xa(null),
    uo = null,
    Fa = null;
  function fr(e, t, a) {
    ye(vd, t._currentValue), (t._currentValue = a);
  }
  function Ga(e) {
    (e._currentValue = vd.current), Ye(vd);
  }
  function bd(e, t, a) {
    for (; e !== null; ) {
      var r = e.alternate;
      if (
        ((e.childLanes & t) !== t
          ? ((e.childLanes |= t), r !== null && (r.childLanes |= t))
          : r !== null && (r.childLanes & t) !== t && (r.childLanes |= t),
        e === a)
      )
        break;
      e = e.return;
    }
  }
  function xd(e, t, a, r) {
    var o = e.child;
    for (o !== null && (o.return = e); o !== null; ) {
      var n = o.dependencies;
      if (n !== null) {
        var l = o.child;
        n = n.firstContext;
        e: for (; n !== null; ) {
          var i = n;
          n = o;
          for (var s = 0; s < t.length; s++)
            if (i.context === t[s]) {
              (n.lanes |= a),
                (i = n.alternate),
                i !== null && (i.lanes |= a),
                bd(n.return, a, e),
                r || (l = null);
              break e;
            }
          n = i.next;
        }
      } else if (o.tag === 18) {
        if (((l = o.return), l === null)) throw Error(C(341));
        (l.lanes |= a), (n = l.alternate), n !== null && (n.lanes |= a), bd(l, a, e), (l = null);
      } else l = o.child;
      if (l !== null) l.return = o;
      else
        for (l = o; l !== null; ) {
          if (l === e) {
            l = null;
            break;
          }
          if (((o = l.sibling), o !== null)) {
            (o.return = l.return), (l = o);
            break;
          }
          l = l.return;
        }
      o = l;
    }
  }
  function xn(e, t, a, r) {
    e = null;
    for (var o = t, n = !1; o !== null; ) {
      if (!n) {
        if ((o.flags & 524288) !== 0) n = !0;
        else if ((o.flags & 262144) !== 0) break;
      }
      if (o.tag === 10) {
        var l = o.alternate;
        if (l === null) throw Error(C(387));
        if (((l = l.memoizedProps), l !== null)) {
          var i = o.type;
          Bt(o.pendingProps.value, l.value) || (e !== null ? e.push(i) : (e = [i]));
        }
      } else if (o === cs.current) {
        if (((l = o.alternate), l === null)) throw Error(C(387));
        l.memoizedState.memoizedState !== o.memoizedState.memoizedState &&
          (e !== null ? e.push(Gl) : (e = [Gl]));
      }
      o = o.return;
    }
    e !== null && xd(t, e, a, r), (t.flags |= 262144);
  }
  function vs(e) {
    for (e = e.firstContext; e !== null; ) {
      if (!Bt(e.context._currentValue, e.memoizedValue)) return !0;
      e = e.next;
    }
    return !1;
  }
  function to(e) {
    (uo = e), (Fa = null), (e = e.dependencies), e !== null && (e.firstContext = null);
  }
  function at(e) {
    return $y(uo, e);
  }
  function Hi(e, t) {
    return uo === null && to(e), $y(e, t);
  }
  function $y(e, t) {
    var a = t._currentValue;
    if (((t = { context: t, memoizedValue: a, next: null }), Fa === null)) {
      if (e === null) throw Error(C(308));
      (Fa = t), (e.dependencies = { lanes: 0, firstContext: t }), (e.flags |= 524288);
    } else Fa = Fa.next = t;
    return a;
  }
  var sw =
      typeof AbortController < 'u'
        ? AbortController
        : function () {
            var e = [],
              t = (this.signal = {
                aborted: !1,
                addEventListener: function (a, r) {
                  e.push(r);
                },
              });
            this.abort = function () {
              (t.aborted = !0),
                e.forEach(function (a) {
                  return a();
                });
            };
          },
    uw = ze.unstable_scheduleCallback,
    cw = ze.unstable_NormalPriority,
    Pe = {
      $$typeof: Ha,
      Consumer: null,
      Provider: null,
      _currentValue: null,
      _currentValue2: null,
      _threadCount: 0,
    };
  function mf() {
    return { controller: new sw(), data: new Map(), refCount: 0 };
  }
  function Jl(e) {
    e.refCount--,
      e.refCount === 0 &&
        uw(cw, function () {
          e.controller.abort();
        });
  }
  var Ll = null,
    Sd = 0,
    cn = 0,
    an = null;
  function dw(e, t) {
    if (Ll === null) {
      var a = (Ll = []);
      (Sd = 0),
        (cn = Nf()),
        (an = {
          status: 'pending',
          value: void 0,
          then: function (r) {
            a.push(r);
          },
        });
    }
    return Sd++, t.then(ug, ug), t;
  }
  function ug() {
    if (--Sd === 0 && Ll !== null) {
      an !== null && (an.status = 'fulfilled');
      var e = Ll;
      (Ll = null), (cn = 0), (an = null);
      for (var t = 0; t < e.length; t++) (0, e[t])();
    }
  }
  function fw(e, t) {
    var a = [],
      r = {
        status: 'pending',
        value: null,
        reason: null,
        then: function (o) {
          a.push(o);
        },
      };
    return (
      e.then(
        function () {
          (r.status = 'fulfilled'), (r.value = t);
          for (var o = 0; o < a.length; o++) (0, a[o])(t);
        },
        function (o) {
          for (r.status = 'rejected', r.reason = o, o = 0; o < a.length; o++) (0, a[o])(void 0);
        }
      ),
      r
    );
  }
  var cg = N.S;
  N.S = function (e, t) {
    (mv = Tt()),
      typeof t == 'object' && t !== null && typeof t.then == 'function' && dw(e, t),
      cg !== null && cg(e, t);
  };
  var Zr = xa(null);
  function pf() {
    var e = Zr.current;
    return e !== null ? e : he.pooledCache;
  }
  function es(e, t) {
    t === null ? ye(Zr, Zr.current) : ye(Zr, t.pool);
  }
  function e0() {
    var e = pf();
    return e === null ? null : { parent: Pe._currentValue, pool: e };
  }
  var Sn = Error(C(460)),
    hf = Error(C(474)),
    js = Error(C(542)),
    bs = { then: function () {} };
  function dg(e) {
    return (e = e.status), e === 'fulfilled' || e === 'rejected';
  }
  function t0(e, t, a) {
    switch (
      ((a = e[a]), a === void 0 ? e.push(t) : a !== t && (t.then(za, za), (t = a)), t.status)
    ) {
      case 'fulfilled':
        return t.value;
      case 'rejected':
        throw ((e = t.reason), mg(e), e);
      default:
        if (typeof t.status == 'string') t.then(za, za);
        else {
          if (((e = he), e !== null && 100 < e.shellSuspendCounter)) throw Error(C(482));
          (e = t),
            (e.status = 'pending'),
            e.then(
              function (r) {
                if (t.status === 'pending') {
                  var o = t;
                  (o.status = 'fulfilled'), (o.value = r);
                }
              },
              function (r) {
                if (t.status === 'pending') {
                  var o = t;
                  (o.status = 'rejected'), (o.reason = r);
                }
              }
            );
        }
        switch (t.status) {
          case 'fulfilled':
            return t.value;
          case 'rejected':
            throw ((e = t.reason), mg(e), e);
        }
        throw ((Jr = t), Sn);
    }
  }
  function Xr(e) {
    try {
      var t = e._init;
      return t(e._payload);
    } catch (a) {
      throw a !== null && typeof a == 'object' && typeof a.then == 'function' ? ((Jr = a), Sn) : a;
    }
  }
  var Jr = null;
  function fg() {
    if (Jr === null) throw Error(C(459));
    var e = Jr;
    return (Jr = null), e;
  }
  function mg(e) {
    if (e === Sn || e === js) throw Error(C(483));
  }
  var rn = null,
    Nl = 0;
  function zi(e) {
    var t = Nl;
    return (Nl += 1), rn === null && (rn = []), t0(rn, e, t);
  }
  function ul(e, t) {
    (t = t.props.ref), (e.ref = t !== void 0 ? t : null);
  }
  function Fi(e, t) {
    throw t.$$typeof === J1
      ? Error(C(525))
      : ((e = Object.prototype.toString.call(t)),
        Error(
          C(
            31,
            e === '[object Object]' ? 'object with keys {' + Object.keys(t).join(', ') + '}' : e
          )
        ));
  }
  function a0(e) {
    function t(m, p) {
      if (e) {
        var g = m.deletions;
        g === null ? ((m.deletions = [p]), (m.flags |= 16)) : g.push(p);
      }
    }
    function a(m, p) {
      if (!e) return null;
      for (; p !== null; ) t(m, p), (p = p.sibling);
      return null;
    }
    function r(m) {
      for (var p = new Map(); m !== null; )
        m.key !== null ? p.set(m.key, m) : p.set(m.index, m), (m = m.sibling);
      return p;
    }
    function o(m, p) {
      return (m = qa(m, p)), (m.index = 0), (m.sibling = null), m;
    }
    function n(m, p, g) {
      return (
        (m.index = g),
        e
          ? ((g = m.alternate),
            g !== null
              ? ((g = g.index), g < p ? ((m.flags |= 67108866), p) : g)
              : ((m.flags |= 67108866), p))
          : ((m.flags |= 1048576), p)
      );
    }
    function l(m) {
      return e && m.alternate === null && (m.flags |= 67108866), m;
    }
    function i(m, p, g, b) {
      return p === null || p.tag !== 6
        ? ((p = Tc(g, m.mode, b)), (p.return = m), p)
        : ((p = o(p, g)), (p.return = m), p);
    }
    function s(m, p, g, b) {
      var R = g.type;
      return R === Fo
        ? d(m, p, g.props.children, b, g.key)
        : p !== null &&
            (p.elementType === R ||
              (typeof R == 'object' && R !== null && R.$$typeof === cr && Xr(R) === p.type))
          ? ((p = o(p, g.props)), ul(p, g), (p.return = m), p)
          : ((p = $i(g.type, g.key, g.props, null, m.mode, b)), ul(p, g), (p.return = m), p);
    }
    function u(m, p, g, b) {
      return p === null ||
        p.tag !== 4 ||
        p.stateNode.containerInfo !== g.containerInfo ||
        p.stateNode.implementation !== g.implementation
        ? ((p = Dc(g, m.mode, b)), (p.return = m), p)
        : ((p = o(p, g.children || [])), (p.return = m), p);
    }
    function d(m, p, g, b, R) {
      return p === null || p.tag !== 7
        ? ((p = Kr(g, m.mode, b, R)), (p.return = m), p)
        : ((p = o(p, g)), (p.return = m), p);
    }
    function c(m, p, g) {
      if ((typeof p == 'string' && p !== '') || typeof p == 'number' || typeof p == 'bigint')
        return (p = Tc('' + p, m.mode, g)), (p.return = m), p;
      if (typeof p == 'object' && p !== null) {
        switch (p.$$typeof) {
          case Ai:
            return (g = $i(p.type, p.key, p.props, null, m.mode, g)), ul(g, p), (g.return = m), g;
          case pl:
            return (p = Dc(p, m.mode, g)), (p.return = m), p;
          case cr:
            return (p = Xr(p)), c(m, p, g);
        }
        if (hl(p) || il(p)) return (p = Kr(p, m.mode, g, null)), (p.return = m), p;
        if (typeof p.then == 'function') return c(m, zi(p), g);
        if (p.$$typeof === Ha) return c(m, Hi(m, p), g);
        Fi(m, p);
      }
      return null;
    }
    function f(m, p, g, b) {
      var R = p !== null ? p.key : null;
      if ((typeof g == 'string' && g !== '') || typeof g == 'number' || typeof g == 'bigint')
        return R !== null ? null : i(m, p, '' + g, b);
      if (typeof g == 'object' && g !== null) {
        switch (g.$$typeof) {
          case Ai:
            return g.key === R ? s(m, p, g, b) : null;
          case pl:
            return g.key === R ? u(m, p, g, b) : null;
          case cr:
            return (g = Xr(g)), f(m, p, g, b);
        }
        if (hl(g) || il(g)) return R !== null ? null : d(m, p, g, b, null);
        if (typeof g.then == 'function') return f(m, p, zi(g), b);
        if (g.$$typeof === Ha) return f(m, p, Hi(m, g), b);
        Fi(m, g);
      }
      return null;
    }
    function h(m, p, g, b, R) {
      if ((typeof b == 'string' && b !== '') || typeof b == 'number' || typeof b == 'bigint')
        return (m = m.get(g) || null), i(p, m, '' + b, R);
      if (typeof b == 'object' && b !== null) {
        switch (b.$$typeof) {
          case Ai:
            return (m = m.get(b.key === null ? g : b.key) || null), s(p, m, b, R);
          case pl:
            return (m = m.get(b.key === null ? g : b.key) || null), u(p, m, b, R);
          case cr:
            return (b = Xr(b)), h(m, p, g, b, R);
        }
        if (hl(b) || il(b)) return (m = m.get(g) || null), d(p, m, b, R, null);
        if (typeof b.then == 'function') return h(m, p, g, zi(b), R);
        if (b.$$typeof === Ha) return h(m, p, g, Hi(p, b), R);
        Fi(p, b);
      }
      return null;
    }
    function v(m, p, g, b) {
      for (var R = null, I = null, w = p, L = (p = 0), M = null; w !== null && L < g.length; L++) {
        w.index > L ? ((M = w), (w = null)) : (M = w.sibling);
        var z = f(m, w, g[L], b);
        if (z === null) {
          w === null && (w = M);
          break;
        }
        e && w && z.alternate === null && t(m, w),
          (p = n(z, p, L)),
          I === null ? (R = z) : (I.sibling = z),
          (I = z),
          (w = M);
      }
      if (L === g.length) return a(m, w), J && Ua(m, L), R;
      if (w === null) {
        for (; L < g.length; L++)
          (w = c(m, g[L], b)),
            w !== null && ((p = n(w, p, L)), I === null ? (R = w) : (I.sibling = w), (I = w));
        return J && Ua(m, L), R;
      }
      for (w = r(w); L < g.length; L++)
        (M = h(w, m, L, g[L], b)),
          M !== null &&
            (e && M.alternate !== null && w.delete(M.key === null ? L : M.key),
            (p = n(M, p, L)),
            I === null ? (R = M) : (I.sibling = M),
            (I = M));
      return (
        e &&
          w.forEach(function (qe) {
            return t(m, qe);
          }),
        J && Ua(m, L),
        R
      );
    }
    function x(m, p, g, b) {
      if (g == null) throw Error(C(151));
      for (
        var R = null, I = null, w = p, L = (p = 0), M = null, z = g.next();
        w !== null && !z.done;
        L++, z = g.next()
      ) {
        w.index > L ? ((M = w), (w = null)) : (M = w.sibling);
        var qe = f(m, w, z.value, b);
        if (qe === null) {
          w === null && (w = M);
          break;
        }
        e && w && qe.alternate === null && t(m, w),
          (p = n(qe, p, L)),
          I === null ? (R = qe) : (I.sibling = qe),
          (I = qe),
          (w = M);
      }
      if (z.done) return a(m, w), J && Ua(m, L), R;
      if (w === null) {
        for (; !z.done; L++, z = g.next())
          (z = c(m, z.value, b)),
            z !== null && ((p = n(z, p, L)), I === null ? (R = z) : (I.sibling = z), (I = z));
        return J && Ua(m, L), R;
      }
      for (w = r(w); !z.done; L++, z = g.next())
        (z = h(w, m, L, z.value, b)),
          z !== null &&
            (e && z.alternate !== null && w.delete(z.key === null ? L : z.key),
            (p = n(z, p, L)),
            I === null ? (R = z) : (I.sibling = z),
            (I = z));
      return (
        e &&
          w.forEach(function (pt) {
            return t(m, pt);
          }),
        J && Ua(m, L),
        R
      );
    }
    function y(m, p, g, b) {
      if (
        (typeof g == 'object' &&
          g !== null &&
          g.type === Fo &&
          g.key === null &&
          (g = g.props.children),
        typeof g == 'object' && g !== null)
      ) {
        switch (g.$$typeof) {
          case Ai:
            e: {
              for (var R = g.key; p !== null; ) {
                if (p.key === R) {
                  if (((R = g.type), R === Fo)) {
                    if (p.tag === 7) {
                      a(m, p.sibling), (b = o(p, g.props.children)), (b.return = m), (m = b);
                      break e;
                    }
                  } else if (
                    p.elementType === R ||
                    (typeof R == 'object' && R !== null && R.$$typeof === cr && Xr(R) === p.type)
                  ) {
                    a(m, p.sibling), (b = o(p, g.props)), ul(b, g), (b.return = m), (m = b);
                    break e;
                  }
                  a(m, p);
                  break;
                } else t(m, p);
                p = p.sibling;
              }
              g.type === Fo
                ? ((b = Kr(g.props.children, m.mode, b, g.key)), (b.return = m), (m = b))
                : ((b = $i(g.type, g.key, g.props, null, m.mode, b)),
                  ul(b, g),
                  (b.return = m),
                  (m = b));
            }
            return l(m);
          case pl:
            e: {
              for (R = g.key; p !== null; ) {
                if (p.key === R)
                  if (
                    p.tag === 4 &&
                    p.stateNode.containerInfo === g.containerInfo &&
                    p.stateNode.implementation === g.implementation
                  ) {
                    a(m, p.sibling), (b = o(p, g.children || [])), (b.return = m), (m = b);
                    break e;
                  } else {
                    a(m, p);
                    break;
                  }
                else t(m, p);
                p = p.sibling;
              }
              (b = Dc(g, m.mode, b)), (b.return = m), (m = b);
            }
            return l(m);
          case cr:
            return (g = Xr(g)), y(m, p, g, b);
        }
        if (hl(g)) return v(m, p, g, b);
        if (il(g)) {
          if (((R = il(g)), typeof R != 'function')) throw Error(C(150));
          return (g = R.call(g)), x(m, p, g, b);
        }
        if (typeof g.then == 'function') return y(m, p, zi(g), b);
        if (g.$$typeof === Ha) return y(m, p, Hi(m, g), b);
        Fi(m, g);
      }
      return (typeof g == 'string' && g !== '') || typeof g == 'number' || typeof g == 'bigint'
        ? ((g = '' + g),
          p !== null && p.tag === 6
            ? (a(m, p.sibling), (b = o(p, g)), (b.return = m), (m = b))
            : (a(m, p), (b = Tc(g, m.mode, b)), (b.return = m), (m = b)),
          l(m))
        : a(m, p);
    }
    return function (m, p, g, b) {
      try {
        Nl = 0;
        var R = y(m, p, g, b);
        return (rn = null), R;
      } catch (w) {
        if (w === Sn || w === js) throw w;
        var I = Et(29, w, null, m.mode);
        return (I.lanes = b), (I.return = m), I;
      } finally {
      }
    };
  }
  var ao = a0(!0),
    r0 = a0(!1),
    dr = !1;
  function gf(e) {
    e.updateQueue = {
      baseState: e.memoizedState,
      firstBaseUpdate: null,
      lastBaseUpdate: null,
      shared: { pending: null, lanes: 0, hiddenCallbacks: null },
      callbacks: null,
    };
  }
  function Ld(e, t) {
    (e = e.updateQueue),
      t.updateQueue === e &&
        (t.updateQueue = {
          baseState: e.baseState,
          firstBaseUpdate: e.firstBaseUpdate,
          lastBaseUpdate: e.lastBaseUpdate,
          shared: e.shared,
          callbacks: null,
        });
  }
  function Cr(e) {
    return { lane: e, tag: 0, payload: null, callback: null, next: null };
  }
  function wr(e, t, a) {
    var r = e.updateQueue;
    if (r === null) return null;
    if (((r = r.shared), (ne & 2) !== 0)) {
      var o = r.pending;
      return (
        o === null ? (t.next = t) : ((t.next = o.next), (o.next = t)),
        (r.pending = t),
        (t = gs(e)),
        Wy(e, null, a),
        t
      );
    }
    return Vs(e, r, t, a), gs(e);
  }
  function Cl(e, t, a) {
    if (((t = t.updateQueue), t !== null && ((t = t.shared), (a & 4194048) !== 0))) {
      var r = t.lanes;
      (r &= e.pendingLanes), (a |= r), (t.lanes = a), Sy(e, a);
    }
  }
  function Pc(e, t) {
    var a = e.updateQueue,
      r = e.alternate;
    if (r !== null && ((r = r.updateQueue), a === r)) {
      var o = null,
        n = null;
      if (((a = a.firstBaseUpdate), a !== null)) {
        do {
          var l = { lane: a.lane, tag: a.tag, payload: a.payload, callback: null, next: null };
          n === null ? (o = n = l) : (n = n.next = l), (a = a.next);
        } while (a !== null);
        n === null ? (o = n = t) : (n = n.next = t);
      } else o = n = t;
      (a = {
        baseState: r.baseState,
        firstBaseUpdate: o,
        lastBaseUpdate: n,
        shared: r.shared,
        callbacks: r.callbacks,
      }),
        (e.updateQueue = a);
      return;
    }
    (e = a.lastBaseUpdate),
      e === null ? (a.firstBaseUpdate = t) : (e.next = t),
      (a.lastBaseUpdate = t);
  }
  var Cd = !1;
  function wl() {
    if (Cd) {
      var e = an;
      if (e !== null) throw e;
    }
  }
  function Rl(e, t, a, r) {
    Cd = !1;
    var o = e.updateQueue;
    dr = !1;
    var n = o.firstBaseUpdate,
      l = o.lastBaseUpdate,
      i = o.shared.pending;
    if (i !== null) {
      o.shared.pending = null;
      var s = i,
        u = s.next;
      (s.next = null), l === null ? (n = u) : (l.next = u), (l = s);
      var d = e.alternate;
      d !== null &&
        ((d = d.updateQueue),
        (i = d.lastBaseUpdate),
        i !== l && (i === null ? (d.firstBaseUpdate = u) : (i.next = u), (d.lastBaseUpdate = s)));
    }
    if (n !== null) {
      var c = o.baseState;
      (l = 0), (d = u = s = null), (i = n);
      do {
        var f = i.lane & -536870913,
          h = f !== i.lane;
        if (h ? (K & f) === f : (r & f) === f) {
          f !== 0 && f === cn && (Cd = !0),
            d !== null &&
              (d = d.next =
                { lane: 0, tag: i.tag, payload: i.payload, callback: null, next: null });
          e: {
            var v = e,
              x = i;
            f = t;
            var y = a;
            switch (x.tag) {
              case 1:
                if (((v = x.payload), typeof v == 'function')) {
                  c = v.call(y, c, f);
                  break e;
                }
                c = v;
                break e;
              case 3:
                v.flags = (v.flags & -65537) | 128;
              case 0:
                if (
                  ((v = x.payload), (f = typeof v == 'function' ? v.call(y, c, f) : v), f == null)
                )
                  break e;
                c = Ce({}, c, f);
                break e;
              case 2:
                dr = !0;
            }
          }
          (f = i.callback),
            f !== null &&
              ((e.flags |= 64),
              h && (e.flags |= 8192),
              (h = o.callbacks),
              h === null ? (o.callbacks = [f]) : h.push(f));
        } else
          (h = { lane: f, tag: i.tag, payload: i.payload, callback: i.callback, next: null }),
            d === null ? ((u = d = h), (s = c)) : (d = d.next = h),
            (l |= f);
        if (((i = i.next), i === null)) {
          if (((i = o.shared.pending), i === null)) break;
          (h = i), (i = h.next), (h.next = null), (o.lastBaseUpdate = h), (o.shared.pending = null);
        }
      } while (!0);
      d === null && (s = c),
        (o.baseState = s),
        (o.firstBaseUpdate = u),
        (o.lastBaseUpdate = d),
        n === null && (o.shared.lanes = 0),
        (Dr |= l),
        (e.lanes = l),
        (e.memoizedState = c);
    }
  }
  function o0(e, t) {
    if (typeof e != 'function') throw Error(C(191, e));
    e.call(t);
  }
  function n0(e, t) {
    var a = e.callbacks;
    if (a !== null) for (e.callbacks = null, e = 0; e < a.length; e++) o0(a[e], t);
  }
  var dn = xa(null),
    xs = xa(0);
  function pg(e, t) {
    (e = Qa), ye(xs, e), ye(dn, t), (Qa = e | t.baseLanes);
  }
  function wd() {
    ye(xs, Qa), ye(dn, dn.current);
  }
  function yf() {
    (Qa = xs.current), Ye(dn), Ye(xs);
  }
  var Ut = xa(null),
    Jt = null;
  function mr(e) {
    var t = e.alternate;
    ye(Ae, Ae.current & 1),
      ye(Ut, e),
      Jt === null && (t === null || dn.current !== null || t.memoizedState !== null) && (Jt = e);
  }
  function Rd(e) {
    ye(Ae, Ae.current), ye(Ut, e), Jt === null && (Jt = e);
  }
  function l0(e) {
    e.tag === 22 ? (ye(Ae, Ae.current), ye(Ut, e), Jt === null && (Jt = e)) : pr(e);
  }
  function pr() {
    ye(Ae, Ae.current), ye(Ut, Ut.current);
  }
  function Mt(e) {
    Ye(Ut), Jt === e && (Jt = null), Ye(Ae);
  }
  var Ae = xa(0);
  function Ss(e) {
    for (var t = e; t !== null; ) {
      if (t.tag === 13) {
        var a = t.memoizedState;
        if (a !== null && ((a = a.dehydrated), a === null || Vd(a) || jd(a))) return t;
      } else if (
        t.tag === 19 &&
        (t.memoizedProps.revealOrder === 'forwards' ||
          t.memoizedProps.revealOrder === 'backwards' ||
          t.memoizedProps.revealOrder === 'unstable_legacy-backwards' ||
          t.memoizedProps.revealOrder === 'together')
      ) {
        if ((t.flags & 128) !== 0) return t;
      } else if (t.child !== null) {
        (t.child.return = t), (t = t.child);
        continue;
      }
      if (t === e) break;
      for (; t.sibling === null; ) {
        if (t.return === null || t.return === e) return null;
        t = t.return;
      }
      (t.sibling.return = t.return), (t = t.sibling);
    }
    return null;
  }
  var Ya = 0,
    G = null,
    me = null,
    De = null,
    Ls = !1,
    on = !1,
    ro = !1,
    Cs = 0,
    Hl = 0,
    nn = null,
    mw = 0;
  function ke() {
    throw Error(C(321));
  }
  function vf(e, t) {
    if (t === null) return !1;
    for (var a = 0; a < t.length && a < e.length; a++) if (!Bt(e[a], t[a])) return !1;
    return !0;
  }
  function bf(e, t, a, r, o, n) {
    return (
      (Ya = n),
      (G = t),
      (t.memoizedState = null),
      (t.updateQueue = null),
      (t.lanes = 0),
      (N.H = e === null || e.memoizedState === null ? U0 : Ef),
      (ro = !1),
      (n = a(r, o)),
      (ro = !1),
      on && (n = s0(t, a, r, o)),
      i0(e),
      n
    );
  }
  function i0(e) {
    N.H = zl;
    var t = me !== null && me.next !== null;
    if (((Ya = 0), (De = me = G = null), (Ls = !1), (Hl = 0), (nn = null), t)) throw Error(C(300));
    e === null || Be || ((e = e.dependencies), e !== null && vs(e) && (Be = !0));
  }
  function s0(e, t, a, r) {
    G = e;
    var o = 0;
    do {
      if ((on && (nn = null), (Hl = 0), (on = !1), 25 <= o)) throw Error(C(301));
      if (((o += 1), (De = me = null), e.updateQueue != null)) {
        var n = e.updateQueue;
        (n.lastEffect = null),
          (n.events = null),
          (n.stores = null),
          n.memoCache != null && (n.memoCache.index = 0);
      }
      (N.H = N0), (n = t(a, r));
    } while (on);
    return n;
  }
  function pw() {
    var e = N.H,
      t = e.useState()[0];
    return (
      (t = typeof t.then == 'function' ? $l(t) : t),
      (e = e.useState()[0]),
      (me !== null ? me.memoizedState : null) !== e && (G.flags |= 1024),
      t
    );
  }
  function xf() {
    var e = Cs !== 0;
    return (Cs = 0), e;
  }
  function Sf(e, t, a) {
    (t.updateQueue = e.updateQueue), (t.flags &= -2053), (e.lanes &= ~a);
  }
  function Lf(e) {
    if (Ls) {
      for (e = e.memoizedState; e !== null; ) {
        var t = e.queue;
        t !== null && (t.pending = null), (e = e.next);
      }
      Ls = !1;
    }
    (Ya = 0), (De = me = G = null), (on = !1), (Hl = Cs = 0), (nn = null);
  }
  function mt() {
    var e = { memoizedState: null, baseState: null, baseQueue: null, queue: null, next: null };
    return De === null ? (G.memoizedState = De = e) : (De = De.next = e), De;
  }
  function Te() {
    if (me === null) {
      var e = G.alternate;
      e = e !== null ? e.memoizedState : null;
    } else e = me.next;
    var t = De === null ? G.memoizedState : De.next;
    if (t !== null) (De = t), (me = e);
    else {
      if (e === null) throw G.alternate === null ? Error(C(467)) : Error(C(310));
      (me = e),
        (e = {
          memoizedState: me.memoizedState,
          baseState: me.baseState,
          baseQueue: me.baseQueue,
          queue: me.queue,
          next: null,
        }),
        De === null ? (G.memoizedState = De = e) : (De = De.next = e);
    }
    return De;
  }
  function Ys() {
    return { lastEffect: null, events: null, stores: null, memoCache: null };
  }
  function $l(e) {
    var t = Hl;
    return (
      (Hl += 1),
      nn === null && (nn = []),
      (e = t0(nn, e, t)),
      (t = G),
      (De === null ? t.memoizedState : De.next) === null &&
        ((t = t.alternate), (N.H = t === null || t.memoizedState === null ? U0 : Ef)),
      e
    );
  }
  function Xs(e) {
    if (e !== null && typeof e == 'object') {
      if (typeof e.then == 'function') return $l(e);
      if (e.$$typeof === Ha) return at(e);
    }
    throw Error(C(438, String(e)));
  }
  function Cf(e) {
    var t = null,
      a = G.updateQueue;
    if ((a !== null && (t = a.memoCache), t == null)) {
      var r = G.alternate;
      r !== null &&
        ((r = r.updateQueue),
        r !== null &&
          ((r = r.memoCache),
          r != null &&
            (t = {
              data: r.data.map(function (o) {
                return o.slice();
              }),
              index: 0,
            })));
    }
    if (
      (t == null && (t = { data: [], index: 0 }),
      a === null && ((a = Ys()), (G.updateQueue = a)),
      (a.memoCache = t),
      (a = t.data[t.index]),
      a === void 0)
    )
      for (a = t.data[t.index] = Array(e), r = 0; r < e; r++) a[r] = $1;
    return t.index++, a;
  }
  function Xa(e, t) {
    return typeof t == 'function' ? t(e) : t;
  }
  function ts(e) {
    var t = Te();
    return wf(t, me, e);
  }
  function wf(e, t, a) {
    var r = e.queue;
    if (r === null) throw Error(C(311));
    r.lastRenderedReducer = a;
    var o = e.baseQueue,
      n = r.pending;
    if (n !== null) {
      if (o !== null) {
        var l = o.next;
        (o.next = n.next), (n.next = l);
      }
      (t.baseQueue = o = n), (r.pending = null);
    }
    if (((n = e.baseState), o === null)) e.memoizedState = n;
    else {
      t = o.next;
      var i = (l = null),
        s = null,
        u = t,
        d = !1;
      do {
        var c = u.lane & -536870913;
        if (c !== u.lane ? (K & c) === c : (Ya & c) === c) {
          var f = u.revertLane;
          if (f === 0)
            s !== null &&
              (s = s.next =
                {
                  lane: 0,
                  revertLane: 0,
                  gesture: null,
                  action: u.action,
                  hasEagerState: u.hasEagerState,
                  eagerState: u.eagerState,
                  next: null,
                }),
              c === cn && (d = !0);
          else if ((Ya & f) === f) {
            (u = u.next), f === cn && (d = !0);
            continue;
          } else
            (c = {
              lane: 0,
              revertLane: u.revertLane,
              gesture: null,
              action: u.action,
              hasEagerState: u.hasEagerState,
              eagerState: u.eagerState,
              next: null,
            }),
              s === null ? ((i = s = c), (l = n)) : (s = s.next = c),
              (G.lanes |= f),
              (Dr |= f);
          (c = u.action), ro && a(n, c), (n = u.hasEagerState ? u.eagerState : a(n, c));
        } else
          (f = {
            lane: c,
            revertLane: u.revertLane,
            gesture: u.gesture,
            action: u.action,
            hasEagerState: u.hasEagerState,
            eagerState: u.eagerState,
            next: null,
          }),
            s === null ? ((i = s = f), (l = n)) : (s = s.next = f),
            (G.lanes |= c),
            (Dr |= c);
        u = u.next;
      } while (u !== null && u !== t);
      if (
        (s === null ? (l = n) : (s.next = i),
        !Bt(n, e.memoizedState) && ((Be = !0), d && ((a = an), a !== null)))
      )
        throw a;
      (e.memoizedState = n), (e.baseState = l), (e.baseQueue = s), (r.lastRenderedState = n);
    }
    return o === null && (r.lanes = 0), [e.memoizedState, r.dispatch];
  }
  function Bc(e) {
    var t = Te(),
      a = t.queue;
    if (a === null) throw Error(C(311));
    a.lastRenderedReducer = e;
    var r = a.dispatch,
      o = a.pending,
      n = t.memoizedState;
    if (o !== null) {
      a.pending = null;
      var l = (o = o.next);
      do (n = e(n, l.action)), (l = l.next);
      while (l !== o);
      Bt(n, t.memoizedState) || (Be = !0),
        (t.memoizedState = n),
        t.baseQueue === null && (t.baseState = n),
        (a.lastRenderedState = n);
    }
    return [n, r];
  }
  function u0(e, t, a) {
    var r = G,
      o = Te(),
      n = J;
    if (n) {
      if (a === void 0) throw Error(C(407));
      a = a();
    } else a = t();
    var l = !Bt((me || o).memoizedState, a);
    if (
      (l && ((o.memoizedState = a), (Be = !0)),
      (o = o.queue),
      Rf(f0.bind(null, r, o, e), [e]),
      o.getSnapshot !== t || l || (De !== null && De.memoizedState.tag & 1))
    ) {
      if (
        ((r.flags |= 2048),
        fn(9, { destroy: void 0 }, d0.bind(null, r, o, a, t), null),
        he === null)
      )
        throw Error(C(349));
      n || (Ya & 127) !== 0 || c0(r, t, a);
    }
    return a;
  }
  function c0(e, t, a) {
    (e.flags |= 16384),
      (e = { getSnapshot: t, value: a }),
      (t = G.updateQueue),
      t === null
        ? ((t = Ys()), (G.updateQueue = t), (t.stores = [e]))
        : ((a = t.stores), a === null ? (t.stores = [e]) : a.push(e));
  }
  function d0(e, t, a, r) {
    (t.value = a), (t.getSnapshot = r), m0(t) && p0(e);
  }
  function f0(e, t, a) {
    return a(function () {
      m0(t) && p0(e);
    });
  }
  function m0(e) {
    var t = e.getSnapshot;
    e = e.value;
    try {
      var a = t();
      return !Bt(e, a);
    } catch {
      return !0;
    }
  }
  function p0(e) {
    var t = so(e, 2);
    t !== null && xt(t, e, 2);
  }
  function _d(e) {
    var t = mt();
    if (typeof e == 'function') {
      var a = e;
      if (((e = a()), ro)) {
        gr(!0);
        try {
          a();
        } finally {
          gr(!1);
        }
      }
    }
    return (
      (t.memoizedState = t.baseState = e),
      (t.queue = {
        pending: null,
        lanes: 0,
        dispatch: null,
        lastRenderedReducer: Xa,
        lastRenderedState: e,
      }),
      t
    );
  }
  function h0(e, t, a, r) {
    return (e.baseState = a), wf(e, me, typeof r == 'function' ? r : Xa);
  }
  function hw(e, t, a, r, o) {
    if (Qs(e)) throw Error(C(485));
    if (((e = t.action), e !== null)) {
      var n = {
        payload: o,
        action: e,
        next: null,
        isTransition: !0,
        status: 'pending',
        value: null,
        reason: null,
        listeners: [],
        then: function (l) {
          n.listeners.push(l);
        },
      };
      N.T !== null ? a(!0) : (n.isTransition = !1),
        r(n),
        (a = t.pending),
        a === null
          ? ((n.next = t.pending = n), g0(t, n))
          : ((n.next = a.next), (t.pending = a.next = n));
    }
  }
  function g0(e, t) {
    var a = t.action,
      r = t.payload,
      o = e.state;
    if (t.isTransition) {
      var n = N.T,
        l = {};
      N.T = l;
      try {
        var i = a(o, r),
          s = N.S;
        s !== null && s(l, i), hg(e, t, i);
      } catch (u) {
        Id(e, t, u);
      } finally {
        n !== null && l.types !== null && (n.types = l.types), (N.T = n);
      }
    } else
      try {
        (n = a(o, r)), hg(e, t, n);
      } catch (u) {
        Id(e, t, u);
      }
  }
  function hg(e, t, a) {
    a !== null && typeof a == 'object' && typeof a.then == 'function'
      ? a.then(
          function (r) {
            gg(e, t, r);
          },
          function (r) {
            return Id(e, t, r);
          }
        )
      : gg(e, t, a);
  }
  function gg(e, t, a) {
    (t.status = 'fulfilled'),
      (t.value = a),
      y0(t),
      (e.state = a),
      (t = e.pending),
      t !== null &&
        ((a = t.next), a === t ? (e.pending = null) : ((a = a.next), (t.next = a), g0(e, a)));
  }
  function Id(e, t, a) {
    var r = e.pending;
    if (((e.pending = null), r !== null)) {
      r = r.next;
      do (t.status = 'rejected'), (t.reason = a), y0(t), (t = t.next);
      while (t !== r);
    }
    e.action = null;
  }
  function y0(e) {
    e = e.listeners;
    for (var t = 0; t < e.length; t++) (0, e[t])();
  }
  function v0(e, t) {
    return t;
  }
  function yg(e, t) {
    if (J) {
      var a = he.formState;
      if (a !== null) {
        e: {
          var r = G;
          if (J) {
            if (Le) {
              t: {
                for (var o = Le, n = Zt; o.nodeType !== 8; ) {
                  if (!n) {
                    o = null;
                    break t;
                  }
                  if (((o = $t(o.nextSibling)), o === null)) {
                    o = null;
                    break t;
                  }
                }
                (n = o.data), (o = n === 'F!' || n === 'F' ? o : null);
              }
              if (o) {
                (Le = $t(o.nextSibling)), (r = o.data === 'F!');
                break e;
              }
            }
            Ar(r);
          }
          r = !1;
        }
        r && (t = a[0]);
      }
    }
    return (
      (a = mt()),
      (a.memoizedState = a.baseState = t),
      (r = {
        pending: null,
        lanes: 0,
        dispatch: null,
        lastRenderedReducer: v0,
        lastRenderedState: t,
      }),
      (a.queue = r),
      (a = O0.bind(null, G, r)),
      (r.dispatch = a),
      (r = _d(!1)),
      (n = Mf.bind(null, G, !1, r.queue)),
      (r = mt()),
      (o = { state: t, dispatch: null, action: e, pending: null }),
      (r.queue = o),
      (a = hw.bind(null, G, o, n, a)),
      (o.dispatch = a),
      (r.memoizedState = e),
      [t, a, !1]
    );
  }
  function vg(e) {
    var t = Te();
    return b0(t, me, e);
  }
  function b0(e, t, a) {
    if (
      ((t = wf(e, t, v0)[0]),
      (e = ts(Xa)[0]),
      typeof t == 'object' && t !== null && typeof t.then == 'function')
    )
      try {
        var r = $l(t);
      } catch (l) {
        throw l === Sn ? js : l;
      }
    else r = t;
    t = Te();
    var o = t.queue,
      n = o.dispatch;
    return (
      a !== t.memoizedState &&
        ((G.flags |= 2048), fn(9, { destroy: void 0 }, gw.bind(null, o, a), null)),
      [r, n, e]
    );
  }
  function gw(e, t) {
    e.action = t;
  }
  function bg(e) {
    var t = Te(),
      a = me;
    if (a !== null) return b0(t, a, e);
    Te(), (t = t.memoizedState), (a = Te());
    var r = a.queue.dispatch;
    return (a.memoizedState = e), [t, r, !1];
  }
  function fn(e, t, a, r) {
    return (
      (e = { tag: e, create: a, deps: r, inst: t, next: null }),
      (t = G.updateQueue),
      t === null && ((t = Ys()), (G.updateQueue = t)),
      (a = t.lastEffect),
      a === null
        ? (t.lastEffect = e.next = e)
        : ((r = a.next), (a.next = e), (e.next = r), (t.lastEffect = e)),
      e
    );
  }
  function x0() {
    return Te().memoizedState;
  }
  function as(e, t, a, r) {
    var o = mt();
    (G.flags |= e), (o.memoizedState = fn(1 | t, { destroy: void 0 }, a, r === void 0 ? null : r));
  }
  function Ws(e, t, a, r) {
    var o = Te();
    r = r === void 0 ? null : r;
    var n = o.memoizedState.inst;
    me !== null && r !== null && vf(r, me.memoizedState.deps)
      ? (o.memoizedState = fn(t, n, a, r))
      : ((G.flags |= e), (o.memoizedState = fn(1 | t, n, a, r)));
  }
  function xg(e, t) {
    as(8390656, 8, e, t);
  }
  function Rf(e, t) {
    Ws(2048, 8, e, t);
  }
  function yw(e) {
    G.flags |= 4;
    var t = G.updateQueue;
    if (t === null) (t = Ys()), (G.updateQueue = t), (t.events = [e]);
    else {
      var a = t.events;
      a === null ? (t.events = [e]) : a.push(e);
    }
  }
  function S0(e) {
    var t = Te().memoizedState;
    return (
      yw({ ref: t, nextImpl: e }),
      function () {
        if ((ne & 2) !== 0) throw Error(C(440));
        return t.impl.apply(void 0, arguments);
      }
    );
  }
  function L0(e, t) {
    return Ws(4, 2, e, t);
  }
  function C0(e, t) {
    return Ws(4, 4, e, t);
  }
  function w0(e, t) {
    if (typeof t == 'function') {
      e = e();
      var a = t(e);
      return function () {
        typeof a == 'function' ? a() : t(null);
      };
    }
    if (t != null)
      return (
        (e = e()),
        (t.current = e),
        function () {
          t.current = null;
        }
      );
  }
  function R0(e, t, a) {
    (a = a != null ? a.concat([e]) : null), Ws(4, 4, w0.bind(null, t, e), a);
  }
  function _f() {}
  function _0(e, t) {
    var a = Te();
    t = t === void 0 ? null : t;
    var r = a.memoizedState;
    return t !== null && vf(t, r[1]) ? r[0] : ((a.memoizedState = [e, t]), e);
  }
  function I0(e, t) {
    var a = Te();
    t = t === void 0 ? null : t;
    var r = a.memoizedState;
    if (t !== null && vf(t, r[1])) return r[0];
    if (((r = e()), ro)) {
      gr(!0);
      try {
        e();
      } finally {
        gr(!1);
      }
    }
    return (a.memoizedState = [r, t]), r;
  }
  function If(e, t, a) {
    return a === void 0 || ((Ya & 1073741824) !== 0 && (K & 261930) === 0)
      ? (e.memoizedState = t)
      : ((e.memoizedState = a), (e = hv()), (G.lanes |= e), (Dr |= e), a);
  }
  function k0(e, t, a, r) {
    return Bt(a, t)
      ? a
      : dn.current !== null
        ? ((e = If(e, a, r)), Bt(e, t) || (Be = !0), e)
        : (Ya & 42) === 0 || ((Ya & 1073741824) !== 0 && (K & 261930) === 0)
          ? ((Be = !0), (e.memoizedState = a))
          : ((e = hv()), (G.lanes |= e), (Dr |= e), t);
  }
  function M0(e, t, a, r, o) {
    var n = le.p;
    le.p = n !== 0 && 8 > n ? n : 8;
    var l = N.T,
      i = {};
    (N.T = i), Mf(e, !1, t, a);
    try {
      var s = o(),
        u = N.S;
      if (
        (u !== null && u(i, s), s !== null && typeof s == 'object' && typeof s.then == 'function')
      ) {
        var d = fw(s, r);
        _l(e, t, d, Pt(e));
      } else _l(e, t, r, Pt(e));
    } catch (c) {
      _l(e, t, { then: function () {}, status: 'rejected', reason: c }, Pt());
    } finally {
      (le.p = n), l !== null && i.types !== null && (l.types = i.types), (N.T = l);
    }
  }
  function vw() {}
  function kd(e, t, a, r) {
    if (e.tag !== 5) throw Error(C(476));
    var o = E0(e).queue;
    M0(
      e,
      o,
      t,
      Qr,
      a === null
        ? vw
        : function () {
            return A0(e), a(r);
          }
    );
  }
  function E0(e) {
    var t = e.memoizedState;
    if (t !== null) return t;
    t = {
      memoizedState: Qr,
      baseState: Qr,
      baseQueue: null,
      queue: {
        pending: null,
        lanes: 0,
        dispatch: null,
        lastRenderedReducer: Xa,
        lastRenderedState: Qr,
      },
      next: null,
    };
    var a = {};
    return (
      (t.next = {
        memoizedState: a,
        baseState: a,
        baseQueue: null,
        queue: {
          pending: null,
          lanes: 0,
          dispatch: null,
          lastRenderedReducer: Xa,
          lastRenderedState: a,
        },
        next: null,
      }),
      (e.memoizedState = t),
      (e = e.alternate),
      e !== null && (e.memoizedState = t),
      t
    );
  }
  function A0(e) {
    var t = E0(e);
    t.next === null && (t = e.alternate.memoizedState), _l(e, t.next.queue, {}, Pt());
  }
  function kf() {
    return at(Gl);
  }
  function T0() {
    return Te().memoizedState;
  }
  function D0() {
    return Te().memoizedState;
  }
  function bw(e) {
    for (var t = e.return; t !== null; ) {
      switch (t.tag) {
        case 24:
        case 3:
          var a = Pt();
          e = Cr(a);
          var r = wr(t, e, a);
          r !== null && (xt(r, t, a), Cl(r, t, a)), (t = { cache: mf() }), (e.payload = t);
          return;
      }
      t = t.return;
    }
  }
  function xw(e, t, a) {
    var r = Pt();
    (a = {
      lane: r,
      revertLane: 0,
      gesture: null,
      action: a,
      hasEagerState: !1,
      eagerState: null,
      next: null,
    }),
      Qs(e) ? P0(t, a) : ((a = uf(e, t, a, r)), a !== null && (xt(a, e, r), B0(a, t, r)));
  }
  function O0(e, t, a) {
    var r = Pt();
    _l(e, t, a, r);
  }
  function _l(e, t, a, r) {
    var o = {
      lane: r,
      revertLane: 0,
      gesture: null,
      action: a,
      hasEagerState: !1,
      eagerState: null,
      next: null,
    };
    if (Qs(e)) P0(t, o);
    else {
      var n = e.alternate;
      if (
        e.lanes === 0 &&
        (n === null || n.lanes === 0) &&
        ((n = t.lastRenderedReducer), n !== null)
      )
        try {
          var l = t.lastRenderedState,
            i = n(l, a);
          if (((o.hasEagerState = !0), (o.eagerState = i), Bt(i, l)))
            return Vs(e, t, o, 0), he === null && Gs(), !1;
        } catch {
        } finally {
        }
      if (((a = uf(e, t, o, r)), a !== null)) return xt(a, e, r), B0(a, t, r), !0;
    }
    return !1;
  }
  function Mf(e, t, a, r) {
    if (
      ((r = {
        lane: 2,
        revertLane: Nf(),
        gesture: null,
        action: r,
        hasEagerState: !1,
        eagerState: null,
        next: null,
      }),
      Qs(e))
    ) {
      if (t) throw Error(C(479));
    } else (t = uf(e, a, r, 2)), t !== null && xt(t, e, 2);
  }
  function Qs(e) {
    var t = e.alternate;
    return e === G || (t !== null && t === G);
  }
  function P0(e, t) {
    on = Ls = !0;
    var a = e.pending;
    a === null ? (t.next = t) : ((t.next = a.next), (a.next = t)), (e.pending = t);
  }
  function B0(e, t, a) {
    if ((a & 4194048) !== 0) {
      var r = t.lanes;
      (r &= e.pendingLanes), (a |= r), (t.lanes = a), Sy(e, a);
    }
  }
  var zl = {
    readContext: at,
    use: Xs,
    useCallback: ke,
    useContext: ke,
    useEffect: ke,
    useImperativeHandle: ke,
    useLayoutEffect: ke,
    useInsertionEffect: ke,
    useMemo: ke,
    useReducer: ke,
    useRef: ke,
    useState: ke,
    useDebugValue: ke,
    useDeferredValue: ke,
    useTransition: ke,
    useSyncExternalStore: ke,
    useId: ke,
    useHostTransitionStatus: ke,
    useFormState: ke,
    useActionState: ke,
    useOptimistic: ke,
    useMemoCache: ke,
    useCacheRefresh: ke,
  };
  zl.useEffectEvent = ke;
  var U0 = {
      readContext: at,
      use: Xs,
      useCallback: function (e, t) {
        return (mt().memoizedState = [e, t === void 0 ? null : t]), e;
      },
      useContext: at,
      useEffect: xg,
      useImperativeHandle: function (e, t, a) {
        (a = a != null ? a.concat([e]) : null), as(4194308, 4, w0.bind(null, t, e), a);
      },
      useLayoutEffect: function (e, t) {
        return as(4194308, 4, e, t);
      },
      useInsertionEffect: function (e, t) {
        as(4, 2, e, t);
      },
      useMemo: function (e, t) {
        var a = mt();
        t = t === void 0 ? null : t;
        var r = e();
        if (ro) {
          gr(!0);
          try {
            e();
          } finally {
            gr(!1);
          }
        }
        return (a.memoizedState = [r, t]), r;
      },
      useReducer: function (e, t, a) {
        var r = mt();
        if (a !== void 0) {
          var o = a(t);
          if (ro) {
            gr(!0);
            try {
              a(t);
            } finally {
              gr(!1);
            }
          }
        } else o = t;
        return (
          (r.memoizedState = r.baseState = o),
          (e = {
            pending: null,
            lanes: 0,
            dispatch: null,
            lastRenderedReducer: e,
            lastRenderedState: o,
          }),
          (r.queue = e),
          (e = e.dispatch = xw.bind(null, G, e)),
          [r.memoizedState, e]
        );
      },
      useRef: function (e) {
        var t = mt();
        return (e = { current: e }), (t.memoizedState = e);
      },
      useState: function (e) {
        e = _d(e);
        var t = e.queue,
          a = O0.bind(null, G, t);
        return (t.dispatch = a), [e.memoizedState, a];
      },
      useDebugValue: _f,
      useDeferredValue: function (e, t) {
        var a = mt();
        return If(a, e, t);
      },
      useTransition: function () {
        var e = _d(!1);
        return (e = M0.bind(null, G, e.queue, !0, !1)), (mt().memoizedState = e), [!1, e];
      },
      useSyncExternalStore: function (e, t, a) {
        var r = G,
          o = mt();
        if (J) {
          if (a === void 0) throw Error(C(407));
          a = a();
        } else {
          if (((a = t()), he === null)) throw Error(C(349));
          (K & 127) !== 0 || c0(r, t, a);
        }
        o.memoizedState = a;
        var n = { value: a, getSnapshot: t };
        return (
          (o.queue = n),
          xg(f0.bind(null, r, n, e), [e]),
          (r.flags |= 2048),
          fn(9, { destroy: void 0 }, d0.bind(null, r, n, a, t), null),
          a
        );
      },
      useId: function () {
        var e = mt(),
          t = he.identifierPrefix;
        if (J) {
          var a = ya,
            r = ga;
          (a = (r & ~(1 << (32 - Ot(r) - 1))).toString(32) + a),
            (t = '_' + t + 'R_' + a),
            (a = Cs++),
            0 < a && (t += 'H' + a.toString(32)),
            (t += '_');
        } else (a = mw++), (t = '_' + t + 'r_' + a.toString(32) + '_');
        return (e.memoizedState = t);
      },
      useHostTransitionStatus: kf,
      useFormState: yg,
      useActionState: yg,
      useOptimistic: function (e) {
        var t = mt();
        t.memoizedState = t.baseState = e;
        var a = {
          pending: null,
          lanes: 0,
          dispatch: null,
          lastRenderedReducer: null,
          lastRenderedState: null,
        };
        return (t.queue = a), (t = Mf.bind(null, G, !0, a)), (a.dispatch = t), [e, t];
      },
      useMemoCache: Cf,
      useCacheRefresh: function () {
        return (mt().memoizedState = bw.bind(null, G));
      },
      useEffectEvent: function (e) {
        var t = mt(),
          a = { impl: e };
        return (
          (t.memoizedState = a),
          function () {
            if ((ne & 2) !== 0) throw Error(C(440));
            return a.impl.apply(void 0, arguments);
          }
        );
      },
    },
    Ef = {
      readContext: at,
      use: Xs,
      useCallback: _0,
      useContext: at,
      useEffect: Rf,
      useImperativeHandle: R0,
      useInsertionEffect: L0,
      useLayoutEffect: C0,
      useMemo: I0,
      useReducer: ts,
      useRef: x0,
      useState: function () {
        return ts(Xa);
      },
      useDebugValue: _f,
      useDeferredValue: function (e, t) {
        var a = Te();
        return k0(a, me.memoizedState, e, t);
      },
      useTransition: function () {
        var e = ts(Xa)[0],
          t = Te().memoizedState;
        return [typeof e == 'boolean' ? e : $l(e), t];
      },
      useSyncExternalStore: u0,
      useId: T0,
      useHostTransitionStatus: kf,
      useFormState: vg,
      useActionState: vg,
      useOptimistic: function (e, t) {
        var a = Te();
        return h0(a, me, e, t);
      },
      useMemoCache: Cf,
      useCacheRefresh: D0,
    };
  Ef.useEffectEvent = S0;
  var N0 = {
    readContext: at,
    use: Xs,
    useCallback: _0,
    useContext: at,
    useEffect: Rf,
    useImperativeHandle: R0,
    useInsertionEffect: L0,
    useLayoutEffect: C0,
    useMemo: I0,
    useReducer: Bc,
    useRef: x0,
    useState: function () {
      return Bc(Xa);
    },
    useDebugValue: _f,
    useDeferredValue: function (e, t) {
      var a = Te();
      return me === null ? If(a, e, t) : k0(a, me.memoizedState, e, t);
    },
    useTransition: function () {
      var e = Bc(Xa)[0],
        t = Te().memoizedState;
      return [typeof e == 'boolean' ? e : $l(e), t];
    },
    useSyncExternalStore: u0,
    useId: T0,
    useHostTransitionStatus: kf,
    useFormState: bg,
    useActionState: bg,
    useOptimistic: function (e, t) {
      var a = Te();
      return me !== null ? h0(a, me, e, t) : ((a.baseState = e), [e, a.queue.dispatch]);
    },
    useMemoCache: Cf,
    useCacheRefresh: D0,
  };
  N0.useEffectEvent = S0;
  function Uc(e, t, a, r) {
    (t = e.memoizedState),
      (a = a(r, t)),
      (a = a == null ? t : Ce({}, t, a)),
      (e.memoizedState = a),
      e.lanes === 0 && (e.updateQueue.baseState = a);
  }
  var Md = {
    enqueueSetState: function (e, t, a) {
      e = e._reactInternals;
      var r = Pt(),
        o = Cr(r);
      (o.payload = t),
        a != null && (o.callback = a),
        (t = wr(e, o, r)),
        t !== null && (xt(t, e, r), Cl(t, e, r));
    },
    enqueueReplaceState: function (e, t, a) {
      e = e._reactInternals;
      var r = Pt(),
        o = Cr(r);
      (o.tag = 1),
        (o.payload = t),
        a != null && (o.callback = a),
        (t = wr(e, o, r)),
        t !== null && (xt(t, e, r), Cl(t, e, r));
    },
    enqueueForceUpdate: function (e, t) {
      e = e._reactInternals;
      var a = Pt(),
        r = Cr(a);
      (r.tag = 2),
        t != null && (r.callback = t),
        (t = wr(e, r, a)),
        t !== null && (xt(t, e, a), Cl(t, e, a));
    },
  };
  function Sg(e, t, a, r, o, n, l) {
    return (
      (e = e.stateNode),
      typeof e.shouldComponentUpdate == 'function'
        ? e.shouldComponentUpdate(r, n, l)
        : t.prototype && t.prototype.isPureReactComponent
          ? !Pl(a, r) || !Pl(o, n)
          : !0
    );
  }
  function Lg(e, t, a, r) {
    (e = t.state),
      typeof t.componentWillReceiveProps == 'function' && t.componentWillReceiveProps(a, r),
      typeof t.UNSAFE_componentWillReceiveProps == 'function' &&
        t.UNSAFE_componentWillReceiveProps(a, r),
      t.state !== e && Md.enqueueReplaceState(t, t.state, null);
  }
  function oo(e, t) {
    var a = t;
    if ('ref' in t) {
      a = {};
      for (var r in t) r !== 'ref' && (a[r] = t[r]);
    }
    if ((e = e.defaultProps)) {
      a === t && (a = Ce({}, a));
      for (var o in e) a[o] === void 0 && (a[o] = e[o]);
    }
    return a;
  }
  function H0(e) {
    hs(e);
  }
  function z0(e) {
    console.error(e);
  }
  function F0(e) {
    hs(e);
  }
  function ws(e, t) {
    try {
      var a = e.onUncaughtError;
      a(t.value, { componentStack: t.stack });
    } catch (r) {
      setTimeout(function () {
        throw r;
      });
    }
  }
  function Cg(e, t, a) {
    try {
      var r = e.onCaughtError;
      r(a.value, { componentStack: a.stack, errorBoundary: t.tag === 1 ? t.stateNode : null });
    } catch (o) {
      setTimeout(function () {
        throw o;
      });
    }
  }
  function Ed(e, t, a) {
    return (
      (a = Cr(a)),
      (a.tag = 3),
      (a.payload = { element: null }),
      (a.callback = function () {
        ws(e, t);
      }),
      a
    );
  }
  function q0(e) {
    return (e = Cr(e)), (e.tag = 3), e;
  }
  function G0(e, t, a, r) {
    var o = a.type.getDerivedStateFromError;
    if (typeof o == 'function') {
      var n = r.value;
      (e.payload = function () {
        return o(n);
      }),
        (e.callback = function () {
          Cg(t, a, r);
        });
    }
    var l = a.stateNode;
    l !== null &&
      typeof l.componentDidCatch == 'function' &&
      (e.callback = function () {
        Cg(t, a, r),
          typeof o != 'function' && (Rr === null ? (Rr = new Set([this])) : Rr.add(this));
        var i = r.stack;
        this.componentDidCatch(r.value, { componentStack: i !== null ? i : '' });
      });
  }
  function Sw(e, t, a, r, o) {
    if (((a.flags |= 32768), r !== null && typeof r == 'object' && typeof r.then == 'function')) {
      if (((t = a.alternate), t !== null && xn(t, a, o, !0), (a = Ut.current), a !== null)) {
        switch (a.tag) {
          case 31:
          case 13:
            return (
              Jt === null ? Ms() : a.alternate === null && Me === 0 && (Me = 3),
              (a.flags &= -257),
              (a.flags |= 65536),
              (a.lanes = o),
              r === bs
                ? (a.flags |= 16384)
                : ((t = a.updateQueue),
                  t === null ? (a.updateQueue = new Set([r])) : t.add(r),
                  Wc(e, r, o)),
              !1
            );
          case 22:
            return (
              (a.flags |= 65536),
              r === bs
                ? (a.flags |= 16384)
                : ((t = a.updateQueue),
                  t === null
                    ? ((t = { transitions: null, markerInstances: null, retryQueue: new Set([r]) }),
                      (a.updateQueue = t))
                    : ((a = t.retryQueue), a === null ? (t.retryQueue = new Set([r])) : a.add(r)),
                  Wc(e, r, o)),
              !1
            );
        }
        throw Error(C(435, a.tag));
      }
      return Wc(e, r, o), Ms(), !1;
    }
    if (J)
      return (
        (t = Ut.current),
        t !== null
          ? ((t.flags & 65536) === 0 && (t.flags |= 256),
            (t.flags |= 65536),
            (t.lanes = o),
            r !== yd && ((e = Error(C(422), { cause: r })), Ul(Kt(e, a))))
          : (r !== yd && ((t = Error(C(423), { cause: r })), Ul(Kt(t, a))),
            (e = e.current.alternate),
            (e.flags |= 65536),
            (o &= -o),
            (e.lanes |= o),
            (r = Kt(r, a)),
            (o = Ed(e.stateNode, r, o)),
            Pc(e, o),
            Me !== 4 && (Me = 2)),
        !1
      );
    var n = Error(C(520), { cause: r });
    if (((n = Kt(n, a)), Ml === null ? (Ml = [n]) : Ml.push(n), Me !== 4 && (Me = 2), t === null))
      return !0;
    (r = Kt(r, a)), (a = t);
    do {
      switch (a.tag) {
        case 3:
          return (
            (a.flags |= 65536),
            (e = o & -o),
            (a.lanes |= e),
            (e = Ed(a.stateNode, r, e)),
            Pc(a, e),
            !1
          );
        case 1:
          if (
            ((t = a.type),
            (n = a.stateNode),
            (a.flags & 128) === 0 &&
              (typeof t.getDerivedStateFromError == 'function' ||
                (n !== null &&
                  typeof n.componentDidCatch == 'function' &&
                  (Rr === null || !Rr.has(n)))))
          )
            return (
              (a.flags |= 65536),
              (o &= -o),
              (a.lanes |= o),
              (o = q0(o)),
              G0(o, e, a, r),
              Pc(a, o),
              !1
            );
      }
      a = a.return;
    } while (a !== null);
    return !1;
  }
  var Af = Error(C(461)),
    Be = !1;
  function $e(e, t, a, r) {
    t.child = e === null ? r0(t, null, a, r) : ao(t, e.child, a, r);
  }
  function wg(e, t, a, r, o) {
    a = a.render;
    var n = t.ref;
    if ('ref' in r) {
      var l = {};
      for (var i in r) i !== 'ref' && (l[i] = r[i]);
    } else l = r;
    return (
      to(t),
      (r = bf(e, t, a, l, n, o)),
      (i = xf()),
      e !== null && !Be
        ? (Sf(e, t, o), Wa(e, t, o))
        : (J && i && df(t), (t.flags |= 1), $e(e, t, r, o), t.child)
    );
  }
  function Rg(e, t, a, r, o) {
    if (e === null) {
      var n = a.type;
      return typeof n == 'function' && !cf(n) && n.defaultProps === void 0 && a.compare === null
        ? ((t.tag = 15), (t.type = n), V0(e, t, n, r, o))
        : ((e = $i(a.type, null, r, t, t.mode, o)), (e.ref = t.ref), (e.return = t), (t.child = e));
    }
    if (((n = e.child), !Tf(e, o))) {
      var l = n.memoizedProps;
      if (((a = a.compare), (a = a !== null ? a : Pl), a(l, r) && e.ref === t.ref))
        return Wa(e, t, o);
    }
    return (t.flags |= 1), (e = qa(n, r)), (e.ref = t.ref), (e.return = t), (t.child = e);
  }
  function V0(e, t, a, r, o) {
    if (e !== null) {
      var n = e.memoizedProps;
      if (Pl(n, r) && e.ref === t.ref)
        if (((Be = !1), (t.pendingProps = r = n), Tf(e, o))) (e.flags & 131072) !== 0 && (Be = !0);
        else return (t.lanes = e.lanes), Wa(e, t, o);
    }
    return Ad(e, t, a, r, o);
  }
  function j0(e, t, a, r) {
    var o = r.children,
      n = e !== null ? e.memoizedState : null;
    if (
      (e === null &&
        t.stateNode === null &&
        (t.stateNode = {
          _visibility: 1,
          _pendingMarkers: null,
          _retryCache: null,
          _transitions: null,
        }),
      r.mode === 'hidden')
    ) {
      if ((t.flags & 128) !== 0) {
        if (((n = n !== null ? n.baseLanes | a : a), e !== null)) {
          for (r = t.child = e.child, o = 0; r !== null; )
            (o = o | r.lanes | r.childLanes), (r = r.sibling);
          r = o & ~n;
        } else (r = 0), (t.child = null);
        return _g(e, t, n, a, r);
      }
      if ((a & 536870912) !== 0)
        (t.memoizedState = { baseLanes: 0, cachePool: null }),
          e !== null && es(t, n !== null ? n.cachePool : null),
          n !== null ? pg(t, n) : wd(),
          l0(t);
      else return (r = t.lanes = 536870912), _g(e, t, n !== null ? n.baseLanes | a : a, a, r);
    } else
      n !== null
        ? (es(t, n.cachePool), pg(t, n), pr(t), (t.memoizedState = null))
        : (e !== null && es(t, null), wd(), pr(t));
    return $e(e, t, o, a), t.child;
  }
  function yl(e, t) {
    return (
      (e !== null && e.tag === 22) ||
        t.stateNode !== null ||
        (t.stateNode = {
          _visibility: 1,
          _pendingMarkers: null,
          _retryCache: null,
          _transitions: null,
        }),
      t.sibling
    );
  }
  function _g(e, t, a, r, o) {
    var n = pf();
    return (
      (n = n === null ? null : { parent: Pe._currentValue, pool: n }),
      (t.memoizedState = { baseLanes: a, cachePool: n }),
      e !== null && es(t, null),
      wd(),
      l0(t),
      e !== null && xn(e, t, r, !0),
      (t.childLanes = o),
      null
    );
  }
  function rs(e, t) {
    return (
      (t = Rs({ mode: t.mode, children: t.children }, e.mode)),
      (t.ref = e.ref),
      (e.child = t),
      (t.return = e),
      t
    );
  }
  function Ig(e, t, a) {
    return (
      ao(t, e.child, null, a),
      (e = rs(t, t.pendingProps)),
      (e.flags |= 2),
      Mt(t),
      (t.memoizedState = null),
      e
    );
  }
  function Lw(e, t, a) {
    var r = t.pendingProps,
      o = (t.flags & 128) !== 0;
    if (((t.flags &= -129), e === null)) {
      if (J) {
        if (r.mode === 'hidden') return (e = rs(t, r)), (t.lanes = 536870912), yl(null, e);
        if (
          (Rd(t),
          (e = Le)
            ? ((e = Uv(e, Zt)),
              (e = e !== null && e.data === '&' ? e : null),
              e !== null &&
                ((t.memoizedState = {
                  dehydrated: e,
                  treeContext: Er !== null ? { id: ga, overflow: ya } : null,
                  retryLane: 536870912,
                  hydrationErrors: null,
                }),
                (a = Ky(e)),
                (a.return = t),
                (t.child = a),
                (tt = t),
                (Le = null)))
            : (e = null),
          e === null)
        )
          throw Ar(t);
        return (t.lanes = 536870912), null;
      }
      return rs(t, r);
    }
    var n = e.memoizedState;
    if (n !== null) {
      var l = n.dehydrated;
      if ((Rd(t), o))
        if (t.flags & 256) (t.flags &= -257), (t = Ig(e, t, a));
        else if (t.memoizedState !== null) (t.child = e.child), (t.flags |= 128), (t = null);
        else throw Error(C(558));
      else if ((Be || xn(e, t, a, !1), (o = (a & e.childLanes) !== 0), Be || o)) {
        if (((r = he), r !== null && ((l = Ly(r, a)), l !== 0 && l !== n.retryLane)))
          throw ((n.retryLane = l), so(e, l), xt(r, e, l), Af);
        Ms(), (t = Ig(e, t, a));
      } else
        (e = n.treeContext),
          (Le = $t(l.nextSibling)),
          (tt = t),
          (J = !0),
          (Lr = null),
          (Zt = !1),
          e !== null && Jy(t, e),
          (t = rs(t, r)),
          (t.flags |= 4096);
      return t;
    }
    return (
      (e = qa(e.child, { mode: r.mode, children: r.children })),
      (e.ref = t.ref),
      (t.child = e),
      (e.return = t),
      e
    );
  }
  function os(e, t) {
    var a = t.ref;
    if (a === null) e !== null && e.ref !== null && (t.flags |= 4194816);
    else {
      if (typeof a != 'function' && typeof a != 'object') throw Error(C(284));
      (e === null || e.ref !== a) && (t.flags |= 4194816);
    }
  }
  function Ad(e, t, a, r, o) {
    return (
      to(t),
      (a = bf(e, t, a, r, void 0, o)),
      (r = xf()),
      e !== null && !Be
        ? (Sf(e, t, o), Wa(e, t, o))
        : (J && r && df(t), (t.flags |= 1), $e(e, t, a, o), t.child)
    );
  }
  function kg(e, t, a, r, o, n) {
    return (
      to(t),
      (t.updateQueue = null),
      (a = s0(t, r, a, o)),
      i0(e),
      (r = xf()),
      e !== null && !Be
        ? (Sf(e, t, n), Wa(e, t, n))
        : (J && r && df(t), (t.flags |= 1), $e(e, t, a, n), t.child)
    );
  }
  function Mg(e, t, a, r, o) {
    if ((to(t), t.stateNode === null)) {
      var n = Qo,
        l = a.contextType;
      typeof l == 'object' && l !== null && (n = at(l)),
        (n = new a(r, n)),
        (t.memoizedState = n.state !== null && n.state !== void 0 ? n.state : null),
        (n.updater = Md),
        (t.stateNode = n),
        (n._reactInternals = t),
        (n = t.stateNode),
        (n.props = r),
        (n.state = t.memoizedState),
        (n.refs = {}),
        gf(t),
        (l = a.contextType),
        (n.context = typeof l == 'object' && l !== null ? at(l) : Qo),
        (n.state = t.memoizedState),
        (l = a.getDerivedStateFromProps),
        typeof l == 'function' && (Uc(t, a, l, r), (n.state = t.memoizedState)),
        typeof a.getDerivedStateFromProps == 'function' ||
          typeof n.getSnapshotBeforeUpdate == 'function' ||
          (typeof n.UNSAFE_componentWillMount != 'function' &&
            typeof n.componentWillMount != 'function') ||
          ((l = n.state),
          typeof n.componentWillMount == 'function' && n.componentWillMount(),
          typeof n.UNSAFE_componentWillMount == 'function' && n.UNSAFE_componentWillMount(),
          l !== n.state && Md.enqueueReplaceState(n, n.state, null),
          Rl(t, r, n, o),
          wl(),
          (n.state = t.memoizedState)),
        typeof n.componentDidMount == 'function' && (t.flags |= 4194308),
        (r = !0);
    } else if (e === null) {
      n = t.stateNode;
      var i = t.memoizedProps,
        s = oo(a, i);
      n.props = s;
      var u = n.context,
        d = a.contextType;
      (l = Qo), typeof d == 'object' && d !== null && (l = at(d));
      var c = a.getDerivedStateFromProps;
      (d = typeof c == 'function' || typeof n.getSnapshotBeforeUpdate == 'function'),
        (i = t.pendingProps !== i),
        d ||
          (typeof n.UNSAFE_componentWillReceiveProps != 'function' &&
            typeof n.componentWillReceiveProps != 'function') ||
          ((i || u !== l) && Lg(t, n, r, l)),
        (dr = !1);
      var f = t.memoizedState;
      (n.state = f),
        Rl(t, r, n, o),
        wl(),
        (u = t.memoizedState),
        i || f !== u || dr
          ? (typeof c == 'function' && (Uc(t, a, c, r), (u = t.memoizedState)),
            (s = dr || Sg(t, a, s, r, f, u, l))
              ? (d ||
                  (typeof n.UNSAFE_componentWillMount != 'function' &&
                    typeof n.componentWillMount != 'function') ||
                  (typeof n.componentWillMount == 'function' && n.componentWillMount(),
                  typeof n.UNSAFE_componentWillMount == 'function' &&
                    n.UNSAFE_componentWillMount()),
                typeof n.componentDidMount == 'function' && (t.flags |= 4194308))
              : (typeof n.componentDidMount == 'function' && (t.flags |= 4194308),
                (t.memoizedProps = r),
                (t.memoizedState = u)),
            (n.props = r),
            (n.state = u),
            (n.context = l),
            (r = s))
          : (typeof n.componentDidMount == 'function' && (t.flags |= 4194308), (r = !1));
    } else {
      (n = t.stateNode),
        Ld(e, t),
        (l = t.memoizedProps),
        (d = oo(a, l)),
        (n.props = d),
        (c = t.pendingProps),
        (f = n.context),
        (u = a.contextType),
        (s = Qo),
        typeof u == 'object' && u !== null && (s = at(u)),
        (i = a.getDerivedStateFromProps),
        (u = typeof i == 'function' || typeof n.getSnapshotBeforeUpdate == 'function') ||
          (typeof n.UNSAFE_componentWillReceiveProps != 'function' &&
            typeof n.componentWillReceiveProps != 'function') ||
          ((l !== c || f !== s) && Lg(t, n, r, s)),
        (dr = !1),
        (f = t.memoizedState),
        (n.state = f),
        Rl(t, r, n, o),
        wl();
      var h = t.memoizedState;
      l !== c || f !== h || dr || (e !== null && e.dependencies !== null && vs(e.dependencies))
        ? (typeof i == 'function' && (Uc(t, a, i, r), (h = t.memoizedState)),
          (d =
            dr ||
            Sg(t, a, d, r, f, h, s) ||
            (e !== null && e.dependencies !== null && vs(e.dependencies)))
            ? (u ||
                (typeof n.UNSAFE_componentWillUpdate != 'function' &&
                  typeof n.componentWillUpdate != 'function') ||
                (typeof n.componentWillUpdate == 'function' && n.componentWillUpdate(r, h, s),
                typeof n.UNSAFE_componentWillUpdate == 'function' &&
                  n.UNSAFE_componentWillUpdate(r, h, s)),
              typeof n.componentDidUpdate == 'function' && (t.flags |= 4),
              typeof n.getSnapshotBeforeUpdate == 'function' && (t.flags |= 1024))
            : (typeof n.componentDidUpdate != 'function' ||
                (l === e.memoizedProps && f === e.memoizedState) ||
                (t.flags |= 4),
              typeof n.getSnapshotBeforeUpdate != 'function' ||
                (l === e.memoizedProps && f === e.memoizedState) ||
                (t.flags |= 1024),
              (t.memoizedProps = r),
              (t.memoizedState = h)),
          (n.props = r),
          (n.state = h),
          (n.context = s),
          (r = d))
        : (typeof n.componentDidUpdate != 'function' ||
            (l === e.memoizedProps && f === e.memoizedState) ||
            (t.flags |= 4),
          typeof n.getSnapshotBeforeUpdate != 'function' ||
            (l === e.memoizedProps && f === e.memoizedState) ||
            (t.flags |= 1024),
          (r = !1));
    }
    return (
      (n = r),
      os(e, t),
      (r = (t.flags & 128) !== 0),
      n || r
        ? ((n = t.stateNode),
          (a = r && typeof a.getDerivedStateFromError != 'function' ? null : n.render()),
          (t.flags |= 1),
          e !== null && r
            ? ((t.child = ao(t, e.child, null, o)), (t.child = ao(t, null, a, o)))
            : $e(e, t, a, o),
          (t.memoizedState = n.state),
          (e = t.child))
        : (e = Wa(e, t, o)),
      e
    );
  }
  function Eg(e, t, a, r) {
    return eo(), (t.flags |= 256), $e(e, t, a, r), t.child;
  }
  var Nc = { dehydrated: null, treeContext: null, retryLane: 0, hydrationErrors: null };
  function Hc(e) {
    return { baseLanes: e, cachePool: e0() };
  }
  function zc(e, t, a) {
    return (e = e !== null ? e.childLanes & ~a : 0), t && (e |= At), e;
  }
  function Y0(e, t, a) {
    var r = t.pendingProps,
      o = !1,
      n = (t.flags & 128) !== 0,
      l;
    if (
      ((l = n) || (l = e !== null && e.memoizedState === null ? !1 : (Ae.current & 2) !== 0),
      l && ((o = !0), (t.flags &= -129)),
      (l = (t.flags & 32) !== 0),
      (t.flags &= -33),
      e === null)
    ) {
      if (J) {
        if (
          (o ? mr(t) : pr(t),
          (e = Le)
            ? ((e = Uv(e, Zt)),
              (e = e !== null && e.data !== '&' ? e : null),
              e !== null &&
                ((t.memoizedState = {
                  dehydrated: e,
                  treeContext: Er !== null ? { id: ga, overflow: ya } : null,
                  retryLane: 536870912,
                  hydrationErrors: null,
                }),
                (a = Ky(e)),
                (a.return = t),
                (t.child = a),
                (tt = t),
                (Le = null)))
            : (e = null),
          e === null)
        )
          throw Ar(t);
        return jd(e) ? (t.lanes = 32) : (t.lanes = 536870912), null;
      }
      var i = r.children;
      return (
        (r = r.fallback),
        o
          ? (pr(t),
            (o = t.mode),
            (i = Rs({ mode: 'hidden', children: i }, o)),
            (r = Kr(r, o, a, null)),
            (i.return = t),
            (r.return = t),
            (i.sibling = r),
            (t.child = i),
            (r = t.child),
            (r.memoizedState = Hc(a)),
            (r.childLanes = zc(e, l, a)),
            (t.memoizedState = Nc),
            yl(null, r))
          : (mr(t), Td(t, i))
      );
    }
    var s = e.memoizedState;
    if (s !== null && ((i = s.dehydrated), i !== null)) {
      if (n)
        t.flags & 256
          ? (mr(t), (t.flags &= -257), (t = Fc(e, t, a)))
          : t.memoizedState !== null
            ? (pr(t), (t.child = e.child), (t.flags |= 128), (t = null))
            : (pr(t),
              (i = r.fallback),
              (o = t.mode),
              (r = Rs({ mode: 'visible', children: r.children }, o)),
              (i = Kr(i, o, a, null)),
              (i.flags |= 2),
              (r.return = t),
              (i.return = t),
              (r.sibling = i),
              (t.child = r),
              ao(t, e.child, null, a),
              (r = t.child),
              (r.memoizedState = Hc(a)),
              (r.childLanes = zc(e, l, a)),
              (t.memoizedState = Nc),
              (t = yl(null, r)));
      else if ((mr(t), jd(i))) {
        if (((l = i.nextSibling && i.nextSibling.dataset), l)) var u = l.dgst;
        (l = u),
          (r = Error(C(419))),
          (r.stack = ''),
          (r.digest = l),
          Ul({ value: r, source: null, stack: null }),
          (t = Fc(e, t, a));
      } else if ((Be || xn(e, t, a, !1), (l = (a & e.childLanes) !== 0), Be || l)) {
        if (((l = he), l !== null && ((r = Ly(l, a)), r !== 0 && r !== s.retryLane)))
          throw ((s.retryLane = r), so(e, r), xt(l, e, r), Af);
        Vd(i) || Ms(), (t = Fc(e, t, a));
      } else
        Vd(i)
          ? ((t.flags |= 192), (t.child = e.child), (t = null))
          : ((e = s.treeContext),
            (Le = $t(i.nextSibling)),
            (tt = t),
            (J = !0),
            (Lr = null),
            (Zt = !1),
            e !== null && Jy(t, e),
            (t = Td(t, r.children)),
            (t.flags |= 4096));
      return t;
    }
    return o
      ? (pr(t),
        (i = r.fallback),
        (o = t.mode),
        (s = e.child),
        (u = s.sibling),
        (r = qa(s, { mode: 'hidden', children: r.children })),
        (r.subtreeFlags = s.subtreeFlags & 65011712),
        u !== null ? (i = qa(u, i)) : ((i = Kr(i, o, a, null)), (i.flags |= 2)),
        (i.return = t),
        (r.return = t),
        (r.sibling = i),
        (t.child = r),
        yl(null, r),
        (r = t.child),
        (i = e.child.memoizedState),
        i === null
          ? (i = Hc(a))
          : ((o = i.cachePool),
            o !== null
              ? ((s = Pe._currentValue), (o = o.parent !== s ? { parent: s, pool: s } : o))
              : (o = e0()),
            (i = { baseLanes: i.baseLanes | a, cachePool: o })),
        (r.memoizedState = i),
        (r.childLanes = zc(e, l, a)),
        (t.memoizedState = Nc),
        yl(e.child, r))
      : (mr(t),
        (a = e.child),
        (e = a.sibling),
        (a = qa(a, { mode: 'visible', children: r.children })),
        (a.return = t),
        (a.sibling = null),
        e !== null &&
          ((l = t.deletions), l === null ? ((t.deletions = [e]), (t.flags |= 16)) : l.push(e)),
        (t.child = a),
        (t.memoizedState = null),
        a);
  }
  function Td(e, t) {
    return (t = Rs({ mode: 'visible', children: t }, e.mode)), (t.return = e), (e.child = t);
  }
  function Rs(e, t) {
    return (e = Et(22, e, null, t)), (e.lanes = 0), e;
  }
  function Fc(e, t, a) {
    return (
      ao(t, e.child, null, a),
      (e = Td(t, t.pendingProps.children)),
      (e.flags |= 2),
      (t.memoizedState = null),
      e
    );
  }
  function Ag(e, t, a) {
    e.lanes |= t;
    var r = e.alternate;
    r !== null && (r.lanes |= t), bd(e.return, t, a);
  }
  function qc(e, t, a, r, o, n) {
    var l = e.memoizedState;
    l === null
      ? (e.memoizedState = {
          isBackwards: t,
          rendering: null,
          renderingStartTime: 0,
          last: r,
          tail: a,
          tailMode: o,
          treeForkCount: n,
        })
      : ((l.isBackwards = t),
        (l.rendering = null),
        (l.renderingStartTime = 0),
        (l.last = r),
        (l.tail = a),
        (l.tailMode = o),
        (l.treeForkCount = n));
  }
  function X0(e, t, a) {
    var r = t.pendingProps,
      o = r.revealOrder,
      n = r.tail;
    r = r.children;
    var l = Ae.current,
      i = (l & 2) !== 0;
    if (
      (i ? ((l = (l & 1) | 2), (t.flags |= 128)) : (l &= 1),
      ye(Ae, l),
      $e(e, t, r, a),
      (r = J ? Bl : 0),
      !i && e !== null && (e.flags & 128) !== 0)
    )
      e: for (e = t.child; e !== null; ) {
        if (e.tag === 13) e.memoizedState !== null && Ag(e, a, t);
        else if (e.tag === 19) Ag(e, a, t);
        else if (e.child !== null) {
          (e.child.return = e), (e = e.child);
          continue;
        }
        if (e === t) break e;
        for (; e.sibling === null; ) {
          if (e.return === null || e.return === t) break e;
          e = e.return;
        }
        (e.sibling.return = e.return), (e = e.sibling);
      }
    switch (o) {
      case 'forwards':
        for (a = t.child, o = null; a !== null; )
          (e = a.alternate), e !== null && Ss(e) === null && (o = a), (a = a.sibling);
        (a = o),
          a === null ? ((o = t.child), (t.child = null)) : ((o = a.sibling), (a.sibling = null)),
          qc(t, !1, o, a, n, r);
        break;
      case 'backwards':
      case 'unstable_legacy-backwards':
        for (a = null, o = t.child, t.child = null; o !== null; ) {
          if (((e = o.alternate), e !== null && Ss(e) === null)) {
            t.child = o;
            break;
          }
          (e = o.sibling), (o.sibling = a), (a = o), (o = e);
        }
        qc(t, !0, a, null, n, r);
        break;
      case 'together':
        qc(t, !1, null, null, void 0, r);
        break;
      default:
        t.memoizedState = null;
    }
    return t.child;
  }
  function Wa(e, t, a) {
    if (
      (e !== null && (t.dependencies = e.dependencies), (Dr |= t.lanes), (a & t.childLanes) === 0)
    )
      if (e !== null) {
        if ((xn(e, t, a, !1), (a & t.childLanes) === 0)) return null;
      } else return null;
    if (e !== null && t.child !== e.child) throw Error(C(153));
    if (t.child !== null) {
      for (e = t.child, a = qa(e, e.pendingProps), t.child = a, a.return = t; e.sibling !== null; )
        (e = e.sibling), (a = a.sibling = qa(e, e.pendingProps)), (a.return = t);
      a.sibling = null;
    }
    return t.child;
  }
  function Tf(e, t) {
    return (e.lanes & t) !== 0 ? !0 : ((e = e.dependencies), !!(e !== null && vs(e)));
  }
  function Cw(e, t, a) {
    switch (t.tag) {
      case 3:
        ds(t, t.stateNode.containerInfo), fr(t, Pe, e.memoizedState.cache), eo();
        break;
      case 27:
      case 5:
        ld(t);
        break;
      case 4:
        ds(t, t.stateNode.containerInfo);
        break;
      case 10:
        fr(t, t.type, t.memoizedProps.value);
        break;
      case 31:
        if (t.memoizedState !== null) return (t.flags |= 128), Rd(t), null;
        break;
      case 13:
        var r = t.memoizedState;
        if (r !== null)
          return r.dehydrated !== null
            ? (mr(t), (t.flags |= 128), null)
            : (a & t.child.childLanes) !== 0
              ? Y0(e, t, a)
              : (mr(t), (e = Wa(e, t, a)), e !== null ? e.sibling : null);
        mr(t);
        break;
      case 19:
        var o = (e.flags & 128) !== 0;
        if (
          ((r = (a & t.childLanes) !== 0),
          r || (xn(e, t, a, !1), (r = (a & t.childLanes) !== 0)),
          o)
        ) {
          if (r) return X0(e, t, a);
          t.flags |= 128;
        }
        if (
          ((o = t.memoizedState),
          o !== null && ((o.rendering = null), (o.tail = null), (o.lastEffect = null)),
          ye(Ae, Ae.current),
          r)
        )
          break;
        return null;
      case 22:
        return (t.lanes = 0), j0(e, t, a, t.pendingProps);
      case 24:
        fr(t, Pe, e.memoizedState.cache);
    }
    return Wa(e, t, a);
  }
  function W0(e, t, a) {
    if (e !== null)
      if (e.memoizedProps !== t.pendingProps) Be = !0;
      else {
        if (!Tf(e, a) && (t.flags & 128) === 0) return (Be = !1), Cw(e, t, a);
        Be = (e.flags & 131072) !== 0;
      }
    else (Be = !1), J && (t.flags & 1048576) !== 0 && Zy(t, Bl, t.index);
    switch (((t.lanes = 0), t.tag)) {
      case 16:
        e: {
          var r = t.pendingProps;
          if (((e = Xr(t.elementType)), (t.type = e), typeof e == 'function'))
            cf(e)
              ? ((r = oo(e, r)), (t.tag = 1), (t = Mg(null, t, e, r, a)))
              : ((t.tag = 0), (t = Ad(null, t, e, r, a)));
          else {
            if (e != null) {
              var o = e.$$typeof;
              if (o === Qd) {
                (t.tag = 11), (t = wg(null, t, e, r, a));
                break e;
              } else if (o === Kd) {
                (t.tag = 14), (t = Rg(null, t, e, r, a));
                break e;
              }
            }
            throw ((t = od(e) || e), Error(C(306, t, '')));
          }
        }
        return t;
      case 0:
        return Ad(e, t, t.type, t.pendingProps, a);
      case 1:
        return (r = t.type), (o = oo(r, t.pendingProps)), Mg(e, t, r, o, a);
      case 3:
        e: {
          if ((ds(t, t.stateNode.containerInfo), e === null)) throw Error(C(387));
          r = t.pendingProps;
          var n = t.memoizedState;
          (o = n.element), Ld(e, t), Rl(t, r, null, a);
          var l = t.memoizedState;
          if (
            ((r = l.cache),
            fr(t, Pe, r),
            r !== n.cache && xd(t, [Pe], a, !0),
            wl(),
            (r = l.element),
            n.isDehydrated)
          )
            if (
              ((n = { element: r, isDehydrated: !1, cache: l.cache }),
              (t.updateQueue.baseState = n),
              (t.memoizedState = n),
              t.flags & 256)
            ) {
              t = Eg(e, t, r, a);
              break e;
            } else if (r !== o) {
              (o = Kt(Error(C(424)), t)), Ul(o), (t = Eg(e, t, r, a));
              break e;
            } else {
              switch (((e = t.stateNode.containerInfo), e.nodeType)) {
                case 9:
                  e = e.body;
                  break;
                default:
                  e = e.nodeName === 'HTML' ? e.ownerDocument.body : e;
              }
              for (
                Le = $t(e.firstChild),
                  tt = t,
                  J = !0,
                  Lr = null,
                  Zt = !0,
                  a = r0(t, null, r, a),
                  t.child = a;
                a;

              )
                (a.flags = (a.flags & -3) | 4096), (a = a.sibling);
            }
          else {
            if ((eo(), r === o)) {
              t = Wa(e, t, a);
              break e;
            }
            $e(e, t, r, a);
          }
          t = t.child;
        }
        return t;
      case 26:
        return (
          os(e, t),
          e === null
            ? (a = $g(t.type, null, t.pendingProps, null))
              ? (t.memoizedState = a)
              : J ||
                ((a = t.type),
                (e = t.pendingProps),
                (r = Ds(Sr.current).createElement(a)),
                (r[et] = t),
                (r[St] = e),
                rt(r, a, e),
                je(r),
                (t.stateNode = r))
            : (t.memoizedState = $g(t.type, e.memoizedProps, t.pendingProps, e.memoizedState)),
          null
        );
      case 27:
        return (
          ld(t),
          e === null &&
            J &&
            ((r = t.stateNode = Nv(t.type, t.pendingProps, Sr.current)),
            (tt = t),
            (Zt = !0),
            (o = Le),
            Pr(t.type) ? ((Yd = o), (Le = $t(r.firstChild))) : (Le = o)),
          $e(e, t, t.pendingProps.children, a),
          os(e, t),
          e === null && (t.flags |= 4194304),
          t.child
        );
      case 5:
        return (
          e === null &&
            J &&
            ((o = r = Le) &&
              ((r = Zw(r, t.type, t.pendingProps, Zt)),
              r !== null
                ? ((t.stateNode = r), (tt = t), (Le = $t(r.firstChild)), (Zt = !1), (o = !0))
                : (o = !1)),
            o || Ar(t)),
          ld(t),
          (o = t.type),
          (n = t.pendingProps),
          (l = e !== null ? e.memoizedProps : null),
          (r = n.children),
          qd(o, n) ? (r = null) : l !== null && qd(o, l) && (t.flags |= 32),
          t.memoizedState !== null && ((o = bf(e, t, pw, null, null, a)), (Gl._currentValue = o)),
          os(e, t),
          $e(e, t, r, a),
          t.child
        );
      case 6:
        return (
          e === null &&
            J &&
            ((e = a = Le) &&
              ((a = Jw(a, t.pendingProps, Zt)),
              a !== null ? ((t.stateNode = a), (tt = t), (Le = null), (e = !0)) : (e = !1)),
            e || Ar(t)),
          null
        );
      case 13:
        return Y0(e, t, a);
      case 4:
        return (
          ds(t, t.stateNode.containerInfo),
          (r = t.pendingProps),
          e === null ? (t.child = ao(t, null, r, a)) : $e(e, t, r, a),
          t.child
        );
      case 11:
        return wg(e, t, t.type, t.pendingProps, a);
      case 7:
        return $e(e, t, t.pendingProps, a), t.child;
      case 8:
        return $e(e, t, t.pendingProps.children, a), t.child;
      case 12:
        return $e(e, t, t.pendingProps.children, a), t.child;
      case 10:
        return (r = t.pendingProps), fr(t, t.type, r.value), $e(e, t, r.children, a), t.child;
      case 9:
        return (
          (o = t.type._context),
          (r = t.pendingProps.children),
          to(t),
          (o = at(o)),
          (r = r(o)),
          (t.flags |= 1),
          $e(e, t, r, a),
          t.child
        );
      case 14:
        return Rg(e, t, t.type, t.pendingProps, a);
      case 15:
        return V0(e, t, t.type, t.pendingProps, a);
      case 19:
        return X0(e, t, a);
      case 31:
        return Lw(e, t, a);
      case 22:
        return j0(e, t, a, t.pendingProps);
      case 24:
        return (
          to(t),
          (r = at(Pe)),
          e === null
            ? ((o = pf()),
              o === null &&
                ((o = he),
                (n = mf()),
                (o.pooledCache = n),
                n.refCount++,
                n !== null && (o.pooledCacheLanes |= a),
                (o = n)),
              (t.memoizedState = { parent: r, cache: o }),
              gf(t),
              fr(t, Pe, o))
            : ((e.lanes & a) !== 0 && (Ld(e, t), Rl(t, null, null, a), wl()),
              (o = e.memoizedState),
              (n = t.memoizedState),
              o.parent !== r
                ? ((o = { parent: r, cache: r }),
                  (t.memoizedState = o),
                  t.lanes === 0 && (t.memoizedState = t.updateQueue.baseState = o),
                  fr(t, Pe, r))
                : ((r = n.cache), fr(t, Pe, r), r !== o.cache && xd(t, [Pe], a, !0))),
          $e(e, t, t.pendingProps.children, a),
          t.child
        );
      case 29:
        throw t.pendingProps;
    }
    throw Error(C(156, t.tag));
  }
  function Da(e) {
    e.flags |= 4;
  }
  function Gc(e, t, a, r, o) {
    if (((t = (e.mode & 32) !== 0) && (t = !1), t)) {
      if (((e.flags |= 16777216), (o & 335544128) === o))
        if (e.stateNode.complete) e.flags |= 8192;
        else if (vv()) e.flags |= 8192;
        else throw ((Jr = bs), hf);
    } else e.flags &= -16777217;
  }
  function Tg(e, t) {
    if (t.type !== 'stylesheet' || (t.state.loading & 4) !== 0) e.flags &= -16777217;
    else if (((e.flags |= 16777216), !Fv(t)))
      if (vv()) e.flags |= 8192;
      else throw ((Jr = bs), hf);
  }
  function qi(e, t) {
    t !== null && (e.flags |= 4),
      e.flags & 16384 && ((t = e.tag !== 22 ? by() : 536870912), (e.lanes |= t), (mn |= t));
  }
  function cl(e, t) {
    if (!J)
      switch (e.tailMode) {
        case 'hidden':
          t = e.tail;
          for (var a = null; t !== null; ) t.alternate !== null && (a = t), (t = t.sibling);
          a === null ? (e.tail = null) : (a.sibling = null);
          break;
        case 'collapsed':
          a = e.tail;
          for (var r = null; a !== null; ) a.alternate !== null && (r = a), (a = a.sibling);
          r === null
            ? t || e.tail === null
              ? (e.tail = null)
              : (e.tail.sibling = null)
            : (r.sibling = null);
      }
  }
  function Se(e) {
    var t = e.alternate !== null && e.alternate.child === e.child,
      a = 0,
      r = 0;
    if (t)
      for (var o = e.child; o !== null; )
        (a |= o.lanes | o.childLanes),
          (r |= o.subtreeFlags & 65011712),
          (r |= o.flags & 65011712),
          (o.return = e),
          (o = o.sibling);
    else
      for (o = e.child; o !== null; )
        (a |= o.lanes | o.childLanes),
          (r |= o.subtreeFlags),
          (r |= o.flags),
          (o.return = e),
          (o = o.sibling);
    return (e.subtreeFlags |= r), (e.childLanes = a), t;
  }
  function ww(e, t, a) {
    var r = t.pendingProps;
    switch ((ff(t), t.tag)) {
      case 16:
      case 15:
      case 0:
      case 11:
      case 7:
      case 8:
      case 12:
      case 9:
      case 14:
        return Se(t), null;
      case 1:
        return Se(t), null;
      case 3:
        return (
          (a = t.stateNode),
          (r = null),
          e !== null && (r = e.memoizedState.cache),
          t.memoizedState.cache !== r && (t.flags |= 2048),
          Ga(Pe),
          ln(),
          a.pendingContext && ((a.context = a.pendingContext), (a.pendingContext = null)),
          (e === null || e.child === null) &&
            (Uo(t)
              ? Da(t)
              : e === null ||
                (e.memoizedState.isDehydrated && (t.flags & 256) === 0) ||
                ((t.flags |= 1024), Oc())),
          Se(t),
          null
        );
      case 26:
        var o = t.type,
          n = t.memoizedState;
        return (
          e === null
            ? (Da(t), n !== null ? (Se(t), Tg(t, n)) : (Se(t), Gc(t, o, null, r, a)))
            : n
              ? n !== e.memoizedState
                ? (Da(t), Se(t), Tg(t, n))
                : (Se(t), (t.flags &= -16777217))
              : ((e = e.memoizedProps), e !== r && Da(t), Se(t), Gc(t, o, e, r, a)),
          null
        );
      case 27:
        if ((fs(t), (a = Sr.current), (o = t.type), e !== null && t.stateNode != null))
          e.memoizedProps !== r && Da(t);
        else {
          if (!r) {
            if (t.stateNode === null) throw Error(C(166));
            return Se(t), null;
          }
          (e = ba.current), Uo(t) ? ig(t, e) : ((e = Nv(o, r, a)), (t.stateNode = e), Da(t));
        }
        return Se(t), null;
      case 5:
        if ((fs(t), (o = t.type), e !== null && t.stateNode != null))
          e.memoizedProps !== r && Da(t);
        else {
          if (!r) {
            if (t.stateNode === null) throw Error(C(166));
            return Se(t), null;
          }
          if (((n = ba.current), Uo(t))) ig(t, n);
          else {
            var l = Ds(Sr.current);
            switch (n) {
              case 1:
                n = l.createElementNS('http://www.w3.org/2000/svg', o);
                break;
              case 2:
                n = l.createElementNS('http://www.w3.org/1998/Math/MathML', o);
                break;
              default:
                switch (o) {
                  case 'svg':
                    n = l.createElementNS('http://www.w3.org/2000/svg', o);
                    break;
                  case 'math':
                    n = l.createElementNS('http://www.w3.org/1998/Math/MathML', o);
                    break;
                  case 'script':
                    (n = l.createElement('div')),
                      (n.innerHTML = '<script><\/script>'),
                      (n = n.removeChild(n.firstChild));
                    break;
                  case 'select':
                    (n =
                      typeof r.is == 'string'
                        ? l.createElement('select', { is: r.is })
                        : l.createElement('select')),
                      r.multiple ? (n.multiple = !0) : r.size && (n.size = r.size);
                    break;
                  default:
                    n =
                      typeof r.is == 'string'
                        ? l.createElement(o, { is: r.is })
                        : l.createElement(o);
                }
            }
            (n[et] = t), (n[St] = r);
            e: for (l = t.child; l !== null; ) {
              if (l.tag === 5 || l.tag === 6) n.appendChild(l.stateNode);
              else if (l.tag !== 4 && l.tag !== 27 && l.child !== null) {
                (l.child.return = l), (l = l.child);
                continue;
              }
              if (l === t) break e;
              for (; l.sibling === null; ) {
                if (l.return === null || l.return === t) break e;
                l = l.return;
              }
              (l.sibling.return = l.return), (l = l.sibling);
            }
            t.stateNode = n;
            e: switch ((rt(n, o, r), o)) {
              case 'button':
              case 'input':
              case 'select':
              case 'textarea':
                r = !!r.autoFocus;
                break e;
              case 'img':
                r = !0;
                break e;
              default:
                r = !1;
            }
            r && Da(t);
          }
        }
        return Se(t), Gc(t, t.type, e === null ? null : e.memoizedProps, t.pendingProps, a), null;
      case 6:
        if (e && t.stateNode != null) e.memoizedProps !== r && Da(t);
        else {
          if (typeof r != 'string' && t.stateNode === null) throw Error(C(166));
          if (((e = Sr.current), Uo(t))) {
            if (((e = t.stateNode), (a = t.memoizedProps), (r = null), (o = tt), o !== null))
              switch (o.tag) {
                case 27:
                case 5:
                  r = o.memoizedProps;
              }
            (e[et] = t),
              (e = !!(
                e.nodeValue === a ||
                (r !== null && r.suppressHydrationWarning === !0) ||
                Ov(e.nodeValue, a)
              )),
              e || Ar(t, !0);
          } else (e = Ds(e).createTextNode(r)), (e[et] = t), (t.stateNode = e);
        }
        return Se(t), null;
      case 31:
        if (((a = t.memoizedState), e === null || e.memoizedState !== null)) {
          if (((r = Uo(t)), a !== null)) {
            if (e === null) {
              if (!r) throw Error(C(318));
              if (((e = t.memoizedState), (e = e !== null ? e.dehydrated : null), !e))
                throw Error(C(557));
              e[et] = t;
            } else eo(), (t.flags & 128) === 0 && (t.memoizedState = null), (t.flags |= 4);
            Se(t), (e = !1);
          } else
            (a = Oc()),
              e !== null && e.memoizedState !== null && (e.memoizedState.hydrationErrors = a),
              (e = !0);
          if (!e) return t.flags & 256 ? (Mt(t), t) : (Mt(t), null);
          if ((t.flags & 128) !== 0) throw Error(C(558));
        }
        return Se(t), null;
      case 13:
        if (
          ((r = t.memoizedState),
          e === null || (e.memoizedState !== null && e.memoizedState.dehydrated !== null))
        ) {
          if (((o = Uo(t)), r !== null && r.dehydrated !== null)) {
            if (e === null) {
              if (!o) throw Error(C(318));
              if (((o = t.memoizedState), (o = o !== null ? o.dehydrated : null), !o))
                throw Error(C(317));
              o[et] = t;
            } else eo(), (t.flags & 128) === 0 && (t.memoizedState = null), (t.flags |= 4);
            Se(t), (o = !1);
          } else
            (o = Oc()),
              e !== null && e.memoizedState !== null && (e.memoizedState.hydrationErrors = o),
              (o = !0);
          if (!o) return t.flags & 256 ? (Mt(t), t) : (Mt(t), null);
        }
        return (
          Mt(t),
          (t.flags & 128) !== 0
            ? ((t.lanes = a), t)
            : ((a = r !== null),
              (e = e !== null && e.memoizedState !== null),
              a &&
                ((r = t.child),
                (o = null),
                r.alternate !== null &&
                  r.alternate.memoizedState !== null &&
                  r.alternate.memoizedState.cachePool !== null &&
                  (o = r.alternate.memoizedState.cachePool.pool),
                (n = null),
                r.memoizedState !== null &&
                  r.memoizedState.cachePool !== null &&
                  (n = r.memoizedState.cachePool.pool),
                n !== o && (r.flags |= 2048)),
              a !== e && a && (t.child.flags |= 8192),
              qi(t, t.updateQueue),
              Se(t),
              null)
        );
      case 4:
        return ln(), e === null && Hf(t.stateNode.containerInfo), Se(t), null;
      case 10:
        return Ga(t.type), Se(t), null;
      case 19:
        if ((Ye(Ae), (r = t.memoizedState), r === null)) return Se(t), null;
        if (((o = (t.flags & 128) !== 0), (n = r.rendering), n === null))
          if (o) cl(r, !1);
          else {
            if (Me !== 0 || (e !== null && (e.flags & 128) !== 0))
              for (e = t.child; e !== null; ) {
                if (((n = Ss(e)), n !== null)) {
                  for (
                    t.flags |= 128,
                      cl(r, !1),
                      e = n.updateQueue,
                      t.updateQueue = e,
                      qi(t, e),
                      t.subtreeFlags = 0,
                      e = a,
                      a = t.child;
                    a !== null;

                  )
                    Qy(a, e), (a = a.sibling);
                  return ye(Ae, (Ae.current & 1) | 2), J && Ua(t, r.treeForkCount), t.child;
                }
                e = e.sibling;
              }
            r.tail !== null &&
              Tt() > Is &&
              ((t.flags |= 128), (o = !0), cl(r, !1), (t.lanes = 4194304));
          }
        else {
          if (!o)
            if (((e = Ss(n)), e !== null)) {
              if (
                ((t.flags |= 128),
                (o = !0),
                (e = e.updateQueue),
                (t.updateQueue = e),
                qi(t, e),
                cl(r, !0),
                r.tail === null && r.tailMode === 'hidden' && !n.alternate && !J)
              )
                return Se(t), null;
            } else
              2 * Tt() - r.renderingStartTime > Is &&
                a !== 536870912 &&
                ((t.flags |= 128), (o = !0), cl(r, !1), (t.lanes = 4194304));
          r.isBackwards
            ? ((n.sibling = t.child), (t.child = n))
            : ((e = r.last), e !== null ? (e.sibling = n) : (t.child = n), (r.last = n));
        }
        return r.tail !== null
          ? ((e = r.tail),
            (r.rendering = e),
            (r.tail = e.sibling),
            (r.renderingStartTime = Tt()),
            (e.sibling = null),
            (a = Ae.current),
            ye(Ae, o ? (a & 1) | 2 : a & 1),
            J && Ua(t, r.treeForkCount),
            e)
          : (Se(t), null);
      case 22:
      case 23:
        return (
          Mt(t),
          yf(),
          (r = t.memoizedState !== null),
          e !== null
            ? (e.memoizedState !== null) !== r && (t.flags |= 8192)
            : r && (t.flags |= 8192),
          r
            ? (a & 536870912) !== 0 &&
              (t.flags & 128) === 0 &&
              (Se(t), t.subtreeFlags & 6 && (t.flags |= 8192))
            : Se(t),
          (a = t.updateQueue),
          a !== null && qi(t, a.retryQueue),
          (a = null),
          e !== null &&
            e.memoizedState !== null &&
            e.memoizedState.cachePool !== null &&
            (a = e.memoizedState.cachePool.pool),
          (r = null),
          t.memoizedState !== null &&
            t.memoizedState.cachePool !== null &&
            (r = t.memoizedState.cachePool.pool),
          r !== a && (t.flags |= 2048),
          e !== null && Ye(Zr),
          null
        );
      case 24:
        return (
          (a = null),
          e !== null && (a = e.memoizedState.cache),
          t.memoizedState.cache !== a && (t.flags |= 2048),
          Ga(Pe),
          Se(t),
          null
        );
      case 25:
        return null;
      case 30:
        return null;
    }
    throw Error(C(156, t.tag));
  }
  function Rw(e, t) {
    switch ((ff(t), t.tag)) {
      case 1:
        return (e = t.flags), e & 65536 ? ((t.flags = (e & -65537) | 128), t) : null;
      case 3:
        return (
          Ga(Pe),
          ln(),
          (e = t.flags),
          (e & 65536) !== 0 && (e & 128) === 0 ? ((t.flags = (e & -65537) | 128), t) : null
        );
      case 26:
      case 27:
      case 5:
        return fs(t), null;
      case 31:
        if (t.memoizedState !== null) {
          if ((Mt(t), t.alternate === null)) throw Error(C(340));
          eo();
        }
        return (e = t.flags), e & 65536 ? ((t.flags = (e & -65537) | 128), t) : null;
      case 13:
        if ((Mt(t), (e = t.memoizedState), e !== null && e.dehydrated !== null)) {
          if (t.alternate === null) throw Error(C(340));
          eo();
        }
        return (e = t.flags), e & 65536 ? ((t.flags = (e & -65537) | 128), t) : null;
      case 19:
        return Ye(Ae), null;
      case 4:
        return ln(), null;
      case 10:
        return Ga(t.type), null;
      case 22:
      case 23:
        return (
          Mt(t),
          yf(),
          e !== null && Ye(Zr),
          (e = t.flags),
          e & 65536 ? ((t.flags = (e & -65537) | 128), t) : null
        );
      case 24:
        return Ga(Pe), null;
      case 25:
        return null;
      default:
        return null;
    }
  }
  function Q0(e, t) {
    switch ((ff(t), t.tag)) {
      case 3:
        Ga(Pe), ln();
        break;
      case 26:
      case 27:
      case 5:
        fs(t);
        break;
      case 4:
        ln();
        break;
      case 31:
        t.memoizedState !== null && Mt(t);
        break;
      case 13:
        Mt(t);
        break;
      case 19:
        Ye(Ae);
        break;
      case 10:
        Ga(t.type);
        break;
      case 22:
      case 23:
        Mt(t), yf(), e !== null && Ye(Zr);
        break;
      case 24:
        Ga(Pe);
    }
  }
  function ei(e, t) {
    try {
      var a = t.updateQueue,
        r = a !== null ? a.lastEffect : null;
      if (r !== null) {
        var o = r.next;
        a = o;
        do {
          if ((a.tag & e) === e) {
            r = void 0;
            var n = a.create,
              l = a.inst;
            (r = n()), (l.destroy = r);
          }
          a = a.next;
        } while (a !== o);
      }
    } catch (i) {
      ce(t, t.return, i);
    }
  }
  function Tr(e, t, a) {
    try {
      var r = t.updateQueue,
        o = r !== null ? r.lastEffect : null;
      if (o !== null) {
        var n = o.next;
        r = n;
        do {
          if ((r.tag & e) === e) {
            var l = r.inst,
              i = l.destroy;
            if (i !== void 0) {
              (l.destroy = void 0), (o = t);
              var s = a,
                u = i;
              try {
                u();
              } catch (d) {
                ce(o, s, d);
              }
            }
          }
          r = r.next;
        } while (r !== n);
      }
    } catch (d) {
      ce(t, t.return, d);
    }
  }
  function K0(e) {
    var t = e.updateQueue;
    if (t !== null) {
      var a = e.stateNode;
      try {
        n0(t, a);
      } catch (r) {
        ce(e, e.return, r);
      }
    }
  }
  function Z0(e, t, a) {
    (a.props = oo(e.type, e.memoizedProps)), (a.state = e.memoizedState);
    try {
      a.componentWillUnmount();
    } catch (r) {
      ce(e, t, r);
    }
  }
  function Il(e, t) {
    try {
      var a = e.ref;
      if (a !== null) {
        switch (e.tag) {
          case 26:
          case 27:
          case 5:
            var r = e.stateNode;
            break;
          case 30:
            r = e.stateNode;
            break;
          default:
            r = e.stateNode;
        }
        typeof a == 'function' ? (e.refCleanup = a(r)) : (a.current = r);
      }
    } catch (o) {
      ce(e, t, o);
    }
  }
  function va(e, t) {
    var a = e.ref,
      r = e.refCleanup;
    if (a !== null)
      if (typeof r == 'function')
        try {
          r();
        } catch (o) {
          ce(e, t, o);
        } finally {
          (e.refCleanup = null), (e = e.alternate), e != null && (e.refCleanup = null);
        }
      else if (typeof a == 'function')
        try {
          a(null);
        } catch (o) {
          ce(e, t, o);
        }
      else a.current = null;
  }
  function J0(e) {
    var t = e.type,
      a = e.memoizedProps,
      r = e.stateNode;
    try {
      e: switch (t) {
        case 'button':
        case 'input':
        case 'select':
        case 'textarea':
          a.autoFocus && r.focus();
          break e;
        case 'img':
          a.src ? (r.src = a.src) : a.srcSet && (r.srcset = a.srcSet);
      }
    } catch (o) {
      ce(e, e.return, o);
    }
  }
  function Vc(e, t, a) {
    try {
      var r = e.stateNode;
      jw(r, e.type, a, t), (r[St] = t);
    } catch (o) {
      ce(e, e.return, o);
    }
  }
  function $0(e) {
    return (
      e.tag === 5 || e.tag === 3 || e.tag === 26 || (e.tag === 27 && Pr(e.type)) || e.tag === 4
    );
  }
  function jc(e) {
    e: for (;;) {
      for (; e.sibling === null; ) {
        if (e.return === null || $0(e.return)) return null;
        e = e.return;
      }
      for (
        e.sibling.return = e.return, e = e.sibling;
        e.tag !== 5 && e.tag !== 6 && e.tag !== 18;

      ) {
        if ((e.tag === 27 && Pr(e.type)) || e.flags & 2 || e.child === null || e.tag === 4)
          continue e;
        (e.child.return = e), (e = e.child);
      }
      if (!(e.flags & 2)) return e.stateNode;
    }
  }
  function Dd(e, t, a) {
    var r = e.tag;
    if (r === 5 || r === 6)
      (e = e.stateNode),
        t
          ? (a.nodeType === 9
              ? a.body
              : a.nodeName === 'HTML'
                ? a.ownerDocument.body
                : a
            ).insertBefore(e, t)
          : ((t = a.nodeType === 9 ? a.body : a.nodeName === 'HTML' ? a.ownerDocument.body : a),
            t.appendChild(e),
            (a = a._reactRootContainer),
            a != null || t.onclick !== null || (t.onclick = za));
    else if (
      r !== 4 &&
      (r === 27 && Pr(e.type) && ((a = e.stateNode), (t = null)), (e = e.child), e !== null)
    )
      for (Dd(e, t, a), e = e.sibling; e !== null; ) Dd(e, t, a), (e = e.sibling);
  }
  function _s(e, t, a) {
    var r = e.tag;
    if (r === 5 || r === 6) (e = e.stateNode), t ? a.insertBefore(e, t) : a.appendChild(e);
    else if (r !== 4 && (r === 27 && Pr(e.type) && (a = e.stateNode), (e = e.child), e !== null))
      for (_s(e, t, a), e = e.sibling; e !== null; ) _s(e, t, a), (e = e.sibling);
  }
  function ev(e) {
    var t = e.stateNode,
      a = e.memoizedProps;
    try {
      for (var r = e.type, o = t.attributes; o.length; ) t.removeAttributeNode(o[0]);
      rt(t, r, a), (t[et] = e), (t[St] = a);
    } catch (n) {
      ce(e, e.return, n);
    }
  }
  var Na = !1,
    Oe = !1,
    Yc = !1,
    Dg = typeof WeakSet == 'function' ? WeakSet : Set,
    Ve = null;
  function _w(e, t) {
    if (((e = e.containerInfo), (zd = Us), (e = Fy(e)), lf(e))) {
      if ('selectionStart' in e) var a = { start: e.selectionStart, end: e.selectionEnd };
      else
        e: {
          a = ((a = e.ownerDocument) && a.defaultView) || window;
          var r = a.getSelection && a.getSelection();
          if (r && r.rangeCount !== 0) {
            a = r.anchorNode;
            var o = r.anchorOffset,
              n = r.focusNode;
            r = r.focusOffset;
            try {
              a.nodeType, n.nodeType;
            } catch {
              a = null;
              break e;
            }
            var l = 0,
              i = -1,
              s = -1,
              u = 0,
              d = 0,
              c = e,
              f = null;
            t: for (;;) {
              for (
                var h;
                c !== a || (o !== 0 && c.nodeType !== 3) || (i = l + o),
                  c !== n || (r !== 0 && c.nodeType !== 3) || (s = l + r),
                  c.nodeType === 3 && (l += c.nodeValue.length),
                  (h = c.firstChild) !== null;

              )
                (f = c), (c = h);
              for (;;) {
                if (c === e) break t;
                if (
                  (f === a && ++u === o && (i = l),
                  f === n && ++d === r && (s = l),
                  (h = c.nextSibling) !== null)
                )
                  break;
                (c = f), (f = c.parentNode);
              }
              c = h;
            }
            a = i === -1 || s === -1 ? null : { start: i, end: s };
          } else a = null;
        }
      a = a || { start: 0, end: 0 };
    } else a = null;
    for (Fd = { focusedElem: e, selectionRange: a }, Us = !1, Ve = t; Ve !== null; )
      if (((t = Ve), (e = t.child), (t.subtreeFlags & 1028) !== 0 && e !== null))
        (e.return = t), (Ve = e);
      else
        for (; Ve !== null; ) {
          switch (((t = Ve), (n = t.alternate), (e = t.flags), t.tag)) {
            case 0:
              if (
                (e & 4) !== 0 &&
                ((e = t.updateQueue), (e = e !== null ? e.events : null), e !== null)
              )
                for (a = 0; a < e.length; a++) (o = e[a]), (o.ref.impl = o.nextImpl);
              break;
            case 11:
            case 15:
              break;
            case 1:
              if ((e & 1024) !== 0 && n !== null) {
                (e = void 0),
                  (a = t),
                  (o = n.memoizedProps),
                  (n = n.memoizedState),
                  (r = a.stateNode);
                try {
                  var v = oo(a.type, o);
                  (e = r.getSnapshotBeforeUpdate(v, n)),
                    (r.__reactInternalSnapshotBeforeUpdate = e);
                } catch (x) {
                  ce(a, a.return, x);
                }
              }
              break;
            case 3:
              if ((e & 1024) !== 0) {
                if (((e = t.stateNode.containerInfo), (a = e.nodeType), a === 9)) Gd(e);
                else if (a === 1)
                  switch (e.nodeName) {
                    case 'HEAD':
                    case 'HTML':
                    case 'BODY':
                      Gd(e);
                      break;
                    default:
                      e.textContent = '';
                  }
              }
              break;
            case 5:
            case 26:
            case 27:
            case 6:
            case 4:
            case 17:
              break;
            default:
              if ((e & 1024) !== 0) throw Error(C(163));
          }
          if (((e = t.sibling), e !== null)) {
            (e.return = t.return), (Ve = e);
            break;
          }
          Ve = t.return;
        }
  }
  function tv(e, t, a) {
    var r = a.flags;
    switch (a.tag) {
      case 0:
      case 11:
      case 15:
        Pa(e, a), r & 4 && ei(5, a);
        break;
      case 1:
        if ((Pa(e, a), r & 4))
          if (((e = a.stateNode), t === null))
            try {
              e.componentDidMount();
            } catch (l) {
              ce(a, a.return, l);
            }
          else {
            var o = oo(a.type, t.memoizedProps);
            t = t.memoizedState;
            try {
              e.componentDidUpdate(o, t, e.__reactInternalSnapshotBeforeUpdate);
            } catch (l) {
              ce(a, a.return, l);
            }
          }
        r & 64 && K0(a), r & 512 && Il(a, a.return);
        break;
      case 3:
        if ((Pa(e, a), r & 64 && ((e = a.updateQueue), e !== null))) {
          if (((t = null), a.child !== null))
            switch (a.child.tag) {
              case 27:
              case 5:
                t = a.child.stateNode;
                break;
              case 1:
                t = a.child.stateNode;
            }
          try {
            n0(e, t);
          } catch (l) {
            ce(a, a.return, l);
          }
        }
        break;
      case 27:
        t === null && r & 4 && ev(a);
      case 26:
      case 5:
        Pa(e, a), t === null && r & 4 && J0(a), r & 512 && Il(a, a.return);
        break;
      case 12:
        Pa(e, a);
        break;
      case 31:
        Pa(e, a), r & 4 && ov(e, a);
        break;
      case 13:
        Pa(e, a),
          r & 4 && nv(e, a),
          r & 64 &&
            ((e = a.memoizedState),
            e !== null && ((e = e.dehydrated), e !== null && ((a = Pw.bind(null, a)), $w(e, a))));
        break;
      case 22:
        if (((r = a.memoizedState !== null || Na), !r)) {
          (t = (t !== null && t.memoizedState !== null) || Oe), (o = Na);
          var n = Oe;
          (Na = r),
            (Oe = t) && !n ? Ba(e, a, (a.subtreeFlags & 8772) !== 0) : Pa(e, a),
            (Na = o),
            (Oe = n);
        }
        break;
      case 30:
        break;
      default:
        Pa(e, a);
    }
  }
  function av(e) {
    var t = e.alternate;
    t !== null && ((e.alternate = null), av(t)),
      (e.child = null),
      (e.deletions = null),
      (e.sibling = null),
      e.tag === 5 && ((t = e.stateNode), t !== null && ef(t)),
      (e.stateNode = null),
      (e.return = null),
      (e.dependencies = null),
      (e.memoizedProps = null),
      (e.memoizedState = null),
      (e.pendingProps = null),
      (e.stateNode = null),
      (e.updateQueue = null);
  }
  var Re = null,
    vt = !1;
  function Oa(e, t, a) {
    for (a = a.child; a !== null; ) rv(e, t, a), (a = a.sibling);
  }
  function rv(e, t, a) {
    if (Dt && typeof Dt.onCommitFiberUnmount == 'function')
      try {
        Dt.onCommitFiberUnmount(Xl, a);
      } catch {}
    switch (a.tag) {
      case 26:
        Oe || va(a, t),
          Oa(e, t, a),
          a.memoizedState
            ? a.memoizedState.count--
            : a.stateNode && ((a = a.stateNode), a.parentNode.removeChild(a));
        break;
      case 27:
        Oe || va(a, t);
        var r = Re,
          o = vt;
        Pr(a.type) && ((Re = a.stateNode), (vt = !1)),
          Oa(e, t, a),
          Al(a.stateNode),
          (Re = r),
          (vt = o);
        break;
      case 5:
        Oe || va(a, t);
      case 6:
        if (((r = Re), (o = vt), (Re = null), Oa(e, t, a), (Re = r), (vt = o), Re !== null))
          if (vt)
            try {
              (Re.nodeType === 9
                ? Re.body
                : Re.nodeName === 'HTML'
                  ? Re.ownerDocument.body
                  : Re
              ).removeChild(a.stateNode);
            } catch (n) {
              ce(a, t, n);
            }
          else
            try {
              Re.removeChild(a.stateNode);
            } catch (n) {
              ce(a, t, n);
            }
        break;
      case 18:
        Re !== null &&
          (vt
            ? ((e = Re),
              Wg(
                e.nodeType === 9 ? e.body : e.nodeName === 'HTML' ? e.ownerDocument.body : e,
                a.stateNode
              ),
              yn(e))
            : Wg(Re, a.stateNode));
        break;
      case 4:
        (r = Re),
          (o = vt),
          (Re = a.stateNode.containerInfo),
          (vt = !0),
          Oa(e, t, a),
          (Re = r),
          (vt = o);
        break;
      case 0:
      case 11:
      case 14:
      case 15:
        Tr(2, a, t), Oe || Tr(4, a, t), Oa(e, t, a);
        break;
      case 1:
        Oe ||
          (va(a, t), (r = a.stateNode), typeof r.componentWillUnmount == 'function' && Z0(a, t, r)),
          Oa(e, t, a);
        break;
      case 21:
        Oa(e, t, a);
        break;
      case 22:
        (Oe = (r = Oe) || a.memoizedState !== null), Oa(e, t, a), (Oe = r);
        break;
      default:
        Oa(e, t, a);
    }
  }
  function ov(e, t) {
    if (
      t.memoizedState === null &&
      ((e = t.alternate), e !== null && ((e = e.memoizedState), e !== null))
    ) {
      e = e.dehydrated;
      try {
        yn(e);
      } catch (a) {
        ce(t, t.return, a);
      }
    }
  }
  function nv(e, t) {
    if (
      t.memoizedState === null &&
      ((e = t.alternate),
      e !== null && ((e = e.memoizedState), e !== null && ((e = e.dehydrated), e !== null)))
    )
      try {
        yn(e);
      } catch (a) {
        ce(t, t.return, a);
      }
  }
  function Iw(e) {
    switch (e.tag) {
      case 31:
      case 13:
      case 19:
        var t = e.stateNode;
        return t === null && (t = e.stateNode = new Dg()), t;
      case 22:
        return (
          (e = e.stateNode), (t = e._retryCache), t === null && (t = e._retryCache = new Dg()), t
        );
      default:
        throw Error(C(435, e.tag));
    }
  }
  function Gi(e, t) {
    var a = Iw(e);
    t.forEach(function (r) {
      if (!a.has(r)) {
        a.add(r);
        var o = Bw.bind(null, e, r);
        r.then(o, o);
      }
    });
  }
  function gt(e, t) {
    var a = t.deletions;
    if (a !== null)
      for (var r = 0; r < a.length; r++) {
        var o = a[r],
          n = e,
          l = t,
          i = l;
        e: for (; i !== null; ) {
          switch (i.tag) {
            case 27:
              if (Pr(i.type)) {
                (Re = i.stateNode), (vt = !1);
                break e;
              }
              break;
            case 5:
              (Re = i.stateNode), (vt = !1);
              break e;
            case 3:
            case 4:
              (Re = i.stateNode.containerInfo), (vt = !0);
              break e;
          }
          i = i.return;
        }
        if (Re === null) throw Error(C(160));
        rv(n, l, o),
          (Re = null),
          (vt = !1),
          (n = o.alternate),
          n !== null && (n.return = null),
          (o.return = null);
      }
    if (t.subtreeFlags & 13886) for (t = t.child; t !== null; ) lv(t, e), (t = t.sibling);
  }
  var la = null;
  function lv(e, t) {
    var a = e.alternate,
      r = e.flags;
    switch (e.tag) {
      case 0:
      case 11:
      case 14:
      case 15:
        gt(t, e), yt(e), r & 4 && (Tr(3, e, e.return), ei(3, e), Tr(5, e, e.return));
        break;
      case 1:
        gt(t, e),
          yt(e),
          r & 512 && (Oe || a === null || va(a, a.return)),
          r & 64 &&
            Na &&
            ((e = e.updateQueue),
            e !== null &&
              ((r = e.callbacks),
              r !== null &&
                ((a = e.shared.hiddenCallbacks),
                (e.shared.hiddenCallbacks = a === null ? r : a.concat(r)))));
        break;
      case 26:
        var o = la;
        if ((gt(t, e), yt(e), r & 512 && (Oe || a === null || va(a, a.return)), r & 4)) {
          var n = a !== null ? a.memoizedState : null;
          if (((r = e.memoizedState), a === null))
            if (r === null)
              if (e.stateNode === null) {
                e: {
                  (r = e.type), (a = e.memoizedProps), (o = o.ownerDocument || o);
                  t: switch (r) {
                    case 'title':
                      (n = o.getElementsByTagName('title')[0]),
                        (!n ||
                          n[Kl] ||
                          n[et] ||
                          n.namespaceURI === 'http://www.w3.org/2000/svg' ||
                          n.hasAttribute('itemprop')) &&
                          ((n = o.createElement(r)),
                          o.head.insertBefore(n, o.querySelector('head > title'))),
                        rt(n, r, a),
                        (n[et] = e),
                        je(n),
                        (r = n);
                      break e;
                    case 'link':
                      var l = ty('link', 'href', o).get(r + (a.href || ''));
                      if (l) {
                        for (var i = 0; i < l.length; i++)
                          if (
                            ((n = l[i]),
                            n.getAttribute('href') ===
                              (a.href == null || a.href === '' ? null : a.href) &&
                              n.getAttribute('rel') === (a.rel == null ? null : a.rel) &&
                              n.getAttribute('title') === (a.title == null ? null : a.title) &&
                              n.getAttribute('crossorigin') ===
                                (a.crossOrigin == null ? null : a.crossOrigin))
                          ) {
                            l.splice(i, 1);
                            break t;
                          }
                      }
                      (n = o.createElement(r)), rt(n, r, a), o.head.appendChild(n);
                      break;
                    case 'meta':
                      if ((l = ty('meta', 'content', o).get(r + (a.content || '')))) {
                        for (i = 0; i < l.length; i++)
                          if (
                            ((n = l[i]),
                            n.getAttribute('content') ===
                              (a.content == null ? null : '' + a.content) &&
                              n.getAttribute('name') === (a.name == null ? null : a.name) &&
                              n.getAttribute('property') ===
                                (a.property == null ? null : a.property) &&
                              n.getAttribute('http-equiv') ===
                                (a.httpEquiv == null ? null : a.httpEquiv) &&
                              n.getAttribute('charset') === (a.charSet == null ? null : a.charSet))
                          ) {
                            l.splice(i, 1);
                            break t;
                          }
                      }
                      (n = o.createElement(r)), rt(n, r, a), o.head.appendChild(n);
                      break;
                    default:
                      throw Error(C(468, r));
                  }
                  (n[et] = e), je(n), (r = n);
                }
                e.stateNode = r;
              } else ay(o, e.type, e.stateNode);
            else e.stateNode = ey(o, r, e.memoizedProps);
          else
            n !== r
              ? (n === null
                  ? a.stateNode !== null && ((a = a.stateNode), a.parentNode.removeChild(a))
                  : n.count--,
                r === null ? ay(o, e.type, e.stateNode) : ey(o, r, e.memoizedProps))
              : r === null && e.stateNode !== null && Vc(e, e.memoizedProps, a.memoizedProps);
        }
        break;
      case 27:
        gt(t, e),
          yt(e),
          r & 512 && (Oe || a === null || va(a, a.return)),
          a !== null && r & 4 && Vc(e, e.memoizedProps, a.memoizedProps);
        break;
      case 5:
        if ((gt(t, e), yt(e), r & 512 && (Oe || a === null || va(a, a.return)), e.flags & 32)) {
          o = e.stateNode;
          try {
            un(o, '');
          } catch (v) {
            ce(e, e.return, v);
          }
        }
        r & 4 &&
          e.stateNode != null &&
          ((o = e.memoizedProps), Vc(e, o, a !== null ? a.memoizedProps : o)),
          r & 1024 && (Yc = !0);
        break;
      case 6:
        if ((gt(t, e), yt(e), r & 4)) {
          if (e.stateNode === null) throw Error(C(162));
          (r = e.memoizedProps), (a = e.stateNode);
          try {
            a.nodeValue = r;
          } catch (v) {
            ce(e, e.return, v);
          }
        }
        break;
      case 3:
        if (
          ((is = null),
          (o = la),
          (la = Os(t.containerInfo)),
          gt(t, e),
          (la = o),
          yt(e),
          r & 4 && a !== null && a.memoizedState.isDehydrated)
        )
          try {
            yn(t.containerInfo);
          } catch (v) {
            ce(e, e.return, v);
          }
        Yc && ((Yc = !1), iv(e));
        break;
      case 4:
        (r = la), (la = Os(e.stateNode.containerInfo)), gt(t, e), yt(e), (la = r);
        break;
      case 12:
        gt(t, e), yt(e);
        break;
      case 31:
        gt(t, e),
          yt(e),
          r & 4 && ((r = e.updateQueue), r !== null && ((e.updateQueue = null), Gi(e, r)));
        break;
      case 13:
        gt(t, e),
          yt(e),
          e.child.flags & 8192 &&
            (e.memoizedState !== null) != (a !== null && a.memoizedState !== null) &&
            (Ks = Tt()),
          r & 4 && ((r = e.updateQueue), r !== null && ((e.updateQueue = null), Gi(e, r)));
        break;
      case 22:
        o = e.memoizedState !== null;
        var s = a !== null && a.memoizedState !== null,
          u = Na,
          d = Oe;
        if (((Na = u || o), (Oe = d || s), gt(t, e), (Oe = d), (Na = u), yt(e), r & 8192))
          e: for (
            t = e.stateNode,
              t._visibility = o ? t._visibility & -2 : t._visibility | 1,
              o && (a === null || s || Na || Oe || Wr(e)),
              a = null,
              t = e;
            ;

          ) {
            if (t.tag === 5 || t.tag === 26) {
              if (a === null) {
                s = a = t;
                try {
                  if (((n = s.stateNode), o))
                    (l = n.style),
                      typeof l.setProperty == 'function'
                        ? l.setProperty('display', 'none', 'important')
                        : (l.display = 'none');
                  else {
                    i = s.stateNode;
                    var c = s.memoizedProps.style,
                      f = c != null && c.hasOwnProperty('display') ? c.display : null;
                    i.style.display = f == null || typeof f == 'boolean' ? '' : ('' + f).trim();
                  }
                } catch (v) {
                  ce(s, s.return, v);
                }
              }
            } else if (t.tag === 6) {
              if (a === null) {
                s = t;
                try {
                  s.stateNode.nodeValue = o ? '' : s.memoizedProps;
                } catch (v) {
                  ce(s, s.return, v);
                }
              }
            } else if (t.tag === 18) {
              if (a === null) {
                s = t;
                try {
                  var h = s.stateNode;
                  o ? Qg(h, !0) : Qg(s.stateNode, !1);
                } catch (v) {
                  ce(s, s.return, v);
                }
              }
            } else if (
              ((t.tag !== 22 && t.tag !== 23) || t.memoizedState === null || t === e) &&
              t.child !== null
            ) {
              (t.child.return = t), (t = t.child);
              continue;
            }
            if (t === e) break e;
            for (; t.sibling === null; ) {
              if (t.return === null || t.return === e) break e;
              a === t && (a = null), (t = t.return);
            }
            a === t && (a = null), (t.sibling.return = t.return), (t = t.sibling);
          }
        r & 4 &&
          ((r = e.updateQueue),
          r !== null && ((a = r.retryQueue), a !== null && ((r.retryQueue = null), Gi(e, a))));
        break;
      case 19:
        gt(t, e),
          yt(e),
          r & 4 && ((r = e.updateQueue), r !== null && ((e.updateQueue = null), Gi(e, r)));
        break;
      case 30:
        break;
      case 21:
        break;
      default:
        gt(t, e), yt(e);
    }
  }
  function yt(e) {
    var t = e.flags;
    if (t & 2) {
      try {
        for (var a, r = e.return; r !== null; ) {
          if ($0(r)) {
            a = r;
            break;
          }
          r = r.return;
        }
        if (a == null) throw Error(C(160));
        switch (a.tag) {
          case 27:
            var o = a.stateNode,
              n = jc(e);
            _s(e, n, o);
            break;
          case 5:
            var l = a.stateNode;
            a.flags & 32 && (un(l, ''), (a.flags &= -33));
            var i = jc(e);
            _s(e, i, l);
            break;
          case 3:
          case 4:
            var s = a.stateNode.containerInfo,
              u = jc(e);
            Dd(e, u, s);
            break;
          default:
            throw Error(C(161));
        }
      } catch (d) {
        ce(e, e.return, d);
      }
      e.flags &= -3;
    }
    t & 4096 && (e.flags &= -4097);
  }
  function iv(e) {
    if (e.subtreeFlags & 1024)
      for (e = e.child; e !== null; ) {
        var t = e;
        iv(t), t.tag === 5 && t.flags & 1024 && t.stateNode.reset(), (e = e.sibling);
      }
  }
  function Pa(e, t) {
    if (t.subtreeFlags & 8772)
      for (t = t.child; t !== null; ) tv(e, t.alternate, t), (t = t.sibling);
  }
  function Wr(e) {
    for (e = e.child; e !== null; ) {
      var t = e;
      switch (t.tag) {
        case 0:
        case 11:
        case 14:
        case 15:
          Tr(4, t, t.return), Wr(t);
          break;
        case 1:
          va(t, t.return);
          var a = t.stateNode;
          typeof a.componentWillUnmount == 'function' && Z0(t, t.return, a), Wr(t);
          break;
        case 27:
          Al(t.stateNode);
        case 26:
        case 5:
          va(t, t.return), Wr(t);
          break;
        case 22:
          t.memoizedState === null && Wr(t);
          break;
        case 30:
          Wr(t);
          break;
        default:
          Wr(t);
      }
      e = e.sibling;
    }
  }
  function Ba(e, t, a) {
    for (a = a && (t.subtreeFlags & 8772) !== 0, t = t.child; t !== null; ) {
      var r = t.alternate,
        o = e,
        n = t,
        l = n.flags;
      switch (n.tag) {
        case 0:
        case 11:
        case 15:
          Ba(o, n, a), ei(4, n);
          break;
        case 1:
          if ((Ba(o, n, a), (r = n), (o = r.stateNode), typeof o.componentDidMount == 'function'))
            try {
              o.componentDidMount();
            } catch (u) {
              ce(r, r.return, u);
            }
          if (((r = n), (o = r.updateQueue), o !== null)) {
            var i = r.stateNode;
            try {
              var s = o.shared.hiddenCallbacks;
              if (s !== null)
                for (o.shared.hiddenCallbacks = null, o = 0; o < s.length; o++) o0(s[o], i);
            } catch (u) {
              ce(r, r.return, u);
            }
          }
          a && l & 64 && K0(n), Il(n, n.return);
          break;
        case 27:
          ev(n);
        case 26:
        case 5:
          Ba(o, n, a), a && r === null && l & 4 && J0(n), Il(n, n.return);
          break;
        case 12:
          Ba(o, n, a);
          break;
        case 31:
          Ba(o, n, a), a && l & 4 && ov(o, n);
          break;
        case 13:
          Ba(o, n, a), a && l & 4 && nv(o, n);
          break;
        case 22:
          n.memoizedState === null && Ba(o, n, a), Il(n, n.return);
          break;
        case 30:
          break;
        default:
          Ba(o, n, a);
      }
      t = t.sibling;
    }
  }
  function Df(e, t) {
    var a = null;
    e !== null &&
      e.memoizedState !== null &&
      e.memoizedState.cachePool !== null &&
      (a = e.memoizedState.cachePool.pool),
      (e = null),
      t.memoizedState !== null &&
        t.memoizedState.cachePool !== null &&
        (e = t.memoizedState.cachePool.pool),
      e !== a && (e != null && e.refCount++, a != null && Jl(a));
  }
  function Of(e, t) {
    (e = null),
      t.alternate !== null && (e = t.alternate.memoizedState.cache),
      (t = t.memoizedState.cache),
      t !== e && (t.refCount++, e != null && Jl(e));
  }
  function na(e, t, a, r) {
    if (t.subtreeFlags & 10256) for (t = t.child; t !== null; ) sv(e, t, a, r), (t = t.sibling);
  }
  function sv(e, t, a, r) {
    var o = t.flags;
    switch (t.tag) {
      case 0:
      case 11:
      case 15:
        na(e, t, a, r), o & 2048 && ei(9, t);
        break;
      case 1:
        na(e, t, a, r);
        break;
      case 3:
        na(e, t, a, r),
          o & 2048 &&
            ((e = null),
            t.alternate !== null && (e = t.alternate.memoizedState.cache),
            (t = t.memoizedState.cache),
            t !== e && (t.refCount++, e != null && Jl(e)));
        break;
      case 12:
        if (o & 2048) {
          na(e, t, a, r), (e = t.stateNode);
          try {
            var n = t.memoizedProps,
              l = n.id,
              i = n.onPostCommit;
            typeof i == 'function' &&
              i(l, t.alternate === null ? 'mount' : 'update', e.passiveEffectDuration, -0);
          } catch (s) {
            ce(t, t.return, s);
          }
        } else na(e, t, a, r);
        break;
      case 31:
        na(e, t, a, r);
        break;
      case 13:
        na(e, t, a, r);
        break;
      case 23:
        break;
      case 22:
        (n = t.stateNode),
          (l = t.alternate),
          t.memoizedState !== null
            ? n._visibility & 2
              ? na(e, t, a, r)
              : kl(e, t)
            : n._visibility & 2
              ? na(e, t, a, r)
              : ((n._visibility |= 2), Ho(e, t, a, r, (t.subtreeFlags & 10256) !== 0 || !1)),
          o & 2048 && Df(l, t);
        break;
      case 24:
        na(e, t, a, r), o & 2048 && Of(t.alternate, t);
        break;
      default:
        na(e, t, a, r);
    }
  }
  function Ho(e, t, a, r, o) {
    for (o = o && ((t.subtreeFlags & 10256) !== 0 || !1), t = t.child; t !== null; ) {
      var n = e,
        l = t,
        i = a,
        s = r,
        u = l.flags;
      switch (l.tag) {
        case 0:
        case 11:
        case 15:
          Ho(n, l, i, s, o), ei(8, l);
          break;
        case 23:
          break;
        case 22:
          var d = l.stateNode;
          l.memoizedState !== null
            ? d._visibility & 2
              ? Ho(n, l, i, s, o)
              : kl(n, l)
            : ((d._visibility |= 2), Ho(n, l, i, s, o)),
            o && u & 2048 && Df(l.alternate, l);
          break;
        case 24:
          Ho(n, l, i, s, o), o && u & 2048 && Of(l.alternate, l);
          break;
        default:
          Ho(n, l, i, s, o);
      }
      t = t.sibling;
    }
  }
  function kl(e, t) {
    if (t.subtreeFlags & 10256)
      for (t = t.child; t !== null; ) {
        var a = e,
          r = t,
          o = r.flags;
        switch (r.tag) {
          case 22:
            kl(a, r), o & 2048 && Df(r.alternate, r);
            break;
          case 24:
            kl(a, r), o & 2048 && Of(r.alternate, r);
            break;
          default:
            kl(a, r);
        }
        t = t.sibling;
      }
  }
  var vl = 8192;
  function No(e, t, a) {
    if (e.subtreeFlags & vl) for (e = e.child; e !== null; ) uv(e, t, a), (e = e.sibling);
  }
  function uv(e, t, a) {
    switch (e.tag) {
      case 26:
        No(e, t, a),
          e.flags & vl && e.memoizedState !== null && dR(a, la, e.memoizedState, e.memoizedProps);
        break;
      case 5:
        No(e, t, a);
        break;
      case 3:
      case 4:
        var r = la;
        (la = Os(e.stateNode.containerInfo)), No(e, t, a), (la = r);
        break;
      case 22:
        e.memoizedState === null &&
          ((r = e.alternate),
          r !== null && r.memoizedState !== null
            ? ((r = vl), (vl = 16777216), No(e, t, a), (vl = r))
            : No(e, t, a));
        break;
      default:
        No(e, t, a);
    }
  }
  function cv(e) {
    var t = e.alternate;
    if (t !== null && ((e = t.child), e !== null)) {
      t.child = null;
      do (t = e.sibling), (e.sibling = null), (e = t);
      while (e !== null);
    }
  }
  function dl(e) {
    var t = e.deletions;
    if ((e.flags & 16) !== 0) {
      if (t !== null)
        for (var a = 0; a < t.length; a++) {
          var r = t[a];
          (Ve = r), fv(r, e);
        }
      cv(e);
    }
    if (e.subtreeFlags & 10256) for (e = e.child; e !== null; ) dv(e), (e = e.sibling);
  }
  function dv(e) {
    switch (e.tag) {
      case 0:
      case 11:
      case 15:
        dl(e), e.flags & 2048 && Tr(9, e, e.return);
        break;
      case 3:
        dl(e);
        break;
      case 12:
        dl(e);
        break;
      case 22:
        var t = e.stateNode;
        e.memoizedState !== null && t._visibility & 2 && (e.return === null || e.return.tag !== 13)
          ? ((t._visibility &= -3), ns(e))
          : dl(e);
        break;
      default:
        dl(e);
    }
  }
  function ns(e) {
    var t = e.deletions;
    if ((e.flags & 16) !== 0) {
      if (t !== null)
        for (var a = 0; a < t.length; a++) {
          var r = t[a];
          (Ve = r), fv(r, e);
        }
      cv(e);
    }
    for (e = e.child; e !== null; ) {
      switch (((t = e), t.tag)) {
        case 0:
        case 11:
        case 15:
          Tr(8, t, t.return), ns(t);
          break;
        case 22:
          (a = t.stateNode), a._visibility & 2 && ((a._visibility &= -3), ns(t));
          break;
        default:
          ns(t);
      }
      e = e.sibling;
    }
  }
  function fv(e, t) {
    for (; Ve !== null; ) {
      var a = Ve;
      switch (a.tag) {
        case 0:
        case 11:
        case 15:
          Tr(8, a, t);
          break;
        case 23:
        case 22:
          if (a.memoizedState !== null && a.memoizedState.cachePool !== null) {
            var r = a.memoizedState.cachePool.pool;
            r != null && r.refCount++;
          }
          break;
        case 24:
          Jl(a.memoizedState.cache);
      }
      if (((r = a.child), r !== null)) (r.return = a), (Ve = r);
      else
        e: for (a = e; Ve !== null; ) {
          r = Ve;
          var o = r.sibling,
            n = r.return;
          if ((av(r), r === a)) {
            Ve = null;
            break e;
          }
          if (o !== null) {
            (o.return = n), (Ve = o);
            break e;
          }
          Ve = n;
        }
    }
  }
  var kw = {
      getCacheForType: function (e) {
        var t = at(Pe),
          a = t.data.get(e);
        return a === void 0 && ((a = e()), t.data.set(e, a)), a;
      },
      cacheSignal: function () {
        return at(Pe).controller.signal;
      },
    },
    Mw = typeof WeakMap == 'function' ? WeakMap : Map,
    ne = 0,
    he = null,
    Q = null,
    K = 0,
    ue = 0,
    kt = null,
    vr = !1,
    Ln = !1,
    Pf = !1,
    Qa = 0,
    Me = 0,
    Dr = 0,
    $r = 0,
    Bf = 0,
    At = 0,
    mn = 0,
    Ml = null,
    bt = null,
    Od = !1,
    Ks = 0,
    mv = 0,
    Is = 1 / 0,
    ks = null,
    Rr = null,
    He = 0,
    _r = null,
    pn = null,
    Va = 0,
    Pd = 0,
    Bd = null,
    pv = null,
    El = 0,
    Ud = null;
  function Pt() {
    return (ne & 2) !== 0 && K !== 0 ? K & -K : N.T !== null ? Nf() : Cy();
  }
  function hv() {
    if (At === 0)
      if ((K & 536870912) === 0 || J) {
        var e = Di;
        (Di <<= 1), (Di & 3932160) === 0 && (Di = 262144), (At = e);
      } else At = 536870912;
    return (e = Ut.current), e !== null && (e.flags |= 32), At;
  }
  function xt(e, t, a) {
    ((e === he && (ue === 2 || ue === 9)) || e.cancelPendingCommit !== null) &&
      (hn(e, 0), br(e, K, At, !1)),
      Ql(e, a),
      ((ne & 2) === 0 || e !== he) &&
        (e === he && ((ne & 2) === 0 && ($r |= a), Me === 4 && br(e, K, At, !1)), Sa(e));
  }
  function gv(e, t, a) {
    if ((ne & 6) !== 0) throw Error(C(327));
    var r = (!a && (t & 127) === 0 && (t & e.expiredLanes) === 0) || Wl(e, t),
      o = r ? Tw(e, t) : Xc(e, t, !0),
      n = r;
    do {
      if (o === 0) {
        Ln && !r && br(e, t, 0, !1);
        break;
      } else {
        if (((a = e.current.alternate), n && !Ew(a))) {
          (o = Xc(e, t, !1)), (n = !1);
          continue;
        }
        if (o === 2) {
          if (((n = t), e.errorRecoveryDisabledLanes & n)) var l = 0;
          else (l = e.pendingLanes & -536870913), (l = l !== 0 ? l : l & 536870912 ? 536870912 : 0);
          if (l !== 0) {
            t = l;
            e: {
              var i = e;
              o = Ml;
              var s = i.current.memoizedState.isDehydrated;
              if ((s && (hn(i, l).flags |= 256), (l = Xc(i, l, !1)), l !== 2)) {
                if (Pf && !s) {
                  (i.errorRecoveryDisabledLanes |= n), ($r |= n), (o = 4);
                  break e;
                }
                (n = bt), (bt = o), n !== null && (bt === null ? (bt = n) : bt.push.apply(bt, n));
              }
              o = l;
            }
            if (((n = !1), o !== 2)) continue;
          }
        }
        if (o === 1) {
          hn(e, 0), br(e, t, 0, !0);
          break;
        }
        e: {
          switch (((r = e), (n = o), n)) {
            case 0:
            case 1:
              throw Error(C(345));
            case 4:
              if ((t & 4194048) !== t) break;
            case 6:
              br(r, t, At, !vr);
              break e;
            case 2:
              bt = null;
              break;
            case 3:
            case 5:
              break;
            default:
              throw Error(C(329));
          }
          if ((t & 62914560) === t && ((o = Ks + 300 - Tt()), 10 < o)) {
            if ((br(r, t, At, !vr), Hs(r, 0, !0) !== 0)) break e;
            (Va = t),
              (r.timeoutHandle = Bv(
                Og.bind(null, r, a, bt, ks, Od, t, At, $r, mn, vr, n, 'Throttled', -0, 0),
                o
              ));
            break e;
          }
          Og(r, a, bt, ks, Od, t, At, $r, mn, vr, n, null, -0, 0);
        }
      }
      break;
    } while (!0);
    Sa(e);
  }
  function Og(e, t, a, r, o, n, l, i, s, u, d, c, f, h) {
    if (((e.timeoutHandle = -1), (c = t.subtreeFlags), c & 8192 || (c & 16785408) === 16785408)) {
      (c = {
        stylesheets: null,
        count: 0,
        imgCount: 0,
        imgBytes: 0,
        suspenseyImages: [],
        waitingForImages: !0,
        waitingForViewTransition: !1,
        unsuspend: za,
      }),
        uv(t, n, c);
      var v = (n & 62914560) === n ? Ks - Tt() : (n & 4194048) === n ? mv - Tt() : 0;
      if (((v = fR(c, v)), v !== null)) {
        (Va = n),
          (e.cancelPendingCommit = v(Bg.bind(null, e, t, n, a, r, o, l, i, s, d, c, null, f, h))),
          br(e, n, l, !u);
        return;
      }
    }
    Bg(e, t, n, a, r, o, l, i, s);
  }
  function Ew(e) {
    for (var t = e; ; ) {
      var a = t.tag;
      if (
        (a === 0 || a === 11 || a === 15) &&
        t.flags & 16384 &&
        ((a = t.updateQueue), a !== null && ((a = a.stores), a !== null))
      )
        for (var r = 0; r < a.length; r++) {
          var o = a[r],
            n = o.getSnapshot;
          o = o.value;
          try {
            if (!Bt(n(), o)) return !1;
          } catch {
            return !1;
          }
        }
      if (((a = t.child), t.subtreeFlags & 16384 && a !== null)) (a.return = t), (t = a);
      else {
        if (t === e) break;
        for (; t.sibling === null; ) {
          if (t.return === null || t.return === e) return !0;
          t = t.return;
        }
        (t.sibling.return = t.return), (t = t.sibling);
      }
    }
    return !0;
  }
  function br(e, t, a, r) {
    (t &= ~Bf),
      (t &= ~$r),
      (e.suspendedLanes |= t),
      (e.pingedLanes &= ~t),
      r && (e.warmLanes |= t),
      (r = e.expirationTimes);
    for (var o = t; 0 < o; ) {
      var n = 31 - Ot(o),
        l = 1 << n;
      (r[n] = -1), (o &= ~l);
    }
    a !== 0 && xy(e, a, t);
  }
  function Zs() {
    return (ne & 6) === 0 ? (ti(0, !1), !1) : !0;
  }
  function Uf() {
    if (Q !== null) {
      if (ue === 0) var e = Q.return;
      else (e = Q), (Fa = uo = null), Lf(e), (rn = null), (Nl = 0), (e = Q);
      for (; e !== null; ) Q0(e.alternate, e), (e = e.return);
      Q = null;
    }
  }
  function hn(e, t) {
    var a = e.timeoutHandle;
    a !== -1 && ((e.timeoutHandle = -1), Ww(a)),
      (a = e.cancelPendingCommit),
      a !== null && ((e.cancelPendingCommit = null), a()),
      (Va = 0),
      Uf(),
      (he = e),
      (Q = a = qa(e.current, null)),
      (K = t),
      (ue = 0),
      (kt = null),
      (vr = !1),
      (Ln = Wl(e, t)),
      (Pf = !1),
      (mn = At = Bf = $r = Dr = Me = 0),
      (bt = Ml = null),
      (Od = !1),
      (t & 8) !== 0 && (t |= t & 32);
    var r = e.entangledLanes;
    if (r !== 0)
      for (e = e.entanglements, r &= t; 0 < r; ) {
        var o = 31 - Ot(r),
          n = 1 << o;
        (t |= e[o]), (r &= ~n);
      }
    return (Qa = t), Gs(), a;
  }
  function yv(e, t) {
    (G = null),
      (N.H = zl),
      t === Sn || t === js
        ? ((t = fg()), (ue = 3))
        : t === hf
          ? ((t = fg()), (ue = 4))
          : (ue =
              t === Af
                ? 8
                : t !== null && typeof t == 'object' && typeof t.then == 'function'
                  ? 6
                  : 1),
      (kt = t),
      Q === null && ((Me = 1), ws(e, Kt(t, e.current)));
  }
  function vv() {
    var e = Ut.current;
    return e === null
      ? !0
      : (K & 4194048) === K
        ? Jt === null
        : (K & 62914560) === K || (K & 536870912) !== 0
          ? e === Jt
          : !1;
  }
  function bv() {
    var e = N.H;
    return (N.H = zl), e === null ? zl : e;
  }
  function xv() {
    var e = N.A;
    return (N.A = kw), e;
  }
  function Ms() {
    (Me = 4),
      vr || ((K & 4194048) !== K && Ut.current !== null) || (Ln = !0),
      ((Dr & 134217727) === 0 && ($r & 134217727) === 0) || he === null || br(he, K, At, !1);
  }
  function Xc(e, t, a) {
    var r = ne;
    ne |= 2;
    var o = bv(),
      n = xv();
    (he !== e || K !== t) && ((ks = null), hn(e, t)), (t = !1);
    var l = Me;
    e: do
      try {
        if (ue !== 0 && Q !== null) {
          var i = Q,
            s = kt;
          switch (ue) {
            case 8:
              Uf(), (l = 6);
              break e;
            case 3:
            case 2:
            case 9:
            case 6:
              Ut.current === null && (t = !0);
              var u = ue;
              if (((ue = 0), (kt = null), Jo(e, i, s, u), a && Ln)) {
                l = 0;
                break e;
              }
              break;
            default:
              (u = ue), (ue = 0), (kt = null), Jo(e, i, s, u);
          }
        }
        Aw(), (l = Me);
        break;
      } catch (d) {
        yv(e, d);
      }
    while (!0);
    return (
      t && e.shellSuspendCounter++,
      (Fa = uo = null),
      (ne = r),
      (N.H = o),
      (N.A = n),
      Q === null && ((he = null), (K = 0), Gs()),
      l
    );
  }
  function Aw() {
    for (; Q !== null; ) Sv(Q);
  }
  function Tw(e, t) {
    var a = ne;
    ne |= 2;
    var r = bv(),
      o = xv();
    he !== e || K !== t ? ((ks = null), (Is = Tt() + 500), hn(e, t)) : (Ln = Wl(e, t));
    e: do
      try {
        if (ue !== 0 && Q !== null) {
          t = Q;
          var n = kt;
          t: switch (ue) {
            case 1:
              (ue = 0), (kt = null), Jo(e, t, n, 1);
              break;
            case 2:
            case 9:
              if (dg(n)) {
                (ue = 0), (kt = null), Pg(t);
                break;
              }
              (t = function () {
                (ue !== 2 && ue !== 9) || he !== e || (ue = 7), Sa(e);
              }),
                n.then(t, t);
              break e;
            case 3:
              ue = 7;
              break e;
            case 4:
              ue = 5;
              break e;
            case 7:
              dg(n) ? ((ue = 0), (kt = null), Pg(t)) : ((ue = 0), (kt = null), Jo(e, t, n, 7));
              break;
            case 5:
              var l = null;
              switch (Q.tag) {
                case 26:
                  l = Q.memoizedState;
                case 5:
                case 27:
                  var i = Q;
                  if (l ? Fv(l) : i.stateNode.complete) {
                    (ue = 0), (kt = null);
                    var s = i.sibling;
                    if (s !== null) Q = s;
                    else {
                      var u = i.return;
                      u !== null ? ((Q = u), Js(u)) : (Q = null);
                    }
                    break t;
                  }
              }
              (ue = 0), (kt = null), Jo(e, t, n, 5);
              break;
            case 6:
              (ue = 0), (kt = null), Jo(e, t, n, 6);
              break;
            case 8:
              Uf(), (Me = 6);
              break e;
            default:
              throw Error(C(462));
          }
        }
        Dw();
        break;
      } catch (d) {
        yv(e, d);
      }
    while (!0);
    return (
      (Fa = uo = null),
      (N.H = r),
      (N.A = o),
      (ne = a),
      Q !== null ? 0 : ((he = null), (K = 0), Gs(), Me)
    );
  }
  function Dw() {
    for (; Q !== null && !aC(); ) Sv(Q);
  }
  function Sv(e) {
    var t = W0(e.alternate, e, Qa);
    (e.memoizedProps = e.pendingProps), t === null ? Js(e) : (Q = t);
  }
  function Pg(e) {
    var t = e,
      a = t.alternate;
    switch (t.tag) {
      case 15:
      case 0:
        t = kg(a, t, t.pendingProps, t.type, void 0, K);
        break;
      case 11:
        t = kg(a, t, t.pendingProps, t.type.render, t.ref, K);
        break;
      case 5:
        Lf(t);
      default:
        Q0(a, t), (t = Q = Qy(t, Qa)), (t = W0(a, t, Qa));
    }
    (e.memoizedProps = e.pendingProps), t === null ? Js(e) : (Q = t);
  }
  function Jo(e, t, a, r) {
    (Fa = uo = null), Lf(t), (rn = null), (Nl = 0);
    var o = t.return;
    try {
      if (Sw(e, o, t, a, K)) {
        (Me = 1), ws(e, Kt(a, e.current)), (Q = null);
        return;
      }
    } catch (n) {
      if (o !== null) throw ((Q = o), n);
      (Me = 1), ws(e, Kt(a, e.current)), (Q = null);
      return;
    }
    t.flags & 32768
      ? (J || r === 1
          ? (e = !0)
          : Ln || (K & 536870912) !== 0
            ? (e = !1)
            : ((vr = e = !0),
              (r === 2 || r === 9 || r === 3 || r === 6) &&
                ((r = Ut.current), r !== null && r.tag === 13 && (r.flags |= 16384))),
        Lv(t, e))
      : Js(t);
  }
  function Js(e) {
    var t = e;
    do {
      if ((t.flags & 32768) !== 0) {
        Lv(t, vr);
        return;
      }
      e = t.return;
      var a = ww(t.alternate, t, Qa);
      if (a !== null) {
        Q = a;
        return;
      }
      if (((t = t.sibling), t !== null)) {
        Q = t;
        return;
      }
      Q = t = e;
    } while (t !== null);
    Me === 0 && (Me = 5);
  }
  function Lv(e, t) {
    do {
      var a = Rw(e.alternate, e);
      if (a !== null) {
        (a.flags &= 32767), (Q = a);
        return;
      }
      if (
        ((a = e.return),
        a !== null && ((a.flags |= 32768), (a.subtreeFlags = 0), (a.deletions = null)),
        !t && ((e = e.sibling), e !== null))
      ) {
        Q = e;
        return;
      }
      Q = e = a;
    } while (e !== null);
    (Me = 6), (Q = null);
  }
  function Bg(e, t, a, r, o, n, l, i, s) {
    e.cancelPendingCommit = null;
    do $s();
    while (He !== 0);
    if ((ne & 6) !== 0) throw Error(C(327));
    if (t !== null) {
      if (t === e.current) throw Error(C(177));
      if (
        ((n = t.lanes | t.childLanes),
        (n |= sf),
        fC(e, a, n, l, i, s),
        e === he && ((Q = he = null), (K = 0)),
        (pn = t),
        (_r = e),
        (Va = a),
        (Pd = n),
        (Bd = o),
        (pv = r),
        (t.subtreeFlags & 10256) !== 0 || (t.flags & 10256) !== 0
          ? ((e.callbackNode = null),
            (e.callbackPriority = 0),
            Uw(ms, function () {
              return Iv(), null;
            }))
          : ((e.callbackNode = null), (e.callbackPriority = 0)),
        (r = (t.flags & 13878) !== 0),
        (t.subtreeFlags & 13878) !== 0 || r)
      ) {
        (r = N.T), (N.T = null), (o = le.p), (le.p = 2), (l = ne), (ne |= 4);
        try {
          _w(e, t, a);
        } finally {
          (ne = l), (le.p = o), (N.T = r);
        }
      }
      (He = 1), Cv(), wv(), Rv();
    }
  }
  function Cv() {
    if (He === 1) {
      He = 0;
      var e = _r,
        t = pn,
        a = (t.flags & 13878) !== 0;
      if ((t.subtreeFlags & 13878) !== 0 || a) {
        (a = N.T), (N.T = null);
        var r = le.p;
        le.p = 2;
        var o = ne;
        ne |= 4;
        try {
          lv(t, e);
          var n = Fd,
            l = Fy(e.containerInfo),
            i = n.focusedElem,
            s = n.selectionRange;
          if (l !== i && i && i.ownerDocument && zy(i.ownerDocument.documentElement, i)) {
            if (s !== null && lf(i)) {
              var u = s.start,
                d = s.end;
              if ((d === void 0 && (d = u), 'selectionStart' in i))
                (i.selectionStart = u), (i.selectionEnd = Math.min(d, i.value.length));
              else {
                var c = i.ownerDocument || document,
                  f = (c && c.defaultView) || window;
                if (f.getSelection) {
                  var h = f.getSelection(),
                    v = i.textContent.length,
                    x = Math.min(s.start, v),
                    y = s.end === void 0 ? x : Math.min(s.end, v);
                  !h.extend && x > y && ((l = y), (y = x), (x = l));
                  var m = og(i, x),
                    p = og(i, y);
                  if (
                    m &&
                    p &&
                    (h.rangeCount !== 1 ||
                      h.anchorNode !== m.node ||
                      h.anchorOffset !== m.offset ||
                      h.focusNode !== p.node ||
                      h.focusOffset !== p.offset)
                  ) {
                    var g = c.createRange();
                    g.setStart(m.node, m.offset),
                      h.removeAllRanges(),
                      x > y
                        ? (h.addRange(g), h.extend(p.node, p.offset))
                        : (g.setEnd(p.node, p.offset), h.addRange(g));
                  }
                }
              }
            }
            for (c = [], h = i; (h = h.parentNode); )
              h.nodeType === 1 && c.push({ element: h, left: h.scrollLeft, top: h.scrollTop });
            for (typeof i.focus == 'function' && i.focus(), i = 0; i < c.length; i++) {
              var b = c[i];
              (b.element.scrollLeft = b.left), (b.element.scrollTop = b.top);
            }
          }
          (Us = !!zd), (Fd = zd = null);
        } finally {
          (ne = o), (le.p = r), (N.T = a);
        }
      }
      (e.current = t), (He = 2);
    }
  }
  function wv() {
    if (He === 2) {
      He = 0;
      var e = _r,
        t = pn,
        a = (t.flags & 8772) !== 0;
      if ((t.subtreeFlags & 8772) !== 0 || a) {
        (a = N.T), (N.T = null);
        var r = le.p;
        le.p = 2;
        var o = ne;
        ne |= 4;
        try {
          tv(e, t.alternate, t);
        } finally {
          (ne = o), (le.p = r), (N.T = a);
        }
      }
      He = 3;
    }
  }
  function Rv() {
    if (He === 4 || He === 3) {
      (He = 0), rC();
      var e = _r,
        t = pn,
        a = Va,
        r = pv;
      (t.subtreeFlags & 10256) !== 0 || (t.flags & 10256) !== 0
        ? (He = 5)
        : ((He = 0), (pn = _r = null), _v(e, e.pendingLanes));
      var o = e.pendingLanes;
      if (
        (o === 0 && (Rr = null),
        $d(a),
        (t = t.stateNode),
        Dt && typeof Dt.onCommitFiberRoot == 'function')
      )
        try {
          Dt.onCommitFiberRoot(Xl, t, void 0, (t.current.flags & 128) === 128);
        } catch {}
      if (r !== null) {
        (t = N.T), (o = le.p), (le.p = 2), (N.T = null);
        try {
          for (var n = e.onRecoverableError, l = 0; l < r.length; l++) {
            var i = r[l];
            n(i.value, { componentStack: i.stack });
          }
        } finally {
          (N.T = t), (le.p = o);
        }
      }
      (Va & 3) !== 0 && $s(),
        Sa(e),
        (o = e.pendingLanes),
        (a & 261930) !== 0 && (o & 42) !== 0 ? (e === Ud ? El++ : ((El = 0), (Ud = e))) : (El = 0),
        ti(0, !1);
    }
  }
  function _v(e, t) {
    (e.pooledCacheLanes &= t) === 0 &&
      ((t = e.pooledCache), t != null && ((e.pooledCache = null), Jl(t)));
  }
  function $s() {
    return Cv(), wv(), Rv(), Iv();
  }
  function Iv() {
    if (He !== 5) return !1;
    var e = _r,
      t = Pd;
    Pd = 0;
    var a = $d(Va),
      r = N.T,
      o = le.p;
    try {
      (le.p = 32 > a ? 32 : a), (N.T = null), (a = Bd), (Bd = null);
      var n = _r,
        l = Va;
      if (((He = 0), (pn = _r = null), (Va = 0), (ne & 6) !== 0)) throw Error(C(331));
      var i = ne;
      if (
        ((ne |= 4),
        dv(n.current),
        sv(n, n.current, l, a),
        (ne = i),
        ti(0, !1),
        Dt && typeof Dt.onPostCommitFiberRoot == 'function')
      )
        try {
          Dt.onPostCommitFiberRoot(Xl, n);
        } catch {}
      return !0;
    } finally {
      (le.p = o), (N.T = r), _v(e, t);
    }
  }
  function Ug(e, t, a) {
    (t = Kt(a, t)), (t = Ed(e.stateNode, t, 2)), (e = wr(e, t, 2)), e !== null && (Ql(e, 2), Sa(e));
  }
  function ce(e, t, a) {
    if (e.tag === 3) Ug(e, e, a);
    else
      for (; t !== null; ) {
        if (t.tag === 3) {
          Ug(t, e, a);
          break;
        } else if (t.tag === 1) {
          var r = t.stateNode;
          if (
            typeof t.type.getDerivedStateFromError == 'function' ||
            (typeof r.componentDidCatch == 'function' && (Rr === null || !Rr.has(r)))
          ) {
            (e = Kt(a, e)),
              (a = q0(2)),
              (r = wr(t, a, 2)),
              r !== null && (G0(a, r, t, e), Ql(r, 2), Sa(r));
            break;
          }
        }
        t = t.return;
      }
  }
  function Wc(e, t, a) {
    var r = e.pingCache;
    if (r === null) {
      r = e.pingCache = new Mw();
      var o = new Set();
      r.set(t, o);
    } else (o = r.get(t)), o === void 0 && ((o = new Set()), r.set(t, o));
    o.has(a) || ((Pf = !0), o.add(a), (e = Ow.bind(null, e, t, a)), t.then(e, e));
  }
  function Ow(e, t, a) {
    var r = e.pingCache;
    r !== null && r.delete(t),
      (e.pingedLanes |= e.suspendedLanes & a),
      (e.warmLanes &= ~a),
      he === e &&
        (K & a) === a &&
        (Me === 4 || (Me === 3 && (K & 62914560) === K && 300 > Tt() - Ks)
          ? (ne & 2) === 0 && hn(e, 0)
          : (Bf |= a),
        mn === K && (mn = 0)),
      Sa(e);
  }
  function kv(e, t) {
    t === 0 && (t = by()), (e = so(e, t)), e !== null && (Ql(e, t), Sa(e));
  }
  function Pw(e) {
    var t = e.memoizedState,
      a = 0;
    t !== null && (a = t.retryLane), kv(e, a);
  }
  function Bw(e, t) {
    var a = 0;
    switch (e.tag) {
      case 31:
      case 13:
        var r = e.stateNode,
          o = e.memoizedState;
        o !== null && (a = o.retryLane);
        break;
      case 19:
        r = e.stateNode;
        break;
      case 22:
        r = e.stateNode._retryCache;
        break;
      default:
        throw Error(C(314));
    }
    r !== null && r.delete(t), kv(e, a);
  }
  function Uw(e, t) {
    return Zd(e, t);
  }
  var Es = null,
    zo = null,
    Nd = !1,
    As = !1,
    Qc = !1,
    xr = 0;
  function Sa(e) {
    e !== zo && e.next === null && (zo === null ? (Es = zo = e) : (zo = zo.next = e)),
      (As = !0),
      Nd || ((Nd = !0), Hw());
  }
  function ti(e, t) {
    if (!Qc && As) {
      Qc = !0;
      do
        for (var a = !1, r = Es; r !== null; ) {
          if (!t)
            if (e !== 0) {
              var o = r.pendingLanes;
              if (o === 0) var n = 0;
              else {
                var l = r.suspendedLanes,
                  i = r.pingedLanes;
                (n = (1 << (31 - Ot(42 | e) + 1)) - 1),
                  (n &= o & ~(l & ~i)),
                  (n = n & 201326741 ? (n & 201326741) | 1 : n ? n | 2 : 0);
              }
              n !== 0 && ((a = !0), Ng(r, n));
            } else
              (n = K),
                (n = Hs(
                  r,
                  r === he ? n : 0,
                  r.cancelPendingCommit !== null || r.timeoutHandle !== -1
                )),
                (n & 3) === 0 || Wl(r, n) || ((a = !0), Ng(r, n));
          r = r.next;
        }
      while (a);
      Qc = !1;
    }
  }
  function Nw() {
    Mv();
  }
  function Mv() {
    As = Nd = !1;
    var e = 0;
    xr !== 0 && Xw() && (e = xr);
    for (var t = Tt(), a = null, r = Es; r !== null; ) {
      var o = r.next,
        n = Ev(r, t);
      n === 0
        ? ((r.next = null), a === null ? (Es = o) : (a.next = o), o === null && (zo = a))
        : ((a = r), (e !== 0 || (n & 3) !== 0) && (As = !0)),
        (r = o);
    }
    (He !== 0 && He !== 5) || ti(e, !1), xr !== 0 && (xr = 0);
  }
  function Ev(e, t) {
    for (
      var a = e.suspendedLanes,
        r = e.pingedLanes,
        o = e.expirationTimes,
        n = e.pendingLanes & -62914561;
      0 < n;

    ) {
      var l = 31 - Ot(n),
        i = 1 << l,
        s = o[l];
      s === -1
        ? ((i & a) === 0 || (i & r) !== 0) && (o[l] = dC(i, t))
        : s <= t && (e.expiredLanes |= i),
        (n &= ~i);
    }
    if (
      ((t = he),
      (a = K),
      (a = Hs(e, e === t ? a : 0, e.cancelPendingCommit !== null || e.timeoutHandle !== -1)),
      (r = e.callbackNode),
      a === 0 || (e === t && (ue === 2 || ue === 9)) || e.cancelPendingCommit !== null)
    )
      return r !== null && r !== null && wc(r), (e.callbackNode = null), (e.callbackPriority = 0);
    if ((a & 3) === 0 || Wl(e, a)) {
      if (((t = a & -a), t === e.callbackPriority)) return t;
      switch ((r !== null && wc(r), $d(a))) {
        case 2:
        case 8:
          a = yy;
          break;
        case 32:
          a = ms;
          break;
        case 268435456:
          a = vy;
          break;
        default:
          a = ms;
      }
      return (
        (r = Av.bind(null, e)), (a = Zd(a, r)), (e.callbackPriority = t), (e.callbackNode = a), t
      );
    }
    return r !== null && r !== null && wc(r), (e.callbackPriority = 2), (e.callbackNode = null), 2;
  }
  function Av(e, t) {
    if (He !== 0 && He !== 5) return (e.callbackNode = null), (e.callbackPriority = 0), null;
    var a = e.callbackNode;
    if ($s() && e.callbackNode !== a) return null;
    var r = K;
    return (
      (r = Hs(e, e === he ? r : 0, e.cancelPendingCommit !== null || e.timeoutHandle !== -1)),
      r === 0
        ? null
        : (gv(e, r, t),
          Ev(e, Tt()),
          e.callbackNode != null && e.callbackNode === a ? Av.bind(null, e) : null)
    );
  }
  function Ng(e, t) {
    if ($s()) return null;
    gv(e, t, !0);
  }
  function Hw() {
    Qw(function () {
      (ne & 6) !== 0 ? Zd(gy, Nw) : Mv();
    });
  }
  function Nf() {
    if (xr === 0) {
      var e = cn;
      e === 0 && ((e = Ti), (Ti <<= 1), (Ti & 261888) === 0 && (Ti = 256)), (xr = e);
    }
    return xr;
  }
  function Hg(e) {
    return e == null || typeof e == 'symbol' || typeof e == 'boolean'
      ? null
      : typeof e == 'function'
        ? e
        : Ki('' + e);
  }
  function zg(e, t) {
    var a = t.ownerDocument.createElement('input');
    return (
      (a.name = t.name),
      (a.value = t.value),
      e.id && a.setAttribute('form', e.id),
      t.parentNode.insertBefore(a, t),
      (e = new FormData(e)),
      a.parentNode.removeChild(a),
      e
    );
  }
  function zw(e, t, a, r, o) {
    if (t === 'submit' && a && a.stateNode === o) {
      var n = Hg((o[St] || null).action),
        l = r.submitter;
      l &&
        ((t = (t = l[St] || null) ? Hg(t.formAction) : l.getAttribute('formAction')),
        t !== null && ((n = t), (l = null)));
      var i = new zs('action', 'action', null, r, o);
      e.push({
        event: i,
        listeners: [
          {
            instance: null,
            listener: function () {
              if (r.defaultPrevented) {
                if (xr !== 0) {
                  var s = l ? zg(o, l) : new FormData(o);
                  kd(a, { pending: !0, data: s, method: o.method, action: n }, null, s);
                }
              } else
                typeof n == 'function' &&
                  (i.preventDefault(),
                  (s = l ? zg(o, l) : new FormData(o)),
                  kd(a, { pending: !0, data: s, method: o.method, action: n }, n, s));
            },
            currentTarget: o,
          },
        ],
      });
    }
  }
  for (Vi = 0; Vi < gd.length; Vi++)
    (ji = gd[Vi]),
      (Fg = ji.toLowerCase()),
      (qg = ji[0].toUpperCase() + ji.slice(1)),
      ia(Fg, 'on' + qg);
  var ji, Fg, qg, Vi;
  ia(Gy, 'onAnimationEnd');
  ia(Vy, 'onAnimationIteration');
  ia(jy, 'onAnimationStart');
  ia('dblclick', 'onDoubleClick');
  ia('focusin', 'onFocus');
  ia('focusout', 'onBlur');
  ia(ow, 'onTransitionRun');
  ia(nw, 'onTransitionStart');
  ia(lw, 'onTransitionCancel');
  ia(Yy, 'onTransitionEnd');
  sn('onMouseEnter', ['mouseout', 'mouseover']);
  sn('onMouseLeave', ['mouseout', 'mouseover']);
  sn('onPointerEnter', ['pointerout', 'pointerover']);
  sn('onPointerLeave', ['pointerout', 'pointerover']);
  no('onChange', 'change click focusin focusout input keydown keyup selectionchange'.split(' '));
  no(
    'onSelect',
    'focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange'.split(
      ' '
    )
  );
  no('onBeforeInput', ['compositionend', 'keypress', 'textInput', 'paste']);
  no('onCompositionEnd', 'compositionend focusout keydown keypress keyup mousedown'.split(' '));
  no('onCompositionStart', 'compositionstart focusout keydown keypress keyup mousedown'.split(' '));
  no(
    'onCompositionUpdate',
    'compositionupdate focusout keydown keypress keyup mousedown'.split(' ')
  );
  var Fl =
      'abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting'.split(
        ' '
      ),
    Fw = new Set(
      'beforetoggle cancel close invalid load scroll scrollend toggle'.split(' ').concat(Fl)
    );
  function Tv(e, t) {
    t = (t & 4) !== 0;
    for (var a = 0; a < e.length; a++) {
      var r = e[a],
        o = r.event;
      r = r.listeners;
      e: {
        var n = void 0;
        if (t)
          for (var l = r.length - 1; 0 <= l; l--) {
            var i = r[l],
              s = i.instance,
              u = i.currentTarget;
            if (((i = i.listener), s !== n && o.isPropagationStopped())) break e;
            (n = i), (o.currentTarget = u);
            try {
              n(o);
            } catch (d) {
              hs(d);
            }
            (o.currentTarget = null), (n = s);
          }
        else
          for (l = 0; l < r.length; l++) {
            if (
              ((i = r[l]),
              (s = i.instance),
              (u = i.currentTarget),
              (i = i.listener),
              s !== n && o.isPropagationStopped())
            )
              break e;
            (n = i), (o.currentTarget = u);
            try {
              n(o);
            } catch (d) {
              hs(d);
            }
            (o.currentTarget = null), (n = s);
          }
      }
    }
  }
  function W(e, t) {
    var a = t[sd];
    a === void 0 && (a = t[sd] = new Set());
    var r = e + '__bubble';
    a.has(r) || (Dv(t, e, 2, !1), a.add(r));
  }
  function Kc(e, t, a) {
    var r = 0;
    t && (r |= 4), Dv(a, e, r, t);
  }
  var Yi = '_reactListening' + Math.random().toString(36).slice(2);
  function Hf(e) {
    if (!e[Yi]) {
      (e[Yi] = !0),
        wy.forEach(function (a) {
          a !== 'selectionchange' && (Fw.has(a) || Kc(a, !1, e), Kc(a, !0, e));
        });
      var t = e.nodeType === 9 ? e : e.ownerDocument;
      t === null || t[Yi] || ((t[Yi] = !0), Kc('selectionchange', !1, t));
    }
  }
  function Dv(e, t, a, r) {
    switch (Yv(t)) {
      case 2:
        var o = hR;
        break;
      case 8:
        o = gR;
        break;
      default:
        o = Gf;
    }
    (a = o.bind(null, t, a, e)),
      (o = void 0),
      !md || (t !== 'touchstart' && t !== 'touchmove' && t !== 'wheel') || (o = !0),
      r
        ? o !== void 0
          ? e.addEventListener(t, a, { capture: !0, passive: o })
          : e.addEventListener(t, a, !0)
        : o !== void 0
          ? e.addEventListener(t, a, { passive: o })
          : e.addEventListener(t, a, !1);
  }
  function Zc(e, t, a, r, o) {
    var n = r;
    if ((t & 1) === 0 && (t & 2) === 0 && r !== null)
      e: for (;;) {
        if (r === null) return;
        var l = r.tag;
        if (l === 3 || l === 4) {
          var i = r.stateNode.containerInfo;
          if (i === o) break;
          if (l === 4)
            for (l = r.return; l !== null; ) {
              var s = l.tag;
              if ((s === 3 || s === 4) && l.stateNode.containerInfo === o) return;
              l = l.return;
            }
          for (; i !== null; ) {
            if (((l = Go(i)), l === null)) return;
            if (((s = l.tag), s === 5 || s === 6 || s === 26 || s === 27)) {
              r = n = l;
              continue e;
            }
            i = i.parentNode;
          }
        }
        r = r.return;
      }
    Ty(function () {
      var u = n,
        d = af(a),
        c = [];
      e: {
        var f = Xy.get(e);
        if (f !== void 0) {
          var h = zs,
            v = e;
          switch (e) {
            case 'keypress':
              if (Ji(a) === 0) break e;
            case 'keydown':
            case 'keyup':
              h = BC;
              break;
            case 'focusin':
              (v = 'focus'), (h = Mc);
              break;
            case 'focusout':
              (v = 'blur'), (h = Mc);
              break;
            case 'beforeblur':
            case 'afterblur':
              h = Mc;
              break;
            case 'click':
              if (a.button === 2) break e;
            case 'auxclick':
            case 'dblclick':
            case 'mousedown':
            case 'mousemove':
            case 'mouseup':
            case 'mouseout':
            case 'mouseover':
            case 'contextmenu':
              h = Qh;
              break;
            case 'drag':
            case 'dragend':
            case 'dragenter':
            case 'dragexit':
            case 'dragleave':
            case 'dragover':
            case 'dragstart':
            case 'drop':
              h = wC;
              break;
            case 'touchcancel':
            case 'touchend':
            case 'touchmove':
            case 'touchstart':
              h = HC;
              break;
            case Gy:
            case Vy:
            case jy:
              h = IC;
              break;
            case Yy:
              h = FC;
              break;
            case 'scroll':
            case 'scrollend':
              h = LC;
              break;
            case 'wheel':
              h = GC;
              break;
            case 'copy':
            case 'cut':
            case 'paste':
              h = MC;
              break;
            case 'gotpointercapture':
            case 'lostpointercapture':
            case 'pointercancel':
            case 'pointerdown':
            case 'pointermove':
            case 'pointerout':
            case 'pointerover':
            case 'pointerup':
              h = Zh;
              break;
            case 'toggle':
            case 'beforetoggle':
              h = jC;
          }
          var x = (t & 4) !== 0,
            y = !x && (e === 'scroll' || e === 'scrollend'),
            m = x ? (f !== null ? f + 'Capture' : null) : f;
          x = [];
          for (var p = u, g; p !== null; ) {
            var b = p;
            if (
              ((g = b.stateNode),
              (b = b.tag),
              (b !== 5 && b !== 26 && b !== 27) ||
                g === null ||
                m === null ||
                ((b = Dl(p, m)), b != null && x.push(ql(p, b, g))),
              y)
            )
              break;
            p = p.return;
          }
          0 < x.length && ((f = new h(f, v, null, a, d)), c.push({ event: f, listeners: x }));
        }
      }
      if ((t & 7) === 0) {
        e: {
          if (
            ((f = e === 'mouseover' || e === 'pointerover'),
            (h = e === 'mouseout' || e === 'pointerout'),
            f && a !== fd && (v = a.relatedTarget || a.fromElement) && (Go(v) || v[vn]))
          )
            break e;
          if (
            (h || f) &&
            ((f =
              d.window === d
                ? d
                : (f = d.ownerDocument)
                  ? f.defaultView || f.parentWindow
                  : window),
            h
              ? ((v = a.relatedTarget || a.toElement),
                (h = u),
                (v = v ? Go(v) : null),
                v !== null &&
                  ((y = Yl(v)), (x = v.tag), v !== y || (x !== 5 && x !== 27 && x !== 6)) &&
                  (v = null))
              : ((h = null), (v = u)),
            h !== v)
          ) {
            if (
              ((x = Qh),
              (b = 'onMouseLeave'),
              (m = 'onMouseEnter'),
              (p = 'mouse'),
              (e === 'pointerout' || e === 'pointerover') &&
                ((x = Zh), (b = 'onPointerLeave'), (m = 'onPointerEnter'), (p = 'pointer')),
              (y = h == null ? f : gl(h)),
              (g = v == null ? f : gl(v)),
              (f = new x(b, p + 'leave', h, a, d)),
              (f.target = y),
              (f.relatedTarget = g),
              (b = null),
              Go(d) === u &&
                ((x = new x(m, p + 'enter', v, a, d)),
                (x.target = g),
                (x.relatedTarget = y),
                (b = x)),
              (y = b),
              h && v)
            )
              t: {
                for (x = qw, m = h, p = v, g = 0, b = m; b; b = x(b)) g++;
                b = 0;
                for (var R = p; R; R = x(R)) b++;
                for (; 0 < g - b; ) (m = x(m)), g--;
                for (; 0 < b - g; ) (p = x(p)), b--;
                for (; g--; ) {
                  if (m === p || (p !== null && m === p.alternate)) {
                    x = m;
                    break t;
                  }
                  (m = x(m)), (p = x(p));
                }
                x = null;
              }
            else x = null;
            h !== null && Gg(c, f, h, x, !1), v !== null && y !== null && Gg(c, y, v, x, !0);
          }
        }
        e: {
          if (
            ((f = u ? gl(u) : window),
            (h = f.nodeName && f.nodeName.toLowerCase()),
            h === 'select' || (h === 'input' && f.type === 'file'))
          )
            var I = tg;
          else if (eg(f))
            if (Ny) I = tw;
            else {
              I = $C;
              var w = JC;
            }
          else
            (h = f.nodeName),
              !h || h.toLowerCase() !== 'input' || (f.type !== 'checkbox' && f.type !== 'radio')
                ? u && tf(u.elementType) && (I = tg)
                : (I = ew);
          if (I && (I = I(e, u))) {
            Uy(c, I, a, d);
            break e;
          }
          w && w(e, f, u),
            e === 'focusout' &&
              u &&
              f.type === 'number' &&
              u.memoizedProps.value != null &&
              dd(f, 'number', f.value);
        }
        switch (((w = u ? gl(u) : window), e)) {
          case 'focusin':
            (eg(w) || w.contentEditable === 'true') && ((Yo = w), (pd = u), (Sl = null));
            break;
          case 'focusout':
            Sl = pd = Yo = null;
            break;
          case 'mousedown':
            hd = !0;
            break;
          case 'contextmenu':
          case 'mouseup':
          case 'dragend':
            (hd = !1), ng(c, a, d);
            break;
          case 'selectionchange':
            if (rw) break;
          case 'keydown':
          case 'keyup':
            ng(c, a, d);
        }
        var L;
        if (nf)
          e: {
            switch (e) {
              case 'compositionstart':
                var M = 'onCompositionStart';
                break e;
              case 'compositionend':
                M = 'onCompositionEnd';
                break e;
              case 'compositionupdate':
                M = 'onCompositionUpdate';
                break e;
            }
            M = void 0;
          }
        else
          jo
            ? Py(e, a) && (M = 'onCompositionEnd')
            : e === 'keydown' && a.keyCode === 229 && (M = 'onCompositionStart');
        M &&
          (Oy &&
            a.locale !== 'ko' &&
            (jo || M !== 'onCompositionStart'
              ? M === 'onCompositionEnd' && jo && (L = Dy())
              : ((yr = d), (rf = 'value' in yr ? yr.value : yr.textContent), (jo = !0))),
          (w = Ts(u, M)),
          0 < w.length &&
            ((M = new Kh(M, e, null, a, d)),
            c.push({ event: M, listeners: w }),
            L ? (M.data = L) : ((L = By(a)), L !== null && (M.data = L)))),
          (L = XC ? WC(e, a) : QC(e, a)) &&
            ((M = Ts(u, 'onBeforeInput')),
            0 < M.length &&
              ((w = new Kh('onBeforeInput', 'beforeinput', null, a, d)),
              c.push({ event: w, listeners: M }),
              (w.data = L))),
          zw(c, e, u, a, d);
      }
      Tv(c, t);
    });
  }
  function ql(e, t, a) {
    return { instance: e, listener: t, currentTarget: a };
  }
  function Ts(e, t) {
    for (var a = t + 'Capture', r = []; e !== null; ) {
      var o = e,
        n = o.stateNode;
      if (
        ((o = o.tag),
        (o !== 5 && o !== 26 && o !== 27) ||
          n === null ||
          ((o = Dl(e, a)),
          o != null && r.unshift(ql(e, o, n)),
          (o = Dl(e, t)),
          o != null && r.push(ql(e, o, n))),
        e.tag === 3)
      )
        return r;
      e = e.return;
    }
    return [];
  }
  function qw(e) {
    if (e === null) return null;
    do e = e.return;
    while (e && e.tag !== 5 && e.tag !== 27);
    return e || null;
  }
  function Gg(e, t, a, r, o) {
    for (var n = t._reactName, l = []; a !== null && a !== r; ) {
      var i = a,
        s = i.alternate,
        u = i.stateNode;
      if (((i = i.tag), s !== null && s === r)) break;
      (i !== 5 && i !== 26 && i !== 27) ||
        u === null ||
        ((s = u),
        o
          ? ((u = Dl(a, n)), u != null && l.unshift(ql(a, u, s)))
          : o || ((u = Dl(a, n)), u != null && l.push(ql(a, u, s)))),
        (a = a.return);
    }
    l.length !== 0 && e.push({ event: t, listeners: l });
  }
  var Gw = /\r\n?/g,
    Vw = /\u0000|\uFFFD/g;
  function Vg(e) {
    return (typeof e == 'string' ? e : '' + e)
      .replace(
        Gw,
        `
`
      )
      .replace(Vw, '');
  }
  function Ov(e, t) {
    return (t = Vg(t)), Vg(e) === t;
  }
  function fe(e, t, a, r, o, n) {
    switch (a) {
      case 'children':
        typeof r == 'string'
          ? t === 'body' || (t === 'textarea' && r === '') || un(e, r)
          : (typeof r == 'number' || typeof r == 'bigint') && t !== 'body' && un(e, '' + r);
        break;
      case 'className':
        Pi(e, 'class', r);
        break;
      case 'tabIndex':
        Pi(e, 'tabindex', r);
        break;
      case 'dir':
      case 'role':
      case 'viewBox':
      case 'width':
      case 'height':
        Pi(e, a, r);
        break;
      case 'style':
        Ay(e, r, n);
        break;
      case 'data':
        if (t !== 'object') {
          Pi(e, 'data', r);
          break;
        }
      case 'src':
      case 'href':
        if (r === '' && (t !== 'a' || a !== 'href')) {
          e.removeAttribute(a);
          break;
        }
        if (r == null || typeof r == 'function' || typeof r == 'symbol' || typeof r == 'boolean') {
          e.removeAttribute(a);
          break;
        }
        (r = Ki('' + r)), e.setAttribute(a, r);
        break;
      case 'action':
      case 'formAction':
        if (typeof r == 'function') {
          e.setAttribute(
            a,
            "javascript:throw new Error('A React form was unexpectedly submitted. If you called form.submit() manually, consider using form.requestSubmit() instead. If you\\'re trying to use event.stopPropagation() in a submit event handler, consider also calling event.preventDefault().')"
          );
          break;
        } else
          typeof n == 'function' &&
            (a === 'formAction'
              ? (t !== 'input' && fe(e, t, 'name', o.name, o, null),
                fe(e, t, 'formEncType', o.formEncType, o, null),
                fe(e, t, 'formMethod', o.formMethod, o, null),
                fe(e, t, 'formTarget', o.formTarget, o, null))
              : (fe(e, t, 'encType', o.encType, o, null),
                fe(e, t, 'method', o.method, o, null),
                fe(e, t, 'target', o.target, o, null)));
        if (r == null || typeof r == 'symbol' || typeof r == 'boolean') {
          e.removeAttribute(a);
          break;
        }
        (r = Ki('' + r)), e.setAttribute(a, r);
        break;
      case 'onClick':
        r != null && (e.onclick = za);
        break;
      case 'onScroll':
        r != null && W('scroll', e);
        break;
      case 'onScrollEnd':
        r != null && W('scrollend', e);
        break;
      case 'dangerouslySetInnerHTML':
        if (r != null) {
          if (typeof r != 'object' || !('__html' in r)) throw Error(C(61));
          if (((a = r.__html), a != null)) {
            if (o.children != null) throw Error(C(60));
            e.innerHTML = a;
          }
        }
        break;
      case 'multiple':
        e.multiple = r && typeof r != 'function' && typeof r != 'symbol';
        break;
      case 'muted':
        e.muted = r && typeof r != 'function' && typeof r != 'symbol';
        break;
      case 'suppressContentEditableWarning':
      case 'suppressHydrationWarning':
      case 'defaultValue':
      case 'defaultChecked':
      case 'innerHTML':
      case 'ref':
        break;
      case 'autoFocus':
        break;
      case 'xlinkHref':
        if (r == null || typeof r == 'function' || typeof r == 'boolean' || typeof r == 'symbol') {
          e.removeAttribute('xlink:href');
          break;
        }
        (a = Ki('' + r)), e.setAttributeNS('http://www.w3.org/1999/xlink', 'xlink:href', a);
        break;
      case 'contentEditable':
      case 'spellCheck':
      case 'draggable':
      case 'value':
      case 'autoReverse':
      case 'externalResourcesRequired':
      case 'focusable':
      case 'preserveAlpha':
        r != null && typeof r != 'function' && typeof r != 'symbol'
          ? e.setAttribute(a, '' + r)
          : e.removeAttribute(a);
        break;
      case 'inert':
      case 'allowFullScreen':
      case 'async':
      case 'autoPlay':
      case 'controls':
      case 'default':
      case 'defer':
      case 'disabled':
      case 'disablePictureInPicture':
      case 'disableRemotePlayback':
      case 'formNoValidate':
      case 'hidden':
      case 'loop':
      case 'noModule':
      case 'noValidate':
      case 'open':
      case 'playsInline':
      case 'readOnly':
      case 'required':
      case 'reversed':
      case 'scoped':
      case 'seamless':
      case 'itemScope':
        r && typeof r != 'function' && typeof r != 'symbol'
          ? e.setAttribute(a, '')
          : e.removeAttribute(a);
        break;
      case 'capture':
      case 'download':
        r === !0
          ? e.setAttribute(a, '')
          : r !== !1 && r != null && typeof r != 'function' && typeof r != 'symbol'
            ? e.setAttribute(a, r)
            : e.removeAttribute(a);
        break;
      case 'cols':
      case 'rows':
      case 'size':
      case 'span':
        r != null && typeof r != 'function' && typeof r != 'symbol' && !isNaN(r) && 1 <= r
          ? e.setAttribute(a, r)
          : e.removeAttribute(a);
        break;
      case 'rowSpan':
      case 'start':
        r == null || typeof r == 'function' || typeof r == 'symbol' || isNaN(r)
          ? e.removeAttribute(a)
          : e.setAttribute(a, r);
        break;
      case 'popover':
        W('beforetoggle', e), W('toggle', e), Qi(e, 'popover', r);
        break;
      case 'xlinkActuate':
        Ta(e, 'http://www.w3.org/1999/xlink', 'xlink:actuate', r);
        break;
      case 'xlinkArcrole':
        Ta(e, 'http://www.w3.org/1999/xlink', 'xlink:arcrole', r);
        break;
      case 'xlinkRole':
        Ta(e, 'http://www.w3.org/1999/xlink', 'xlink:role', r);
        break;
      case 'xlinkShow':
        Ta(e, 'http://www.w3.org/1999/xlink', 'xlink:show', r);
        break;
      case 'xlinkTitle':
        Ta(e, 'http://www.w3.org/1999/xlink', 'xlink:title', r);
        break;
      case 'xlinkType':
        Ta(e, 'http://www.w3.org/1999/xlink', 'xlink:type', r);
        break;
      case 'xmlBase':
        Ta(e, 'http://www.w3.org/XML/1998/namespace', 'xml:base', r);
        break;
      case 'xmlLang':
        Ta(e, 'http://www.w3.org/XML/1998/namespace', 'xml:lang', r);
        break;
      case 'xmlSpace':
        Ta(e, 'http://www.w3.org/XML/1998/namespace', 'xml:space', r);
        break;
      case 'is':
        Qi(e, 'is', r);
        break;
      case 'innerText':
      case 'textContent':
        break;
      default:
        (!(2 < a.length) || (a[0] !== 'o' && a[0] !== 'O') || (a[1] !== 'n' && a[1] !== 'N')) &&
          ((a = xC.get(a) || a), Qi(e, a, r));
    }
  }
  function Hd(e, t, a, r, o, n) {
    switch (a) {
      case 'style':
        Ay(e, r, n);
        break;
      case 'dangerouslySetInnerHTML':
        if (r != null) {
          if (typeof r != 'object' || !('__html' in r)) throw Error(C(61));
          if (((a = r.__html), a != null)) {
            if (o.children != null) throw Error(C(60));
            e.innerHTML = a;
          }
        }
        break;
      case 'children':
        typeof r == 'string'
          ? un(e, r)
          : (typeof r == 'number' || typeof r == 'bigint') && un(e, '' + r);
        break;
      case 'onScroll':
        r != null && W('scroll', e);
        break;
      case 'onScrollEnd':
        r != null && W('scrollend', e);
        break;
      case 'onClick':
        r != null && (e.onclick = za);
        break;
      case 'suppressContentEditableWarning':
      case 'suppressHydrationWarning':
      case 'innerHTML':
      case 'ref':
        break;
      case 'innerText':
      case 'textContent':
        break;
      default:
        if (!Ry.hasOwnProperty(a))
          e: {
            if (
              a[0] === 'o' &&
              a[1] === 'n' &&
              ((o = a.endsWith('Capture')),
              (t = a.slice(2, o ? a.length - 7 : void 0)),
              (n = e[St] || null),
              (n = n != null ? n[a] : null),
              typeof n == 'function' && e.removeEventListener(t, n, o),
              typeof r == 'function')
            ) {
              typeof n != 'function' &&
                n !== null &&
                (a in e ? (e[a] = null) : e.hasAttribute(a) && e.removeAttribute(a)),
                e.addEventListener(t, r, o);
              break e;
            }
            a in e ? (e[a] = r) : r === !0 ? e.setAttribute(a, '') : Qi(e, a, r);
          }
    }
  }
  function rt(e, t, a) {
    switch (t) {
      case 'div':
      case 'span':
      case 'svg':
      case 'path':
      case 'a':
      case 'g':
      case 'p':
      case 'li':
        break;
      case 'img':
        W('error', e), W('load', e);
        var r = !1,
          o = !1,
          n;
        for (n in a)
          if (a.hasOwnProperty(n)) {
            var l = a[n];
            if (l != null)
              switch (n) {
                case 'src':
                  r = !0;
                  break;
                case 'srcSet':
                  o = !0;
                  break;
                case 'children':
                case 'dangerouslySetInnerHTML':
                  throw Error(C(137, t));
                default:
                  fe(e, t, n, l, a, null);
              }
          }
        o && fe(e, t, 'srcSet', a.srcSet, a, null), r && fe(e, t, 'src', a.src, a, null);
        return;
      case 'input':
        W('invalid', e);
        var i = (n = l = o = null),
          s = null,
          u = null;
        for (r in a)
          if (a.hasOwnProperty(r)) {
            var d = a[r];
            if (d != null)
              switch (r) {
                case 'name':
                  o = d;
                  break;
                case 'type':
                  l = d;
                  break;
                case 'checked':
                  s = d;
                  break;
                case 'defaultChecked':
                  u = d;
                  break;
                case 'value':
                  n = d;
                  break;
                case 'defaultValue':
                  i = d;
                  break;
                case 'children':
                case 'dangerouslySetInnerHTML':
                  if (d != null) throw Error(C(137, t));
                  break;
                default:
                  fe(e, t, r, d, a, null);
              }
          }
        ky(e, n, i, s, u, l, o, !1);
        return;
      case 'select':
        W('invalid', e), (r = l = n = null);
        for (o in a)
          if (a.hasOwnProperty(o) && ((i = a[o]), i != null))
            switch (o) {
              case 'value':
                n = i;
                break;
              case 'defaultValue':
                l = i;
                break;
              case 'multiple':
                r = i;
              default:
                fe(e, t, o, i, a, null);
            }
        (t = n),
          (a = l),
          (e.multiple = !!r),
          t != null ? en(e, !!r, t, !1) : a != null && en(e, !!r, a, !0);
        return;
      case 'textarea':
        W('invalid', e), (n = o = r = null);
        for (l in a)
          if (a.hasOwnProperty(l) && ((i = a[l]), i != null))
            switch (l) {
              case 'value':
                r = i;
                break;
              case 'defaultValue':
                o = i;
                break;
              case 'children':
                n = i;
                break;
              case 'dangerouslySetInnerHTML':
                if (i != null) throw Error(C(91));
                break;
              default:
                fe(e, t, l, i, a, null);
            }
        Ey(e, r, o, n);
        return;
      case 'option':
        for (s in a)
          if (a.hasOwnProperty(s) && ((r = a[s]), r != null))
            switch (s) {
              case 'selected':
                e.selected = r && typeof r != 'function' && typeof r != 'symbol';
                break;
              default:
                fe(e, t, s, r, a, null);
            }
        return;
      case 'dialog':
        W('beforetoggle', e), W('toggle', e), W('cancel', e), W('close', e);
        break;
      case 'iframe':
      case 'object':
        W('load', e);
        break;
      case 'video':
      case 'audio':
        for (r = 0; r < Fl.length; r++) W(Fl[r], e);
        break;
      case 'image':
        W('error', e), W('load', e);
        break;
      case 'details':
        W('toggle', e);
        break;
      case 'embed':
      case 'source':
      case 'link':
        W('error', e), W('load', e);
      case 'area':
      case 'base':
      case 'br':
      case 'col':
      case 'hr':
      case 'keygen':
      case 'meta':
      case 'param':
      case 'track':
      case 'wbr':
      case 'menuitem':
        for (u in a)
          if (a.hasOwnProperty(u) && ((r = a[u]), r != null))
            switch (u) {
              case 'children':
              case 'dangerouslySetInnerHTML':
                throw Error(C(137, t));
              default:
                fe(e, t, u, r, a, null);
            }
        return;
      default:
        if (tf(t)) {
          for (d in a)
            a.hasOwnProperty(d) && ((r = a[d]), r !== void 0 && Hd(e, t, d, r, a, void 0));
          return;
        }
    }
    for (i in a) a.hasOwnProperty(i) && ((r = a[i]), r != null && fe(e, t, i, r, a, null));
  }
  function jw(e, t, a, r) {
    switch (t) {
      case 'div':
      case 'span':
      case 'svg':
      case 'path':
      case 'a':
      case 'g':
      case 'p':
      case 'li':
        break;
      case 'input':
        var o = null,
          n = null,
          l = null,
          i = null,
          s = null,
          u = null,
          d = null;
        for (h in a) {
          var c = a[h];
          if (a.hasOwnProperty(h) && c != null)
            switch (h) {
              case 'checked':
                break;
              case 'value':
                break;
              case 'defaultValue':
                s = c;
              default:
                r.hasOwnProperty(h) || fe(e, t, h, null, r, c);
            }
        }
        for (var f in r) {
          var h = r[f];
          if (((c = a[f]), r.hasOwnProperty(f) && (h != null || c != null)))
            switch (f) {
              case 'type':
                n = h;
                break;
              case 'name':
                o = h;
                break;
              case 'checked':
                u = h;
                break;
              case 'defaultChecked':
                d = h;
                break;
              case 'value':
                l = h;
                break;
              case 'defaultValue':
                i = h;
                break;
              case 'children':
              case 'dangerouslySetInnerHTML':
                if (h != null) throw Error(C(137, t));
                break;
              default:
                h !== c && fe(e, t, f, h, r, c);
            }
        }
        cd(e, l, i, s, u, d, n, o);
        return;
      case 'select':
        h = l = i = f = null;
        for (n in a)
          if (((s = a[n]), a.hasOwnProperty(n) && s != null))
            switch (n) {
              case 'value':
                break;
              case 'multiple':
                h = s;
              default:
                r.hasOwnProperty(n) || fe(e, t, n, null, r, s);
            }
        for (o in r)
          if (((n = r[o]), (s = a[o]), r.hasOwnProperty(o) && (n != null || s != null)))
            switch (o) {
              case 'value':
                f = n;
                break;
              case 'defaultValue':
                i = n;
                break;
              case 'multiple':
                l = n;
              default:
                n !== s && fe(e, t, o, n, r, s);
            }
        (t = i),
          (a = l),
          (r = h),
          f != null
            ? en(e, !!a, f, !1)
            : !!r != !!a && (t != null ? en(e, !!a, t, !0) : en(e, !!a, a ? [] : '', !1));
        return;
      case 'textarea':
        h = f = null;
        for (i in a)
          if (((o = a[i]), a.hasOwnProperty(i) && o != null && !r.hasOwnProperty(i)))
            switch (i) {
              case 'value':
                break;
              case 'children':
                break;
              default:
                fe(e, t, i, null, r, o);
            }
        for (l in r)
          if (((o = r[l]), (n = a[l]), r.hasOwnProperty(l) && (o != null || n != null)))
            switch (l) {
              case 'value':
                f = o;
                break;
              case 'defaultValue':
                h = o;
                break;
              case 'children':
                break;
              case 'dangerouslySetInnerHTML':
                if (o != null) throw Error(C(91));
                break;
              default:
                o !== n && fe(e, t, l, o, r, n);
            }
        My(e, f, h);
        return;
      case 'option':
        for (var v in a)
          if (((f = a[v]), a.hasOwnProperty(v) && f != null && !r.hasOwnProperty(v)))
            switch (v) {
              case 'selected':
                e.selected = !1;
                break;
              default:
                fe(e, t, v, null, r, f);
            }
        for (s in r)
          if (((f = r[s]), (h = a[s]), r.hasOwnProperty(s) && f !== h && (f != null || h != null)))
            switch (s) {
              case 'selected':
                e.selected = f && typeof f != 'function' && typeof f != 'symbol';
                break;
              default:
                fe(e, t, s, f, r, h);
            }
        return;
      case 'img':
      case 'link':
      case 'area':
      case 'base':
      case 'br':
      case 'col':
      case 'embed':
      case 'hr':
      case 'keygen':
      case 'meta':
      case 'param':
      case 'source':
      case 'track':
      case 'wbr':
      case 'menuitem':
        for (var x in a)
          (f = a[x]),
            a.hasOwnProperty(x) && f != null && !r.hasOwnProperty(x) && fe(e, t, x, null, r, f);
        for (u in r)
          if (((f = r[u]), (h = a[u]), r.hasOwnProperty(u) && f !== h && (f != null || h != null)))
            switch (u) {
              case 'children':
              case 'dangerouslySetInnerHTML':
                if (f != null) throw Error(C(137, t));
                break;
              default:
                fe(e, t, u, f, r, h);
            }
        return;
      default:
        if (tf(t)) {
          for (var y in a)
            (f = a[y]),
              a.hasOwnProperty(y) &&
                f !== void 0 &&
                !r.hasOwnProperty(y) &&
                Hd(e, t, y, void 0, r, f);
          for (d in r)
            (f = r[d]),
              (h = a[d]),
              !r.hasOwnProperty(d) ||
                f === h ||
                (f === void 0 && h === void 0) ||
                Hd(e, t, d, f, r, h);
          return;
        }
    }
    for (var m in a)
      (f = a[m]),
        a.hasOwnProperty(m) && f != null && !r.hasOwnProperty(m) && fe(e, t, m, null, r, f);
    for (c in r)
      (f = r[c]),
        (h = a[c]),
        !r.hasOwnProperty(c) || f === h || (f == null && h == null) || fe(e, t, c, f, r, h);
  }
  function jg(e) {
    switch (e) {
      case 'css':
      case 'script':
      case 'font':
      case 'img':
      case 'image':
      case 'input':
      case 'link':
        return !0;
      default:
        return !1;
    }
  }
  function Yw() {
    if (typeof performance.getEntriesByType == 'function') {
      for (
        var e = 0, t = 0, a = performance.getEntriesByType('resource'), r = 0;
        r < a.length;
        r++
      ) {
        var o = a[r],
          n = o.transferSize,
          l = o.initiatorType,
          i = o.duration;
        if (n && i && jg(l)) {
          for (l = 0, i = o.responseEnd, r += 1; r < a.length; r++) {
            var s = a[r],
              u = s.startTime;
            if (u > i) break;
            var d = s.transferSize,
              c = s.initiatorType;
            d && jg(c) && ((s = s.responseEnd), (l += d * (s < i ? 1 : (i - u) / (s - u))));
          }
          if ((--r, (t += (8 * (n + l)) / (o.duration / 1e3)), e++, 10 < e)) break;
        }
      }
      if (0 < e) return t / e / 1e6;
    }
    return navigator.connection && ((e = navigator.connection.downlink), typeof e == 'number')
      ? e
      : 5;
  }
  var zd = null,
    Fd = null;
  function Ds(e) {
    return e.nodeType === 9 ? e : e.ownerDocument;
  }
  function Yg(e) {
    switch (e) {
      case 'http://www.w3.org/2000/svg':
        return 1;
      case 'http://www.w3.org/1998/Math/MathML':
        return 2;
      default:
        return 0;
    }
  }
  function Pv(e, t) {
    if (e === 0)
      switch (t) {
        case 'svg':
          return 1;
        case 'math':
          return 2;
        default:
          return 0;
      }
    return e === 1 && t === 'foreignObject' ? 0 : e;
  }
  function qd(e, t) {
    return (
      e === 'textarea' ||
      e === 'noscript' ||
      typeof t.children == 'string' ||
      typeof t.children == 'number' ||
      typeof t.children == 'bigint' ||
      (typeof t.dangerouslySetInnerHTML == 'object' &&
        t.dangerouslySetInnerHTML !== null &&
        t.dangerouslySetInnerHTML.__html != null)
    );
  }
  var Jc = null;
  function Xw() {
    var e = window.event;
    return e && e.type === 'popstate' ? (e === Jc ? !1 : ((Jc = e), !0)) : ((Jc = null), !1);
  }
  var Bv = typeof setTimeout == 'function' ? setTimeout : void 0,
    Ww = typeof clearTimeout == 'function' ? clearTimeout : void 0,
    Xg = typeof Promise == 'function' ? Promise : void 0,
    Qw =
      typeof queueMicrotask == 'function'
        ? queueMicrotask
        : typeof Xg < 'u'
          ? function (e) {
              return Xg.resolve(null).then(e).catch(Kw);
            }
          : Bv;
  function Kw(e) {
    setTimeout(function () {
      throw e;
    });
  }
  function Pr(e) {
    return e === 'head';
  }
  function Wg(e, t) {
    var a = t,
      r = 0;
    do {
      var o = a.nextSibling;
      if ((e.removeChild(a), o && o.nodeType === 8))
        if (((a = o.data), a === '/$' || a === '/&')) {
          if (r === 0) {
            e.removeChild(o), yn(t);
            return;
          }
          r--;
        } else if (a === '$' || a === '$?' || a === '$~' || a === '$!' || a === '&') r++;
        else if (a === 'html') Al(e.ownerDocument.documentElement);
        else if (a === 'head') {
          (a = e.ownerDocument.head), Al(a);
          for (var n = a.firstChild; n; ) {
            var l = n.nextSibling,
              i = n.nodeName;
            n[Kl] ||
              i === 'SCRIPT' ||
              i === 'STYLE' ||
              (i === 'LINK' && n.rel.toLowerCase() === 'stylesheet') ||
              a.removeChild(n),
              (n = l);
          }
        } else a === 'body' && Al(e.ownerDocument.body);
      a = o;
    } while (a);
    yn(t);
  }
  function Qg(e, t) {
    var a = e;
    e = 0;
    do {
      var r = a.nextSibling;
      if (
        (a.nodeType === 1
          ? t
            ? ((a._stashedDisplay = a.style.display), (a.style.display = 'none'))
            : ((a.style.display = a._stashedDisplay || ''),
              a.getAttribute('style') === '' && a.removeAttribute('style'))
          : a.nodeType === 3 &&
            (t
              ? ((a._stashedText = a.nodeValue), (a.nodeValue = ''))
              : (a.nodeValue = a._stashedText || '')),
        r && r.nodeType === 8)
      )
        if (((a = r.data), a === '/$')) {
          if (e === 0) break;
          e--;
        } else (a !== '$' && a !== '$?' && a !== '$~' && a !== '$!') || e++;
      a = r;
    } while (a);
  }
  function Gd(e) {
    var t = e.firstChild;
    for (t && t.nodeType === 10 && (t = t.nextSibling); t; ) {
      var a = t;
      switch (((t = t.nextSibling), a.nodeName)) {
        case 'HTML':
        case 'HEAD':
        case 'BODY':
          Gd(a), ef(a);
          continue;
        case 'SCRIPT':
        case 'STYLE':
          continue;
        case 'LINK':
          if (a.rel.toLowerCase() === 'stylesheet') continue;
      }
      e.removeChild(a);
    }
  }
  function Zw(e, t, a, r) {
    for (; e.nodeType === 1; ) {
      var o = a;
      if (e.nodeName.toLowerCase() !== t.toLowerCase()) {
        if (!r && (e.nodeName !== 'INPUT' || e.type !== 'hidden')) break;
      } else if (r) {
        if (!e[Kl])
          switch (t) {
            case 'meta':
              if (!e.hasAttribute('itemprop')) break;
              return e;
            case 'link':
              if (
                ((n = e.getAttribute('rel')),
                n === 'stylesheet' && e.hasAttribute('data-precedence'))
              )
                break;
              if (
                n !== o.rel ||
                e.getAttribute('href') !== (o.href == null || o.href === '' ? null : o.href) ||
                e.getAttribute('crossorigin') !== (o.crossOrigin == null ? null : o.crossOrigin) ||
                e.getAttribute('title') !== (o.title == null ? null : o.title)
              )
                break;
              return e;
            case 'style':
              if (e.hasAttribute('data-precedence')) break;
              return e;
            case 'script':
              if (
                ((n = e.getAttribute('src')),
                (n !== (o.src == null ? null : o.src) ||
                  e.getAttribute('type') !== (o.type == null ? null : o.type) ||
                  e.getAttribute('crossorigin') !==
                    (o.crossOrigin == null ? null : o.crossOrigin)) &&
                  n &&
                  e.hasAttribute('async') &&
                  !e.hasAttribute('itemprop'))
              )
                break;
              return e;
            default:
              return e;
          }
      } else if (t === 'input' && e.type === 'hidden') {
        var n = o.name == null ? null : '' + o.name;
        if (o.type === 'hidden' && e.getAttribute('name') === n) return e;
      } else return e;
      if (((e = $t(e.nextSibling)), e === null)) break;
    }
    return null;
  }
  function Jw(e, t, a) {
    if (t === '') return null;
    for (; e.nodeType !== 3; )
      if (
        ((e.nodeType !== 1 || e.nodeName !== 'INPUT' || e.type !== 'hidden') && !a) ||
        ((e = $t(e.nextSibling)), e === null)
      )
        return null;
    return e;
  }
  function Uv(e, t) {
    for (; e.nodeType !== 8; )
      if (
        ((e.nodeType !== 1 || e.nodeName !== 'INPUT' || e.type !== 'hidden') && !t) ||
        ((e = $t(e.nextSibling)), e === null)
      )
        return null;
    return e;
  }
  function Vd(e) {
    return e.data === '$?' || e.data === '$~';
  }
  function jd(e) {
    return e.data === '$!' || (e.data === '$?' && e.ownerDocument.readyState !== 'loading');
  }
  function $w(e, t) {
    var a = e.ownerDocument;
    if (e.data === '$~') e._reactRetry = t;
    else if (e.data !== '$?' || a.readyState !== 'loading') t();
    else {
      var r = function () {
        t(), a.removeEventListener('DOMContentLoaded', r);
      };
      a.addEventListener('DOMContentLoaded', r), (e._reactRetry = r);
    }
  }
  function $t(e) {
    for (; e != null; e = e.nextSibling) {
      var t = e.nodeType;
      if (t === 1 || t === 3) break;
      if (t === 8) {
        if (
          ((t = e.data),
          t === '$' ||
            t === '$!' ||
            t === '$?' ||
            t === '$~' ||
            t === '&' ||
            t === 'F!' ||
            t === 'F')
        )
          break;
        if (t === '/$' || t === '/&') return null;
      }
    }
    return e;
  }
  var Yd = null;
  function Kg(e) {
    e = e.nextSibling;
    for (var t = 0; e; ) {
      if (e.nodeType === 8) {
        var a = e.data;
        if (a === '/$' || a === '/&') {
          if (t === 0) return $t(e.nextSibling);
          t--;
        } else (a !== '$' && a !== '$!' && a !== '$?' && a !== '$~' && a !== '&') || t++;
      }
      e = e.nextSibling;
    }
    return null;
  }
  function Zg(e) {
    e = e.previousSibling;
    for (var t = 0; e; ) {
      if (e.nodeType === 8) {
        var a = e.data;
        if (a === '$' || a === '$!' || a === '$?' || a === '$~' || a === '&') {
          if (t === 0) return e;
          t--;
        } else (a !== '/$' && a !== '/&') || t++;
      }
      e = e.previousSibling;
    }
    return null;
  }
  function Nv(e, t, a) {
    switch (((t = Ds(a)), e)) {
      case 'html':
        if (((e = t.documentElement), !e)) throw Error(C(452));
        return e;
      case 'head':
        if (((e = t.head), !e)) throw Error(C(453));
        return e;
      case 'body':
        if (((e = t.body), !e)) throw Error(C(454));
        return e;
      default:
        throw Error(C(451));
    }
  }
  function Al(e) {
    for (var t = e.attributes; t.length; ) e.removeAttributeNode(t[0]);
    ef(e);
  }
  var ea = new Map(),
    Jg = new Set();
  function Os(e) {
    return typeof e.getRootNode == 'function'
      ? e.getRootNode()
      : e.nodeType === 9
        ? e
        : e.ownerDocument;
  }
  var Ka = le.d;
  le.d = { f: eR, r: tR, D: aR, C: rR, L: oR, m: nR, X: iR, S: lR, M: sR };
  function eR() {
    var e = Ka.f(),
      t = Zs();
    return e || t;
  }
  function tR(e) {
    var t = bn(e);
    t !== null && t.tag === 5 && t.type === 'form' ? A0(t) : Ka.r(e);
  }
  var Cn = typeof document > 'u' ? null : document;
  function Hv(e, t, a) {
    var r = Cn;
    if (r && typeof t == 'string' && t) {
      var o = Qt(t);
      (o = 'link[rel="' + e + '"][href="' + o + '"]'),
        typeof a == 'string' && (o += '[crossorigin="' + a + '"]'),
        Jg.has(o) ||
          (Jg.add(o),
          (e = { rel: e, crossOrigin: a, href: t }),
          r.querySelector(o) === null &&
            ((t = r.createElement('link')), rt(t, 'link', e), je(t), r.head.appendChild(t)));
    }
  }
  function aR(e) {
    Ka.D(e), Hv('dns-prefetch', e, null);
  }
  function rR(e, t) {
    Ka.C(e, t), Hv('preconnect', e, t);
  }
  function oR(e, t, a) {
    Ka.L(e, t, a);
    var r = Cn;
    if (r && e && t) {
      var o = 'link[rel="preload"][as="' + Qt(t) + '"]';
      t === 'image' && a && a.imageSrcSet
        ? ((o += '[imagesrcset="' + Qt(a.imageSrcSet) + '"]'),
          typeof a.imageSizes == 'string' && (o += '[imagesizes="' + Qt(a.imageSizes) + '"]'))
        : (o += '[href="' + Qt(e) + '"]');
      var n = o;
      switch (t) {
        case 'style':
          n = gn(e);
          break;
        case 'script':
          n = wn(e);
      }
      ea.has(n) ||
        ((e = Ce(
          { rel: 'preload', href: t === 'image' && a && a.imageSrcSet ? void 0 : e, as: t },
          a
        )),
        ea.set(n, e),
        r.querySelector(o) !== null ||
          (t === 'style' && r.querySelector(ai(n))) ||
          (t === 'script' && r.querySelector(ri(n))) ||
          ((t = r.createElement('link')), rt(t, 'link', e), je(t), r.head.appendChild(t)));
    }
  }
  function nR(e, t) {
    Ka.m(e, t);
    var a = Cn;
    if (a && e) {
      var r = t && typeof t.as == 'string' ? t.as : 'script',
        o = 'link[rel="modulepreload"][as="' + Qt(r) + '"][href="' + Qt(e) + '"]',
        n = o;
      switch (r) {
        case 'audioworklet':
        case 'paintworklet':
        case 'serviceworker':
        case 'sharedworker':
        case 'worker':
        case 'script':
          n = wn(e);
      }
      if (
        !ea.has(n) &&
        ((e = Ce({ rel: 'modulepreload', href: e }, t)), ea.set(n, e), a.querySelector(o) === null)
      ) {
        switch (r) {
          case 'audioworklet':
          case 'paintworklet':
          case 'serviceworker':
          case 'sharedworker':
          case 'worker':
          case 'script':
            if (a.querySelector(ri(n))) return;
        }
        (r = a.createElement('link')), rt(r, 'link', e), je(r), a.head.appendChild(r);
      }
    }
  }
  function lR(e, t, a) {
    Ka.S(e, t, a);
    var r = Cn;
    if (r && e) {
      var o = $o(r).hoistableStyles,
        n = gn(e);
      t = t || 'default';
      var l = o.get(n);
      if (!l) {
        var i = { loading: 0, preload: null };
        if ((l = r.querySelector(ai(n)))) i.loading = 5;
        else {
          (e = Ce({ rel: 'stylesheet', href: e, 'data-precedence': t }, a)),
            (a = ea.get(n)) && zf(e, a);
          var s = (l = r.createElement('link'));
          je(s),
            rt(s, 'link', e),
            (s._p = new Promise(function (u, d) {
              (s.onload = u), (s.onerror = d);
            })),
            s.addEventListener('load', function () {
              i.loading |= 1;
            }),
            s.addEventListener('error', function () {
              i.loading |= 2;
            }),
            (i.loading |= 4),
            ls(l, t, r);
        }
        (l = { type: 'stylesheet', instance: l, count: 1, state: i }), o.set(n, l);
      }
    }
  }
  function iR(e, t) {
    Ka.X(e, t);
    var a = Cn;
    if (a && e) {
      var r = $o(a).hoistableScripts,
        o = wn(e),
        n = r.get(o);
      n ||
        ((n = a.querySelector(ri(o))),
        n ||
          ((e = Ce({ src: e, async: !0 }, t)),
          (t = ea.get(o)) && Ff(e, t),
          (n = a.createElement('script')),
          je(n),
          rt(n, 'link', e),
          a.head.appendChild(n)),
        (n = { type: 'script', instance: n, count: 1, state: null }),
        r.set(o, n));
    }
  }
  function sR(e, t) {
    Ka.M(e, t);
    var a = Cn;
    if (a && e) {
      var r = $o(a).hoistableScripts,
        o = wn(e),
        n = r.get(o);
      n ||
        ((n = a.querySelector(ri(o))),
        n ||
          ((e = Ce({ src: e, async: !0, type: 'module' }, t)),
          (t = ea.get(o)) && Ff(e, t),
          (n = a.createElement('script')),
          je(n),
          rt(n, 'link', e),
          a.head.appendChild(n)),
        (n = { type: 'script', instance: n, count: 1, state: null }),
        r.set(o, n));
    }
  }
  function $g(e, t, a, r) {
    var o = (o = Sr.current) ? Os(o) : null;
    if (!o) throw Error(C(446));
    switch (e) {
      case 'meta':
      case 'title':
        return null;
      case 'style':
        return typeof a.precedence == 'string' && typeof a.href == 'string'
          ? ((t = gn(a.href)),
            (a = $o(o).hoistableStyles),
            (r = a.get(t)),
            r || ((r = { type: 'style', instance: null, count: 0, state: null }), a.set(t, r)),
            r)
          : { type: 'void', instance: null, count: 0, state: null };
      case 'link':
        if (
          a.rel === 'stylesheet' &&
          typeof a.href == 'string' &&
          typeof a.precedence == 'string'
        ) {
          e = gn(a.href);
          var n = $o(o).hoistableStyles,
            l = n.get(e);
          if (
            (l ||
              ((o = o.ownerDocument || o),
              (l = {
                type: 'stylesheet',
                instance: null,
                count: 0,
                state: { loading: 0, preload: null },
              }),
              n.set(e, l),
              (n = o.querySelector(ai(e))) && !n._p && ((l.instance = n), (l.state.loading = 5)),
              ea.has(e) ||
                ((a = {
                  rel: 'preload',
                  as: 'style',
                  href: a.href,
                  crossOrigin: a.crossOrigin,
                  integrity: a.integrity,
                  media: a.media,
                  hrefLang: a.hrefLang,
                  referrerPolicy: a.referrerPolicy,
                }),
                ea.set(e, a),
                n || uR(o, e, a, l.state))),
            t && r === null)
          )
            throw Error(C(528, ''));
          return l;
        }
        if (t && r !== null) throw Error(C(529, ''));
        return null;
      case 'script':
        return (
          (t = a.async),
          (a = a.src),
          typeof a == 'string' && t && typeof t != 'function' && typeof t != 'symbol'
            ? ((t = wn(a)),
              (a = $o(o).hoistableScripts),
              (r = a.get(t)),
              r || ((r = { type: 'script', instance: null, count: 0, state: null }), a.set(t, r)),
              r)
            : { type: 'void', instance: null, count: 0, state: null }
        );
      default:
        throw Error(C(444, e));
    }
  }
  function gn(e) {
    return 'href="' + Qt(e) + '"';
  }
  function ai(e) {
    return 'link[rel="stylesheet"][' + e + ']';
  }
  function zv(e) {
    return Ce({}, e, { 'data-precedence': e.precedence, precedence: null });
  }
  function uR(e, t, a, r) {
    e.querySelector('link[rel="preload"][as="style"][' + t + ']')
      ? (r.loading = 1)
      : ((t = e.createElement('link')),
        (r.preload = t),
        t.addEventListener('load', function () {
          return (r.loading |= 1);
        }),
        t.addEventListener('error', function () {
          return (r.loading |= 2);
        }),
        rt(t, 'link', a),
        je(t),
        e.head.appendChild(t));
  }
  function wn(e) {
    return '[src="' + Qt(e) + '"]';
  }
  function ri(e) {
    return 'script[async]' + e;
  }
  function ey(e, t, a) {
    if ((t.count++, t.instance === null))
      switch (t.type) {
        case 'style':
          var r = e.querySelector('style[data-href~="' + Qt(a.href) + '"]');
          if (r) return (t.instance = r), je(r), r;
          var o = Ce({}, a, {
            'data-href': a.href,
            'data-precedence': a.precedence,
            href: null,
            precedence: null,
          });
          return (
            (r = (e.ownerDocument || e).createElement('style')),
            je(r),
            rt(r, 'style', o),
            ls(r, a.precedence, e),
            (t.instance = r)
          );
        case 'stylesheet':
          o = gn(a.href);
          var n = e.querySelector(ai(o));
          if (n) return (t.state.loading |= 4), (t.instance = n), je(n), n;
          (r = zv(a)),
            (o = ea.get(o)) && zf(r, o),
            (n = (e.ownerDocument || e).createElement('link')),
            je(n);
          var l = n;
          return (
            (l._p = new Promise(function (i, s) {
              (l.onload = i), (l.onerror = s);
            })),
            rt(n, 'link', r),
            (t.state.loading |= 4),
            ls(n, a.precedence, e),
            (t.instance = n)
          );
        case 'script':
          return (
            (n = wn(a.src)),
            (o = e.querySelector(ri(n)))
              ? ((t.instance = o), je(o), o)
              : ((r = a),
                (o = ea.get(n)) && ((r = Ce({}, a)), Ff(r, o)),
                (e = e.ownerDocument || e),
                (o = e.createElement('script')),
                je(o),
                rt(o, 'link', r),
                e.head.appendChild(o),
                (t.instance = o))
          );
        case 'void':
          return null;
        default:
          throw Error(C(443, t.type));
      }
    else
      t.type === 'stylesheet' &&
        (t.state.loading & 4) === 0 &&
        ((r = t.instance), (t.state.loading |= 4), ls(r, a.precedence, e));
    return t.instance;
  }
  function ls(e, t, a) {
    for (
      var r = a.querySelectorAll('link[rel="stylesheet"][data-precedence],style[data-precedence]'),
        o = r.length ? r[r.length - 1] : null,
        n = o,
        l = 0;
      l < r.length;
      l++
    ) {
      var i = r[l];
      if (i.dataset.precedence === t) n = i;
      else if (n !== o) break;
    }
    n
      ? n.parentNode.insertBefore(e, n.nextSibling)
      : ((t = a.nodeType === 9 ? a.head : a), t.insertBefore(e, t.firstChild));
  }
  function zf(e, t) {
    e.crossOrigin == null && (e.crossOrigin = t.crossOrigin),
      e.referrerPolicy == null && (e.referrerPolicy = t.referrerPolicy),
      e.title == null && (e.title = t.title);
  }
  function Ff(e, t) {
    e.crossOrigin == null && (e.crossOrigin = t.crossOrigin),
      e.referrerPolicy == null && (e.referrerPolicy = t.referrerPolicy),
      e.integrity == null && (e.integrity = t.integrity);
  }
  var is = null;
  function ty(e, t, a) {
    if (is === null) {
      var r = new Map(),
        o = (is = new Map());
      o.set(a, r);
    } else (o = is), (r = o.get(a)), r || ((r = new Map()), o.set(a, r));
    if (r.has(e)) return r;
    for (r.set(e, null), a = a.getElementsByTagName(e), o = 0; o < a.length; o++) {
      var n = a[o];
      if (
        !(n[Kl] || n[et] || (e === 'link' && n.getAttribute('rel') === 'stylesheet')) &&
        n.namespaceURI !== 'http://www.w3.org/2000/svg'
      ) {
        var l = n.getAttribute(t) || '';
        l = e + l;
        var i = r.get(l);
        i ? i.push(n) : r.set(l, [n]);
      }
    }
    return r;
  }
  function ay(e, t, a) {
    (e = e.ownerDocument || e),
      e.head.insertBefore(a, t === 'title' ? e.querySelector('head > title') : null);
  }
  function cR(e, t, a) {
    if (a === 1 || t.itemProp != null) return !1;
    switch (e) {
      case 'meta':
      case 'title':
        return !0;
      case 'style':
        if (typeof t.precedence != 'string' || typeof t.href != 'string' || t.href === '') break;
        return !0;
      case 'link':
        if (
          typeof t.rel != 'string' ||
          typeof t.href != 'string' ||
          t.href === '' ||
          t.onLoad ||
          t.onError
        )
          break;
        switch (t.rel) {
          case 'stylesheet':
            return (e = t.disabled), typeof t.precedence == 'string' && e == null;
          default:
            return !0;
        }
      case 'script':
        if (
          t.async &&
          typeof t.async != 'function' &&
          typeof t.async != 'symbol' &&
          !t.onLoad &&
          !t.onError &&
          t.src &&
          typeof t.src == 'string'
        )
          return !0;
    }
    return !1;
  }
  function Fv(e) {
    return !(e.type === 'stylesheet' && (e.state.loading & 3) === 0);
  }
  function dR(e, t, a, r) {
    if (
      a.type === 'stylesheet' &&
      (typeof r.media != 'string' || matchMedia(r.media).matches !== !1) &&
      (a.state.loading & 4) === 0
    ) {
      if (a.instance === null) {
        var o = gn(r.href),
          n = t.querySelector(ai(o));
        if (n) {
          (t = n._p),
            t !== null &&
              typeof t == 'object' &&
              typeof t.then == 'function' &&
              (e.count++, (e = Ps.bind(e)), t.then(e, e)),
            (a.state.loading |= 4),
            (a.instance = n),
            je(n);
          return;
        }
        (n = t.ownerDocument || t),
          (r = zv(r)),
          (o = ea.get(o)) && zf(r, o),
          (n = n.createElement('link')),
          je(n);
        var l = n;
        (l._p = new Promise(function (i, s) {
          (l.onload = i), (l.onerror = s);
        })),
          rt(n, 'link', r),
          (a.instance = n);
      }
      e.stylesheets === null && (e.stylesheets = new Map()),
        e.stylesheets.set(a, t),
        (t = a.state.preload) &&
          (a.state.loading & 3) === 0 &&
          (e.count++,
          (a = Ps.bind(e)),
          t.addEventListener('load', a),
          t.addEventListener('error', a));
    }
  }
  var $c = 0;
  function fR(e, t) {
    return (
      e.stylesheets && e.count === 0 && ss(e, e.stylesheets),
      0 < e.count || 0 < e.imgCount
        ? function (a) {
            var r = setTimeout(function () {
              if ((e.stylesheets && ss(e, e.stylesheets), e.unsuspend)) {
                var n = e.unsuspend;
                (e.unsuspend = null), n();
              }
            }, 6e4 + t);
            0 < e.imgBytes && $c === 0 && ($c = 62500 * Yw());
            var o = setTimeout(
              function () {
                if (
                  ((e.waitingForImages = !1),
                  e.count === 0 && (e.stylesheets && ss(e, e.stylesheets), e.unsuspend))
                ) {
                  var n = e.unsuspend;
                  (e.unsuspend = null), n();
                }
              },
              (e.imgBytes > $c ? 50 : 800) + t
            );
            return (
              (e.unsuspend = a),
              function () {
                (e.unsuspend = null), clearTimeout(r), clearTimeout(o);
              }
            );
          }
        : null
    );
  }
  function Ps() {
    if ((this.count--, this.count === 0 && (this.imgCount === 0 || !this.waitingForImages))) {
      if (this.stylesheets) ss(this, this.stylesheets);
      else if (this.unsuspend) {
        var e = this.unsuspend;
        (this.unsuspend = null), e();
      }
    }
  }
  var Bs = null;
  function ss(e, t) {
    (e.stylesheets = null),
      e.unsuspend !== null &&
        (e.count++, (Bs = new Map()), t.forEach(mR, e), (Bs = null), Ps.call(e));
  }
  function mR(e, t) {
    if (!(t.state.loading & 4)) {
      var a = Bs.get(e);
      if (a) var r = a.get(null);
      else {
        (a = new Map()), Bs.set(e, a);
        for (
          var o = e.querySelectorAll('link[data-precedence],style[data-precedence]'), n = 0;
          n < o.length;
          n++
        ) {
          var l = o[n];
          (l.nodeName === 'LINK' || l.getAttribute('media') !== 'not all') &&
            (a.set(l.dataset.precedence, l), (r = l));
        }
        r && a.set(null, r);
      }
      (o = t.instance),
        (l = o.getAttribute('data-precedence')),
        (n = a.get(l) || r),
        n === r && a.set(null, o),
        a.set(l, o),
        this.count++,
        (r = Ps.bind(this)),
        o.addEventListener('load', r),
        o.addEventListener('error', r),
        n
          ? n.parentNode.insertBefore(o, n.nextSibling)
          : ((e = e.nodeType === 9 ? e.head : e), e.insertBefore(o, e.firstChild)),
        (t.state.loading |= 4);
    }
  }
  var Gl = {
    $$typeof: Ha,
    Provider: null,
    Consumer: null,
    _currentValue: Qr,
    _currentValue2: Qr,
    _threadCount: 0,
  };
  function pR(e, t, a, r, o, n, l, i, s) {
    (this.tag = 1),
      (this.containerInfo = e),
      (this.pingCache = this.current = this.pendingChildren = null),
      (this.timeoutHandle = -1),
      (this.callbackNode =
        this.next =
        this.pendingContext =
        this.context =
        this.cancelPendingCommit =
          null),
      (this.callbackPriority = 0),
      (this.expirationTimes = Rc(-1)),
      (this.entangledLanes =
        this.shellSuspendCounter =
        this.errorRecoveryDisabledLanes =
        this.expiredLanes =
        this.warmLanes =
        this.pingedLanes =
        this.suspendedLanes =
        this.pendingLanes =
          0),
      (this.entanglements = Rc(0)),
      (this.hiddenUpdates = Rc(null)),
      (this.identifierPrefix = r),
      (this.onUncaughtError = o),
      (this.onCaughtError = n),
      (this.onRecoverableError = l),
      (this.pooledCache = null),
      (this.pooledCacheLanes = 0),
      (this.formState = s),
      (this.incompleteTransitions = new Map());
  }
  function qv(e, t, a, r, o, n, l, i, s, u, d, c) {
    return (
      (e = new pR(e, t, a, l, s, u, d, c, i)),
      (t = 1),
      n === !0 && (t |= 24),
      (n = Et(3, null, null, t)),
      (e.current = n),
      (n.stateNode = e),
      (t = mf()),
      t.refCount++,
      (e.pooledCache = t),
      t.refCount++,
      (n.memoizedState = { element: r, isDehydrated: a, cache: t }),
      gf(n),
      e
    );
  }
  function Gv(e) {
    return e ? ((e = Qo), e) : Qo;
  }
  function Vv(e, t, a, r, o, n) {
    (o = Gv(o)),
      r.context === null ? (r.context = o) : (r.pendingContext = o),
      (r = Cr(t)),
      (r.payload = { element: a }),
      (n = n === void 0 ? null : n),
      n !== null && (r.callback = n),
      (a = wr(e, r, t)),
      a !== null && (xt(a, e, t), Cl(a, e, t));
  }
  function ry(e, t) {
    if (((e = e.memoizedState), e !== null && e.dehydrated !== null)) {
      var a = e.retryLane;
      e.retryLane = a !== 0 && a < t ? a : t;
    }
  }
  function qf(e, t) {
    ry(e, t), (e = e.alternate) && ry(e, t);
  }
  function jv(e) {
    if (e.tag === 13 || e.tag === 31) {
      var t = so(e, 67108864);
      t !== null && xt(t, e, 67108864), qf(e, 67108864);
    }
  }
  function oy(e) {
    if (e.tag === 13 || e.tag === 31) {
      var t = Pt();
      t = Jd(t);
      var a = so(e, t);
      a !== null && xt(a, e, t), qf(e, t);
    }
  }
  var Us = !0;
  function hR(e, t, a, r) {
    var o = N.T;
    N.T = null;
    var n = le.p;
    try {
      (le.p = 2), Gf(e, t, a, r);
    } finally {
      (le.p = n), (N.T = o);
    }
  }
  function gR(e, t, a, r) {
    var o = N.T;
    N.T = null;
    var n = le.p;
    try {
      (le.p = 8), Gf(e, t, a, r);
    } finally {
      (le.p = n), (N.T = o);
    }
  }
  function Gf(e, t, a, r) {
    if (Us) {
      var o = Xd(r);
      if (o === null) Zc(e, t, r, Ns, a), ny(e, r);
      else if (vR(o, e, t, a, r)) r.stopPropagation();
      else if ((ny(e, r), t & 4 && -1 < yR.indexOf(e))) {
        for (; o !== null; ) {
          var n = bn(o);
          if (n !== null)
            switch (n.tag) {
              case 3:
                if (((n = n.stateNode), n.current.memoizedState.isDehydrated)) {
                  var l = Yr(n.pendingLanes);
                  if (l !== 0) {
                    var i = n;
                    for (i.pendingLanes |= 2, i.entangledLanes |= 2; l; ) {
                      var s = 1 << (31 - Ot(l));
                      (i.entanglements[1] |= s), (l &= ~s);
                    }
                    Sa(n), (ne & 6) === 0 && ((Is = Tt() + 500), ti(0, !1));
                  }
                }
                break;
              case 31:
              case 13:
                (i = so(n, 2)), i !== null && xt(i, n, 2), Zs(), qf(n, 2);
            }
          if (((n = Xd(r)), n === null && Zc(e, t, r, Ns, a), n === o)) break;
          o = n;
        }
        o !== null && r.stopPropagation();
      } else Zc(e, t, r, null, a);
    }
  }
  function Xd(e) {
    return (e = af(e)), Vf(e);
  }
  var Ns = null;
  function Vf(e) {
    if (((Ns = null), (e = Go(e)), e !== null)) {
      var t = Yl(e);
      if (t === null) e = null;
      else {
        var a = t.tag;
        if (a === 13) {
          if (((e = dy(t)), e !== null)) return e;
          e = null;
        } else if (a === 31) {
          if (((e = fy(t)), e !== null)) return e;
          e = null;
        } else if (a === 3) {
          if (t.stateNode.current.memoizedState.isDehydrated)
            return t.tag === 3 ? t.stateNode.containerInfo : null;
          e = null;
        } else t !== e && (e = null);
      }
    }
    return (Ns = e), null;
  }
  function Yv(e) {
    switch (e) {
      case 'beforetoggle':
      case 'cancel':
      case 'click':
      case 'close':
      case 'contextmenu':
      case 'copy':
      case 'cut':
      case 'auxclick':
      case 'dblclick':
      case 'dragend':
      case 'dragstart':
      case 'drop':
      case 'focusin':
      case 'focusout':
      case 'input':
      case 'invalid':
      case 'keydown':
      case 'keypress':
      case 'keyup':
      case 'mousedown':
      case 'mouseup':
      case 'paste':
      case 'pause':
      case 'play':
      case 'pointercancel':
      case 'pointerdown':
      case 'pointerup':
      case 'ratechange':
      case 'reset':
      case 'resize':
      case 'seeked':
      case 'submit':
      case 'toggle':
      case 'touchcancel':
      case 'touchend':
      case 'touchstart':
      case 'volumechange':
      case 'change':
      case 'selectionchange':
      case 'textInput':
      case 'compositionstart':
      case 'compositionend':
      case 'compositionupdate':
      case 'beforeblur':
      case 'afterblur':
      case 'beforeinput':
      case 'blur':
      case 'fullscreenchange':
      case 'focus':
      case 'hashchange':
      case 'popstate':
      case 'select':
      case 'selectstart':
        return 2;
      case 'drag':
      case 'dragenter':
      case 'dragexit':
      case 'dragleave':
      case 'dragover':
      case 'mousemove':
      case 'mouseout':
      case 'mouseover':
      case 'pointermove':
      case 'pointerout':
      case 'pointerover':
      case 'scroll':
      case 'touchmove':
      case 'wheel':
      case 'mouseenter':
      case 'mouseleave':
      case 'pointerenter':
      case 'pointerleave':
        return 8;
      case 'message':
        switch (oC()) {
          case gy:
            return 2;
          case yy:
            return 8;
          case ms:
          case nC:
            return 32;
          case vy:
            return 268435456;
          default:
            return 32;
        }
      default:
        return 32;
    }
  }
  var Wd = !1,
    Ir = null,
    kr = null,
    Mr = null,
    Vl = new Map(),
    jl = new Map(),
    hr = [],
    yR =
      'mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset'.split(
        ' '
      );
  function ny(e, t) {
    switch (e) {
      case 'focusin':
      case 'focusout':
        Ir = null;
        break;
      case 'dragenter':
      case 'dragleave':
        kr = null;
        break;
      case 'mouseover':
      case 'mouseout':
        Mr = null;
        break;
      case 'pointerover':
      case 'pointerout':
        Vl.delete(t.pointerId);
        break;
      case 'gotpointercapture':
      case 'lostpointercapture':
        jl.delete(t.pointerId);
    }
  }
  function fl(e, t, a, r, o, n) {
    return e === null || e.nativeEvent !== n
      ? ((e = {
          blockedOn: t,
          domEventName: a,
          eventSystemFlags: r,
          nativeEvent: n,
          targetContainers: [o],
        }),
        t !== null && ((t = bn(t)), t !== null && jv(t)),
        e)
      : ((e.eventSystemFlags |= r),
        (t = e.targetContainers),
        o !== null && t.indexOf(o) === -1 && t.push(o),
        e);
  }
  function vR(e, t, a, r, o) {
    switch (t) {
      case 'focusin':
        return (Ir = fl(Ir, e, t, a, r, o)), !0;
      case 'dragenter':
        return (kr = fl(kr, e, t, a, r, o)), !0;
      case 'mouseover':
        return (Mr = fl(Mr, e, t, a, r, o)), !0;
      case 'pointerover':
        var n = o.pointerId;
        return Vl.set(n, fl(Vl.get(n) || null, e, t, a, r, o)), !0;
      case 'gotpointercapture':
        return (n = o.pointerId), jl.set(n, fl(jl.get(n) || null, e, t, a, r, o)), !0;
    }
    return !1;
  }
  function Xv(e) {
    var t = Go(e.target);
    if (t !== null) {
      var a = Yl(t);
      if (a !== null) {
        if (((t = a.tag), t === 13)) {
          if (((t = dy(a)), t !== null)) {
            (e.blockedOn = t),
              qh(e.priority, function () {
                oy(a);
              });
            return;
          }
        } else if (t === 31) {
          if (((t = fy(a)), t !== null)) {
            (e.blockedOn = t),
              qh(e.priority, function () {
                oy(a);
              });
            return;
          }
        } else if (t === 3 && a.stateNode.current.memoizedState.isDehydrated) {
          e.blockedOn = a.tag === 3 ? a.stateNode.containerInfo : null;
          return;
        }
      }
    }
    e.blockedOn = null;
  }
  function us(e) {
    if (e.blockedOn !== null) return !1;
    for (var t = e.targetContainers; 0 < t.length; ) {
      var a = Xd(e.nativeEvent);
      if (a === null) {
        a = e.nativeEvent;
        var r = new a.constructor(a.type, a);
        (fd = r), a.target.dispatchEvent(r), (fd = null);
      } else return (t = bn(a)), t !== null && jv(t), (e.blockedOn = a), !1;
      t.shift();
    }
    return !0;
  }
  function ly(e, t, a) {
    us(e) && a.delete(t);
  }
  function bR() {
    (Wd = !1),
      Ir !== null && us(Ir) && (Ir = null),
      kr !== null && us(kr) && (kr = null),
      Mr !== null && us(Mr) && (Mr = null),
      Vl.forEach(ly),
      jl.forEach(ly);
  }
  function Xi(e, t) {
    e.blockedOn === t &&
      ((e.blockedOn = null),
      Wd || ((Wd = !0), ze.unstable_scheduleCallback(ze.unstable_NormalPriority, bR)));
  }
  var Wi = null;
  function iy(e) {
    Wi !== e &&
      ((Wi = e),
      ze.unstable_scheduleCallback(ze.unstable_NormalPriority, function () {
        Wi === e && (Wi = null);
        for (var t = 0; t < e.length; t += 3) {
          var a = e[t],
            r = e[t + 1],
            o = e[t + 2];
          if (typeof r != 'function') {
            if (Vf(r || a) === null) continue;
            break;
          }
          var n = bn(a);
          n !== null &&
            (e.splice(t, 3),
            (t -= 3),
            kd(n, { pending: !0, data: o, method: a.method, action: r }, r, o));
        }
      }));
  }
  function yn(e) {
    function t(s) {
      return Xi(s, e);
    }
    Ir !== null && Xi(Ir, e),
      kr !== null && Xi(kr, e),
      Mr !== null && Xi(Mr, e),
      Vl.forEach(t),
      jl.forEach(t);
    for (var a = 0; a < hr.length; a++) {
      var r = hr[a];
      r.blockedOn === e && (r.blockedOn = null);
    }
    for (; 0 < hr.length && ((a = hr[0]), a.blockedOn === null); )
      Xv(a), a.blockedOn === null && hr.shift();
    if (((a = (e.ownerDocument || e).$$reactFormReplay), a != null))
      for (r = 0; r < a.length; r += 3) {
        var o = a[r],
          n = a[r + 1],
          l = o[St] || null;
        if (typeof n == 'function') l || iy(a);
        else if (l) {
          var i = null;
          if (n && n.hasAttribute('formAction')) {
            if (((o = n), (l = n[St] || null))) i = l.formAction;
            else if (Vf(o) !== null) continue;
          } else i = l.action;
          typeof i == 'function' ? (a[r + 1] = i) : (a.splice(r, 3), (r -= 3)), iy(a);
        }
      }
  }
  function Wv() {
    function e(n) {
      n.canIntercept &&
        n.info === 'react-transition' &&
        n.intercept({
          handler: function () {
            return new Promise(function (l) {
              return (o = l);
            });
          },
          focusReset: 'manual',
          scroll: 'manual',
        });
    }
    function t() {
      o !== null && (o(), (o = null)), r || setTimeout(a, 20);
    }
    function a() {
      if (!r && !navigation.transition) {
        var n = navigation.currentEntry;
        n &&
          n.url != null &&
          navigation.navigate(n.url, {
            state: n.getState(),
            info: 'react-transition',
            history: 'replace',
          });
      }
    }
    if (typeof navigation == 'object') {
      var r = !1,
        o = null;
      return (
        navigation.addEventListener('navigate', e),
        navigation.addEventListener('navigatesuccess', t),
        navigation.addEventListener('navigateerror', t),
        setTimeout(a, 100),
        function () {
          (r = !0),
            navigation.removeEventListener('navigate', e),
            navigation.removeEventListener('navigatesuccess', t),
            navigation.removeEventListener('navigateerror', t),
            o !== null && (o(), (o = null));
        }
      );
    }
  }
  function jf(e) {
    this._internalRoot = e;
  }
  eu.prototype.render = jf.prototype.render = function (e) {
    var t = this._internalRoot;
    if (t === null) throw Error(C(409));
    var a = t.current,
      r = Pt();
    Vv(a, r, e, t, null, null);
  };
  eu.prototype.unmount = jf.prototype.unmount = function () {
    var e = this._internalRoot;
    if (e !== null) {
      this._internalRoot = null;
      var t = e.containerInfo;
      Vv(e.current, 2, null, e, null, null), Zs(), (t[vn] = null);
    }
  };
  function eu(e) {
    this._internalRoot = e;
  }
  eu.prototype.unstable_scheduleHydration = function (e) {
    if (e) {
      var t = Cy();
      e = { blockedOn: null, target: e, priority: t };
      for (var a = 0; a < hr.length && t !== 0 && t < hr[a].priority; a++);
      hr.splice(a, 0, e), a === 0 && Xv(e);
    }
  };
  var sy = uy.version;
  if (sy !== '19.2.8') throw Error(C(527, sy, '19.2.8'));
  le.findDOMNode = function (e) {
    var t = e._reactInternals;
    if (t === void 0)
      throw typeof e.render == 'function'
        ? Error(C(188))
        : ((e = Object.keys(e).join(',')), Error(C(268, e)));
    return (e = Z1(t)), (e = e !== null ? my(e) : null), (e = e === null ? null : e.stateNode), e;
  };
  var xR = {
    bundleType: 0,
    version: '19.2.8',
    rendererPackageName: 'react-dom',
    currentDispatcherRef: N,
    reconcilerVersion: '19.2.8',
  };
  if (
    typeof __REACT_DEVTOOLS_GLOBAL_HOOK__ < 'u' &&
    ((ml = __REACT_DEVTOOLS_GLOBAL_HOOK__), !ml.isDisabled && ml.supportsFiber)
  )
    try {
      (Xl = ml.inject(xR)), (Dt = ml);
    } catch {}
  var ml;
  tu.createRoot = function (e, t) {
    if (!cy(e)) throw Error(C(299));
    var a = !1,
      r = '',
      o = H0,
      n = z0,
      l = F0;
    return (
      t != null &&
        (t.unstable_strictMode === !0 && (a = !0),
        t.identifierPrefix !== void 0 && (r = t.identifierPrefix),
        t.onUncaughtError !== void 0 && (o = t.onUncaughtError),
        t.onCaughtError !== void 0 && (n = t.onCaughtError),
        t.onRecoverableError !== void 0 && (l = t.onRecoverableError)),
      (t = qv(e, 1, !1, null, null, a, r, null, o, n, l, Wv)),
      (e[vn] = t.current),
      Hf(e),
      new jf(t)
    );
  };
  tu.hydrateRoot = function (e, t, a) {
    if (!cy(e)) throw Error(C(299));
    var r = !1,
      o = '',
      n = H0,
      l = z0,
      i = F0,
      s = null;
    return (
      a != null &&
        (a.unstable_strictMode === !0 && (r = !0),
        a.identifierPrefix !== void 0 && (o = a.identifierPrefix),
        a.onUncaughtError !== void 0 && (n = a.onUncaughtError),
        a.onCaughtError !== void 0 && (l = a.onCaughtError),
        a.onRecoverableError !== void 0 && (i = a.onRecoverableError),
        a.formState !== void 0 && (s = a.formState)),
      (t = qv(e, 1, !0, t, a ?? null, r, o, s, n, l, i, Wv)),
      (t.context = Gv(null)),
      (a = t.current),
      (r = Pt()),
      (r = Jd(r)),
      (o = Cr(r)),
      (o.callback = null),
      wr(a, o, r),
      (a = r),
      (t.current.lanes = a),
      Ql(t, a),
      Sa(t),
      (e[vn] = t.current),
      Hf(e),
      new eu(t)
    );
  };
  tu.version = '19.2.8';
});
var SR = pa((pA, Zv) => {
  'use strict';
  function Kv() {
    if (
      !(
        typeof __REACT_DEVTOOLS_GLOBAL_HOOK__ > 'u' ||
        typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE != 'function'
      )
    )
      try {
        __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Kv);
      } catch (e) {
        console.error(e);
      }
  }
  Kv(), (Zv.exports = Qv());
});
var Kb = pa((mu) => {
  'use strict';
  var nI = Symbol.for('react.transitional.element'),
    lI = Symbol.for('react.fragment');
  function Qb(e, t, a) {
    var r = null;
    if ((a !== void 0 && (r = '' + a), t.key !== void 0 && (r = '' + t.key), 'key' in t)) {
      a = {};
      for (var o in t) o !== 'key' && (a[o] = t[o]);
    } else a = t;
    return (t = a.ref), { $$typeof: nI, type: e, key: r, ref: t !== void 0 ? t : null, props: a };
  }
  mu.Fragment = lI;
  mu.jsx = Qb;
  mu.jsxs = Qb;
});
var $ = pa((FT, Zb) => {
  'use strict';
  Zb.exports = Kb();
});
var wt = E(te(), 1),
  D = E(te(), 1),
  ge = E(te(), 1),
  fm = E(te(), 1),
  Pb = E(te(), 1),
  Z = E(te(), 1),
  q_ = E(te(), 1),
  G_ = E(te(), 1),
  V_ = E(te(), 1),
  P = E(te(), 1),
  Xb = E(te(), 1);
var em = /^(?:[a-z][a-z0-9+.-]*:|[\\/]{2})/i,
  ib = /^[\\/]{2}/;
function LR(e, t) {
  return t + e.replace(/\\/g, '/');
}
var Jv = 'popstate';
function $v(e) {
  return (
    typeof e == 'object' &&
    e != null &&
    'pathname' in e &&
    'search' in e &&
    'hash' in e &&
    'state' in e &&
    'key' in e
  );
}
function sb(e = {}) {
  function t(r, o) {
    let n = o.state?.masked,
      { pathname: l, search: i, hash: s } = n || r.location;
    return Kf(
      '',
      { pathname: l, search: i, hash: s },
      (o.state && o.state.usr) || null,
      (o.state && o.state.key) || 'default',
      n
        ? { pathname: r.location.pathname, search: r.location.search, hash: r.location.hash }
        : void 0
    );
  }
  function a(r, o) {
    return typeof o == 'string' ? o : Br(o);
  }
  return wR(t, a, null, e);
}
function ve(e, t) {
  if (e === !1 || e === null || typeof e > 'u') throw new Error(t);
}
function Ct(e, t) {
  if (!e) {
    typeof console < 'u' && console.warn(t);
    try {
      throw new Error(t);
    } catch {}
  }
}
function CR() {
  return Math.random().toString(36).substring(2, 10);
}
function eb(e, t) {
  return {
    usr: e.state,
    key: e.key,
    idx: t,
    masked: e.mask ? { pathname: e.pathname, search: e.search, hash: e.hash } : void 0,
  };
}
function Kf(e, t, a = null, r, o) {
  return {
    pathname: typeof e == 'string' ? e : e.pathname,
    search: '',
    hash: '',
    ...(typeof t == 'string' ? co(t) : t),
    state: a,
    key: (t && t.key) || r || CR(),
    mask: o,
  };
}
function Br({ pathname: e = '/', search: t = '', hash: a = '' }) {
  return (
    t && t !== '?' && (e += t.charAt(0) === '?' ? t : '?' + t),
    a && a !== '#' && (e += a.charAt(0) === '#' ? a : '#' + a),
    e
  );
}
function co(e) {
  let t = {};
  if (e) {
    let a = e.indexOf('#');
    a >= 0 && ((t.hash = e.substring(a)), (e = e.substring(0, a)));
    let r = e.indexOf('?');
    r >= 0 && ((t.search = e.substring(r)), (e = e.substring(0, r))), e && (t.pathname = e);
  }
  return t;
}
function wR(e, t, a, r = {}) {
  let { window: o = document.defaultView, v5Compat: n = !1 } = r,
    l = o.history,
    i = 'POP',
    s = null,
    u = d();
  u == null && ((u = 0), l.replaceState({ ...l.state, idx: u }, ''));
  function d() {
    return (l.state || { idx: null }).idx;
  }
  function c() {
    i = 'POP';
    let y = d(),
      m = y == null ? null : y - u;
    (u = y), s && s({ action: i, location: x.location, delta: m });
  }
  function f(y, m) {
    i = 'PUSH';
    let p = $v(y) ? y : Kf(x.location, y, m);
    a && a(p, y), (u = d() + 1);
    let g = eb(p, u),
      b = x.createHref(p.mask || p);
    try {
      l.pushState(g, '', b);
    } catch (R) {
      if (R instanceof DOMException && R.name === 'DataCloneError') throw R;
      o.location.assign(b);
    }
    n && s && s({ action: i, location: x.location, delta: 1 });
  }
  function h(y, m) {
    i = 'REPLACE';
    let p = $v(y) ? y : Kf(x.location, y, m);
    a && a(p, y), (u = d());
    let g = eb(p, u),
      b = x.createHref(p.mask || p);
    l.replaceState(g, '', b), n && s && s({ action: i, location: x.location, delta: 0 });
  }
  function v(y) {
    return RR(o, y);
  }
  let x = {
    get action() {
      return i;
    },
    get location() {
      return e(o, l);
    },
    listen(y) {
      if (s) throw new Error('A history only accepts one active listener');
      return (
        o.addEventListener(Jv, c),
        (s = y),
        () => {
          o.removeEventListener(Jv, c), (s = null);
        }
      );
    },
    createHref(y) {
      return t(o, y);
    },
    createURL: v,
    encodeLocation(y) {
      let m = v(y);
      return { pathname: m.pathname, search: m.search, hash: m.hash };
    },
    push: f,
    replace: h,
    go(y) {
      return l.go(y);
    },
  };
  return x;
}
function RR(e, t, a = !1) {
  let r = 'http://localhost';
  e && (r = e.location.origin !== 'null' ? e.location.origin : e.location.href),
    ve(r, 'No window.location.(origin|href) available to create URL');
  let o = typeof t == 'string' ? t : Br(t);
  return (o = o.replace(/ $/, '%20')), !a && ib.test(o) && (o = r + o), new URL(o, r);
}
var _R;
_R = new WeakMap();
function tm(e, t, a = '/') {
  return IR(e, t, a, !1);
}
function IR(e, t, a, r, o) {
  let n = typeof t == 'string' ? co(t) : t,
    l = La(n.pathname || '/', a);
  if (l == null) return null;
  let i = o ?? MR(e),
    s = null,
    u = zR(l);
  for (let d = 0; s == null && d < i.length; ++d) s = HR(i[d], u, r);
  return s;
}
function kR(e, t) {
  let { route: a, pathname: r, params: o } = e;
  return { id: a.id, pathname: r, params: o, data: t[a.id], loaderData: t[a.id], handle: a.handle };
}
function MR(e) {
  let t = ub(e);
  return ER(t), t;
}
function ub(e, t = [], a = [], r = '', o = !1) {
  let n = (l, i, s = o, u) => {
    let d = {
      relativePath: u === void 0 ? l.path || '' : u,
      caseSensitive: l.caseSensitive === !0,
      childrenIndex: i,
      route: l,
    };
    if (d.relativePath.startsWith('/')) {
      if (!d.relativePath.startsWith(r) && s) return;
      ve(
        d.relativePath.startsWith(r),
        `Absolute route path "${d.relativePath}" nested under path "${r}" is not valid. An absolute child route path must start with the combined path of all its parent routes.`
      ),
        (d.relativePath = d.relativePath.slice(r.length));
    }
    let c = sa([r, d.relativePath]),
      f = a.concat(d);
    l.children &&
      l.children.length > 0 &&
      (ve(
        l.index !== !0,
        `Index routes must not have child routes. Please remove all child routes from route path "${c}".`
      ),
      ub(l.children, t, f, c, s)),
      !(l.path == null && !l.index) &&
        t.push({
          path: c,
          score: UR(c, l.index),
          routesMeta: f.map((h, v) => {
            let [x, y] = fb(h.relativePath, h.caseSensitive, v === f.length - 1);
            return { ...h, matcher: x, compiledParams: y };
          }),
        });
  };
  return (
    e.forEach((l, i) => {
      if (l.path === '' || !l.path?.includes('?')) n(l, i);
      else for (let s of cb(l.path)) n(l, i, !0, s);
    }),
    t
  );
}
function cb(e) {
  let t = e.split('/');
  if (t.length === 0) return [];
  let [a, ...r] = t,
    o = a.endsWith('?'),
    n = a.replace(/\?$/, '');
  if (r.length === 0) return o ? [n, ''] : [n];
  let l = cb(r.join('/')),
    i = [];
  return (
    i.push(...l.map((s) => (s === '' ? n : [n, s].join('/')))),
    o && i.push(...l),
    i.map((s) => (e.startsWith('/') && s === '' ? '/' : s))
  );
}
function ER(e) {
  e.sort((t, a) =>
    t.score !== a.score
      ? a.score - t.score
      : NR(
          t.routesMeta.map((r) => r.childrenIndex),
          a.routesMeta.map((r) => r.childrenIndex)
        )
  );
}
var AR = /^:[\w-]+$/,
  TR = 3,
  DR = 2,
  OR = 1,
  PR = 10,
  BR = -2,
  tb = (e) => e === '*';
function UR(e, t) {
  let a = e.split('/'),
    r = a.length;
  return (
    a.some(tb) && (r += BR),
    t && (r += DR),
    a.filter((o) => !tb(o)).reduce((o, n) => o + (AR.test(n) ? TR : n === '' ? OR : PR), r)
  );
}
function NR(e, t) {
  return e.length === t.length && e.slice(0, -1).every((r, o) => r === t[o])
    ? e[e.length - 1] - t[t.length - 1]
    : 0;
}
function HR(e, t, a = !1) {
  let { routesMeta: r } = e,
    o = {},
    n = '/',
    l = [];
  for (let i = 0; i < r.length; ++i) {
    let s = r[i],
      u = i === r.length - 1,
      d = n === '/' ? t : t.slice(n.length) || '/',
      c = { path: s.relativePath, caseSensitive: s.caseSensitive, end: u },
      f = s.matcher && s.compiledParams ? db(c, d, s.matcher, s.compiledParams) : ni(c, d),
      h = s.route;
    if (
      (!f &&
        u &&
        a &&
        !r[r.length - 1].route.index &&
        (f = ni({ path: s.relativePath, caseSensitive: s.caseSensitive, end: !1 }, d)),
      !f)
    )
      return null;
    Object.assign(o, f.params),
      l.push({
        params: o,
        pathname: sa([n, f.pathname]),
        pathnameBase: qR(sa([n, f.pathnameBase])),
        route: h,
      }),
      f.pathnameBase !== '/' && (n = sa([n, f.pathnameBase]));
  }
  return l;
}
function ni(e, t) {
  typeof e == 'string' && (e = { path: e, caseSensitive: !1, end: !0 });
  let [a, r] = fb(e.path, e.caseSensitive, e.end);
  return db(e, t, a, r);
}
function db(e, t, a, r) {
  let o = t.match(a);
  if (!o) return null;
  let n = o[0],
    l = Rn(n, 1),
    i = o.slice(1);
  return {
    params: r.reduce((u, { paramName: d, isOptional: c }, f) => {
      if (d === '*') {
        let v = i[f] || '';
        l = Rn(n.slice(0, n.length - v.length), 1);
      }
      let h = i[f];
      return c && !h ? (u[d] = void 0) : (u[d] = (h || '').replace(/%2F/g, '/')), u;
    }, {}),
    pathname: n,
    pathnameBase: l,
    pattern: e,
  };
}
function fb(e, t = !1, a = !0) {
  Ct(
    e === '*' || !e.endsWith('*') || e.endsWith('/*'),
    `Route path "${e}" will be treated as if it were "${e.replace(/\*$/, '/*')}" because the \`*\` character must always follow a \`/\` in the pattern. To get rid of this warning, please change the route path to "${e.replace(/\*$/, '/*')}".`
  );
  let r = [],
    o =
      '^' +
      e
        .replace(/\/*\*?$/, '')
        .replace(/^\/*/, '/')
        .replace(/[\\.*+^${}|()[\]]/g, '\\$&')
        .replace(/\/:([\w-]+)(\?)?/g, (l, i, s, u, d) => {
          if ((r.push({ paramName: i, isOptional: s != null }), s)) {
            let c = d.charAt(u + l.length);
            return c && c !== '/' ? '/([^\\/]*)' : '(?:/([^\\/]*))?';
          }
          return '/([^\\/]+)';
        })
        .replace(/\/([\w-]+)\?(\/|$)/g, '(/$1)?$2');
  return (
    e.endsWith('*')
      ? (r.push({ paramName: '*' }), (o += e === '*' || e === '/*' ? '(.*)$' : '(?:\\/(.+)|\\/*)$'))
      : a
        ? (o += '\\/*$')
        : e !== '' && e !== '/' && (o += '(?:(?=\\/|$))'),
    [new RegExp(o, t ? void 0 : 'i'), r]
  );
}
function zR(e) {
  try {
    return e
      .split('/')
      .map((t) => decodeURIComponent(t).replace(/\//g, '%2F'))
      .join('/');
  } catch (t) {
    return (
      Ct(
        !1,
        `The URL path "${e}" could not be decoded because it is a malformed URL segment. This is probably due to a bad percent encoding (${t}).`
      ),
      e
    );
  }
}
function La(e, t) {
  if (t === '/') return e;
  if (!e.toLowerCase().startsWith(t.toLowerCase())) return null;
  let a = t.endsWith('/') ? t.length - 1 : t.length,
    r = e.charAt(a);
  return r && r !== '/' ? null : e.slice(a) || '/';
}
function mb(e, t = '/') {
  let { pathname: a, search: r = '', hash: o = '' } = typeof e == 'string' ? co(e) : e,
    n;
  return (
    a
      ? ((a = pb(a)),
        a.startsWith('/') || a.startsWith('\\') ? (n = ab(a.substring(1), '/')) : (n = ab(a, t)))
      : (n = t),
    { pathname: n, search: GR(r), hash: VR(o) }
  );
}
function ab(e, t) {
  let a = Rn(t).split('/');
  return (
    e.split('/').forEach((o) => {
      o === '..' ? a.length > 1 && a.pop() : o !== '.' && a.push(o);
    }),
    a.length > 1 ? a.join('/') : '/'
  );
}
function Yf(e, t, a, r) {
  return `Cannot include a '${e}' character in a manually specified \`to.${t}\` field [${JSON.stringify(r)}].  Please separate it out to the \`to.${a}\` field. Alternatively you may provide the full path as a string in <Link to="..."> and the router will parse it for you.`;
}
function FR(e) {
  return e.filter((t, a) => a === 0 || (t.route.path && t.route.path.length > 0));
}
function am(e) {
  let t = FR(e);
  return t.map((a, r) => (r === t.length - 1 ? a.pathname : a.pathnameBase));
}
function su(e, t, a, r = !1) {
  let o;
  typeof e == 'string'
    ? (o = co(e))
    : ((o = { ...e }),
      ve(!o.pathname || !o.pathname.includes('?'), Yf('?', 'pathname', 'search', o)),
      ve(!o.pathname || !o.pathname.includes('#'), Yf('#', 'pathname', 'hash', o)),
      ve(!o.search || !o.search.includes('#'), Yf('#', 'search', 'hash', o)));
  let n = e === '' || o.pathname === '',
    l = n ? '/' : o.pathname,
    i;
  if (l == null) i = a;
  else {
    let c = t.length - 1;
    if (!r && l.startsWith('..')) {
      let f = l.split('/');
      for (; f[0] === '..'; ) f.shift(), (c -= 1);
      o.pathname = f.join('/');
    }
    i = c >= 0 ? t[c] : '/';
  }
  let s = mb(o, i),
    u = l && l !== '/' && l.endsWith('/'),
    d = (n || l === '.') && a.endsWith('/');
  return !s.pathname.endsWith('/') && (u || d) && (s.pathname += '/'), s;
}
var pb = (e) => e.replace(/[\\/]{2,}/g, '/'),
  sa = (e) => pb(e.join('/'));
function Rn(e, t = 0) {
  let a = e.length;
  for (; a > t && e.charCodeAt(a - 1) === 47; ) a--;
  return a === e.length ? e : e.slice(0, a);
}
var qR = (e) => Rn(e).replace(/^\/*/, '/'),
  GR = (e) => (!e || e === '?' ? '' : e.startsWith('?') ? e : '?' + e),
  VR = (e) => (!e || e === '#' ? '' : e.startsWith('#') ? e : '#' + e);
var hb = class {
  constructor(e, t, a, r = !1) {
    (this.status = e),
      (this.statusText = t || ''),
      (this.internal = r),
      a instanceof Error ? ((this.data = a.toString()), (this.error = a)) : (this.data = a);
  }
};
function gb(e) {
  return (
    e != null &&
    typeof e.status == 'number' &&
    typeof e.statusText == 'string' &&
    typeof e.internal == 'boolean' &&
    'data' in e
  );
}
function jR(e) {
  let t = e.map((a) => a.route.path).filter(Boolean);
  return sa(t) || '/';
}
var yb =
  typeof window < 'u' && typeof window.document < 'u' && typeof window.document.createElement < 'u';
function vb(e, t) {
  let a = e;
  if (typeof a != 'string' || !em.test(a)) return { absoluteURL: void 0, isExternal: !1, to: a };
  let r = a,
    o = !1;
  if (yb)
    try {
      let n = new URL(window.location.href),
        l = ib.test(a) ? new URL(LR(a, n.protocol)) : new URL(a),
        i = La(l.pathname, t);
      l.origin === n.origin && i != null ? (a = i + l.search + l.hash) : (o = !0);
    } catch {
      Ct(
        !1,
        `<Link to="${a}"> contains an invalid URL which will probably break when clicked - please update to a valid URL path.`
      );
    }
  return { absoluteURL: r, isExternal: o, to: a };
}
var hA = Symbol('Uninstrumented');
var gA = Object.getOwnPropertyNames(Object.prototype).sort().join('\0');
var rb = new URL('http://localhost');
function rm(e) {
  if (e.createURL) return e.createURL('/');
  try {
    return new URL(e.createHref('/'), rb);
  } catch {
    return rb;
  }
}
function Xf(e, t) {
  return (
    e.origin === t.origin &&
    (e.origin !== 'null' || (e.protocol === t.protocol && e.host === t.host))
  );
}
function YR(e, t) {
  if (e.startsWith('//')) return !0;
  let a = t.protocol.toLowerCase();
  return e.toLowerCase().startsWith(a) ? t.host === '' || e.slice(a.length).startsWith('//') : !1;
}
function om(e, t, a, r) {
  let o = null;
  try {
    o = e == null ? null : new URL(e, a);
  } catch {}
  let n = new URL(t, a),
    l = o != null && !Xf(o, a),
    i = !Xf(n, a);
  if (r === 'reject') {
    if (l || i) throw new Error('External navigation is not allowed');
  } else if (i && (o == null || !YR(e, o) || !Xf(o, n)))
    throw new Error('External navigation is not allowed');
}
var bb = ['POST', 'PUT', 'PATCH', 'DELETE'],
  yA = new Set(bb),
  XR = ['GET', ...bb],
  vA = new Set(XR);
var bA = Symbol('ResetLoaderData'),
  WR,
  QR,
  KR,
  ZR;
WR = new WeakMap();
QR = new WeakMap();
KR = new WeakMap();
ZR = new WeakMap();
var JR = [
  'about:',
  'blob:',
  'chrome:',
  'chrome-untrusted:',
  'content:',
  'data:',
  'devtools:',
  'file:',
  'filesystem:',
  'javascript:',
];
function $R(e) {
  try {
    return JR.includes(new URL(e).protocol);
  } catch {
    return !1;
  }
}
var fo = wt.createContext(null);
fo.displayName = 'DataRouter';
var _n = wt.createContext(null);
_n.displayName = 'DataRouterState';
var xb = wt.createContext(!1);
function e_() {
  return wt.useContext(xb);
}
var nm = wt.createContext({ isTransitioning: !1 });
nm.displayName = 'ViewTransition';
var Sb = wt.createContext(new Map());
Sb.displayName = 'Fetchers';
var t_ = wt.createContext(null);
t_.displayName = 'Await';
var st = wt.createContext(null);
st.displayName = 'Navigation';
var In = wt.createContext(null);
In.displayName = 'Location';
var Nt = wt.createContext({ outlet: null, matches: [], isDataRoute: !1 });
Nt.displayName = 'Route';
var lm = wt.createContext(null);
lm.displayName = 'RouteError';
var Zf = !0,
  Lb = 'REACT_ROUTER_ERROR',
  a_ = 'REDIRECT',
  r_ = 'ROUTE_ERROR_RESPONSE';
function o_(e) {
  if (e.startsWith(`${Lb}:${a_}:{`))
    try {
      let t = JSON.parse(e.slice(28));
      if (
        typeof t == 'object' &&
        t &&
        typeof t.status == 'number' &&
        typeof t.statusText == 'string' &&
        typeof t.location == 'string' &&
        typeof t.reloadDocument == 'boolean' &&
        typeof t.replace == 'boolean'
      )
        return t;
    } catch {}
}
function n_(e) {
  if (e.startsWith(`${Lb}:${r_}:{`))
    try {
      let t = JSON.parse(e.slice(40));
      if (
        typeof t == 'object' &&
        t &&
        typeof t.status == 'number' &&
        typeof t.statusText == 'string'
      )
        return new hb(t.status, t.statusText, t.data);
    } catch {}
}
function Cb(e, { relative: t } = {}) {
  ve(mo(), 'useHref() may be used only in the context of a <Router> component.');
  let { basename: a, navigator: r } = D.useContext(st),
    { hash: o, pathname: n, search: l } = kn(e, { relative: t }),
    i = n;
  return (
    a !== '/' && (i = n === '/' ? a : sa([a, n])), r.createHref({ pathname: i, search: l, hash: o })
  );
}
function mo() {
  return D.useContext(In) != null;
}
function Rt() {
  return (
    ve(mo(), 'useLocation() may be used only in the context of a <Router> component.'),
    D.useContext(In).location
  );
}
var wb =
  'You should call navigate() in a React.useEffect(), not when your component is first rendered.';
function Rb(e) {
  D.useContext(st).static || D.useLayoutEffect(e);
}
function uu() {
  let { isDataRoute: e } = D.useContext(Nt);
  return e ? y_() : l_();
}
function l_() {
  ve(mo(), 'useNavigate() may be used only in the context of a <Router> component.');
  let e = D.useContext(fo),
    { basename: t, navigator: a } = D.useContext(st),
    { matches: r } = D.useContext(Nt),
    { pathname: o } = Rt(),
    n = JSON.stringify(am(r)),
    l = D.useRef(!1);
  return (
    Rb(() => {
      l.current = !0;
    }),
    D.useCallback(
      (s, u = {}) => {
        if ((Ct(l.current, wb), !l.current)) return;
        if (typeof s == 'number') {
          a.go(s);
          return;
        }
        let d = su(s, JSON.parse(n), o, u.relative === 'path');
        e == null && t !== '/' && (d.pathname = d.pathname === '/' ? t : sa([t, d.pathname])),
          om(typeof s == 'string' ? s : Br(s), a.createHref(d), rm(a), 'reject'),
          (u.replace ? a.replace : a.push)(d, u.state, u);
      },
      [t, a, n, o, e]
    )
  );
}
var i_ = D.createContext(null);
function _b(e) {
  let t = D.useContext(Nt).outlet;
  return D.useMemo(() => t && D.createElement(i_.Provider, { value: e }, t), [t, e]);
}
function s_() {
  let { matches: e } = D.useContext(Nt);
  return e[e.length - 1]?.params ?? {};
}
function kn(e, { relative: t } = {}) {
  let { matches: a } = D.useContext(Nt),
    { pathname: r } = Rt(),
    o = JSON.stringify(am(a));
  return D.useMemo(() => su(e, JSON.parse(o), r, t === 'path'), [e, o, r, t]);
}
function Ib(e, t) {
  return kb(e, t);
}
function kb(e, t, a) {
  ve(mo(), 'useRoutes() may be used only in the context of a <Router> component.');
  let { navigator: r } = D.useContext(st),
    { matches: o } = D.useContext(Nt),
    n = o[o.length - 1],
    l = n ? n.params : {},
    i = n ? n.pathname : '/',
    s = n ? n.pathnameBase : '/',
    u = n && n.route;
  if (Zf) {
    let y = (u && u.path) || '';
    Tb(
      i,
      !u || y.endsWith('*') || y.endsWith('*?'),
      `You rendered descendant <Routes> (or called \`useRoutes()\`) at "${i}" (under <Route path="${y}">) but the parent route path has no trailing "*". This means if you navigate deeper, the parent won't match anymore and therefore the child routes will never render.

Please change the parent <Route path="${y}"> to <Route path="${y === '/' ? '*' : `${y}/*`}">.`
    );
  }
  let d = Rt(),
    c;
  if (t) {
    let y = typeof t == 'string' ? co(t) : t;
    ve(
      s === '/' || y.pathname?.startsWith(s),
      `When overriding the location using \`<Routes location>\` or \`useRoutes(routes, location)\`, the location pathname must begin with the portion of the URL pathname that was matched by all parent routes. The current pathname base is "${s}" but pathname "${y.pathname}" was given in the \`location\` prop.`
    ),
      (c = y);
  } else c = d;
  let f = c.pathname || '/',
    h = f;
  if (s !== '/') {
    let y = s.replace(/^\//, '').split('/');
    h = '/' + f.replace(/^\//, '').split('/').slice(y.length).join('/');
  }
  let v =
    a && a.state.matches.length
      ? a.state.matches.map((y) => Object.assign(y, { route: a.manifest[y.route.id] || y.route }))
      : tm(e, { pathname: h });
  Zf &&
    (Ct(u || v != null, `No routes matched location "${c.pathname}${c.search}${c.hash}" `),
    Ct(
      v == null ||
        v[v.length - 1].route.element !== void 0 ||
        v[v.length - 1].route.Component !== void 0 ||
        v[v.length - 1].route.lazy !== void 0,
      `Matched leaf route at location "${c.pathname}${c.search}${c.hash}" does not have an element or Component. This means it will render an <Outlet /> with a null value by default resulting in an "empty" page.`
    ));
  let x = m_(
    v &&
      v.map((y) =>
        Object.assign({}, y, {
          params: Object.assign({}, l, y.params),
          pathname: sa([
            s,
            r.encodeLocation
              ? r.encodeLocation(
                  y.pathname.replace(/%/g, '%25').replace(/\?/g, '%3F').replace(/#/g, '%23')
                ).pathname
              : y.pathname,
          ]),
          pathnameBase:
            y.pathnameBase === '/'
              ? s
              : sa([
                  s,
                  r.encodeLocation
                    ? r.encodeLocation(
                        y.pathnameBase
                          .replace(/%/g, '%25')
                          .replace(/\?/g, '%3F')
                          .replace(/#/g, '%23')
                      ).pathname
                    : y.pathnameBase,
                ]),
        })
      ),
    o,
    a
  );
  return t && x
    ? D.createElement(
        In.Provider,
        {
          value: {
            location: {
              pathname: '/',
              search: '',
              hash: '',
              state: null,
              key: 'default',
              mask: void 0,
              ...c,
            },
            navigationType: 'POP',
          },
        },
        x
      )
    : x;
}
function u_() {
  let e = Ab(),
    t = gb(e) ? `${e.status} ${e.statusText}` : e instanceof Error ? e.message : JSON.stringify(e),
    a = e instanceof Error ? e.stack : null,
    r = 'rgba(200,200,200, 0.5)',
    o = { padding: '0.5rem', backgroundColor: r },
    n = { padding: '2px 4px', backgroundColor: r },
    l = null;
  return (
    Zf &&
      (console.error('Error handled by React Router default ErrorBoundary:', e),
      (l = D.createElement(
        D.Fragment,
        null,
        D.createElement('p', null, '\u{1F4BF} Hey developer \u{1F44B}'),
        D.createElement(
          'p',
          null,
          'You can provide a way better UX than this when your app throws errors by providing your own ',
          D.createElement('code', { style: n }, 'ErrorBoundary'),
          ' or',
          ' ',
          D.createElement('code', { style: n }, 'errorElement'),
          ' prop on your route.'
        )
      ))),
    D.createElement(
      D.Fragment,
      null,
      D.createElement('h2', null, 'Unexpected Application Error!'),
      D.createElement('h3', { style: { fontStyle: 'italic' } }, t),
      a ? D.createElement('pre', { style: o }, a) : null,
      l
    )
  );
}
var c_ = D.createElement(u_, null),
  Mb = class extends D.Component {
    constructor(e) {
      super(e),
        (this.state = { location: e.location, revalidation: e.revalidation, error: e.error });
    }
    static getDerivedStateFromError(e) {
      return { error: e };
    }
    static getDerivedStateFromProps(e, t) {
      return t.location !== e.location || (t.revalidation !== 'idle' && e.revalidation === 'idle')
        ? { error: e.error, location: e.location, revalidation: e.revalidation }
        : {
            error: e.error !== void 0 ? e.error : t.error,
            location: t.location,
            revalidation: e.revalidation || t.revalidation,
          };
    }
    componentDidCatch(e, t) {
      this.props.onError
        ? this.props.onError(e, t)
        : console.error('React Router caught the following error during render', e);
    }
    render() {
      let e = this.state.error;
      if (
        this.context &&
        typeof e == 'object' &&
        e &&
        'digest' in e &&
        typeof e.digest == 'string'
      ) {
        let a = n_(e.digest);
        a && (e = a);
      }
      let t =
        e !== void 0
          ? D.createElement(
              Nt.Provider,
              { value: this.props.routeContext },
              D.createElement(lm.Provider, { value: e, children: this.props.component })
            )
          : this.props.children;
      return this.context ? D.createElement(d_, { error: e }, t) : t;
    }
  };
Mb.contextType = xb;
var Wf = new WeakMap();
function d_({ children: e, error: t }) {
  let { basename: a, navigator: r } = D.useContext(st);
  if (typeof t == 'object' && t && 'digest' in t && typeof t.digest == 'string') {
    let o = o_(t.digest);
    if (o) {
      let n = Wf.get(t);
      if (n) throw n;
      let l = vb(o.location, a),
        i = l.absoluteURL || l.to;
      if ((om(o.location, i, rm(r), 'allow-explicit'), $R(i)))
        throw new Error('Invalid redirect location');
      if (yb && !Wf.get(t))
        if (l.isExternal || o.reloadDocument) window.location.href = i;
        else {
          let s = Promise.resolve().then(() =>
            window.__reactRouterDataRouter.navigate(l.to, { replace: o.replace })
          );
          throw (Wf.set(t, s), s);
        }
      return D.createElement('meta', { httpEquiv: 'refresh', content: `0;url=${i}` });
    }
  }
  return e;
}
function f_({ routeContext: e, match: t, children: a }) {
  let r = D.useContext(fo);
  return (
    r &&
      r.static &&
      r.staticContext &&
      (t.route.errorElement || t.route.ErrorBoundary) &&
      (r.staticContext._deepestRenderedBoundaryId = t.route.id),
    D.createElement(Nt.Provider, { value: e }, a)
  );
}
function m_(e, t = [], a) {
  let r = a?.state;
  if (e == null) {
    if (!r) return null;
    if (r.errors) e = r.matches;
    else if (t.length === 0 && !r.initialized && r.matches.length > 0) e = r.matches;
    else return null;
  }
  let o = e,
    n = r?.errors;
  if (n != null) {
    let d = o.findIndex((c) => c.route.id && n?.[c.route.id] !== void 0);
    ve(
      d >= 0,
      `Could not find a matching route for errors on route IDs: ${Object.keys(n).join(',')}`
    ),
      (o = o.slice(0, Math.min(o.length, d + 1)));
  }
  let l = !1,
    i = -1;
  if (a && r) {
    l = r.renderFallback;
    for (let d = 0; d < o.length; d++) {
      let c = o[d];
      if (((c.route.HydrateFallback || c.route.hydrateFallbackElement) && (i = d), c.route.id)) {
        let { loaderData: f, errors: h } = r,
          v = c.route.loader && !f.hasOwnProperty(c.route.id) && (!h || h[c.route.id] === void 0);
        if (c.route.lazy || v) {
          a.isStatic && (l = !0), i >= 0 ? (o = o.slice(0, i + 1)) : (o = [o[0]]);
          break;
        }
      }
    }
  }
  let s = a?.onError,
    u =
      r && s
        ? (d, c) => {
            s(d, {
              location: r.location,
              params: r.matches?.[0]?.params ?? {},
              pattern: jR(r.matches),
              errorInfo: c,
            });
          }
        : void 0;
  return o.reduceRight((d, c, f) => {
    let h,
      v = !1,
      x = null,
      y = null;
    r &&
      ((h = n && c.route.id ? n[c.route.id] : void 0),
      (x = c.route.errorElement || c_),
      l &&
        (i < 0 && f === 0
          ? (Tb(
              'route-fallback',
              !1,
              'No `HydrateFallback` element provided to render during initial hydration'
            ),
            (v = !0),
            (y = null))
          : i === f && ((v = !0), (y = c.route.hydrateFallbackElement || null))));
    let m = t.concat(o.slice(0, f + 1)),
      p = () => {
        let g;
        return (
          h
            ? (g = x)
            : v
              ? (g = y)
              : c.route.Component
                ? (g = D.createElement(c.route.Component, null))
                : c.route.element
                  ? (g = c.route.element)
                  : (g = d),
          D.createElement(f_, {
            match: c,
            routeContext: { outlet: d, matches: m, isDataRoute: r != null },
            children: g,
          })
        );
      };
    return r && (c.route.ErrorBoundary || c.route.errorElement || f === 0)
      ? D.createElement(Mb, {
          location: r.location,
          revalidation: r.revalidation,
          component: x,
          error: h,
          children: p(),
          routeContext: { outlet: null, matches: m, isDataRoute: !0 },
          onError: u,
        })
      : p();
  }, null);
}
function im(e) {
  return `${e} must be used within a data router.  See https://reactrouter.com/en/main/routers/picking-a-router.`;
}
function p_(e) {
  let t = D.useContext(fo);
  return ve(t, im(e)), t;
}
function sm(e) {
  let t = D.useContext(_n);
  return ve(t, im(e)), t;
}
function h_(e) {
  let t = D.useContext(Nt);
  return ve(t, im(e)), t;
}
function um(e) {
  let t = h_(e),
    a = t.matches[t.matches.length - 1];
  return ve(a.route.id, `${e} can only be used on routes that contain a unique "id"`), a.route.id;
}
function g_() {
  return um('useRouteId');
}
function Eb() {
  let e = sm('useNavigation');
  return D.useMemo(() => {
    let { matches: t, historyAction: a, ...r } = e.navigation;
    return r;
  }, [e.navigation]);
}
function cm() {
  let { matches: e, loaderData: t } = sm('useMatches');
  return D.useMemo(() => e.map((a) => kR(a, t)), [e, t]);
}
function Ab() {
  let e = D.useContext(lm),
    t = sm('useRouteError'),
    a = um('useRouteError');
  return e !== void 0 ? e : t.errors?.[a];
}
function y_() {
  let { router: e } = p_('useNavigate'),
    t = um('useNavigate'),
    a = D.useRef(!1);
  return (
    Rb(() => {
      a.current = !0;
    }),
    D.useCallback(
      async (o, n = {}) => {
        Ct(a.current, wb),
          a.current &&
            (typeof o == 'number'
              ? await e.navigate(o)
              : await e.navigate(o, { fromRouteId: t, ...n }));
      },
      [e, t]
    )
  );
}
var ob = {};
function Tb(e, t, a) {
  !t && !ob[e] && ((ob[e] = !0), Ct(!1, a));
}
var v_ = 'useOptimistic',
  xA = ge[v_];
var SA = ge.memo(b_);
function b_({ routes: e, manifest: t, future: a, state: r, isStatic: o, onError: n }) {
  return kb(e, void 0, { manifest: t, state: r, isStatic: o, onError: n, future: a });
}
function po({ to: e, replace: t, state: a, relative: r }) {
  ve(mo(), '<Navigate> may be used only in the context of a <Router> component.');
  let { static: o, navigator: n } = ge.useContext(st);
  Ct(
    !o,
    '<Navigate> must not be used on the initial render in a <StaticRouter>. This is a no-op, but you should modify your code so the <Navigate> is only ever rendered in response to some user interaction or state change.'
  );
  let { matches: l } = ge.useContext(Nt),
    { pathname: i } = Rt(),
    s = uu(),
    u = su(e, am(l), i, r === 'path');
  om(typeof e == 'string' ? e : Br(e), n.createHref(u), rm(n), 'reject');
  let d = JSON.stringify(u);
  return (
    ge.useEffect(() => {
      s(JSON.parse(d), { replace: t, state: a, relative: r });
    }, [s, d, r, t, a]),
    null
  );
}
function x_(e) {
  return _b(e.context);
}
function Db(e) {
  ve(
    !1,
    'A <Route> is only ever to be used as the child of <Routes> element, never rendered directly. Please wrap your <Route> in a <Routes>.'
  );
}
function dm({
  basename: e = '/',
  children: t = null,
  location: a,
  navigationType: r = 'POP',
  navigator: o,
  static: n = !1,
  useTransitions: l,
}) {
  ve(
    !mo(),
    'You cannot render a <Router> inside another <Router>. You should never have more than one in your app.'
  );
  let i = e.replace(/^\/*/, '/'),
    s = ge.useMemo(
      () => ({ basename: i, navigator: o, static: n, useTransitions: l, future: {} }),
      [i, o, n, l]
    );
  typeof a == 'string' && (a = co(a));
  let {
      pathname: u = '/',
      search: d = '',
      hash: c = '',
      state: f = null,
      key: h = 'default',
      mask: v,
    } = a,
    x = ge.useMemo(() => {
      let y = La(u, i);
      return y == null
        ? null
        : {
            location: { pathname: y, search: d, hash: c, state: f, key: h, mask: v },
            navigationType: r,
          };
    }, [i, u, d, c, f, h, r, v]);
  return (
    Ct(
      x != null,
      `<Router basename="${i}"> is not able to match the URL "${u}${d}${c}" because it does not start with the basename, so the <Router> won't render anything.`
    ),
    x == null
      ? null
      : ge.createElement(
          st.Provider,
          { value: s },
          ge.createElement(In.Provider, { children: t, value: x })
        )
  );
}
function S_({ children: e, location: t }) {
  return Ib(lu(e), t);
}
function lu(e, t = []) {
  let a = [];
  return (
    ge.Children.forEach(e, (r, o) => {
      if (!ge.isValidElement(r)) return;
      let n = [...t, o];
      if (r.type === ge.Fragment) {
        a.push.apply(a, lu(r.props.children, n));
        return;
      }
      ve(
        r.type === Db,
        `[${typeof r.type == 'string' ? r.type : r.type.name}] is not a <Route> component. All component children of <Routes> must be a <Route> or <React.Fragment>`
      ),
        ve(!r.props.index || !r.props.children, 'An index route cannot have child routes.');
      let l = {
        id: r.props.id || n.join('-'),
        caseSensitive: r.props.caseSensitive,
        element: r.props.element,
        Component: r.props.Component,
        index: r.props.index,
        path: r.props.path,
        middleware: r.props.middleware,
        loader: r.props.loader,
        action: r.props.action,
        hydrateFallbackElement: r.props.hydrateFallbackElement,
        HydrateFallback: r.props.HydrateFallback,
        errorElement: r.props.errorElement,
        ErrorBoundary: r.props.ErrorBoundary,
        hasErrorBoundary:
          r.props.hasErrorBoundary === !0 ||
          r.props.ErrorBoundary != null ||
          r.props.errorElement != null,
        shouldRevalidate: r.props.shouldRevalidate,
        handle: r.props.handle,
        lazy: r.props.lazy,
      };
      r.props.children && (l.children = lu(r.props.children, n)), a.push(l);
    }),
    a
  );
}
var ou = 'get',
  nu = 'application/x-www-form-urlencoded';
function cu(e) {
  return typeof HTMLElement < 'u' && e instanceof HTMLElement;
}
function L_(e) {
  return cu(e) && e.tagName.toLowerCase() === 'button';
}
function C_(e) {
  return cu(e) && e.tagName.toLowerCase() === 'form';
}
function w_(e) {
  return cu(e) && e.tagName.toLowerCase() === 'input';
}
function R_(e) {
  return !!(e.metaKey || e.altKey || e.ctrlKey || e.shiftKey);
}
function __(e, t) {
  return e.button === 0 && (!t || t === '_self') && !R_(e);
}
function iu(e = '') {
  return new URLSearchParams(
    typeof e == 'string' || Array.isArray(e) || e instanceof URLSearchParams
      ? e
      : Object.keys(e).reduce((t, a) => {
          let r = e[a];
          return t.concat(Array.isArray(r) ? r.map((o) => [a, o]) : [[a, r]]);
        }, [])
  );
}
function I_(e, t) {
  let a = iu(e);
  return (
    t &&
      t.forEach((r, o) => {
        a.has(o) ||
          t.getAll(o).forEach((n) => {
            a.append(o, n);
          });
      }),
    a
  );
}
var au = null;
function k_() {
  if (au === null)
    try {
      new FormData(document.createElement('form'), 0), (au = !1);
    } catch {
      au = !0;
    }
  return au;
}
var M_ = new Set(['application/x-www-form-urlencoded', 'multipart/form-data', 'text/plain']);
function Qf(e) {
  return e != null && !M_.has(e)
    ? (Ct(
        !1,
        `"${e}" is not a valid \`encType\` for \`<Form>\`/\`<fetcher.Form>\` and will default to "${nu}"`
      ),
      null)
    : e;
}
function E_(e, t) {
  let a, r, o, n, l;
  if (C_(e)) {
    let i = e.getAttribute('action');
    (r = i ? La(i, t) : null),
      (a = e.getAttribute('method') || ou),
      (o = Qf(e.getAttribute('enctype')) || nu),
      (n = new FormData(e));
  } else if (L_(e) || (w_(e) && (e.type === 'submit' || e.type === 'image'))) {
    let i = e.form;
    if (i == null)
      throw new Error('Cannot submit a <button> or <input type="submit"> without a <form>');
    let s = e.getAttribute('formaction') || i.getAttribute('action');
    if (
      ((r = s ? La(s, t) : null),
      (a = e.getAttribute('formmethod') || i.getAttribute('method') || ou),
      (o = Qf(e.getAttribute('formenctype')) || Qf(i.getAttribute('enctype')) || nu),
      (n = new FormData(i, e)),
      !k_())
    ) {
      let { name: u, type: d, value: c } = e;
      if (d === 'image') {
        let f = u ? `${u}.` : '';
        n.append(`${f}x`, '0'), n.append(`${f}y`, '0');
      } else u && n.append(u, c);
    }
  } else {
    if (cu(e))
      throw new Error(
        'Cannot submit element that is not <form>, <button>, or <input type="submit|image">'
      );
    (a = ou), (r = null), (o = nu), (l = e);
  }
  return (
    n && o === 'text/plain' && ((l = n), (n = void 0)),
    { action: r, method: a.toLowerCase(), encType: o, formData: n, body: l }
  );
}
var LA = Object.getOwnPropertyNames(Object.prototype).sort().join('\0');
var A_ = {
    '&': '\\u0026',
    '>': '\\u003e',
    '<': '\\u003c',
    '\u2028': '\\u2028',
    '\u2029': '\\u2029',
  },
  T_ = /[&><\u2028\u2029]/g;
function nb(e) {
  return e.replace(T_, (t) => A_[t]);
}
function mm(e, t) {
  if (e === !1 || e === null || typeof e > 'u') throw new Error(t);
}
var D_ = Symbol('SingleFetchRedirect');
function Ob(e, t, a, r) {
  let o =
    typeof e == 'string'
      ? new URL(e, typeof window > 'u' ? 'server://singlefetch/' : window.location.origin)
      : e;
  return (
    a
      ? o.pathname.endsWith('/')
        ? (o.pathname = `${o.pathname}_.${r}`)
        : (o.pathname = `${o.pathname}.${r}`)
      : o.pathname === '/'
        ? (o.pathname = `_root.${r}`)
        : t && La(o.pathname, t) === '/'
          ? (o.pathname = `${Rn(t)}/_root.${r}`)
          : (o.pathname = `${Rn(o.pathname)}.${r}`),
    o
  );
}
async function O_(e, t) {
  if (e.id in t) return t[e.id];
  try {
    let a = await import(e.module);
    return (t[e.id] = a), a;
  } catch (a) {
    if (
      (console.error(`Error loading route module \`${e.module}\`, reloading page...`),
      console.error(a),
      window.__reactRouterContext && window.__reactRouterContext.isSpaMode && import.meta.hot)
    )
      throw a;
    return window.location.reload(), new Promise(() => {});
  }
}
function P_(e) {
  return e != null && typeof e.page == 'string';
}
function B_(e) {
  return e == null
    ? !1
    : e.href == null
      ? e.rel === 'preload' && typeof e.imageSrcSet == 'string' && typeof e.imageSizes == 'string'
      : typeof e.rel == 'string' && typeof e.href == 'string';
}
async function U_(e, t, a) {
  let r = await Promise.all(
    e.map(async (o) => {
      let n = t.routes[o.route.id];
      if (n) {
        let l = await O_(n, a);
        return l.links ? l.links() : [];
      }
      return [];
    })
  );
  return F_(
    r
      .flat(1)
      .filter(B_)
      .filter((o) => o.rel === 'stylesheet' || o.rel === 'preload')
      .map((o) =>
        o.rel === 'stylesheet' ? { ...o, rel: 'prefetch', as: 'style' } : { ...o, rel: 'prefetch' }
      )
  );
}
function lb(e, t, a, r, o, n) {
  let l = (s, u) => (a[u] ? s.route.id !== a[u].route.id : !0),
    i = (s, u) =>
      a[u].pathname !== s.pathname ||
      (a[u].route.path?.endsWith('*') && a[u].params['*'] !== s.params['*']);
  return n === 'assets'
    ? t.filter((s, u) => l(s, u) || i(s, u))
    : n === 'data'
      ? t.filter((s, u) => {
          let d = r.routes[s.route.id];
          if (!d || !d.hasLoader) return !1;
          if (l(s, u) || i(s, u)) return !0;
          if (s.route.shouldRevalidate) {
            let c = s.route.shouldRevalidate({
              currentUrl: new URL(o.pathname + o.search + o.hash, window.origin),
              currentParams: a[0]?.params || {},
              nextUrl: new URL(e, window.origin),
              nextParams: s.params,
              defaultShouldRevalidate: !0,
            });
            if (typeof c == 'boolean') return c;
          }
          return !0;
        })
      : [];
}
function N_(e, t, { includeHydrateFallback: a } = {}) {
  return H_(
    e
      .map((r) => {
        let o = t.routes[r.route.id];
        if (!o) return [];
        let n = [o.module];
        return (
          o.clientActionModule && (n = n.concat(o.clientActionModule)),
          o.clientLoaderModule && (n = n.concat(o.clientLoaderModule)),
          a && o.hydrateFallbackModule && (n = n.concat(o.hydrateFallbackModule)),
          o.imports && (n = n.concat(o.imports)),
          n
        );
      })
      .flat(1)
  );
}
function H_(e) {
  return [...new Set(e)];
}
function z_(e) {
  let t = {},
    a = Object.keys(e).sort();
  for (let r of a) t[r] = e[r];
  return t;
}
function F_(e, t) {
  let a = new Set(),
    r = new Set(t);
  return e.reduce((o, n) => {
    if (t && !P_(n) && n.as === 'script' && n.href && r.has(n.href)) return o;
    let i = JSON.stringify(z_(n));
    return a.has(i) || (a.add(i), o.push({ key: i, link: n })), o;
  }, []);
}
function pm() {
  let e = Z.useContext(fo);
  return mm(e, 'You must render this element inside a <DataRouterContext.Provider> element'), e;
}
function j_() {
  let e = Z.useContext(_n);
  return (
    mm(e, 'You must render this element inside a <DataRouterStateContext.Provider> element'), e
  );
}
var li = Z.createContext(void 0);
li.displayName = 'FrameworkContext';
function du() {
  let e = Z.useContext(li);
  return mm(e, 'You must render this element inside a <HydratedRouter> element'), e;
}
function Y_(e, t) {
  let a = Z.useContext(li),
    [r, o] = Z.useState(!1),
    [n, l] = Z.useState(!1),
    { onFocus: i, onBlur: s, onMouseEnter: u, onMouseLeave: d, onTouchStart: c } = t,
    f = Z.useRef(null);
  Z.useEffect(() => {
    if ((e === 'render' && l(!0), e === 'viewport')) {
      let x = (m) => {
          m.forEach((p) => {
            l(p.isIntersecting);
          });
        },
        y = new IntersectionObserver(x, { threshold: 0.5 });
      return (
        f.current && y.observe(f.current),
        () => {
          y.disconnect();
        }
      );
    }
  }, [e]),
    Z.useEffect(() => {
      if (r) {
        let x = setTimeout(() => {
          l(!0);
        }, 100);
        return () => {
          clearTimeout(x);
        };
      }
    }, [r]);
  let h = () => {
      o(!0);
    },
    v = () => {
      o(!1), l(!1);
    };
  return a
    ? e !== 'intent'
      ? [n, f, {}]
      : [
          n,
          f,
          {
            onFocus: oi(i, h),
            onBlur: oi(s, v),
            onMouseEnter: oi(u, h),
            onMouseLeave: oi(d, v),
            onTouchStart: oi(c, h),
          },
        ]
    : [!1, f, {}];
}
function oi(e, t) {
  return (a) => {
    e && e(a), a.defaultPrevented || t(a);
  };
}
function Bb({ page: e, ...t }) {
  let a = e_(),
    { nonce: r } = du(),
    { router: o } = pm(),
    n = Z.useMemo(() => tm(o.routes, e, o.basename), [o.routes, e, o.basename]);
  return n
    ? (t.nonce == null && r && (t = { ...t, nonce: r }),
      a
        ? Z.createElement(W_, { page: e, matches: n, ...t })
        : Z.createElement(Q_, { page: e, matches: n, ...t }))
    : null;
}
function X_(e) {
  let { manifest: t, routeModules: a } = du(),
    [r, o] = Z.useState([]);
  return (
    Z.useEffect(() => {
      let n = !1;
      return (
        U_(e, t, a).then((l) => {
          n || o(l);
        }),
        () => {
          n = !0;
        }
      );
    }, [e, t, a]),
    r
  );
}
function W_({ page: e, matches: t, ...a }) {
  let r = Rt(),
    { future: o } = du(),
    { basename: n } = pm(),
    l = Z.useMemo(() => {
      if (e === r.pathname + r.search + r.hash) return [];
      let i = Ob(e, n, o.v8_trailingSlashAwareDataRequests, 'rsc'),
        s = !1,
        u = [];
      for (let d of t)
        typeof d.route.shouldRevalidate == 'function' ? (s = !0) : u.push(d.route.id);
      return (
        s && u.length > 0 && i.searchParams.set('_routes', u.join(',')), [i.pathname + i.search]
      );
    }, [n, o.v8_trailingSlashAwareDataRequests, e, r, t]);
  return Z.createElement(
    Z.Fragment,
    null,
    l.map((i) => Z.createElement('link', { key: i, rel: 'prefetch', as: 'fetch', href: i, ...a }))
  );
}
function Q_({ page: e, matches: t, ...a }) {
  let r = Rt(),
    { future: o, manifest: n, routeModules: l } = du(),
    { basename: i } = pm(),
    { loaderData: s, matches: u } = j_(),
    d = Z.useMemo(() => lb(e, t, u, n, r, 'data'), [e, t, u, n, r]),
    c = Z.useMemo(() => lb(e, t, u, n, r, 'assets'), [e, t, u, n, r]),
    f = Z.useMemo(() => {
      if (e === r.pathname + r.search + r.hash) return [];
      let x = new Set(),
        y = !1;
      if (
        (t.forEach((p) => {
          let g = n.routes[p.route.id];
          !g ||
            !g.hasLoader ||
            ((!d.some((b) => b.route.id === p.route.id) &&
              p.route.id in s &&
              l[p.route.id]?.shouldRevalidate) ||
            g.hasClientLoader
              ? (y = !0)
              : x.add(p.route.id));
        }),
        x.size === 0)
      )
        return [];
      let m = Ob(e, i, o.v8_trailingSlashAwareDataRequests, 'data');
      return (
        y &&
          x.size > 0 &&
          m.searchParams.set(
            '_routes',
            t
              .filter((p) => x.has(p.route.id))
              .map((p) => p.route.id)
              .join(',')
          ),
        [m.pathname + m.search]
      );
    }, [i, o.v8_trailingSlashAwareDataRequests, s, r, n, d, t, e, l]),
    h = Z.useMemo(() => N_(c, n), [c, n]),
    v = X_(c);
  return Z.createElement(
    Z.Fragment,
    null,
    f.map((x) => Z.createElement('link', { key: x, rel: 'prefetch', as: 'fetch', href: x, ...a })),
    h.map((x) => Z.createElement('link', { key: x, rel: 'modulepreload', href: x, ...a })),
    v.map(({ key: x, link: y }) =>
      Z.createElement('link', {
        key: x,
        nonce: a.nonce,
        ...y,
        crossOrigin: y.crossOrigin ?? a.crossOrigin,
      })
    )
  );
}
function K_(...e) {
  return (t) => {
    e.forEach((a) => {
      typeof a == 'function' ? a(t) : a != null && (a.current = t);
    });
  };
}
var Z_ =
  typeof window < 'u' && typeof window.document < 'u' && typeof window.document.createElement < 'u';
try {
  Z_ && (window.__reactRouterVersion = '7.18.3');
} catch {}
function J_({ basename: e, children: t, useTransitions: a, window: r }) {
  let o = P.useRef();
  o.current == null && (o.current = sb({ window: r, v5Compat: !0 }));
  let n = o.current,
    [l, i] = P.useState({ action: n.action, location: n.location }),
    s = P.useCallback(
      (u) => {
        a === !1 ? i(u) : P.startTransition(() => i(u));
      },
      [a]
    );
  return (
    P.useLayoutEffect(() => n.listen(s), [n, s]),
    P.createElement(dm, {
      basename: e,
      children: t,
      location: l.location,
      navigationType: l.action,
      navigator: n,
      useTransitions: a,
    })
  );
}
function Ub({ basename: e, children: t, history: a, useTransitions: r }) {
  let [o, n] = P.useState({ action: a.action, location: a.location }),
    l = P.useCallback(
      (i) => {
        r === !1 ? n(i) : P.startTransition(() => n(i));
      },
      [r]
    );
  return (
    P.useLayoutEffect(() => a.listen(l), [a, l]),
    P.createElement(dm, {
      basename: e,
      children: t,
      location: o.location,
      navigationType: o.action,
      navigator: a,
      useTransitions: r,
    })
  );
}
Ub.displayName = 'unstable_HistoryRouter';
var Ca = P.forwardRef(function (
  {
    onClick: t,
    discover: a = 'render',
    prefetch: r = 'none',
    relative: o,
    reloadDocument: n,
    replace: l,
    mask: i,
    state: s,
    target: u,
    to: d,
    preventScrollReset: c,
    viewTransition: f,
    defaultShouldRevalidate: h,
    ...v
  },
  x
) {
  let { basename: y, navigator: m, useTransitions: p } = P.useContext(st),
    g = typeof d == 'string' && em.test(d),
    b = vb(d, y);
  d = b.to;
  let R = Cb(d, { relative: o }),
    I = Rt(),
    w = null;
  if (i) {
    let Ee = su(i, [], I.mask ? I.mask.pathname : '/', !0);
    y !== '/' && (Ee.pathname = Ee.pathname === '/' ? y : sa([y, Ee.pathname])),
      (w = m.createHref(Ee));
  }
  let [L, M, z] = Y_(r, v),
    qe = qb(d, {
      replace: l,
      mask: i,
      state: s,
      target: u,
      preventScrollReset: c,
      relative: o,
      viewTransition: f,
      defaultShouldRevalidate: h,
      useTransitions: p,
    });
  function pt(Ee) {
    t && t(Ee), Ee.defaultPrevented || qe(Ee);
  }
  let aa = !(b.isExternal || n),
    nt = P.createElement('a', {
      ...v,
      ...z,
      href: (aa ? w : void 0) || b.absoluteURL || R,
      onClick: aa ? pt : t,
      ref: K_(x, M),
      target: u,
      'data-discover': !g && a === 'render' ? 'true' : void 0,
    });
  return L && !g ? P.createElement(P.Fragment, null, nt, P.createElement(Bb, { page: R })) : nt;
});
Ca.displayName = 'Link';
var Nb = P.forwardRef(function (
  {
    'aria-current': t = 'page',
    caseSensitive: a = !1,
    className: r = '',
    end: o = !1,
    style: n,
    to: l,
    viewTransition: i,
    children: s,
    ...u
  },
  d
) {
  let c = kn(l, { relative: u.relative }),
    f = Rt(),
    h = P.useContext(_n),
    { navigator: v, basename: x } = P.useContext(st),
    y = h != null && Yb(c) && i === !0,
    m = v.encodeLocation ? v.encodeLocation(c).pathname : c.pathname,
    p = f.pathname,
    g = h && h.navigation && h.navigation.location ? h.navigation.location.pathname : null;
  a || ((p = p.toLowerCase()), (g = g ? g.toLowerCase() : null), (m = m.toLowerCase())),
    g && x && (g = La(g, x) || g);
  let b = m !== '/' && m.endsWith('/') ? m.length - 1 : m.length,
    R = p === m || (!o && p.startsWith(m) && p.charAt(b) === '/'),
    I = g != null && (g === m || (!o && g.startsWith(m) && g.charAt(m.length) === '/')),
    w = { isActive: R, isPending: I, isTransitioning: y },
    L = R ? t : void 0,
    M;
  typeof r == 'function'
    ? (M = r(w))
    : (M = [r, R ? 'active' : null, I ? 'pending' : null, y ? 'transitioning' : null]
        .filter(Boolean)
        .join(' '));
  let z = typeof n == 'function' ? n(w) : n;
  return P.createElement(
    Ca,
    { ...u, 'aria-current': L, className: M, ref: d, style: z, to: l, viewTransition: i },
    typeof s == 'function' ? s(w) : s
  );
});
Nb.displayName = 'NavLink';
var Hb = P.forwardRef(
  (
    {
      discover: e = 'render',
      fetcherKey: t,
      navigate: a,
      reloadDocument: r,
      replace: o,
      state: n,
      method: l = ou,
      action: i,
      onSubmit: s,
      relative: u,
      preventScrollReset: d,
      viewTransition: c,
      defaultShouldRevalidate: f,
      ...h
    },
    v
  ) => {
    let { useTransitions: x } = P.useContext(st),
      y = Gb(),
      m = Vb(i, { relative: u }),
      p = l.toLowerCase() === 'get' ? 'get' : 'post',
      g = typeof i == 'string' && em.test(i);
    return P.createElement('form', {
      ref: v,
      method: p,
      action: m,
      onSubmit: r
        ? s
        : (R) => {
            if ((s && s(R), R.defaultPrevented)) return;
            R.preventDefault();
            let I = R.nativeEvent.submitter,
              w = I?.getAttribute('formmethod') || l,
              L = () =>
                y(I || R.currentTarget, {
                  fetcherKey: t,
                  method: w,
                  navigate: a,
                  replace: o,
                  state: n,
                  relative: u,
                  preventScrollReset: d,
                  viewTransition: c,
                  defaultShouldRevalidate: f,
                });
            x && a !== !1 ? P.startTransition(() => L()) : L();
          },
      ...h,
      'data-discover': !g && e === 'render' ? 'true' : void 0,
    });
  }
);
Hb.displayName = 'Form';
function zb({ getKey: e, storageKey: t, ...a }) {
  let r = P.useContext(li),
    { basename: o } = P.useContext(st),
    n = Rt(),
    l = cm();
  jb({ getKey: e, storageKey: t });
  let i = P.useMemo(() => {
    if (!r || !e) return null;
    let u = $f(n, l, o, e);
    return u !== n.key ? u : null;
  }, []);
  if (!r || r.isSpaMode) return null;
  let s = ((u, d) => {
    if (!window.history.state || !window.history.state.key) {
      let c = Math.random().toString(32).slice(2);
      window.history.replaceState({ key: c }, '');
    }
    try {
      let f = JSON.parse(sessionStorage.getItem(u) || '{}')[d || window.history.state.key];
      typeof f == 'number' && window.scrollTo(0, f);
    } catch (c) {
      console.error(c), sessionStorage.removeItem(u);
    }
  }).toString();
  return (
    a.nonce == null && r?.nonce && (a.nonce = r.nonce),
    P.createElement('script', {
      ...a,
      suppressHydrationWarning: !0,
      dangerouslySetInnerHTML: {
        __html: `(${s})(${nb(JSON.stringify(t || Jf))}, ${nb(JSON.stringify(i))})`,
      },
    })
  );
}
zb.displayName = 'ScrollRestoration';
function Fb(e) {
  return `${e} must be used within a data router.  See https://reactrouter.com/en/main/routers/picking-a-router.`;
}
function hm(e) {
  let t = P.useContext(fo);
  return ve(t, Fb(e)), t;
}
function $_(e) {
  let t = P.useContext(_n);
  return ve(t, Fb(e)), t;
}
function qb(
  e,
  {
    target: t,
    replace: a,
    mask: r,
    state: o,
    preventScrollReset: n,
    relative: l,
    viewTransition: i,
    defaultShouldRevalidate: s,
    useTransitions: u,
  } = {}
) {
  let d = uu(),
    c = Rt(),
    f = kn(e, { relative: l });
  return P.useCallback(
    (h) => {
      if (__(h, t)) {
        h.preventDefault();
        let v = a !== void 0 ? a : Br(c) === Br(f),
          x = () =>
            d(e, {
              replace: v,
              mask: r,
              state: o,
              preventScrollReset: n,
              relative: l,
              viewTransition: i,
              defaultShouldRevalidate: s,
            });
        u ? P.startTransition(() => x()) : x();
      }
    },
    [c, d, f, a, r, o, t, e, n, l, i, s, u]
  );
}
function eI(e) {
  Ct(
    typeof URLSearchParams < 'u',
    'You cannot use the `useSearchParams` hook in a browser that does not support the URLSearchParams API. If you need to support Internet Explorer 11, we recommend you load a polyfill such as https://github.com/ungap/url-search-params.'
  );
  let t = P.useRef(iu(e)),
    a = P.useRef(!1),
    r = Rt(),
    o = P.useMemo(() => I_(r.search, a.current ? null : t.current), [r.search]),
    n = uu(),
    l = P.useCallback(
      (i, s) => {
        let u = iu(typeof i == 'function' ? i(new URLSearchParams(o)) : i);
        (a.current = !0), n('?' + u, s);
      },
      [n, o]
    );
  return [o, l];
}
var tI = 0,
  aI = () => `__${String(++tI)}__`;
function Gb() {
  let { router: e } = hm('useSubmit'),
    { basename: t } = P.useContext(st),
    a = g_(),
    r = e.fetch,
    o = e.navigate;
  return P.useCallback(
    async (n, l = {}) => {
      let { action: i, method: s, encType: u, formData: d, body: c } = E_(n, t);
      if (l.navigate === !1) {
        let f = l.fetcherKey || aI();
        await r(f, a, l.action || i, {
          defaultShouldRevalidate: l.defaultShouldRevalidate,
          preventScrollReset: l.preventScrollReset,
          formData: d,
          body: c,
          formMethod: l.method || s,
          formEncType: l.encType || u,
          flushSync: l.flushSync,
        });
      } else
        await o(l.action || i, {
          defaultShouldRevalidate: l.defaultShouldRevalidate,
          preventScrollReset: l.preventScrollReset,
          formData: d,
          body: c,
          formMethod: l.method || s,
          formEncType: l.encType || u,
          replace: l.replace,
          state: l.state,
          fromRouteId: a,
          flushSync: l.flushSync,
          viewTransition: l.viewTransition,
        });
    },
    [r, o, t, a]
  );
}
function Vb(e, { relative: t } = {}) {
  let { basename: a } = P.useContext(st),
    r = P.useContext(Nt);
  ve(r, 'useFormAction must be used inside a RouteContext');
  let [o] = r.matches.slice(-1),
    n = { ...kn(e || '.', { relative: t }) },
    l = Rt();
  if (e == null) {
    n.search = l.search;
    let i = new URLSearchParams(n.search),
      s = i.getAll('index');
    if (s.some((d) => d === '')) {
      i.delete('index'), s.filter((c) => c).forEach((c) => i.append('index', c));
      let d = i.toString();
      n.search = d ? `?${d}` : '';
    }
  }
  return (
    (!e || e === '.') &&
      o.route.index &&
      (n.search = n.search ? n.search.replace(/^\?/, '?index&') : '?index'),
    a !== '/' && (n.pathname = n.pathname === '/' ? a : sa([a, n.pathname])),
    Br(n)
  );
}
var Jf = 'react-router-scroll-positions',
  ru = {};
function $f(e, t, a, r) {
  let o = null;
  return (
    r &&
      (a !== '/' ? (o = r({ ...e, pathname: La(e.pathname, a) || e.pathname }, t)) : (o = r(e, t))),
    o == null && (o = e.key),
    o
  );
}
function jb({ getKey: e, storageKey: t } = {}) {
  let { router: a } = hm('useScrollRestoration'),
    { restoreScrollPosition: r, preventScrollReset: o } = $_('useScrollRestoration'),
    { basename: n } = P.useContext(st),
    l = Rt(),
    i = cm(),
    s = Eb();
  P.useEffect(
    () => (
      (window.history.scrollRestoration = 'manual'),
      () => {
        window.history.scrollRestoration = 'auto';
      }
    ),
    []
  ),
    rI(
      P.useCallback(() => {
        if (s.state === 'idle') {
          let u = $f(l, i, n, e);
          ru[u] = window.scrollY;
        }
        try {
          sessionStorage.setItem(t || Jf, JSON.stringify(ru));
        } catch (u) {
          Ct(
            !1,
            `Failed to save scroll positions in sessionStorage, <ScrollRestoration /> will not work properly (${u}).`
          );
        }
        window.history.scrollRestoration = 'auto';
      }, [s.state, e, n, l, i, t])
    ),
    typeof document < 'u' &&
      (P.useLayoutEffect(() => {
        try {
          let u = sessionStorage.getItem(t || Jf);
          u && (ru = JSON.parse(u));
        } catch {}
      }, [t]),
      P.useLayoutEffect(() => {
        let u = a?.enableScrollRestoration(
          ru,
          () => window.scrollY,
          e ? (d, c) => $f(d, c, n, e) : void 0
        );
        return () => u && u();
      }, [a, n, e]),
      P.useLayoutEffect(() => {
        if (r !== !1) {
          if (typeof r == 'number') {
            window.scrollTo(0, r);
            return;
          }
          try {
            if (l.hash) {
              let u = document.getElementById(decodeURIComponent(l.hash.slice(1)));
              if (u) {
                u.scrollIntoView();
                return;
              }
            }
          } catch {
            Ct(
              !1,
              `"${l.hash.slice(1)}" is not a decodable element ID. The view will not scroll to it.`
            );
          }
          o !== !0 && window.scrollTo(0, 0);
        }
      }, [l, r, o]));
}
function rI(e, t) {
  let { capture: a } = t || {};
  P.useEffect(() => {
    let r = a != null ? { capture: a } : void 0;
    return (
      window.addEventListener('pagehide', e, r),
      () => {
        window.removeEventListener('pagehide', e, r);
      }
    );
  }, [e, a]);
}
function Yb(e, { relative: t } = {}) {
  let a = P.useContext(nm);
  ve(
    a != null,
    "`useViewTransitionState` must be used within `react-router-dom`'s `RouterProvider`.  Did you accidentally import `RouterProvider` from `react-router`?"
  );
  let { basename: r } = hm('useViewTransitionState'),
    o = kn(e, { relative: t });
  if (!a.isTransitioning) return !1;
  let n = La(a.currentLocation.pathname, r) || a.currentLocation.pathname,
    l = La(a.nextLocation.pathname, r) || a.nextLocation.pathname;
  return ni(o.pathname, l) != null || ni(o.pathname, n) != null;
}
var gm = 'adminDevMode',
  Mn = !1;
function oI() {
  if (typeof window > 'u') return !1;
  try {
    return window.localStorage.getItem(gm) === '1';
  } catch {
    return !1;
  }
}
function fu(e) {
  if (!(typeof window > 'u'))
    try {
      e ? window.localStorage.setItem(gm, '1') : window.localStorage.removeItem(gm);
    } catch {}
}
function UT() {
  if (typeof window > 'u') return;
  let e = new URLSearchParams(window.location.search),
    t = e.get('admin_dev');
  if (t === '1') {
    (Mn = !0), fu(!0), e.delete('admin_dev');
    let a = e.toString(),
      r = `${window.location.pathname}${a ? `?${a}` : ''}${window.location.hash}`;
    window.history.replaceState(null, '', r);
    return;
  }
  if (t === '0') {
    (Mn = !1), fu(!1), e.delete('admin_dev');
    let a = e.toString(),
      r = `${window.location.pathname}${a ? `?${a}` : ''}${window.location.hash}`;
    window.history.replaceState(null, '', r);
    return;
  }
  Mn = oI();
}
function En() {
  return Mn;
}
function Wb() {
  (Mn = !0), fu(!0);
}
function NT() {
  (Mn = !1), fu(!1);
}
var ua = E(te());
var ii = 'aed-admin-theme',
  Jb = 'aed-admin-theme-light-default-v2';
function pu() {
  try {
    return iI(), window.localStorage.getItem(ii) === 'dark' ? 'dark' : 'light';
  } catch {
    return 'light';
  }
}
function iI() {
  try {
    if (window.localStorage.getItem(Jb) === '1') return;
    window.localStorage.setItem(ii, 'light'), window.localStorage.setItem(Jb, '1');
  } catch {}
}
function $b() {
  let e = document.documentElement;
  return e.classList.contains('light') ? 'light' : e.classList.contains('dark') ? 'dark' : pu();
}
function hu(e) {
  let t = document.documentElement,
    a = e === 'dark' ? 'light' : 'dark';
  if (t.classList.contains(e) && !t.classList.contains(a)) {
    t.style.colorScheme !== e && (t.style.colorScheme = e);
    return;
  }
  t.classList.remove('light', 'dark'), t.classList.add(e), (t.style.colorScheme = e);
}
function gu(e) {
  try {
    window.localStorage.setItem(ii, e);
  } catch {}
}
function qT(e) {
  return e === 'dark' ? 'Switch to light theme' : 'Switch to dark theme';
}
var ex = E($()),
  sI = (0, ua.createContext)(null);
function jT({ children: e }) {
  let [t, a] = (0, ua.useState)(() => $b()),
    r = (0, ua.useCallback)((n) => {
      a((l) => (l === n ? l : (hu(n), gu(n), n)));
    }, []);
  (0, ua.useEffect)(() => {
    let n = pu();
    hu(n), gu(n), a((l) => (l === n ? l : n));
  }, []),
    (0, ua.useEffect)(() => {
      let n = (l) => {
        if (l.key !== ii) return;
        let i = pu();
        a((s) => (s === i ? s : (hu(i), gu(i), i)));
      };
      return window.addEventListener('storage', n), () => window.removeEventListener('storage', n);
    }, []);
  let o = (0, ua.useMemo)(() => ({ theme: t, setTheme: r }), [t, r]);
  return (0, ex.jsx)(sI.Provider, { value: o, children: e });
}
var Gu = E(te());
var $a = E(te());
function O(e = 0, t = 0) {
  let a = Date.now() - e * 864e5 - t * 36e5;
  return new Date(a).toISOString();
}
function tx(e = 0) {
  let t = new Date();
  t.setUTCDate(1), t.setUTCMonth(t.getUTCMonth() + e);
  let a = t.getUTCFullYear(),
    r = String(t.getUTCMonth() + 1).padStart(2, '0');
  return `${a}-${r}`;
}
function ae(e) {
  return Math.round(e * 1e6);
}
function ho(e, t = 'USD') {
  let a = e / 1e6;
  return `${t} ${a.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}
function ym(e) {
  return e.toLocaleString('en-US');
}
function go(e, t, a) {
  let r = Math.max(1, t),
    o = Math.max(0, a);
  return { items: e.slice(o, o + r), total: e.length };
}
function yo(e, t = 50) {
  let a = Number.parseInt(e.searchParams.get('limit') ?? String(t), 10) || t,
    r = Number.parseInt(e.searchParams.get('offset') ?? '0', 10) || 0;
  return { limit: a, offset: r };
}
function vm(e, t) {
  return (e << t) | (e >>> (32 - t));
}
function yu(e) {
  let t = [];
  for (let d = 0; d < e.length; d += 1) t[d >> 2] |= e[d] << (24 - (d % 4) * 8);
  t[e.length >> 2] |= 128 << (24 - (e.length % 4) * 8);
  let a = (((e.length + 8) >> 6) + 1) * 16;
  for (; t.length < a; ) t.push(0);
  t[a - 1] = e.length * 8;
  let r = 1732584193,
    o = 4023233417,
    n = 2562383102,
    l = 271733878,
    i = 3285377520;
  for (let d = 0; d < t.length; d += 16) {
    let c = new Array(80);
    for (let m = 0; m < 16; m += 1) c[m] = t[d + m] | 0;
    for (let m = 16; m < 80; m += 1) c[m] = vm(c[m - 3] ^ c[m - 8] ^ c[m - 14] ^ c[m - 16], 1);
    let f = r,
      h = o,
      v = n,
      x = l,
      y = i;
    for (let m = 0; m < 80; m += 1) {
      let p, g;
      m < 20
        ? ((p = (h & v) | (~h & x)), (g = 1518500249))
        : m < 40
          ? ((p = h ^ v ^ x), (g = 1859775393))
          : m < 60
            ? ((p = (h & v) | (h & x) | (v & x)), (g = 2400959708))
            : ((p = h ^ v ^ x), (g = 3395469782));
      let b = (vm(f, 5) + p + y + g + c[m]) | 0;
      (y = x), (x = v), (v = vm(h, 30)), (h = f), (f = b);
    }
    (r = (r + f) | 0), (o = (o + h) | 0), (n = (n + v) | 0), (l = (l + x) | 0), (i = (i + y) | 0);
  }
  let s = new Uint8Array(20),
    u = [r, o, n, l, i];
  for (let d = 0; d < u.length; d += 1)
    (s[d * 4] = (u[d] >>> 24) & 255),
      (s[d * 4 + 1] = (u[d] >>> 16) & 255),
      (s[d * 4 + 2] = (u[d] >>> 8) & 255),
      (s[d * 4 + 3] = u[d] & 255);
  return s;
}
var uI = 'f9ceb2a2-97a8-5602-afe5-076194dce015',
  cI = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
function dI(e) {
  let t = e.replace(/-/g, ''),
    a = new Uint8Array(16);
  for (let r = 0; r < 16; r += 1) a[r] = Number.parseInt(t.slice(r * 2, r * 2 + 2), 16);
  return a;
}
function ax(e) {
  let t = Array.from(e, (a) => a.toString(16).padStart(2, '0')).join('');
  return `${t.slice(0, 8)}-${t.slice(8, 12)}-${t.slice(12, 16)}-${t.slice(16, 20)}-${t.slice(20)}`;
}
function fI(e, t) {
  let a = dI(t),
    r = new TextEncoder().encode(e),
    o = new Uint8Array(a.length + r.length);
  o.set(a, 0), o.set(r, a.length);
  let n = yu(o);
  return (n[6] = (n[6] & 15) | 80), (n[8] = (n[8] & 63) | 128), ax(n.subarray(0, 16));
}
function U(e, t) {
  return fI(`${e}:${t}`, uI);
}
function vu() {
  if (typeof crypto < 'u' && typeof crypto.randomUUID == 'function') return crypto.randomUUID();
  let e = new Uint8Array(16);
  if (typeof crypto < 'u' && typeof crypto.getRandomValues == 'function') crypto.getRandomValues(e);
  else for (let t = 0; t < e.length; t += 1) e[t] = Math.floor(Math.random() * 256);
  return (e[6] = (e[6] & 15) | 64), (e[8] = (e[8] & 63) | 128), ax(e);
}
function mI(e) {
  return e?.trim() ? cI.test(e.trim()) : !1;
}
var ox = [
    'Velox Checkout',
    'Northstar Finance',
    'Pulse Health',
    'Orbit Travel',
    'Nova SaaS',
    'Harbor Insurance',
    'Kite Mobility',
    'Summit Ecom',
    'Lumen EdTech',
    'Crestline VPN',
  ],
  nx = [
    'Meta social split',
    'Push subscriber route',
    'Native article path',
    'Search brand lane',
    'Popunder direct',
    'Display prospecting',
    'Remarketing checkout',
    'Affiliate coupon mix',
  ],
  si = [
    'lander/checkout-v2',
    'lander/hero-video',
    'lander/compare-table',
    'lander/quiz-funnel',
    'lander/app-install',
    'lander/finance-offer-v3',
    'lander/sweeps-mobile',
    'lander/vsl-17min',
  ],
  bu = [
    'CreditLine Pro (CPL)',
    'Casino Welcome (CPA)',
    'Travel Booking (CPS)',
    'Finance App (CPI)',
    'Insurance Quote (CPL)',
    'Sweeps Entry (CPL)',
    'Solar panel CPL west',
    'VPN annual plans',
  ],
  bm = ['pub_northstar', 'pub_velocity', 'pub_summit', 'pub_harbor', 'pub_cedar', 'pub_orbit'],
  lx = [
    'Northstar Publishing',
    'Velocity Media Network',
    'Summit Content Group',
    'Harborfront Digital',
    'Cedar Lane Media',
    'Orbit Audience Co',
  ],
  xu = ['facebook', 'google', 'tiktok', 'snapchat', 'native-ads', 'taboola', 'email', 'affiliate'],
  ix = [
    'Hero carousel',
    'Video pre-roll',
    'Static banner',
    'Native card',
    'Interstitial',
    'Playable unit',
  ],
  An = [
    'track.horizon-media.io',
    'track.pacific-ads.studio',
    'track.nordic-performance.co',
    'track.atlas-buying.com',
    'track.summit-traffiq.net',
    'track.bluewave-partners.io',
  ],
  rx = [
    { name: 'Horizon deals assistant', username: 'horizon_deals_bot' },
    { name: 'Pacific offers desk', username: 'pacific_offers_bot' },
    { name: 'Summit promo alerts', username: 'summit_promo_bot' },
    { name: 'Atlas checkout helper', username: 'atlas_checkout_bot' },
  ],
  sx = [
    'Pause on margin breach',
    'Throttle high-IVT placement',
    'Resume after budget refill',
    'Blacklist proxy subnet',
    'Notify on pacing drift',
  ],
  ux = [
    'Spend velocity spike',
    'Conversion rate drop',
    'Budget cap proximity',
    'Postback DLQ backlog',
  ],
  cx = [
    'Campaign overview - last 7 days',
    'GEO ROI - EU focus',
    'Fraud breakdown - weekly',
    'Click log - checkout funnel',
    'RTB no-bid audit',
  ],
  dx = [
    'Portfolio floor 12% ROI',
    'Finance vertical guard',
    'Weekend throttle policy',
    'Tier-1 GEO protection',
  ],
  fx = [
    'Weighted path optimizer',
    'Bid floor lift on winners',
    'Creative rotation balancer',
    'Source quality rebalancer',
    'Weekend traffic shift',
  ];
function Su(e) {
  let t = [
      'horizon-media.io',
      'pacific-ads.studio',
      'nordic-performance.co',
      'atlas-buying.com',
      'summit-traffiq.net',
    ],
    a = [
      'ops',
      'media.buyer',
      'finance',
      'growth',
      'traffic',
      'analytics',
      'partnerships',
      'dev',
      'campaigns',
      'billing',
    ],
    r = t[e % t.length];
  return `${a[e % a.length]}+${100 + e}@${r}`;
}
function de(e, t) {
  return e[(t - 1) % e.length];
}
function Tn(e) {
  return U('click', e);
}
function Dn(e) {
  return de(si, e);
}
function mx(e) {
  let t = yu(new TextEncoder().encode(`dev-mock-ip:${e}`));
  return Array.from(t, (a) => a.toString(16).padStart(2, '0')).join('');
}
function px(e) {
  let a = U('external_campaign', e).replace(/-/g, '').slice(0, 12);
  return BigInt(`0x${a}`).toString().padStart(11, '0').slice(0, 11);
}
function hx(e) {
  let t = ['PMP', 'PG', 'Preferred'],
    a = ['US', 'EU', 'APAC'],
    r = de(t, e),
    o = de(a, e + 1);
  return `${r}-${o}-display-${String(e).padStart(2, '0')}`;
}
function xm(e) {
  return `${de(bm, e).replace('pub_', '')}-media.example`;
}
function gx(e) {
  return `google.com, ${12e8 + e * 17431}, DIRECT, f08c47fec0942fa0`;
}
function Lu(e) {
  return `${de(si, e).split('/').pop() ?? 'index'}.html`;
}
function yx(e) {
  let t = de(An, e),
    a = de(si, e);
  return `https://${t}/${a}.html`;
}
function Sm(e) {
  return rx[(e - 1) % rx.length];
}
function vx(e) {
  return `https://api.telegram.org/bot/${Sm(e).username}/postback`;
}
function bx(e) {
  return `https://${de(An, e)}/postback/conversion?click_id={click_id}`;
}
function xx(e, t) {
  return U(e, t);
}
function ui() {
  return [];
}
function pI(e) {
  let t = 0;
  for (let a = 0; a < e.length; a += 1) t = (t * 31 + e.charCodeAt(a)) >>> 0;
  return String(1e7 + (t % 9e7)).padStart(8, '0');
}
var hI = [
    'Horizon Media Group',
    'Pacific Ads Studio',
    'Nordic Performance Co',
    'Atlas Buying Desk',
    'Summit Traffiq',
  ],
  Sx = [
    'Summer checkout retarget',
    'Velox trial onboarding',
    'Horizon brand lift Q3',
    'Sportsbook install tier-1',
    'Insurance quote funnel',
    'Solar panel CPL west',
    'Fintech card signup',
    'Ecom cart abandoners',
    'Mobile game level-10',
    'B2B SaaS demo requests',
    'Travel meta search spring',
    'Telco prepaid acquisition',
    'Crypto exchange KYC',
    'Nutra sweepstakes LP',
    'Remittance app re-engagement',
    'UPI wallet funding',
    'Remarketing catalog sales',
    'Native article placements',
    'Push notification winback',
    'Search non-brand conquest',
    'Display prospecting broad',
    'Connected TV awareness',
    'Podcast host read spots',
    'Influencer whitelisting burst',
    'Affiliate coupon codes',
  ];
var Cu = ['US', 'GB', 'DE', 'CA', 'UA', 'FR', 'JP', 'AU', 'BR', 'MX'],
  se = hI.map((e, t) => ({ id: U('customer', t + 1), name: e })),
  Xe = [
    { id: U('user', 1), email: Su(1) },
    { id: U('user', 2), email: Su(2) },
    { id: U('user', 3), email: Su(3) },
  ];
function gI(e) {
  return e % 11 === 0 ? 'ARCHIVED' : e % 4 === 0 ? 'PAUSED' : 'ACTIVE';
}
function yI(e) {
  let t = U('campaign', e),
    a = se[(e - 1) % se.length],
    r = Xe[(e - 1) % Xe.length],
    o = gI(e),
    n = 5e6 + (e % 7) * 125e4,
    l = Math.floor(n * (0.12 + (e % 9) * 0.07)),
    i = new Date().toISOString();
  return {
    ...{
      id: t,
      name: Sx[(e - 1) % Sx.length],
      status: o,
      budget_limit: (n / 1e6).toFixed(6),
      current_spend: (l / 1e6).toFixed(6),
      customer_id: a.id,
      pacing_mode: e % 3 === 0 ? 'ASAP' : 'EVEN',
      daily_budget: (n / 30 / 1e6).toFixed(6),
      timezone: 'UTC',
      freq_limit: 3,
      freq_window: 86400,
      target_countries: [Cu[e % Cu.length], Cu[(e + 3) % Cu.length]],
      daypart_hours: Array.from({ length: 24 }, (u, d) => d),
      owner_user_id: r.id,
      created_at: i,
      updated_at: i,
      margin_breach: e % 13 === 0,
    },
    display_id: pI(t),
  };
}
function ci() {
  return Array.from({ length: 25 }, (e, t) => yI(t + 1));
}
var Lm;
function oe() {
  return Lm || (Lm = { campaigns: ci() }), Lm;
}
function di(e, t) {
  return { status: e, body: t, contentType: 'application/json' };
}
var vI = [
  {
    id: 'postgres',
    status: 'pass',
    message: 'Postgres reachable',
    hint: 'Synthetic preview',
    latency_ms: 3,
  },
  {
    id: 'redis',
    status: 'pass',
    message: 'Redis shards healthy',
    hint: 'Synthetic preview',
    latency_ms: 2,
  },
  {
    id: 'clickhouse',
    status: 'pass',
    message: 'ClickHouse lag within SLA',
    hint: 'Synthetic preview',
    latency_ms: 8,
  },
];
function Cm() {
  let e = de(An, 1);
  return {
    overall: 'ok',
    checks: [...vI],
    tracking_domain: e,
    rtb_mode: 'shadow',
    rtb_enabled: !1,
    click_url_template: `https://${e}/click?cid={campaign_id}`,
  };
}
function wm() {
  return {
    status: 'ok',
    clickhouse_lag_seconds: 1.2,
    outbox_oldest_pending_seconds: 0.4,
    redis_shard_reachable: !0,
    redis_shards_reachable: 4,
    redis_shards_total: 4,
    license_state: 'ACTIVE',
    cost_sync_last_success_seconds: 120,
    automation_worker_last_tick_seconds: 15,
  };
}
function Rm() {
  let e = new Date().toISOString();
  return {
    generated_at: e,
    generated_at_display: e,
    services: [
      { id: 'tracker', name: 'Tracker', status: 'ok', detail: 'Within SLA' },
      { id: 'control', name: 'Control plane', status: 'ok', detail: 'Within SLA' },
      { id: 'processor', name: 'Processor', status: 'ok', detail: 'Within SLA' },
    ],
    rps_estimate: 1240.5,
    outbox_pending: 0,
    drift_micro_max: 0,
    drift_alert: !1,
    emergency_breaker: '',
  };
}
function Lx() {
  return di(200, { doctor: Cm(), stackHealth: wm(), dashboardSummary: Rm() });
}
function Cx() {
  return di(200, {
    emergency_breaker: '',
    shards: [
      { shard_id: 0, ping_ok: !0, ping_latency_ms: 1.4, config_version_synced: !0 },
      { shard_id: 1, ping_ok: !0, ping_latency_ms: 1.8, config_version_synced: !0 },
    ],
    affected_campaigns: [],
    partial: !1,
    stale_dashboard: !1,
  });
}
function wx() {
  return di(200, {
    emergency_breaker: '',
    shards: [
      {
        shard_id: 0,
        ping_ok: !0,
        ping_latency_ms: 1.2,
        config_version: 42,
        config_version_lag: 0,
        config_version_synced: !0,
      },
      {
        shard_id: 1,
        ping_ok: !0,
        ping_latency_ms: 1.5,
        config_version: 42,
        config_version_lag: 0,
        config_version_synced: !0,
      },
    ],
  });
}
function Rx() {
  return di(200, {});
}
function _x(e, t = 50, a = 0) {
  let r = O(0),
    o = [];
  if (e.includes('/ops/blacklist'))
    o = Array.from({ length: 12 }, (l, i) => ({
      id: 2e3 + i,
      ip: `203.0.113.${10 + i}`,
      reason: i % 2 === 0 ? 'fraud_score' : 'manual_block',
      created_at: O(i),
      created_at_display: O(i),
      expires_at: O(-i - 1),
      expires_at_display: O(-i - 1),
    }));
  else if (e.includes('/ops/outbox'))
    o = Array.from({ length: 15 }, (l, i) => ({
      id: 3e3 + i,
      event_type: ['campaign_update', 'billing_sync', 'fraud_action'][i % 3],
      status: i % 4 === 0 ? 'pending' : 'applied',
      created_at: O(i % 10, i),
    }));
  else if (e.includes('/ops/dlq/inbox')) {
    let l = oe().campaigns;
    o = Array.from({ length: 8 }, (i, s) => ({
      id: U('ops_dlq', s + 1),
      source: 'processor',
      campaign_id: l[s % l.length]?.id,
      event_type: 'track',
      error: 'redis timeout',
      failed_at: O(s),
      failed_at_display: O(s),
      status: 'pending',
      retry_count: s % 3,
      shard_id: s % 4,
    }));
  } else
    (e.includes('/ops/recon') || e.includes('/recon/runs')) &&
      (o = Array.from({ length: 6 }, (l, i) => ({
        id: U('recon_run', i + 1),
        customer_id: se[i % se.length].id,
        status: i % 2 === 0 ? 'ok' : 'drift',
        diff_micro: i % 2 === 0 ? 0 : 125e5,
        created_at: r,
      })));
  let n = o.slice(a, a + t);
  return di(200, { items: n, total: o.length, limit: t, offset: a });
}
var Ix = [
    {
      key: 'meta_social_funnel',
      title: 'Meta social funnel',
      description: 'Facebook and Instagram click-to-lander flow with conversion mapping.',
      traffic_family: 'social',
      default_flow: {
        flow_name: 'Meta default',
        lander: { name: 'Social pre-lander', url: 'https://example.com/lander' },
        offer: { name: 'Main offer', url: 'https://example.com/offer' },
      },
      integration_schema_refs: ['binom_v1', 'keitaro_v1'],
      sample_macros: { click_id: '{clickid}', campaign_id: '{campaign_id}' },
    },
    {
      key: 'popunder_propeller',
      title: 'Popunder Propeller',
      description: 'Pop traffic with direct offer routing and spend cap defaults.',
      traffic_family: 'pop',
      default_flow: {
        flow_name: 'Popunder main',
        lander: { name: 'Pop bridge', url: 'https://example.com/pop' },
        offer: { name: 'Pop offer', url: 'https://example.com/pop-offer' },
      },
      integration_schema_refs: ['binom_v1'],
      sample_macros: { zone_id: '{zoneid}' },
    },
    {
      key: 'push_house_funnel',
      title: 'Push house funnel',
      description: 'Push notification source with hold-friendly postback mapping.',
      traffic_family: 'push',
      default_flow: {
        flow_name: 'Push route',
        lander: { name: 'Push lander', url: 'https://example.com/push' },
        offer: { name: 'Push offer', url: 'https://example.com/push-offer' },
      },
      integration_schema_refs: ['keitaro_v1'],
      sample_macros: { push_id: '{pushid}' },
    },
    {
      key: 'native_mgid_funnel',
      title: 'Native MGID funnel',
      description: 'Native placements with teaser URL macros and geo targeting.',
      traffic_family: 'native',
      default_flow: {
        flow_name: 'Native MGID',
        lander: { name: 'Native article', url: 'https://example.com/native' },
        offer: { name: 'Native offer', url: 'https://example.com/native-offer' },
      },
      integration_schema_refs: ['binom_v1', 'keitaro_v1'],
      sample_macros: { teaser_id: '{teaser_id}' },
    },
  ],
  fi = new Map();
function bI(e) {
  return Ix.find((t) => t.key === e);
}
function xI(e) {
  let t = e.integration_schema_refs?.[0] ?? 'binom_v1';
  return {
    traffic_source: {
      name: e.title ?? 'Campaign',
      traffic_template_id: 'default_rtb',
      click_query_params: e.sample_macros ?? {},
    },
    integration_template: { integration_schema: t, affiliate_network: '', tracking_domain: '' },
    flow_skeleton: {
      flow_name: e.default_flow?.flow_name ?? 'Main flow',
      lander: {
        name: e.default_flow?.lander?.name ?? 'Lander',
        url: e.default_flow?.lander?.url ?? 'https://example.com/lander',
      },
      offer: {
        name: e.default_flow?.offer?.name ?? 'Offer',
        url: e.default_flow?.offer?.url ?? 'https://example.com/offer',
      },
    },
    budget: { budget_limit_micro: 5e8, timezone: 'UTC', target_countries: ['US'] },
  };
}
function SI(e) {
  return {
    preview: {
      campaign_name: e.traffic_source?.name,
      traffic_template_id: e.traffic_source?.traffic_template_id,
      integration_schema: e.integration_template?.integration_schema,
      flow_name: e.flow_skeleton?.flow_name,
      budget_limit_micro: e.budget?.budget_limit_micro,
      target_url: e.flow_skeleton?.offer?.url,
    },
    warning_slugs: [],
  };
}
function kx() {
  return Ix;
}
function Mx(e) {
  let t = fi.get(e);
  return t
    ? { status: 200, body: t }
    : { status: 404, body: { error: 'campaign wizard session not found' } };
}
function Ex(e) {
  let t = String(e.action ?? '');
  if (t === 'create') {
    let o = String(e.customer_id ?? ''),
      n = String(e.template_key ?? '');
    if (!o || !n)
      return { status: 400, body: { error: 'customer_id and template_key are required' } };
    let l = bI(n);
    if (!l) return { status: 400, body: { error: 'unknown template_key' } };
    let i = vu(),
      s = new Date(),
      u = new Date(s.getTime() + 1440 * 60 * 1e3),
      d = xI(l),
      c = {
        session_id: i,
        customer_id: o,
        current_step: 'traffic_source',
        completed_steps: [],
        steps: d,
        ready_to_commit: !1,
        expires_at: u.toISOString(),
        updated_at: s.toISOString(),
        template_key: n,
      };
    return fi.set(i, c), { status: 201, body: c };
  }
  let a = String(e.session_id ?? ''),
    r = fi.get(a);
  if (!r) return { status: 404, body: { error: 'campaign wizard session not found' } };
  if (t === 'update') {
    let o = String(e.step ?? ''),
      n = e.payload ?? {},
      l = { ...r.steps };
    if (o === 'traffic_source') l.traffic_source = n;
    else if (o === 'integration_template') l.integration_template = n;
    else if (o === 'flow_skeleton') l.flow_skeleton = n;
    else if (o === 'budget') l.budget = n;
    else return { status: 400, body: { error: 'unknown wizard step' } };
    let i = Array.from(new Set([...r.completed_steps, o])),
      s = ['traffic_source', 'integration_template', 'flow_skeleton', 'budget'],
      u = s.every((f) => i.includes(f)),
      d = u ? 'review' : (s.find((f) => !i.includes(f)) ?? 'review'),
      c = {
        ...r,
        steps: l,
        completed_steps: i,
        current_step: d,
        ready_to_commit: u,
        review: u ? SI(l) : void 0,
        updated_at: new Date().toISOString(),
      };
    return fi.set(a, c), { status: 200, body: c };
  }
  if (t === 'commit') {
    if (!r.ready_to_commit)
      return { status: 409, body: { error: 'campaign wizard session incomplete' } };
    let o = se.find((l) => l.id === r.customer_id),
      n = {
        campaign: { id: vu(), name: r.steps.traffic_source?.name ?? o?.name ?? 'New campaign' },
        published: !!e.publish,
      };
    return fi.delete(a), { status: 200, body: n };
  }
  return { status: 400, body: { error: 'invalid action' } };
}
function Ax(e, t) {
  let a = Number.parseFloat(e ?? ''),
    r = Number.parseFloat(t ?? '');
  if (!(!Number.isFinite(a) || a <= 0 || !Number.isFinite(r)))
    return Math.min(100, Math.max(0, (r / a) * 100));
}
function yD(e) {
  return e >= 100 ? '100%' : e < 0.1 && e > 0 ? '<0.1%' : `${e.toFixed(1)}%`;
}
function _m(e) {
  let t = e.display_id?.trim();
  if (t && /^\d{8}$/.test(t)) return t;
  let a = 0;
  for (let r = 0; r < e.id.length; r += 1) a = (a * 31 + e.id.charCodeAt(r)) >>> 0;
  return String(1e7 + (a % 9e7)).padStart(8, '0');
}
function LI(e, t) {
  let a = Date.parse(e),
    r = Date.parse(t);
  return !Number.isFinite(a) || !Number.isFinite(r) || r <= a
    ? 1
    : Math.max(1, (r - a) / 864e5) / 7;
}
function mi(e, t, a, r) {
  let o = LI(a, r),
    n = (p) => Math.max(0, Math.round(p * o)),
    l = n(12e3 + t * 1340),
    i = n(420 + t * 37),
    s = n(18 + (t % 11)),
    u = n(3 + (t % 5)),
    d = n(2 + (t % 4)),
    c = s + u + d,
    f = n(Math.floor((420 + t * 37) * (0.42 + (t % 7) * 0.04))),
    h = Math.max(f, n(Math.floor((420 + t * 37) * 0.88))),
    v = n(4 + (t % 9)),
    x = n(12e5 + t * 88e3),
    y = n(18e4 + t * 12e3 - (t % 5) * 4e4),
    m = x + y;
  return {
    impressions: l,
    clicks: i,
    conversions: s,
    unique_clicks: n(Math.floor((420 + t * 37) * 0.82)),
    blocks: t % 7,
    rtb_cost_micro: x,
    profit_micro: y,
    revenue_micro: m,
    leads_raw: c,
    hold_leads: u,
    rejected_leads: d,
    lp_clicks: f,
    lp_views: h,
    bots: v,
  };
}
function Im(e) {
  if (e.leads_raw > 0) return;
  let t = e.conversions + e.hold_leads + e.rejected_leads;
  e.leads_raw = t > 0 ? t : e.conversions;
}
function On(e, t) {
  if (!(t <= 0 || e <= 0)) return (e / t) * 100;
}
function wu(e, t) {
  Im(t);
  let a = t.revenue_micro,
    r = t.rtb_cost_micro,
    o = t.profit_micro;
  (e.revenue_micro = a),
    (e.cost_micro = r),
    (e.profit_micro = o),
    t.clicks > 0 &&
      ((e.epc_micro = Math.trunc(a / t.clicks)), (e.cpc_micro = Math.trunc(r / t.clicks))),
    t.leads_raw > 0 && (e.cpa_micro = Math.trunc(r / t.leads_raw)),
    t.conversions > 0 && (e.ecpa_micro = Math.trunc(r / t.conversions));
  let n = On(t.clicks, t.impressions);
  n != null && (e.ctr_pct = n);
  let l = On(t.lp_clicks, t.clicks);
  l != null && (e.lp_ctr_pct = l), (e.cr_pct = On(t.conversions, t.clicks) ?? 0);
  let i = On(t.conversions, t.leads_raw);
  i != null && (e.approve_rate_pct = i);
  let s = On(t.blocks, t.clicks);
  s != null && (e.block_pct = s);
  let u = On(t.bots, t.clicks);
  if (
    (u != null && (e.bot_pct = u), r > 0 && (e.roi_pct = (o / r) * 100), t.impressions > 0 && r > 0)
  ) {
    let d = (r * 1e3) / t.impressions;
    e.cpm_usd = (d / 1e6).toFixed(2);
  }
}
function Tx(e, t, a) {
  Im(t), Im(a);
  let r = (o) => {
    switch (e) {
      case 'clicks':
        return o.clicks;
      case 'impressions':
        return o.impressions;
      case 'conversions':
        return o.conversions;
      case 'unique_clicks':
        return o.unique_clicks;
      case 'blocks':
        return o.blocks;
      case 'cost':
        return o.rtb_cost_micro;
      case 'revenue':
        return o.revenue_micro;
      case 'profit':
        return o.profit_micro;
      case 'ctr':
        return o.impressions > 0 ? o.clicks / o.impressions : 0;
      case 'cr':
        return o.clicks > 0 ? o.conversions / o.clicks : 0;
      case 'block_pct':
        return o.clicks > 0 ? o.blocks / o.clicks : 0;
      case 'cpc':
        return o.clicks > 0 ? o.rtb_cost_micro / o.clicks : 0;
      case 'cpa':
      case 'ecpa':
        return o.conversions > 0 ? o.rtb_cost_micro / o.conversions : 0;
      case 'epc':
        return o.clicks > 0 ? o.revenue_micro / o.clicks : 0;
      case 'cpm':
        return o.impressions > 0 && o.rtb_cost_micro > 0 ? o.rtb_cost_micro / o.impressions : 0;
      case 'roi':
        return o.rtb_cost_micro > 0 ? o.profit_micro / o.rtb_cost_micro : 0;
      case 'leads':
        return o.leads_raw;
      case 'approved':
        return o.conversions;
      case 'hold_leads':
        return o.hold_leads;
      case 'rejected_leads':
        return o.rejected_leads;
      case 'approve_rate':
        return o.leads_raw > 0 ? o.conversions / o.leads_raw : 0;
      case 'lp_clicks':
        return o.lp_clicks;
      case 'lp_views':
        return o.lp_views;
      case 'lp_ctr':
        return o.clicks > 0 ? o.lp_clicks / o.clicks : 0;
      case 'bots':
        return o.bots;
      case 'bot_pct':
        return o.clicks > 0 ? o.bots / o.clicks : 0;
      default:
        return 0;
    }
  };
  return r(t) - r(a);
}
function km(e) {
  switch (e) {
    case 'clicks':
    case 'impressions':
    case 'conversions':
    case 'unique_clicks':
    case 'blocks':
    case 'cost':
    case 'revenue':
    case 'profit':
    case 'roi':
    case 'ctr':
    case 'cr':
    case 'cpc':
    case 'cpa':
    case 'ecpa':
    case 'epc':
    case 'cpm':
    case 'block_pct':
    case 'leads':
    case 'approved':
    case 'hold_leads':
    case 'rejected_leads':
    case 'approve_rate':
    case 'lp_clicks':
    case 'lp_views':
    case 'lp_ctr':
    case 'bots':
    case 'bot_pct':
      return !0;
    default:
      return !1;
  }
}
function Pn(e, t) {
  return { status: e, body: t };
}
function CI(e, t) {
  let a = {};
  for (let r of t) {
    let o = e.searchParams.get(r);
    o != null && o !== '' && (a[r] = o);
  }
  return a;
}
function wI(e) {
  let t = 0,
    a = 0,
    r = 0;
  for (let o of e)
    switch (o.status) {
      case 'ACTIVE':
        t++;
        break;
      case 'PAUSED':
        a++;
        break;
      case 'ARCHIVED':
        r++;
        break;
      default:
        break;
    }
  return { active: t, paused: a, archived: r, total: e.length };
}
function Ht(e) {
  let t = Number.parseFloat(e ?? '');
  return Number.isFinite(t) ? Math.round(t * 1e6) : 0;
}
function RI(e, t, a, r) {
  switch (a) {
    case 'id':
      return Number.parseInt(_m(e), 10) - Number.parseInt(_m(t), 10);
    case 'name':
      return e.name.localeCompare(t.name);
    case 'updated_at':
      return e.updated_at.localeCompare(t.updated_at);
    case 'spend':
      return Ht(e.current_spend) - Ht(t.current_spend);
    case 'budget_limit':
      return Ht(e.budget_limit) - Ht(t.budget_limit);
    case 'status':
      return (e.status ?? '').localeCompare(t.status ?? '');
    case 'budget_pct': {
      let o = Ht(e.budget_limit),
        n = Ht(t.budget_limit),
        l = o > 0 ? Ht(e.current_spend) / o : 0,
        i = n > 0 ? Ht(t.current_spend) / n : 0;
      return l - i;
    }
    default: {
      let o = r.get(e.id),
        n = r.get(t.id);
      return !o || !n ? e.name.localeCompare(t.name) : Tx(a, o, n);
    }
  }
}
function Dx(e, t) {
  let a = e.searchParams.get('customer_id') ?? '',
    r = e.searchParams.get('status') ?? '',
    o = (e.searchParams.get('q') ?? '').trim().toLowerCase(),
    n = e.searchParams.get('pacing_mode') ?? '',
    l = e.searchParams.get('owner_user_id') ?? '',
    i = e.searchParams.get('country') ?? '',
    s = e.searchParams.get('budget_min_micro'),
    u = e.searchParams.get('budget_max_micro'),
    d = e.searchParams.get('sort') ?? 'name',
    c = e.searchParams.get('order') === 'desc' ? 'desc' : 'asc',
    f = e.searchParams.get('from') ?? '',
    h = e.searchParams.get('to') ?? '',
    v = Math.min(200, Math.max(1, Number.parseInt(e.searchParams.get('limit') ?? '50', 10) || 50)),
    x = Math.max(0, Number.parseInt(e.searchParams.get('offset') ?? '0', 10) || 0);
  if (km(d) && (!f.trim() || !h.trim()))
    return Pn(400, {
      error: { code: 'INVALID_QUERY', message: 'from and to required for metric sort' },
    });
  let y = [...t];
  if (
    (a && (y = y.filter((I) => I.customer_id === a)),
    n && (y = y.filter((I) => I.pacing_mode === n)),
    l && (y = y.filter((I) => I.owner_user_id === l)),
    i && (y = y.filter((I) => I.target_countries?.includes(i))),
    o && (y = y.filter((I) => I.name.toLowerCase().includes(o) || I.id.includes(o))),
    s)
  ) {
    let I = Number.parseInt(s, 10);
    Number.isFinite(I) && (y = y.filter((w) => Ht(w.budget_limit) >= I));
  }
  if (u) {
    let I = Number.parseInt(u, 10);
    Number.isFinite(I) && (y = y.filter((w) => Ht(w.budget_limit) <= I));
  }
  let m = wI(y);
  r && (y = y.filter((I) => I.status === r));
  let p = new Map();
  if (km(d)) {
    let I = f.trim() || new Date(Date.now() - 6048e5).toISOString(),
      w = h.trim() || new Date().toISOString();
    y.forEach((L, M) => {
      p.set(L.id, mi(L.id, M + 1, I, w));
    });
  }
  y.sort((I, w) => {
    let L = RI(I, w, d, p);
    return L === 0 && (L = I.name.localeCompare(w.name)), c === 'desc' ? -L : L;
  });
  let g = y.length,
    b = y.slice(x, x + v).map((I) => {
      let w = Ax(I.budget_limit, I.current_spend);
      return w == null ? I : { ...I, budget_used_pct: w };
    }),
    R = CI(e, [
      'customer_id',
      'status',
      'q',
      'pacing_mode',
      'owner_user_id',
      'country',
      'budget_min_micro',
      'budget_max_micro',
      'from',
      'to',
    ]);
  return Pn(200, {
    items: b,
    total: g,
    limit: v,
    offset: x,
    sort: { field: d, order: c },
    status_totals: m,
    ...(Object.keys(R).length > 0 ? { filters_applied: R } : {}),
  });
}
function Ox(e, t, a) {
  let r = e.searchParams.get('customer_id') ?? '',
    o = [...t];
  r && (o = o.filter((i) => i.customer_id === r));
  let n = new Set(),
    l = new Map();
  for (let i of o) {
    for (let d of i.target_countries ?? []) {
      let c = d?.trim();
      c && n.add(c);
    }
    let s = i.owner_user_id?.trim();
    if (!s || l.has(s)) continue;
    let u = a[s];
    l.set(s, u ? { user_id: s, email: u } : { user_id: s });
  }
  return Pn(200, {
    countries: [...n].sort(),
    owners: [...l.values()].sort((i, s) => {
      let u = i.email ?? i.user_id,
        d = s.email ?? s.user_id;
      return u.localeCompare(d);
    }),
  });
}
function _I(e, t) {
  let a = e.searchParams.get('customer_id') ?? '',
    r = e.searchParams.get('status') ?? '',
    o = (e.searchParams.get('q') ?? '').trim().toLowerCase(),
    n = e.searchParams.get('pacing_mode') ?? '',
    l = e.searchParams.get('owner_user_id') ?? '',
    i = e.searchParams.get('country') ?? '',
    s = e.searchParams.get('budget_min_micro'),
    u = e.searchParams.get('budget_max_micro'),
    d = [...t];
  if (
    (a && (d = d.filter((c) => c.customer_id === a)),
    n && (d = d.filter((c) => c.pacing_mode === n)),
    l && (d = d.filter((c) => c.owner_user_id === l)),
    i && (d = d.filter((c) => c.target_countries?.includes(i))),
    o && (d = d.filter((c) => c.name.toLowerCase().includes(o) || c.id.includes(o))),
    s)
  ) {
    let c = Number.parseInt(s, 10);
    Number.isFinite(c) && (d = d.filter((f) => Ht(f.budget_limit) >= c));
  }
  if (u) {
    let c = Number.parseInt(u, 10);
    Number.isFinite(c) && (d = d.filter((f) => Ht(f.budget_limit) <= c));
  }
  return r && (d = d.filter((c) => c.status === r)), d;
}
function Px(e, t) {
  let a = e.searchParams.get('from') ?? new Date(Date.now() - 6048e5).toISOString(),
    r = e.searchParams.get('to') ?? new Date().toISOString(),
    o = _I(e, t),
    n = o.filter((u) => u.flow_id).length,
    l = {
      impressions: 0,
      clicks: 0,
      conversions: 0,
      unique_clicks: 0,
      blocks: 0,
      leads_raw: 0,
      hold_leads: 0,
      rejected_leads: 0,
      lp_clicks: 0,
      lp_views: 0,
      bots: 0,
      advertiser_spend_micro: 0,
      rtb_cost_micro: 0,
      operator_margin_micro: 0,
      publisher_payout_micro: 0,
    },
    i = 0;
  o.forEach((u, d) => {
    (d + 1) % 13 === 0 && (i += 1);
    let c = mi(u.id, d + 1, a, r);
    (l.impressions += c.impressions),
      (l.clicks += c.clicks),
      (l.conversions += c.conversions),
      (l.unique_clicks += c.unique_clicks),
      (l.blocks += c.blocks),
      (l.leads_raw += c.leads_raw),
      (l.hold_leads += c.hold_leads),
      (l.rejected_leads += c.rejected_leads),
      (l.lp_clicks += c.lp_clicks),
      (l.lp_views += c.lp_views),
      (l.bots += c.bots),
      (l.advertiser_spend_micro += c.rtb_cost_micro + c.profit_micro),
      (l.rtb_cost_micro += c.rtb_cost_micro),
      (l.operator_margin_micro += c.profit_micro),
      (l.publisher_payout_micro += Math.floor(c.rtb_cost_micro * 0.72));
  });
  let s = { ...l };
  return (
    wu(s, {
      impressions: l.impressions,
      clicks: l.clicks,
      conversions: l.conversions,
      unique_clicks: l.unique_clicks,
      blocks: l.blocks,
      rtb_cost_micro: l.rtb_cost_micro,
      profit_micro: l.operator_margin_micro,
      revenue_micro: l.advertiser_spend_micro,
      leads_raw: l.leads_raw,
      hold_leads: l.hold_leads,
      rejected_leads: l.rejected_leads,
      lp_clicks: l.lp_clicks,
      lp_views: l.lp_views,
      bots: l.bots,
    }),
    Pn(200, {
      campaign_count: o.length,
      flow_count: n,
      margin_breach_count: i,
      totals: s,
      from: a,
      to: r,
      stale: !1,
    })
  );
}
function II(e, t, a) {
  let r = [],
    o = new Date();
  o.setUTCMinutes(0, 0, 0);
  for (let n = 23; n >= 0; n -= 1) {
    let l = new Date(o.getTime() - n * 60 * 60 * 1e3),
      i = (n % 5) + 1;
    r.push({
      hour: l.toISOString(),
      impressions: Math.floor((e * i) / 90),
      clicks: Math.floor((t * i) / 90),
      conversions: Math.floor((a * i) / 90),
    });
  }
  return r;
}
function Bx(e, t) {
  let a = oe().campaigns.find((u) => u.id === e),
    r = new Date(),
    o = t.searchParams.get('from') ?? new Date(r.getTime() - 10080 * 60 * 1e3).toISOString(),
    n = t.searchParams.get('to') ?? r.toISOString(),
    l = 42e3,
    i = 1240,
    s = 86;
  return Pn(200, {
    campaign_id: e,
    current_spend: a?.current_spend ?? '0.00',
    metrics: { impressions: l, clicks: i, conversions: s },
    hourly: II(l, i, s),
    granularity: t.searchParams.get('granularity') ?? 'hour',
    from: o,
    to: n,
    stale: !0,
    source: 'pg',
    consistency: 'strong',
  });
}
function Ux(e) {
  let a = (e.searchParams.get('ids') ?? '')
      .split(',')
      .map((i) => i.trim())
      .filter((i) => i.length > 0),
    { campaigns: r } = oe(),
    o = {},
    n = [],
    l = new Date().toISOString();
  for (let i of a) {
    let s = r.find((u) => u.id === i);
    if (!s) {
      n.push({ id: i, error_code: 'NOT_FOUND' });
      continue;
    }
    o[i] = {
      export_version: '1',
      exported_at: l,
      campaign: {
        name: s.name,
        budget_limit_micro: Ht(s.budget_limit),
        pacing_mode: s.pacing_mode ?? 'ASAP',
        timezone: s.timezone ?? 'UTC',
      },
    };
  }
  return Pn(200, { items: o, errors: n });
}
var Nx = ['finalized', 'draft', 'void'];
function kI() {
  let e = [];
  for (let t = 0; t < 18; t += 1) {
    let a = se[t % se.length],
      r = ae(4200 + t * 317.5),
      o = Math.floor(r * 0.08),
      n = r + o,
      l = -(t % 6);
    e.push({
      id: U('invoice', t + 1),
      customer_id: a.id,
      billing_month: tx(l),
      subtotal_micro: r,
      subtotal_micro_display: ho(r),
      tax_micro: o,
      tax_micro_display: ho(o),
      total_micro: n,
      total_micro_display: ho(n),
      currency: 'USD',
      tax_scheme: 'US_SALES',
      tax_rate_bps: 800,
      status: Nx[t % Nx.length],
      created_at: O(t % 20),
      updated_at: O(t % 12, t),
    });
  }
  return e;
}
var Mm = kI();
function Hx() {
  let e = Mm.filter((r) => r.status === 'finalized'),
    t = e.reduce((r, o) => r + o.total_micro, 0),
    a = new Set(e.map((r) => r.customer_id));
  return {
    invoiced_mtd_micro: t,
    invoiced_mtd_display: ho(t),
    invoice_count_mtd: e.length,
    invoice_count_mtd_display: ym(e.length),
    undelivered_invoice_notifications: 2,
    undelivered_invoice_notifications_display: '2',
    customers_with_spend_in_month: a.size,
    customers_with_spend_in_month_display: ym(a.size),
  };
}
function zx(e) {
  let { limit: t, offset: a } = yo(e),
    r = e.searchParams.get('month')?.trim(),
    o = e.searchParams.get('status')?.trim(),
    n = Mm;
  return (
    r && (n = n.filter((i) => i.billing_month === r)),
    o && (n = n.filter((i) => i.status === o)),
    { ...go(n, t, a), limit: t, offset: a }
  );
}
function Fx(e) {
  return Mm.find((t) => t.id === e);
}
function qx(e) {
  let t = e ? se.find((r) => r.id === e) : se[0],
    a = ae(18420.55);
  return {
    ok: !0,
    customer_id: t?.id,
    balance_micro: a,
    ledger_sum_micro: a,
    diff_micro: 0,
    fleet_scan_limit: 500,
  };
}
function Em(e) {
  let t = se.findIndex((r) => r.id === e),
    a = ae(12e3 + (t >= 0 ? t : 0) * 2450.75);
  return {
    customer_id: e,
    balance_micro: a,
    balance_micro_display: ho(a),
    currency: 'USD',
    updated_at: O(0, 2),
  };
}
function Gx(e) {
  let t = Em(e);
  return {
    customer_id: e,
    currency: 'USD',
    available_micro: t.balance_micro,
    reserved_micro: ae(250),
    updated_at: t.updated_at,
  };
}
function Vx(e, t) {
  let { limit: a, offset: r } = yo(t, 100),
    o = Array.from({ length: 24 }, (l, i) => {
      let s = i % 3 !== 0,
        u = ae(35 + i * 12.4);
      return {
        id: U('ledger', i + 1),
        customer_id: e,
        campaign_id: U('campaign', (i % 12) + 1),
        entry_type: s ? 'debit' : 'credit',
        amount_micro: s ? -u : u,
        amount_micro_display: ho(s ? -u : u),
        balance_after_micro: ae(12e3 - i * 40),
        description: s ? 'Campaign spend sync' : 'Invoice payment',
        created_at: O(i % 30, i),
        created_at_display: O(i % 30, i),
      };
    });
  return { ...go(o, a, r), limit: a, offset: r };
}
function jx(e, t) {
  let a = t.searchParams.get('from') ?? O(30),
    r = t.searchParams.get('to') ?? O(0),
    o = Array.from({ length: 8 }, (n, l) => ({
      date: O(l * 3),
      description: l % 2 === 0 ? 'Media spend' : 'Adjustment',
      amount_micro: ae(l % 2 === 0 ? -(180 + l * 22) : 95 + l * 11),
      running_balance_micro: ae(1e4 - l * 120),
    }));
  return {
    customer_id: e,
    from: a,
    to: r,
    currency: 'USD',
    opening_balance_micro: ae(10960),
    closing_balance_micro: ae(9040),
    lines: o,
  };
}
function Bn(e) {
  let t = oe().campaigns;
  return t[(e - 1) % t.length]?.id ?? U('campaign', e);
}
function MI(e) {
  let t = oe().campaigns;
  return t[(e - 1) % t.length]?.name ?? `Campaign ${e}`;
}
function zt(e) {
  return se[(e - 1) % se.length].id;
}
function pi() {
  return {
    networks: [
      { network: 'facebook', display_name: 'Meta Ads', enabled: !0 },
      { network: 'google', display_name: 'Google Ads', enabled: !0 },
      { network: 'tiktok', display_name: 'TikTok Ads', enabled: !1 },
    ],
    credentials: [
      {
        id: U('cost_cred', 1),
        network: 'facebook',
        customer_id: zt(1),
        sync_interval_minutes: 60,
        last_sync_at: O(0, 3),
        status: 'ok',
      },
      {
        id: U('cost_cred', 2),
        network: 'google',
        customer_id: zt(2),
        sync_interval_minutes: 120,
        last_sync_at: O(1, 5),
        status: 'degraded',
      },
    ],
    history: Array.from({ length: 6 }, (e, t) => ({
      id: U('cost_hist', t + 1),
      network: t % 2 === 0 ? 'facebook' : 'google',
      customer_id: zt(t + 1),
      started_at: O(t),
      finished_at: O(t, 1),
      status: t % 3 === 0 ? 'failed' : 'completed',
      rows_upserted: 120 + t * 40,
    })),
  };
}
function hi() {
  let e = oe().campaigns.slice(0, 6);
  return {
    configs: e.map((t, a) => ({
      campaign_id: t.id,
      provider: ['webhook', 'facebook', 'google', 'tiktok'][a % 4],
      url_template: bx(a + 1),
      target_event: 'conversion',
      test_event_code: a % 2 === 0 ? 'TEST_EVENT_42' : '',
      has_api_token: a % 3 !== 0,
    })),
    dlq: Array.from({ length: 4 }, (t, a) => ({
      id: 1e3 + a,
      outbox_event_id: 5e3 + a,
      campaign_id: e[a % e.length].id,
      click_id: Tn(a + 1),
      event_type: 'conversion',
      payload: { goal: 'lead' },
      failures_count: 2 + a,
      last_error: 'upstream timeout',
      status: 'pending',
    })),
    campaignStatus: e.map((t, a) => ({
      campaign_id: t.id,
      provider: ['webhook', 'facebook'][a % 2],
      last_success_at: O(a % 5),
      dlq_pending_count: a % 3,
    })),
  };
}
function Ru() {
  return {
    schemas: [
      { id: U('schema', 1), name: 'Affiliate webhook v2', version: '2.1', updated_at: O(4) },
      { id: U('schema', 2), name: 'Facebook CAPI', version: '1.0', updated_at: O(12) },
    ],
    templates: [
      {
        name: 'Affiliate webhook v2',
        file: 'affiliate_webhook_v2.json',
        version: 2,
        category: 'postback',
        kind: 'schema',
      },
      {
        name: 'Facebook purchase',
        file: 'facebook_purchase.json',
        version: 1,
        category: 'capi',
        kind: 'schema',
      },
    ],
  };
}
function Yx() {
  return oe()
    .campaigns.slice(0, 10)
    .map((t, a) => ({
      id: U('platform_link', a + 1),
      campaign_id: t.id,
      customer_id: t.customer_id,
      network: ['facebook', 'google', 'tiktok', 'taboola'][a % 4],
      external_campaign_id: px(a + 1),
      status: a % 5 === 0 ? 'paused' : 'active',
      last_sync_at: O(a % 8),
      updated_at: O(a % 6),
    }));
}
function Am(e) {
  let t = O(0),
    a = oe().campaigns;
  switch (e) {
    case '/api/v1/flows':
      return Array.from({ length: 8 }, (o, n) => ({
        id: U('flow', n + 1),
        name: de(nx, n + 1),
        paths: [{ weight: 70, lander_id: U('lander', n + 1), offer_id: U('offer', n + 1) }],
        created_at: O(n + 3),
      }));
    case '/api/v1/landers':
      return Array.from({ length: 10 }, (o, n) => ({
        id: U('lander', n + 1),
        name: Lu(n + 1),
        url: yx(n + 1),
        customer_id: zt(n + 1),
        created_at: O(n + 2),
        updated_at: t,
      }));
    case '/api/v1/offers':
      return Array.from({ length: 10 }, (o, n) => ({
        id: U('offer', n + 1),
        name: de(bu, n + 1),
        payout_micro: ae(18 + n * 3.5),
        currency: 'USD',
        customer_id: zt(n + 1),
        created_at: O(n + 1),
      }));
    case '/api/v1/brands':
      return se.map((o, n) => ({
        id: U('brand', n + 1),
        customer_id: o.id,
        name: `${de(ox, n + 1)} - US East`,
        created_at: O(n + 5),
        updated_at: t,
        freq_limit: 3,
        freq_window: 86400,
      }));
    case '/api/v1/domains':
      return Array.from({ length: 6 }, (o, n) => ({
        id: U('domain', n + 1),
        hostname: de(An, n + 1),
        customer_id: zt(n + 1),
        status: n % 4 === 0 ? 'pending' : 'active',
        ssl_status: 'issued',
        created_at: O(n + 4),
      }));
    case '/api/v1/automation/presets':
      return [
        {
          key: 'pause_on_loss',
          title: 'Pause on loss',
          description: 'Pause when ROI stays below zero for 24 hours.',
          parameters_schema: ui(),
        },
        {
          key: 'boost_winner_geo',
          title: 'Boost winning GEO',
          description: 'Raise budget on the best-performing country slice.',
          parameters_schema: ui(),
        },
      ];
    case '/api/v1/automation/rules':
      return Array.from({ length: 5 }, (o, n) => ({
        id: U('automation_rule', n + 1),
        name: de(sx, n + 1),
        enabled: n % 3 !== 0,
        trigger: 'margin_breach',
        customer_id: zt(n + 1),
        updated_at: O(n),
      }));
    case '/api/v1/fraud/presets':
      return [
        {
          name: 'strict',
          pass: 20,
          suspect: 45,
          ivt: 70,
          block: 85,
          updated_at: t,
          updated_at_display: t,
        },
        {
          name: 'balanced',
          pass: 35,
          suspect: 60,
          ivt: 80,
          block: 92,
          updated_at: O(2),
          updated_at_display: O(2),
        },
      ];
    case '/api/v1/integration/affiliate-status-presets':
      return [
        {
          name: 'Standard affiliate network',
          statuses: [
            { inbound_status: 'approved', goal_name: 'sale' },
            { inbound_status: 'rejected', goal_name: 'lead' },
          ],
        },
        {
          name: 'Lead gen network',
          statuses: [
            { inbound_status: 'confirmed', goal_name: 'lead' },
            { inbound_status: 'hold', goal_name: 'lead' },
          ],
        },
      ];
    case '/api/v1/integration/schemas':
      return Ru().schemas;
    case '/api/v1/integration/templates':
      return Ru().templates;
    case '/api/v1/margin-guard/activity':
      return Array.from({ length: 8 }, (o, n) => ({
        id: U('margin_activity', n + 1),
        campaign_id: Bn(n + 1),
        placement_id: Dn(n + 1),
        action: n % 2 === 0 ? 'throttle' : 'pause',
        created_at: O(n),
      }));
    case '/api/v1/margin-guard/policies':
      return Array.from({ length: 4 }, (o, n) => ({
        id: U('margin_policy', n + 1),
        name: de(dx, n + 1),
        min_roi_pct: 8 + n * 2,
        customer_id: zt(n + 1),
        enabled: !0,
      }));
    case '/api/v1/postbacks/campaign-status':
      return hi().campaignStatus;
    case '/api/v1/postbacks/config':
      return hi().configs;
    case '/api/v1/postbacks/dlq':
      return hi().dlq;
    case '/api/v1/report-schedules':
      return Array.from({ length: 4 }, (o, n) => ({
        id: U('report_schedule', n + 1),
        customer_id: zt(n + 1),
        report_key: 'campaign-overview',
        cron: '0 8 * * 1',
        enabled: n % 2 === 0,
      }));
    case '/api/v1/rtb/deals':
      return Array.from({ length: 6 }, (o, n) => ({
        id: n + 1,
        deal_id: hx(n + 1),
        floor_micro: ae(0.35 + n * 0.08),
        geo_mask: 1 << n % 5,
        cat_mask: 1,
        pacing: n % 2 === 0 ? 'even' : 'asap',
        seats: 2 + n,
        customer_id: zt(n + 1),
        created_at: O(n + 6),
        updated_at: t,
      }));
    case '/api/v1/smart-alerts/history':
      return Array.from({ length: 10 }, (o, n) => ({
        id: U('alert_hist', n + 1),
        rule_id: U('alert_rule', (n % 3) + 1),
        campaign_id: Bn(n + 1),
        severity: n % 3 === 0 ? 'critical' : 'warn',
        message: `Spend velocity spike on ${MI(n + 1)}`,
        created_at: O(n),
      }));
    case '/api/v1/smart-alerts/rules':
      return Array.from({ length: 4 }, (o, n) => ({
        id: U('alert_rule', n + 1),
        name: de(ux, n + 1),
        enabled: !0,
        metric: 'spend_velocity',
        customer_id: zt(n + 1),
      }));
    case '/api/v1/supply/ads-txt':
      return Array.from({ length: 5 }, (o, n) => ({
        id: U('ads_txt', n + 1),
        domain: xm(n + 1),
        line: gx(n + 1),
        updated_at: O(n),
      }));
    case '/api/v1/supply/sellers':
      return Array.from({ length: 6 }, (o, n) => ({
        id: n + 1,
        seller_id: de(bm, n + 1),
        name: de(lx, n + 1),
        domain: xm(n + 1),
        seller_type: n % 2 === 0 ? 'PUBLISHER' : 'INTERMEDIARY',
        is_confidential: n % 4 === 0,
        customer_id: zt(n + 1),
        created_at: O(n + 2),
        updated_at: t,
      }));
    case '/api/v1/telegram/bots':
      return Array.from({ length: 4 }, (o, n) => {
        let l = Sm(n + 1);
        return {
          id: U('telegram_bot', n + 1),
          name: l.name,
          campaign_id: Bn(n + 1),
          username: l.username,
          status: n % 3 === 0 ? 'paused' : 'active',
        };
      });
    case '/api/v1/traffic-optimizer/presets':
      return [
        {
          key: 'conservative_split',
          title: 'Conservative split',
          description: 'Slow weight shifts with strict minimum data gates.',
          parameters_schema: ui(),
        },
        {
          key: 'aggressive_scale',
          title: 'Aggressive scale',
          description: 'Faster reallocation toward top paths and sources.',
          parameters_schema: ui(),
        },
      ];
    case '/api/v1/traffic-optimizer/rules':
      return Array.from({ length: 5 }, (o, n) => ({
        id: U('traffic_opt', n + 1),
        name: de(fx, n + 1),
        campaign_id: Bn(n + 1),
        enabled: n % 2 === 0,
      }));
    case '/api/v1/views':
      return Array.from({ length: 5 }, (o, n) => ({
        id: U('saved_view', n + 1),
        name: de(cx, n + 1),
        report_key: 'campaign-overview',
        customer_id: zt(n + 1),
        owner_user_id: Xe[n % Xe.length].id,
      }));
    case '/api/v1/cost-sync/networks':
      return pi().networks;
    case '/api/v1/cost-sync/credentials':
      return pi().credentials;
    case '/api/v1/cost-sync/history':
      return pi().history;
    case '/api/v1/platform-campaigns/links':
      return Yx();
    default:
      break;
  }
  if (e.startsWith('/api/v1/fraud/integrations'))
    return a
      .slice(0, 8)
      .map((o, n) => ({
        campaign_id: o.id,
        provider: ['ivt-detector', 'manual'][n % 2],
        enabled: n % 4 !== 0,
        updated_at: O(n),
      }));
  if (e.startsWith('/api/v1/fraud/labels'))
    return Array.from({ length: 12 }, (o, n) => ({
      id: U('fraud_label', n + 1),
      ip_hash: mx(n + 1),
      label: n % 2 === 0 ? 'block' : 'suspect',
      campaign_id: Bn(n + 1),
      created_at: O(n),
    }));
  if (e.startsWith('/api/v1/integration/platform-campaigns')) return Yx();
  if (e.startsWith('/api/v1/telegram/postbacks'))
    return Array.from({ length: 6 }, (o, n) => ({
      id: U('tg_postback', n + 1),
      campaign_id: Bn(n + 1),
      postback_url: vx(n + 1),
      status: n % 3 === 0 ? 'failed' : 'ok',
      updated_at: O(n),
    }));
  if (e.startsWith('/api/v1/ops/recon'))
    return Array.from({ length: 6 }, (o, n) => ({
      id: U('recon', n + 1),
      status: n % 2 === 0 ? 'matched' : 'drift',
      diff_micro: n % 2 === 0 ? 0 : ae(12.5),
      created_at: O(n),
    }));
  let r = /^\/api\/v1\/brands\/([^/]+)\/creatives$/.exec(e);
  if (r) {
    let o = decodeURIComponent(r[1]);
    return Array.from({ length: 6 }, (n, l) => ({
      id: U('creative', l + 1),
      brand_id: o,
      name: de(ix, l + 1),
      format: l % 2 === 0 ? 'banner' : 'native',
      width: 300,
      height: 250,
      created_at: O(l),
    }));
  }
}
var Xx = [
    'campaign.update',
    'campaign.pause',
    'customer.create',
    'settings.patch',
    'billing.invoice.finalize',
    'fraud.preset.update',
    'rtb.deal.create',
    'team.member.invite',
  ],
  Wx = [
    'campaign',
    'customer',
    'platform_settings',
    'invoice',
    'fraud_preset',
    'rtb_deal',
    'team_member',
  ];
function Qx(e) {
  let { limit: t, offset: a } = yo(e),
    r = Array.from({ length: 40 }, (n, l) => {
      let i = Xe[l % Xe.length],
        s = Xx[l % Xx.length],
        u = Wx[l % Wx.length];
      return {
        id: 1e4 + l,
        admin_id: i.id,
        action: s,
        target_type: u,
        target_id: xx(u, l + 1),
        changes: { field: 'status', from: 'ACTIVE', to: 'PAUSED' },
        metadata: { source: 'dev_mock' },
        is_masked: l % 9 === 0,
        created_at: O(l % 25, l),
        created_at_display: O(l % 25, l),
      };
    });
  return { ...go(r, t, a), limit: t, offset: a };
}
var EI = Math.pow(10, 8) * 24 * 60 * 60 * 1e3,
  qD = -EI,
  _u = 6048e5,
  Kx = 864e5,
  GD = 6e4,
  VD = 36e5;
var AI = 3600;
var Zx = AI * 24,
  jD = Zx * 7,
  TI = Zx * 365.2425,
  DI = TI / 12,
  YD = DI * 3,
  Tm = Symbol.for('constructDateFrom');
function ut(e, t) {
  return typeof e == 'function'
    ? e(t)
    : e && typeof e == 'object' && Tm in e
      ? e[Tm](t)
      : e instanceof Date
        ? new e.constructor(t)
        : new Date(t);
}
function pe(e, t) {
  return ut(t || e, e);
}
var OI = {};
function Ur() {
  return OI;
}
function Za(e, t) {
  let a = Ur(),
    r =
      t?.weekStartsOn ??
      t?.locale?.options?.weekStartsOn ??
      a.weekStartsOn ??
      a.locale?.options?.weekStartsOn ??
      0,
    o = pe(e, t?.in),
    n = o.getDay(),
    l = (n < r ? 7 : 0) + n - r;
  return o.setDate(o.getDate() - l), o.setHours(0, 0, 0, 0), o;
}
function vo(e, t) {
  return Za(e, { ...t, weekStartsOn: 1 });
}
function Iu(e, t) {
  let a = pe(e, t?.in),
    r = a.getFullYear(),
    o = ut(a, 0);
  o.setFullYear(r + 1, 0, 4), o.setHours(0, 0, 0, 0);
  let n = vo(o),
    l = ut(a, 0);
  l.setFullYear(r, 0, 4), l.setHours(0, 0, 0, 0);
  let i = vo(l);
  return a.getTime() >= n.getTime() ? r + 1 : a.getTime() >= i.getTime() ? r : r - 1;
}
function Dm(e) {
  let t = pe(e),
    a = new Date(
      Date.UTC(
        t.getFullYear(),
        t.getMonth(),
        t.getDate(),
        t.getHours(),
        t.getMinutes(),
        t.getSeconds(),
        t.getMilliseconds()
      )
    );
  return a.setUTCFullYear(t.getFullYear()), +e - +a;
}
function ku(e, ...t) {
  let a = ut.bind(null, e || t.find((r) => typeof r == 'object'));
  return t.map(a);
}
function Om(e, t) {
  let a = pe(e, t?.in);
  return a.setHours(0, 0, 0, 0), a;
}
function Jx(e, t, a) {
  let [r, o] = ku(a?.in, e, t),
    n = Om(r),
    l = Om(o),
    i = +n - Dm(n),
    s = +l - Dm(l);
  return Math.round((i - s) / Kx);
}
function $x(e, t) {
  let a = Iu(e, t),
    r = ut(t?.in || e, 0);
  return r.setFullYear(a, 0, 4), r.setHours(0, 0, 0, 0), vo(r);
}
function eS(e) {
  return (
    e instanceof Date ||
    (typeof e == 'object' && Object.prototype.toString.call(e) === '[object Date]')
  );
}
function tS(e) {
  return !((!eS(e) && typeof e != 'number') || isNaN(+pe(e)));
}
function aS(e, t) {
  let [a, r] = ku(e, t.start, t.end);
  return { start: a, end: r };
}
function rS(e, t) {
  let { start: a, end: r } = aS(t?.in, e),
    o = +a > +r,
    n = o ? +a : +r,
    l = o ? r : a;
  l.setHours(0, 0, 0, 0);
  let i = t?.step ?? 1;
  if (!i) return [];
  i < 0 && ((i = -i), (o = !o));
  let s = [];
  for (; +l <= n; ) s.push(ut(a, l)), l.setDate(l.getDate() + i), l.setHours(0, 0, 0, 0);
  return o ? s.reverse() : s;
}
function oS(e, t) {
  let a = pe(e, t?.in);
  return a.setFullYear(a.getFullYear(), 0, 1), a.setHours(0, 0, 0, 0), a;
}
var PI = {
    lessThanXSeconds: { one: 'less than a second', other: 'less than {{count}} seconds' },
    xSeconds: { one: '1 second', other: '{{count}} seconds' },
    halfAMinute: 'half a minute',
    lessThanXMinutes: { one: 'less than a minute', other: 'less than {{count}} minutes' },
    xMinutes: { one: '1 minute', other: '{{count}} minutes' },
    aboutXHours: { one: 'about 1 hour', other: 'about {{count}} hours' },
    xHours: { one: '1 hour', other: '{{count}} hours' },
    xDays: { one: '1 day', other: '{{count}} days' },
    aboutXWeeks: { one: 'about 1 week', other: 'about {{count}} weeks' },
    xWeeks: { one: '1 week', other: '{{count}} weeks' },
    aboutXMonths: { one: 'about 1 month', other: 'about {{count}} months' },
    xMonths: { one: '1 month', other: '{{count}} months' },
    aboutXYears: { one: 'about 1 year', other: 'about {{count}} years' },
    xYears: { one: '1 year', other: '{{count}} years' },
    overXYears: { one: 'over 1 year', other: 'over {{count}} years' },
    almostXYears: { one: 'almost 1 year', other: 'almost {{count}} years' },
  },
  nS = (e, t, a) => {
    let r,
      o = PI[e];
    return (
      typeof o == 'string'
        ? (r = o)
        : t === 1
          ? (r = o.one)
          : (r = o.other.replace('{{count}}', t.toString())),
      a?.addSuffix ? (a.comparison && a.comparison > 0 ? 'in ' + r : r + ' ago') : r
    );
  };
function Mu(e) {
  return (t = {}) => {
    let a = t.width ? String(t.width) : e.defaultWidth;
    return e.formats[a] || e.formats[e.defaultWidth];
  };
}
var BI = { full: 'EEEE, MMMM do, y', long: 'MMMM do, y', medium: 'MMM d, y', short: 'MM/dd/yyyy' },
  UI = { full: 'h:mm:ss a zzzz', long: 'h:mm:ss a z', medium: 'h:mm:ss a', short: 'h:mm a' },
  NI = {
    full: "{{date}} 'at' {{time}}",
    long: "{{date}} 'at' {{time}}",
    medium: '{{date}}, {{time}}',
    short: '{{date}}, {{time}}',
  },
  lS = {
    date: Mu({ formats: BI, defaultWidth: 'full' }),
    time: Mu({ formats: UI, defaultWidth: 'full' }),
    dateTime: Mu({ formats: NI, defaultWidth: 'full' }),
  };
var HI = {
    lastWeek: "'last' eeee 'at' p",
    yesterday: "'yesterday at' p",
    today: "'today at' p",
    tomorrow: "'tomorrow at' p",
    nextWeek: "eeee 'at' p",
    other: 'P',
  },
  iS = (e, t, a, r) => HI[e];
function Un(e) {
  return (t, a) => {
    let r = a?.context ? String(a.context) : 'standalone',
      o;
    if (r === 'formatting' && e.formattingValues) {
      let l = e.defaultFormattingWidth || e.defaultWidth,
        i = a?.width ? String(a.width) : l;
      o = e.formattingValues[i] || e.formattingValues[l];
    } else {
      let l = e.defaultWidth,
        i = a?.width ? String(a.width) : e.defaultWidth;
      o = e.values[i] || e.values[l];
    }
    let n = e.argumentCallback ? e.argumentCallback(t) : t;
    return o[n];
  };
}
var zI = { narrow: ['B', 'A'], abbreviated: ['BC', 'AD'], wide: ['Before Christ', 'Anno Domini'] },
  FI = {
    narrow: ['1', '2', '3', '4'],
    abbreviated: ['Q1', 'Q2', 'Q3', 'Q4'],
    wide: ['1st quarter', '2nd quarter', '3rd quarter', '4th quarter'],
  },
  qI = {
    narrow: ['J', 'F', 'M', 'A', 'M', 'J', 'J', 'A', 'S', 'O', 'N', 'D'],
    abbreviated: [
      'Jan',
      'Feb',
      'Mar',
      'Apr',
      'May',
      'Jun',
      'Jul',
      'Aug',
      'Sep',
      'Oct',
      'Nov',
      'Dec',
    ],
    wide: [
      'January',
      'February',
      'March',
      'April',
      'May',
      'June',
      'July',
      'August',
      'September',
      'October',
      'November',
      'December',
    ],
  },
  GI = {
    narrow: ['S', 'M', 'T', 'W', 'T', 'F', 'S'],
    short: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'],
    abbreviated: ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'],
    wide: ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'],
  },
  VI = {
    narrow: {
      am: 'a',
      pm: 'p',
      midnight: 'mi',
      noon: 'n',
      morning: 'morning',
      afternoon: 'afternoon',
      evening: 'evening',
      night: 'night',
    },
    abbreviated: {
      am: 'AM',
      pm: 'PM',
      midnight: 'midnight',
      noon: 'noon',
      morning: 'morning',
      afternoon: 'afternoon',
      evening: 'evening',
      night: 'night',
    },
    wide: {
      am: 'a.m.',
      pm: 'p.m.',
      midnight: 'midnight',
      noon: 'noon',
      morning: 'morning',
      afternoon: 'afternoon',
      evening: 'evening',
      night: 'night',
    },
  },
  jI = {
    narrow: {
      am: 'a',
      pm: 'p',
      midnight: 'mi',
      noon: 'n',
      morning: 'in the morning',
      afternoon: 'in the afternoon',
      evening: 'in the evening',
      night: 'at night',
    },
    abbreviated: {
      am: 'AM',
      pm: 'PM',
      midnight: 'midnight',
      noon: 'noon',
      morning: 'in the morning',
      afternoon: 'in the afternoon',
      evening: 'in the evening',
      night: 'at night',
    },
    wide: {
      am: 'a.m.',
      pm: 'p.m.',
      midnight: 'midnight',
      noon: 'noon',
      morning: 'in the morning',
      afternoon: 'in the afternoon',
      evening: 'in the evening',
      night: 'at night',
    },
  },
  YI = (e, t) => {
    let a = Number(e),
      r = a % 100;
    if (r > 20 || r < 10)
      switch (r % 10) {
        case 1:
          return a + 'st';
        case 2:
          return a + 'nd';
        case 3:
          return a + 'rd';
      }
    return a + 'th';
  },
  sS = {
    ordinalNumber: YI,
    era: Un({ values: zI, defaultWidth: 'wide' }),
    quarter: Un({ values: FI, defaultWidth: 'wide', argumentCallback: (e) => e - 1 }),
    month: Un({ values: qI, defaultWidth: 'wide' }),
    day: Un({ values: GI, defaultWidth: 'wide' }),
    dayPeriod: Un({
      values: VI,
      defaultWidth: 'wide',
      formattingValues: jI,
      defaultFormattingWidth: 'wide',
    }),
  };
function Nn(e) {
  return (t, a = {}) => {
    let r = a.width,
      o = (r && e.matchPatterns[r]) || e.matchPatterns[e.defaultMatchWidth],
      n = t.match(o);
    if (!n) return null;
    let l = n[0],
      i = (r && e.parsePatterns[r]) || e.parsePatterns[e.defaultParseWidth],
      s = Array.isArray(i) ? WI(i, (c) => c.test(l)) : XI(i, (c) => c.test(l)),
      u;
    (u = e.valueCallback ? e.valueCallback(s) : s), (u = a.valueCallback ? a.valueCallback(u) : u);
    let d = t.slice(l.length);
    return { value: u, rest: d };
  };
}
function XI(e, t) {
  for (let a in e) if (Object.prototype.hasOwnProperty.call(e, a) && t(e[a])) return a;
}
function WI(e, t) {
  for (let a = 0; a < e.length; a++) if (t(e[a])) return a;
}
function uS(e) {
  return (t, a = {}) => {
    let r = t.match(e.matchPattern);
    if (!r) return null;
    let o = r[0],
      n = t.match(e.parsePattern);
    if (!n) return null;
    let l = e.valueCallback ? e.valueCallback(n[0]) : n[0];
    l = a.valueCallback ? a.valueCallback(l) : l;
    let i = t.slice(o.length);
    return { value: l, rest: i };
  };
}
var QI = /^(\d+)(th|st|nd|rd)?/i,
  KI = /\d+/i,
  ZI = {
    narrow: /^(b|a)/i,
    abbreviated: /^(b\.?\s?c\.?|b\.?\s?c\.?\s?e\.?|a\.?\s?d\.?|c\.?\s?e\.?)/i,
    wide: /^(before christ|before common era|anno domini|common era)/i,
  },
  JI = { any: [/^b/i, /^(a|c)/i] },
  $I = { narrow: /^[1234]/i, abbreviated: /^q[1234]/i, wide: /^[1234](th|st|nd|rd)? quarter/i },
  ek = { any: [/1/i, /2/i, /3/i, /4/i] },
  tk = {
    narrow: /^[jfmasond]/i,
    abbreviated: /^(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)/i,
    wide: /^(january|february|march|april|may|june|july|august|september|october|november|december)/i,
  },
  ak = {
    narrow: [/^j/i, /^f/i, /^m/i, /^a/i, /^m/i, /^j/i, /^j/i, /^a/i, /^s/i, /^o/i, /^n/i, /^d/i],
    any: [
      /^ja/i,
      /^f/i,
      /^mar/i,
      /^ap/i,
      /^may/i,
      /^jun/i,
      /^jul/i,
      /^au/i,
      /^s/i,
      /^o/i,
      /^n/i,
      /^d/i,
    ],
  },
  rk = {
    narrow: /^[smtwf]/i,
    short: /^(su|mo|tu|we|th|fr|sa)/i,
    abbreviated: /^(sun|mon|tue|wed|thu|fri|sat)/i,
    wide: /^(sunday|monday|tuesday|wednesday|thursday|friday|saturday)/i,
  },
  ok = {
    narrow: [/^s/i, /^m/i, /^t/i, /^w/i, /^t/i, /^f/i, /^s/i],
    any: [/^su/i, /^m/i, /^tu/i, /^w/i, /^th/i, /^f/i, /^sa/i],
  },
  nk = {
    narrow: /^(a|p|mi|n|(in the|at) (morning|afternoon|evening|night))/i,
    any: /^([ap]\.?\s?m\.?|midnight|noon|(in the|at) (morning|afternoon|evening|night))/i,
  },
  lk = {
    any: {
      am: /^a/i,
      pm: /^p/i,
      midnight: /^mi/i,
      noon: /^no/i,
      morning: /morning/i,
      afternoon: /afternoon/i,
      evening: /evening/i,
      night: /night/i,
    },
  },
  cS = {
    ordinalNumber: uS({
      matchPattern: QI,
      parsePattern: KI,
      valueCallback: (e) => parseInt(e, 10),
    }),
    era: Nn({
      matchPatterns: ZI,
      defaultMatchWidth: 'wide',
      parsePatterns: JI,
      defaultParseWidth: 'any',
    }),
    quarter: Nn({
      matchPatterns: $I,
      defaultMatchWidth: 'wide',
      parsePatterns: ek,
      defaultParseWidth: 'any',
      valueCallback: (e) => e + 1,
    }),
    month: Nn({
      matchPatterns: tk,
      defaultMatchWidth: 'wide',
      parsePatterns: ak,
      defaultParseWidth: 'any',
    }),
    day: Nn({
      matchPatterns: rk,
      defaultMatchWidth: 'wide',
      parsePatterns: ok,
      defaultParseWidth: 'any',
    }),
    dayPeriod: Nn({
      matchPatterns: nk,
      defaultMatchWidth: 'any',
      parsePatterns: lk,
      defaultParseWidth: 'any',
    }),
  };
var Pm = {
  code: 'en-US',
  formatDistance: nS,
  formatLong: lS,
  formatRelative: iS,
  localize: sS,
  match: cS,
  options: { weekStartsOn: 0, firstWeekContainsDate: 1 },
};
function dS(e, t) {
  let a = pe(e, t?.in);
  return Jx(a, oS(a)) + 1;
}
function fS(e, t) {
  let a = pe(e, t?.in),
    r = +vo(a) - +$x(a);
  return Math.round(r / _u) + 1;
}
function Eu(e, t) {
  let a = pe(e, t?.in),
    r = a.getFullYear(),
    o = Ur(),
    n =
      t?.firstWeekContainsDate ??
      t?.locale?.options?.firstWeekContainsDate ??
      o.firstWeekContainsDate ??
      o.locale?.options?.firstWeekContainsDate ??
      1,
    l = ut(t?.in || e, 0);
  l.setFullYear(r + 1, 0, n), l.setHours(0, 0, 0, 0);
  let i = Za(l, t),
    s = ut(t?.in || e, 0);
  s.setFullYear(r, 0, n), s.setHours(0, 0, 0, 0);
  let u = Za(s, t);
  return +a >= +i ? r + 1 : +a >= +u ? r : r - 1;
}
function mS(e, t) {
  let a = Ur(),
    r =
      t?.firstWeekContainsDate ??
      t?.locale?.options?.firstWeekContainsDate ??
      a.firstWeekContainsDate ??
      a.locale?.options?.firstWeekContainsDate ??
      1,
    o = Eu(e, t),
    n = ut(t?.in || e, 0);
  return n.setFullYear(o, 0, r), n.setHours(0, 0, 0, 0), Za(n, t);
}
function pS(e, t) {
  let a = pe(e, t?.in),
    r = +Za(a, t) - +mS(a, t);
  return Math.round(r / _u) + 1;
}
function ie(e, t) {
  let a = e < 0 ? '-' : '',
    r = Math.abs(e).toString().padStart(t, '0');
  return a + r;
}
var Ja = {
  y(e, t) {
    let a = e.getFullYear(),
      r = a > 0 ? a : 1 - a;
    return ie(t === 'yy' ? r % 100 : r, t.length);
  },
  M(e, t) {
    let a = e.getMonth();
    return t === 'M' ? String(a + 1) : ie(a + 1, 2);
  },
  d(e, t) {
    return ie(e.getDate(), t.length);
  },
  a(e, t) {
    let a = e.getHours() / 12 >= 1 ? 'pm' : 'am';
    switch (t) {
      case 'a':
      case 'aa':
        return a.toUpperCase();
      case 'aaa':
        return a;
      case 'aaaaa':
        return a[0];
      case 'aaaa':
      default:
        return a === 'am' ? 'a.m.' : 'p.m.';
    }
  },
  h(e, t) {
    return ie(e.getHours() % 12 || 12, t.length);
  },
  H(e, t) {
    return ie(e.getHours(), t.length);
  },
  m(e, t) {
    return ie(e.getMinutes(), t.length);
  },
  s(e, t) {
    return ie(e.getSeconds(), t.length);
  },
  S(e, t) {
    let a = t.length,
      r = e.getMilliseconds(),
      o = Math.trunc(r * Math.pow(10, a - 3));
    return ie(o, t.length);
  },
};
var Hn = {
    am: 'am',
    pm: 'pm',
    midnight: 'midnight',
    noon: 'noon',
    morning: 'morning',
    afternoon: 'afternoon',
    evening: 'evening',
    night: 'night',
  },
  Bm = {
    G: function (e, t, a) {
      let r = e.getFullYear() > 0 ? 1 : 0;
      switch (t) {
        case 'G':
        case 'GG':
        case 'GGG':
          return a.era(r, { width: 'abbreviated' });
        case 'GGGGG':
          return a.era(r, { width: 'narrow' });
        case 'GGGG':
        default:
          return a.era(r, { width: 'wide' });
      }
    },
    y: function (e, t, a) {
      if (t === 'yo') {
        let r = e.getFullYear(),
          o = r > 0 ? r : 1 - r;
        return a.ordinalNumber(o, { unit: 'year' });
      }
      return Ja.y(e, t);
    },
    Y: function (e, t, a, r) {
      let o = Eu(e, r),
        n = o > 0 ? o : 1 - o;
      if (t === 'YY') {
        let l = n % 100;
        return ie(l, 2);
      }
      return t === 'Yo' ? a.ordinalNumber(n, { unit: 'year' }) : ie(n, t.length);
    },
    R: function (e, t) {
      let a = Iu(e);
      return ie(a, t.length);
    },
    u: function (e, t) {
      let a = e.getFullYear();
      return ie(a, t.length);
    },
    Q: function (e, t, a) {
      let r = Math.ceil((e.getMonth() + 1) / 3);
      switch (t) {
        case 'Q':
          return String(r);
        case 'QQ':
          return ie(r, 2);
        case 'Qo':
          return a.ordinalNumber(r, { unit: 'quarter' });
        case 'QQQ':
          return a.quarter(r, { width: 'abbreviated', context: 'formatting' });
        case 'QQQQQ':
          return a.quarter(r, { width: 'narrow', context: 'formatting' });
        case 'QQQQ':
        default:
          return a.quarter(r, { width: 'wide', context: 'formatting' });
      }
    },
    q: function (e, t, a) {
      let r = Math.ceil((e.getMonth() + 1) / 3);
      switch (t) {
        case 'q':
          return String(r);
        case 'qq':
          return ie(r, 2);
        case 'qo':
          return a.ordinalNumber(r, { unit: 'quarter' });
        case 'qqq':
          return a.quarter(r, { width: 'abbreviated', context: 'standalone' });
        case 'qqqqq':
          return a.quarter(r, { width: 'narrow', context: 'standalone' });
        case 'qqqq':
        default:
          return a.quarter(r, { width: 'wide', context: 'standalone' });
      }
    },
    M: function (e, t, a) {
      let r = e.getMonth();
      switch (t) {
        case 'M':
        case 'MM':
          return Ja.M(e, t);
        case 'Mo':
          return a.ordinalNumber(r + 1, { unit: 'month' });
        case 'MMM':
          return a.month(r, { width: 'abbreviated', context: 'formatting' });
        case 'MMMMM':
          return a.month(r, { width: 'narrow', context: 'formatting' });
        case 'MMMM':
        default:
          return a.month(r, { width: 'wide', context: 'formatting' });
      }
    },
    L: function (e, t, a) {
      let r = e.getMonth();
      switch (t) {
        case 'L':
          return String(r + 1);
        case 'LL':
          return ie(r + 1, 2);
        case 'Lo':
          return a.ordinalNumber(r + 1, { unit: 'month' });
        case 'LLL':
          return a.month(r, { width: 'abbreviated', context: 'standalone' });
        case 'LLLLL':
          return a.month(r, { width: 'narrow', context: 'standalone' });
        case 'LLLL':
        default:
          return a.month(r, { width: 'wide', context: 'standalone' });
      }
    },
    w: function (e, t, a, r) {
      let o = pS(e, r);
      return t === 'wo' ? a.ordinalNumber(o, { unit: 'week' }) : ie(o, t.length);
    },
    I: function (e, t, a) {
      let r = fS(e);
      return t === 'Io' ? a.ordinalNumber(r, { unit: 'week' }) : ie(r, t.length);
    },
    d: function (e, t, a) {
      return t === 'do' ? a.ordinalNumber(e.getDate(), { unit: 'date' }) : Ja.d(e, t);
    },
    D: function (e, t, a) {
      let r = dS(e);
      return t === 'Do' ? a.ordinalNumber(r, { unit: 'dayOfYear' }) : ie(r, t.length);
    },
    E: function (e, t, a) {
      let r = e.getDay();
      switch (t) {
        case 'E':
        case 'EE':
        case 'EEE':
          return a.day(r, { width: 'abbreviated', context: 'formatting' });
        case 'EEEEE':
          return a.day(r, { width: 'narrow', context: 'formatting' });
        case 'EEEEEE':
          return a.day(r, { width: 'short', context: 'formatting' });
        case 'EEEE':
        default:
          return a.day(r, { width: 'wide', context: 'formatting' });
      }
    },
    e: function (e, t, a, r) {
      let o = e.getDay(),
        n = (o - r.weekStartsOn + 8) % 7 || 7;
      switch (t) {
        case 'e':
          return String(n);
        case 'ee':
          return ie(n, 2);
        case 'eo':
          return a.ordinalNumber(n, { unit: 'day' });
        case 'eee':
          return a.day(o, { width: 'abbreviated', context: 'formatting' });
        case 'eeeee':
          return a.day(o, { width: 'narrow', context: 'formatting' });
        case 'eeeeee':
          return a.day(o, { width: 'short', context: 'formatting' });
        case 'eeee':
        default:
          return a.day(o, { width: 'wide', context: 'formatting' });
      }
    },
    c: function (e, t, a, r) {
      let o = e.getDay(),
        n = (o - r.weekStartsOn + 8) % 7 || 7;
      switch (t) {
        case 'c':
          return String(n);
        case 'cc':
          return ie(n, t.length);
        case 'co':
          return a.ordinalNumber(n, { unit: 'day' });
        case 'ccc':
          return a.day(o, { width: 'abbreviated', context: 'standalone' });
        case 'ccccc':
          return a.day(o, { width: 'narrow', context: 'standalone' });
        case 'cccccc':
          return a.day(o, { width: 'short', context: 'standalone' });
        case 'cccc':
        default:
          return a.day(o, { width: 'wide', context: 'standalone' });
      }
    },
    i: function (e, t, a) {
      let r = e.getDay(),
        o = r === 0 ? 7 : r;
      switch (t) {
        case 'i':
          return String(o);
        case 'ii':
          return ie(o, t.length);
        case 'io':
          return a.ordinalNumber(o, { unit: 'day' });
        case 'iii':
          return a.day(r, { width: 'abbreviated', context: 'formatting' });
        case 'iiiii':
          return a.day(r, { width: 'narrow', context: 'formatting' });
        case 'iiiiii':
          return a.day(r, { width: 'short', context: 'formatting' });
        case 'iiii':
        default:
          return a.day(r, { width: 'wide', context: 'formatting' });
      }
    },
    a: function (e, t, a) {
      let o = e.getHours() / 12 >= 1 ? 'pm' : 'am';
      switch (t) {
        case 'a':
        case 'aa':
          return a.dayPeriod(o, { width: 'abbreviated', context: 'formatting' });
        case 'aaa':
          return a.dayPeriod(o, { width: 'abbreviated', context: 'formatting' }).toLowerCase();
        case 'aaaaa':
          return a.dayPeriod(o, { width: 'narrow', context: 'formatting' });
        case 'aaaa':
        default:
          return a.dayPeriod(o, { width: 'wide', context: 'formatting' });
      }
    },
    b: function (e, t, a) {
      let r = e.getHours(),
        o;
      switch (
        (r === 12 ? (o = Hn.noon) : r === 0 ? (o = Hn.midnight) : (o = r / 12 >= 1 ? 'pm' : 'am'),
        t)
      ) {
        case 'b':
        case 'bb':
          return a.dayPeriod(o, { width: 'abbreviated', context: 'formatting' });
        case 'bbb':
          return a.dayPeriod(o, { width: 'abbreviated', context: 'formatting' }).toLowerCase();
        case 'bbbbb':
          return a.dayPeriod(o, { width: 'narrow', context: 'formatting' });
        case 'bbbb':
        default:
          return a.dayPeriod(o, { width: 'wide', context: 'formatting' });
      }
    },
    B: function (e, t, a) {
      let r = e.getHours(),
        o;
      switch (
        (r >= 17
          ? (o = Hn.evening)
          : r >= 12
            ? (o = Hn.afternoon)
            : r >= 4
              ? (o = Hn.morning)
              : (o = Hn.night),
        t)
      ) {
        case 'B':
        case 'BB':
        case 'BBB':
          return a.dayPeriod(o, { width: 'abbreviated', context: 'formatting' });
        case 'BBBBB':
          return a.dayPeriod(o, { width: 'narrow', context: 'formatting' });
        case 'BBBB':
        default:
          return a.dayPeriod(o, { width: 'wide', context: 'formatting' });
      }
    },
    h: function (e, t, a) {
      if (t === 'ho') {
        let r = e.getHours() % 12;
        return r === 0 && (r = 12), a.ordinalNumber(r, { unit: 'hour' });
      }
      return Ja.h(e, t);
    },
    H: function (e, t, a) {
      return t === 'Ho' ? a.ordinalNumber(e.getHours(), { unit: 'hour' }) : Ja.H(e, t);
    },
    K: function (e, t, a) {
      let r = e.getHours() % 12;
      return t === 'Ko' ? a.ordinalNumber(r, { unit: 'hour' }) : ie(r, t.length);
    },
    k: function (e, t, a) {
      let r = e.getHours();
      return (
        r === 0 && (r = 24), t === 'ko' ? a.ordinalNumber(r, { unit: 'hour' }) : ie(r, t.length)
      );
    },
    m: function (e, t, a) {
      return t === 'mo' ? a.ordinalNumber(e.getMinutes(), { unit: 'minute' }) : Ja.m(e, t);
    },
    s: function (e, t, a) {
      return t === 'so' ? a.ordinalNumber(e.getSeconds(), { unit: 'second' }) : Ja.s(e, t);
    },
    S: function (e, t) {
      return Ja.S(e, t);
    },
    X: function (e, t, a) {
      let r = e.getTimezoneOffset();
      if (r === 0) return 'Z';
      switch (t) {
        case 'X':
          return gS(r);
        case 'XXXX':
        case 'XX':
          return bo(r);
        case 'XXXXX':
        case 'XXX':
        default:
          return bo(r, ':');
      }
    },
    x: function (e, t, a) {
      let r = e.getTimezoneOffset();
      switch (t) {
        case 'x':
          return gS(r);
        case 'xxxx':
        case 'xx':
          return bo(r);
        case 'xxxxx':
        case 'xxx':
        default:
          return bo(r, ':');
      }
    },
    O: function (e, t, a) {
      let r = e.getTimezoneOffset();
      switch (t) {
        case 'O':
        case 'OO':
        case 'OOO':
          return 'GMT' + hS(r, ':');
        case 'OOOO':
        default:
          return 'GMT' + bo(r, ':');
      }
    },
    z: function (e, t, a) {
      let r = e.getTimezoneOffset();
      switch (t) {
        case 'z':
        case 'zz':
        case 'zzz':
          return 'GMT' + hS(r, ':');
        case 'zzzz':
        default:
          return 'GMT' + bo(r, ':');
      }
    },
    t: function (e, t, a) {
      let r = Math.trunc(+e / 1e3);
      return ie(r, t.length);
    },
    T: function (e, t, a) {
      return ie(+e, t.length);
    },
  };
function hS(e, t = '') {
  let a = e > 0 ? '-' : '+',
    r = Math.abs(e),
    o = Math.trunc(r / 60),
    n = r % 60;
  return n === 0 ? a + String(o) : a + String(o) + t + ie(n, 2);
}
function gS(e, t) {
  return e % 60 === 0 ? (e > 0 ? '-' : '+') + ie(Math.abs(e) / 60, 2) : bo(e, t);
}
function bo(e, t = '') {
  let a = e > 0 ? '-' : '+',
    r = Math.abs(e),
    o = ie(Math.trunc(r / 60), 2),
    n = ie(r % 60, 2);
  return a + o + t + n;
}
var yS = (e, t) => {
    switch (e) {
      case 'P':
        return t.date({ width: 'short' });
      case 'PP':
        return t.date({ width: 'medium' });
      case 'PPP':
        return t.date({ width: 'long' });
      case 'PPPP':
      default:
        return t.date({ width: 'full' });
    }
  },
  vS = (e, t) => {
    switch (e) {
      case 'p':
        return t.time({ width: 'short' });
      case 'pp':
        return t.time({ width: 'medium' });
      case 'ppp':
        return t.time({ width: 'long' });
      case 'pppp':
      default:
        return t.time({ width: 'full' });
    }
  },
  ik = (e, t) => {
    let a = e.match(/(P+)(p+)?/) || [],
      r = a[1],
      o = a[2];
    if (!o) return yS(e, t);
    let n;
    switch (r) {
      case 'P':
        n = t.dateTime({ width: 'short' });
        break;
      case 'PP':
        n = t.dateTime({ width: 'medium' });
        break;
      case 'PPP':
        n = t.dateTime({ width: 'long' });
        break;
      case 'PPPP':
      default:
        n = t.dateTime({ width: 'full' });
        break;
    }
    return n.replace('{{date}}', yS(r, t)).replace('{{time}}', vS(o, t));
  },
  bS = { p: vS, P: ik };
var sk = /^D+$/,
  uk = /^Y+$/,
  ck = ['D', 'DD', 'YY', 'YYYY'];
function xS(e) {
  return sk.test(e);
}
function SS(e) {
  return uk.test(e);
}
function LS(e, t, a) {
  let r = dk(e, t, a);
  if ((console.warn(r), ck.includes(e))) throw new RangeError(r);
}
function dk(e, t, a) {
  let r = e[0] === 'Y' ? 'years' : 'days of the month';
  return `Use \`${e.toLowerCase()}\` instead of \`${e}\` (in \`${t}\`) for formatting ${r} to the input \`${a}\`; see: https://github.com/date-fns/date-fns/blob/master/docs/unicodeTokens.md`;
}
var fk = /[yYQqMLwIdDecihHKkms]o|(\w)\1*|''|'(''|[^'])+('|$)|./g,
  mk = /P+p+|P+|p+|''|'(''|[^'])+('|$)|./g,
  pk = /^'([^]*?)'?$/,
  hk = /''/g,
  gk = /[a-zA-Z]/;
function CS(e, t, a) {
  let r = Ur(),
    o = a?.locale ?? r.locale ?? Pm,
    n =
      a?.firstWeekContainsDate ??
      a?.locale?.options?.firstWeekContainsDate ??
      r.firstWeekContainsDate ??
      r.locale?.options?.firstWeekContainsDate ??
      1,
    l =
      a?.weekStartsOn ??
      a?.locale?.options?.weekStartsOn ??
      r.weekStartsOn ??
      r.locale?.options?.weekStartsOn ??
      0,
    i = pe(e, a?.in);
  if (!tS(i)) throw new RangeError('Invalid time value');
  let s = t
    .match(mk)
    .map((d) => {
      let c = d[0];
      if (c === 'p' || c === 'P') {
        let f = bS[c];
        return f(d, o.formatLong);
      }
      return d;
    })
    .join('')
    .match(fk)
    .map((d) => {
      if (d === "''") return { isToken: !1, value: "'" };
      let c = d[0];
      if (c === "'") return { isToken: !1, value: yk(d) };
      if (Bm[c]) return { isToken: !0, value: d };
      if (c.match(gk))
        throw new RangeError(
          'Format string contains an unescaped latin alphabet character `' + c + '`'
        );
      return { isToken: !1, value: d };
    });
  o.localize.preprocessor && (s = o.localize.preprocessor(i, s));
  let u = { firstWeekContainsDate: n, weekStartsOn: l, locale: o };
  return s
    .map((d) => {
      if (!d.isToken) return d.value;
      let c = d.value;
      ((!a?.useAdditionalWeekYearTokens && SS(c)) || (!a?.useAdditionalDayOfYearTokens && xS(c))) &&
        LS(c, t, String(e));
      let f = Bm[c[0]];
      return f(i, c, o.localize, u);
    })
    .join('');
}
function yk(e) {
  let t = e.match(pk);
  return t ? t[1].replace(hk, "'") : e;
}
function wS(e, t) {
  return pe(e, t?.in).getDate();
}
function RS(e, t) {
  return pe(e, t?.in).getDay();
}
function _S(e, t) {
  return pe(e, t?.in).getMonth();
}
var vk = 1e6,
  TS = new Date(2026, 6, 1),
  DS = new Date(2026, 8, 1),
  bk = [0.72, 1.04, 1.08, 1.05, 1, 0.88, 0.69],
  IS = [0.31, 0.19, 0.16, 0.12, 0.11, 0.07, 0.04],
  xk = [0.08, 0.04, -0.06, 0.02, 0.11, -0.12, -0.18],
  Um = ci()
    .slice(0, IS.length)
    .map((e, t) => ({ id: e.id, name: e.name, share: IS[t], roiSkew: xk[t] })),
  Sk = [0.28, 0.22, 0.18, 0.14, 0.11, 0.07],
  Lk = Sk.map((e, t) => ({ id: de(si, t + 1), name: Lu(t + 1), share: e })),
  Ck = [0.26, 0.21, 0.17, 0.15, 0.12, 0.09],
  wk = Ck.map((e, t) => ({ id: U('offer', t + 1), name: de(bu, t + 1), share: e })),
  Rk = [0.34, 0.22, 0.18, 0.14, 0.07, 0.05],
  _k = Rk.map((e, t) => ({ id: U('traffic_source', t + 1), name: de(xu, t + 1), share: e })),
  kS = ['US', 'DE', 'BR', 'PL', 'GB', 'CA', 'IN', 'AU'],
  MS = [
    'facebook.com',
    'push_subscribers',
    'google_ads',
    'tiktok_ads',
    'taboola',
    'outbrain',
    'mgid',
    'revcontent',
  ],
  ES = ['lead', 'sale', 'install', 'signup', 'deposit'];
function gi(e) {
  return Math.round(e * vk);
}
function We(e) {
  let t = Math.imul(e ^ (e >>> 16), 2146121005);
  return (t = Math.imul(t ^ (t >>> 15), 2221713035)), (t ^= t >>> 16), (t >>> 0) / 4294967295;
}
function Ik(e) {
  let t = _S(e),
    a = wS(e);
  return t === 6 && a === 4
    ? 0.76
    : t === 6 && a === 3
      ? 0.9
      : t === 7 && a >= 20 && a <= 27
        ? 0.93
        : a === 1 || a === 15
          ? 1.08
          : 1;
}
function kk(e, t, a) {
  let r = RS(e),
    o = bk[r] ?? 1;
  o *= Ik(e);
  let n = 0.9 + (t / Math.max(a - 1, 1)) * 0.22,
    l = 0.9 + We(e.getTime() >>> 0) * 0.2;
  return o * n * l;
}
function OS() {
  if (typeof window > 'u') return !1;
  let e = new URLSearchParams(window.location.search);
  return e.get('chart_mock') === '0' ? !1 : e.get('chart_mock') === '1';
}
function Mk(e) {
  let a = (e.series ?? []).some(
      (l) =>
        (l.clicks ?? 0) > 0 ||
        (l.conversions ?? 0) > 0 ||
        (l.spend_micro ?? l.spend_micros ?? 0) > 0 ||
        (l.revenue_micro ?? 0) > 0
    ),
    r =
      (e.kpis?.conversions ?? 0) > 0 ||
      (e.kpis?.cost_micro ?? e.kpis?.spend_micro ?? 0) > 0 ||
      (e.clicks_7d ?? 0) > 0,
    o =
      (e.breakdowns?.campaigns?.rows?.length ?? 0) > 0 ||
      (e.breakdowns?.landers?.rows?.length ?? 0) > 0 ||
      (e.breakdowns?.offers?.rows?.length ?? 0) > 0 ||
      (e.breakdowns?.sources?.rows?.length ?? 0) > 0,
    n = (e.recent_clicks?.length ?? 0) > 0;
  return !a && !r && !o && !n;
}
function Ek() {
  if (typeof window > 'u') return !1;
  let { hostname: e, port: t } = window.location;
  return e === 'localhost' || e === '127.0.0.1' ? !0 : t === '5173';
}
function Ak(e) {
  return OS() ? !0 : Ek() && Mk(e);
}
function PS(e = TS, t = DS) {
  let a = rS({ start: e, end: t }),
    r = a.length;
  return a.map((o, n) => {
    let l = kk(o, n, r),
      i = 38600 + n * 142,
      s = Math.max(120, Math.round(i * l + (We(n * 17 + 3) > 0.96 ? 6200 : 0))),
      u = 0.11 + We(n * 7 + 2) * 0.14,
      d = 0.0034 + We(n * 9 + 5) * 0.0028,
      c = Math.max(1, Math.round(s * d)),
      f = s * u * (0.96 + We(n * 13 + 1) * 0.08),
      h = 19 + We(n * 11 + 7) * 31,
      v = c * h * (0.92 + We(n * 19 + 4) * 0.16) + (We(n * 23) > 0.94 ? 180 : 0),
      x = v - f;
    return {
      label: CS(o, 'yyyy-MM-dd'),
      clicks: s,
      conversions: c,
      spend_micro: gi(f),
      revenue_micro: gi(v),
      profit_micro: gi(x),
    };
  });
}
function Tk(e) {
  let t = e.reduce((s, u) => s + (u.clicks ?? 0), 0),
    a = e.reduce((s, u) => s + (u.conversions ?? 0), 0),
    r = e.reduce((s, u) => s + (u.spend_micro ?? 0), 0),
    o = e.reduce((s, u) => s + (u.revenue_micro ?? 0), 0),
    n = o - r,
    l = Math.round(t * (0.857 + We(t) * 0.04)),
    i = r > 0 ? (n / r) * 100 : 0;
  return {
    clicks: t,
    unique_clicks: l,
    conversions: a,
    cost_micro: r,
    revenue_micro: o,
    profit_micro: n,
    roi_pct: i,
  };
}
function BS(e, t) {
  return t <= 0 || e <= 0 ? 0 : Math.round(e / t);
}
function US(e, t) {
  return t <= 0 || e <= 0 ? 0 : Math.round(e / t);
}
function NS(e, t) {
  return t <= 0 || e <= 0 ? 0 : Math.round(e / t);
}
function HS(e, t) {
  return t <= 0 || e <= 0 ? 0 : (e / t) * 100;
}
function zS(e) {
  let t = e.profit_micro ?? (e.revenue_micro ?? 0) - (e.cost_micro ?? 0),
    a = e.clicks ?? 0,
    r = e.conversions ?? 0,
    o = e.cost_micro ?? 0,
    n = e.revenue_micro ?? 0;
  return {
    ...e,
    profit_micro: t,
    roi_pct: o > 0 ? (t / o) * 100 : 0,
    cpc_micro: BS(o, a),
    cpa_micro: US(o, r),
    cr_pct: HS(r, a),
    epc_micro: NS(n, a),
  };
}
function Dk(e) {
  return zS(e);
}
function Ok(e, t) {
  return e.map((a, r) => {
    let o = 0.94 + We(r * 23 + 1) * 0.12,
      n = a.share * o,
      l = Math.max(0, Math.round(t.clicks * n)),
      i = Math.max(0, Math.round(l * (0.82 + We(r + 4) * 0.1))),
      s = Math.max(0, Math.round(l * (0.0038 + We(r + 9) * 0.0024))),
      u = Math.max(0, Math.round(t.cost_micro * n)),
      d = 1 + (a.roiSkew ?? 0) + (We(r + 15) - 0.5) * 0.08,
      c = Math.max(0, Math.round(t.revenue_micro * n * d)),
      f = c - u,
      h = u > 0 ? (f / u) * 100 : 0;
    return zS({
      id: a.id,
      name: a.name,
      clicks: l,
      unique_clicks: i,
      conversions: s,
      cost_micro: u,
      revenue_micro: c,
      profit_micro: f,
      roi_pct: h,
    });
  });
}
function Pk(e) {
  let t = e.reduce(
      (n, l) => ({
        clicks: (n.clicks ?? 0) + (l.clicks ?? 0),
        unique_clicks: (n.unique_clicks ?? 0) + (l.unique_clicks ?? 0),
        conversions: (n.conversions ?? 0) + (l.conversions ?? 0),
        cost_micro: (n.cost_micro ?? 0) + (l.cost_micro ?? 0),
        revenue_micro: (n.revenue_micro ?? 0) + (l.revenue_micro ?? 0),
        profit_micro: (n.profit_micro ?? 0) + (l.profit_micro ?? 0),
      }),
      {
        clicks: 0,
        unique_clicks: 0,
        conversions: 0,
        cost_micro: 0,
        revenue_micro: 0,
        profit_micro: 0,
      }
    ),
    a = t.cost_micro ?? 0,
    r = t.profit_micro ?? 0,
    o = a > 0 ? (r / a) * 100 : 0;
  return Dk({ ...t, roi_pct: o });
}
function Au(e, t) {
  let a = Ok(e, t);
  return { rows: a, totals: Pk(a), truncated: !1, total: a.length };
}
function Bk(e) {
  let t = [];
  for (let a = 0; a < 10; a += 1) {
    let r = Math.round(a * 53 + We(a * 31) * 140),
      o = new Date(e.getTime() - r * 36e5),
      n = Um[a % Um.length],
      l = We(a + 8) > 0.28;
    t.push({
      event_type: 'click',
      click_id: Tn(a + 41),
      campaign_id: n.id,
      placement_id: Dn(a + 3),
      created_at: o.toISOString(),
      country: kS[a % kS.length],
      sub1: MS[a % MS.length],
      goal_name: ES[a % ES.length],
      attributed_cost_micro: gi(0.09 + We(a + 2) * 0.19),
      revenue_micro: l ? gi(12 + We(a + 19) * 38) : 0,
      inbound_status: 'accepted',
    });
  }
  return t;
}
function AS(e) {
  if (!e) return;
  let t = new Date(e);
  if (!Number.isNaN(t.getTime())) return t;
}
function Uk(e) {
  let t = AS(e.period?.from) ?? TS,
    a = AS(e.period?.to) ?? DS;
  return t <= a ? { from: t, to: a } : { from: a, to: t };
}
function yi(e) {
  let { from: t, to: a } = Uk(e),
    r = PS(t, a),
    o = Tk(r);
  return {
    ...e,
    period: { ...e.period, from: t.toISOString(), to: a.toISOString() },
    clicks_7d: o.clicks,
    unique_clicks_7d: o.unique_clicks,
    impressions_7d: Math.round(o.clicks * (1.12 + We(o.clicks) * 0.08)),
    kpis: {
      ...e.kpis,
      conversions: o.conversions,
      unique_clicks: o.unique_clicks,
      cost_micro: o.cost_micro,
      spend_micro: o.cost_micro,
      revenue_micro: o.revenue_micro,
      profit_micro: o.profit_micro,
      roi_pct: o.roi_pct,
      cpc_micro: BS(o.cost_micro, o.clicks),
      cpa_micro: US(o.cost_micro, o.conversions),
      cr_pct: HS(o.conversions, o.clicks),
      epc_micro: NS(o.revenue_micro, o.clicks),
      freshness: { stale: !1, label: 'Synthetic preview' },
    },
    series: r,
    breakdowns: { campaigns: Au(Um, o), landers: Au(Lk, o), offers: Au(wk, o), sources: Au(_k, o) },
    recent_clicks: Bk(a),
  };
}
function aB(e) {
  return e && e.length > 0 ? e : OS() ? PS() : [];
}
function rB(e) {
  return Ak(e) ? yi(e) : e;
}
function Tu() {
  return { stale: !1, label: 'Synthetic preview' };
}
function FS(e, t) {
  let a = t.slice(19).split('/')[0],
    r = e.searchParams.get('customer_id')?.trim(),
    o = e.searchParams.get('from') ?? O(7),
    n = e.searchParams.get('to') ?? O(0);
  if (a === 'buyer')
    return yi({
      customer_id: r,
      period: { from: o, to: n, timezone: 'UTC' },
      series: [],
      breakdowns: {
        campaigns: { rows: [] },
        landers: { rows: [] },
        offers: { rows: [] },
        sources: { rows: [] },
      },
      recent_clicks: [],
    });
  if (a === 'campaign') {
    let s = t.split('/').pop() ?? U('campaign', 1),
      u = oe().campaigns.find((d) => d.id === s);
    return yi({
      customer_id: u?.customer_id ?? r,
      period: { from: o, to: n },
      series: [],
      campaigns: u ? [{ id: s, name: u.name, status: u.status }] : void 0,
    });
  }
  let l = yi({ period: { from: o, to: n }, series: [] }).series,
    i = {
      spend_micro: ae(84200),
      revenue_micro: ae(112450),
      profit_micro: ae(28250),
      conversions: 1842,
      clicks: 286400,
      impressions: 142e4,
      freshness: Tu(),
    };
  return a === 'cfo' || a === 'accountant'
    ? {
        role: a,
        period: { from: o, to: n },
        totals: i,
        series: l,
        tables: {
          invoices: {
            rows: [
              { month: '2026-08', invoiced_micro: ae(42100), paid_micro: ae(39800) },
              { month: '2026-07', invoiced_micro: ae(38400), paid_micro: ae(38400) },
            ],
          },
        },
        freshness: Tu(),
      }
    : a === 'fraud'
      ? {
          role: a,
          period: { from: o, to: n },
          totals: { blocks: 12840, silent_rejects: 3420, ivt_rate_pct: 4.2, freshness: Tu() },
          series: l,
          breakdowns: {
            categories: {
              rows: [
                { id: 'bot', name: 'Automated bot', blocks: 6200 },
                { id: 'proxy', name: 'Proxy / VPN', blocks: 4100 },
                { id: 'datacenter', name: 'Datacenter ASN', blocks: 2540 },
              ],
            },
          },
        }
      : {
          role: a,
          period: { from: o, to: n },
          totals: i,
          series: l,
          services: [
            { id: 'tracker', name: 'Tracker', status: 'ok', detail: 'Within SLA' },
            { id: 'processor', name: 'Processor', status: 'ok', detail: 'Within SLA' },
          ],
          freshness: Tu(),
        };
}
var Nk = [
  {
    key: 'campaign-overview',
    title: 'Campaign overview',
    description: 'Spend, conversions, and margin by campaign.',
    category: 'campaigns',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'click-log',
    title: 'Click log',
    description: 'Searchable click and postback events.',
    category: 'traffic',
    default_range: '7d',
    export_formats: ['csv', 'ndjson'],
    license_gated: !1,
  },
  {
    key: 'clicks',
    title: 'Clicks',
    description: 'Aggregated click metrics.',
    category: 'traffic',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'geo-roi',
    title: 'GEO ROI',
    description: 'Country-level ROI and spend.',
    category: 'campaigns',
    default_range: '30d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'fraud-breakdown',
    title: 'Fraud breakdown',
    description: 'Blocks and silent rejects by category.',
    category: 'fraud',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'silent-reject-impression-funnel',
    title: 'Silent reject funnel',
    description: 'Impression to silent reject funnel.',
    category: 'fraud',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'rtb/overview',
    title: 'RTB overview',
    description: 'Auction volume and win rate.',
    category: 'rtb',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'rtb/no-bid-reasons',
    title: 'RTB no-bid reasons',
    description: 'Top no-bid reason codes.',
    category: 'rtb',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'traffic-sources',
    title: 'Traffic sources',
    description: 'Source quality and conversion rate.',
    category: 'traffic',
    default_range: '30d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'pacing-drift',
    title: 'Pacing drift',
    description: 'Budget pacing vs actual spend.',
    category: 'campaigns',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'postback-reconciliation',
    title: 'Postback reconciliation',
    description: 'Inbound vs outbound postback parity.',
    category: 'integrations',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'cost-sync-coverage',
    title: 'Cost sync coverage',
    description: 'Network cost sync completeness.',
    category: 'integrations',
    default_range: '30d',
    export_formats: ['csv'],
    license_gated: !1,
  },
  {
    key: 'telegram/summary',
    title: 'Telegram summary',
    description: 'Mini-app funnel summary.',
    category: 'telegram',
    default_range: '7d',
    export_formats: ['csv'],
    license_gated: !0,
    feature_key: 'telegram',
  },
];
function qS() {
  return { rows: [...Nk] };
}
var Du = ['US', 'DE', 'GB', 'BR', 'CA', 'PL', 'AU', 'IN'];
function Hk(e) {
  return oe()
    .campaigns.slice(0, 12)
    .map((a, r) => {
      let o = 1200 + r * 340,
        n = Math.max(8, Math.round(o * (0.028 + r * 0.0015))),
        l = ae(420 + r * 88),
        i = ae(610 + r * 124),
        s = {
          campaign_id: a.id,
          campaign_name: a.name,
          country: Du[r % Du.length],
          clicks: o,
          conversions: n,
          spend_micro: l,
          revenue_micro: i,
          profit_micro: i - l,
          roi_pct: l > 0 ? ((i - l) / l) * 100 : 0,
        };
      return e.includes('rtb')
        ? {
            ...s,
            bids: o * 14,
            wins: Math.round(o * 0.62),
            no_bid_reason: r % 3 === 0 ? 'floor' : r % 3 === 1 ? 'geo' : 'category',
          }
        : e.includes('fraud') || e.includes('silent-reject')
          ? {
              ...s,
              blocks: Math.round(o * 0.04),
              silent_rejects: Math.round(o * 0.012),
              fraud_category: r % 2 === 0 ? 'bot' : 'proxy',
            }
          : s;
    });
}
function GS(e) {
  let t = Hk(e);
  return { rows: t, total: t.length, freshness: { stale: !1, label: 'Synthetic preview' } };
}
function VS(e) {
  let t = e.searchParams.get('customer_id') ?? se[0].id,
    a = oe().campaigns.filter((n) => n.customer_id === t),
    r = Array.from({ length: 25 }, (n, l) => {
      let i = a[l % Math.max(a.length, 1)] ?? oe().campaigns[0],
        s = l % 3 !== 0;
      return {
        event_type: 'click',
        click_id: Tn(l + 1),
        campaign_id: i.id,
        placement_id: Dn(l + 1),
        created_at: O(0, l * 2),
        country: Du[l % Du.length],
        sub1: de(xu, l + 1),
        goal_name: ['lead', 'sale', 'install'][l % 3],
        attributed_cost_micro: ae(0.12 + l * 0.03),
        revenue_micro: s ? ae(18 + l * 4.5) : 0,
        inbound_status: 'accepted',
      };
    }),
    o = r
      .slice(0, 8)
      .map((n, l) => ({
        click_id: n.click_id,
        provider: ['webhook', 'facebook', 'google'][l % 3],
        status: l % 4 === 0 ? 'failed' : 'delivered',
        created_at: n.created_at,
        response_code: l % 4 === 0 ? 502 : 200,
      }));
  return {
    events: r,
    postbacks: o,
    freshness: { stale: !1, label: 'Synthetic preview' },
    next_cursor: r.length >= 25 ? 'cursor-mock-2' : void 0,
  };
}
function jS(e) {
  let t = se.find((o) => o.id === e) ?? se[0],
    a = oe().campaigns.filter((o) => o.customer_id === t.id),
    r = Xe.map((o, n) => ({
      user_id: o.id,
      email: o.email,
      role: n === 0 ? 'admin' : n === 1 ? 'buyer' : 'analyst',
      campaigns_owned: a.filter((l) => l.owner_user_id === o.id).length,
      created_at: O(40 + n),
      created_at_display: O(40 + n),
      is_blocked: !1,
      spend_cap_micro: ae(5e4 + n * 1e4),
    }));
  return {
    customer_id: t.id,
    customer_name: t.name,
    cost_center: `CC-${t.name.slice(0, 3).toUpperCase()}`,
    balance_micro: ae(18420.55),
    currency: 'USD',
    license: { state: 'ACTIVE', valid_until: O(-120), plan_code: 'enterprise' },
    members: r,
  };
}
function YS() {
  let e = oe().campaigns,
    t = Xe.map((a, r) => ({
      user_id: a.id,
      email: a.email,
      role: r === 0 ? 'admin' : r === 1 ? 'buyer' : 'analyst',
      campaigns_owned: e.filter((o) => o.owner_user_id === a.id).length,
      created_at: O(30 + r),
      created_at_display: O(30 + r),
      is_blocked: r === 2 && !1,
      spend_cap_micro: ae(4e4 + r * 8500),
    }));
  return { items: t, total: t.length };
}
function XS(e) {
  let { limit: t, offset: a } = yo(e, 100),
    o = oe()
      .campaigns.slice(0, 8)
      .map((l, i) => ({
        id: U('budget_approval', i + 1),
        user_id: l.owner_user_id ?? Xe[1].id,
        campaign_id: l.id,
        requested_budget_micro: ae(8e3 + i * 1250),
        previous_budget_micro: ae(5e3 + i * 900),
        status: i % 4 === 0 ? 'pending' : i % 4 === 1 ? 'approved' : 'denied',
        created_at: O(i % 14),
        created_at_display: O(i % 14),
      }));
  return { ...go(o, t, a), limit: t, offset: a };
}
var zk = {
    A: [
      '*',
      'customers:write',
      'customers:read',
      'campaigns:write',
      'campaigns:read',
      'brands:write',
      'brands:read',
      'settings:write',
      'settings:read',
      'blacklist:write',
      'blacklist:read',
      'audit:read',
      'users:write',
      'shards:write',
      'shards:read',
      'ops:write',
      'rtb:read',
      'rtb:write',
      'billing:read',
    ],
    M: [
      'customers:write',
      'customers:read',
      'campaigns:write',
      'campaigns:read',
      'brands:write',
      'brands:read',
      'audit:read',
    ],
    U: ['campaigns:write', 'campaigns:read', 'customers:read', 'brands:write', 'brands:read'],
    B: ['campaigns:read:masked', 'campaigns:pause'],
    TL: ['campaigns:read', 'campaigns:write', 'campaigns:pause', 'customers:read', 'billing:read'],
    MB: ['campaigns:read', 'campaigns:write', 'customers:read'],
    S: ['campaigns:read:masked', 'audit:read'],
    P: ['supply:read:scoped', 'customers:read'],
  },
  QS = 'adminDevMockRole',
  WS = null;
function Fk(e) {
  switch (e.trim().toUpperCase()) {
    case 'ADMIN':
    case 'SA':
    case 'SUPERADMIN':
    case 'A':
      return 'A';
    case 'MANAGER':
    case 'M':
      return 'M';
    case 'CUSTOMER':
    case 'USER':
    case 'C':
    case 'U':
      return 'U';
    case 'BUYER':
    case 'B':
      return 'B';
    case 'TEAM_LEAD':
    case 'TEAMLEAD':
    case 'TL':
      return 'TL';
    case 'MEDIA_BUYER':
    case 'MEDIABUYER':
    case 'MB':
      return 'MB';
    case 'SUPPORT':
    case 'S':
      return 'S';
    case 'PUBLISHER':
    case 'P':
      return 'P';
    default:
      return 'A';
  }
}
function qk() {
  if (typeof window > 'u') return null;
  try {
    let e = window.localStorage.getItem(QS);
    return e ? Fk(e) : null;
  } catch {
    return null;
  }
}
function xB(e) {
  if (!(typeof window > 'u'))
    try {
      window.localStorage.setItem(QS, e);
    } catch {}
}
function Nm() {
  return WS || (qk() ?? 'A');
}
function KS(e) {
  return e === 'A' ? 'admin' : e;
}
function Gk(e) {
  return zk[e];
}
function Ou() {
  return Gk(Nm());
}
function Vk(e, t = Ou()) {
  return t.some((a) => a === '*' || a === e);
}
function jk(e, t = Ou()) {
  return e.some((a) => Vk(a, t));
}
function Yk() {
  return {
    status: 403,
    body: { error: { code: 'FORBIDDEN', message: 'forbidden: insufficient permissions' } },
    contentType: 'application/json',
  };
}
function Xk(e, t) {
  return e.startsWith('/api/v1/ops/')
    ? t === 'GET'
      ? ['shards:read']
      : e === '/api/v1/ops/roles/reload'
        ? ['settings:write']
        : e.startsWith('/api/v1/ops/blacklist')
          ? ['blacklist:write', 'blacklist:read']
          : ['shards:write', 'ops:write', 'blacklist:write', 'settings:write']
    : e.startsWith('/api/v1/settings/')
      ? t === 'GET'
        ? ['settings:read']
        : ['settings:write']
      : e.startsWith('/api/v1/audit')
        ? ['audit:read']
        : e.startsWith('/api/v1/rtb/')
          ? t === 'GET'
            ? ['rtb:read']
            : ['rtb:write']
          : e.startsWith('/api/v1/dashboards/buyer')
            ? ['campaigns:read', 'campaigns:read:masked']
            : e.startsWith('/api/v1/dashboards/fraud')
              ? ['audit:read']
              : e.startsWith('/api/v1/dashboards/operator') ||
                  e.startsWith('/api/v1/dashboards/adops')
                ? ['shards:read']
                : null;
}
function ZS(e, t) {
  let a = Xk(e, t);
  return a ? (jk(a) ? null : Yk()) : null;
}
function vi(e) {
  return e != null && typeof e == 'object' && !Array.isArray(e);
}
function JS(e) {
  let t = e.trim();
  return t.length <= 8 ? '********' : `${t.slice(0, 4)}...${t.slice(-4)}`;
}
var Wk = new Set([
  'tracking_domain',
  'default_currency',
  'timezone',
  'ingress_schema',
  'telemetry_enabled',
  'profile',
  'edge_xdp',
  'edge_expose_click',
  'edge_expose_openrtb',
  'network_interface',
]);
function Qk() {
  return {
    bootstrap_complete: !0,
    restart_required: !1,
    click_url_template: 'https://track.local/click/{campaign_id}',
    openrtb_endpoint_template: 'https://track.local/openrtb/auction',
    config: {
      tracking_domain: 'track.local',
      default_currency: 'USD',
      timezone: 'UTC',
      ingress_schema: 'ad_event_processor_native',
      telemetry_enabled: !0,
      profile: 'single_vps',
      edge_xdp: !1,
      edge_expose_click: !1,
      edge_expose_openrtb: !1,
      network_interface: 'eth0',
      stripe: { enabled: !1 },
    },
    secrets: { stripe_secret_key: 'sk_dev_redacted', stripe_webhook_secret: 'whsec_dev_redacted' },
  };
}
var Hm = Qk();
function zm() {
  return structuredClone(Hm);
}
function $S(e) {
  let t = structuredClone(Hm),
    a = vi(t.config) ? { ...t.config } : {},
    r = vi(t.secrets) ? { ...t.secrets } : {};
  if ((vi(e.config) && Object.assign(a, e.config), vi(e.stripe))) {
    let o = vi(a.stripe) ? { ...a.stripe } : {};
    typeof e.stripe.enabled == 'boolean' && (o.enabled = e.stripe.enabled),
      (a.stripe = o),
      typeof e.stripe.secret_key == 'string' &&
        e.stripe.secret_key.trim() &&
        (r.stripe_secret_key = JS(e.stripe.secret_key)),
      typeof e.stripe.webhook_secret == 'string' &&
        e.stripe.webhook_secret.trim() &&
        (r.stripe_webhook_secret = JS(e.stripe.webhook_secret));
  }
  for (let [o, n] of Object.entries(e))
    if (!(o === 'config' || o === 'stripe')) {
      if (Wk.has(o)) {
        a[o] = n;
        continue;
      }
      t[o] = n;
    }
  return (t.config = a), (t.secrets = r), (Hm = t), zm();
}
function eL(e) {
  return {
    written_path: `${(e?.trim() || '/var/lib/ad-event-processor').replace(/\/$/, '')}/platform_config.json`,
  };
}
function A(e, t) {
  return { status: e, body: t, contentType: 'application/json' };
}
function Pu(e = 50, t = 0) {
  return A(200, { items: [], total: 0, limit: e, offset: t });
}
function Kk(e, t) {
  let a = Number.parseInt(t.searchParams.get('limit') ?? '50', 10) || 50,
    r = Number.parseInt(t.searchParams.get('offset') ?? '0', 10) || 0;
  return A(200, { items: e.slice(r, r + a), total: e.length, limit: a, offset: r });
}
function bi(e) {
  if (!(!e?.body || typeof e.body != 'string'))
    try {
      return JSON.parse(e.body);
    } catch {
      return;
    }
}
function Zk(e) {
  return Dx(e, oe().campaigns);
}
function Jk(e) {
  let t = Object.fromEntries(Xe.map((a) => [a.id, a.email]));
  return Ox(e, oe().campaigns, t);
}
function $k(e) {
  let t = (e.searchParams.get('ids') ?? '')
      .split(',')
      .map((n) => n.trim())
      .filter(Boolean),
    a = e.searchParams.get('from') ?? new Date(Date.now() - 7 * 864e5).toISOString(),
    r = e.searchParams.get('to') ?? new Date().toISOString(),
    o = {};
  for (let [n, l] of t.entries()) {
    let i = mi(l, n + 1, a, r),
      s = {
        campaign_id: l,
        impressions: i.impressions,
        clicks: i.clicks,
        conversions: i.conversions,
        leads_raw: i.leads_raw,
        hold_leads: i.hold_leads,
        rejected_leads: i.rejected_leads,
        lp_clicks: i.lp_clicks,
        lp_views: i.lp_views,
        unique_clicks: i.unique_clicks,
        blocks: i.blocks,
        bots: i.bots,
        advertiser_spend_micro: i.rtb_cost_micro + i.profit_micro,
        rtb_cost_micro: i.rtb_cost_micro,
        operator_margin_micro: i.profit_micro,
        publisher_payout_micro: Math.floor(i.rtb_cost_micro * 0.72),
        margin_breach: (n + 1) % 13 === 0,
      };
    wu(s, i), (o[l] = s);
  }
  return A(200, { items: o, from: a, to: r, stale: !1 });
}
function e2(e, t) {
  let a = t && typeof t == 'object' ? t : {},
    { campaigns: r } = oe(),
    o = r.findIndex((i) => i.id === e);
  if (o < 0) return A(404, { error: { code: 'NOT_FOUND', message: 'Campaign not found' } });
  let n = r[o],
    l = { ...n, ...a, id: n.id, updated_at: new Date().toISOString() };
  return (r[o] = l), A(200, l);
}
function t2(e) {
  let t = e && typeof e == 'object' ? e : {},
    a = typeof t.action == 'string' ? t.action : 'pause',
    r = a === 'resume' ? 'resume' : a === 'archive' ? 'archive' : 'pause',
    o = Array.isArray(t.campaign_ids) ? t.campaign_ids : [],
    { campaigns: n } = oe(),
    l = o.map((i) => {
      let s = n.find((u) => u.id === i);
      return s
        ? (r === 'archive'
            ? (s.status = 'ARCHIVED')
            : (s.status = r === 'pause' ? 'PAUSED' : 'ACTIVE'),
          (s.updated_at = new Date().toISOString()),
          { id: i, ok: !0 })
        : { id: i, ok: !1, error_code: 'NOT_FOUND' };
    });
  return A(200, { results: l });
}
function Bu() {
  let e = se[0].id,
    t = Nm(),
    a = Ou(),
    r = KS(t);
  return A(200, {
    user: { id: Xe[0].id, email: Xe[0].email, role: r, customer_id: e, permissions: a },
    session: {
      role: r,
      mask_level: t === 'B' || t === 'S' ? 'masked' : 'full',
      default_customer_id: e,
      timezone: 'UTC',
    },
    eula_required: !1,
    eula_accepted: !0,
    eula_version: 'dev',
  });
}
function a2() {
  return A(200, {
    product_name: 'ad-event-processor',
    vendor_name: 'dev',
    version: 'dev-mock',
    bootstrap_complete: !0,
    eula_required: !1,
    eula_accepted: !0,
    payment_enabled: !0,
    license: { state: 'ACTIVE', tier: 'dev' },
  });
}
function r2(e) {
  let t = Number.parseInt(e.searchParams.get('limit') ?? '50', 10) || 50,
    a = Number.parseInt(e.searchParams.get('offset') ?? '0', 10) || 0,
    r = se.map((o, n) => ({
      id: o.id,
      name: o.name,
      status: 'ACTIVE',
      currency: 'USD',
      balance: (12e3 + n * 2450.75).toFixed(6),
      cost_center: `CC-${o.name.slice(0, 3).toUpperCase()}`,
      active_campaigns: oe().campaigns.filter((l) => l.customer_id === o.id).length,
      total_spend: (84200 + n * 12400).toFixed(6),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }));
  return A(200, { items: r.slice(a, a + t), total: r.length, limit: t, offset: a });
}
function o2() {
  return A(200, YS());
}
function n2() {
  return A(200, {
    items: [
      {
        id: 'meta_social_funnel',
        name: 'Meta social funnel',
        description: 'Facebook and Instagram click-to-lander flow with conversion mapping.',
      },
      {
        id: 'push_house_funnel',
        name: 'Push house funnel',
        description: 'Push notification source with hold-friendly postback mapping.',
      },
      {
        id: 'native_mgid_funnel',
        name: 'Native MGID funnel',
        description: 'Native placements with teaser URL macros and geo targeting.',
      },
    ],
  });
}
function l2(e) {
  let t = oe().campaigns.find((a) => a.id === e);
  return t ? A(200, t) : A(404, { error: { code: 'NOT_FOUND', message: 'Campaign not found' } });
}
function i2(e) {
  let t = new Date().toISOString();
  return A(200, {
    id: e,
    customer_id: se[0].id,
    name: `Brand ${e.slice(0, 8)}`,
    created_at: t,
    updated_at: t,
    freq_limit: 0,
    freq_window: 0,
  });
}
function s2(e) {
  return A(200, {
    campaign_id: e,
    sections: [
      { id: 'general', title: 'General', order: 1, visible: !0, complete: !0, issue_count: 0 },
      {
        id: 'targeting',
        title: 'Targeting',
        order: 2,
        visible: !0,
        complete: !1,
        issue_count: 1,
        issue_tone: 'warn',
      },
      { id: 'tracking', title: 'Tracking', order: 3, visible: !0, complete: !0, issue_count: 0 },
    ],
    completion_pct: 67,
    allowed_actions: ['publish', 'pause'],
  });
}
function u2(e, t) {
  let r = (t && typeof t == 'object' ? t : {}).user_id;
  if (typeof r != 'string' || r.length === 0)
    return A(400, { error: { code: 'BAD_REQUEST', message: 'user_id required' } });
  let { campaigns: o } = oe(),
    n = o.findIndex((l) => l.id === e);
  return n < 0
    ? A(404, { error: { code: 'NOT_FOUND', message: 'Campaign not found' } })
    : ((o[n] = { ...o[n], owner_user_id: r, updated_at: new Date().toISOString() }),
      A(200, { status: 'ok' }));
}
function c2(e, t) {
  let a = oe().campaigns.find((i) => i.id === e);
  if (!a) return A(404, { error: { code: 'NOT_FOUND', message: 'Campaign not found' } });
  let r = t && typeof t == 'object' ? t : {},
    o = r.options && typeof r.options == 'object' ? r.options : {},
    n = typeof r.name_prefix == 'string' ? r.name_prefix : '',
    l = typeof r.name_suffix == 'string' ? r.name_suffix : ' (copy)';
  return A(200, {
    source_id: e,
    name: `${n}${a.name}${l}`,
    would_create: {
      include_flow: o.include_flow !== !1,
      include_postbacks: o.include_postbacks !== !1,
      include_fraud: o.include_fraud !== !1,
      include_placement_blocks: o.include_placement_blocks !== !1,
      reset_spend: o.reset_spend === !0,
    },
  });
}
function d2() {
  return A(501, {
    error: { code: 'NOT_IMPLEMENTED', message: 'Route not implemented in dev mock' },
  });
}
function f2() {
  let t = Bu().body;
  return A(200, { user: t.user });
}
function m2() {
  return A(200, { status: 'ok' });
}
function p2() {
  return A(200, qS());
}
function h2() {
  return A(200, zm());
}
function tL(e) {
  let t = e?.body;
  if (!(typeof t != 'string' || !t.trim()))
    try {
      let a = JSON.parse(t);
      return g2(a) ? a : void 0;
    } catch {
      return;
    }
}
function g2(e) {
  return e != null && typeof e == 'object' && !Array.isArray(e);
}
function aL(e, t) {
  if (!e.startsWith('/api/')) return;
  let a = new URL(e, 'http://dev.local'),
    r = (t?.method ?? 'GET').toUpperCase(),
    o = a.pathname;
  if (r === 'GET' && o === '/api/v1/meta') return a2();
  if (r === 'GET' && o === '/api/v1/session/bootstrap') return Bu();
  if (r === 'GET' && o === '/api/v1/auth/me') {
    let i = Bu().body;
    return A(200, i?.user ?? {});
  }
  if (r === 'GET' && o === '/api/v1/session') {
    let i = Bu().body;
    return A(200, i?.session ?? {});
  }
  if (r === 'POST' && (o === '/api/v1/auth/login' || o === '/api/v1/auth/refresh'))
    return o.endsWith('/login') ? f2() : m2();
  if (r === 'POST' && o === '/api/v1/auth/logout') return { status: 204 };
  let n = ZS(o, r);
  if (n) return n;
  if (r === 'GET' && o === '/api/v1/customers') return r2(a);
  if (r === 'GET' && o.startsWith('/api/v1/customers/')) {
    let l = o.slice(18),
      [i, ...s] = l.split('/'),
      u = decodeURIComponent(i),
      d = se.find((c) => c.id === u);
    if (!d) return A(404, { error: { code: 'NOT_FOUND', message: 'Customer not found' } });
    if (s.length === 0) {
      let c = se.findIndex((f) => f.id === u);
      return A(200, {
        id: d.id,
        name: d.name,
        status: 'ACTIVE',
        currency: 'USD',
        balance: (12e3 + c * 2450.75).toFixed(6),
        cost_center: `CC-${d.name.slice(0, 3).toUpperCase()}`,
        active_campaigns: oe().campaigns.filter((f) => f.customer_id === d.id).length,
        total_spend: (84200 + c * 12400).toFixed(6),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      });
    }
    if (s[0] === 'balance') return A(200, Em(u));
    if (s[0] === 'wallet') return A(200, Gx(u));
    if (s[0] === 'ledger') return A(200, Vx(u, a));
    if (s[0] === 'billing' && s[1] === 'statement') return A(200, jx(u, a));
  }
  if (r === 'GET' && o === '/api/v1/campaigns') return Zk(a);
  if (r === 'GET' && o === '/api/v1/campaigns/list-facets') return Jk(a);
  if (r === 'GET' && o === '/api/v1/campaigns/metrics') return $k(a);
  if (r === 'GET' && o === '/api/v1/campaigns/metrics-totals') return Px(a, oe().campaigns);
  if (r === 'GET' && o === '/api/v1/campaigns/export') return Ux(a);
  if (r === 'GET' && o === '/api/v1/campaigns/onboarding-templates') return A(200, kx());
  if (r === 'GET' && o === '/api/v1/campaigns/wizard/session') {
    let l = a.searchParams.get('session_id') ?? '',
      i = Mx(l);
    return A(i.status, i.body);
  }
  if (r === 'POST' && o === '/api/v1/campaigns/wizard/session') {
    let l = bi(t) ?? {},
      i = Ex(l);
    return A(i.status, i.body);
  }
  if (r === 'POST' && (o === '/api/v1/campaigns/bulk' || o === '/api/v1/campaigns/bulk-action'))
    return t2(bi(t));
  if (r === 'PUT' && o.startsWith('/api/v1/campaigns/')) {
    let l = o.slice(18),
      [i, ...s] = l.split('/');
    if (s.length === 1 && s[0] === 'owner') return u2(decodeURIComponent(i), bi(t));
  }
  if (r === 'POST' && o.startsWith('/api/v1/campaigns/')) {
    let l = o.slice(18),
      [i, ...s] = l.split('/');
    if (s.length === 1 && s[0] === 'clone-preview') return c2(decodeURIComponent(i), bi(t));
  }
  if (r === 'PATCH' && o.startsWith('/api/v1/campaigns/')) {
    let l = o.slice(18);
    if (!l.includes('/')) return e2(decodeURIComponent(l), bi(t));
  }
  if (r === 'GET' && o.startsWith('/api/v1/campaigns/')) {
    let l = o.slice(18),
      [i, ...s] = l.split('/');
    if (s.length === 0) return l2(decodeURIComponent(i));
    if (s[0] === 'stats') return Bx(decodeURIComponent(i), a);
    if (s[0] === 'margin')
      return A(200, {
        campaign_id: decodeURIComponent(i),
        window_start: new Date().toISOString(),
        window_hours: 24,
        advertiser_spend_micro: 42e5,
        rtb_cost_micro: 31e5,
        operator_margin_micro: 11e5,
        publisher_payout_micro: 22e5,
      });
    if (s[0] === 'events') return Pu();
    if (s[0] === 'integration-panel') return A(200, { sections: [] });
    if (s[0] === 'integration-health')
      return A(200, { campaign_id: decodeURIComponent(i), summary: 'ok', rows: [] });
    if (s[0] === 'fraud' || s[0] === 'fraud-editor')
      return A(200, { campaign_id: decodeURIComponent(i), enabled: !1 });
    if (s[0] === 'geo-summary') return A(200, { countries: [] });
    if (s[0] === 'conversion-mappings') return A(200, { items: [], total: 0 });
    if (s[0] === 'editor') return s2(decodeURIComponent(i));
  }
  if (r === 'GET' && o === '/api/v1/selfserve/templates') return n2();
  if (r === 'GET') {
    let l = /^\/api\/v1\/brands\/([^/]+)$/.exec(o);
    if (l) return i2(decodeURIComponent(l[1]));
  }
  if (r === 'GET' && o === '/api/v1/team/overview') {
    let l = a.searchParams.get('customer_id')?.trim();
    return A(200, jS(l));
  }
  if (r === 'GET' && o === '/api/v1/team/budget-approvals') return A(200, XS(a));
  if (r === 'GET' && o === '/api/v1/team/members') return o2();
  if (r === 'GET' && o === '/api/v1/reports/catalog') return p2();
  if (r === 'GET' && o === '/api/v1/settings/platform') return h2();
  if (r === 'PATCH' && o === '/api/v1/settings/platform') {
    let l = tL(t);
    return l
      ? A(200, $S(l))
      : A(400, { error: { code: 'BAD_REQUEST', message: 'Patch must be a JSON object' } });
  }
  if (r === 'POST' && o === '/api/v1/settings/platform/apply') {
    let l = tL(t);
    return A(200, eL(typeof l?.install_root == 'string' ? l.install_root : void 0));
  }
  if (r === 'GET' && o === '/api/v1/license/status')
    return A(200, { state: 'ACTIVE', tier: 'dev' });
  if (r === 'GET' && o === '/api/v1/eula') return A(200, { accepted: !0, version: 'dev' });
  if (r === 'GET' && o.startsWith('/api/v1/dashboards/')) return A(200, FS(a, o));
  if (r === 'GET' && o === '/api/v1/reports/click-log') return A(200, VS(a));
  if (r === 'GET' && o.startsWith('/api/v1/reports/')) {
    let l = decodeURIComponent(o.slice(16));
    return l === 'catalog' || l.startsWith('jobs') ? Pu() : A(200, GS(l));
  }
  if (r === 'GET' && o.startsWith('/api/v1/audit')) return A(200, Qx(a));
  if (r === 'GET' && o === '/api/v1/ops/home') return Lx();
  if (r === 'GET' && o === '/api/v1/ops/doctor') return A(200, Cm());
  if (r === 'GET' && o === '/api/v1/ops/health/snapshot') return A(200, wm());
  if (r === 'GET' && o === '/api/v1/ops/dashboard/summary') return A(200, Rm());
  if (r === 'GET' && o === '/api/v1/ops/incidents') return Cx();
  if (r === 'GET' && o === '/api/v1/ops/shards') return wx();
  if (r === 'GET' && o === '/api/v1/ops/dlq') return A(200, { items: [], partial: !1 });
  if (r === 'POST' && o.startsWith('/api/v1/ops/dlq/') && o.endsWith('/retry')) {
    let l = decodeURIComponent(o.slice(16, o.length - 6));
    if (l && !l.includes('/')) return { status: 202 };
  }
  if (
    r === 'GET' &&
    (o === '/api/v1/ops/dlq/inbox' ||
      o === '/api/v1/ops/blacklist' ||
      o === '/api/v1/ops/outbox' ||
      o.startsWith('/api/v1/ops/recon'))
  ) {
    let l = new URL(e, 'http://dev.local'),
      i = Number.parseInt(l.searchParams.get('limit') ?? '50', 10) || 50,
      s = Number.parseInt(l.searchParams.get('offset') ?? '0', 10) || 0;
    return _x(o, i, s);
  }
  if (r === 'GET' && o.startsWith('/api/v1/ops/')) return Rx();
  if (r === 'GET' && o === '/api/v1/billing/summary') return A(200, Hx());
  if (r === 'GET' && o === '/api/v1/billing/invariant') {
    let l = a.searchParams.get('customer_id')?.trim();
    return A(200, qx(l));
  }
  if (r === 'GET' && o === '/api/v1/billing/invoices') return A(200, zx(a));
  if (r === 'GET' && o.startsWith('/api/v1/billing/invoices/')) {
    let l = o.slice(25);
    if (!l.includes('/')) {
      let i = Fx(decodeURIComponent(l));
      return i ? A(200, i) : A(404, { error: { code: 'NOT_FOUND', message: 'Invoice not found' } });
    }
  }
  if (r === 'GET' && o.startsWith('/api/v1/billing')) return Pu();
  if (r === 'POST' && o === '/api/v1/consent') return { status: 204 };
  if (r === 'GET' && o === '/api/v1/cost-sync/snapshot') return A(200, pi());
  if (r === 'GET' && o === '/api/v1/postbacks/snapshot') return A(200, hi());
  if (r === 'GET' && o === '/api/v1/integration/snapshot') return A(200, Ru());
  if (r === 'GET') {
    let l = Am(o);
    if (l) return A(200, l);
  }
  if (r === 'GET' && o.startsWith('/api/v1/')) {
    let l = Am(o);
    return l ? Kk(l, a) : Pu();
  }
  if (r === 'POST' || r === 'PATCH' || r === 'PUT' || r === 'DELETE') return d2();
}
function Fm(e, t) {
  let a = aL(e, t);
  if (a)
    return a.status === 204 || a.status === 202
      ? new Response(null, { status: a.status })
      : new Response(JSON.stringify(a.body ?? {}), {
          status: a.status,
          headers: { 'Content-Type': a.contentType ?? 'application/json' },
        });
}
var Ue = class extends Error {
  status;
  code;
  constructor(t, a, r) {
    super(r), (this.name = 'ApiError'), (this.status = t), (this.code = a);
  }
};
var rL = 15e3;
function y2(e) {
  if (typeof document > 'u') return;
  let t = `${e}=`;
  for (let a of document.cookie.split(';')) {
    let r = a.trim();
    if (r.startsWith(t)) return decodeURIComponent(r.slice(t.length));
  }
}
function v2(e) {
  let t = new AbortController(),
    a = () => {
      t.signal.aborted || t.abort();
    };
  for (let r of e) {
    if (r.aborted) return a(), t.signal;
    r.addEventListener('abort', a, { once: !0 });
  }
  return t.signal;
}
function b2(e) {
  let t = e.toUpperCase();
  return t !== 'GET' && t !== 'HEAD' && t !== 'OPTIONS';
}
async function Uu(e) {
  let t = 'HTTP_ERROR',
    a = e.statusText || `HTTP ${e.status}`;
  try {
    let r = await e.json();
    if (r && typeof r == 'object') {
      let n = r.error;
      if (n && typeof n == 'object') {
        let l = n;
        typeof l.code == 'string' && (t = l.code), typeof l.message == 'string' && (a = l.message);
      } else typeof n == 'string' && (a = n);
    }
  } catch {}
  return new Ue(e.status, t, a);
}
function qm(e) {
  return e instanceof DOMException && e.name === 'AbortError'
    ? !0
    : e instanceof Error && e.name === 'AbortError';
}
async function Nu(e, t = {}) {
  if (En()) {
    let i = Fm(e, t);
    if (i) return i;
  }
  let a = new AbortController(),
    r = setTimeout(() => {
      a.abort();
    }, rL),
    o = [a.signal];
  t.signal && o.push(t.signal);
  let n = (t.method ?? 'GET').toUpperCase(),
    l = new Headers(t.headers ?? void 0);
  if (b2(n)) {
    let i = y2('csrfToken');
    i && l.set('X-CSRF-Token', i);
  }
  t.body != null && !l.has('Content-Type') && l.set('Content-Type', 'application/json');
  try {
    return await fetch(e, { ...t, method: n, headers: l, credentials: 'include', signal: v2(o) });
  } catch (i) {
    throw qm(i) && a.signal.aborted && !t.signal?.aborted
      ? new Ue(0, 'TIMEOUT', `Request timed out after ${rL}ms`)
      : i;
  } finally {
    clearTimeout(r);
  }
}
async function YB(e, t, a) {
  let r = await Nu(e, t);
  if (!r.ok) throw await Uu(r);
  if (r.status === 204 || r.headers.get('Content-Length') === '0') return;
  let n = await r.json();
  return a(n);
}
async function _e(e, t = {}) {
  let a = await Nu(e, t);
  if (!a.ok) throw await Uu(a);
  if (!(a.status === 204 || a.headers.get('Content-Length') === '0')) return await a.json();
}
async function XB(e, t = {}) {
  let a = await _e(e, t);
  if (!Array.isArray(a)) throw new Ue(502, 'INVALID_RESPONSE', 'Expected JSON array response');
  return a;
}
async function WB(e, t, a, r = {}) {
  let o = await Nu(e, t);
  if (!o.ok) throw await Uu(o);
  let n = await o.json();
  if (!Array.isArray(n)) throw new Ue(502, 'INVALID_RESPONSE', 'Expected JSON array response');
  let l = n.map(a),
    i = r.totalHeader ?? 'X-Total-Count',
    s = o.headers.get(i),
    u = s != null ? Number.parseInt(s, 10) : l.length,
    d = Number.isFinite(u) ? u : l.length;
  return { items: l, total: d };
}
function oL(e, t) {
  let [a, r] = (0, $a.useState)(void 0),
    [o, n] = (0, $a.useState)(void 0),
    [l, i] = (0, $a.useState)(!0),
    s = (0, $a.useRef)(0),
    u = (0, $a.useRef)(!1);
  return (
    (0, $a.useEffect)(() => {
      let d = new AbortController(),
        c = ++s.current;
      return (
        u.current || i(!0),
        e(d.signal)
          .then((f) => {
            c === s.current && ((u.current = !0), r(f), n(void 0));
          })
          .catch((f) => {
            c === s.current &&
              (qm(f) || ((u.current = !0), n(f instanceof Error ? f : new Error(String(f)))));
          })
          .finally(() => {
            c === s.current && i(!1);
          }),
        () => {
          d.abort();
        }
      );
    }, t),
    { data: a, error: o, fetching: l }
  );
}
var zu = E(te());
var Hu = E(te());
function nL(e) {
  return e.inFlightGuard && e.inFlight
    ? 'skip_in_flight'
    : e.nowMs - e.lastFiredAtMs < e.windowMs
      ? 'skip_window'
      : 'allow';
}
function lL(e, t = {}) {
  let { windowMs: a = 500, inFlightGuard: r = !1, inFlight: o = !1 } = t,
    n = (0, Hu.useRef)(0);
  return (0, Hu.useCallback)(() => {
    nL({
      lastFiredAtMs: n.current,
      nowMs: Date.now(),
      windowMs: a,
      inFlight: o,
      inFlightGuard: r,
    }) === 'allow' && ((n.current = Date.now()), e());
  }, [e, o, r, a]);
}
function iL() {
  let [e, t] = (0, zu.useState)(0),
    a = (0, zu.useCallback)(() => {
      t((r) => r + 1);
    }, []);
  return { refreshToken: e, bumpRefresh: a };
}
function sL(e, t) {
  return lL(e, { inFlightGuard: !0, inFlight: t });
}
async function uL(e) {
  return _e('/api/v1/meta', { signal: e });
}
async function n4(e) {
  return _e('/api/v1/eula', { signal: e });
}
async function l4(e, t) {
  return _e('/api/v1/eula/accept', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function i4(e) {
  return _e('/api/v1/license/status', { signal: e });
}
async function s4(e, t) {
  return _e('/api/v1/license/apply', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function u4(e = {}, t) {
  let a = new URLSearchParams();
  e.customer_id && a.set('customer_id', e.customer_id),
    e.limit != null && a.set('limit', String(e.limit)),
    e.offset != null && a.set('offset', String(e.offset));
  let r = a.toString(),
    o = r ? `/api/v1/disputes?${r}` : '/api/v1/disputes';
  return _e(o, { signal: t });
}
async function c4(e) {
  return _e('/api/v1/support/feedback/meta', { signal: e });
}
async function d4(e, t) {
  return _e('/api/v1/support/feedback', { method: 'POST', body: JSON.stringify(e), signal: t });
}
var Fu, zn;
function qu(e) {
  return Fu
    ? Promise.resolve(Fu)
    : zn ||
        ((zn = uL(e)
          .then((t) => ((Fu = t), t))
          .finally(() => {
            zn = void 0;
          })),
        zn);
}
function cL() {
  (Fu = void 0), (zn = void 0);
}
var S2 = new Set(['EXPIRED', 'REVOKED']);
function dL(e) {
  return e?.bootstrap_complete === !0;
}
function fL(e) {
  let t = e?.license?.state?.trim();
  return t ? S2.has(t) : !0;
}
function h4(e) {
  return e?.license?.state?.trim() || 'missing';
}
var mL = E($()),
  Gm = (0, Gu.createContext)(void 0);
function S4({ children: e }) {
  let { refreshToken: t, bumpRefresh: a } = iL(),
    { data: r, error: o, fetching: n } = oL((s) => qu(s), [t]),
    l = sL(() => {
      cL(), a();
    }, n),
    i = (0, Gu.useMemo)(
      () => ({
        meta: r,
        error: o,
        loading: n,
        bootstrapComplete: dL(r),
        licenseNeedsSetup: fL(r),
        refreshMeta: l,
      }),
      [r, o, n, l]
    );
  return (0, mL.jsx)(Gm.Provider, { value: i, children: e });
}
async function pL(e, t) {
  return _e('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function R4(e) {
  await _e('/api/v1/auth/logout', { method: 'POST', signal: e });
}
async function L2(e) {
  return _e('/api/v1/auth/me', { signal: e });
}
async function C2(e) {
  return _e('/api/v1/session', { signal: e });
}
async function w2(e) {
  let [t, a, r] = await Promise.all([L2(e), C2(e), qu(e).catch(() => {})]);
  return {
    user: t,
    session: a,
    eula_required: r?.eula_required,
    eula_accepted: r?.eula_accepted,
    eula_version: r?.eula_version,
  };
}
async function _4(e) {
  try {
    return await _e('/api/v1/session/bootstrap', { signal: e });
  } catch (t) {
    if (!(t instanceof Ue) || (t.status !== 401 && t.status !== 404)) throw t;
    try {
      return await w2(e);
    } catch (a) {
      if (a instanceof Ue && a.status === 401) return null;
      throw a;
    }
  }
}
async function hL(e, t) {
  return _e('/api/v1/public/activate', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function I4(e, t) {
  return _e('/api/v1/public/invite/accept', { method: 'POST', body: JSON.stringify(e), signal: t });
}
function gL(e) {
  var t,
    a,
    r = '';
  if (typeof e == 'string' || typeof e == 'number') r += e;
  else if (typeof e == 'object')
    if (Array.isArray(e)) {
      var o = e.length;
      for (t = 0; t < o; t++) e[t] && (a = gL(e[t])) && (r && (r += ' '), (r += a));
    } else for (a in e) e[a] && (r && (r += ' '), (r += a));
  return r;
}
function Vm() {
  for (var e, t, a = 0, r = '', o = arguments.length; a < o; a++)
    (e = arguments[a]) && (t = gL(e)) && (r && (r += ' '), (r += t));
  return r;
}
var M4 = Vm;
var R2 = (e, t) => {
    let a = new Array(e.length + t.length);
    for (let r = 0; r < e.length; r++) a[r] = e[r];
    for (let r = 0; r < t.length; r++) a[e.length + r] = t[r];
    return a;
  },
  _2 = (e, t) => ({ classGroupId: e, validator: t }),
  LL = (e = new Map(), t = null, a) => ({ nextPart: e, validators: t, classGroupId: a });
var yL = [],
  I2 = 'arbitrary..',
  k2 = (e) => {
    let t = E2(e),
      { conflictingClassGroups: a, conflictingClassGroupModifiers: r } = e;
    return {
      getClassGroupId: (l) => {
        if (l.startsWith('[') && l.endsWith(']')) return M2(l);
        let i = l.split('-'),
          s = i[0] === '' && i.length > 1 ? 1 : 0;
        return CL(i, s, t);
      },
      getConflictingClassGroupIds: (l, i) => {
        if (i) {
          let s = r[l],
            u = a[l];
          return s ? (u ? R2(u, s) : s) : u || yL;
        }
        return a[l] || yL;
      },
    };
  },
  CL = (e, t, a) => {
    if (e.length - t === 0) return a.classGroupId;
    let o = e[t],
      n = a.nextPart.get(o);
    if (n) {
      let u = CL(e, t + 1, n);
      if (u) return u;
    }
    let l = a.validators;
    if (l === null) return;
    let i = t === 0 ? e.join('-') : e.slice(t).join('-'),
      s = l.length;
    for (let u = 0; u < s; u++) {
      let d = l[u];
      if (d.validator(i)) return d.classGroupId;
    }
  },
  M2 = (e) =>
    e.slice(1, -1).indexOf(':') === -1
      ? void 0
      : (() => {
          let t = e.slice(1, -1),
            a = t.indexOf(':'),
            r = t.slice(0, a);
          return r ? I2 + r : void 0;
        })(),
  E2 = (e) => {
    let { theme: t, classGroups: a } = e;
    return A2(a, t);
  },
  A2 = (e, t) => {
    let a = LL();
    for (let r in e) {
      let o = e[r];
      Ym(o, a, r, t);
    }
    return a;
  },
  Ym = (e, t, a, r) => {
    let o = e.length;
    for (let n = 0; n < o; n++) {
      let l = e[n];
      T2(l, t, a, r);
    }
  },
  T2 = (e, t, a, r) => {
    if (typeof e == 'string') {
      D2(e, t, a);
      return;
    }
    if (typeof e == 'function') {
      O2(e, t, a, r);
      return;
    }
    P2(e, t, a, r);
  },
  D2 = (e, t, a) => {
    let r = e === '' ? t : wL(t, e);
    r.classGroupId = a;
  },
  O2 = (e, t, a, r) => {
    if (B2(e)) {
      Ym(e(r), t, a, r);
      return;
    }
    t.validators === null && (t.validators = []), t.validators.push(_2(a, e));
  },
  P2 = (e, t, a, r) => {
    let o = Object.entries(e),
      n = o.length;
    for (let l = 0; l < n; l++) {
      let [i, s] = o[l];
      Ym(s, wL(t, i), a, r);
    }
  },
  wL = (e, t) => {
    let a = e,
      r = t.split('-'),
      o = r.length;
    for (let n = 0; n < o; n++) {
      let l = r[n],
        i = a.nextPart.get(l);
      i || ((i = LL()), a.nextPart.set(l, i)), (a = i);
    }
    return a;
  },
  B2 = (e) => 'isThemeGetter' in e && e.isThemeGetter === !0,
  U2 = (e) => {
    if (e < 1) return { get: () => {}, set: () => {} };
    let t = 0,
      a = Object.create(null),
      r = Object.create(null),
      o = (n, l) => {
        (a[n] = l), t++, t > e && ((t = 0), (r = a), (a = Object.create(null)));
      };
    return {
      get(n) {
        let l = a[n];
        if (l !== void 0) return l;
        if ((l = r[n]) !== void 0) return o(n, l), l;
      },
      set(n, l) {
        n in a ? (a[n] = l) : o(n, l);
      },
    };
  };
var N2 = [],
  vL = (e, t, a, r, o) => ({
    modifiers: e,
    hasImportantModifier: t,
    baseClassName: a,
    maybePostfixModifierPosition: r,
    isExternal: o,
  }),
  H2 = (e) => {
    let { prefix: t, experimentalParseClassName: a } = e,
      r = (o) => {
        let n = [],
          l = 0,
          i = 0,
          s = 0,
          u,
          d = o.length;
        for (let x = 0; x < d; x++) {
          let y = o[x];
          if (l === 0 && i === 0) {
            if (y === ':') {
              n.push(o.slice(s, x)), (s = x + 1);
              continue;
            }
            if (y === '/') {
              u = x;
              continue;
            }
          }
          y === '[' ? l++ : y === ']' ? l-- : y === '(' ? i++ : y === ')' && i--;
        }
        let c = n.length === 0 ? o : o.slice(s),
          f = c,
          h = !1;
        c.endsWith('!')
          ? ((f = c.slice(0, -1)), (h = !0))
          : c.startsWith('!') && ((f = c.slice(1)), (h = !0));
        let v = u && u > s ? u - s : void 0;
        return vL(n, h, f, v);
      };
    if (t) {
      let o = t + ':',
        n = r;
      r = (l) => (l.startsWith(o) ? n(l.slice(o.length)) : vL(N2, !1, l, void 0, !0));
    }
    if (a) {
      let o = r;
      r = (n) => a({ className: n, parseClassName: o });
    }
    return r;
  },
  z2 = (e) => {
    let t = new Map();
    return (
      e.orderSensitiveModifiers.forEach((a, r) => {
        t.set(a, 1e6 + r);
      }),
      (a) => {
        let r = [],
          o = [];
        for (let n = 0; n < a.length; n++) {
          let l = a[n],
            i = l[0] === '[',
            s = t.has(l);
          i || s ? (o.length > 0 && (o.sort(), r.push(...o), (o = [])), r.push(l)) : o.push(l);
        }
        return o.length > 0 && (o.sort(), r.push(...o)), r;
      }
    );
  },
  F2 = (e) => ({
    cache: U2(e.cacheSize),
    parseClassName: H2(e),
    sortModifiers: z2(e),
    postfixLookupClassGroupIds: q2(e),
    ...k2(e),
  }),
  q2 = (e) => {
    let t = Object.create(null),
      a = e.postfixLookupClassGroups;
    if (a) for (let r = 0; r < a.length; r++) t[a[r]] = !0;
    return t;
  },
  G2 = /\s+/,
  V2 = (e, t) => {
    let {
        parseClassName: a,
        getClassGroupId: r,
        getConflictingClassGroupIds: o,
        sortModifiers: n,
        postfixLookupClassGroupIds: l,
      } = t,
      i = [],
      s = e.trim().split(G2),
      u = '';
    for (let d = s.length - 1; d >= 0; d -= 1) {
      let c = s[d],
        {
          isExternal: f,
          modifiers: h,
          hasImportantModifier: v,
          baseClassName: x,
          maybePostfixModifierPosition: y,
        } = a(c);
      if (f) {
        u = c + (u.length > 0 ? ' ' + u : u);
        continue;
      }
      let m = !!y,
        p;
      if (m) {
        let w = x.substring(0, y);
        p = r(w);
        let L = p && l[p] ? r(x) : void 0;
        L && L !== p && ((p = L), (m = !1));
      } else p = r(x);
      if (!p) {
        if (!m) {
          u = c + (u.length > 0 ? ' ' + u : u);
          continue;
        }
        if (((p = r(x)), !p)) {
          u = c + (u.length > 0 ? ' ' + u : u);
          continue;
        }
        m = !1;
      }
      let g = h.length === 0 ? '' : h.length === 1 ? h[0] : n(h).join(':'),
        b = v ? g + '!' : g,
        R = b + p;
      if (i.indexOf(R) > -1) continue;
      i.push(R);
      let I = o(p, m);
      for (let w = 0; w < I.length; ++w) {
        let L = I[w];
        i.push(b + L);
      }
      u = c + (u.length > 0 ? ' ' + u : u);
    }
    return u;
  },
  j2 = (...e) => {
    let t = 0,
      a,
      r,
      o = '';
    for (; t < e.length; ) (a = e[t++]) && (r = RL(a)) && (o && (o += ' '), (o += r));
    return o;
  },
  RL = (e) => {
    if (typeof e == 'string') return e;
    let t,
      a = '';
    for (let r = 0; r < e.length; r++) e[r] && (t = RL(e[r])) && (a && (a += ' '), (a += t));
    return a;
  },
  Y2 = (e, ...t) => {
    let a,
      r,
      o,
      n,
      l = (s) => {
        let u = t.reduce((d, c) => c(d), e());
        return (a = F2(u)), (r = a.cache.get), (o = a.cache.set), (n = i), i(s);
      },
      i = (s) => {
        let u = r(s);
        if (u) return u;
        let d = V2(s, a);
        return o(s, d), d;
      };
    return (n = l), (...s) => n(j2(...s));
  },
  X2 = [],
  Fe = (e) => {
    let t = (a) => a[e] || X2;
    return (t.isThemeGetter = !0), t;
  },
  _L = /^\[(?:(\w[\w-]*):)?(.+)\]$/i,
  IL = /^\((?:(\w[\w-]*):)?(.+)\)$/i,
  W2 = /^\d+(?:\.\d+)?\/\d+(?:\.\d+)?$/,
  Q2 = /^(\d+(\.\d+)?)?(xs|sm|md|lg|xl)$/,
  K2 =
    /\d+(%|px|r?em|[sdl]?v([hwib]|min|max)|pt|pc|in|cm|mm|cap|ch|ex|r?lh|cq(w|h|i|b|min|max))|\b(calc|min|max|clamp)\(.+\)|^0$/,
  Z2 = /^(rgba?|hsla?|hwb|(ok)?(lab|lch)|color-mix)\(.+\)$/,
  J2 = /^(inset_)?-?((\d+)?\.?(\d+)[a-z]+|0)_-?((\d+)?\.?(\d+)[a-z]+|0)/,
  $2 =
    /^(url|image|image-set|cross-fade|element|(repeating-)?(linear|radial|conic)-gradient)\(.+\)$/,
  Nr = (e) => W2.test(e),
  V = (e) => !!e && !Number.isNaN(Number(e)),
  wa = (e) => !!e && Number.isInteger(Number(e)),
  jm = (e) => e.endsWith('%') && V(e.slice(0, -1)),
  er = (e) => Q2.test(e),
  kL = () => !0,
  eM = (e) => K2.test(e) && !Z2.test(e),
  Xm = () => !1,
  tM = (e) => J2.test(e),
  aM = (e) => $2.test(e),
  rM = (e) => !k(e) && !T(e),
  oM = (e) =>
    e.startsWith('@container') &&
    ((e[10] === '/' && e[11] !== void 0) ||
      (e[11] === 's' && e[16] !== void 0 && e.startsWith('-size/', 10)) ||
      (e[11] === 'n' && e[18] !== void 0 && e.startsWith('-normal/', 10))),
  nM = (e) => Hr(e, AL, Xm),
  k = (e) => _L.test(e),
  xo = (e) => Hr(e, TL, eM),
  bL = (e) => Hr(e, mM, V),
  lM = (e) => Hr(e, OL, kL),
  iM = (e) => Hr(e, DL, Xm),
  xL = (e) => Hr(e, ML, Xm),
  sM = (e) => Hr(e, EL, aM),
  Vu = (e) => Hr(e, PL, tM),
  T = (e) => IL.test(e),
  xi = (e) => So(e, TL),
  uM = (e) => So(e, DL),
  SL = (e) => So(e, ML),
  cM = (e) => So(e, AL),
  dM = (e) => So(e, EL),
  ju = (e) => So(e, PL, !0),
  fM = (e) => So(e, OL, !0),
  Hr = (e, t, a) => {
    let r = _L.exec(e);
    return r ? (r[1] ? t(r[1]) : a(r[2])) : !1;
  },
  So = (e, t, a = !1) => {
    let r = IL.exec(e);
    return r ? (r[1] ? t(r[1]) : a) : !1;
  },
  ML = (e) => e === 'position' || e === 'percentage',
  EL = (e) => e === 'image' || e === 'url',
  AL = (e) => e === 'length' || e === 'size' || e === 'bg-size',
  TL = (e) => e === 'length',
  mM = (e) => e === 'number',
  DL = (e) => e === 'family-name',
  OL = (e) => e === 'number' || e === 'weight',
  PL = (e) => e === 'shadow';
var pM = () => {
  let e = Fe('color'),
    t = Fe('font'),
    a = Fe('text'),
    r = Fe('font-weight'),
    o = Fe('tracking'),
    n = Fe('leading'),
    l = Fe('breakpoint'),
    i = Fe('container'),
    s = Fe('spacing'),
    u = Fe('radius'),
    d = Fe('shadow'),
    c = Fe('inset-shadow'),
    f = Fe('text-shadow'),
    h = Fe('drop-shadow'),
    v = Fe('blur'),
    x = Fe('perspective'),
    y = Fe('aspect'),
    m = Fe('ease'),
    p = Fe('animate'),
    g = () => ['auto', 'avoid', 'all', 'avoid-page', 'page', 'left', 'right', 'column'],
    b = () => [
      'center',
      'top',
      'bottom',
      'left',
      'right',
      'top-left',
      'left-top',
      'top-right',
      'right-top',
      'bottom-right',
      'right-bottom',
      'bottom-left',
      'left-bottom',
    ],
    R = () => [...b(), T, k],
    I = () => ['auto', 'hidden', 'clip', 'visible', 'scroll'],
    w = () => ['auto', 'contain', 'none'],
    L = () => [T, k, s],
    M = () => [Nr, 'full', 'auto', ...L()],
    z = () => [wa, 'none', 'subgrid', T, k],
    qe = () => ['auto', { span: ['full', wa, T, k] }, wa, T, k],
    pt = () => [wa, 'auto', T, k],
    aa = () => ['auto', 'min', 'max', 'fr', T, k],
    nt = () => [
      'start',
      'end',
      'center',
      'between',
      'around',
      'evenly',
      'stretch',
      'baseline',
      'center-safe',
      'end-safe',
    ],
    Ee = () => ['start', 'end', 'center', 'stretch', 'center-safe', 'end-safe'],
    Y = () => ['auto', ...L()],
    Ne = () => [
      Nr,
      'auto',
      'full',
      'dvw',
      'dvh',
      'lvw',
      'lvh',
      'svw',
      'svh',
      'min',
      'max',
      'fit',
      ...L(),
    ],
    It = () => [Nr, 'screen', 'full', 'dvw', 'lvw', 'svw', 'min', 'max', 'fit', ...L()],
    Ke = () => [Nr, 'screen', 'full', 'lh', 'dvh', 'lvh', 'svh', 'min', 'max', 'fit', ...L()],
    B = () => [e, T, k],
    qt = () => [...b(), SL, xL, { position: [T, k] }],
    ra = () => ['no-repeat', { repeat: ['', 'x', 'y', 'space', 'round'] }],
    Ia = () => ['auto', 'cover', 'contain', cM, nM, { size: [T, k] }],
    ee = () => [jm, xi, xo],
    j = () => ['', 'none', 'full', u, T, k],
    X = () => ['', V, xi, xo],
    Ze = () => ['solid', 'dashed', 'dotted', 'double'],
    oa = () => [
      'normal',
      'multiply',
      'screen',
      'overlay',
      'darken',
      'lighten',
      'color-dodge',
      'color-burn',
      'hard-light',
      'soft-light',
      'difference',
      'exclusion',
      'hue',
      'saturation',
      'color',
      'luminosity',
    ],
    q = () => [V, jm, SL, xL],
    nr = () => ['', 'none', v, T, k],
    ka = () => ['none', V, T, k],
    ca = () => ['none', V, T, k],
    Ma = () => [V, T, k],
    lr = () => [Nr, 'full', ...L()];
  return {
    cacheSize: 500,
    theme: {
      animate: ['spin', 'ping', 'pulse', 'bounce'],
      aspect: ['video'],
      blur: [er],
      breakpoint: [er],
      color: [kL],
      container: [er],
      'drop-shadow': [er],
      ease: ['in', 'out', 'in-out'],
      font: [rM],
      'font-weight': [
        'thin',
        'extralight',
        'light',
        'normal',
        'medium',
        'semibold',
        'bold',
        'extrabold',
        'black',
      ],
      'inset-shadow': [er],
      leading: ['none', 'tight', 'snug', 'normal', 'relaxed', 'loose'],
      perspective: ['dramatic', 'near', 'normal', 'midrange', 'distant', 'none'],
      radius: [er],
      shadow: [er],
      spacing: ['px', V],
      text: [er],
      'text-shadow': [er],
      tracking: ['tighter', 'tight', 'normal', 'wide', 'wider', 'widest'],
    },
    classGroups: {
      aspect: [{ aspect: ['auto', 'square', Nr, k, T, y] }],
      container: ['container'],
      'container-type': [{ '@container': ['', 'normal', 'size', T, k] }],
      'container-named': [oM],
      columns: [{ columns: [V, k, T, i] }],
      'break-after': [{ 'break-after': g() }],
      'break-before': [{ 'break-before': g() }],
      'break-inside': [{ 'break-inside': ['auto', 'avoid', 'avoid-page', 'avoid-column'] }],
      'box-decoration': [{ 'box-decoration': ['slice', 'clone'] }],
      box: [{ box: ['border', 'content'] }],
      display: [
        'block',
        'inline-block',
        'inline',
        'flex',
        'inline-flex',
        'table',
        'inline-table',
        'table-caption',
        'table-cell',
        'table-column',
        'table-column-group',
        'table-footer-group',
        'table-header-group',
        'table-row-group',
        'table-row',
        'flow-root',
        'grid',
        'inline-grid',
        'contents',
        'list-item',
        'hidden',
      ],
      sr: ['sr-only', 'not-sr-only'],
      float: [{ float: ['right', 'left', 'none', 'start', 'end'] }],
      clear: [{ clear: ['left', 'right', 'both', 'none', 'start', 'end'] }],
      isolation: ['isolate', 'isolation-auto'],
      'object-fit': [{ object: ['contain', 'cover', 'fill', 'none', 'scale-down'] }],
      'object-position': [{ object: R() }],
      overflow: [{ overflow: I() }],
      'overflow-x': [{ 'overflow-x': I() }],
      'overflow-y': [{ 'overflow-y': I() }],
      overscroll: [{ overscroll: w() }],
      'overscroll-x': [{ 'overscroll-x': w() }],
      'overscroll-y': [{ 'overscroll-y': w() }],
      position: ['static', 'fixed', 'absolute', 'relative', 'sticky'],
      inset: [{ inset: M() }],
      'inset-x': [{ 'inset-x': M() }],
      'inset-y': [{ 'inset-y': M() }],
      start: [{ 'inset-s': M(), start: M() }],
      end: [{ 'inset-e': M(), end: M() }],
      'inset-bs': [{ 'inset-bs': M() }],
      'inset-be': [{ 'inset-be': M() }],
      top: [{ top: M() }],
      right: [{ right: M() }],
      bottom: [{ bottom: M() }],
      left: [{ left: M() }],
      visibility: ['visible', 'invisible', 'collapse'],
      z: [{ z: [wa, 'auto', T, k] }],
      basis: [{ basis: [Nr, 'full', 'auto', i, ...L()] }],
      'flex-direction': [{ flex: ['row', 'row-reverse', 'col', 'col-reverse'] }],
      'flex-wrap': [{ flex: ['nowrap', 'wrap', 'wrap-reverse'] }],
      flex: [{ flex: [V, Nr, 'auto', 'initial', 'none', k] }],
      grow: [{ grow: ['', V, T, k] }],
      shrink: [{ shrink: ['', V, T, k] }],
      order: [{ order: [wa, 'first', 'last', 'none', T, k] }],
      'grid-cols': [{ 'grid-cols': z() }],
      'col-start-end': [{ col: qe() }],
      'col-start': [{ 'col-start': pt() }],
      'col-end': [{ 'col-end': pt() }],
      'grid-rows': [{ 'grid-rows': z() }],
      'row-start-end': [{ row: qe() }],
      'row-start': [{ 'row-start': pt() }],
      'row-end': [{ 'row-end': pt() }],
      'grid-flow': [{ 'grid-flow': ['row', 'col', 'dense', 'row-dense', 'col-dense'] }],
      'auto-cols': [{ 'auto-cols': aa() }],
      'auto-rows': [{ 'auto-rows': aa() }],
      gap: [{ gap: L() }],
      'gap-x': [{ 'gap-x': L() }],
      'gap-y': [{ 'gap-y': L() }],
      'justify-content': [{ justify: [...nt(), 'normal'] }],
      'justify-items': [{ 'justify-items': [...Ee(), 'normal'] }],
      'justify-self': [{ 'justify-self': ['auto', ...Ee()] }],
      'align-content': [{ content: ['normal', ...nt()] }],
      'align-items': [{ items: [...Ee(), { baseline: ['', 'last'] }] }],
      'align-self': [{ self: ['auto', ...Ee(), { baseline: ['', 'last'] }] }],
      'place-content': [{ 'place-content': nt() }],
      'place-items': [{ 'place-items': [...Ee(), 'baseline'] }],
      'place-self': [{ 'place-self': ['auto', ...Ee()] }],
      p: [{ p: L() }],
      px: [{ px: L() }],
      py: [{ py: L() }],
      ps: [{ ps: L() }],
      pe: [{ pe: L() }],
      pbs: [{ pbs: L() }],
      pbe: [{ pbe: L() }],
      pt: [{ pt: L() }],
      pr: [{ pr: L() }],
      pb: [{ pb: L() }],
      pl: [{ pl: L() }],
      m: [{ m: Y() }],
      mx: [{ mx: Y() }],
      my: [{ my: Y() }],
      ms: [{ ms: Y() }],
      me: [{ me: Y() }],
      mbs: [{ mbs: Y() }],
      mbe: [{ mbe: Y() }],
      mt: [{ mt: Y() }],
      mr: [{ mr: Y() }],
      mb: [{ mb: Y() }],
      ml: [{ ml: Y() }],
      'space-x': [{ 'space-x': L() }],
      'space-x-reverse': ['space-x-reverse'],
      'space-y': [{ 'space-y': L() }],
      'space-y-reverse': ['space-y-reverse'],
      size: [{ size: Ne() }],
      'inline-size': [{ inline: ['auto', ...It()] }],
      'min-inline-size': [{ 'min-inline': ['auto', ...It()] }],
      'max-inline-size': [{ 'max-inline': ['none', ...It()] }],
      'block-size': [{ block: ['auto', ...Ke()] }],
      'min-block-size': [{ 'min-block': ['auto', ...Ke()] }],
      'max-block-size': [{ 'max-block': ['none', ...Ke()] }],
      w: [{ w: [i, 'screen', ...Ne()] }],
      'min-w': [{ 'min-w': [i, 'screen', 'none', ...Ne()] }],
      'max-w': [{ 'max-w': [i, 'screen', 'none', 'prose', { screen: [l] }, ...Ne()] }],
      h: [{ h: ['screen', 'lh', ...Ne()] }],
      'min-h': [{ 'min-h': ['screen', 'lh', 'none', ...Ne()] }],
      'max-h': [{ 'max-h': ['screen', 'lh', ...Ne()] }],
      'font-size': [{ text: ['base', a, xi, xo] }],
      'font-smoothing': ['antialiased', 'subpixel-antialiased'],
      'font-style': ['italic', 'not-italic'],
      'font-weight': [{ font: [r, fM, lM] }],
      'font-stretch': [
        {
          'font-stretch': [
            'ultra-condensed',
            'extra-condensed',
            'condensed',
            'semi-condensed',
            'normal',
            'semi-expanded',
            'expanded',
            'extra-expanded',
            'ultra-expanded',
            jm,
            k,
          ],
        },
      ],
      'font-family': [{ font: [uM, iM, t] }],
      'font-features': [{ 'font-features': [k] }],
      'fvn-normal': ['normal-nums'],
      'fvn-ordinal': ['ordinal'],
      'fvn-slashed-zero': ['slashed-zero'],
      'fvn-figure': ['lining-nums', 'oldstyle-nums'],
      'fvn-spacing': ['proportional-nums', 'tabular-nums'],
      'fvn-fraction': ['diagonal-fractions', 'stacked-fractions'],
      tracking: [{ tracking: [o, T, k] }],
      'line-clamp': [{ 'line-clamp': [V, 'none', T, bL] }],
      leading: [{ leading: [n, ...L()] }],
      'list-image': [{ 'list-image': ['none', T, k] }],
      'list-style-position': [{ list: ['inside', 'outside'] }],
      'list-style-type': [{ list: ['disc', 'decimal', 'none', T, k] }],
      'text-alignment': [{ text: ['left', 'center', 'right', 'justify', 'start', 'end'] }],
      'placeholder-color': [{ placeholder: B() }],
      'text-color': [{ text: B() }],
      'text-decoration': ['underline', 'overline', 'line-through', 'no-underline'],
      'text-decoration-style': [{ decoration: [...Ze(), 'wavy'] }],
      'text-decoration-thickness': [{ decoration: [V, 'from-font', 'auto', T, xo] }],
      'text-decoration-color': [{ decoration: B() }],
      'underline-offset': [{ 'underline-offset': [V, 'auto', T, k] }],
      'text-transform': ['uppercase', 'lowercase', 'capitalize', 'normal-case'],
      'text-overflow': ['truncate', 'text-ellipsis', 'text-clip'],
      'text-wrap': [{ text: ['wrap', 'nowrap', 'balance', 'pretty'] }],
      indent: [{ indent: L() }],
      'tab-size': [{ tab: [wa, T, k] }],
      'vertical-align': [
        {
          align: [
            'baseline',
            'top',
            'middle',
            'bottom',
            'text-top',
            'text-bottom',
            'sub',
            'super',
            T,
            k,
          ],
        },
      ],
      whitespace: [
        { whitespace: ['normal', 'nowrap', 'pre', 'pre-line', 'pre-wrap', 'break-spaces'] },
      ],
      break: [{ break: ['normal', 'words', 'all', 'keep'] }],
      wrap: [{ wrap: ['break-word', 'anywhere', 'normal'] }],
      hyphens: [{ hyphens: ['none', 'manual', 'auto'] }],
      content: [{ content: ['none', T, k] }],
      'bg-attachment': [{ bg: ['fixed', 'local', 'scroll'] }],
      'bg-clip': [{ 'bg-clip': ['border', 'padding', 'content', 'text'] }],
      'bg-origin': [{ 'bg-origin': ['border', 'padding', 'content'] }],
      'bg-position': [{ bg: qt() }],
      'bg-repeat': [{ bg: ra() }],
      'bg-size': [{ bg: Ia() }],
      'bg-image': [
        {
          bg: [
            'none',
            {
              linear: [{ to: ['t', 'tr', 'r', 'br', 'b', 'bl', 'l', 'tl'] }, wa, T, k],
              radial: ['', T, k],
              conic: [wa, T, k],
            },
            dM,
            sM,
          ],
        },
      ],
      'bg-color': [{ bg: B() }],
      'gradient-from-pos': [{ from: ee() }],
      'gradient-via-pos': [{ via: ee() }],
      'gradient-to-pos': [{ to: ee() }],
      'gradient-from': [{ from: B() }],
      'gradient-via': [{ via: B() }],
      'gradient-to': [{ to: B() }],
      rounded: [{ rounded: j() }],
      'rounded-s': [{ 'rounded-s': j() }],
      'rounded-e': [{ 'rounded-e': j() }],
      'rounded-t': [{ 'rounded-t': j() }],
      'rounded-r': [{ 'rounded-r': j() }],
      'rounded-b': [{ 'rounded-b': j() }],
      'rounded-l': [{ 'rounded-l': j() }],
      'rounded-ss': [{ 'rounded-ss': j() }],
      'rounded-se': [{ 'rounded-se': j() }],
      'rounded-ee': [{ 'rounded-ee': j() }],
      'rounded-es': [{ 'rounded-es': j() }],
      'rounded-tl': [{ 'rounded-tl': j() }],
      'rounded-tr': [{ 'rounded-tr': j() }],
      'rounded-br': [{ 'rounded-br': j() }],
      'rounded-bl': [{ 'rounded-bl': j() }],
      'border-w': [{ border: X() }],
      'border-w-x': [{ 'border-x': X() }],
      'border-w-y': [{ 'border-y': X() }],
      'border-w-s': [{ 'border-s': X() }],
      'border-w-e': [{ 'border-e': X() }],
      'border-w-bs': [{ 'border-bs': X() }],
      'border-w-be': [{ 'border-be': X() }],
      'border-w-t': [{ 'border-t': X() }],
      'border-w-r': [{ 'border-r': X() }],
      'border-w-b': [{ 'border-b': X() }],
      'border-w-l': [{ 'border-l': X() }],
      'divide-x': [{ 'divide-x': X() }],
      'divide-x-reverse': ['divide-x-reverse'],
      'divide-y': [{ 'divide-y': X() }],
      'divide-y-reverse': ['divide-y-reverse'],
      'border-style': [{ border: [...Ze(), 'hidden', 'none'] }],
      'divide-style': [{ divide: [...Ze(), 'hidden', 'none'] }],
      'border-color': [{ border: B() }],
      'border-color-x': [{ 'border-x': B() }],
      'border-color-y': [{ 'border-y': B() }],
      'border-color-s': [{ 'border-s': B() }],
      'border-color-e': [{ 'border-e': B() }],
      'border-color-bs': [{ 'border-bs': B() }],
      'border-color-be': [{ 'border-be': B() }],
      'border-color-t': [{ 'border-t': B() }],
      'border-color-r': [{ 'border-r': B() }],
      'border-color-b': [{ 'border-b': B() }],
      'border-color-l': [{ 'border-l': B() }],
      'divide-color': [{ divide: B() }],
      'outline-style': [{ outline: [...Ze(), 'none', 'hidden'] }],
      'outline-offset': [{ 'outline-offset': [V, T, k] }],
      'outline-w': [{ outline: ['', V, xi, xo] }],
      'outline-color': [{ outline: B() }],
      shadow: [{ shadow: ['', 'none', d, ju, Vu] }],
      'shadow-color': [{ shadow: B() }],
      'inset-shadow': [{ 'inset-shadow': ['none', c, ju, Vu] }],
      'inset-shadow-color': [{ 'inset-shadow': B() }],
      'ring-w': [{ ring: X() }],
      'ring-w-inset': ['ring-inset'],
      'ring-color': [{ ring: B() }],
      'ring-offset-w': [{ 'ring-offset': [V, xo] }],
      'ring-offset-color': [{ 'ring-offset': B() }],
      'inset-ring-w': [{ 'inset-ring': X() }],
      'inset-ring-color': [{ 'inset-ring': B() }],
      'text-shadow': [{ 'text-shadow': ['none', f, ju, Vu] }],
      'text-shadow-color': [{ 'text-shadow': B() }],
      opacity: [{ opacity: [V, T, k] }],
      'mix-blend': [{ 'mix-blend': [...oa(), 'plus-darker', 'plus-lighter'] }],
      'bg-blend': [{ 'bg-blend': oa() }],
      'mask-clip': [
        { 'mask-clip': ['border', 'padding', 'content', 'fill', 'stroke', 'view'] },
        'mask-no-clip',
      ],
      'mask-composite': [{ mask: ['add', 'subtract', 'intersect', 'exclude'] }],
      'mask-image-linear-pos': [{ 'mask-linear': [V] }],
      'mask-image-linear-from-pos': [{ 'mask-linear-from': q() }],
      'mask-image-linear-to-pos': [{ 'mask-linear-to': q() }],
      'mask-image-linear-from-color': [{ 'mask-linear-from': B() }],
      'mask-image-linear-to-color': [{ 'mask-linear-to': B() }],
      'mask-image-t-from-pos': [{ 'mask-t-from': q() }],
      'mask-image-t-to-pos': [{ 'mask-t-to': q() }],
      'mask-image-t-from-color': [{ 'mask-t-from': B() }],
      'mask-image-t-to-color': [{ 'mask-t-to': B() }],
      'mask-image-r-from-pos': [{ 'mask-r-from': q() }],
      'mask-image-r-to-pos': [{ 'mask-r-to': q() }],
      'mask-image-r-from-color': [{ 'mask-r-from': B() }],
      'mask-image-r-to-color': [{ 'mask-r-to': B() }],
      'mask-image-b-from-pos': [{ 'mask-b-from': q() }],
      'mask-image-b-to-pos': [{ 'mask-b-to': q() }],
      'mask-image-b-from-color': [{ 'mask-b-from': B() }],
      'mask-image-b-to-color': [{ 'mask-b-to': B() }],
      'mask-image-l-from-pos': [{ 'mask-l-from': q() }],
      'mask-image-l-to-pos': [{ 'mask-l-to': q() }],
      'mask-image-l-from-color': [{ 'mask-l-from': B() }],
      'mask-image-l-to-color': [{ 'mask-l-to': B() }],
      'mask-image-x-from-pos': [{ 'mask-x-from': q() }],
      'mask-image-x-to-pos': [{ 'mask-x-to': q() }],
      'mask-image-x-from-color': [{ 'mask-x-from': B() }],
      'mask-image-x-to-color': [{ 'mask-x-to': B() }],
      'mask-image-y-from-pos': [{ 'mask-y-from': q() }],
      'mask-image-y-to-pos': [{ 'mask-y-to': q() }],
      'mask-image-y-from-color': [{ 'mask-y-from': B() }],
      'mask-image-y-to-color': [{ 'mask-y-to': B() }],
      'mask-image-radial': [{ 'mask-radial': [T, k] }],
      'mask-image-radial-from-pos': [{ 'mask-radial-from': q() }],
      'mask-image-radial-to-pos': [{ 'mask-radial-to': q() }],
      'mask-image-radial-from-color': [{ 'mask-radial-from': B() }],
      'mask-image-radial-to-color': [{ 'mask-radial-to': B() }],
      'mask-image-radial-shape': [{ 'mask-radial': ['circle', 'ellipse'] }],
      'mask-image-radial-size': [
        { 'mask-radial': [{ closest: ['side', 'corner'], farthest: ['side', 'corner'] }] },
      ],
      'mask-image-radial-pos': [{ 'mask-radial-at': b() }],
      'mask-image-conic-pos': [{ 'mask-conic': [V] }],
      'mask-image-conic-from-pos': [{ 'mask-conic-from': q() }],
      'mask-image-conic-to-pos': [{ 'mask-conic-to': q() }],
      'mask-image-conic-from-color': [{ 'mask-conic-from': B() }],
      'mask-image-conic-to-color': [{ 'mask-conic-to': B() }],
      'mask-mode': [{ mask: ['alpha', 'luminance', 'match'] }],
      'mask-origin': [
        { 'mask-origin': ['border', 'padding', 'content', 'fill', 'stroke', 'view'] },
      ],
      'mask-position': [{ mask: qt() }],
      'mask-repeat': [{ mask: ra() }],
      'mask-size': [{ mask: Ia() }],
      'mask-type': [{ 'mask-type': ['alpha', 'luminance'] }],
      'mask-image': [{ mask: ['none', T, k] }],
      filter: [{ filter: ['', 'none', T, k] }],
      blur: [{ blur: nr() }],
      brightness: [{ brightness: [V, T, k] }],
      contrast: [{ contrast: [V, T, k] }],
      'drop-shadow': [{ 'drop-shadow': ['', 'none', h, ju, Vu] }],
      'drop-shadow-color': [{ 'drop-shadow': B() }],
      grayscale: [{ grayscale: ['', V, T, k] }],
      'hue-rotate': [{ 'hue-rotate': [V, T, k] }],
      invert: [{ invert: ['', V, T, k] }],
      saturate: [{ saturate: [V, T, k] }],
      sepia: [{ sepia: ['', V, T, k] }],
      'backdrop-filter': [{ 'backdrop-filter': ['', 'none', T, k] }],
      'backdrop-blur': [{ 'backdrop-blur': nr() }],
      'backdrop-brightness': [{ 'backdrop-brightness': [V, T, k] }],
      'backdrop-contrast': [{ 'backdrop-contrast': [V, T, k] }],
      'backdrop-grayscale': [{ 'backdrop-grayscale': ['', V, T, k] }],
      'backdrop-hue-rotate': [{ 'backdrop-hue-rotate': [V, T, k] }],
      'backdrop-invert': [{ 'backdrop-invert': ['', V, T, k] }],
      'backdrop-opacity': [{ 'backdrop-opacity': [V, T, k] }],
      'backdrop-saturate': [{ 'backdrop-saturate': [V, T, k] }],
      'backdrop-sepia': [{ 'backdrop-sepia': ['', V, T, k] }],
      'border-collapse': [{ border: ['collapse', 'separate'] }],
      'border-spacing': [{ 'border-spacing': L() }],
      'border-spacing-x': [{ 'border-spacing-x': L() }],
      'border-spacing-y': [{ 'border-spacing-y': L() }],
      'table-layout': [{ table: ['auto', 'fixed'] }],
      caption: [{ caption: ['top', 'bottom'] }],
      transition: [
        { transition: ['', 'all', 'colors', 'opacity', 'shadow', 'transform', 'none', T, k] },
      ],
      'transition-behavior': [{ transition: ['normal', 'discrete'] }],
      duration: [{ duration: [V, 'initial', T, k] }],
      ease: [{ ease: ['linear', 'initial', m, T, k] }],
      delay: [{ delay: [V, T, k] }],
      animate: [{ animate: ['none', p, T, k] }],
      backface: [{ backface: ['hidden', 'visible'] }],
      perspective: [{ perspective: [x, T, k] }],
      'perspective-origin': [{ 'perspective-origin': R() }],
      rotate: [{ rotate: ka() }],
      'rotate-x': [{ 'rotate-x': ka() }],
      'rotate-y': [{ 'rotate-y': ka() }],
      'rotate-z': [{ 'rotate-z': ka() }],
      scale: [{ scale: ca() }],
      'scale-x': [{ 'scale-x': ca() }],
      'scale-y': [{ 'scale-y': ca() }],
      'scale-z': [{ 'scale-z': ca() }],
      'scale-3d': ['scale-3d'],
      skew: [{ skew: Ma() }],
      'skew-x': [{ 'skew-x': Ma() }],
      'skew-y': [{ 'skew-y': Ma() }],
      transform: [{ transform: [T, k, '', 'none', 'gpu', 'cpu'] }],
      'transform-origin': [{ origin: R() }],
      'transform-style': [{ transform: ['3d', 'flat'] }],
      translate: [{ translate: lr() }],
      'translate-x': [{ 'translate-x': lr() }],
      'translate-y': [{ 'translate-y': lr() }],
      'translate-z': [{ 'translate-z': lr() }],
      'translate-none': ['translate-none'],
      zoom: [{ zoom: [wa, T, k] }],
      accent: [{ accent: B() }],
      appearance: [{ appearance: ['none', 'auto'] }],
      'caret-color': [{ caret: B() }],
      'color-scheme': [
        { scheme: ['normal', 'dark', 'light', 'light-dark', 'only-dark', 'only-light'] },
      ],
      cursor: [
        {
          cursor: [
            'auto',
            'default',
            'pointer',
            'wait',
            'text',
            'move',
            'help',
            'not-allowed',
            'none',
            'context-menu',
            'progress',
            'cell',
            'crosshair',
            'vertical-text',
            'alias',
            'copy',
            'no-drop',
            'grab',
            'grabbing',
            'all-scroll',
            'col-resize',
            'row-resize',
            'n-resize',
            'e-resize',
            's-resize',
            'w-resize',
            'ne-resize',
            'nw-resize',
            'se-resize',
            'sw-resize',
            'ew-resize',
            'ns-resize',
            'nesw-resize',
            'nwse-resize',
            'zoom-in',
            'zoom-out',
            T,
            k,
          ],
        },
      ],
      'field-sizing': [{ 'field-sizing': ['fixed', 'content'] }],
      'pointer-events': [{ 'pointer-events': ['auto', 'none'] }],
      resize: [{ resize: ['none', '', 'y', 'x'] }],
      'scroll-behavior': [{ scroll: ['auto', 'smooth'] }],
      'scrollbar-thumb-color': [{ 'scrollbar-thumb': B() }],
      'scrollbar-track-color': [{ 'scrollbar-track': B() }],
      'scrollbar-gutter': [{ 'scrollbar-gutter': ['auto', 'stable', 'both'] }],
      'scrollbar-w': [{ scrollbar: ['auto', 'thin', 'none'] }],
      'scroll-m': [{ 'scroll-m': L() }],
      'scroll-mx': [{ 'scroll-mx': L() }],
      'scroll-my': [{ 'scroll-my': L() }],
      'scroll-ms': [{ 'scroll-ms': L() }],
      'scroll-me': [{ 'scroll-me': L() }],
      'scroll-mbs': [{ 'scroll-mbs': L() }],
      'scroll-mbe': [{ 'scroll-mbe': L() }],
      'scroll-mt': [{ 'scroll-mt': L() }],
      'scroll-mr': [{ 'scroll-mr': L() }],
      'scroll-mb': [{ 'scroll-mb': L() }],
      'scroll-ml': [{ 'scroll-ml': L() }],
      'scroll-p': [{ 'scroll-p': L() }],
      'scroll-px': [{ 'scroll-px': L() }],
      'scroll-py': [{ 'scroll-py': L() }],
      'scroll-ps': [{ 'scroll-ps': L() }],
      'scroll-pe': [{ 'scroll-pe': L() }],
      'scroll-pbs': [{ 'scroll-pbs': L() }],
      'scroll-pbe': [{ 'scroll-pbe': L() }],
      'scroll-pt': [{ 'scroll-pt': L() }],
      'scroll-pr': [{ 'scroll-pr': L() }],
      'scroll-pb': [{ 'scroll-pb': L() }],
      'scroll-pl': [{ 'scroll-pl': L() }],
      'snap-align': [{ snap: ['start', 'end', 'center', 'align-none'] }],
      'snap-stop': [{ snap: ['normal', 'always'] }],
      'snap-type': [{ snap: ['none', 'x', 'y', 'both'] }],
      'snap-strictness': [{ snap: ['mandatory', 'proximity'] }],
      touch: [{ touch: ['auto', 'none', 'manipulation'] }],
      'touch-x': [{ 'touch-pan': ['x', 'left', 'right'] }],
      'touch-y': [{ 'touch-pan': ['y', 'up', 'down'] }],
      'touch-pz': ['touch-pinch-zoom'],
      select: [{ select: ['none', 'text', 'all', 'auto'] }],
      'will-change': [{ 'will-change': ['auto', 'scroll', 'contents', 'transform', T, k] }],
      fill: [{ fill: ['none', ...B()] }],
      'stroke-w': [{ stroke: [V, xi, xo, bL] }],
      stroke: [{ stroke: ['none', ...B()] }],
      'forced-color-adjust': [{ 'forced-color-adjust': ['auto', 'none'] }],
    },
    conflictingClassGroups: {
      'container-named': ['container-type'],
      overflow: ['overflow-x', 'overflow-y'],
      overscroll: ['overscroll-x', 'overscroll-y'],
      inset: [
        'inset-x',
        'inset-y',
        'inset-bs',
        'inset-be',
        'start',
        'end',
        'top',
        'right',
        'bottom',
        'left',
      ],
      'inset-x': ['right', 'left'],
      'inset-y': ['top', 'bottom'],
      flex: ['basis', 'grow', 'shrink'],
      gap: ['gap-x', 'gap-y'],
      p: ['px', 'py', 'ps', 'pe', 'pbs', 'pbe', 'pt', 'pr', 'pb', 'pl'],
      px: ['pr', 'pl'],
      py: ['pt', 'pb'],
      m: ['mx', 'my', 'ms', 'me', 'mbs', 'mbe', 'mt', 'mr', 'mb', 'ml'],
      mx: ['mr', 'ml'],
      my: ['mt', 'mb'],
      size: ['w', 'h'],
      'font-size': ['leading'],
      'fvn-normal': [
        'fvn-ordinal',
        'fvn-slashed-zero',
        'fvn-figure',
        'fvn-spacing',
        'fvn-fraction',
      ],
      'fvn-ordinal': ['fvn-normal'],
      'fvn-slashed-zero': ['fvn-normal'],
      'fvn-figure': ['fvn-normal'],
      'fvn-spacing': ['fvn-normal'],
      'fvn-fraction': ['fvn-normal'],
      'line-clamp': ['display', 'overflow'],
      rounded: [
        'rounded-s',
        'rounded-e',
        'rounded-t',
        'rounded-r',
        'rounded-b',
        'rounded-l',
        'rounded-ss',
        'rounded-se',
        'rounded-ee',
        'rounded-es',
        'rounded-tl',
        'rounded-tr',
        'rounded-br',
        'rounded-bl',
      ],
      'rounded-s': ['rounded-ss', 'rounded-es'],
      'rounded-e': ['rounded-se', 'rounded-ee'],
      'rounded-t': ['rounded-tl', 'rounded-tr'],
      'rounded-r': ['rounded-tr', 'rounded-br'],
      'rounded-b': ['rounded-br', 'rounded-bl'],
      'rounded-l': ['rounded-tl', 'rounded-bl'],
      'border-spacing': ['border-spacing-x', 'border-spacing-y'],
      'border-w': [
        'border-w-x',
        'border-w-y',
        'border-w-s',
        'border-w-e',
        'border-w-bs',
        'border-w-be',
        'border-w-t',
        'border-w-r',
        'border-w-b',
        'border-w-l',
      ],
      'border-w-x': ['border-w-r', 'border-w-l'],
      'border-w-y': ['border-w-t', 'border-w-b'],
      'border-color': [
        'border-color-x',
        'border-color-y',
        'border-color-s',
        'border-color-e',
        'border-color-bs',
        'border-color-be',
        'border-color-t',
        'border-color-r',
        'border-color-b',
        'border-color-l',
      ],
      'border-color-x': ['border-color-r', 'border-color-l'],
      'border-color-y': ['border-color-t', 'border-color-b'],
      translate: ['translate-x', 'translate-y', 'translate-none'],
      'translate-none': ['translate', 'translate-x', 'translate-y', 'translate-z'],
      'scroll-m': [
        'scroll-mx',
        'scroll-my',
        'scroll-ms',
        'scroll-me',
        'scroll-mbs',
        'scroll-mbe',
        'scroll-mt',
        'scroll-mr',
        'scroll-mb',
        'scroll-ml',
      ],
      'scroll-mx': ['scroll-mr', 'scroll-ml'],
      'scroll-my': ['scroll-mt', 'scroll-mb'],
      'scroll-p': [
        'scroll-px',
        'scroll-py',
        'scroll-ps',
        'scroll-pe',
        'scroll-pbs',
        'scroll-pbe',
        'scroll-pt',
        'scroll-pr',
        'scroll-pb',
        'scroll-pl',
      ],
      'scroll-px': ['scroll-pr', 'scroll-pl'],
      'scroll-py': ['scroll-pt', 'scroll-pb'],
      touch: ['touch-x', 'touch-y', 'touch-pz'],
      'touch-x': ['touch'],
      'touch-y': ['touch'],
      'touch-pz': ['touch'],
    },
    conflictingClassGroupModifiers: { 'font-size': ['leading'] },
    postfixLookupClassGroups: ['container-type'],
    orderSensitiveModifiers: [
      '*',
      '**',
      'after',
      'backdrop',
      'before',
      'details-content',
      'file',
      'first-letter',
      'first-line',
      'marker',
      'placeholder',
      'selection',
    ],
  };
};
var BL = Y2(pM);
function re(...e) {
  return BL(Vm(e));
}
var tr = E($());
function UL({ columns: e = 4, rows: t = 6, className: a }) {
  return (0, tr.jsxs)('div', {
    'aria-busy': 'true',
    'aria-label': 'Loading table',
    className: re(
      'motion-safe:animate-pulse overflow-hidden rounded-md border border-border bg-card',
      a
    ),
    children: [
      (0, tr.jsx)('div', {
        className: 'flex h-10 border-b border-border/40',
        children: Array.from({ length: e }, (r, o) =>
          (0, tr.jsx)(
            'div',
            {
              className: 'flex flex-1 items-center px-2',
              children: (0, tr.jsx)('div', { className: 'h-3 w-20 rounded bg-muted' }),
            },
            `head-${o}`
          )
        ),
      }),
      Array.from({ length: t }, (r, o) =>
        (0, tr.jsx)(
          'div',
          {
            className: 'flex border-b border-border/40 last:border-0',
            children: Array.from({ length: e }, (n, l) =>
              (0, tr.jsx)(
                'div',
                {
                  className: 'flex flex-1 items-center p-2',
                  children: (0, tr.jsx)('div', {
                    className: re(
                      'h-3 rounded bg-muted',
                      l === 0 ? 'w-32' : l === e - 1 ? 'w-16' : 'w-24'
                    ),
                  }),
                },
                `cell-${o}-${l}`
              )
            ),
          },
          `row-${o}`
        )
      ),
    ],
  });
}
var Fn = E($());
function qn({ variant: e = 'page', columns: t = 4, rows: a = 6 }) {
  return e === 'directory'
    ? (0, Fn.jsx)(UL, { columns: t, rows: a })
    : (0, Fn.jsxs)('div', {
        className: 'motion-safe:animate-pulse grid gap-4 p-6',
        'aria-busy': 'true',
        'aria-label': 'Loading',
        children: [
          (0, Fn.jsx)('div', { className: 'h-8 w-48 rounded bg-muted' }),
          (0, Fn.jsx)('div', { className: 'h-64 rounded bg-muted' }),
        ],
      });
}
var NL = E(te());
function Gn() {
  let e = (0, NL.useContext)(Gm);
  if (!e) throw new Error('useMeta must be used within MetaProvider');
  return e;
}
var ko = E(te());
var YL = E(te());
var Xu = E(te());
var HL = (e) => e.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase(),
  hM = (e) =>
    e.replace(/^([A-Z])|[\s-_]+(\w)/g, (t, a, r) => (r ? r.toUpperCase() : a.toLowerCase())),
  Wm = (e) => {
    let t = hM(e);
    return t.charAt(0).toUpperCase() + t.slice(1);
  },
  Yu = (...e) =>
    e
      .filter((t, a, r) => !!t && t.trim() !== '' && r.indexOf(t) === a)
      .join(' ')
      .trim(),
  zL = (e) => {
    for (let t in e) if (t.startsWith('aria-') || t === 'role' || t === 'title') return !0;
  };
var Si = E(te());
var FL = {
  xmlns: 'http://www.w3.org/2000/svg',
  width: 24,
  height: 24,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
};
var qL = (0, Si.forwardRef)(
  (
    {
      color: e = 'currentColor',
      size: t = 24,
      strokeWidth: a = 2,
      absoluteStrokeWidth: r,
      className: o = '',
      children: n,
      iconNode: l,
      ...i
    },
    s
  ) =>
    (0, Si.createElement)(
      'svg',
      {
        ref: s,
        ...FL,
        width: t,
        height: t,
        stroke: e,
        strokeWidth: r ? (Number(a) * 24) / Number(t) : a,
        className: Yu('lucide', o),
        ...(!n && !zL(i) && { 'aria-hidden': 'true' }),
        ...i,
      },
      [...l.map(([u, d]) => (0, Si.createElement)(u, d)), ...(Array.isArray(n) ? n : [n])]
    )
);
var S = (e, t) => {
  let a = (0, Xu.forwardRef)(({ className: r, ...o }, n) =>
    (0, Xu.createElement)(qL, {
      ref: n,
      iconNode: t,
      className: Yu(`lucide-${HL(Wm(e))}`, `lucide-${e}`, r),
      ...o,
    })
  );
  return (a.displayName = Wm(e)), a;
};
var gM = [
    [
      'path',
      {
        d: 'M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2',
        key: '169zse',
      },
    ],
  ],
  Qm = S('activity', gM);
var yM = [
    ['rect', { x: '2', y: '4', width: '20', height: '16', rx: '2', key: 'izxlao' }],
    ['path', { d: 'M10 4v4', key: 'pp8u80' }],
    ['path', { d: 'M2 8h20', key: 'd11cs7' }],
    ['path', { d: 'M6 4v4', key: '1svtjw' }],
  ],
  Km = S('app-window', yM);
var vM = [
    ['path', { d: 'M12 5v14', key: 's699le' }],
    ['path', { d: 'm19 12-7 7-7-7', key: '1idqje' }],
  ],
  Zm = S('arrow-down', vM);
var bM = [
    ['path', { d: 'm21 16-4 4-4-4', key: 'f6ql7i' }],
    ['path', { d: 'M17 20V4', key: '1ejh1v' }],
    ['path', { d: 'm3 8 4-4 4 4', key: '11wl7u' }],
    ['path', { d: 'M7 4v16', key: '1glfcx' }],
  ],
  Jm = S('arrow-up-down', bM);
var xM = [
    ['path', { d: 'm5 12 7-7 7 7', key: 'hav0vg' }],
    ['path', { d: 'M12 19V5', key: 'x0mq9r' }],
  ],
  $m = S('arrow-up', xM);
var SM = [
    ['path', { d: 'M10.268 21a2 2 0 0 0 3.464 0', key: 'vwvbt9' }],
    [
      'path',
      {
        d: 'M3.262 15.326A1 1 0 0 0 4 17h16a1 1 0 0 0 .74-1.673C19.41 13.956 18 12.499 18 8A6 6 0 0 0 6 8c0 4.499-1.411 5.956-2.738 7.326',
        key: '11g9vi',
      },
    ],
  ],
  ep = S('bell', SM);
var LM = [
    ['path', { d: 'M12 7v14', key: '1akyts' }],
    [
      'path',
      {
        d: 'M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z',
        key: 'ruj8y',
      },
    ],
  ],
  tp = S('book-open', LM);
var CM = [
    ['path', { d: 'M12 18V5', key: 'adv99a' }],
    ['path', { d: 'M15 13a4.17 4.17 0 0 1-3-4 4.17 4.17 0 0 1-3 4', key: '1e3is1' }],
    ['path', { d: 'M17.598 6.5A3 3 0 1 0 12 5a3 3 0 1 0-5.598 1.5', key: '1gqd8o' }],
    ['path', { d: 'M17.997 5.125a4 4 0 0 1 2.526 5.77', key: 'iwvgf7' }],
    ['path', { d: 'M18 18a4 4 0 0 0 2-7.464', key: 'efp6ie' }],
    ['path', { d: 'M19.967 17.483A4 4 0 1 1 12 18a4 4 0 1 1-7.967-.517', key: '1gq6am' }],
    ['path', { d: 'M6 18a4 4 0 0 1-2-7.464', key: 'k1g0md' }],
    ['path', { d: 'M6.003 5.125a4 4 0 0 0-2.526 5.77', key: 'q97ue3' }],
  ],
  ap = S('brain', CM);
var wM = [
    ['path', { d: 'M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z', key: '1b4qmf' }],
    ['path', { d: 'M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2', key: 'i71pzd' }],
    ['path', { d: 'M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2', key: '10jefs' }],
    ['path', { d: 'M10 6h4', key: '1itunk' }],
    ['path', { d: 'M10 10h4', key: 'tcdvrf' }],
    ['path', { d: 'M10 14h4', key: 'kelpxr' }],
    ['path', { d: 'M10 18h4', key: '1ulq68' }],
  ],
  rp = S('building-2', wM);
var RM = [
    ['path', { d: 'M8 2v4', key: '1cmpym' }],
    ['path', { d: 'M16 2v4', key: '4m81vk' }],
    ['rect', { width: '18', height: '18', x: '3', y: '4', rx: '2', key: '1hopcy' }],
    ['path', { d: 'M3 10h18', key: '8toen8' }],
  ],
  op = S('calendar', RM);
var _M = [
    ['path', { d: 'M3 3v16a2 2 0 0 0 2 2h16', key: 'c24i48' }],
    ['path', { d: 'M18 17V9', key: '2bz60n' }],
    ['path', { d: 'M13 17V5', key: '1frdt8' }],
    ['path', { d: 'M8 17v-3', key: '17ska0' }],
  ],
  Vn = S('chart-column', _M);
var IM = [['path', { d: 'M20 6 9 17l-5-5', key: '1gmf2c' }]],
  np = S('check', IM);
var kM = [['path', { d: 'm6 9 6 6 6-6', key: 'qrunsl' }]],
  lp = S('chevron-down', kM);
var MM = [['path', { d: 'm15 18-6-6 6-6', key: '1wnfg3' }]],
  ip = S('chevron-left', MM);
var EM = [['path', { d: 'm9 18 6-6-6-6', key: 'mthhwq' }]],
  sp = S('chevron-right', EM);
var AM = [
    ['path', { d: 'm7 15 5 5 5-5', key: '1hf1tw' }],
    ['path', { d: 'm7 9 5-5 5 5', key: 'sgt6xg' }],
  ],
  up = S('chevrons-up-down', AM);
var TM = [
    ['rect', { width: '18', height: '18', x: '3', y: '3', rx: '2', key: 'afitv7' }],
    ['path', { d: 'M9 3v18', key: 'fh3hqa' }],
    ['path', { d: 'M15 3v18', key: '14nvp0' }],
  ],
  jn = S('columns-3', TM);
var DM = [
    ['rect', { width: '14', height: '14', x: '8', y: '8', rx: '2', ry: '2', key: '17jyea' }],
    ['path', { d: 'M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2', key: 'zix9uf' }],
  ],
  cp = S('copy', DM);
var OM = [
    ['circle', { cx: '12', cy: '12', r: '1', key: '41hilf' }],
    ['circle', { cx: '19', cy: '12', r: '1', key: '1wjl8i' }],
    ['circle', { cx: '5', cy: '12', r: '1', key: '1pcz8c' }],
  ],
  Yn = S('ellipsis', OM);
var PM = [
    ['path', { d: 'M14 2v4a2 2 0 0 0 2 2h4', key: 'tnqrlb' }],
    [
      'path',
      { d: 'M4.268 21a2 2 0 0 0 1.727 1H18a2 2 0 0 0 2-2V7l-5-5H6a2 2 0 0 0-2 2v3', key: 'ms7g94' },
    ],
    ['path', { d: 'm9 18-1.5-1.5', key: '1j6qii' }],
    ['circle', { cx: '5', cy: '14', r: '3', key: 'ufru5t' }],
  ],
  dp = S('file-search', PM);
var BM = [
    ['path', { d: 'm12 14 4-4', key: '9kzdfg' }],
    ['path', { d: 'M3.34 19a10 10 0 1 1 17.32 0', key: '19p75a' }],
  ],
  fp = S('gauge', BM);
var UM = [
    ['path', { d: 'm14 13-8.381 8.38a1 1 0 0 1-3.001-3l8.384-8.381', key: 'pgg06f' }],
    ['path', { d: 'm16 16 6-6', key: 'vzrcl6' }],
    ['path', { d: 'm21.5 10.5-8-8', key: 'a17d9x' }],
    ['path', { d: 'm8 8 6-6', key: '18bi4p' }],
    ['path', { d: 'm8.5 7.5 8 8', key: '1oyaui' }],
  ],
  mp = S('gavel', UM);
var NM = [
    ['line', { x1: '6', x2: '6', y1: '3', y2: '15', key: '17qcm7' }],
    ['circle', { cx: '18', cy: '6', r: '3', key: '1h7g24' }],
    ['circle', { cx: '6', cy: '18', r: '3', key: 'fqmcym' }],
    ['path', { d: 'M18 9a9 9 0 0 1-9 9', key: 'n2h4wq' }],
  ],
  pp = S('git-branch', NM);
var HM = [
    ['circle', { cx: '12', cy: '12', r: '10', key: '1mglay' }],
    ['path', { d: 'M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20', key: '13o1zl' }],
    ['path', { d: 'M2 12h20', key: '9i4pu4' }],
  ],
  hp = S('globe', HM);
var zM = [
    ['circle', { cx: '9', cy: '12', r: '1', key: '1vctgf' }],
    ['circle', { cx: '9', cy: '5', r: '1', key: 'hp0tcf' }],
    ['circle', { cx: '9', cy: '19', r: '1', key: 'fkjjf6' }],
    ['circle', { cx: '15', cy: '12', r: '1', key: '1tmaij' }],
    ['circle', { cx: '15', cy: '5', r: '1', key: '19l28e' }],
    ['circle', { cx: '15', cy: '19', r: '1', key: 'f4zoj3' }],
  ],
  gp = S('grip-vertical', zM);
var FM = [
    ['line', { x1: '4', x2: '20', y1: '9', y2: '9', key: '4lhtct' }],
    ['line', { x1: '4', x2: '20', y1: '15', y2: '15', key: 'vyu0kd' }],
    ['line', { x1: '10', x2: '8', y1: '3', y2: '21', key: '1ggp8o' }],
    ['line', { x1: '16', x2: '14', y1: '3', y2: '21', key: 'weycgp' }],
  ],
  yp = S('hash', FM);
var qM = [
    ['polyline', { points: '22 12 16 12 14 15 10 15 8 12 2 12', key: 'o97t9d' }],
    [
      'path',
      {
        d: 'M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z',
        key: 'oot6mr',
      },
    ],
  ],
  vp = S('inbox', qM);
var GM = [
    [
      'path',
      {
        d: 'M12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83z',
        key: 'zw3jo',
      },
    ],
    [
      'path',
      {
        d: 'M2 12a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 12',
        key: '1wduqc',
      },
    ],
    [
      'path',
      {
        d: 'M2 17a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 17',
        key: 'kqbvx6',
      },
    ],
  ],
  Xn = S('layers', GM);
var VM = [
    ['rect', { width: '7', height: '9', x: '3', y: '3', rx: '1', key: '10lvy0' }],
    ['rect', { width: '7', height: '5', x: '14', y: '3', rx: '1', key: '16une8' }],
    ['rect', { width: '7', height: '9', x: '14', y: '12', rx: '1', key: '1hutg5' }],
    ['rect', { width: '7', height: '5', x: '3', y: '16', rx: '1', key: 'ldoo1y' }],
  ],
  bp = S('layout-dashboard', VM);
var jM = [
    ['rect', { width: '18', height: '7', x: '3', y: '3', rx: '1', key: 'f1a2em' }],
    ['rect', { width: '9', height: '7', x: '3', y: '14', rx: '1', key: 'jqznyg' }],
    ['rect', { width: '5', height: '7', x: '16', y: '14', rx: '1', key: 'q5h2i8' }],
  ],
  xp = S('layout-template', jM);
var YM = [
    ['path', { d: 'M9 17H7A5 5 0 0 1 7 7h2', key: '8i5ue5' }],
    ['path', { d: 'M15 7h2a5 5 0 1 1 0 10h-2', key: '1b9ql8' }],
    ['line', { x1: '8', x2: '16', y1: '12', y2: '12', key: '1jonct' }],
  ],
  Sp = S('link-2', YM);
var XM = [
    ['path', { d: 'M21 12h-8', key: '1bmf0i' }],
    ['path', { d: 'M21 6H8', key: '1pqkrb' }],
    ['path', { d: 'M21 18h-8', key: '1tm79t' }],
    ['path', { d: 'M3 6v4c0 1.1.9 2 2 2h3', key: '1ywdgy' }],
    ['path', { d: 'M3 10v6c0 1.1.9 2 2 2h3', key: '2wc746' }],
  ],
  Lp = S('list-tree', XM);
var WM = [['path', { d: 'M21 12a9 9 0 1 1-6.219-8.56', key: '13zald' }]],
  zr = S('loader-circle', WM);
var QM = [
    [
      'path',
      {
        d: 'M11 6a13 13 0 0 0 8.4-2.8A1 1 0 0 1 21 4v12a1 1 0 0 1-1.6.8A13 13 0 0 0 11 14H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2z',
        key: 'q8bfy3',
      },
    ],
    ['path', { d: 'M6 14a12 12 0 0 0 2.4 7.2 2 2 0 0 0 3.2-2.4A8 8 0 0 1 10 14', key: '1853fq' }],
    ['path', { d: 'M8 6v8', key: '15ugcq' }],
  ],
  Cp = S('megaphone', QM);
var KM = [
    [
      'path',
      {
        d: 'M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401',
        key: 'kfwtm',
      },
    ],
  ],
  wp = S('moon', KM);
var ZM = [
    [
      'path',
      {
        d: 'M12 22a1 1 0 0 1 0-20 10 9 0 0 1 10 9 5 5 0 0 1-5 5h-2.25a1.75 1.75 0 0 0-1.4 2.8l.3.4a1.75 1.75 0 0 1-1.4 2.8z',
        key: 'e79jfc',
      },
    ],
    ['circle', { cx: '13.5', cy: '6.5', r: '.5', fill: 'currentColor', key: '1okk4w' }],
    ['circle', { cx: '17.5', cy: '10.5', r: '.5', fill: 'currentColor', key: 'f64h9f' }],
    ['circle', { cx: '6.5', cy: '12.5', r: '.5', fill: 'currentColor', key: 'qy21gx' }],
    ['circle', { cx: '8.5', cy: '7.5', r: '.5', fill: 'currentColor', key: 'fotxhn' }],
  ],
  Rp = S('palette', ZM);
var JM = [
    ['rect', { width: '18', height: '18', x: '3', y: '3', rx: '2', key: 'afitv7' }],
    ['path', { d: 'M9 3v18', key: 'fh3hqa' }],
  ],
  Wn = S('panel-left', JM);
var $M = [
    ['path', { d: 'M12 22v-5', key: '1ega77' }],
    ['path', { d: 'M9 8V2', key: '14iosj' }],
    ['path', { d: 'M15 8V2', key: '18g5xt' }],
    ['path', { d: 'M18 8v5a4 4 0 0 1-4 4h-4a4 4 0 0 1-4-4V8Z', key: 'osxo6l' }],
  ],
  _p = S('plug', $M);
var eE = [
    ['path', { d: 'M5 12h14', key: '1ays0h' }],
    ['path', { d: 'M12 5v14', key: 's699le' }],
  ],
  Ip = S('plus', eE);
var tE = [
    [
      'path',
      {
        d: 'M4 2v20l2-1 2 1 2-1 2 1 2-1 2 1 2-1 2 1V2l-2 1-2-1-2 1-2-1-2 1-2-1-2 1Z',
        key: 'q3az6g',
      },
    ],
    ['path', { d: 'M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8', key: '1h4pet' }],
    ['path', { d: 'M12 17.5v-11', key: '1jc1ny' }],
  ],
  kp = S('receipt', tE);
var aE = [
    ['path', { d: 'M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8', key: 'v9h5vc' }],
    ['path', { d: 'M21 3v5h-5', key: '1q7to0' }],
    ['path', { d: 'M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16', key: '3uifl3' }],
    ['path', { d: 'M8 16H3v5', key: '1cv678' }],
  ],
  Mp = S('refresh-cw', aE);
var rE = [
    ['circle', { cx: '6', cy: '19', r: '3', key: '1kj8tv' }],
    ['path', { d: 'M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15', key: '1d8sl' }],
    ['circle', { cx: '18', cy: '5', r: '3', key: 'gq8acd' }],
  ],
  Ep = S('route', rE);
var oE = [
    ['path', { d: 'm16 16 3-8 3 8c-.87.65-1.92 1-3 1s-2.13-.35-3-1Z', key: '7g6ntu' }],
    ['path', { d: 'm2 16 3-8 3 8c-.87.65-1.92 1-3 1s-2.13-.35-3-1Z', key: 'ijws7r' }],
    ['path', { d: 'M7 21h10', key: '1b0cd5' }],
    ['path', { d: 'M12 3v18', key: '108xh3' }],
    ['path', { d: 'M3 7h2c2 0 5-1 7-2 2 1 5 2 7 2h2', key: '3gwbw2' }],
  ],
  Ap = S('scale', oE);
var nE = [
    ['path', { d: 'M15 12h-5', key: 'r7krc0' }],
    ['path', { d: 'M15 8h-5', key: '1khuty' }],
    ['path', { d: 'M19 17V5a2 2 0 0 0-2-2H4', key: 'zz82l3' }],
    [
      'path',
      {
        d: 'M8 21h12a2 2 0 0 0 2-2v-1a1 1 0 0 0-1-1H11a1 1 0 0 0-1 1v1a2 2 0 1 1-4 0V5a2 2 0 1 0-4 0v2a1 1 0 0 0 1 1h3',
        key: '1ph1d7',
      },
    ],
  ],
  Tp = S('scroll-text', nE);
var lE = [
    ['path', { d: 'm21 21-4.34-4.34', key: '14j7rj' }],
    ['circle', { cx: '11', cy: '11', r: '8', key: '4ej97u' }],
  ],
  Dp = S('search', lE);
var iE = [
    ['path', { d: 'M14 17H5', key: 'gfn3mx' }],
    ['path', { d: 'M19 7h-9', key: '6i9tg' }],
    ['circle', { cx: '17', cy: '17', r: '3', key: '18b49y' }],
    ['circle', { cx: '7', cy: '7', r: '3', key: 'dfmy0x' }],
  ],
  Op = S('settings-2', iE);
var sE = [
    [
      'path',
      {
        d: 'M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915',
        key: '1i5ecw',
      },
    ],
    ['circle', { cx: '12', cy: '12', r: '3', key: '1v7zrd' }],
  ],
  Pp = S('settings', sE);
var uE = [
    ['circle', { cx: '18', cy: '5', r: '3', key: 'gq8acd' }],
    ['circle', { cx: '6', cy: '12', r: '3', key: 'w7nqdw' }],
    ['circle', { cx: '18', cy: '19', r: '3', key: '1xt0gg' }],
    ['line', { x1: '8.59', x2: '15.42', y1: '13.51', y2: '17.49', key: '47mynk' }],
    ['line', { x1: '15.41', x2: '8.59', y1: '6.51', y2: '10.49', key: '1n3mei' }],
  ],
  Bp = S('share-2', uE);
var cE = [
    [
      'path',
      {
        d: 'M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z',
        key: 'oel41y',
      },
    ],
    ['path', { d: 'M12 8v4', key: '1got3b' }],
    ['path', { d: 'M12 16h.01', key: '1drbdi' }],
  ],
  Up = S('shield-alert', cE);
var dE = [
    [
      'path',
      {
        d: 'M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z',
        key: 'oel41y',
      },
    ],
  ],
  Np = S('shield', dE);
var fE = [
    [
      'path',
      {
        d: 'M18 7V5a1 1 0 0 0-1-1H6.5a.5.5 0 0 0-.4.8l4.5 6a2 2 0 0 1 0 2.4l-4.5 6a.5.5 0 0 0 .4.8H17a1 1 0 0 0 1-1v-2',
        key: 'wuwx1p',
      },
    ],
  ],
  Hp = S('sigma', fE);
var mE = [
    ['line', { x1: '21', x2: '14', y1: '4', y2: '4', key: 'obuewd' }],
    ['line', { x1: '10', x2: '3', y1: '4', y2: '4', key: '1q6298' }],
    ['line', { x1: '21', x2: '12', y1: '12', y2: '12', key: '1iu8h1' }],
    ['line', { x1: '8', x2: '3', y1: '12', y2: '12', key: 'ntss68' }],
    ['line', { x1: '21', x2: '16', y1: '20', y2: '20', key: '14d8ph' }],
    ['line', { x1: '12', x2: '3', y1: '20', y2: '20', key: 'm0wm8r' }],
    ['line', { x1: '14', x2: '14', y1: '2', y2: '6', key: '14e1ph' }],
    ['line', { x1: '8', x2: '8', y1: '10', y2: '14', key: '1i6ji0' }],
    ['line', { x1: '16', x2: '16', y1: '18', y2: '22', key: '1lctlv' }],
  ],
  zp = S('sliders-horizontal', mE);
var pE = [
    [
      'path',
      {
        d: 'M11.017 2.814a1 1 0 0 1 1.966 0l1.051 5.558a2 2 0 0 0 1.594 1.594l5.558 1.051a1 1 0 0 1 0 1.966l-5.558 1.051a2 2 0 0 0-1.594 1.594l-1.051 5.558a1 1 0 0 1-1.966 0l-1.051-5.558a2 2 0 0 0-1.594-1.594l-5.558-1.051a1 1 0 0 1 0-1.966l5.558-1.051a2 2 0 0 0 1.594-1.594z',
        key: '1s2grr',
      },
    ],
    ['path', { d: 'M20 2v4', key: '1rf3ol' }],
    ['path', { d: 'M22 4h-4', key: 'gwowj6' }],
    ['circle', { cx: '4', cy: '20', r: '2', key: '6kqj1y' }],
  ],
  Qn = S('sparkles', pE);
var hE = [
    ['circle', { cx: '12', cy: '12', r: '4', key: '4exip2' }],
    ['path', { d: 'M12 2v2', key: 'tus03m' }],
    ['path', { d: 'M12 20v2', key: '1lh1kg' }],
    ['path', { d: 'm4.93 4.93 1.41 1.41', key: '149t6j' }],
    ['path', { d: 'm17.66 17.66 1.41 1.41', key: 'ptbguv' }],
    ['path', { d: 'M2 12h2', key: '1t8f8n' }],
    ['path', { d: 'M20 12h2', key: '1q8mjw' }],
    ['path', { d: 'm6.34 17.66-1.41 1.41', key: '1m8zz5' }],
    ['path', { d: 'm19.07 4.93-1.41 1.41', key: '1shlcs' }],
  ],
  Fp = S('sun', hE);
var gE = [
    [
      'path',
      {
        d: 'M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z',
        key: 'vktsd0',
      },
    ],
    ['circle', { cx: '7.5', cy: '7.5', r: '.5', fill: 'currentColor', key: 'kqv944' }],
  ],
  qp = S('tag', gE);
var yE = [
    [
      'path',
      {
        d: 'M13.172 2a2 2 0 0 1 1.414.586l6.71 6.71a2.4 2.4 0 0 1 0 3.408l-4.592 4.592a2.4 2.4 0 0 1-3.408 0l-6.71-6.71A2 2 0 0 1 6 9.172V3a1 1 0 0 1 1-1z',
        key: '16rjxf',
      },
    ],
    [
      'path',
      { d: 'M2 7v6.172a2 2 0 0 0 .586 1.414l6.71 6.71a2.4 2.4 0 0 0 3.191.193', key: '178nd4' },
    ],
    ['circle', { cx: '10.5', cy: '6.5', r: '.5', fill: 'currentColor', key: '12ikhr' }],
  ],
  Gp = S('tags', yE);
var vE = [
    ['circle', { cx: '9', cy: '12', r: '3', key: 'u3jwor' }],
    ['rect', { width: '20', height: '14', x: '2', y: '5', rx: '7', key: 'g7kal2' }],
  ],
  Vp = S('toggle-left', vE);
var bE = [
    [
      'path',
      {
        d: 'm21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3',
        key: 'wmoenq',
      },
    ],
    ['path', { d: 'M12 9v4', key: 'juzpu7' }],
    ['path', { d: 'M12 17h.01', key: 'p32p05' }],
  ],
  Kn = S('triangle-alert', bE);
var xE = [
    ['path', { d: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2', key: '1yyitq' }],
    ['path', { d: 'M16 3.128a4 4 0 0 1 0 7.744', key: '16gr8j' }],
    ['path', { d: 'M22 21v-2a4 4 0 0 0-3-3.87', key: 'kshegd' }],
    ['circle', { cx: '9', cy: '7', r: '4', key: 'nufk8' }],
  ],
  jp = S('users', xE);
var SE = [
    ['rect', { width: '8', height: '8', x: '3', y: '3', rx: '2', key: 'by2w9f' }],
    ['path', { d: 'M7 11v4a2 2 0 0 0 2 2h4', key: 'xkn7yn' }],
    ['rect', { width: '8', height: '8', x: '13', y: '13', rx: '2', key: '1cgmvn' }],
  ],
  Yp = S('workflow', SE);
var LE = [
    [
      'path',
      {
        d: 'M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.106-3.105c.32-.322.863-.22.983.218a6 6 0 0 1-8.259 7.057l-7.91 7.91a1 1 0 0 1-2.999-3l7.91-7.91a6 6 0 0 1 7.057-8.259c.438.12.54.662.219.984z',
        key: '1ngwbx',
      },
    ],
  ],
  Xp = S('wrench', LE);
var CE = [
    ['path', { d: 'M18 6 6 18', key: '1bl5f8' }],
    ['path', { d: 'm6 6 12 12', key: 'd8bk6v' }],
  ],
  Wp = S('x', CE);
var wE = [
    [
      'path',
      {
        d: 'M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z',
        key: '1xq2db',
      },
    ],
  ],
  Qp = S('zap', wE);
var Qe = {
    controlRadius: 'rounded-sm',
    panelRadius: 'rounded-md',
    pillRadius: 'rounded-full',
    focusRing:
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
    controlHeight: 'min-h-7',
    controlText: 'text-[13px] leading-[18px]',
    buttonShell:
      'inline-flex h-7 shrink-0 items-center justify-center gap-2 py-0 text-[13px] leading-none [&_svg]:block [&_svg]:shrink-0',
    labelCaps:
      'text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
    tableHeader: 'text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground',
    tableRowHeight: 'h-[34px]',
  },
  CH = {
    active: 'border-transparent bg-admin-status-active/10 text-admin-status-active',
    paused: 'border-transparent bg-admin-status-paused/10 text-admin-status-paused',
    archived: 'border-border bg-muted text-muted-foreground',
    error: 'border-transparent bg-destructive/10 text-destructive',
    draft: 'border-transparent bg-admin-status-draft/10 text-admin-status-draft',
    scheduled: 'border-transparent bg-admin-status-scheduled/10 text-admin-status-scheduled',
    muted: 'border-border bg-muted/50 text-muted-foreground',
  },
  wH =
    'inline-flex max-w-full shrink-0 items-center whitespace-nowrap rounded-full border px-2 py-0.5 text-xs font-normal leading-4';
function RH(e, t) {
  if (t === 'success') return 'active';
  if (t === 'warning') return 'paused';
  if (t === 'muted') return 'archived';
  switch (e.trim().toUpperCase()) {
    case 'ACTIVE':
      return 'active';
    case 'PAUSED':
      return 'paused';
    case 'ARCHIVED':
      return 'archived';
    case 'DRAFT':
      return 'draft';
    case 'SCHEDULED':
      return 'scheduled';
    case 'ERROR':
    case 'FAILED':
      return 'error';
    default:
      return 'muted';
  }
}
var Wu = {
  control: GL(),
  controlFieldGroup: re(GL(), 'flex items-center gap-2'),
  controlFieldInset:
    'min-w-0 flex-1 border-0 bg-transparent p-0 text-foreground shadow-none outline-none placeholder:text-muted-foreground focus-visible:ring-0 focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50',
  controlGhost: re(
    Qe.buttonShell,
    Qe.controlRadius,
    'border border-transparent bg-transparent px-2 text-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
  ),
  panel: re(Qe.panelRadius, 'border border-border bg-card text-card-foreground'),
  panelMuted: re(Qe.panelRadius, 'bg-muted text-muted-foreground'),
  overlayBackdrop: 'fixed inset-0 z-50 bg-foreground/20 dark:bg-background/75',
  floating: re(
    'z-50 border border-border bg-popover text-popover-foreground p-1 shadow-lg',
    Qe.controlRadius
  ),
  menuList: 'flex flex-col gap-1',
  menuItem: re(
    'relative flex w-full cursor-pointer select-none items-center whitespace-nowrap px-2 py-1.5 text-[13px] text-foreground outline-none hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50',
    Qe.controlRadius
  ),
  tableHead:
    'h-[34px] bg-muted/50 px-4 text-left align-middle text-[11px] font-normal uppercase leading-[14px] text-muted-foreground',
  tableCell: 'px-4 py-0 align-middle text-[13px] leading-[18px] text-foreground',
  muted: 'text-muted-foreground',
  pageTitle: 'text-lg font-normal tracking-tight text-foreground',
};
function GL() {
  return [
    Qe.controlHeight,
    Qe.controlRadius,
    Qe.controlText,
    'border border-input bg-background px-2 py-1 text-foreground transition-colors',
    'placeholder:text-muted-foreground',
    Qe.focusRing,
    'disabled:cursor-not-allowed disabled:opacity-50',
    'aria-[invalid=true]:border-destructive aria-[invalid=true]:ring-destructive/30',
  ].join(' ');
}
var Kp = {
    default: 'border-primary bg-primary text-primary-foreground hover:bg-primary/90',
    brand:
      'border-admin-brand bg-admin-brand text-admin-brand-foreground hover:border-admin-brand-hover hover:bg-admin-brand-hover',
    secondary: 'border-border bg-secondary text-secondary-foreground hover:bg-secondary/80',
    outline:
      'border-border bg-background text-foreground hover:bg-accent hover:text-accent-foreground',
    ghost:
      'border-transparent bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground',
    destructive:
      'border-destructive bg-destructive text-destructive-foreground hover:bg-destructive/90',
    link: 'border-0 bg-transparent text-primary underline-offset-4 hover:underline',
  },
  MH = {
    default: 'border-transparent bg-primary text-primary-foreground',
    secondary: 'border-transparent bg-secondary text-secondary-foreground',
    destructive: 'border-transparent bg-destructive/10 text-destructive',
    outline: 'border-border text-foreground',
    active: 'border-transparent bg-admin-status-active/10 text-admin-status-active',
    paused: 'border-transparent bg-admin-status-paused/10 text-admin-status-paused',
    archived: 'border-border bg-muted text-muted-foreground',
    error: 'border-transparent bg-destructive/10 text-destructive',
    draft: 'border-transparent bg-admin-status-draft/10 text-admin-status-draft',
    scheduled: 'border-transparent bg-admin-status-scheduled/10 text-admin-status-scheduled',
  };
var Zn = E(te());
function RE(...e) {
  return (t) => {
    for (let a of e) typeof a == 'function' ? a(t) : a != null && (a.current = t);
  };
}
function VL(...e) {
  return (t) => {
    for (let a of e) a?.(t);
  };
}
var jL = Zn.forwardRef(function ({ children: t, className: a, ...r }, o) {
  if (!Zn.isValidElement(t)) return t ?? null;
  let n = t,
    l = n.ref;
  return Zn.cloneElement(n, {
    ...r,
    ...n.props,
    className: re(a, n.props.className),
    ref: RE(o, l),
    onClick: VL(r.onClick, n.props.onClick),
    onKeyDown: VL(r.onKeyDown, n.props.onKeyDown),
  });
});
var Li = E($()),
  XL = {
    default: 'h-7 leading-none',
    sm: 'h-7 px-2 text-xs leading-none',
    lg: 'h-7 px-5 leading-none',
    icon: 'h-7 w-7 min-h-7 min-w-7 p-0 leading-none',
  },
  WL = { default: '', pill: 'rounded-full', square: 'rounded-sm' },
  Lo = YL.forwardRef(
    (
      {
        className: e,
        variant: t = 'default',
        size: a = 'default',
        shape: r = 'default',
        asChild: o = !1,
        loading: n = !1,
        disabled: l,
        children: i,
        type: s = 'button',
        ...u
      },
      d
    ) => {
      let c = re(
        Qe.buttonShell,
        Qe.controlRadius,
        'border px-2.5 font-normal transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50',
        Kp[t],
        XL[a],
        WL[r],
        e
      );
      return o
        ? (0, Li.jsx)(jL, {
            ref: d,
            'aria-busy': n || void 0,
            'aria-disabled': l || n || void 0,
            className: c,
            ...u,
            children: i,
          })
        : (0, Li.jsxs)('button', {
            className: c,
            ref: d,
            disabled: l || n,
            'aria-busy': n || void 0,
            type: s,
            ...u,
            children: [
              n
                ? (0, Li.jsx)(zr, { className: 'h-4 w-4 animate-spin', 'aria-hidden': 'true' })
                : null,
              i,
            ],
          });
    }
  );
Lo.displayName = 'Button';
function NH({ variant: e = 'default', size: t = 'default', shape: a = 'default' } = {}) {
  return re(
    Qe.buttonShell,
    Qe.controlRadius,
    'border px-2.5 font-normal transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50',
    Kp[e],
    XL[t],
    WL[a]
  );
}
var Qu = E($());
function Jn({ className: e, shape: t = 'default', variant: a = 'brand', ...r }) {
  return (0, Qu.jsx)(Lo, { shape: t, variant: a, className: re(e), ...r });
}
function QL({ className: e, shape: t = 'pill', variant: a = 'outline', ...r }) {
  return (0, Qu.jsx)(Lo, {
    shape: t,
    variant: a,
    className: re('px-4 text-sm leading-none', e),
    ...r,
  });
}
function qH({ className: e, shape: t = 'pill', type: a = 'submit', variant: r = 'brand', ...o }) {
  return (0, Qu.jsx)(Lo, {
    shape: t,
    type: a,
    variant: r,
    className: re('text-sm leading-none', e),
    ...o,
  });
}
var $L = E(te());
var Zp = {
    load: 'This page could not be loaded. Try again or return to the home page.',
    render: 'This page failed to render. Try reloading or go back.',
    route: 'Navigation failed. The link may be invalid or the server returned an error.',
    'not-found': 'The page you requested does not exist in this console.',
    forbidden: 'You do not have permission to view this page.',
  },
  _E = {
    load: 'Could not load page',
    render: 'Page error',
    route: 'Navigation error',
    'not-found': 'Page not found',
    forbidden: 'Access denied',
  };
function YH(e) {
  return _E[e];
}
function XH(e) {
  return Zp[e];
}
function Ku() {
  return En();
}
function Jp(e) {
  return e !== null && typeof e == 'object';
}
function Zu(e) {
  if (
    Jp(e) &&
    (typeof e.status == 'number' ||
      (typeof e.statusText == 'string' && typeof e.status == 'number'))
  )
    return e.status;
}
function KL(e) {
  if (Jp(e)) {
    if (typeof e.statusText == 'string' && e.statusText.trim() !== '') return e.statusText;
    if (typeof e.data == 'string' && e.data.trim() !== '') return e.data;
  }
}
function ZL(e, t = 'Something went wrong. Try again or return to the home page.') {
  if (e instanceof Ue)
    return e.code === 'PAYMENT_UNAVAILABLE'
      ? 'Payment history is not available in this deployment. Enable the payment module or use the ledger tab for balance activity.'
      : e.status === 404
        ? 'The requested resource was not found.'
        : e.status === 403
          ? 'You do not have permission to view this resource.'
          : e.status === 401
            ? 'Your session expired. Sign in again.'
            : e.status === 0 && e.code === 'TIMEOUT'
              ? 'The request timed out. Check your connection and try again.'
              : e.status >= 500
                ? 'The server encountered an error. Try again later.'
                : e.message.trim() !== ''
                  ? e.message
                  : t;
  let a = Zu(e);
  if (a === 404) return Zp['not-found'];
  if (a === 403) return Zp.forbidden;
  if (a != null && a >= 500) return 'The server encountered an error. Try again later.';
  let r = KL(e);
  return (
    r ||
    (e instanceof Error && e.message.trim() !== ''
      ? Ku()
        ? e.message
        : t
      : typeof e == 'string' && e.trim() !== '' && Ku()
        ? e
        : t)
  );
}
function WH(e) {
  return e instanceof Ue && e.code === 'PAYMENT_UNAVAILABLE';
}
function JL(e, t) {
  let a = [];
  if (e instanceof Ue)
    a.push(`name: ${e.name}`),
      a.push(`status: ${e.status}`),
      a.push(`code: ${e.code}`),
      a.push(`message: ${e.message}`),
      e.stack && a.push('', 'stack:', e.stack);
  else if (e instanceof Error)
    a.push(`name: ${e.name}`),
      a.push(`message: ${e.message}`),
      e.stack && a.push('', 'stack:', e.stack);
  else if (Jp(e)) {
    let r = Zu(e);
    r != null && a.push(`status: ${r}`);
    let o = KL(e);
    o && a.push(`statusText: ${o}`),
      typeof e.message == 'string' && a.push(`message: ${e.message}`);
    try {
      a.push('', 'payload:', JSON.stringify(e, null, 2));
    } catch {
      a.push('', 'payload: [unserializable]');
    }
  } else e != null && a.push(String(e));
  return (
    t?.trim() && a.push('', 'component stack:', t.trim()),
    typeof window < 'u' && a.push('', `location: ${window.location.href}`),
    a.join(`
`)
  );
}
function QH(e) {
  return Zu(e) === 404
    ? 'not-found'
    : Zu(e) === 403
      ? 'forbidden'
      : e instanceof Ue
        ? e.status === 404
          ? 'not-found'
          : e.status === 403
            ? 'forbidden'
            : 'load'
        : 'route';
}
var Co = E($());
function e1({ details: e }) {
  let [t, a] = (0, $L.useState)(!1);
  if (!Ku() || e.trim() === '') return null;
  async function r() {
    try {
      await navigator.clipboard.writeText(e), a(!0), window.setTimeout(() => a(!1), 2e3);
    } catch {
      a(!1);
    }
  }
  return (0, Co.jsxs)('div', {
    className: 'rounded-md border border-border bg-card',
    children: [
      (0, Co.jsxs)('div', {
        className: 'flex items-center justify-between gap-2 border-b border-border px-3 py-2',
        children: [
          (0, Co.jsx)('p', {
            className: 'm-0 text-xs font-semibold text-muted-foreground',
            children: 'Developer details',
          }),
          (0, Co.jsx)(Lo, {
            type: 'button',
            variant: 'outline',
            onClick: () => void r(),
            children: t ? 'Copied' : 'Copy',
          }),
        ],
      }),
      (0, Co.jsx)('pre', {
        className: 'max-h-48 overflow-auto p-3 text-xs font-mono',
        children: e,
      }),
    ],
  });
}
var wo = E(te());
var Io = E($()),
  ar = wo.forwardRef(({ className: e, ...t }, a) =>
    (0, Io.jsx)('div', {
      ref: a,
      'data-ui-card': '',
      className: re(
        re('grid gap-2 border border-border bg-card p-3 text-card-foreground', Qe.panelRadius),
        e
      ),
      ...t,
    })
  );
ar.displayName = 'Card';
var rr = wo.forwardRef(({ className: e, ...t }, a) =>
  (0, Io.jsx)('div', { ref: a, className: re(e), ...t })
);
rr.displayName = 'CardHeader';
var Ro = wo.forwardRef(({ className: e, ...t }, a) =>
  (0, Io.jsx)('div', { ref: a, className: re('font-semibold', e), ...t })
);
Ro.displayName = 'CardTitle';
var _o = wo.forwardRef(({ className: e, ...t }, a) =>
  (0, Io.jsx)('div', { ref: a, className: re('text-muted-foreground text-sm', e), ...t })
);
_o.displayName = 'CardDescription';
var or = wo.forwardRef(({ className: e, ...t }, a) =>
  (0, Io.jsx)('div', { ref: a, className: re('grid gap-3', e), ...t })
);
or.displayName = 'CardContent';
var IE = wo.forwardRef(({ className: e, ...t }, a) =>
  (0, Io.jsx)('div', { ref: a, className: re('grid gap-2', e), ...t })
);
IE.displayName = 'CardFooter';
var Fr = E($());
function qr({ title: e = 'Error', message: t, error: a, componentStack: r }) {
  let o = t ?? ZL(a, 'Request failed.'),
    n = a != null || r ? JL(a, r) : '';
  return (0, Fr.jsxs)(ar, {
    className: 'border-destructive/50 bg-destructive/5',
    children: [
      (0, Fr.jsx)(rr, {
        children: (0, Fr.jsx)(Ro, { className: 'text-base text-destructive', children: e }),
      }),
      (0, Fr.jsxs)(or, {
        children: [
          (0, Fr.jsx)('p', { className: 'text-sm text-muted-foreground', children: o }),
          (0, Fr.jsx)(e1, { details: n }),
        ],
      }),
    ],
  });
}
var t1 = E(te());
var a1 = E($()),
  kE =
    '[appearance:textfield] [-moz-appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none',
  Ra = t1.forwardRef(({ className: e, type: t, ...a }, r) =>
    (0, a1.jsx)('input', {
      type: t,
      className: re(Wu.control, 'w-full', t === 'number' && kE, e),
      ref: r,
      ...a,
    })
  );
Ra.displayName = 'Input';
var r1 = E(te());
var o1 = E($()),
  ta = r1.forwardRef(({ className: e, ...t }, a) =>
    (0, o1.jsx)('label', {
      ref: a,
      className: re('text-sm font-medium leading-none text-foreground', e),
      ...t,
    })
  );
ta.displayName = 'Label';
var n1 = E(te()),
  el = E(te());
var $n = E($());
function ME(...e) {
  return (t) => {
    for (let a of e) a && (typeof a == 'function' ? a(t) : (a.current = t));
  };
}
var Ci = n1.forwardRef(
  ({ className: e, maxLength: t, showCount: a, value: r, onChange: o, rows: n = 3, ...l }, i) => {
    let s = (0, el.useRef)(null),
      u = a ?? t != null,
      d = typeof r == 'string' ? r : Array.isArray(r) ? r.join('') : r != null ? String(r) : '',
      c = d.length,
      f = (0, el.useCallback)(() => {
        let h = s.current;
        h && ((h.style.height = 'auto'), (h.style.height = `${h.scrollHeight}px`));
      }, []);
    return (
      (0, el.useLayoutEffect)(() => {
        f();
      }, [f, d]),
      (0, $n.jsxs)('div', {
        className: 'relative',
        children: [
          (0, $n.jsx)('div', {
            className: re(Wu.panel, 'overflow-hidden'),
            children: (0, $n.jsx)('textarea', {
              ref: ME(i, s),
              rows: n,
              className: re(
                'flex min-h-[5rem] w-full resize-none overflow-hidden border-0 bg-transparent px-3 py-2 text-sm text-foreground transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
                !e?.includes('ui-editor-mono-extralight') && 'font-mono',
                u && t != null && 'pb-7',
                e
              ),
              ...l,
              maxLength: t,
              value: d,
              onChange: (h) => {
                o?.(h), f();
              },
            }),
          }),
          u && t != null
            ? (0, $n.jsxs)('span', {
                className:
                  'pointer-events-none absolute bottom-2 right-3 text-xs tabular-nums text-muted-foreground',
                'aria-hidden': 'true',
                children: [c, '/', t.toLocaleString()],
              })
            : null,
        ],
      })
    );
  }
);
Ci.displayName = 'Textarea';
var be = E($());
function _5() {
  let { bootstrapComplete: e, loading: t } = Gn(),
    [a, r] = (0, ko.useState)(''),
    [o, n] = (0, ko.useState)(''),
    [l, i] = (0, ko.useState)(''),
    [s, u] = (0, ko.useState)(''),
    [d, c] = (0, ko.useState)(),
    [f, h] = (0, ko.useState)(!1);
  async function v(x) {
    x.preventDefault(), c(void 0), h(!0);
    try {
      await hL({ license_token: a.trim(), email: o.trim(), password: l, team_name: s.trim() }),
        window.location.replace('/');
    } catch (y) {
      let m = y instanceof Ue || y instanceof Error ? y.message : 'Activation failed';
      c(m);
    } finally {
      h(!1);
    }
  }
  return t
    ? (0, be.jsx)(qn, {})
    : e
      ? (0, be.jsx)(po, { replace: !0, to: '/login' })
      : (0, be.jsx)('div', {
          className: 'flex min-h-screen items-center justify-center bg-background p-4',
          children: (0, be.jsxs)(ar, {
            className: 'w-full max-w-lg',
            children: [
              (0, be.jsxs)(rr, {
                children: [
                  (0, be.jsx)(Ro, { children: 'Activate deployment' }),
                  (0, be.jsxs)(_o, {
                    children: [
                      'Create the owner account and apply your license in one step. Alternative to JSON setup on',
                      ' ',
                      (0, be.jsx)(Ca, {
                        className: 'text-foreground underline',
                        to: '/setup',
                        children: 'Initial setup',
                      }),
                      '.',
                    ],
                  }),
                ],
              }),
              (0, be.jsxs)(or, {
                className: 'grid gap-4',
                children: [
                  d ? (0, be.jsx)(qr, { title: 'Activation failed', message: d }) : null,
                  (0, be.jsxs)('form', {
                    className: 'grid gap-4',
                    onSubmit: v,
                    children: [
                      (0, be.jsxs)('div', {
                        className: 'grid gap-2',
                        children: [
                          (0, be.jsx)(ta, { htmlFor: 'activate-license', children: 'License JWT' }),
                          (0, be.jsx)(Ci, {
                            id: 'activate-license',
                            className: 'font-mono text-xs',
                            required: !0,
                            value: a,
                            onChange: (x) => r(x.target.value),
                          }),
                        ],
                      }),
                      (0, be.jsxs)('div', {
                        className: 'grid gap-2',
                        children: [
                          (0, be.jsx)(ta, { htmlFor: 'activate-email', children: 'Owner email' }),
                          (0, be.jsx)(Ra, {
                            id: 'activate-email',
                            type: 'email',
                            autoComplete: 'username',
                            required: !0,
                            value: o,
                            onChange: (x) => n(x.target.value),
                          }),
                        ],
                      }),
                      (0, be.jsxs)('div', {
                        className: 'grid gap-2',
                        children: [
                          (0, be.jsx)(ta, { htmlFor: 'activate-password', children: 'Password' }),
                          (0, be.jsx)(Ra, {
                            id: 'activate-password',
                            type: 'password',
                            autoComplete: 'new-password',
                            required: !0,
                            value: l,
                            onChange: (x) => i(x.target.value),
                          }),
                        ],
                      }),
                      (0, be.jsxs)('div', {
                        className: 'grid gap-2',
                        children: [
                          (0, be.jsx)(ta, { htmlFor: 'activate-team', children: 'Team name' }),
                          (0, be.jsx)(Ra, {
                            id: 'activate-team',
                            required: !0,
                            value: s,
                            onChange: (x) => u(x.target.value),
                          }),
                        ],
                      }),
                      (0, be.jsx)(Jn, {
                        className: 'w-full',
                        loading: f,
                        type: 'submit',
                        children: 'Activate and sign in',
                      }),
                    ],
                  }),
                ],
              }),
            ],
          }),
        });
}
var wi = E(te());
var l1 = E($());
function Ju() {
  return En()
    ? null
    : (0, l1.jsx)(QL, {
        className: 'w-full',
        type: 'button',
        onClick: () => {
          Wb(), window.location.replace('/');
        },
        children: 'Enter dev mode (mock API)',
      });
}
var Ie = E($());
function q5() {
  let { bootstrapComplete: e, loading: t } = Gn(),
    [a, r] = (0, wi.useState)(''),
    [o, n] = (0, wi.useState)(''),
    [l, i] = (0, wi.useState)(),
    [s, u] = (0, wi.useState)(!1);
  async function d(c) {
    c.preventDefault(), i(void 0), u(!0);
    try {
      await pL({ email: a, password: o }), window.location.replace('/');
    } catch (f) {
      let h = f instanceof Ue || f instanceof Error ? f.message : 'Sign in failed';
      i(h);
    } finally {
      u(!1);
    }
  }
  return t
    ? (0, Ie.jsx)(qn, {})
    : e
      ? (0, Ie.jsx)('div', {
          className: 'flex min-h-screen items-center justify-center bg-background p-4',
          children: (0, Ie.jsxs)(ar, {
            className: 'w-full max-w-sm',
            children: [
              (0, Ie.jsxs)(rr, {
                children: [
                  (0, Ie.jsx)('h1', { className: 'font-semibold', children: 'Sign in' }),
                  (0, Ie.jsx)(_o, { children: 'ad-event-processor operator console' }),
                ],
              }),
              (0, Ie.jsxs)(or, {
                className: 'grid gap-4',
                children: [
                  l ? (0, Ie.jsx)(qr, { title: 'Sign in failed', message: l }) : null,
                  (0, Ie.jsxs)('form', {
                    className: 'grid gap-4',
                    onSubmit: d,
                    children: [
                      (0, Ie.jsxs)('div', {
                        className: 'grid gap-2',
                        children: [
                          (0, Ie.jsx)(ta, { htmlFor: 'email', children: 'Email' }),
                          (0, Ie.jsx)(Ra, {
                            id: 'email',
                            type: 'email',
                            autoComplete: 'username',
                            required: !0,
                            value: a,
                            onChange: (c) => r(c.target.value),
                          }),
                        ],
                      }),
                      (0, Ie.jsxs)('div', {
                        className: 'grid gap-2',
                        children: [
                          (0, Ie.jsx)(ta, { htmlFor: 'password', children: 'Password' }),
                          (0, Ie.jsx)(Ra, {
                            id: 'password',
                            type: 'password',
                            autoComplete: 'current-password',
                            required: !0,
                            value: o,
                            onChange: (c) => n(c.target.value),
                          }),
                        ],
                      }),
                      (0, Ie.jsx)(Jn, {
                        className: 'w-full',
                        loading: s,
                        type: 'submit',
                        children: 'Sign in',
                      }),
                    ],
                  }),
                  (0, Ie.jsx)(Ju, {}),
                  (0, Ie.jsxs)('p', {
                    className: 'text-center text-sm text-muted-foreground',
                    children: [
                      'First install?',
                      ' ',
                      (0, Ie.jsx)(Ca, {
                        className: 'text-foreground underline',
                        to: '/setup',
                        children: 'Run setup',
                      }),
                      ' ',
                      'or',
                      ' ',
                      (0, Ie.jsx)(Ca, {
                        className: 'text-foreground underline',
                        to: '/activate',
                        children: 'activate with license',
                      }),
                      '.',
                    ],
                  }),
                ],
              }),
            ],
          }),
        })
      : (0, Ie.jsx)(po, { replace: !0, to: '/setup' });
}
var Gr = E(te());
var _ = E(te(), 1),
  c1 = E(xc(), 1);
function EE(e) {
  if (!e || typeof document > 'u') return;
  let t = document.head || document.getElementsByTagName('head')[0],
    a = document.createElement('style');
  (a.type = 'text/css'),
    t.appendChild(a),
    a.styleSheet ? (a.styleSheet.cssText = e) : a.appendChild(document.createTextNode(e));
}
var AE = (e) => {
    switch (e) {
      case 'success':
        return OE;
      case 'info':
        return BE;
      case 'warning':
        return PE;
      case 'error':
        return UE;
      default:
        return null;
    }
  },
  TE = Array(12).fill(0),
  DE = ({ visible: e, className: t }) =>
    _.default.createElement(
      'div',
      { className: ['sonner-loading-wrapper', t].filter(Boolean).join(' '), 'data-visible': e },
      _.default.createElement(
        'div',
        { className: 'sonner-spinner' },
        TE.map((a, r) =>
          _.default.createElement('div', {
            className: 'sonner-loading-bar',
            key: `spinner-bar-${r}`,
          })
        )
      )
    ),
  OE = _.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      viewBox: '0 0 20 20',
      fill: 'currentColor',
      height: '20',
      width: '20',
      'aria-hidden': 'true',
    },
    _.default.createElement('path', {
      fillRule: 'evenodd',
      d: 'M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z',
      clipRule: 'evenodd',
    })
  ),
  PE = _.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      viewBox: '0 0 24 24',
      fill: 'currentColor',
      height: '20',
      width: '20',
      'aria-hidden': 'true',
    },
    _.default.createElement('path', {
      fillRule: 'evenodd',
      d: 'M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003zM12 8.25a.75.75 0 01.75.75v3.75a.75.75 0 01-1.5 0V9a.75.75 0 01.75-.75zm0 8.25a.75.75 0 100-1.5.75.75 0 000 1.5z',
      clipRule: 'evenodd',
    })
  ),
  BE = _.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      viewBox: '0 0 20 20',
      fill: 'currentColor',
      height: '20',
      width: '20',
      'aria-hidden': 'true',
    },
    _.default.createElement('path', {
      fillRule: 'evenodd',
      d: 'M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z',
      clipRule: 'evenodd',
    })
  ),
  UE = _.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      viewBox: '0 0 20 20',
      fill: 'currentColor',
      height: '20',
      width: '20',
      'aria-hidden': 'true',
    },
    _.default.createElement('path', {
      fillRule: 'evenodd',
      d: 'M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-5a.75.75 0 01.75.75v4.5a.75.75 0 01-1.5 0v-4.5A.75.75 0 0110 5zm0 10a1 1 0 100-2 1 1 0 000 2z',
      clipRule: 'evenodd',
    })
  ),
  NE = _.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      width: '12',
      height: '12',
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      strokeWidth: '1.5',
      strokeLinecap: 'round',
      strokeLinejoin: 'round',
      'aria-hidden': 'true',
    },
    _.default.createElement('line', { x1: '18', y1: '6', x2: '6', y2: '18' }),
    _.default.createElement('line', { x1: '6', y1: '6', x2: '18', y2: '18' })
  ),
  HE = () => {
    let [e, t] = _.default.useState(document.hidden);
    return (
      _.default.useEffect(() => {
        let a = () => {
          t(document.hidden);
        };
        return (
          document.addEventListener('visibilitychange', a),
          () => document.removeEventListener('visibilitychange', a)
        );
      }, []),
      e
    );
  },
  zE = 1,
  FE = 100,
  i1 = (e) => {
    var t;
    return typeof e?.id == 'number' || (e == null || (t = e.id) == null ? void 0 : t.length) > 0
      ? e.id
      : zE++;
  },
  $p = class {
    constructor() {
      (this.subscribe = (t) => (
        this.subscribers.push(t),
        this.getActiveToasts().forEach((a) => t(a)),
        () => {
          let a = this.subscribers.indexOf(t);
          this.subscribers.splice(a, 1);
        }
      )),
        (this.publish = (t) => {
          this.subscribers.forEach((a) => a(t));
        }),
        (this.addToast = (t) => {
          this.publish(t), (this.toasts = [...this.toasts, t]), this.trimHistory();
        }),
        (this.trimHistory = () => {
          let t = this.toasts.length - FE;
          t <= 0 ||
            (this.toasts = this.toasts.filter((a) =>
              t > 0 && this.dismissedToasts.has(a.id)
                ? (this.dismissedToasts.delete(a.id), t--, !1)
                : !0
            ));
        }),
        (this.create = (t) => {
          let { message: a, ...r } = t,
            o = i1(t),
            n = this.pendingDismissals.get(o);
          n !== void 0 &&
            (cancelAnimationFrame(n),
            this.pendingDismissals.delete(o),
            this.dismissedToasts.delete(o));
          let l = this.dismissedToasts.has(o),
            i = t.dismissible === void 0 ? !0 : t.dismissible;
          return (
            l &&
              (this.dismissedToasts.delete(o),
              (this.toasts = this.toasts.filter((u) => u.id !== o))),
            (l ? void 0 : this.toasts.find((u) => u.id === o))
              ? (this.toasts = this.toasts.map((u) =>
                  u.id === o
                    ? (this.publish({ ...u, ...t, id: o, title: a }),
                      { ...u, ...t, id: o, dismissible: i, title: a })
                    : u
                ))
              : this.addToast({ title: a, ...r, dismissible: i, id: o }),
            o
          );
        }),
        (this.dismiss = (t) => {
          if (t == null)
            return (
              this.getActiveToasts().forEach((r) => {
                this.dismissedToasts.add(r.id),
                  this.subscribers.forEach((o) => o({ id: r.id, dismiss: !0 }));
              }),
              t
            );
          this.dismissedToasts.add(t);
          let a = this.pendingDismissals.get(t);
          return (
            a !== void 0 && cancelAnimationFrame(a),
            this.pendingDismissals.set(
              t,
              requestAnimationFrame(() => {
                this.pendingDismissals.delete(t),
                  this.subscribers.forEach((r) => r({ id: t, dismiss: !0 }));
              })
            ),
            t
          );
        }),
        (this.message = (t, a) => this.create({ ...a, message: t, type: void 0 })),
        (this.error = (t, a) => this.create({ ...a, message: t, type: 'error' })),
        (this.success = (t, a) => this.create({ ...a, type: 'success', message: t })),
        (this.info = (t, a) => this.create({ ...a, type: 'info', message: t })),
        (this.warning = (t, a) => this.create({ ...a, type: 'warning', message: t })),
        (this.loading = (t, a) => this.create({ ...a, type: 'loading', message: t })),
        (this.promise = (t, a) => {
          if (!a) return;
          let r;
          a.loading !== void 0 &&
            (r = this.create({
              ...a,
              promise: t,
              type: 'loading',
              message: a.loading,
              description: typeof a.description != 'function' ? a.description : void 0,
            }));
          let o = Promise.resolve(t instanceof Function ? t() : t),
            n = r !== void 0,
            l,
            i = o
              .then(async (u) => {
                if (((l = ['resolve', u]), _.default.isValidElement(u)))
                  (n = !1), this.create({ id: r, type: 'default', message: u });
                else if (GE(u) && !u.ok) {
                  n = !1;
                  let c =
                      typeof a.error == 'function'
                        ? await a.error(`HTTP error! status: ${u.status}`)
                        : a.error,
                    f =
                      typeof a.description == 'function'
                        ? await a.description(`HTTP error! status: ${u.status}`)
                        : a.description,
                    v = typeof c == 'object' && !_.default.isValidElement(c) ? c : { message: c };
                  this.create({ id: r, type: 'error', description: f, ...v });
                } else if (u instanceof Error) {
                  n = !1;
                  let c = typeof a.error == 'function' ? await a.error(u) : a.error,
                    f = typeof a.description == 'function' ? await a.description(u) : a.description,
                    v = typeof c == 'object' && !_.default.isValidElement(c) ? c : { message: c };
                  this.create({ id: r, type: 'error', description: f, ...v });
                } else if (a.success !== void 0) {
                  n = !1;
                  let c = typeof a.success == 'function' ? await a.success(u) : a.success,
                    f = typeof a.description == 'function' ? await a.description(u) : a.description,
                    v = typeof c == 'object' && !_.default.isValidElement(c) ? c : { message: c };
                  this.create({ id: r, type: 'success', description: f, ...v });
                }
              })
              .catch(async (u) => {
                if (((l = ['reject', u]), a.error !== void 0)) {
                  n = !1;
                  let d = typeof a.error == 'function' ? await a.error(u) : a.error,
                    c = typeof a.description == 'function' ? await a.description(u) : a.description,
                    h = typeof d == 'object' && !_.default.isValidElement(d) ? d : { message: d };
                  this.create({ id: r, type: 'error', description: c, ...h });
                }
              })
              .finally(() => {
                n && (this.dismiss(r), (r = void 0)), a.finally == null || a.finally.call(a);
              }),
            s = () =>
              new Promise((u, d) => i.then(() => (l[0] === 'reject' ? d(l[1]) : u(l[1]))).catch(d));
          return typeof r != 'string' && typeof r != 'number'
            ? { unwrap: s }
            : Object.assign(r, { unwrap: s });
        }),
        (this.custom = (t, a) => {
          let r = i1(a);
          return this.create({ ...a, jsx: t(r), id: r, type: void 0 }), r;
        }),
        (this.getActiveToasts = () => this.toasts.filter((t) => !this.dismissedToasts.has(t.id))),
        (this.subscribers = []),
        (this.toasts = []),
        (this.dismissedToasts = new Set()),
        (this.pendingDismissals = new Map());
    }
  },
  _t = new $p(),
  qE = (e, t) => _t.message(e, t),
  GE = (e) =>
    e &&
    typeof e == 'object' &&
    'ok' in e &&
    typeof e.ok == 'boolean' &&
    'status' in e &&
    typeof e.status == 'number',
  VE = qE,
  jE = () => _t.toasts,
  YE = () => _t.getActiveToasts(),
  d1 = Object.assign(
    VE,
    {
      success: _t.success,
      info: _t.info,
      warning: _t.warning,
      error: _t.error,
      custom: _t.custom,
      message: _t.message,
      promise: _t.promise,
      dismiss: _t.dismiss,
      loading: _t.loading,
    },
    { getHistory: jE, getToasts: YE }
  );
EE(
  "[data-sonner-toaster][dir=ltr],html[dir=ltr]{--toast-icon-margin-start:-3px;--toast-icon-margin-end:4px;--toast-svg-margin-start:-1px;--toast-svg-margin-end:0px;--toast-button-margin-start:auto;--toast-button-margin-end:0;--toast-close-button-start:0;--toast-close-button-end:unset;--toast-close-button-transform:translate(-35%, -35%)}[data-sonner-toaster][dir=rtl],html[dir=rtl]{--toast-icon-margin-start:4px;--toast-icon-margin-end:-3px;--toast-svg-margin-start:0px;--toast-svg-margin-end:-1px;--toast-button-margin-start:0;--toast-button-margin-end:auto;--toast-close-button-start:unset;--toast-close-button-end:0;--toast-close-button-transform:translate(35%, -35%)}[data-sonner-toaster]{position:fixed;width:var(--width);font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;--gray1:hsl(0, 0%, 99%);--gray2:hsl(0, 0%, 97.3%);--gray3:hsl(0, 0%, 95.1%);--gray4:hsl(0, 0%, 93%);--gray5:hsl(0, 0%, 90.9%);--gray6:hsl(0, 0%, 88.7%);--gray7:hsl(0, 0%, 85.8%);--gray8:hsl(0, 0%, 78%);--gray9:hsl(0, 0%, 56.1%);--gray10:hsl(0, 0%, 52.3%);--gray11:hsl(0, 0%, 43.5%);--gray12:hsl(0, 0%, 9%);--border-radius:8px;box-sizing:border-box;padding:0;margin:0;list-style:none;outline:0;z-index:999999999;transition:transform .4s ease}@media (hover:none) and (pointer:coarse){[data-sonner-toaster][data-lifted=true]{transform:none}}[data-sonner-toaster][data-x-position=right]{right:var(--offset-right)}[data-sonner-toaster][data-x-position=left]{left:var(--offset-left)}[data-sonner-toaster][data-x-position=center]{left:50%;transform:translateX(-50%)}[data-sonner-toaster][data-y-position=top]{top:var(--offset-top)}[data-sonner-toaster][data-y-position=bottom]{bottom:var(--offset-bottom)}[data-sonner-toast]{--y:translateY(100%);--lift-amount:calc(var(--lift) * var(--gap));z-index:var(--z-index);position:absolute;opacity:0;transform:var(--y);touch-action:none;transition:transform .4s,opacity .4s,height .4s,box-shadow .2s;box-sizing:border-box;outline:0;overflow-wrap:anywhere}[data-sonner-toast][data-styled=true]{padding:16px;background:var(--normal-bg);border:1px solid var(--normal-border);color:var(--normal-text);border-radius:var(--border-radius);box-shadow:0 4px 12px rgba(0,0,0,.1);width:var(--width);font-size:13px;display:flex;align-items:center;gap:6px}[data-sonner-toast]:focus-visible{box-shadow:0 4px 12px rgba(0,0,0,.1),0 0 0 2px rgba(0,0,0,.2)}[data-sonner-toast][data-y-position=top]{top:0;--y:translateY(-100%);--lift:1;--lift-amount:calc(1 * var(--gap))}[data-sonner-toast][data-y-position=bottom]{bottom:0;--y:translateY(100%);--lift:-1;--lift-amount:calc(var(--lift) * var(--gap))}[data-sonner-toast][data-styled=true] [data-description]{font-weight:400;line-height:1.4;color:#3f3f3f}[data-rich-colors=true][data-sonner-toast][data-styled=true] [data-description]{color:inherit}[data-sonner-toaster][data-sonner-theme=dark] [data-description]{color:#e8e8e8}[data-sonner-toast][data-styled=true] [data-title]{font-weight:500;line-height:1.5;color:inherit}[data-sonner-toast][data-styled=true] [data-icon]{display:flex;height:16px;width:16px;position:relative;justify-content:flex-start;align-items:center;flex-shrink:0;margin-left:var(--toast-icon-margin-start);margin-right:var(--toast-icon-margin-end)}[data-sonner-toast][data-promise=true] [data-icon]>svg{opacity:0;transform:scale(.8);transform-origin:center;animation:sonner-fade-in .3s ease forwards}[data-sonner-toast][data-styled=true] [data-icon]>*{flex-shrink:0}[data-sonner-toast][data-styled=true] [data-icon] svg{margin-left:var(--toast-svg-margin-start);margin-right:var(--toast-svg-margin-end)}[data-sonner-toast][data-styled=true] [data-content]{display:flex;flex-direction:column;gap:2px;flex:1;min-width:0}[data-sonner-toast][data-styled=true] [data-button]{border-radius:4px;padding-left:8px;padding-right:8px;height:24px;font-size:12px;color:var(--normal-bg);background:var(--normal-text);margin-left:var(--toast-button-margin-start);margin-right:var(--toast-button-margin-end);border:none;font-weight:500;cursor:pointer;outline:0;display:flex;align-items:center;flex-shrink:0;transition:opacity .4s,box-shadow .2s}[data-sonner-toast][data-styled=true] [data-button]:focus-visible{box-shadow:0 0 0 2px rgba(0,0,0,.4)}[data-sonner-toast][data-styled=true] [data-button]:first-of-type{margin-left:var(--toast-button-margin-start);margin-right:var(--toast-button-margin-end)}[data-sonner-toast][data-styled=true] [data-cancel]{color:var(--normal-text);background:rgba(0,0,0,.08)}[data-sonner-toaster][data-sonner-theme=dark] [data-sonner-toast][data-styled=true] [data-cancel]{background:rgba(255,255,255,.3)}[data-sonner-toast][data-styled=true] [data-close-button]{position:absolute;left:var(--toast-close-button-start);right:var(--toast-close-button-end);top:0;height:20px;width:20px;display:flex;justify-content:center;align-items:center;padding:0;color:var(--normal-text);background:var(--normal-bg);border:1px solid var(--normal-border);transform:var(--toast-close-button-transform);border-radius:50%;cursor:pointer;z-index:1;transition:opacity .1s,background .2s,border-color .2s}[data-sonner-toast][data-styled=true] [data-close-button]:focus-visible{box-shadow:0 4px 12px rgba(0,0,0,.1),0 0 0 2px rgba(0,0,0,.2)}[data-sonner-toast][data-styled=true] [data-disabled=true]{cursor:not-allowed}[data-sonner-toast][data-styled=true]:hover [data-close-button]:hover{background:var(--gray2);border-color:var(--gray5)}[data-sonner-toast][data-swiping=true]::before{content:'';position:absolute;left:-100%;right:-100%;height:100%;z-index:-1}[data-sonner-toast][data-y-position=top][data-swiping=true]::before{bottom:50%;transform:scaleY(3) translateY(50%)}[data-sonner-toast][data-y-position=bottom][data-swiping=true]::before{top:50%;transform:scaleY(3) translateY(-50%)}[data-sonner-toast][data-swiping=false][data-removed=true]::before{content:'';position:absolute;inset:0;transform:scaleY(2)}[data-sonner-toast][data-expanded=true]::after{content:'';position:absolute;left:0;height:calc(var(--gap) + 1px);bottom:100%;width:100%}[data-sonner-toast][data-mounted=true]{--y:translateY(0);opacity:1}[data-sonner-toast][data-expanded=false][data-front=false]{--scale:var(--toasts-before) * 0.05 + 1;--y:translateY(calc(var(--lift-amount) * var(--toasts-before))) scale(calc(-1 * var(--scale)));height:var(--front-toast-height)}[data-sonner-toast]>*{transition:opacity .4s}[data-sonner-toast][data-x-position=right]{right:0}[data-sonner-toast][data-x-position=left]{left:0}[data-sonner-toast][data-expanded=false][data-front=false][data-styled=true]>*{opacity:0}[data-sonner-toast][data-visible=false]{opacity:0;pointer-events:none}[data-sonner-toast][data-mounted=true][data-expanded=true]{--y:translateY(calc(var(--lift) * var(--offset)));height:var(--initial-height)}[data-sonner-toast][data-removed=true][data-front=true][data-swipe-out=false]{--y:translateY(calc(var(--lift) * -100%));opacity:0}[data-sonner-toast][data-removed=true][data-front=false][data-swipe-out=false][data-expanded=true]{--y:translateY(calc(var(--lift) * var(--offset) + var(--lift) * -100%));opacity:0}[data-sonner-toast][data-removed=true][data-front=false][data-swipe-out=false][data-expanded=false]{--y:translateY(40%);opacity:0;transition:transform .5s,opacity .2s}[data-sonner-toast][data-removed=true][data-front=false]::before{height:calc(var(--initial-height) + 20%)}[data-sonner-toast][data-swiping=true]{transform:var(--y) translateY(var(--swipe-amount-y,0)) translateX(var(--swipe-amount-x,0));transition:none}[data-sonner-toast][data-swiped=true]{-webkit-user-select:none;user-select:none}[data-sonner-toast][data-swipe-out=true][data-y-position=bottom],[data-sonner-toast][data-swipe-out=true][data-y-position=top]{animation-duration:.2s;animation-timing-function:ease-out;animation-fill-mode:forwards}[data-sonner-toast][data-swipe-out=true][data-swipe-direction=left]{animation-name:swipe-out-left}[data-sonner-toast][data-swipe-out=true][data-swipe-direction=right]{animation-name:swipe-out-right}[data-sonner-toast][data-swipe-out=true][data-swipe-direction=up]{animation-name:swipe-out-up}[data-sonner-toast][data-swipe-out=true][data-swipe-direction=down]{animation-name:swipe-out-down}@keyframes swipe-out-left{from{transform:var(--y) translateX(var(--swipe-amount-x));opacity:1}to{transform:var(--y) translateX(calc(var(--swipe-amount-x) - 100%));opacity:0}}@keyframes swipe-out-right{from{transform:var(--y) translateX(var(--swipe-amount-x));opacity:1}to{transform:var(--y) translateX(calc(var(--swipe-amount-x) + 100%));opacity:0}}@keyframes swipe-out-up{from{transform:var(--y) translateY(var(--swipe-amount-y));opacity:1}to{transform:var(--y) translateY(calc(var(--swipe-amount-y) - 100%));opacity:0}}@keyframes swipe-out-down{from{transform:var(--y) translateY(var(--swipe-amount-y));opacity:1}to{transform:var(--y) translateY(calc(var(--swipe-amount-y) + 100%));opacity:0}}@media (max-width:600px){[data-sonner-toaster]{position:fixed;right:var(--mobile-offset-right);left:var(--mobile-offset-left);width:100%}[data-sonner-toaster][dir=rtl]{left:calc(var(--mobile-offset-left) * -1)}[data-sonner-toaster] [data-sonner-toast]{left:0;right:0;width:calc(100% - var(--mobile-offset-left) * 2)}[data-sonner-toaster][data-x-position=left]{left:var(--mobile-offset-left)}[data-sonner-toaster][data-y-position=bottom]{bottom:var(--mobile-offset-bottom)}[data-sonner-toaster][data-y-position=top]{top:var(--mobile-offset-top)}[data-sonner-toaster][data-x-position=center]{left:var(--mobile-offset-left);right:var(--mobile-offset-right);transform:none}}[data-sonner-toaster][data-sonner-theme=light]{--normal-bg:#fff;--normal-border:var(--gray4);--normal-text:var(--gray12);--success-bg:hsl(143, 85%, 96%);--success-border:hsl(145, 92%, 87%);--success-text:hsl(140, 100%, 27%);--info-bg:hsl(208, 100%, 97%);--info-border:hsl(221, 91%, 93%);--info-text:hsl(210, 92%, 45%);--warning-bg:hsl(49, 100%, 97%);--warning-border:hsl(49, 91%, 84%);--warning-text:hsl(31, 92%, 45%);--error-bg:hsl(359, 100%, 97%);--error-border:hsl(359, 100%, 94%);--error-text:hsl(360, 100%, 45%)}[data-sonner-toaster][data-sonner-theme=light] [data-sonner-toast][data-invert=true]{--normal-bg:#000;--normal-border:hsl(0, 0%, 20%);--normal-text:var(--gray1)}[data-sonner-toaster][data-sonner-theme=dark] [data-sonner-toast][data-invert=true]{--normal-bg:#fff;--normal-border:var(--gray3);--normal-text:var(--gray12)}[data-sonner-toaster][data-sonner-theme=dark]{--normal-bg:#000;--normal-bg-hover:hsl(0, 0%, 12%);--normal-border:hsl(0, 0%, 20%);--normal-border-hover:hsl(0, 0%, 25%);--normal-text:var(--gray1);--success-bg:hsl(150, 100%, 6%);--success-border:hsl(147, 100%, 12%);--success-text:hsl(150, 86%, 65%);--info-bg:hsl(215, 100%, 6%);--info-border:hsl(223, 43%, 17%);--info-text:hsl(216, 87%, 65%);--warning-bg:hsl(64, 100%, 6%);--warning-border:hsl(60, 100%, 9%);--warning-text:hsl(46, 87%, 65%);--error-bg:hsl(358, 76%, 10%);--error-border:hsl(357, 89%, 16%);--error-text:hsl(358, 100%, 81%)}[data-sonner-toaster][data-sonner-theme=dark] [data-sonner-toast] [data-close-button]{background:var(--normal-bg);border-color:var(--normal-border);color:var(--normal-text)}[data-sonner-toaster][data-sonner-theme=dark] [data-sonner-toast] [data-close-button]:hover{background:var(--normal-bg-hover);border-color:var(--normal-border-hover)}[data-rich-colors=true][data-sonner-toast][data-type=success]{background:var(--success-bg);border-color:var(--success-border);color:var(--success-text)}[data-rich-colors=true][data-sonner-toast][data-type=success] [data-close-button]{background:var(--success-bg);border-color:var(--success-border);color:var(--success-text)}[data-rich-colors=true][data-sonner-toast][data-type=info]{background:var(--info-bg);border-color:var(--info-border);color:var(--info-text)}[data-rich-colors=true][data-sonner-toast][data-type=info] [data-close-button]{background:var(--info-bg);border-color:var(--info-border);color:var(--info-text)}[data-rich-colors=true][data-sonner-toast][data-type=warning]{background:var(--warning-bg);border-color:var(--warning-border);color:var(--warning-text)}[data-rich-colors=true][data-sonner-toast][data-type=warning] [data-close-button]{background:var(--warning-bg);border-color:var(--warning-border);color:var(--warning-text)}[data-rich-colors=true][data-sonner-toast][data-type=error]{background:var(--error-bg);border-color:var(--error-border);color:var(--error-text)}[data-rich-colors=true][data-sonner-toast][data-type=error] [data-close-button]{background:var(--error-bg);border-color:var(--error-border);color:var(--error-text)}.sonner-loading-wrapper{--size:16px;height:var(--size);width:var(--size);position:absolute;inset:0;z-index:10}.sonner-loading-wrapper[data-visible=false]{transform-origin:center;animation:sonner-fade-out .2s ease forwards}.sonner-spinner{position:relative;top:50%;left:50%;height:var(--size);width:var(--size)}.sonner-loading-bar{animation:sonner-spin 1.2s linear infinite;background:var(--gray11);border-radius:6px;height:8%;left:-10%;position:absolute;top:-3.9%;width:24%}.sonner-loading-bar:first-child{animation-delay:-1.2s;transform:rotate(.0001deg) translate(146%)}.sonner-loading-bar:nth-child(2){animation-delay:-1.1s;transform:rotate(30deg) translate(146%)}.sonner-loading-bar:nth-child(3){animation-delay:-1s;transform:rotate(60deg) translate(146%)}.sonner-loading-bar:nth-child(4){animation-delay:-.9s;transform:rotate(90deg) translate(146%)}.sonner-loading-bar:nth-child(5){animation-delay:-.8s;transform:rotate(120deg) translate(146%)}.sonner-loading-bar:nth-child(6){animation-delay:-.7s;transform:rotate(150deg) translate(146%)}.sonner-loading-bar:nth-child(7){animation-delay:-.6s;transform:rotate(180deg) translate(146%)}.sonner-loading-bar:nth-child(8){animation-delay:-.5s;transform:rotate(210deg) translate(146%)}.sonner-loading-bar:nth-child(9){animation-delay:-.4s;transform:rotate(240deg) translate(146%)}.sonner-loading-bar:nth-child(10){animation-delay:-.3s;transform:rotate(270deg) translate(146%)}.sonner-loading-bar:nth-child(11){animation-delay:-.2s;transform:rotate(300deg) translate(146%)}.sonner-loading-bar:nth-child(12){animation-delay:-.1s;transform:rotate(330deg) translate(146%)}@keyframes sonner-fade-in{0%{opacity:0;transform:scale(.8)}100%{opacity:1;transform:scale(1)}}@keyframes sonner-fade-out{0%{opacity:1;transform:scale(1)}100%{opacity:0;transform:scale(.8)}}@keyframes sonner-spin{0%{opacity:1}100%{opacity:.15}}@media (prefers-reduced-motion){.sonner-loading-bar,[data-sonner-toast],[data-sonner-toast]>*{transition:none!important;animation:none!important}}.sonner-loader{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);transform-origin:center;transition:opacity .2s,transform .2s}.sonner-loader[data-visible=false]{opacity:0;transform:scale(.8) translate(-50%,-50%)}"
);
function $u(e) {
  return e.label !== void 0;
}
var XE = 3,
  WE = '24px',
  QE = '16px',
  s1 = 4e3,
  KE = 356,
  ZE = 14,
  JE = 45,
  $E = 200;
function _a(...e) {
  return e.filter(Boolean).join(' ');
}
function eA(e) {
  let [t, a] = e.split('-'),
    r = [];
  return t && r.push(t), a && r.push(a), r;
}
var tA = (e) => {
  var t, a, r, o, n, l, i, s, u;
  let {
      invert: d,
      toast: c,
      unstyled: f,
      interacting: h,
      setHeights: v,
      visibleToasts: x,
      heights: y,
      index: m,
      toasts: p,
      expanded: g,
      removeToast: b,
      defaultRichColors: R,
      closeButton: I,
      style: w,
      cancelButtonStyle: L,
      actionButtonStyle: M,
      className: z = '',
      descriptionClassName: qe = '',
      duration: pt,
      position: aa,
      gap: nt,
      expandByDefault: Ee,
      classNames: Y,
      icons: Ne,
      closeButtonAriaLabel: It = 'Close toast',
    } = e,
    [Ke, B] = _.default.useState(null),
    [qt, ra] = _.default.useState(null),
    [Ia, ee] = _.default.useState(!1),
    [j, X] = _.default.useState(!1),
    [Ze, oa] = _.default.useState(!1),
    [q, nr] = _.default.useState(!1),
    [ka, ca] = _.default.useState(!1),
    [Ma, lr] = _.default.useState(0),
    [p1, eh] = _.default.useState(0),
    tl = _.default.useRef(c.duration || pt || s1),
    th = _.default.useRef(null),
    Ea = _.default.useRef(null),
    h1 = m === 0,
    g1 = m + 1 <= x,
    ct = c.type,
    ah = ct ?? 'default',
    Mo = c.dismissible !== !1,
    y1 = c.className || '',
    v1 = c.descriptionClassName || '',
    Ri = _.default.useMemo(() => y.findIndex((H) => H.toastId === c.id) || 0, [y, c.id]),
    b1 = _.default.useMemo(() => {
      var H;
      return (H = c.closeButton) != null ? H : I;
    }, [c.closeButton, I]),
    rh = _.default.useMemo(() => c.duration || pt || s1, [c.duration, pt]),
    ec = _.default.useRef(0),
    Eo = _.default.useRef(0),
    oh = _.default.useRef(0),
    Ao = _.default.useRef(null),
    [x1, S1] = aa.split('-'),
    nh = _.default.useMemo(
      () => y.reduce((H, Ge, lt) => (lt >= Ri ? H : H + Ge.height), 0),
      [y, Ri]
    ),
    lh = HE(),
    da = _.default.useMemo(() => {
      var H;
      return (H = e.swipeDirections) != null ? H : eA(aa);
    }, [e.swipeDirections, aa]),
    L1 = c.invert || d,
    tc = ct === 'loading';
  (Eo.current = _.default.useMemo(() => Ri * nt + nh, [Ri, nh])),
    _.default.useEffect(() => {
      tl.current = rh;
    }, [rh]),
    _.default.useEffect(() => {
      ee(!0);
    }, []),
    _.default.useEffect(() => {
      let H = Ea.current;
      if (H) {
        let Ge = H.getBoundingClientRect().height;
        return (
          eh(Ge),
          v((lt) => [{ toastId: c.id, height: Ge, position: c.position }, ...lt]),
          () => v((lt) => lt.filter((ht) => ht.toastId !== c.id))
        );
      }
    }, [v, c.id]),
    _.default.useLayoutEffect(() => {
      if (!Ia) return;
      let H = Ea.current,
        Ge = H.style.height;
      H.style.height = 'auto';
      let lt = H.getBoundingClientRect().height;
      (H.style.height = Ge),
        eh(lt),
        v((ht) =>
          ht.find((Je) => Je.toastId === c.id)
            ? ht.map((Je) => (Je.toastId === c.id ? { ...Je, height: lt } : Je))
            : [{ toastId: c.id, height: lt, position: c.position }, ...ht]
        );
    }, [Ia, c.title, c.description, v, c.id, c.jsx, c.action, c.cancel]);
  let ir = _.default.useCallback(() => {
    X(!0),
      lr(Eo.current),
      v((H) => H.filter((Ge) => Ge.toastId !== c.id)),
      setTimeout(() => {
        b(c);
      }, $E);
  }, [c, b, v, Eo]);
  _.default.useEffect(() => {
    if ((c.promise && ct === 'loading') || c.duration === 1 / 0 || c.type === 'loading') return;
    let H;
    return (
      g || h || lh
        ? (() => {
            if (oh.current < ec.current) {
              let ht = new Date().getTime() - ec.current;
              tl.current = tl.current - ht;
            }
            oh.current = new Date().getTime();
          })()
        : (() => {
            tl.current !== 1 / 0 &&
              ((ec.current = new Date().getTime()),
              (H = setTimeout(() => {
                c.onAutoClose == null || c.onAutoClose.call(c, c), ir();
              }, tl.current)));
          })(),
      () => clearTimeout(H)
    );
  }, [g, h, c, ct, lh, ir]),
    _.default.useEffect(() => {
      c.delete && (ir(), c.onDismiss == null || c.onDismiss.call(c, c));
    }, [ir, c.delete]);
  function ih() {
    var H;
    if (Ne?.loading) {
      var Ge;
      return _.default.createElement(
        'div',
        {
          className: _a(
            Y?.loader,
            c == null || (Ge = c.classNames) == null ? void 0 : Ge.loader,
            'sonner-loader'
          ),
          'data-visible': ct === 'loading',
        },
        Ne.loading
      );
    }
    return _.default.createElement(DE, {
      className: _a(Y?.loader, c == null || (H = c.classNames) == null ? void 0 : H.loader),
      visible: ct === 'loading',
    });
  }
  let C1 = c.icon || Ne?.[ct] || AE(ct);
  var sh, uh;
  return _.default.createElement(
    'li',
    {
      tabIndex: 0,
      ref: Ea,
      className: _a(
        z,
        y1,
        Y?.toast,
        c == null || (t = c.classNames) == null ? void 0 : t.toast,
        Y?.[ah],
        c == null || (a = c.classNames) == null ? void 0 : a[ah]
      ),
      'data-sonner-toast': '',
      'data-rich-colors': (sh = c.richColors) != null ? sh : R,
      'data-styled': !(c.jsx || c.unstyled || f),
      'data-mounted': Ia,
      'data-promise': !!c.promise,
      'data-swiped': ka,
      'data-removed': j,
      'data-visible': g1,
      'data-y-position': x1,
      'data-x-position': S1,
      'data-index': m,
      'data-front': h1,
      'data-swiping': Ze,
      'data-dismissible': Mo,
      'data-type': ct,
      'data-invert': L1,
      'data-swipe-out': q,
      'data-swipe-direction': qt,
      'data-expanded': !!(g || (Ee && Ia)),
      'data-testid': c.testId,
      style: {
        '--index': m,
        '--toasts-before': m,
        '--z-index': p.length - m,
        '--offset': `${j ? Ma : Eo.current}px`,
        '--initial-height': Ee ? 'auto' : `${p1}px`,
        ...w,
        ...c.style,
      },
      onDragEnd: () => {
        oa(!1), B(null), (Ao.current = null);
      },
      onPointerDown: (H) => {
        H.button !== 2 &&
          (tc ||
            !Mo ||
            ((th.current = new Date()),
            lr(Eo.current),
            H.target.setPointerCapture(H.pointerId),
            H.target.tagName !== 'BUTTON' &&
              (oa(!0), (Ao.current = { x: H.clientX, y: H.clientY }))));
      },
      onPointerUp: () => {
        var H, Ge, lt;
        if (q || !Mo) return;
        Ao.current = null;
        let ht = Number(
            ((H = Ea.current) == null
              ? void 0
              : H.style.getPropertyValue('--swipe-amount-x').replace('px', '')) || 0
          ),
          al = Number(
            ((Ge = Ea.current) == null
              ? void 0
              : Ge.style.getPropertyValue('--swipe-amount-y').replace('px', '')) || 0
          ),
          Je = new Date().getTime() - ((lt = th.current) == null ? void 0 : lt.getTime()),
          Gt = Ke === 'x' ? ht : al,
          fa = Math.abs(Gt) / Je;
        if (
          (Ke === 'x'
            ? da.includes(ht > 0 ? 'right' : 'left')
            : da.includes(al > 0 ? 'bottom' : 'top')) &&
          (Math.abs(Gt) >= JE || fa > 0.11)
        ) {
          lr(Eo.current),
            c.onDismiss == null || c.onDismiss.call(c, c),
            ra(Ke === 'x' ? (ht > 0 ? 'right' : 'left') : al > 0 ? 'down' : 'up'),
            ir(),
            nr(!0);
          return;
        } else {
          var ma, rc;
          (ma = Ea.current) == null || ma.style.setProperty('--swipe-amount-x', '0px'),
            (rc = Ea.current) == null || rc.style.setProperty('--swipe-amount-y', '0px');
        }
        ca(!1), oa(!1), B(null);
      },
      onPointerMove: (H) => {
        var Ge, lt, ht;
        if (
          !Ao.current ||
          !Mo ||
          ((Ge = window.getSelection()) == null ? void 0 : Ge.toString().length) > 0
        )
          return;
        let Je = H.clientY - Ao.current.y,
          Gt = H.clientX - Ao.current.x;
        !Ke && (Math.abs(Gt) > 1 || Math.abs(Je) > 1) && B(Math.abs(Gt) > Math.abs(Je) ? 'x' : 'y');
        let fa = { x: 0, y: 0 },
          ac = (ma) => 1 / (1.5 + Math.abs(ma) / 20);
        if (Ke === 'y') {
          if (da.includes('top') || da.includes('bottom'))
            if ((da.includes('top') && Je < 0) || (da.includes('bottom') && Je > 0)) fa.y = Je;
            else {
              let ma = Je * ac(Je);
              fa.y = Math.abs(ma) < Math.abs(Je) ? ma : Je;
            }
        } else if (Ke === 'x' && (da.includes('left') || da.includes('right')))
          if ((da.includes('left') && Gt < 0) || (da.includes('right') && Gt > 0)) fa.x = Gt;
          else {
            let ma = Gt * ac(Gt);
            fa.x = Math.abs(ma) < Math.abs(Gt) ? ma : Gt;
          }
        (Math.abs(fa.x) > 0 || Math.abs(fa.y) > 0) && ca(!0),
          (lt = Ea.current) == null || lt.style.setProperty('--swipe-amount-x', `${fa.x}px`),
          (ht = Ea.current) == null || ht.style.setProperty('--swipe-amount-y', `${fa.y}px`);
      },
    },
    b1 && !c.jsx && ct !== 'loading'
      ? _.default.createElement(
          'button',
          {
            'aria-label': It,
            'data-disabled': tc,
            'data-close-button': !0,
            onClick:
              tc || !Mo
                ? () => {}
                : () => {
                    ir(), c.onDismiss == null || c.onDismiss.call(c, c);
                  },
            className: _a(
              Y?.closeButton,
              c == null || (r = c.classNames) == null ? void 0 : r.closeButton
            ),
          },
          (uh = Ne?.close) != null ? uh : NE
        )
      : null,
    (ct || c.icon || c.promise) && c.icon !== null && (Ne?.[ct] !== null || c.icon)
      ? _.default.createElement(
          'div',
          {
            'data-icon': '',
            className: _a(Y?.icon, c == null || (o = c.classNames) == null ? void 0 : o.icon),
          },
          ct === 'loading' ? c.icon || ih() : c.promise ? ih() : null,
          ct !== 'loading' ? C1 : null
        )
      : null,
    _.default.createElement(
      'div',
      {
        'data-content': '',
        className: _a(Y?.content, c == null || (n = c.classNames) == null ? void 0 : n.content),
      },
      _.default.createElement(
        'div',
        {
          'data-title': '',
          className: _a(Y?.title, c == null || (l = c.classNames) == null ? void 0 : l.title),
        },
        c.jsx ? c.jsx : typeof c.title == 'function' ? c.title() : c.title
      ),
      c.description
        ? _.default.createElement(
            'div',
            {
              'data-description': '',
              className: _a(
                qe,
                v1,
                Y?.description,
                c == null || (i = c.classNames) == null ? void 0 : i.description
              ),
            },
            typeof c.description == 'function' ? c.description() : c.description
          )
        : null
    ),
    _.default.isValidElement(c.cancel)
      ? c.cancel
      : c.cancel && $u(c.cancel)
        ? _.default.createElement(
            'button',
            {
              'data-button': !0,
              'data-cancel': !0,
              style: c.cancelButtonStyle || L,
              onClick: (H) => {
                $u(c.cancel) &&
                  Mo &&
                  (c.cancel.onClick == null || c.cancel.onClick.call(c.cancel, H), ir());
              },
              className: _a(
                Y?.cancelButton,
                c == null || (s = c.classNames) == null ? void 0 : s.cancelButton
              ),
            },
            c.cancel.label
          )
        : null,
    _.default.isValidElement(c.action)
      ? c.action
      : c.action && $u(c.action)
        ? _.default.createElement(
            'button',
            {
              'data-button': !0,
              'data-action': !0,
              style: c.actionButtonStyle || M,
              onClick: (H) => {
                $u(c.action) &&
                  (c.action.onClick == null || c.action.onClick.call(c.action, H),
                  !H.defaultPrevented && ir());
              },
              className: _a(
                Y?.actionButton,
                c == null || (u = c.classNames) == null ? void 0 : u.actionButton
              ),
            },
            c.action.label
          )
        : null
  );
};
function u1() {
  if (typeof window > 'u' || typeof document > 'u') return 'ltr';
  let e = document.documentElement.getAttribute('dir');
  return e === 'auto' || !e ? window.getComputedStyle(document.documentElement).direction : e;
}
function aA(e, t) {
  let a = {};
  return (
    [e, t].forEach((r, o) => {
      let n = o === 1,
        l = n ? '--mobile-offset' : '--offset',
        i = n ? QE : WE;
      function s(u) {
        ['top', 'right', 'bottom', 'left'].forEach((d) => {
          a[`${l}-${d}`] = typeof u == 'number' ? `${u}px` : u;
        });
      }
      typeof r == 'number' || typeof r == 'string'
        ? s(r)
        : typeof r == 'object'
          ? ['top', 'right', 'bottom', 'left'].forEach((u) => {
              r[u] === void 0
                ? (a[`${l}-${u}`] = i)
                : (a[`${l}-${u}`] = typeof r[u] == 'number' ? `${r[u]}px` : r[u]);
            })
          : s(i);
    }),
    a
  );
}
var V5 = _.default.forwardRef(function (t, a) {
  let {
      id: r,
      invert: o,
      position: n = 'bottom-right',
      hotkey: l = ['altKey', 'KeyT'],
      expand: i,
      closeButton: s,
      className: u,
      offset: d,
      mobileOffset: c,
      theme: f = 'light',
      richColors: h,
      duration: v,
      style: x,
      visibleToasts: y = XE,
      toastOptions: m,
      dir: p = u1(),
      gap: g = ZE,
      icons: b,
      customAriaLabel: R,
      containerAriaLabel: I = 'Notifications',
    } = t,
    [w, L] = _.default.useState([]),
    M = _.default.useMemo(
      () => (r ? w.filter((ee) => ee.toasterId === r) : w.filter((ee) => !ee.toasterId)),
      [w, r]
    ),
    z = _.default.useMemo(
      () => Array.from(new Set([n].concat(M.filter((ee) => ee.position).map((ee) => ee.position)))),
      [M, n]
    ),
    [qe, pt] = _.default.useState([]),
    [aa, nt] = _.default.useState(!1),
    [Ee, Y] = _.default.useState(!1),
    [Ne, It] = _.default.useState(
      f !== 'system'
        ? f
        : typeof window < 'u' &&
            window.matchMedia &&
            window.matchMedia('(prefers-color-scheme: dark)').matches
          ? 'dark'
          : 'light'
    ),
    Ke = _.default.useRef(null),
    B = l.join('+').replace(/Key/g, '').replace(/Digit/g, ''),
    qt = _.default.useRef(null),
    ra = _.default.useRef(!1),
    Ia = _.default.useCallback((ee) => {
      L((j) => {
        var X;
        return (
          ((X = j.find((Ze) => Ze.id === ee.id)) != null && X.delete) || _t.dismiss(ee.id),
          j.filter(({ id: Ze }) => Ze !== ee.id)
        );
      });
    }, []);
  return (
    _.default.useEffect(
      () =>
        _t.subscribe((ee) => {
          if (ee.dismiss) {
            requestAnimationFrame(() => {
              L((j) => j.map((X) => (X.id === ee.id ? { ...X, delete: !0 } : X)));
            });
            return;
          }
          setTimeout(() => {
            c1.default.flushSync(() => {
              L((j) => {
                let X = j.findIndex((Ze) => Ze.id === ee.id);
                return X !== -1
                  ? [...j.slice(0, X), { ...j[X], ...ee }, ...j.slice(X + 1)]
                  : [ee, ...j];
              });
            });
          });
        }),
      []
    ),
    _.default.useEffect(() => {
      if (f !== 'system') {
        It(f);
        return;
      }
      if (
        (f === 'system' &&
          (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
            ? It('dark')
            : It('light')),
        typeof window > 'u')
      )
        return;
      let ee = window.matchMedia('(prefers-color-scheme: dark)');
      try {
        ee.addEventListener('change', ({ matches: j }) => {
          It(j ? 'dark' : 'light');
        });
      } catch {
        ee.addListener(({ matches: X }) => {
          try {
            It(X ? 'dark' : 'light');
          } catch (Ze) {
            console.error(Ze);
          }
        });
      }
    }, [f]),
    _.default.useEffect(() => {
      w.length <= 1 && nt(!1);
    }, [w]),
    _.default.useEffect(() => {
      let ee = (j) => {
        var X;
        if (l.length > 0 && l.every((q) => j[q] || j.code === q)) {
          var oa;
          nt(!0), (oa = Ke.current) == null || oa.focus();
        }
        j.code === 'Escape' &&
          (document.activeElement === Ke.current ||
            ((X = Ke.current) != null && X.contains(document.activeElement))) &&
          nt(!1);
      };
      return (
        document.addEventListener('keydown', ee), () => document.removeEventListener('keydown', ee)
      );
    }, [l]),
    _.default.useEffect(() => {
      if (Ke.current)
        return () => {
          qt.current &&
            (qt.current.focus({ preventScroll: !0 }), (qt.current = null), (ra.current = !1));
        };
    }, [Ke.current]),
    _.default.createElement(
      'section',
      {
        ref: a,
        'aria-label': R ?? `${I} ${B}`,
        tabIndex: -1,
        'aria-live': 'polite',
        'aria-relevant': 'additions text',
        'aria-atomic': 'false',
        suppressHydrationWarning: !0,
        'data-react-aria-top-layer': !0,
      },
      z.map((ee, j) => {
        var X;
        let [Ze, oa] = ee.split('-');
        return M.length
          ? _.default.createElement(
              'ol',
              {
                key: ee,
                dir: p === 'auto' ? u1() : p,
                tabIndex: -1,
                ref: Ke,
                className: u,
                'data-sonner-toaster': !0,
                'data-sonner-theme': Ne,
                'data-y-position': Ze,
                'data-x-position': oa,
                style: {
                  '--front-toast-height': `${((X = qe[0]) == null ? void 0 : X.height) || 0}px`,
                  '--width': `${KE}px`,
                  '--gap': `${g}px`,
                  ...x,
                  ...aA(d, c),
                },
                onBlur: (q) => {
                  ra.current &&
                    !q.currentTarget.contains(q.relatedTarget) &&
                    ((ra.current = !1),
                    qt.current && (qt.current.focus({ preventScroll: !0 }), (qt.current = null)));
                },
                onFocus: (q) => {
                  (q.target instanceof HTMLElement && q.target.dataset.dismissible === 'false') ||
                    ra.current ||
                    ((ra.current = !0), (qt.current = q.relatedTarget));
                },
                onMouseEnter: () => nt(!0),
                onMouseMove: () => nt(!0),
                onMouseLeave: () => {
                  Ee || nt(!1);
                },
                onDragEnd: () => nt(!1),
                onPointerDown: (q) => {
                  (q.target instanceof HTMLElement && q.target.dataset.dismissible === 'false') ||
                    Y(!0);
                },
                onPointerUp: () => Y(!1),
              },
              M.filter((q) => (!q.position && j === 0) || q.position === ee).map((q, nr) => {
                var ka, ca;
                return _.default.createElement(tA, {
                  key: q.id,
                  icons: b,
                  index: nr,
                  toast: q,
                  defaultRichColors: h,
                  duration: (ka = m?.duration) != null ? ka : v,
                  className: m?.className,
                  descriptionClassName: m?.descriptionClassName,
                  invert: o,
                  visibleToasts: y,
                  closeButton: (ca = m?.closeButton) != null ? ca : s,
                  interacting: Ee,
                  position: ee,
                  style: m?.style,
                  unstyled: m?.unstyled,
                  classNames: m?.classNames,
                  cancelButtonStyle: m?.cancelButtonStyle,
                  actionButtonStyle: m?.actionButtonStyle,
                  closeButtonAriaLabel: m?.closeButtonAriaLabel,
                  removeToast: Ia,
                  toasts: M.filter((Ma) => Ma.position == q.position),
                  heights: qe.filter((Ma) => Ma.position == q.position),
                  setHeights: pt,
                  expandByDefault: i,
                  gap: g,
                  expanded: aa,
                  swipeDirections: t.swipeDirections,
                });
              })
            )
          : null;
      })
    )
  );
});
async function X5(e) {
  return _e('/api/v1/settings/platform', { signal: e });
}
async function W5(e, t) {
  return _e('/api/v1/settings/platform', { method: 'PATCH', body: JSON.stringify(e), signal: t });
}
async function f1(e, t, a) {
  return _e('/api/v1/settings/platform/bootstrap', {
    method: 'POST',
    body: JSON.stringify(t),
    headers: { 'X-Install-Token': e },
    signal: a,
  });
}
async function Q5(e, t) {
  let a = e?.install_root?.trim(),
    r = { method: 'POST', signal: t };
  return (
    a && (r.body = JSON.stringify({ install_root: a })), _e('/api/v1/settings/platform/apply', r)
  );
}
var Ft = E($()),
  oA = `{
  "admin_email": "ops@example.com",
  "admin_password": "change-me",
  "config": {
    "tracking_domain": "track.example.com"
  }
}`;
function m1({ onComplete: e }) {
  let [t, a] = (0, Gr.useState)(''),
    [r, o] = (0, Gr.useState)(oA),
    [n, l] = (0, Gr.useState)(!1),
    [i, s] = (0, Gr.useState)(),
    [u, d] = (0, Gr.useState)(!1),
    c = (0, Gr.useCallback)(async () => {
      let f = t.trim(),
        h = r.trim();
      if (!(!f || !h)) {
        l(!0), s(void 0), d(!1);
        try {
          let v = JSON.parse(h);
          if (v == null || typeof v != 'object' || Array.isArray(v))
            throw new Error('Setup configuration must be a JSON object');
          await f1(f, v), d(!0), d1.success('Platform setup complete'), a(''), e?.();
        } catch (v) {
          s(v instanceof Error ? v : new Error(String(v)));
        } finally {
          l(!1);
        }
      }
    }, [r, t, e]);
  return (0, Ft.jsxs)('div', {
    className: 'grid gap-4',
    children: [
      (0, Ft.jsxs)('div', {
        className: 'grid gap-2',
        children: [
          (0, Ft.jsx)(ta, { htmlFor: 'setup-install-token', children: 'Setup token' }),
          (0, Ft.jsx)(Ra, {
            id: 'setup-install-token',
            type: 'password',
            autoComplete: 'off',
            value: t,
            onChange: (f) => a(f.target.value),
          }),
        ],
      }),
      (0, Ft.jsxs)('div', {
        className: 'grid gap-2',
        children: [
          (0, Ft.jsx)(ta, { htmlFor: 'setup-bootstrap-json', children: 'Setup configuration' }),
          (0, Ft.jsx)(Ci, {
            id: 'setup-bootstrap-json',
            value: r,
            maxLength: 65536,
            onChange: (f) => o(f.target.value),
          }),
        ],
      }),
      (0, Ft.jsx)('div', {
        children: (0, Ft.jsx)(Jn, {
          className: 'w-full',
          disabled: !t.trim() || !r.trim(),
          loading: n,
          onClick: () => void c(),
          type: 'button',
          children: 'Complete setup',
        }),
      }),
      u
        ? (0, Ft.jsx)('p', {
            className: 'text-sm text-muted-foreground',
            children: 'Setup complete. Sign in with the admin account you configured.',
          })
        : null,
      i ? (0, Ft.jsx)(qr, { title: 'Setup failed', message: i.message }) : null,
    ],
  });
}
var ot = E($());
function pz() {
  let { bootstrapComplete: e, error: t, loading: a, refreshMeta: r } = Gn();
  return a
    ? (0, ot.jsx)(qn, {})
    : t
      ? (0, ot.jsxs)('div', {
          className:
            'flex min-h-screen flex-col items-center justify-center gap-4 bg-background p-4',
          children: [
            (0, ot.jsx)(qr, { title: 'Could not load install status', message: t.message }),
            (0, ot.jsx)('div', { className: 'w-full max-w-sm', children: (0, ot.jsx)(Ju, {}) }),
          ],
        })
      : e
        ? (0, ot.jsx)(po, { replace: !0, to: '/login' })
        : (0, ot.jsx)('div', {
            className: 'flex min-h-screen items-center justify-center bg-background p-4',
            children: (0, ot.jsxs)(ar, {
              className: 'w-full max-w-lg',
              children: [
                (0, ot.jsxs)(rr, {
                  children: [
                    (0, ot.jsx)(Ro, { children: 'Initial setup' }),
                    (0, ot.jsx)(_o, {
                      children:
                        'First-run platform configuration for ad-event-processor. Requires the setup token from your deployment bundle.',
                    }),
                  ],
                }),
                (0, ot.jsxs)(or, {
                  className: 'grid gap-4',
                  children: [
                    (0, ot.jsx)(m1, {
                      onComplete: () => {
                        r();
                      },
                    }),
                    (0, ot.jsxs)('p', {
                      className: 'm-0 text-center text-sm text-muted-foreground',
                      children: [
                        'Have a license JWT already?',
                        ' ',
                        (0, ot.jsx)(Ca, {
                          className: 'text-foreground underline',
                          to: '/activate',
                          children: 'Activate with license',
                        }),
                        '.',
                      ],
                    }),
                  ],
                }),
              ],
            }),
          });
}
export {
  pa as a,
  nA as b,
  E as c,
  te as d,
  xc as e,
  SR as f,
  gb as g,
  Rt as h,
  uu as i,
  s_ as j,
  Ab as k,
  po as l,
  x_ as m,
  Db as n,
  S_ as o,
  J_ as p,
  Ca as q,
  Nb as r,
  eI as s,
  mI as t,
  Ax as u,
  yD as v,
  _m as w,
  GD as x,
  VD as y,
  ut as z,
  pe as A,
  Ur as B,
  Za as C,
  vo as D,
  ku as E,
  Om as F,
  Jx as G,
  eS as H,
  tS as I,
  aS as J,
  oS as K,
  Pm as L,
  fS as M,
  pS as N,
  CS as O,
  _S as P,
  OS as Q,
  aB as R,
  rB as S,
  Fk as T,
  xB as U,
  Ue as V,
  UT as W,
  En as X,
  NT as Y,
  Uu as Z,
  qm as _,
  Nu as $,
  YB as aa,
  _e as ba,
  XB as ca,
  WB as da,
  n4 as ea,
  l4 as fa,
  i4 as ga,
  s4 as ha,
  u4 as ia,
  c4 as ja,
  d4 as ka,
  qu as la,
  R4 as ma,
  _4 as na,
  I4 as oa,
  Qm as pa,
  Km as qa,
  Zm as ra,
  Jm as sa,
  $m as ta,
  ep as ua,
  tp as va,
  ap as wa,
  rp as xa,
  op as ya,
  Vn as za,
  np as Aa,
  lp as Ba,
  ip as Ca,
  sp as Da,
  up as Ea,
  jn as Fa,
  cp as Ga,
  Yn as Ha,
  dp as Ia,
  fp as Ja,
  mp as Ka,
  pp as La,
  hp as Ma,
  gp as Na,
  yp as Oa,
  vp as Pa,
  Xn as Qa,
  bp as Ra,
  xp as Sa,
  Sp as Ta,
  Lp as Ua,
  Cp as Va,
  wp as Wa,
  Rp as Xa,
  Wn as Ya,
  _p as Za,
  Ip as _a,
  kp as $a,
  Mp as ab,
  Ep as bb,
  Ap as cb,
  Tp as db,
  Dp as eb,
  Op as fb,
  Pp as gb,
  Bp as hb,
  Up as ib,
  Np as jb,
  Hp as kb,
  zp as lb,
  Qn as mb,
  Fp as nb,
  qp as ob,
  Gp as pb,
  Vp as qb,
  Kn as rb,
  jp as sb,
  Yp as tb,
  Xp as ub,
  Wp as vb,
  Qp as wb,
  Qe as xb,
  CH as yb,
  wH as zb,
  RH as Ab,
  M4 as Bb,
  re as Cb,
  Wu as Db,
  Kp as Eb,
  MH as Fb,
  RE as Gb,
  jL as Hb,
  $ as Ib,
  Lo as Jb,
  NH as Kb,
  Ra as Lb,
  qT as Mb,
  sI as Nb,
  jT as Ob,
  YH as Pb,
  XH as Qb,
  Ku as Rb,
  ZL as Sb,
  WH as Tb,
  JL as Ub,
  QH as Vb,
  e1 as Wb,
  ar as Xb,
  rr as Yb,
  Ro as Zb,
  _o as _b,
  or as $b,
  qr as ac,
  oL as bc,
  Jn as cc,
  QL as dc,
  qH as ec,
  d1 as fc,
  V5 as gc,
  qn as hc,
  lL as ic,
  iL as jc,
  sL as kc,
  h4 as lc,
  S4 as mc,
  Gn as nc,
  ta as oc,
  Ci as pc,
  _5 as qc,
  q5 as rc,
  X5 as sc,
  W5 as tc,
  f1 as uc,
  Q5 as vc,
  pz as wc,
};
/*! Bundled license information:

scheduler/cjs/scheduler.production.js:
  (**
   * @license React
   * scheduler.production.js
   *
   * Copyright (c) Meta Platforms, Inc. and affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)

react/cjs/react.production.js:
  (**
   * @license React
   * react.production.js
   *
   * Copyright (c) Meta Platforms, Inc. and affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)

react-dom/cjs/react-dom.production.js:
  (**
   * @license React
   * react-dom.production.js
   *
   * Copyright (c) Meta Platforms, Inc. and affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)

react-dom/cjs/react-dom-client.production.js:
  (**
   * @license React
   * react-dom-client.production.js
   *
   * Copyright (c) Meta Platforms, Inc. and affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)

react/cjs/react-jsx-runtime.production.js:
  (**
   * @license React
   * react-jsx-runtime.production.js
   *
   * Copyright (c) Meta Platforms, Inc. and affiliates.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE file in the root directory of this source tree.
   *)

react-router/dist/development/chunk-BV7QT456.mjs:
react-router/dist/development/index.mjs:
  (**
   * react-router v7.18.3
   *
   * Copyright (c) Remix Software Inc.
   *
   * This source code is licensed under the MIT license found in the
   * LICENSE.md file in the root directory of this source tree.
   *
   * @license MIT
   *)

lucide-react/dist/esm/shared/src/utils.js:
lucide-react/dist/esm/defaultAttributes.js:
lucide-react/dist/esm/Icon.js:
lucide-react/dist/esm/createLucideIcon.js:
lucide-react/dist/esm/icons/activity.js:
lucide-react/dist/esm/icons/app-window.js:
lucide-react/dist/esm/icons/arrow-down.js:
lucide-react/dist/esm/icons/arrow-up-down.js:
lucide-react/dist/esm/icons/arrow-up.js:
lucide-react/dist/esm/icons/bell.js:
lucide-react/dist/esm/icons/book-open.js:
lucide-react/dist/esm/icons/brain.js:
lucide-react/dist/esm/icons/building-2.js:
lucide-react/dist/esm/icons/calendar.js:
lucide-react/dist/esm/icons/chart-column.js:
lucide-react/dist/esm/icons/check.js:
lucide-react/dist/esm/icons/chevron-down.js:
lucide-react/dist/esm/icons/chevron-left.js:
lucide-react/dist/esm/icons/chevron-right.js:
lucide-react/dist/esm/icons/chevrons-up-down.js:
lucide-react/dist/esm/icons/columns-3.js:
lucide-react/dist/esm/icons/copy.js:
lucide-react/dist/esm/icons/ellipsis.js:
lucide-react/dist/esm/icons/file-search.js:
lucide-react/dist/esm/icons/gauge.js:
lucide-react/dist/esm/icons/gavel.js:
lucide-react/dist/esm/icons/git-branch.js:
lucide-react/dist/esm/icons/globe.js:
lucide-react/dist/esm/icons/grip-vertical.js:
lucide-react/dist/esm/icons/hash.js:
lucide-react/dist/esm/icons/inbox.js:
lucide-react/dist/esm/icons/layers.js:
lucide-react/dist/esm/icons/layout-dashboard.js:
lucide-react/dist/esm/icons/layout-template.js:
lucide-react/dist/esm/icons/link-2.js:
lucide-react/dist/esm/icons/list-tree.js:
lucide-react/dist/esm/icons/loader-circle.js:
lucide-react/dist/esm/icons/megaphone.js:
lucide-react/dist/esm/icons/moon.js:
lucide-react/dist/esm/icons/palette.js:
lucide-react/dist/esm/icons/panel-left.js:
lucide-react/dist/esm/icons/plug.js:
lucide-react/dist/esm/icons/plus.js:
lucide-react/dist/esm/icons/receipt.js:
lucide-react/dist/esm/icons/refresh-cw.js:
lucide-react/dist/esm/icons/route.js:
lucide-react/dist/esm/icons/scale.js:
lucide-react/dist/esm/icons/scroll-text.js:
lucide-react/dist/esm/icons/search.js:
lucide-react/dist/esm/icons/settings-2.js:
lucide-react/dist/esm/icons/settings.js:
lucide-react/dist/esm/icons/share-2.js:
lucide-react/dist/esm/icons/shield-alert.js:
lucide-react/dist/esm/icons/shield.js:
lucide-react/dist/esm/icons/sigma.js:
lucide-react/dist/esm/icons/sliders-horizontal.js:
lucide-react/dist/esm/icons/sparkles.js:
lucide-react/dist/esm/icons/sun.js:
lucide-react/dist/esm/icons/tag.js:
lucide-react/dist/esm/icons/tags.js:
lucide-react/dist/esm/icons/toggle-left.js:
lucide-react/dist/esm/icons/triangle-alert.js:
lucide-react/dist/esm/icons/users.js:
lucide-react/dist/esm/icons/workflow.js:
lucide-react/dist/esm/icons/wrench.js:
lucide-react/dist/esm/icons/x.js:
lucide-react/dist/esm/icons/zap.js:
lucide-react/dist/esm/lucide-react.js:
  (**
   * @license lucide-react v0.542.0 - ISC
   *
   * This source code is licensed under the ISC license.
   * See the LICENSE file in the root directory of this source tree.
   *)
*/
//# sourceMappingURL=chunk-XD36ALSM.js.map
