var bL = Object.create;
var $i = Object.defineProperty;
var CL = Object.getOwnPropertyDescriptor;
var wL = Object.getOwnPropertyNames;
var RL = Object.getPrototypeOf,
  IL = Object.prototype.hasOwnProperty;
var aa = (e, t) => () => (t || e((t = { exports: {} }).exports, t), t.exports),
  TR = (e, t) => {
    for (var a in t) $i(e, a, { get: t[a], enumerable: !0 });
  },
  EL = (e, t, a, l) => {
    if ((t && typeof t == 'object') || typeof t == 'function')
      for (let r of wL(t))
        !IL.call(e, r) &&
          r !== a &&
          $i(e, r, { get: () => t[r], enumerable: !(l = CL(t, r)) || l.enumerable });
    return e;
  };
var E = (e, t, a) => (
  (a = e != null ? bL(RL(e)) : {}),
  EL(t || !e || !e.__esModule ? $i(a, 'default', { value: e, enumerable: !0 }) : a, e)
);
var xm = aa((xe) => {
  'use strict';
  function ls(e, t) {
    var a = e.length;
    e.push(t);
    e: for (; 0 < a; ) {
      var l = (a - 1) >>> 1,
        r = e[l];
      if (0 < zn(r, t)) (e[l] = t), (e[a] = r), (a = l);
      else break e;
    }
  }
  function la(e) {
    return e.length === 0 ? null : e[0];
  }
  function qn(e) {
    if (e.length === 0) return null;
    var t = e[0],
      a = e.pop();
    if (a !== t) {
      e[0] = a;
      e: for (var l = 0, r = e.length, o = r >>> 1; l < o; ) {
        var n = 2 * (l + 1) - 1,
          u = e[n],
          i = n + 1,
          s = e[i];
        if (0 > zn(u, a))
          i < r && 0 > zn(s, u)
            ? ((e[l] = s), (e[i] = a), (l = i))
            : ((e[l] = u), (e[n] = a), (l = n));
        else if (i < r && 0 > zn(s, a)) (e[l] = s), (e[i] = a), (l = i);
        else break e;
      }
    }
    return t;
  }
  function zn(e, t) {
    var a = e.sortIndex - t.sortIndex;
    return a !== 0 ? a : e.id - t.id;
  }
  xe.unstable_now = void 0;
  typeof performance == 'object' && typeof performance.now == 'function'
    ? ((sm = performance),
      (xe.unstable_now = function () {
        return sm.now();
      }))
    : ((es = Date),
      (dm = es.now()),
      (xe.unstable_now = function () {
        return es.now() - dm;
      }));
  var sm,
    es,
    dm,
    ya = [],
    Va = [],
    AL = 1,
    kt = null,
    We = 3,
    rs = !1,
    So = !1,
    bo = !1,
    os = !1,
    mm = typeof setTimeout == 'function' ? setTimeout : null,
    pm = typeof clearTimeout == 'function' ? clearTimeout : null,
    fm = typeof setImmediate < 'u' ? setImmediate : null;
  function Fn(e) {
    for (var t = la(Va); t !== null; ) {
      if (t.callback === null) qn(Va);
      else if (t.startTime <= e) qn(Va), (t.sortIndex = t.expirationTime), ls(ya, t);
      else break;
      t = la(Va);
    }
  }
  function ns(e) {
    if (((bo = !1), Fn(e), !So))
      if (la(ya) !== null) (So = !0), nr || ((nr = !0), or());
      else {
        var t = la(Va);
        t !== null && us(ns, t.startTime - e);
      }
  }
  var nr = !1,
    Co = -1,
    hm = 5,
    gm = -1;
  function ym() {
    return os ? !0 : !(xe.unstable_now() - gm < hm);
  }
  function ts() {
    if (((os = !1), nr)) {
      var e = xe.unstable_now();
      gm = e;
      var t = !0;
      try {
        e: {
          (So = !1), bo && ((bo = !1), pm(Co), (Co = -1)), (rs = !0);
          var a = We;
          try {
            t: {
              for (Fn(e), kt = la(ya); kt !== null && !(kt.expirationTime > e && ym()); ) {
                var l = kt.callback;
                if (typeof l == 'function') {
                  (kt.callback = null), (We = kt.priorityLevel);
                  var r = l(kt.expirationTime <= e);
                  if (((e = xe.unstable_now()), typeof r == 'function')) {
                    (kt.callback = r), Fn(e), (t = !0);
                    break t;
                  }
                  kt === la(ya) && qn(ya), Fn(e);
                } else qn(ya);
                kt = la(ya);
              }
              if (kt !== null) t = !0;
              else {
                var o = la(Va);
                o !== null && us(ns, o.startTime - e), (t = !1);
              }
            }
            break e;
          } finally {
            (kt = null), (We = a), (rs = !1);
          }
          t = void 0;
        }
      } finally {
        t ? or() : (nr = !1);
      }
    }
  }
  var or;
  typeof fm == 'function'
    ? (or = function () {
        fm(ts);
      })
    : typeof MessageChannel < 'u'
      ? ((as = new MessageChannel()),
        (cm = as.port2),
        (as.port1.onmessage = ts),
        (or = function () {
          cm.postMessage(null);
        }))
      : (or = function () {
          mm(ts, 0);
        });
  var as, cm;
  function us(e, t) {
    Co = mm(function () {
      e(xe.unstable_now());
    }, t);
  }
  xe.unstable_IdlePriority = 5;
  xe.unstable_ImmediatePriority = 1;
  xe.unstable_LowPriority = 4;
  xe.unstable_NormalPriority = 3;
  xe.unstable_Profiling = null;
  xe.unstable_UserBlockingPriority = 2;
  xe.unstable_cancelCallback = function (e) {
    e.callback = null;
  };
  xe.unstable_forceFrameRate = function (e) {
    0 > e || 125 < e
      ? console.error(
          'forceFrameRate takes a positive int between 0 and 125, forcing frame rates higher than 125 fps is not supported'
        )
      : (hm = 0 < e ? Math.floor(1e3 / e) : 5);
  };
  xe.unstable_getCurrentPriorityLevel = function () {
    return We;
  };
  xe.unstable_next = function (e) {
    switch (We) {
      case 1:
      case 2:
      case 3:
        var t = 3;
        break;
      default:
        t = We;
    }
    var a = We;
    We = t;
    try {
      return e();
    } finally {
      We = a;
    }
  };
  xe.unstable_requestPaint = function () {
    os = !0;
  };
  xe.unstable_runWithPriority = function (e, t) {
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
    var a = We;
    We = e;
    try {
      return t();
    } finally {
      We = a;
    }
  };
  xe.unstable_scheduleCallback = function (e, t, a) {
    var l = xe.unstable_now();
    switch (
      (typeof a == 'object' && a !== null
        ? ((a = a.delay), (a = typeof a == 'number' && 0 < a ? l + a : l))
        : (a = l),
      e)
    ) {
      case 1:
        var r = -1;
        break;
      case 2:
        r = 250;
        break;
      case 5:
        r = 1073741823;
        break;
      case 4:
        r = 1e4;
        break;
      default:
        r = 5e3;
    }
    return (
      (r = a + r),
      (e = {
        id: AL++,
        callback: t,
        priorityLevel: e,
        startTime: a,
        expirationTime: r,
        sortIndex: -1,
      }),
      a > l
        ? ((e.sortIndex = a),
          ls(Va, e),
          la(ya) === null && e === la(Va) && (bo ? (pm(Co), (Co = -1)) : (bo = !0), us(ns, a - l)))
        : ((e.sortIndex = r), ls(ya, e), So || rs || ((So = !0), nr || ((nr = !0), or()))),
      e
    );
  };
  xe.unstable_shouldYield = ym;
  xe.unstable_wrapCallback = function (e) {
    var t = We;
    return function () {
      var a = We;
      We = t;
      try {
        return e.apply(this, arguments);
      } finally {
        We = a;
      }
    };
  };
});
var Lm = aa((kR, vm) => {
  'use strict';
  vm.exports = xm();
});
var Dm = aa((z) => {
  'use strict';
  var ds = Symbol.for('react.transitional.element'),
    TL = Symbol.for('react.portal'),
    ML = Symbol.for('react.fragment'),
    DL = Symbol.for('react.strict_mode'),
    kL = Symbol.for('react.profiler'),
    BL = Symbol.for('react.consumer'),
    OL = Symbol.for('react.context'),
    UL = Symbol.for('react.forward_ref'),
    HL = Symbol.for('react.suspense'),
    NL = Symbol.for('react.memo'),
    Rm = Symbol.for('react.lazy'),
    PL = Symbol.for('react.activity'),
    Sm = Symbol.iterator;
  function _L(e) {
    return e === null || typeof e != 'object'
      ? null
      : ((e = (Sm && e[Sm]) || e['@@iterator']), typeof e == 'function' ? e : null);
  }
  var Im = {
      isMounted: function () {
        return !1;
      },
      enqueueForceUpdate: function () {},
      enqueueReplaceState: function () {},
      enqueueSetState: function () {},
    },
    Em = Object.assign,
    Am = {};
  function ir(e, t, a) {
    (this.props = e), (this.context = t), (this.refs = Am), (this.updater = a || Im);
  }
  ir.prototype.isReactComponent = {};
  ir.prototype.setState = function (e, t) {
    if (typeof e != 'object' && typeof e != 'function' && e != null)
      throw Error(
        'takes an object of state variables to update or a function which returns an object of state variables.'
      );
    this.updater.enqueueSetState(this, e, t, 'setState');
  };
  ir.prototype.forceUpdate = function (e) {
    this.updater.enqueueForceUpdate(this, e, 'forceUpdate');
  };
  function Tm() {}
  Tm.prototype = ir.prototype;
  function fs(e, t, a) {
    (this.props = e), (this.context = t), (this.refs = Am), (this.updater = a || Im);
  }
  var cs = (fs.prototype = new Tm());
  cs.constructor = fs;
  Em(cs, ir.prototype);
  cs.isPureReactComponent = !0;
  var bm = Array.isArray;
  function ss() {}
  var pe = { H: null, A: null, T: null, S: null },
    Mm = Object.prototype.hasOwnProperty;
  function ms(e, t, a) {
    var l = a.ref;
    return { $$typeof: ds, type: e, key: t, ref: l !== void 0 ? l : null, props: a };
  }
  function zL(e, t) {
    return ms(e.type, t, e.props);
  }
  function ps(e) {
    return typeof e == 'object' && e !== null && e.$$typeof === ds;
  }
  function FL(e) {
    var t = { '=': '=0', ':': '=2' };
    return (
      '$' +
      e.replace(/[=:]/g, function (a) {
        return t[a];
      })
    );
  }
  var Cm = /\/+/g;
  function is(e, t) {
    return typeof e == 'object' && e !== null && e.key != null ? FL('' + e.key) : t.toString(36);
  }
  function qL(e) {
    switch (e.status) {
      case 'fulfilled':
        return e.value;
      case 'rejected':
        throw e.reason;
      default:
        switch (
          (typeof e.status == 'string'
            ? e.then(ss, ss)
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
  function ur(e, t, a, l, r) {
    var o = typeof e;
    (o === 'undefined' || o === 'boolean') && (e = null);
    var n = !1;
    if (e === null) n = !0;
    else
      switch (o) {
        case 'bigint':
        case 'string':
        case 'number':
          n = !0;
          break;
        case 'object':
          switch (e.$$typeof) {
            case ds:
            case TL:
              n = !0;
              break;
            case Rm:
              return (n = e._init), ur(n(e._payload), t, a, l, r);
          }
      }
    if (n)
      return (
        (r = r(e)),
        (n = l === '' ? '.' + is(e, 0) : l),
        bm(r)
          ? ((a = ''),
            n != null && (a = n.replace(Cm, '$&/') + '/'),
            ur(r, t, a, '', function (s) {
              return s;
            }))
          : r != null &&
            (ps(r) &&
              (r = zL(
                r,
                a +
                  (r.key == null || (e && e.key === r.key)
                    ? ''
                    : ('' + r.key).replace(Cm, '$&/') + '/') +
                  n
              )),
            t.push(r)),
        1
      );
    n = 0;
    var u = l === '' ? '.' : l + ':';
    if (bm(e))
      for (var i = 0; i < e.length; i++) (l = e[i]), (o = u + is(l, i)), (n += ur(l, t, a, o, r));
    else if (((i = _L(e)), typeof i == 'function'))
      for (e = i.call(e), i = 0; !(l = e.next()).done; )
        (l = l.value), (o = u + is(l, i++)), (n += ur(l, t, a, o, r));
    else if (o === 'object') {
      if (typeof e.then == 'function') return ur(qL(e), t, a, l, r);
      throw (
        ((t = String(e)),
        Error(
          'Objects are not valid as a React child (found: ' +
            (t === '[object Object]' ? 'object with keys {' + Object.keys(e).join(', ') + '}' : t) +
            '). If you meant to render a collection of children, use an array instead.'
        ))
      );
    }
    return n;
  }
  function Gn(e, t, a) {
    if (e == null) return e;
    var l = [],
      r = 0;
    return (
      ur(e, l, '', '', function (o) {
        return t.call(a, o, r++);
      }),
      l
    );
  }
  function GL(e) {
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
  var wm =
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
    VL = {
      map: Gn,
      forEach: function (e, t, a) {
        Gn(
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
          Gn(e, function () {
            t++;
          }),
          t
        );
      },
      toArray: function (e) {
        return (
          Gn(e, function (t) {
            return t;
          }) || []
        );
      },
      only: function (e) {
        if (!ps(e))
          throw Error('React.Children.only expected to receive a single React element child.');
        return e;
      },
    };
  z.Activity = PL;
  z.Children = VL;
  z.Component = ir;
  z.Fragment = ML;
  z.Profiler = kL;
  z.PureComponent = fs;
  z.StrictMode = DL;
  z.Suspense = HL;
  z.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE = pe;
  z.__COMPILER_RUNTIME = {
    __proto__: null,
    c: function (e) {
      return pe.H.useMemoCache(e);
    },
  };
  z.cache = function (e) {
    return function () {
      return e.apply(null, arguments);
    };
  };
  z.cacheSignal = function () {
    return null;
  };
  z.cloneElement = function (e, t, a) {
    if (e == null) throw Error('The argument must be a React element, but you passed ' + e + '.');
    var l = Em({}, e.props),
      r = e.key;
    if (t != null)
      for (o in (t.key !== void 0 && (r = '' + t.key), t))
        !Mm.call(t, o) ||
          o === 'key' ||
          o === '__self' ||
          o === '__source' ||
          (o === 'ref' && t.ref === void 0) ||
          (l[o] = t[o]);
    var o = arguments.length - 2;
    if (o === 1) l.children = a;
    else if (1 < o) {
      for (var n = Array(o), u = 0; u < o; u++) n[u] = arguments[u + 2];
      l.children = n;
    }
    return ms(e.type, r, l);
  };
  z.createContext = function (e) {
    return (
      (e = {
        $$typeof: OL,
        _currentValue: e,
        _currentValue2: e,
        _threadCount: 0,
        Provider: null,
        Consumer: null,
      }),
      (e.Provider = e),
      (e.Consumer = { $$typeof: BL, _context: e }),
      e
    );
  };
  z.createElement = function (e, t, a) {
    var l,
      r = {},
      o = null;
    if (t != null)
      for (l in (t.key !== void 0 && (o = '' + t.key), t))
        Mm.call(t, l) && l !== 'key' && l !== '__self' && l !== '__source' && (r[l] = t[l]);
    var n = arguments.length - 2;
    if (n === 1) r.children = a;
    else if (1 < n) {
      for (var u = Array(n), i = 0; i < n; i++) u[i] = arguments[i + 2];
      r.children = u;
    }
    if (e && e.defaultProps) for (l in ((n = e.defaultProps), n)) r[l] === void 0 && (r[l] = n[l]);
    return ms(e, o, r);
  };
  z.createRef = function () {
    return { current: null };
  };
  z.forwardRef = function (e) {
    return { $$typeof: UL, render: e };
  };
  z.isValidElement = ps;
  z.lazy = function (e) {
    return { $$typeof: Rm, _payload: { _status: -1, _result: e }, _init: GL };
  };
  z.memo = function (e, t) {
    return { $$typeof: NL, type: e, compare: t === void 0 ? null : t };
  };
  z.startTransition = function (e) {
    var t = pe.T,
      a = {};
    pe.T = a;
    try {
      var l = e(),
        r = pe.S;
      r !== null && r(a, l),
        typeof l == 'object' && l !== null && typeof l.then == 'function' && l.then(ss, wm);
    } catch (o) {
      wm(o);
    } finally {
      t !== null && a.types !== null && (t.types = a.types), (pe.T = t);
    }
  };
  z.unstable_useCacheRefresh = function () {
    return pe.H.useCacheRefresh();
  };
  z.use = function (e) {
    return pe.H.use(e);
  };
  z.useActionState = function (e, t, a) {
    return pe.H.useActionState(e, t, a);
  };
  z.useCallback = function (e, t) {
    return pe.H.useCallback(e, t);
  };
  z.useContext = function (e) {
    return pe.H.useContext(e);
  };
  z.useDebugValue = function () {};
  z.useDeferredValue = function (e, t) {
    return pe.H.useDeferredValue(e, t);
  };
  z.useEffect = function (e, t) {
    return pe.H.useEffect(e, t);
  };
  z.useEffectEvent = function (e) {
    return pe.H.useEffectEvent(e);
  };
  z.useId = function () {
    return pe.H.useId();
  };
  z.useImperativeHandle = function (e, t, a) {
    return pe.H.useImperativeHandle(e, t, a);
  };
  z.useInsertionEffect = function (e, t) {
    return pe.H.useInsertionEffect(e, t);
  };
  z.useLayoutEffect = function (e, t) {
    return pe.H.useLayoutEffect(e, t);
  };
  z.useMemo = function (e, t) {
    return pe.H.useMemo(e, t);
  };
  z.useOptimistic = function (e, t) {
    return pe.H.useOptimistic(e, t);
  };
  z.useReducer = function (e, t, a) {
    return pe.H.useReducer(e, t, a);
  };
  z.useRef = function (e) {
    return pe.H.useRef(e);
  };
  z.useState = function (e) {
    return pe.H.useState(e);
  };
  z.useSyncExternalStore = function (e, t, a) {
    return pe.H.useSyncExternalStore(e, t, a);
  };
  z.useTransition = function () {
    return pe.H.useTransition();
  };
  z.version = '19.2.8';
});
var te = aa((OR, km) => {
  'use strict';
  km.exports = Dm();
});
var Om = aa((lt) => {
  'use strict';
  var jL = te();
  function Bm(e) {
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
  function ja() {}
  var at = {
      d: {
        f: ja,
        r: function () {
          throw Error(Bm(522));
        },
        D: ja,
        C: ja,
        L: ja,
        m: ja,
        X: ja,
        S: ja,
        M: ja,
      },
      p: 0,
      findDOMNode: null,
    },
    XL = Symbol.for('react.portal');
  function YL(e, t, a) {
    var l = 3 < arguments.length && arguments[3] !== void 0 ? arguments[3] : null;
    return {
      $$typeof: XL,
      key: l == null ? null : '' + l,
      children: e,
      containerInfo: t,
      implementation: a,
    };
  }
  var wo = jL.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE;
  function Vn(e, t) {
    if (e === 'font') return '';
    if (typeof t == 'string') return t === 'use-credentials' ? t : '';
  }
  lt.__DOM_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE = at;
  lt.createPortal = function (e, t) {
    var a = 2 < arguments.length && arguments[2] !== void 0 ? arguments[2] : null;
    if (!t || (t.nodeType !== 1 && t.nodeType !== 9 && t.nodeType !== 11)) throw Error(Bm(299));
    return YL(e, t, null, a);
  };
  lt.flushSync = function (e) {
    var t = wo.T,
      a = at.p;
    try {
      if (((wo.T = null), (at.p = 2), e)) return e();
    } finally {
      (wo.T = t), (at.p = a), at.d.f();
    }
  };
  lt.preconnect = function (e, t) {
    typeof e == 'string' &&
      (t
        ? ((t = t.crossOrigin),
          (t = typeof t == 'string' ? (t === 'use-credentials' ? t : '') : void 0))
        : (t = null),
      at.d.C(e, t));
  };
  lt.prefetchDNS = function (e) {
    typeof e == 'string' && at.d.D(e);
  };
  lt.preinit = function (e, t) {
    if (typeof e == 'string' && t && typeof t.as == 'string') {
      var a = t.as,
        l = Vn(a, t.crossOrigin),
        r = typeof t.integrity == 'string' ? t.integrity : void 0,
        o = typeof t.fetchPriority == 'string' ? t.fetchPriority : void 0;
      a === 'style'
        ? at.d.S(e, typeof t.precedence == 'string' ? t.precedence : void 0, {
            crossOrigin: l,
            integrity: r,
            fetchPriority: o,
          })
        : a === 'script' &&
          at.d.X(e, {
            crossOrigin: l,
            integrity: r,
            fetchPriority: o,
            nonce: typeof t.nonce == 'string' ? t.nonce : void 0,
          });
    }
  };
  lt.preinitModule = function (e, t) {
    if (typeof e == 'string')
      if (typeof t == 'object' && t !== null) {
        if (t.as == null || t.as === 'script') {
          var a = Vn(t.as, t.crossOrigin);
          at.d.M(e, {
            crossOrigin: a,
            integrity: typeof t.integrity == 'string' ? t.integrity : void 0,
            nonce: typeof t.nonce == 'string' ? t.nonce : void 0,
          });
        }
      } else t == null && at.d.M(e);
  };
  lt.preload = function (e, t) {
    if (typeof e == 'string' && typeof t == 'object' && t !== null && typeof t.as == 'string') {
      var a = t.as,
        l = Vn(a, t.crossOrigin);
      at.d.L(e, a, {
        crossOrigin: l,
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
  lt.preloadModule = function (e, t) {
    if (typeof e == 'string')
      if (t) {
        var a = Vn(t.as, t.crossOrigin);
        at.d.m(e, {
          as: typeof t.as == 'string' && t.as !== 'script' ? t.as : void 0,
          crossOrigin: a,
          integrity: typeof t.integrity == 'string' ? t.integrity : void 0,
        });
      } else at.d.m(e);
  };
  lt.requestFormReset = function (e) {
    at.d.r(e);
  };
  lt.unstable_batchedUpdates = function (e, t) {
    return e(t);
  };
  lt.useFormState = function (e, t, a) {
    return wo.H.useFormState(e, t, a);
  };
  lt.useFormStatus = function () {
    return wo.H.useHostTransitionStatus();
  };
  lt.version = '19.2.8';
});
var hs = aa((HR, Hm) => {
  'use strict';
  function Um() {
    if (
      !(
        typeof __REACT_DEVTOOLS_GLOBAL_HOOK__ > 'u' ||
        typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE != 'function'
      )
    )
      try {
        __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Um);
      } catch (e) {
        console.error(e);
      }
  }
  Um(), (Hm.exports = Om());
});
var Ky = aa((hi) => {
  'use strict';
  var ke = Lm(),
    ih = te(),
    KL = hs();
  function S(e) {
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
  function sh(e) {
    return !(!e || (e.nodeType !== 1 && e.nodeType !== 9 && e.nodeType !== 11));
  }
  function fn(e) {
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
  function dh(e) {
    if (e.tag === 13) {
      var t = e.memoizedState;
      if ((t === null && ((e = e.alternate), e !== null && (t = e.memoizedState)), t !== null))
        return t.dehydrated;
    }
    return null;
  }
  function fh(e) {
    if (e.tag === 31) {
      var t = e.memoizedState;
      if ((t === null && ((e = e.alternate), e !== null && (t = e.memoizedState)), t !== null))
        return t.dehydrated;
    }
    return null;
  }
  function Nm(e) {
    if (fn(e) !== e) throw Error(S(188));
  }
  function QL(e) {
    var t = e.alternate;
    if (!t) {
      if (((t = fn(e)), t === null)) throw Error(S(188));
      return t !== e ? null : e;
    }
    for (var a = e, l = t; ; ) {
      var r = a.return;
      if (r === null) break;
      var o = r.alternate;
      if (o === null) {
        if (((l = r.return), l !== null)) {
          a = l;
          continue;
        }
        break;
      }
      if (r.child === o.child) {
        for (o = r.child; o; ) {
          if (o === a) return Nm(r), e;
          if (o === l) return Nm(r), t;
          o = o.sibling;
        }
        throw Error(S(188));
      }
      if (a.return !== l.return) (a = r), (l = o);
      else {
        for (var n = !1, u = r.child; u; ) {
          if (u === a) {
            (n = !0), (a = r), (l = o);
            break;
          }
          if (u === l) {
            (n = !0), (l = r), (a = o);
            break;
          }
          u = u.sibling;
        }
        if (!n) {
          for (u = o.child; u; ) {
            if (u === a) {
              (n = !0), (a = o), (l = r);
              break;
            }
            if (u === l) {
              (n = !0), (l = o), (a = r);
              break;
            }
            u = u.sibling;
          }
          if (!n) throw Error(S(189));
        }
      }
      if (a.alternate !== l) throw Error(S(190));
    }
    if (a.tag !== 3) throw Error(S(188));
    return a.stateNode.current === a ? e : t;
  }
  function ch(e) {
    var t = e.tag;
    if (t === 5 || t === 26 || t === 27 || t === 6) return e;
    for (e = e.child; e !== null; ) {
      if (((t = ch(e)), t !== null)) return t;
      e = e.sibling;
    }
    return null;
  }
  var ye = Object.assign,
    ZL = Symbol.for('react.element'),
    jn = Symbol.for('react.transitional.element'),
    ko = Symbol.for('react.portal'),
    pr = Symbol.for('react.fragment'),
    mh = Symbol.for('react.strict_mode'),
    Qs = Symbol.for('react.profiler'),
    ph = Symbol.for('react.consumer'),
    Ra = Symbol.for('react.context'),
    Vd = Symbol.for('react.forward_ref'),
    Zs = Symbol.for('react.suspense'),
    Ws = Symbol.for('react.suspense_list'),
    jd = Symbol.for('react.memo'),
    Xa = Symbol.for('react.lazy');
  Symbol.for('react.scope');
  var Js = Symbol.for('react.activity');
  Symbol.for('react.legacy_hidden');
  Symbol.for('react.tracing_marker');
  var WL = Symbol.for('react.memo_cache_sentinel');
  Symbol.for('react.view_transition');
  var Pm = Symbol.iterator;
  function Ro(e) {
    return e === null || typeof e != 'object'
      ? null
      : ((e = (Pm && e[Pm]) || e['@@iterator']), typeof e == 'function' ? e : null);
  }
  var JL = Symbol.for('react.client.reference');
  function $s(e) {
    if (e == null) return null;
    if (typeof e == 'function') return e.$$typeof === JL ? null : e.displayName || e.name || null;
    if (typeof e == 'string') return e;
    switch (e) {
      case pr:
        return 'Fragment';
      case Qs:
        return 'Profiler';
      case mh:
        return 'StrictMode';
      case Zs:
        return 'Suspense';
      case Ws:
        return 'SuspenseList';
      case Js:
        return 'Activity';
    }
    if (typeof e == 'object')
      switch (e.$$typeof) {
        case ko:
          return 'Portal';
        case Ra:
          return e.displayName || 'Context';
        case ph:
          return (e._context.displayName || 'Context') + '.Consumer';
        case Vd:
          var t = e.render;
          return (
            (e = e.displayName),
            e ||
              ((e = t.displayName || t.name || ''),
              (e = e !== '' ? 'ForwardRef(' + e + ')' : 'ForwardRef')),
            e
          );
        case jd:
          return (t = e.displayName || null), t !== null ? t : $s(e.type) || 'Memo';
        case Xa:
          (t = e._payload), (e = e._init);
          try {
            return $s(e(t));
          } catch {}
      }
    return null;
  }
  var Bo = Array.isArray,
    N = ih.__CLIENT_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE,
    le = KL.__DOM_INTERNALS_DO_NOT_USE_OR_WARN_USERS_THEY_CANNOT_UPGRADE,
    Al = { pending: !1, data: null, method: null, action: null },
    ed = [],
    hr = -1;
  function ia(e) {
    return { current: e };
  }
  function _e(e) {
    0 > hr || ((e.current = ed[hr]), (ed[hr] = null), hr--);
  }
  function ce(e, t) {
    hr++, (ed[hr] = e.current), (e.current = t);
  }
  var ua = ia(null),
    Zo = ia(null),
    ll = ia(null),
    wu = ia(null);
  function Ru(e, t) {
    switch ((ce(ll, t), ce(Zo, e), ce(ua, null), t.nodeType)) {
      case 9:
      case 11:
        e = (e = t.documentElement) && (e = e.namespaceURI) ? jp(e) : 0;
        break;
      default:
        if (((e = t.tagName), (t = t.namespaceURI))) (t = jp(t)), (e = Oy(t, e));
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
    _e(ua), ce(ua, e);
  }
  function Br() {
    _e(ua), _e(Zo), _e(ll);
  }
  function td(e) {
    e.memoizedState !== null && ce(wu, e);
    var t = ua.current,
      a = Oy(t, e.type);
    t !== a && (ce(Zo, e), ce(ua, a));
  }
  function Iu(e) {
    Zo.current === e && (_e(ua), _e(Zo)), wu.current === e && (_e(wu), (un._currentValue = Al));
  }
  var gs, _m;
  function wl(e) {
    if (gs === void 0)
      try {
        throw Error();
      } catch (a) {
        var t = a.stack.trim().match(/\n( *(at )?)/);
        (gs = (t && t[1]) || ''),
          (_m =
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
      gs +
      e +
      _m
    );
  }
  var ys = !1;
  function xs(e, t) {
    if (!e || ys) return '';
    ys = !0;
    var a = Error.prepareStackTrace;
    Error.prepareStackTrace = void 0;
    try {
      var l = {
        DetermineComponentFrameRoot: function () {
          try {
            if (t) {
              var d = function () {
                throw Error();
              };
              if (
                (Object.defineProperty(d.prototype, 'props', {
                  set: function () {
                    throw Error();
                  },
                }),
                typeof Reflect == 'object' && Reflect.construct)
              ) {
                try {
                  Reflect.construct(d, []);
                } catch (h) {
                  var p = h;
                }
                Reflect.construct(e, [], d);
              } else {
                try {
                  d.call();
                } catch (h) {
                  p = h;
                }
                e.call(d.prototype);
              }
            } else {
              try {
                throw Error();
              } catch (h) {
                p = h;
              }
              (d = e()) && typeof d.catch == 'function' && d.catch(function () {});
            }
          } catch (h) {
            if (h && p && typeof h.stack == 'string') return [h.stack, p.stack];
          }
          return [null, null];
        },
      };
      l.DetermineComponentFrameRoot.displayName = 'DetermineComponentFrameRoot';
      var r = Object.getOwnPropertyDescriptor(l.DetermineComponentFrameRoot, 'name');
      r &&
        r.configurable &&
        Object.defineProperty(l.DetermineComponentFrameRoot, 'name', {
          value: 'DetermineComponentFrameRoot',
        });
      var o = l.DetermineComponentFrameRoot(),
        n = o[0],
        u = o[1];
      if (n && u) {
        var i = n.split(`
`),
          s = u.split(`
`);
        for (r = l = 0; l < i.length && !i[l].includes('DetermineComponentFrameRoot'); ) l++;
        for (; r < s.length && !s[r].includes('DetermineComponentFrameRoot'); ) r++;
        if (l === i.length || r === s.length)
          for (l = i.length - 1, r = s.length - 1; 1 <= l && 0 <= r && i[l] !== s[r]; ) r--;
        for (; 1 <= l && 0 <= r; l--, r--)
          if (i[l] !== s[r]) {
            if (l !== 1 || r !== 1)
              do
                if ((l--, r--, 0 > r || i[l] !== s[r])) {
                  var f =
                    `
` + i[l].replace(' at new ', ' at ');
                  return (
                    e.displayName &&
                      f.includes('<anonymous>') &&
                      (f = f.replace('<anonymous>', e.displayName)),
                    f
                  );
                }
              while (1 <= l && 0 <= r);
            break;
          }
      }
    } finally {
      (ys = !1), (Error.prepareStackTrace = a);
    }
    return (a = e ? e.displayName || e.name : '') ? wl(a) : '';
  }
  function $L(e, t) {
    switch (e.tag) {
      case 26:
      case 27:
      case 5:
        return wl(e.type);
      case 16:
        return wl('Lazy');
      case 13:
        return e.child !== t && t !== null ? wl('Suspense Fallback') : wl('Suspense');
      case 19:
        return wl('SuspenseList');
      case 0:
      case 15:
        return xs(e.type, !1);
      case 11:
        return xs(e.type.render, !1);
      case 1:
        return xs(e.type, !0);
      case 31:
        return wl('Activity');
      default:
        return '';
    }
  }
  function zm(e) {
    try {
      var t = '',
        a = null;
      do (t += $L(e, a)), (a = e), (e = e.return);
      while (e);
      return t;
    } catch (l) {
      return (
        `
Error generating stack: ` +
        l.message +
        `
` +
        l.stack
      );
    }
  }
  var ad = Object.prototype.hasOwnProperty,
    Xd = ke.unstable_scheduleCallback,
    vs = ke.unstable_cancelCallback,
    eS = ke.unstable_shouldYield,
    tS = ke.unstable_requestPaint,
    Ct = ke.unstable_now,
    aS = ke.unstable_getCurrentPriorityLevel,
    hh = ke.unstable_ImmediatePriority,
    gh = ke.unstable_UserBlockingPriority,
    Eu = ke.unstable_NormalPriority,
    lS = ke.unstable_LowPriority,
    yh = ke.unstable_IdlePriority,
    rS = ke.log,
    oS = ke.unstable_setDisableYieldValue,
    cn = null,
    wt = null;
  function Ja(e) {
    if ((typeof rS == 'function' && oS(e), wt && typeof wt.setStrictMode == 'function'))
      try {
        wt.setStrictMode(cn, e);
      } catch {}
  }
  var Rt = Math.clz32 ? Math.clz32 : iS,
    nS = Math.log,
    uS = Math.LN2;
  function iS(e) {
    return (e >>>= 0), e === 0 ? 32 : (31 - ((nS(e) / uS) | 0)) | 0;
  }
  var Xn = 256,
    Yn = 262144,
    Kn = 4194304;
  function Rl(e) {
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
  function $u(e, t, a) {
    var l = e.pendingLanes;
    if (l === 0) return 0;
    var r = 0,
      o = e.suspendedLanes,
      n = e.pingedLanes;
    e = e.warmLanes;
    var u = l & 134217727;
    return (
      u !== 0
        ? ((l = u & ~o),
          l !== 0
            ? (r = Rl(l))
            : ((n &= u), n !== 0 ? (r = Rl(n)) : a || ((a = u & ~e), a !== 0 && (r = Rl(a)))))
        : ((u = l & ~o),
          u !== 0
            ? (r = Rl(u))
            : n !== 0
              ? (r = Rl(n))
              : a || ((a = l & ~e), a !== 0 && (r = Rl(a)))),
      r === 0
        ? 0
        : t !== 0 &&
            t !== r &&
            (t & o) === 0 &&
            ((o = r & -r), (a = t & -t), o >= a || (o === 32 && (a & 4194048) !== 0))
          ? t
          : r
    );
  }
  function mn(e, t) {
    return (e.pendingLanes & ~(e.suspendedLanes & ~e.pingedLanes) & t) === 0;
  }
  function sS(e, t) {
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
  function xh() {
    var e = Kn;
    return (Kn <<= 1), (Kn & 62914560) === 0 && (Kn = 4194304), e;
  }
  function Ls(e) {
    for (var t = [], a = 0; 31 > a; a++) t.push(e);
    return t;
  }
  function pn(e, t) {
    (e.pendingLanes |= t),
      t !== 268435456 && ((e.suspendedLanes = 0), (e.pingedLanes = 0), (e.warmLanes = 0));
  }
  function dS(e, t, a, l, r, o) {
    var n = e.pendingLanes;
    (e.pendingLanes = a),
      (e.suspendedLanes = 0),
      (e.pingedLanes = 0),
      (e.warmLanes = 0),
      (e.expiredLanes &= a),
      (e.entangledLanes &= a),
      (e.errorRecoveryDisabledLanes &= a),
      (e.shellSuspendCounter = 0);
    var u = e.entanglements,
      i = e.expirationTimes,
      s = e.hiddenUpdates;
    for (a = n & ~a; 0 < a; ) {
      var f = 31 - Rt(a),
        d = 1 << f;
      (u[f] = 0), (i[f] = -1);
      var p = s[f];
      if (p !== null)
        for (s[f] = null, f = 0; f < p.length; f++) {
          var h = p[f];
          h !== null && (h.lane &= -536870913);
        }
      a &= ~d;
    }
    l !== 0 && vh(e, l, 0),
      o !== 0 && r === 0 && e.tag !== 0 && (e.suspendedLanes |= o & ~(n & ~t));
  }
  function vh(e, t, a) {
    (e.pendingLanes |= t), (e.suspendedLanes &= ~t);
    var l = 31 - Rt(t);
    (e.entangledLanes |= t), (e.entanglements[l] = e.entanglements[l] | 1073741824 | (a & 261930));
  }
  function Lh(e, t) {
    var a = (e.entangledLanes |= t);
    for (e = e.entanglements; a; ) {
      var l = 31 - Rt(a),
        r = 1 << l;
      (r & t) | (e[l] & t) && (e[l] |= t), (a &= ~r);
    }
  }
  function Sh(e, t) {
    var a = t & -t;
    return (a = (a & 42) !== 0 ? 1 : Yd(a)), (a & (e.suspendedLanes | t)) !== 0 ? 0 : a;
  }
  function Yd(e) {
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
  function Kd(e) {
    return (e &= -e), 2 < e ? (8 < e ? ((e & 134217727) !== 0 ? 32 : 268435456) : 8) : 2;
  }
  function bh() {
    var e = le.p;
    return e !== 0 ? e : ((e = window.event), e === void 0 ? 32 : jy(e.type));
  }
  function Fm(e, t) {
    var a = le.p;
    try {
      return (le.p = e), t();
    } finally {
      le.p = a;
    }
  }
  var gl = Math.random().toString(36).slice(2),
    Ve = '__reactFiber$' + gl,
    ct = '__reactProps$' + gl,
    Vr = '__reactContainer$' + gl,
    ld = '__reactEvents$' + gl,
    fS = '__reactListeners$' + gl,
    cS = '__reactHandles$' + gl,
    qm = '__reactResources$' + gl,
    hn = '__reactMarker$' + gl;
  function Qd(e) {
    delete e[Ve], delete e[ct], delete e[ld], delete e[fS], delete e[cS];
  }
  function gr(e) {
    var t = e[Ve];
    if (t) return t;
    for (var a = e.parentNode; a; ) {
      if ((t = a[Vr] || a[Ve])) {
        if (((a = t.alternate), t.child !== null || (a !== null && a.child !== null)))
          for (e = Zp(e); e !== null; ) {
            if ((a = e[Ve])) return a;
            e = Zp(e);
          }
        return t;
      }
      (e = a), (a = e.parentNode);
    }
    return null;
  }
  function jr(e) {
    if ((e = e[Ve] || e[Vr])) {
      var t = e.tag;
      if (t === 5 || t === 6 || t === 13 || t === 31 || t === 26 || t === 27 || t === 3) return e;
    }
    return null;
  }
  function Oo(e) {
    var t = e.tag;
    if (t === 5 || t === 26 || t === 27 || t === 6) return e.stateNode;
    throw Error(S(33));
  }
  function Ir(e) {
    var t = e[qm];
    return t || (t = e[qm] = { hoistableStyles: new Map(), hoistableScripts: new Map() }), t;
  }
  function Pe(e) {
    e[hn] = !0;
  }
  var Ch = new Set(),
    wh = {};
  function Pl(e, t) {
    Or(e, t), Or(e + 'Capture', t);
  }
  function Or(e, t) {
    for (wh[e] = t, e = 0; e < t.length; e++) Ch.add(t[e]);
  }
  var mS = RegExp(
      '^[:A-Z_a-z\\u00C0-\\u00D6\\u00D8-\\u00F6\\u00F8-\\u02FF\\u0370-\\u037D\\u037F-\\u1FFF\\u200C-\\u200D\\u2070-\\u218F\\u2C00-\\u2FEF\\u3001-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFFD][:A-Z_a-z\\u00C0-\\u00D6\\u00D8-\\u00F6\\u00F8-\\u02FF\\u0370-\\u037D\\u037F-\\u1FFF\\u200C-\\u200D\\u2070-\\u218F\\u2C00-\\u2FEF\\u3001-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFFD\\-.0-9\\u00B7\\u0300-\\u036F\\u203F-\\u2040]*$'
    ),
    Gm = {},
    Vm = {};
  function pS(e) {
    return ad.call(Vm, e)
      ? !0
      : ad.call(Gm, e)
        ? !1
        : mS.test(e)
          ? (Vm[e] = !0)
          : ((Gm[e] = !0), !1);
  }
  function su(e, t, a) {
    if (pS(t))
      if (a === null) e.removeAttribute(t);
      else {
        switch (typeof a) {
          case 'undefined':
          case 'function':
          case 'symbol':
            e.removeAttribute(t);
            return;
          case 'boolean':
            var l = t.toLowerCase().slice(0, 5);
            if (l !== 'data-' && l !== 'aria-') {
              e.removeAttribute(t);
              return;
            }
        }
        e.setAttribute(t, '' + a);
      }
  }
  function Qn(e, t, a) {
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
  function xa(e, t, a, l) {
    if (l === null) e.removeAttribute(a);
    else {
      switch (typeof l) {
        case 'undefined':
        case 'function':
        case 'symbol':
        case 'boolean':
          e.removeAttribute(a);
          return;
      }
      e.setAttributeNS(t, a, '' + l);
    }
  }
  function Ot(e) {
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
  function Rh(e) {
    var t = e.type;
    return (e = e.nodeName) && e.toLowerCase() === 'input' && (t === 'checkbox' || t === 'radio');
  }
  function hS(e, t, a) {
    var l = Object.getOwnPropertyDescriptor(e.constructor.prototype, t);
    if (
      !e.hasOwnProperty(t) &&
      typeof l < 'u' &&
      typeof l.get == 'function' &&
      typeof l.set == 'function'
    ) {
      var r = l.get,
        o = l.set;
      return (
        Object.defineProperty(e, t, {
          configurable: !0,
          get: function () {
            return r.call(this);
          },
          set: function (n) {
            (a = '' + n), o.call(this, n);
          },
        }),
        Object.defineProperty(e, t, { enumerable: l.enumerable }),
        {
          getValue: function () {
            return a;
          },
          setValue: function (n) {
            a = '' + n;
          },
          stopTracking: function () {
            (e._valueTracker = null), delete e[t];
          },
        }
      );
    }
  }
  function rd(e) {
    if (!e._valueTracker) {
      var t = Rh(e) ? 'checked' : 'value';
      e._valueTracker = hS(e, t, '' + e[t]);
    }
  }
  function Ih(e) {
    if (!e) return !1;
    var t = e._valueTracker;
    if (!t) return !0;
    var a = t.getValue(),
      l = '';
    return (
      e && (l = Rh(e) ? (e.checked ? 'true' : 'false') : e.value),
      (e = l),
      e !== a ? (t.setValue(e), !0) : !1
    );
  }
  function Au(e) {
    if (((e = e || (typeof document < 'u' ? document : void 0)), typeof e > 'u')) return null;
    try {
      return e.activeElement || e.body;
    } catch {
      return e.body;
    }
  }
  var gS = /[\n"\\]/g;
  function Nt(e) {
    return e.replace(gS, function (t) {
      return '\\' + t.charCodeAt(0).toString(16) + ' ';
    });
  }
  function od(e, t, a, l, r, o, n, u) {
    (e.name = ''),
      n != null && typeof n != 'function' && typeof n != 'symbol' && typeof n != 'boolean'
        ? (e.type = n)
        : e.removeAttribute('type'),
      t != null
        ? n === 'number'
          ? ((t === 0 && e.value === '') || e.value != t) && (e.value = '' + Ot(t))
          : e.value !== '' + Ot(t) && (e.value = '' + Ot(t))
        : (n !== 'submit' && n !== 'reset') || e.removeAttribute('value'),
      t != null
        ? nd(e, n, Ot(t))
        : a != null
          ? nd(e, n, Ot(a))
          : l != null && e.removeAttribute('value'),
      r == null && o != null && (e.defaultChecked = !!o),
      r != null && (e.checked = r && typeof r != 'function' && typeof r != 'symbol'),
      u != null && typeof u != 'function' && typeof u != 'symbol' && typeof u != 'boolean'
        ? (e.name = '' + Ot(u))
        : e.removeAttribute('name');
  }
  function Eh(e, t, a, l, r, o, n, u) {
    if (
      (o != null &&
        typeof o != 'function' &&
        typeof o != 'symbol' &&
        typeof o != 'boolean' &&
        (e.type = o),
      t != null || a != null)
    ) {
      if (!((o !== 'submit' && o !== 'reset') || t != null)) {
        rd(e);
        return;
      }
      (a = a != null ? '' + Ot(a) : ''),
        (t = t != null ? '' + Ot(t) : a),
        u || t === e.value || (e.value = t),
        (e.defaultValue = t);
    }
    (l = l ?? r),
      (l = typeof l != 'function' && typeof l != 'symbol' && !!l),
      (e.checked = u ? e.checked : !!l),
      (e.defaultChecked = !!l),
      n != null &&
        typeof n != 'function' &&
        typeof n != 'symbol' &&
        typeof n != 'boolean' &&
        (e.name = n),
      rd(e);
  }
  function nd(e, t, a) {
    (t === 'number' && Au(e.ownerDocument) === e) ||
      e.defaultValue === '' + a ||
      (e.defaultValue = '' + a);
  }
  function Er(e, t, a, l) {
    if (((e = e.options), t)) {
      t = {};
      for (var r = 0; r < a.length; r++) t['$' + a[r]] = !0;
      for (a = 0; a < e.length; a++)
        (r = t.hasOwnProperty('$' + e[a].value)),
          e[a].selected !== r && (e[a].selected = r),
          r && l && (e[a].defaultSelected = !0);
    } else {
      for (a = '' + Ot(a), t = null, r = 0; r < e.length; r++) {
        if (e[r].value === a) {
          (e[r].selected = !0), l && (e[r].defaultSelected = !0);
          return;
        }
        t !== null || e[r].disabled || (t = e[r]);
      }
      t !== null && (t.selected = !0);
    }
  }
  function Ah(e, t, a) {
    if (t != null && ((t = '' + Ot(t)), t !== e.value && (e.value = t), a == null)) {
      e.defaultValue !== t && (e.defaultValue = t);
      return;
    }
    e.defaultValue = a != null ? '' + Ot(a) : '';
  }
  function Th(e, t, a, l) {
    if (t == null) {
      if (l != null) {
        if (a != null) throw Error(S(92));
        if (Bo(l)) {
          if (1 < l.length) throw Error(S(93));
          l = l[0];
        }
        a = l;
      }
      a == null && (a = ''), (t = a);
    }
    (a = Ot(t)),
      (e.defaultValue = a),
      (l = e.textContent),
      l === a && l !== '' && l !== null && (e.value = l),
      rd(e);
  }
  function Ur(e, t) {
    if (t) {
      var a = e.firstChild;
      if (a && a === e.lastChild && a.nodeType === 3) {
        a.nodeValue = t;
        return;
      }
    }
    e.textContent = t;
  }
  var yS = new Set(
    'animationIterationCount aspectRatio borderImageOutset borderImageSlice borderImageWidth boxFlex boxFlexGroup boxOrdinalGroup columnCount columns flex flexGrow flexPositive flexShrink flexNegative flexOrder gridArea gridRow gridRowEnd gridRowSpan gridRowStart gridColumn gridColumnEnd gridColumnSpan gridColumnStart fontWeight lineClamp lineHeight opacity order orphans scale tabSize widows zIndex zoom fillOpacity floodOpacity stopOpacity strokeDasharray strokeDashoffset strokeMiterlimit strokeOpacity strokeWidth MozAnimationIterationCount MozBoxFlex MozBoxFlexGroup MozLineClamp msAnimationIterationCount msFlex msZoom msFlexGrow msFlexNegative msFlexOrder msFlexPositive msFlexShrink msGridColumn msGridColumnSpan msGridRow msGridRowSpan WebkitAnimationIterationCount WebkitBoxFlex WebKitBoxFlexGroup WebkitBoxOrdinalGroup WebkitColumnCount WebkitColumns WebkitFlex WebkitFlexGrow WebkitFlexPositive WebkitFlexShrink WebkitLineClamp'.split(
      ' '
    )
  );
  function jm(e, t, a) {
    var l = t.indexOf('--') === 0;
    a == null || typeof a == 'boolean' || a === ''
      ? l
        ? e.setProperty(t, '')
        : t === 'float'
          ? (e.cssFloat = '')
          : (e[t] = '')
      : l
        ? e.setProperty(t, a)
        : typeof a != 'number' || a === 0 || yS.has(t)
          ? t === 'float'
            ? (e.cssFloat = a)
            : (e[t] = ('' + a).trim())
          : (e[t] = a + 'px');
  }
  function Mh(e, t, a) {
    if (t != null && typeof t != 'object') throw Error(S(62));
    if (((e = e.style), a != null)) {
      for (var l in a)
        !a.hasOwnProperty(l) ||
          (t != null && t.hasOwnProperty(l)) ||
          (l.indexOf('--') === 0
            ? e.setProperty(l, '')
            : l === 'float'
              ? (e.cssFloat = '')
              : (e[l] = ''));
      for (var r in t) (l = t[r]), t.hasOwnProperty(r) && a[r] !== l && jm(e, r, l);
    } else for (var o in t) t.hasOwnProperty(o) && jm(e, o, t[o]);
  }
  function Zd(e) {
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
  var xS = new Map([
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
    vS =
      /^[\u0000-\u001F ]*j[\r\n\t]*a[\r\n\t]*v[\r\n\t]*a[\r\n\t]*s[\r\n\t]*c[\r\n\t]*r[\r\n\t]*i[\r\n\t]*p[\r\n\t]*t[\r\n\t]*:/i;
  function du(e) {
    return vS.test('' + e)
      ? "javascript:throw new Error('React has blocked a javascript: URL as a security precaution.')"
      : e;
  }
  function Ia() {}
  var ud = null;
  function Wd(e) {
    return (
      (e = e.target || e.srcElement || window),
      e.correspondingUseElement && (e = e.correspondingUseElement),
      e.nodeType === 3 ? e.parentNode : e
    );
  }
  var yr = null,
    Ar = null;
  function Xm(e) {
    var t = jr(e);
    if (t && (e = t.stateNode)) {
      var a = e[ct] || null;
      e: switch (((e = t.stateNode), t.type)) {
        case 'input':
          if (
            (od(
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
              a = a.querySelectorAll('input[name="' + Nt('' + t) + '"][type="radio"]'), t = 0;
              t < a.length;
              t++
            ) {
              var l = a[t];
              if (l !== e && l.form === e.form) {
                var r = l[ct] || null;
                if (!r) throw Error(S(90));
                od(
                  l,
                  r.value,
                  r.defaultValue,
                  r.defaultValue,
                  r.checked,
                  r.defaultChecked,
                  r.type,
                  r.name
                );
              }
            }
            for (t = 0; t < a.length; t++) (l = a[t]), l.form === e.form && Ih(l);
          }
          break e;
        case 'textarea':
          Ah(e, a.value, a.defaultValue);
          break e;
        case 'select':
          (t = a.value), t != null && Er(e, !!a.multiple, t, !1);
      }
    }
  }
  var Ss = !1;
  function Dh(e, t, a) {
    if (Ss) return e(t, a);
    Ss = !0;
    try {
      var l = e(t);
      return l;
    } finally {
      if (
        ((Ss = !1),
        (yr !== null || Ar !== null) &&
          (fi(), yr && ((t = yr), (e = Ar), (Ar = yr = null), Xm(t), e)))
      )
        for (t = 0; t < e.length; t++) Xm(e[t]);
    }
  }
  function Wo(e, t) {
    var a = e.stateNode;
    if (a === null) return null;
    var l = a[ct] || null;
    if (l === null) return null;
    a = l[t];
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
        (l = !l.disabled) ||
          ((e = e.type),
          (l = !(e === 'button' || e === 'input' || e === 'select' || e === 'textarea'))),
          (e = !l);
        break e;
      default:
        e = !1;
    }
    if (e) return null;
    if (a && typeof a != 'function') throw Error(S(231, t, typeof a));
    return a;
  }
  var Da = !(
      typeof window > 'u' ||
      typeof window.document > 'u' ||
      typeof window.document.createElement > 'u'
    ),
    id = !1;
  if (Da)
    try {
      (sr = {}),
        Object.defineProperty(sr, 'passive', {
          get: function () {
            id = !0;
          },
        }),
        window.addEventListener('test', sr, sr),
        window.removeEventListener('test', sr, sr);
    } catch {
      id = !1;
    }
  var sr,
    $a = null,
    Jd = null,
    fu = null;
  function kh() {
    if (fu) return fu;
    var e,
      t = Jd,
      a = t.length,
      l,
      r = 'value' in $a ? $a.value : $a.textContent,
      o = r.length;
    for (e = 0; e < a && t[e] === r[e]; e++);
    var n = a - e;
    for (l = 1; l <= n && t[a - l] === r[o - l]; l++);
    return (fu = r.slice(e, 1 < l ? 1 - l : void 0));
  }
  function cu(e) {
    var t = e.keyCode;
    return (
      'charCode' in e ? ((e = e.charCode), e === 0 && t === 13 && (e = 13)) : (e = t),
      e === 10 && (e = 13),
      32 <= e || e === 13 ? e : 0
    );
  }
  function Zn() {
    return !0;
  }
  function Ym() {
    return !1;
  }
  function mt(e) {
    function t(a, l, r, o, n) {
      (this._reactName = a),
        (this._targetInst = r),
        (this.type = l),
        (this.nativeEvent = o),
        (this.target = n),
        (this.currentTarget = null);
      for (var u in e) e.hasOwnProperty(u) && ((a = e[u]), (this[u] = a ? a(o) : o[u]));
      return (
        (this.isDefaultPrevented = (
          o.defaultPrevented != null ? o.defaultPrevented : o.returnValue === !1
        )
          ? Zn
          : Ym),
        (this.isPropagationStopped = Ym),
        this
      );
    }
    return (
      ye(t.prototype, {
        preventDefault: function () {
          this.defaultPrevented = !0;
          var a = this.nativeEvent;
          a &&
            (a.preventDefault
              ? a.preventDefault()
              : typeof a.returnValue != 'unknown' && (a.returnValue = !1),
            (this.isDefaultPrevented = Zn));
        },
        stopPropagation: function () {
          var a = this.nativeEvent;
          a &&
            (a.stopPropagation
              ? a.stopPropagation()
              : typeof a.cancelBubble != 'unknown' && (a.cancelBubble = !0),
            (this.isPropagationStopped = Zn));
        },
        persist: function () {},
        isPersistent: Zn,
      }),
      t
    );
  }
  var _l = {
      eventPhase: 0,
      bubbles: 0,
      cancelable: 0,
      timeStamp: function (e) {
        return e.timeStamp || Date.now();
      },
      defaultPrevented: 0,
      isTrusted: 0,
    },
    ei = mt(_l),
    gn = ye({}, _l, { view: 0, detail: 0 }),
    LS = mt(gn),
    bs,
    Cs,
    Io,
    ti = ye({}, gn, {
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
      getModifierState: $d,
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
          : (e !== Io &&
              (Io && e.type === 'mousemove'
                ? ((bs = e.screenX - Io.screenX), (Cs = e.screenY - Io.screenY))
                : (Cs = bs = 0),
              (Io = e)),
            bs);
      },
      movementY: function (e) {
        return 'movementY' in e ? e.movementY : Cs;
      },
    }),
    Km = mt(ti),
    SS = ye({}, ti, { dataTransfer: 0 }),
    bS = mt(SS),
    CS = ye({}, gn, { relatedTarget: 0 }),
    ws = mt(CS),
    wS = ye({}, _l, { animationName: 0, elapsedTime: 0, pseudoElement: 0 }),
    RS = mt(wS),
    IS = ye({}, _l, {
      clipboardData: function (e) {
        return 'clipboardData' in e ? e.clipboardData : window.clipboardData;
      },
    }),
    ES = mt(IS),
    AS = ye({}, _l, { data: 0 }),
    Qm = mt(AS),
    TS = {
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
    MS = {
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
    DS = { Alt: 'altKey', Control: 'ctrlKey', Meta: 'metaKey', Shift: 'shiftKey' };
  function kS(e) {
    var t = this.nativeEvent;
    return t.getModifierState ? t.getModifierState(e) : (e = DS[e]) ? !!t[e] : !1;
  }
  function $d() {
    return kS;
  }
  var BS = ye({}, gn, {
      key: function (e) {
        if (e.key) {
          var t = TS[e.key] || e.key;
          if (t !== 'Unidentified') return t;
        }
        return e.type === 'keypress'
          ? ((e = cu(e)), e === 13 ? 'Enter' : String.fromCharCode(e))
          : e.type === 'keydown' || e.type === 'keyup'
            ? MS[e.keyCode] || 'Unidentified'
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
      getModifierState: $d,
      charCode: function (e) {
        return e.type === 'keypress' ? cu(e) : 0;
      },
      keyCode: function (e) {
        return e.type === 'keydown' || e.type === 'keyup' ? e.keyCode : 0;
      },
      which: function (e) {
        return e.type === 'keypress'
          ? cu(e)
          : e.type === 'keydown' || e.type === 'keyup'
            ? e.keyCode
            : 0;
      },
    }),
    OS = mt(BS),
    US = ye({}, ti, {
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
    Zm = mt(US),
    HS = ye({}, gn, {
      touches: 0,
      targetTouches: 0,
      changedTouches: 0,
      altKey: 0,
      metaKey: 0,
      ctrlKey: 0,
      shiftKey: 0,
      getModifierState: $d,
    }),
    NS = mt(HS),
    PS = ye({}, _l, { propertyName: 0, elapsedTime: 0, pseudoElement: 0 }),
    _S = mt(PS),
    zS = ye({}, ti, {
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
    FS = mt(zS),
    qS = ye({}, _l, { newState: 0, oldState: 0 }),
    GS = mt(qS),
    VS = [9, 13, 27, 32],
    ef = Da && 'CompositionEvent' in window,
    No = null;
  Da && 'documentMode' in document && (No = document.documentMode);
  var jS = Da && 'TextEvent' in window && !No,
    Bh = Da && (!ef || (No && 8 < No && 11 >= No)),
    Wm = ' ',
    Jm = !1;
  function Oh(e, t) {
    switch (e) {
      case 'keyup':
        return VS.indexOf(t.keyCode) !== -1;
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
  function Uh(e) {
    return (e = e.detail), typeof e == 'object' && 'data' in e ? e.data : null;
  }
  var xr = !1;
  function XS(e, t) {
    switch (e) {
      case 'compositionend':
        return Uh(t);
      case 'keypress':
        return t.which !== 32 ? null : ((Jm = !0), Wm);
      case 'textInput':
        return (e = t.data), e === Wm && Jm ? null : e;
      default:
        return null;
    }
  }
  function YS(e, t) {
    if (xr)
      return e === 'compositionend' || (!ef && Oh(e, t))
        ? ((e = kh()), (fu = Jd = $a = null), (xr = !1), e)
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
        return Bh && t.locale !== 'ko' ? null : t.data;
      default:
        return null;
    }
  }
  var KS = {
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
  function $m(e) {
    var t = e && e.nodeName && e.nodeName.toLowerCase();
    return t === 'input' ? !!KS[e.type] : t === 'textarea';
  }
  function Hh(e, t, a, l) {
    yr ? (Ar ? Ar.push(l) : (Ar = [l])) : (yr = l),
      (t = Xu(t, 'onChange')),
      0 < t.length &&
        ((a = new ei('onChange', 'change', null, a, l)), e.push({ event: a, listeners: t }));
  }
  var Po = null,
    Jo = null;
  function QS(e) {
    Dy(e, 0);
  }
  function ai(e) {
    var t = Oo(e);
    if (Ih(t)) return e;
  }
  function ep(e, t) {
    if (e === 'change') return t;
  }
  var Nh = !1;
  Da &&
    (Da
      ? ((Jn = 'oninput' in document),
        Jn ||
          ((Rs = document.createElement('div')),
          Rs.setAttribute('oninput', 'return;'),
          (Jn = typeof Rs.oninput == 'function')),
        (Wn = Jn))
      : (Wn = !1),
    (Nh = Wn && (!document.documentMode || 9 < document.documentMode)));
  var Wn, Jn, Rs;
  function tp() {
    Po && (Po.detachEvent('onpropertychange', Ph), (Jo = Po = null));
  }
  function Ph(e) {
    if (e.propertyName === 'value' && ai(Jo)) {
      var t = [];
      Hh(t, Jo, e, Wd(e)), Dh(QS, t);
    }
  }
  function ZS(e, t, a) {
    e === 'focusin'
      ? (tp(), (Po = t), (Jo = a), Po.attachEvent('onpropertychange', Ph))
      : e === 'focusout' && tp();
  }
  function WS(e) {
    if (e === 'selectionchange' || e === 'keyup' || e === 'keydown') return ai(Jo);
  }
  function JS(e, t) {
    if (e === 'click') return ai(t);
  }
  function $S(e, t) {
    if (e === 'input' || e === 'change') return ai(t);
  }
  function eb(e, t) {
    return (e === t && (e !== 0 || 1 / e === 1 / t)) || (e !== e && t !== t);
  }
  var Et = typeof Object.is == 'function' ? Object.is : eb;
  function $o(e, t) {
    if (Et(e, t)) return !0;
    if (typeof e != 'object' || e === null || typeof t != 'object' || t === null) return !1;
    var a = Object.keys(e),
      l = Object.keys(t);
    if (a.length !== l.length) return !1;
    for (l = 0; l < a.length; l++) {
      var r = a[l];
      if (!ad.call(t, r) || !Et(e[r], t[r])) return !1;
    }
    return !0;
  }
  function ap(e) {
    for (; e && e.firstChild; ) e = e.firstChild;
    return e;
  }
  function lp(e, t) {
    var a = ap(e);
    e = 0;
    for (var l; a; ) {
      if (a.nodeType === 3) {
        if (((l = e + a.textContent.length), e <= t && l >= t)) return { node: a, offset: t - e };
        e = l;
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
      a = ap(a);
    }
  }
  function _h(e, t) {
    return e && t
      ? e === t
        ? !0
        : e && e.nodeType === 3
          ? !1
          : t && t.nodeType === 3
            ? _h(e, t.parentNode)
            : 'contains' in e
              ? e.contains(t)
              : e.compareDocumentPosition
                ? !!(e.compareDocumentPosition(t) & 16)
                : !1
      : !1;
  }
  function zh(e) {
    e =
      e != null && e.ownerDocument != null && e.ownerDocument.defaultView != null
        ? e.ownerDocument.defaultView
        : window;
    for (var t = Au(e.document); t instanceof e.HTMLIFrameElement; ) {
      try {
        var a = typeof t.contentWindow.location.href == 'string';
      } catch {
        a = !1;
      }
      if (a) e = t.contentWindow;
      else break;
      t = Au(e.document);
    }
    return t;
  }
  function tf(e) {
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
  var tb = Da && 'documentMode' in document && 11 >= document.documentMode,
    vr = null,
    sd = null,
    _o = null,
    dd = !1;
  function rp(e, t, a) {
    var l = a.window === a ? a.document : a.nodeType === 9 ? a : a.ownerDocument;
    dd ||
      vr == null ||
      vr !== Au(l) ||
      ((l = vr),
      'selectionStart' in l && tf(l)
        ? (l = { start: l.selectionStart, end: l.selectionEnd })
        : ((l = ((l.ownerDocument && l.ownerDocument.defaultView) || window).getSelection()),
          (l = {
            anchorNode: l.anchorNode,
            anchorOffset: l.anchorOffset,
            focusNode: l.focusNode,
            focusOffset: l.focusOffset,
          })),
      (_o && $o(_o, l)) ||
        ((_o = l),
        (l = Xu(sd, 'onSelect')),
        0 < l.length &&
          ((t = new ei('onSelect', 'select', null, t, a)),
          e.push({ event: t, listeners: l }),
          (t.target = vr))));
  }
  function Cl(e, t) {
    var a = {};
    return (
      (a[e.toLowerCase()] = t.toLowerCase()),
      (a['Webkit' + e] = 'webkit' + t),
      (a['Moz' + e] = 'moz' + t),
      a
    );
  }
  var Lr = {
      animationend: Cl('Animation', 'AnimationEnd'),
      animationiteration: Cl('Animation', 'AnimationIteration'),
      animationstart: Cl('Animation', 'AnimationStart'),
      transitionrun: Cl('Transition', 'TransitionRun'),
      transitionstart: Cl('Transition', 'TransitionStart'),
      transitioncancel: Cl('Transition', 'TransitionCancel'),
      transitionend: Cl('Transition', 'TransitionEnd'),
    },
    Is = {},
    Fh = {};
  Da &&
    ((Fh = document.createElement('div').style),
    'AnimationEvent' in window ||
      (delete Lr.animationend.animation,
      delete Lr.animationiteration.animation,
      delete Lr.animationstart.animation),
    'TransitionEvent' in window || delete Lr.transitionend.transition);
  function zl(e) {
    if (Is[e]) return Is[e];
    if (!Lr[e]) return e;
    var t = Lr[e],
      a;
    for (a in t) if (t.hasOwnProperty(a) && a in Fh) return (Is[e] = t[a]);
    return e;
  }
  var qh = zl('animationend'),
    Gh = zl('animationiteration'),
    Vh = zl('animationstart'),
    ab = zl('transitionrun'),
    lb = zl('transitionstart'),
    rb = zl('transitioncancel'),
    jh = zl('transitionend'),
    Xh = new Map(),
    fd =
      'abort auxClick beforeToggle cancel canPlay canPlayThrough click close contextMenu copy cut drag dragEnd dragEnter dragExit dragLeave dragOver dragStart drop durationChange emptied encrypted ended error gotPointerCapture input invalid keyDown keyPress keyUp load loadedData loadedMetadata loadStart lostPointerCapture mouseDown mouseMove mouseOut mouseOver mouseUp paste pause play playing pointerCancel pointerDown pointerMove pointerOut pointerOver pointerUp progress rateChange reset resize seeked seeking stalled submit suspend timeUpdate touchCancel touchEnd touchStart volumeChange scroll toggle touchMove waiting wheel'.split(
        ' '
      );
  fd.push('scrollEnd');
  function Kt(e, t) {
    Xh.set(e, t), Pl(t, [e]);
  }
  var Tu =
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
    Bt = [],
    Sr = 0,
    af = 0;
  function li() {
    for (var e = Sr, t = (af = Sr = 0); t < e; ) {
      var a = Bt[t];
      Bt[t++] = null;
      var l = Bt[t];
      Bt[t++] = null;
      var r = Bt[t];
      Bt[t++] = null;
      var o = Bt[t];
      if (((Bt[t++] = null), l !== null && r !== null)) {
        var n = l.pending;
        n === null ? (r.next = r) : ((r.next = n.next), (n.next = r)), (l.pending = r);
      }
      o !== 0 && Yh(a, r, o);
    }
  }
  function ri(e, t, a, l) {
    (Bt[Sr++] = e),
      (Bt[Sr++] = t),
      (Bt[Sr++] = a),
      (Bt[Sr++] = l),
      (af |= l),
      (e.lanes |= l),
      (e = e.alternate),
      e !== null && (e.lanes |= l);
  }
  function lf(e, t, a, l) {
    return ri(e, t, a, l), Mu(e);
  }
  function Fl(e, t) {
    return ri(e, null, null, t), Mu(e);
  }
  function Yh(e, t, a) {
    e.lanes |= a;
    var l = e.alternate;
    l !== null && (l.lanes |= a);
    for (var r = !1, o = e.return; o !== null; )
      (o.childLanes |= a),
        (l = o.alternate),
        l !== null && (l.childLanes |= a),
        o.tag === 22 && ((e = o.stateNode), e === null || e._visibility & 1 || (r = !0)),
        (e = o),
        (o = o.return);
    return e.tag === 3
      ? ((o = e.stateNode),
        r &&
          t !== null &&
          ((r = 31 - Rt(a)),
          (e = o.hiddenUpdates),
          (l = e[r]),
          l === null ? (e[r] = [t]) : l.push(t),
          (t.lane = a | 536870912)),
        o)
      : null;
  }
  function Mu(e) {
    if (50 < Ko) throw ((Ko = 0), (kd = null), Error(S(185)));
    for (var t = e.return; t !== null; ) (e = t), (t = e.return);
    return e.tag === 3 ? e.stateNode : null;
  }
  var br = {};
  function ob(e, t, a, l) {
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
      (this.mode = l),
      (this.subtreeFlags = this.flags = 0),
      (this.deletions = null),
      (this.childLanes = this.lanes = 0),
      (this.alternate = null);
  }
  function St(e, t, a, l) {
    return new ob(e, t, a, l);
  }
  function rf(e) {
    return (e = e.prototype), !(!e || !e.isReactComponent);
  }
  function Aa(e, t) {
    var a = e.alternate;
    return (
      a === null
        ? ((a = St(e.tag, t, e.key, e.mode)),
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
  function Kh(e, t) {
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
  function mu(e, t, a, l, r, o) {
    var n = 0;
    if (((l = e), typeof e == 'function')) rf(e) && (n = 1);
    else if (typeof e == 'string')
      n = i0(e, a, ua.current) ? 26 : e === 'html' || e === 'head' || e === 'body' ? 27 : 5;
    else
      e: switch (e) {
        case Js:
          return (e = St(31, a, t, r)), (e.elementType = Js), (e.lanes = o), e;
        case pr:
          return Tl(a.children, r, o, t);
        case mh:
          (n = 8), (r |= 24);
          break;
        case Qs:
          return (e = St(12, a, t, r | 2)), (e.elementType = Qs), (e.lanes = o), e;
        case Zs:
          return (e = St(13, a, t, r)), (e.elementType = Zs), (e.lanes = o), e;
        case Ws:
          return (e = St(19, a, t, r)), (e.elementType = Ws), (e.lanes = o), e;
        default:
          if (typeof e == 'object' && e !== null)
            switch (e.$$typeof) {
              case Ra:
                n = 10;
                break e;
              case ph:
                n = 9;
                break e;
              case Vd:
                n = 11;
                break e;
              case jd:
                n = 14;
                break e;
              case Xa:
                (n = 16), (l = null);
                break e;
            }
          (n = 29), (a = Error(S(130, e === null ? 'null' : typeof e, ''))), (l = null);
      }
    return (t = St(n, a, t, r)), (t.elementType = e), (t.type = l), (t.lanes = o), t;
  }
  function Tl(e, t, a, l) {
    return (e = St(7, e, l, t)), (e.lanes = a), e;
  }
  function Es(e, t, a) {
    return (e = St(6, e, null, t)), (e.lanes = a), e;
  }
  function Qh(e) {
    var t = St(18, null, null, 0);
    return (t.stateNode = e), t;
  }
  function As(e, t, a) {
    return (
      (t = St(4, e.children !== null ? e.children : [], e.key, t)),
      (t.lanes = a),
      (t.stateNode = {
        containerInfo: e.containerInfo,
        pendingChildren: null,
        implementation: e.implementation,
      }),
      t
    );
  }
  var op = new WeakMap();
  function Pt(e, t) {
    if (typeof e == 'object' && e !== null) {
      var a = op.get(e);
      return a !== void 0 ? a : ((t = { value: e, source: t, stack: zm(t) }), op.set(e, t), t);
    }
    return { value: e, source: t, stack: zm(t) };
  }
  var Cr = [],
    wr = 0,
    Du = null,
    en = 0,
    Ut = [],
    Ht = 0,
    cl = null,
    ra = 1,
    oa = '';
  function Ca(e, t) {
    (Cr[wr++] = en), (Cr[wr++] = Du), (Du = e), (en = t);
  }
  function Zh(e, t, a) {
    (Ut[Ht++] = ra), (Ut[Ht++] = oa), (Ut[Ht++] = cl), (cl = e);
    var l = ra;
    e = oa;
    var r = 32 - Rt(l) - 1;
    (l &= ~(1 << r)), (a += 1);
    var o = 32 - Rt(t) + r;
    if (30 < o) {
      var n = r - (r % 5);
      (o = (l & ((1 << n) - 1)).toString(32)),
        (l >>= n),
        (r -= n),
        (ra = (1 << (32 - Rt(t) + r)) | (a << r) | l),
        (oa = o + e);
    } else (ra = (1 << o) | (a << r) | l), (oa = e);
  }
  function of(e) {
    e.return !== null && (Ca(e, 1), Zh(e, 1, 0));
  }
  function nf(e) {
    for (; e === Du; ) (Du = Cr[--wr]), (Cr[wr] = null), (en = Cr[--wr]), (Cr[wr] = null);
    for (; e === cl; )
      (cl = Ut[--Ht]),
        (Ut[Ht] = null),
        (oa = Ut[--Ht]),
        (Ut[Ht] = null),
        (ra = Ut[--Ht]),
        (Ut[Ht] = null);
  }
  function Wh(e, t) {
    (Ut[Ht++] = ra), (Ut[Ht++] = oa), (Ut[Ht++] = cl), (ra = t.id), (oa = t.overflow), (cl = e);
  }
  var je = null,
    ge = null,
    $ = !1,
    rl = null,
    _t = !1,
    cd = Error(S(519));
  function ml(e) {
    var t = Error(
      S(418, 1 < arguments.length && arguments[1] !== void 0 && arguments[1] ? 'text' : 'HTML', '')
    );
    throw (tn(Pt(t, e)), cd);
  }
  function np(e) {
    var t = e.stateNode,
      a = e.type,
      l = e.memoizedProps;
    switch (((t[Ve] = e), (t[ct] = l), a)) {
      case 'dialog':
        Y('cancel', t), Y('close', t);
        break;
      case 'iframe':
      case 'object':
      case 'embed':
        Y('load', t);
        break;
      case 'video':
      case 'audio':
        for (a = 0; a < on.length; a++) Y(on[a], t);
        break;
      case 'source':
        Y('error', t);
        break;
      case 'img':
      case 'image':
      case 'link':
        Y('error', t), Y('load', t);
        break;
      case 'details':
        Y('toggle', t);
        break;
      case 'input':
        Y('invalid', t),
          Eh(t, l.value, l.defaultValue, l.checked, l.defaultChecked, l.type, l.name, !0);
        break;
      case 'select':
        Y('invalid', t);
        break;
      case 'textarea':
        Y('invalid', t), Th(t, l.value, l.defaultValue, l.children);
    }
    (a = l.children),
      (typeof a != 'string' && typeof a != 'number' && typeof a != 'bigint') ||
      t.textContent === '' + a ||
      l.suppressHydrationWarning === !0 ||
      By(t.textContent, a)
        ? (l.popover != null && (Y('beforetoggle', t), Y('toggle', t)),
          l.onScroll != null && Y('scroll', t),
          l.onScrollEnd != null && Y('scrollend', t),
          l.onClick != null && (t.onclick = Ia),
          (t = !0))
        : (t = !1),
      t || ml(e, !0);
  }
  function up(e) {
    for (je = e.return; je; )
      switch (je.tag) {
        case 5:
        case 31:
        case 13:
          _t = !1;
          return;
        case 27:
        case 3:
          _t = !0;
          return;
        default:
          je = je.return;
      }
  }
  function dr(e) {
    if (e !== je) return !1;
    if (!$) return up(e), ($ = !0), !1;
    var t = e.tag,
      a;
    if (
      ((a = t !== 3 && t !== 27) &&
        ((a = t === 5) &&
          ((a = e.type), (a = !(a !== 'form' && a !== 'button') || Nd(e.type, e.memoizedProps))),
        (a = !a)),
      a && ge && ml(e),
      up(e),
      t === 13)
    ) {
      if (((e = e.memoizedState), (e = e !== null ? e.dehydrated : null), !e)) throw Error(S(317));
      ge = Qp(e);
    } else if (t === 31) {
      if (((e = e.memoizedState), (e = e !== null ? e.dehydrated : null), !e)) throw Error(S(317));
      ge = Qp(e);
    } else
      t === 27
        ? ((t = ge), yl(e.type) ? ((e = Fd), (Fd = null), (ge = e)) : (ge = t))
        : (ge = je ? Ft(e.stateNode.nextSibling) : null);
    return !0;
  }
  function Bl() {
    (ge = je = null), ($ = !1);
  }
  function Ts() {
    var e = rl;
    return e !== null && (dt === null ? (dt = e) : dt.push.apply(dt, e), (rl = null)), e;
  }
  function tn(e) {
    rl === null ? (rl = [e]) : rl.push(e);
  }
  var md = ia(null),
    ql = null,
    Ea = null;
  function Ka(e, t, a) {
    ce(md, t._currentValue), (t._currentValue = a);
  }
  function Ta(e) {
    (e._currentValue = md.current), _e(md);
  }
  function pd(e, t, a) {
    for (; e !== null; ) {
      var l = e.alternate;
      if (
        ((e.childLanes & t) !== t
          ? ((e.childLanes |= t), l !== null && (l.childLanes |= t))
          : l !== null && (l.childLanes & t) !== t && (l.childLanes |= t),
        e === a)
      )
        break;
      e = e.return;
    }
  }
  function hd(e, t, a, l) {
    var r = e.child;
    for (r !== null && (r.return = e); r !== null; ) {
      var o = r.dependencies;
      if (o !== null) {
        var n = r.child;
        o = o.firstContext;
        e: for (; o !== null; ) {
          var u = o;
          o = r;
          for (var i = 0; i < t.length; i++)
            if (u.context === t[i]) {
              (o.lanes |= a),
                (u = o.alternate),
                u !== null && (u.lanes |= a),
                pd(o.return, a, e),
                l || (n = null);
              break e;
            }
          o = u.next;
        }
      } else if (r.tag === 18) {
        if (((n = r.return), n === null)) throw Error(S(341));
        (n.lanes |= a), (o = n.alternate), o !== null && (o.lanes |= a), pd(n, a, e), (n = null);
      } else n = r.child;
      if (n !== null) n.return = r;
      else
        for (n = r; n !== null; ) {
          if (n === e) {
            n = null;
            break;
          }
          if (((r = n.sibling), r !== null)) {
            (r.return = n.return), (n = r);
            break;
          }
          n = n.return;
        }
      r = n;
    }
  }
  function Xr(e, t, a, l) {
    e = null;
    for (var r = t, o = !1; r !== null; ) {
      if (!o) {
        if ((r.flags & 524288) !== 0) o = !0;
        else if ((r.flags & 262144) !== 0) break;
      }
      if (r.tag === 10) {
        var n = r.alternate;
        if (n === null) throw Error(S(387));
        if (((n = n.memoizedProps), n !== null)) {
          var u = r.type;
          Et(r.pendingProps.value, n.value) || (e !== null ? e.push(u) : (e = [u]));
        }
      } else if (r === wu.current) {
        if (((n = r.alternate), n === null)) throw Error(S(387));
        n.memoizedState.memoizedState !== r.memoizedState.memoizedState &&
          (e !== null ? e.push(un) : (e = [un]));
      }
      r = r.return;
    }
    e !== null && hd(t, e, a, l), (t.flags |= 262144);
  }
  function ku(e) {
    for (e = e.firstContext; e !== null; ) {
      if (!Et(e.context._currentValue, e.memoizedValue)) return !0;
      e = e.next;
    }
    return !1;
  }
  function Ol(e) {
    (ql = e), (Ea = null), (e = e.dependencies), e !== null && (e.firstContext = null);
  }
  function Xe(e) {
    return Jh(ql, e);
  }
  function $n(e, t) {
    return ql === null && Ol(e), Jh(e, t);
  }
  function Jh(e, t) {
    var a = t._currentValue;
    if (((t = { context: t, memoizedValue: a, next: null }), Ea === null)) {
      if (e === null) throw Error(S(308));
      (Ea = t), (e.dependencies = { lanes: 0, firstContext: t }), (e.flags |= 524288);
    } else Ea = Ea.next = t;
    return a;
  }
  var nb =
      typeof AbortController < 'u'
        ? AbortController
        : function () {
            var e = [],
              t = (this.signal = {
                aborted: !1,
                addEventListener: function (a, l) {
                  e.push(l);
                },
              });
            this.abort = function () {
              (t.aborted = !0),
                e.forEach(function (a) {
                  return a();
                });
            };
          },
    ub = ke.unstable_scheduleCallback,
    ib = ke.unstable_NormalPriority,
    Ae = {
      $$typeof: Ra,
      Consumer: null,
      Provider: null,
      _currentValue: null,
      _currentValue2: null,
      _threadCount: 0,
    };
  function uf() {
    return { controller: new nb(), data: new Map(), refCount: 0 };
  }
  function yn(e) {
    e.refCount--,
      e.refCount === 0 &&
        ub(ib, function () {
          e.controller.abort();
        });
  }
  var zo = null,
    gd = 0,
    Hr = 0,
    Tr = null;
  function sb(e, t) {
    if (zo === null) {
      var a = (zo = []);
      (gd = 0),
        (Hr = Bf()),
        (Tr = {
          status: 'pending',
          value: void 0,
          then: function (l) {
            a.push(l);
          },
        });
    }
    return gd++, t.then(ip, ip), t;
  }
  function ip() {
    if (--gd === 0 && zo !== null) {
      Tr !== null && (Tr.status = 'fulfilled');
      var e = zo;
      (zo = null), (Hr = 0), (Tr = null);
      for (var t = 0; t < e.length; t++) (0, e[t])();
    }
  }
  function db(e, t) {
    var a = [],
      l = {
        status: 'pending',
        value: null,
        reason: null,
        then: function (r) {
          a.push(r);
        },
      };
    return (
      e.then(
        function () {
          (l.status = 'fulfilled'), (l.value = t);
          for (var r = 0; r < a.length; r++) (0, a[r])(t);
        },
        function (r) {
          for (l.status = 'rejected', l.reason = r, r = 0; r < a.length; r++) (0, a[r])(void 0);
        }
      ),
      l
    );
  }
  var sp = N.S;
  N.S = function (e, t) {
    (cy = Ct()),
      typeof t == 'object' && t !== null && typeof t.then == 'function' && sb(e, t),
      sp !== null && sp(e, t);
  };
  var Ml = ia(null);
  function sf() {
    var e = Ml.current;
    return e !== null ? e : se.pooledCache;
  }
  function pu(e, t) {
    t === null ? ce(Ml, Ml.current) : ce(Ml, t.pool);
  }
  function $h() {
    var e = sf();
    return e === null ? null : { parent: Ae._currentValue, pool: e };
  }
  var Yr = Error(S(460)),
    df = Error(S(474)),
    oi = Error(S(542)),
    Bu = { then: function () {} };
  function dp(e) {
    return (e = e.status), e === 'fulfilled' || e === 'rejected';
  }
  function eg(e, t, a) {
    switch (
      ((a = e[a]), a === void 0 ? e.push(t) : a !== t && (t.then(Ia, Ia), (t = a)), t.status)
    ) {
      case 'fulfilled':
        return t.value;
      case 'rejected':
        throw ((e = t.reason), cp(e), e);
      default:
        if (typeof t.status == 'string') t.then(Ia, Ia);
        else {
          if (((e = se), e !== null && 100 < e.shellSuspendCounter)) throw Error(S(482));
          (e = t),
            (e.status = 'pending'),
            e.then(
              function (l) {
                if (t.status === 'pending') {
                  var r = t;
                  (r.status = 'fulfilled'), (r.value = l);
                }
              },
              function (l) {
                if (t.status === 'pending') {
                  var r = t;
                  (r.status = 'rejected'), (r.reason = l);
                }
              }
            );
        }
        switch (t.status) {
          case 'fulfilled':
            return t.value;
          case 'rejected':
            throw ((e = t.reason), cp(e), e);
        }
        throw ((Dl = t), Yr);
    }
  }
  function Il(e) {
    try {
      var t = e._init;
      return t(e._payload);
    } catch (a) {
      throw a !== null && typeof a == 'object' && typeof a.then == 'function' ? ((Dl = a), Yr) : a;
    }
  }
  var Dl = null;
  function fp() {
    if (Dl === null) throw Error(S(459));
    var e = Dl;
    return (Dl = null), e;
  }
  function cp(e) {
    if (e === Yr || e === oi) throw Error(S(483));
  }
  var Mr = null,
    an = 0;
  function eu(e) {
    var t = an;
    return (an += 1), Mr === null && (Mr = []), eg(Mr, e, t);
  }
  function Eo(e, t) {
    (t = t.props.ref), (e.ref = t !== void 0 ? t : null);
  }
  function tu(e, t) {
    throw t.$$typeof === ZL
      ? Error(S(525))
      : ((e = Object.prototype.toString.call(t)),
        Error(
          S(
            31,
            e === '[object Object]' ? 'object with keys {' + Object.keys(t).join(', ') + '}' : e
          )
        ));
  }
  function tg(e) {
    function t(c, m) {
      if (e) {
        var g = c.deletions;
        g === null ? ((c.deletions = [m]), (c.flags |= 16)) : g.push(m);
      }
    }
    function a(c, m) {
      if (!e) return null;
      for (; m !== null; ) t(c, m), (m = m.sibling);
      return null;
    }
    function l(c) {
      for (var m = new Map(); c !== null; )
        c.key !== null ? m.set(c.key, c) : m.set(c.index, c), (c = c.sibling);
      return m;
    }
    function r(c, m) {
      return (c = Aa(c, m)), (c.index = 0), (c.sibling = null), c;
    }
    function o(c, m, g) {
      return (
        (c.index = g),
        e
          ? ((g = c.alternate),
            g !== null
              ? ((g = g.index), g < m ? ((c.flags |= 67108866), m) : g)
              : ((c.flags |= 67108866), m))
          : ((c.flags |= 1048576), m)
      );
    }
    function n(c) {
      return e && c.alternate === null && (c.flags |= 67108866), c;
    }
    function u(c, m, g, y) {
      return m === null || m.tag !== 6
        ? ((m = Es(g, c.mode, y)), (m.return = c), m)
        : ((m = r(m, g)), (m.return = c), m);
    }
    function i(c, m, g, y) {
      var w = g.type;
      return w === pr
        ? f(c, m, g.props.children, y, g.key)
        : m !== null &&
            (m.elementType === w ||
              (typeof w == 'object' && w !== null && w.$$typeof === Xa && Il(w) === m.type))
          ? ((m = r(m, g.props)), Eo(m, g), (m.return = c), m)
          : ((m = mu(g.type, g.key, g.props, null, c.mode, y)), Eo(m, g), (m.return = c), m);
    }
    function s(c, m, g, y) {
      return m === null ||
        m.tag !== 4 ||
        m.stateNode.containerInfo !== g.containerInfo ||
        m.stateNode.implementation !== g.implementation
        ? ((m = As(g, c.mode, y)), (m.return = c), m)
        : ((m = r(m, g.children || [])), (m.return = c), m);
    }
    function f(c, m, g, y, w) {
      return m === null || m.tag !== 7
        ? ((m = Tl(g, c.mode, y, w)), (m.return = c), m)
        : ((m = r(m, g)), (m.return = c), m);
    }
    function d(c, m, g) {
      if ((typeof m == 'string' && m !== '') || typeof m == 'number' || typeof m == 'bigint')
        return (m = Es('' + m, c.mode, g)), (m.return = c), m;
      if (typeof m == 'object' && m !== null) {
        switch (m.$$typeof) {
          case jn:
            return (g = mu(m.type, m.key, m.props, null, c.mode, g)), Eo(g, m), (g.return = c), g;
          case ko:
            return (m = As(m, c.mode, g)), (m.return = c), m;
          case Xa:
            return (m = Il(m)), d(c, m, g);
        }
        if (Bo(m) || Ro(m)) return (m = Tl(m, c.mode, g, null)), (m.return = c), m;
        if (typeof m.then == 'function') return d(c, eu(m), g);
        if (m.$$typeof === Ra) return d(c, $n(c, m), g);
        tu(c, m);
      }
      return null;
    }
    function p(c, m, g, y) {
      var w = m !== null ? m.key : null;
      if ((typeof g == 'string' && g !== '') || typeof g == 'number' || typeof g == 'bigint')
        return w !== null ? null : u(c, m, '' + g, y);
      if (typeof g == 'object' && g !== null) {
        switch (g.$$typeof) {
          case jn:
            return g.key === w ? i(c, m, g, y) : null;
          case ko:
            return g.key === w ? s(c, m, g, y) : null;
          case Xa:
            return (g = Il(g)), p(c, m, g, y);
        }
        if (Bo(g) || Ro(g)) return w !== null ? null : f(c, m, g, y, null);
        if (typeof g.then == 'function') return p(c, m, eu(g), y);
        if (g.$$typeof === Ra) return p(c, m, $n(c, g), y);
        tu(c, g);
      }
      return null;
    }
    function h(c, m, g, y, w) {
      if ((typeof y == 'string' && y !== '') || typeof y == 'number' || typeof y == 'bigint')
        return (c = c.get(g) || null), u(m, c, '' + y, w);
      if (typeof y == 'object' && y !== null) {
        switch (y.$$typeof) {
          case jn:
            return (c = c.get(y.key === null ? g : y.key) || null), i(m, c, y, w);
          case ko:
            return (c = c.get(y.key === null ? g : y.key) || null), s(m, c, y, w);
          case Xa:
            return (y = Il(y)), h(c, m, g, y, w);
        }
        if (Bo(y) || Ro(y)) return (c = c.get(g) || null), f(m, c, y, w, null);
        if (typeof y.then == 'function') return h(c, m, g, eu(y), w);
        if (y.$$typeof === Ra) return h(c, m, g, $n(m, y), w);
        tu(m, y);
      }
      return null;
    }
    function x(c, m, g, y) {
      for (var w = null, U = null, C = m, b = (m = 0), M = null; C !== null && b < g.length; b++) {
        C.index > b ? ((M = C), (C = null)) : (M = C.sibling);
        var _ = p(c, C, g[b], y);
        if (_ === null) {
          C === null && (C = M);
          break;
        }
        e && C && _.alternate === null && t(c, C),
          (m = o(_, m, b)),
          U === null ? (w = _) : (U.sibling = _),
          (U = _),
          (C = M);
      }
      if (b === g.length) return a(c, C), $ && Ca(c, b), w;
      if (C === null) {
        for (; b < g.length; b++)
          (C = d(c, g[b], y)),
            C !== null && ((m = o(C, m, b)), U === null ? (w = C) : (U.sibling = C), (U = C));
        return $ && Ca(c, b), w;
      }
      for (C = l(C); b < g.length; b++)
        (M = h(C, c, b, g[b], y)),
          M !== null &&
            (e && M.alternate !== null && C.delete(M.key === null ? b : M.key),
            (m = o(M, m, b)),
            U === null ? (w = M) : (U.sibling = M),
            (U = M));
      return (
        e &&
          C.forEach(function (Ue) {
            return t(c, Ue);
          }),
        $ && Ca(c, b),
        w
      );
    }
    function v(c, m, g, y) {
      if (g == null) throw Error(S(151));
      for (
        var w = null, U = null, C = m, b = (m = 0), M = null, _ = g.next();
        C !== null && !_.done;
        b++, _ = g.next()
      ) {
        C.index > b ? ((M = C), (C = null)) : (M = C.sibling);
        var Ue = p(c, C, _.value, y);
        if (Ue === null) {
          C === null && (C = M);
          break;
        }
        e && C && Ue.alternate === null && t(c, C),
          (m = o(Ue, m, b)),
          U === null ? (w = Ue) : (U.sibling = Ue),
          (U = Ue),
          (C = M);
      }
      if (_.done) return a(c, C), $ && Ca(c, b), w;
      if (C === null) {
        for (; !_.done; b++, _ = g.next())
          (_ = d(c, _.value, y)),
            _ !== null && ((m = o(_, m, b)), U === null ? (w = _) : (U.sibling = _), (U = _));
        return $ && Ca(c, b), w;
      }
      for (C = l(C); !_.done; b++, _ = g.next())
        (_ = h(C, c, b, _.value, y)),
          _ !== null &&
            (e && _.alternate !== null && C.delete(_.key === null ? b : _.key),
            (m = o(_, m, b)),
            U === null ? (w = _) : (U.sibling = _),
            (U = _));
      return (
        e &&
          C.forEach(function (ot) {
            return t(c, ot);
          }),
        $ && Ca(c, b),
        w
      );
    }
    function L(c, m, g, y) {
      if (
        (typeof g == 'object' &&
          g !== null &&
          g.type === pr &&
          g.key === null &&
          (g = g.props.children),
        typeof g == 'object' && g !== null)
      ) {
        switch (g.$$typeof) {
          case jn:
            e: {
              for (var w = g.key; m !== null; ) {
                if (m.key === w) {
                  if (((w = g.type), w === pr)) {
                    if (m.tag === 7) {
                      a(c, m.sibling), (y = r(m, g.props.children)), (y.return = c), (c = y);
                      break e;
                    }
                  } else if (
                    m.elementType === w ||
                    (typeof w == 'object' && w !== null && w.$$typeof === Xa && Il(w) === m.type)
                  ) {
                    a(c, m.sibling), (y = r(m, g.props)), Eo(y, g), (y.return = c), (c = y);
                    break e;
                  }
                  a(c, m);
                  break;
                } else t(c, m);
                m = m.sibling;
              }
              g.type === pr
                ? ((y = Tl(g.props.children, c.mode, y, g.key)), (y.return = c), (c = y))
                : ((y = mu(g.type, g.key, g.props, null, c.mode, y)),
                  Eo(y, g),
                  (y.return = c),
                  (c = y));
            }
            return n(c);
          case ko:
            e: {
              for (w = g.key; m !== null; ) {
                if (m.key === w)
                  if (
                    m.tag === 4 &&
                    m.stateNode.containerInfo === g.containerInfo &&
                    m.stateNode.implementation === g.implementation
                  ) {
                    a(c, m.sibling), (y = r(m, g.children || [])), (y.return = c), (c = y);
                    break e;
                  } else {
                    a(c, m);
                    break;
                  }
                else t(c, m);
                m = m.sibling;
              }
              (y = As(g, c.mode, y)), (y.return = c), (c = y);
            }
            return n(c);
          case Xa:
            return (g = Il(g)), L(c, m, g, y);
        }
        if (Bo(g)) return x(c, m, g, y);
        if (Ro(g)) {
          if (((w = Ro(g)), typeof w != 'function')) throw Error(S(150));
          return (g = w.call(g)), v(c, m, g, y);
        }
        if (typeof g.then == 'function') return L(c, m, eu(g), y);
        if (g.$$typeof === Ra) return L(c, m, $n(c, g), y);
        tu(c, g);
      }
      return (typeof g == 'string' && g !== '') || typeof g == 'number' || typeof g == 'bigint'
        ? ((g = '' + g),
          m !== null && m.tag === 6
            ? (a(c, m.sibling), (y = r(m, g)), (y.return = c), (c = y))
            : (a(c, m), (y = Es(g, c.mode, y)), (y.return = c), (c = y)),
          n(c))
        : a(c, m);
    }
    return function (c, m, g, y) {
      try {
        an = 0;
        var w = L(c, m, g, y);
        return (Mr = null), w;
      } catch (C) {
        if (C === Yr || C === oi) throw C;
        var U = St(29, C, null, c.mode);
        return (U.lanes = y), (U.return = c), U;
      } finally {
      }
    };
  }
  var Ul = tg(!0),
    ag = tg(!1),
    Ya = !1;
  function ff(e) {
    e.updateQueue = {
      baseState: e.memoizedState,
      firstBaseUpdate: null,
      lastBaseUpdate: null,
      shared: { pending: null, lanes: 0, hiddenCallbacks: null },
      callbacks: null,
    };
  }
  function yd(e, t) {
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
  function ol(e) {
    return { lane: e, tag: 0, payload: null, callback: null, next: null };
  }
  function nl(e, t, a) {
    var l = e.updateQueue;
    if (l === null) return null;
    if (((l = l.shared), (ae & 2) !== 0)) {
      var r = l.pending;
      return (
        r === null ? (t.next = t) : ((t.next = r.next), (r.next = t)),
        (l.pending = t),
        (t = Mu(e)),
        Yh(e, null, a),
        t
      );
    }
    return ri(e, l, t, a), Mu(e);
  }
  function Fo(e, t, a) {
    if (((t = t.updateQueue), t !== null && ((t = t.shared), (a & 4194048) !== 0))) {
      var l = t.lanes;
      (l &= e.pendingLanes), (a |= l), (t.lanes = a), Lh(e, a);
    }
  }
  function Ms(e, t) {
    var a = e.updateQueue,
      l = e.alternate;
    if (l !== null && ((l = l.updateQueue), a === l)) {
      var r = null,
        o = null;
      if (((a = a.firstBaseUpdate), a !== null)) {
        do {
          var n = { lane: a.lane, tag: a.tag, payload: a.payload, callback: null, next: null };
          o === null ? (r = o = n) : (o = o.next = n), (a = a.next);
        } while (a !== null);
        o === null ? (r = o = t) : (o = o.next = t);
      } else r = o = t;
      (a = {
        baseState: l.baseState,
        firstBaseUpdate: r,
        lastBaseUpdate: o,
        shared: l.shared,
        callbacks: l.callbacks,
      }),
        (e.updateQueue = a);
      return;
    }
    (e = a.lastBaseUpdate),
      e === null ? (a.firstBaseUpdate = t) : (e.next = t),
      (a.lastBaseUpdate = t);
  }
  var xd = !1;
  function qo() {
    if (xd) {
      var e = Tr;
      if (e !== null) throw e;
    }
  }
  function Go(e, t, a, l) {
    xd = !1;
    var r = e.updateQueue;
    Ya = !1;
    var o = r.firstBaseUpdate,
      n = r.lastBaseUpdate,
      u = r.shared.pending;
    if (u !== null) {
      r.shared.pending = null;
      var i = u,
        s = i.next;
      (i.next = null), n === null ? (o = s) : (n.next = s), (n = i);
      var f = e.alternate;
      f !== null &&
        ((f = f.updateQueue),
        (u = f.lastBaseUpdate),
        u !== n && (u === null ? (f.firstBaseUpdate = s) : (u.next = s), (f.lastBaseUpdate = i)));
    }
    if (o !== null) {
      var d = r.baseState;
      (n = 0), (f = s = i = null), (u = o);
      do {
        var p = u.lane & -536870913,
          h = p !== u.lane;
        if (h ? (W & p) === p : (l & p) === p) {
          p !== 0 && p === Hr && (xd = !0),
            f !== null &&
              (f = f.next =
                { lane: 0, tag: u.tag, payload: u.payload, callback: null, next: null });
          e: {
            var x = e,
              v = u;
            p = t;
            var L = a;
            switch (v.tag) {
              case 1:
                if (((x = v.payload), typeof x == 'function')) {
                  d = x.call(L, d, p);
                  break e;
                }
                d = x;
                break e;
              case 3:
                x.flags = (x.flags & -65537) | 128;
              case 0:
                if (
                  ((x = v.payload), (p = typeof x == 'function' ? x.call(L, d, p) : x), p == null)
                )
                  break e;
                d = ye({}, d, p);
                break e;
              case 2:
                Ya = !0;
            }
          }
          (p = u.callback),
            p !== null &&
              ((e.flags |= 64),
              h && (e.flags |= 8192),
              (h = r.callbacks),
              h === null ? (r.callbacks = [p]) : h.push(p));
        } else
          (h = { lane: p, tag: u.tag, payload: u.payload, callback: u.callback, next: null }),
            f === null ? ((s = f = h), (i = d)) : (f = f.next = h),
            (n |= p);
        if (((u = u.next), u === null)) {
          if (((u = r.shared.pending), u === null)) break;
          (h = u), (u = h.next), (h.next = null), (r.lastBaseUpdate = h), (r.shared.pending = null);
        }
      } while (!0);
      f === null && (i = d),
        (r.baseState = i),
        (r.firstBaseUpdate = s),
        (r.lastBaseUpdate = f),
        o === null && (r.shared.lanes = 0),
        (hl |= n),
        (e.lanes = n),
        (e.memoizedState = d);
    }
  }
  function lg(e, t) {
    if (typeof e != 'function') throw Error(S(191, e));
    e.call(t);
  }
  function rg(e, t) {
    var a = e.callbacks;
    if (a !== null) for (e.callbacks = null, e = 0; e < a.length; e++) lg(a[e], t);
  }
  var Nr = ia(null),
    Ou = ia(0);
  function mp(e, t) {
    (e = Ua), ce(Ou, e), ce(Nr, t), (Ua = e | t.baseLanes);
  }
  function vd() {
    ce(Ou, Ua), ce(Nr, Nr.current);
  }
  function cf() {
    (Ua = Ou.current), _e(Nr), _e(Ou);
  }
  var At = ia(null),
    zt = null;
  function Qa(e) {
    var t = e.alternate;
    ce(Ce, Ce.current & 1),
      ce(At, e),
      zt === null && (t === null || Nr.current !== null || t.memoizedState !== null) && (zt = e);
  }
  function Ld(e) {
    ce(Ce, Ce.current), ce(At, e), zt === null && (zt = e);
  }
  function og(e) {
    e.tag === 22 ? (ce(Ce, Ce.current), ce(At, e), zt === null && (zt = e)) : Za(e);
  }
  function Za() {
    ce(Ce, Ce.current), ce(At, At.current);
  }
  function Lt(e) {
    _e(At), zt === e && (zt = null), _e(Ce);
  }
  var Ce = ia(0);
  function Uu(e) {
    for (var t = e; t !== null; ) {
      if (t.tag === 13) {
        var a = t.memoizedState;
        if (a !== null && ((a = a.dehydrated), a === null || _d(a) || zd(a))) return t;
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
  var ka = 0,
    q = null,
    ue = null,
    Ie = null,
    Hu = !1,
    Dr = !1,
    Hl = !1,
    Nu = 0,
    ln = 0,
    kr = null,
    fb = 0;
  function Le() {
    throw Error(S(321));
  }
  function mf(e, t) {
    if (t === null) return !1;
    for (var a = 0; a < t.length && a < e.length; a++) if (!Et(e[a], t[a])) return !1;
    return !0;
  }
  function pf(e, t, a, l, r, o) {
    return (
      (ka = o),
      (q = t),
      (t.memoizedState = null),
      (t.updateQueue = null),
      (t.lanes = 0),
      (N.H = e === null || e.memoizedState === null ? Hg : Rf),
      (Hl = !1),
      (o = a(l, r)),
      (Hl = !1),
      Dr && (o = ug(t, a, l, r)),
      ng(e),
      o
    );
  }
  function ng(e) {
    N.H = rn;
    var t = ue !== null && ue.next !== null;
    if (((ka = 0), (Ie = ue = q = null), (Hu = !1), (ln = 0), (kr = null), t)) throw Error(S(300));
    e === null || Te || ((e = e.dependencies), e !== null && ku(e) && (Te = !0));
  }
  function ug(e, t, a, l) {
    q = e;
    var r = 0;
    do {
      if ((Dr && (kr = null), (ln = 0), (Dr = !1), 25 <= r)) throw Error(S(301));
      if (((r += 1), (Ie = ue = null), e.updateQueue != null)) {
        var o = e.updateQueue;
        (o.lastEffect = null),
          (o.events = null),
          (o.stores = null),
          o.memoCache != null && (o.memoCache.index = 0);
      }
      (N.H = Ng), (o = t(a, l));
    } while (Dr);
    return o;
  }
  function cb() {
    var e = N.H,
      t = e.useState()[0];
    return (
      (t = typeof t.then == 'function' ? xn(t) : t),
      (e = e.useState()[0]),
      (ue !== null ? ue.memoizedState : null) !== e && (q.flags |= 1024),
      t
    );
  }
  function hf() {
    var e = Nu !== 0;
    return (Nu = 0), e;
  }
  function gf(e, t, a) {
    (t.updateQueue = e.updateQueue), (t.flags &= -2053), (e.lanes &= ~a);
  }
  function yf(e) {
    if (Hu) {
      for (e = e.memoizedState; e !== null; ) {
        var t = e.queue;
        t !== null && (t.pending = null), (e = e.next);
      }
      Hu = !1;
    }
    (ka = 0), (Ie = ue = q = null), (Dr = !1), (ln = Nu = 0), (kr = null);
  }
  function rt() {
    var e = { memoizedState: null, baseState: null, baseQueue: null, queue: null, next: null };
    return Ie === null ? (q.memoizedState = Ie = e) : (Ie = Ie.next = e), Ie;
  }
  function we() {
    if (ue === null) {
      var e = q.alternate;
      e = e !== null ? e.memoizedState : null;
    } else e = ue.next;
    var t = Ie === null ? q.memoizedState : Ie.next;
    if (t !== null) (Ie = t), (ue = e);
    else {
      if (e === null) throw q.alternate === null ? Error(S(467)) : Error(S(310));
      (ue = e),
        (e = {
          memoizedState: ue.memoizedState,
          baseState: ue.baseState,
          baseQueue: ue.baseQueue,
          queue: ue.queue,
          next: null,
        }),
        Ie === null ? (q.memoizedState = Ie = e) : (Ie = Ie.next = e);
    }
    return Ie;
  }
  function ni() {
    return { lastEffect: null, events: null, stores: null, memoCache: null };
  }
  function xn(e) {
    var t = ln;
    return (
      (ln += 1),
      kr === null && (kr = []),
      (e = eg(kr, e, t)),
      (t = q),
      (Ie === null ? t.memoizedState : Ie.next) === null &&
        ((t = t.alternate), (N.H = t === null || t.memoizedState === null ? Hg : Rf)),
      e
    );
  }
  function ui(e) {
    if (e !== null && typeof e == 'object') {
      if (typeof e.then == 'function') return xn(e);
      if (e.$$typeof === Ra) return Xe(e);
    }
    throw Error(S(438, String(e)));
  }
  function xf(e) {
    var t = null,
      a = q.updateQueue;
    if ((a !== null && (t = a.memoCache), t == null)) {
      var l = q.alternate;
      l !== null &&
        ((l = l.updateQueue),
        l !== null &&
          ((l = l.memoCache),
          l != null &&
            (t = {
              data: l.data.map(function (r) {
                return r.slice();
              }),
              index: 0,
            })));
    }
    if (
      (t == null && (t = { data: [], index: 0 }),
      a === null && ((a = ni()), (q.updateQueue = a)),
      (a.memoCache = t),
      (a = t.data[t.index]),
      a === void 0)
    )
      for (a = t.data[t.index] = Array(e), l = 0; l < e; l++) a[l] = WL;
    return t.index++, a;
  }
  function Ba(e, t) {
    return typeof t == 'function' ? t(e) : t;
  }
  function hu(e) {
    var t = we();
    return vf(t, ue, e);
  }
  function vf(e, t, a) {
    var l = e.queue;
    if (l === null) throw Error(S(311));
    l.lastRenderedReducer = a;
    var r = e.baseQueue,
      o = l.pending;
    if (o !== null) {
      if (r !== null) {
        var n = r.next;
        (r.next = o.next), (o.next = n);
      }
      (t.baseQueue = r = o), (l.pending = null);
    }
    if (((o = e.baseState), r === null)) e.memoizedState = o;
    else {
      t = r.next;
      var u = (n = null),
        i = null,
        s = t,
        f = !1;
      do {
        var d = s.lane & -536870913;
        if (d !== s.lane ? (W & d) === d : (ka & d) === d) {
          var p = s.revertLane;
          if (p === 0)
            i !== null &&
              (i = i.next =
                {
                  lane: 0,
                  revertLane: 0,
                  gesture: null,
                  action: s.action,
                  hasEagerState: s.hasEagerState,
                  eagerState: s.eagerState,
                  next: null,
                }),
              d === Hr && (f = !0);
          else if ((ka & p) === p) {
            (s = s.next), p === Hr && (f = !0);
            continue;
          } else
            (d = {
              lane: 0,
              revertLane: s.revertLane,
              gesture: null,
              action: s.action,
              hasEagerState: s.hasEagerState,
              eagerState: s.eagerState,
              next: null,
            }),
              i === null ? ((u = i = d), (n = o)) : (i = i.next = d),
              (q.lanes |= p),
              (hl |= p);
          (d = s.action), Hl && a(o, d), (o = s.hasEagerState ? s.eagerState : a(o, d));
        } else
          (p = {
            lane: d,
            revertLane: s.revertLane,
            gesture: s.gesture,
            action: s.action,
            hasEagerState: s.hasEagerState,
            eagerState: s.eagerState,
            next: null,
          }),
            i === null ? ((u = i = p), (n = o)) : (i = i.next = p),
            (q.lanes |= d),
            (hl |= d);
        s = s.next;
      } while (s !== null && s !== t);
      if (
        (i === null ? (n = o) : (i.next = u),
        !Et(o, e.memoizedState) && ((Te = !0), f && ((a = Tr), a !== null)))
      )
        throw a;
      (e.memoizedState = o), (e.baseState = n), (e.baseQueue = i), (l.lastRenderedState = o);
    }
    return r === null && (l.lanes = 0), [e.memoizedState, l.dispatch];
  }
  function Ds(e) {
    var t = we(),
      a = t.queue;
    if (a === null) throw Error(S(311));
    a.lastRenderedReducer = e;
    var l = a.dispatch,
      r = a.pending,
      o = t.memoizedState;
    if (r !== null) {
      a.pending = null;
      var n = (r = r.next);
      do (o = e(o, n.action)), (n = n.next);
      while (n !== r);
      Et(o, t.memoizedState) || (Te = !0),
        (t.memoizedState = o),
        t.baseQueue === null && (t.baseState = o),
        (a.lastRenderedState = o);
    }
    return [o, l];
  }
  function ig(e, t, a) {
    var l = q,
      r = we(),
      o = $;
    if (o) {
      if (a === void 0) throw Error(S(407));
      a = a();
    } else a = t();
    var n = !Et((ue || r).memoizedState, a);
    if (
      (n && ((r.memoizedState = a), (Te = !0)),
      (r = r.queue),
      Lf(fg.bind(null, l, r, e), [e]),
      r.getSnapshot !== t || n || (Ie !== null && Ie.memoizedState.tag & 1))
    ) {
      if (
        ((l.flags |= 2048),
        Pr(9, { destroy: void 0 }, dg.bind(null, l, r, a, t), null),
        se === null)
      )
        throw Error(S(349));
      o || (ka & 127) !== 0 || sg(l, t, a);
    }
    return a;
  }
  function sg(e, t, a) {
    (e.flags |= 16384),
      (e = { getSnapshot: t, value: a }),
      (t = q.updateQueue),
      t === null
        ? ((t = ni()), (q.updateQueue = t), (t.stores = [e]))
        : ((a = t.stores), a === null ? (t.stores = [e]) : a.push(e));
  }
  function dg(e, t, a, l) {
    (t.value = a), (t.getSnapshot = l), cg(t) && mg(e);
  }
  function fg(e, t, a) {
    return a(function () {
      cg(t) && mg(e);
    });
  }
  function cg(e) {
    var t = e.getSnapshot;
    e = e.value;
    try {
      var a = t();
      return !Et(e, a);
    } catch {
      return !0;
    }
  }
  function mg(e) {
    var t = Fl(e, 2);
    t !== null && ft(t, e, 2);
  }
  function Sd(e) {
    var t = rt();
    if (typeof e == 'function') {
      var a = e;
      if (((e = a()), Hl)) {
        Ja(!0);
        try {
          a();
        } finally {
          Ja(!1);
        }
      }
    }
    return (
      (t.memoizedState = t.baseState = e),
      (t.queue = {
        pending: null,
        lanes: 0,
        dispatch: null,
        lastRenderedReducer: Ba,
        lastRenderedState: e,
      }),
      t
    );
  }
  function pg(e, t, a, l) {
    return (e.baseState = a), vf(e, ue, typeof l == 'function' ? l : Ba);
  }
  function mb(e, t, a, l, r) {
    if (si(e)) throw Error(S(485));
    if (((e = t.action), e !== null)) {
      var o = {
        payload: r,
        action: e,
        next: null,
        isTransition: !0,
        status: 'pending',
        value: null,
        reason: null,
        listeners: [],
        then: function (n) {
          o.listeners.push(n);
        },
      };
      N.T !== null ? a(!0) : (o.isTransition = !1),
        l(o),
        (a = t.pending),
        a === null
          ? ((o.next = t.pending = o), hg(t, o))
          : ((o.next = a.next), (t.pending = a.next = o));
    }
  }
  function hg(e, t) {
    var a = t.action,
      l = t.payload,
      r = e.state;
    if (t.isTransition) {
      var o = N.T,
        n = {};
      N.T = n;
      try {
        var u = a(r, l),
          i = N.S;
        i !== null && i(n, u), pp(e, t, u);
      } catch (s) {
        bd(e, t, s);
      } finally {
        o !== null && n.types !== null && (o.types = n.types), (N.T = o);
      }
    } else
      try {
        (o = a(r, l)), pp(e, t, o);
      } catch (s) {
        bd(e, t, s);
      }
  }
  function pp(e, t, a) {
    a !== null && typeof a == 'object' && typeof a.then == 'function'
      ? a.then(
          function (l) {
            hp(e, t, l);
          },
          function (l) {
            return bd(e, t, l);
          }
        )
      : hp(e, t, a);
  }
  function hp(e, t, a) {
    (t.status = 'fulfilled'),
      (t.value = a),
      gg(t),
      (e.state = a),
      (t = e.pending),
      t !== null &&
        ((a = t.next), a === t ? (e.pending = null) : ((a = a.next), (t.next = a), hg(e, a)));
  }
  function bd(e, t, a) {
    var l = e.pending;
    if (((e.pending = null), l !== null)) {
      l = l.next;
      do (t.status = 'rejected'), (t.reason = a), gg(t), (t = t.next);
      while (t !== l);
    }
    e.action = null;
  }
  function gg(e) {
    e = e.listeners;
    for (var t = 0; t < e.length; t++) (0, e[t])();
  }
  function yg(e, t) {
    return t;
  }
  function gp(e, t) {
    if ($) {
      var a = se.formState;
      if (a !== null) {
        e: {
          var l = q;
          if ($) {
            if (ge) {
              t: {
                for (var r = ge, o = _t; r.nodeType !== 8; ) {
                  if (!o) {
                    r = null;
                    break t;
                  }
                  if (((r = Ft(r.nextSibling)), r === null)) {
                    r = null;
                    break t;
                  }
                }
                (o = r.data), (r = o === 'F!' || o === 'F' ? r : null);
              }
              if (r) {
                (ge = Ft(r.nextSibling)), (l = r.data === 'F!');
                break e;
              }
            }
            ml(l);
          }
          l = !1;
        }
        l && (t = a[0]);
      }
    }
    return (
      (a = rt()),
      (a.memoizedState = a.baseState = t),
      (l = {
        pending: null,
        lanes: 0,
        dispatch: null,
        lastRenderedReducer: yg,
        lastRenderedState: t,
      }),
      (a.queue = l),
      (a = Bg.bind(null, q, l)),
      (l.dispatch = a),
      (l = Sd(!1)),
      (o = wf.bind(null, q, !1, l.queue)),
      (l = rt()),
      (r = { state: t, dispatch: null, action: e, pending: null }),
      (l.queue = r),
      (a = mb.bind(null, q, r, o, a)),
      (r.dispatch = a),
      (l.memoizedState = e),
      [t, a, !1]
    );
  }
  function yp(e) {
    var t = we();
    return xg(t, ue, e);
  }
  function xg(e, t, a) {
    if (
      ((t = vf(e, t, yg)[0]),
      (e = hu(Ba)[0]),
      typeof t == 'object' && t !== null && typeof t.then == 'function')
    )
      try {
        var l = xn(t);
      } catch (n) {
        throw n === Yr ? oi : n;
      }
    else l = t;
    t = we();
    var r = t.queue,
      o = r.dispatch;
    return (
      a !== t.memoizedState &&
        ((q.flags |= 2048), Pr(9, { destroy: void 0 }, pb.bind(null, r, a), null)),
      [l, o, e]
    );
  }
  function pb(e, t) {
    e.action = t;
  }
  function xp(e) {
    var t = we(),
      a = ue;
    if (a !== null) return xg(t, a, e);
    we(), (t = t.memoizedState), (a = we());
    var l = a.queue.dispatch;
    return (a.memoizedState = e), [t, l, !1];
  }
  function Pr(e, t, a, l) {
    return (
      (e = { tag: e, create: a, deps: l, inst: t, next: null }),
      (t = q.updateQueue),
      t === null && ((t = ni()), (q.updateQueue = t)),
      (a = t.lastEffect),
      a === null
        ? (t.lastEffect = e.next = e)
        : ((l = a.next), (a.next = e), (e.next = l), (t.lastEffect = e)),
      e
    );
  }
  function vg() {
    return we().memoizedState;
  }
  function gu(e, t, a, l) {
    var r = rt();
    (q.flags |= e), (r.memoizedState = Pr(1 | t, { destroy: void 0 }, a, l === void 0 ? null : l));
  }
  function ii(e, t, a, l) {
    var r = we();
    l = l === void 0 ? null : l;
    var o = r.memoizedState.inst;
    ue !== null && l !== null && mf(l, ue.memoizedState.deps)
      ? (r.memoizedState = Pr(t, o, a, l))
      : ((q.flags |= e), (r.memoizedState = Pr(1 | t, o, a, l)));
  }
  function vp(e, t) {
    gu(8390656, 8, e, t);
  }
  function Lf(e, t) {
    ii(2048, 8, e, t);
  }
  function hb(e) {
    q.flags |= 4;
    var t = q.updateQueue;
    if (t === null) (t = ni()), (q.updateQueue = t), (t.events = [e]);
    else {
      var a = t.events;
      a === null ? (t.events = [e]) : a.push(e);
    }
  }
  function Lg(e) {
    var t = we().memoizedState;
    return (
      hb({ ref: t, nextImpl: e }),
      function () {
        if ((ae & 2) !== 0) throw Error(S(440));
        return t.impl.apply(void 0, arguments);
      }
    );
  }
  function Sg(e, t) {
    return ii(4, 2, e, t);
  }
  function bg(e, t) {
    return ii(4, 4, e, t);
  }
  function Cg(e, t) {
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
  function wg(e, t, a) {
    (a = a != null ? a.concat([e]) : null), ii(4, 4, Cg.bind(null, t, e), a);
  }
  function Sf() {}
  function Rg(e, t) {
    var a = we();
    t = t === void 0 ? null : t;
    var l = a.memoizedState;
    return t !== null && mf(t, l[1]) ? l[0] : ((a.memoizedState = [e, t]), e);
  }
  function Ig(e, t) {
    var a = we();
    t = t === void 0 ? null : t;
    var l = a.memoizedState;
    if (t !== null && mf(t, l[1])) return l[0];
    if (((l = e()), Hl)) {
      Ja(!0);
      try {
        e();
      } finally {
        Ja(!1);
      }
    }
    return (a.memoizedState = [l, t]), l;
  }
  function bf(e, t, a) {
    return a === void 0 || ((ka & 1073741824) !== 0 && (W & 261930) === 0)
      ? (e.memoizedState = t)
      : ((e.memoizedState = a), (e = py()), (q.lanes |= e), (hl |= e), a);
  }
  function Eg(e, t, a, l) {
    return Et(a, t)
      ? a
      : Nr.current !== null
        ? ((e = bf(e, a, l)), Et(e, t) || (Te = !0), e)
        : (ka & 42) === 0 || ((ka & 1073741824) !== 0 && (W & 261930) === 0)
          ? ((Te = !0), (e.memoizedState = a))
          : ((e = py()), (q.lanes |= e), (hl |= e), t);
  }
  function Ag(e, t, a, l, r) {
    var o = le.p;
    le.p = o !== 0 && 8 > o ? o : 8;
    var n = N.T,
      u = {};
    (N.T = u), wf(e, !1, t, a);
    try {
      var i = r(),
        s = N.S;
      if (
        (s !== null && s(u, i), i !== null && typeof i == 'object' && typeof i.then == 'function')
      ) {
        var f = db(i, l);
        Vo(e, t, f, It(e));
      } else Vo(e, t, l, It(e));
    } catch (d) {
      Vo(e, t, { then: function () {}, status: 'rejected', reason: d }, It());
    } finally {
      (le.p = o), n !== null && u.types !== null && (n.types = u.types), (N.T = n);
    }
  }
  function gb() {}
  function Cd(e, t, a, l) {
    if (e.tag !== 5) throw Error(S(476));
    var r = Tg(e).queue;
    Ag(
      e,
      r,
      t,
      Al,
      a === null
        ? gb
        : function () {
            return Mg(e), a(l);
          }
    );
  }
  function Tg(e) {
    var t = e.memoizedState;
    if (t !== null) return t;
    t = {
      memoizedState: Al,
      baseState: Al,
      baseQueue: null,
      queue: {
        pending: null,
        lanes: 0,
        dispatch: null,
        lastRenderedReducer: Ba,
        lastRenderedState: Al,
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
          lastRenderedReducer: Ba,
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
  function Mg(e) {
    var t = Tg(e);
    t.next === null && (t = e.alternate.memoizedState), Vo(e, t.next.queue, {}, It());
  }
  function Cf() {
    return Xe(un);
  }
  function Dg() {
    return we().memoizedState;
  }
  function kg() {
    return we().memoizedState;
  }
  function yb(e) {
    for (var t = e.return; t !== null; ) {
      switch (t.tag) {
        case 24:
        case 3:
          var a = It();
          e = ol(a);
          var l = nl(t, e, a);
          l !== null && (ft(l, t, a), Fo(l, t, a)), (t = { cache: uf() }), (e.payload = t);
          return;
      }
      t = t.return;
    }
  }
  function xb(e, t, a) {
    var l = It();
    (a = {
      lane: l,
      revertLane: 0,
      gesture: null,
      action: a,
      hasEagerState: !1,
      eagerState: null,
      next: null,
    }),
      si(e) ? Og(t, a) : ((a = lf(e, t, a, l)), a !== null && (ft(a, e, l), Ug(a, t, l)));
  }
  function Bg(e, t, a) {
    var l = It();
    Vo(e, t, a, l);
  }
  function Vo(e, t, a, l) {
    var r = {
      lane: l,
      revertLane: 0,
      gesture: null,
      action: a,
      hasEagerState: !1,
      eagerState: null,
      next: null,
    };
    if (si(e)) Og(t, r);
    else {
      var o = e.alternate;
      if (
        e.lanes === 0 &&
        (o === null || o.lanes === 0) &&
        ((o = t.lastRenderedReducer), o !== null)
      )
        try {
          var n = t.lastRenderedState,
            u = o(n, a);
          if (((r.hasEagerState = !0), (r.eagerState = u), Et(u, n)))
            return ri(e, t, r, 0), se === null && li(), !1;
        } catch {
        } finally {
        }
      if (((a = lf(e, t, r, l)), a !== null)) return ft(a, e, l), Ug(a, t, l), !0;
    }
    return !1;
  }
  function wf(e, t, a, l) {
    if (
      ((l = {
        lane: 2,
        revertLane: Bf(),
        gesture: null,
        action: l,
        hasEagerState: !1,
        eagerState: null,
        next: null,
      }),
      si(e))
    ) {
      if (t) throw Error(S(479));
    } else (t = lf(e, a, l, 2)), t !== null && ft(t, e, 2);
  }
  function si(e) {
    var t = e.alternate;
    return e === q || (t !== null && t === q);
  }
  function Og(e, t) {
    Dr = Hu = !0;
    var a = e.pending;
    a === null ? (t.next = t) : ((t.next = a.next), (a.next = t)), (e.pending = t);
  }
  function Ug(e, t, a) {
    if ((a & 4194048) !== 0) {
      var l = t.lanes;
      (l &= e.pendingLanes), (a |= l), (t.lanes = a), Lh(e, a);
    }
  }
  var rn = {
    readContext: Xe,
    use: ui,
    useCallback: Le,
    useContext: Le,
    useEffect: Le,
    useImperativeHandle: Le,
    useLayoutEffect: Le,
    useInsertionEffect: Le,
    useMemo: Le,
    useReducer: Le,
    useRef: Le,
    useState: Le,
    useDebugValue: Le,
    useDeferredValue: Le,
    useTransition: Le,
    useSyncExternalStore: Le,
    useId: Le,
    useHostTransitionStatus: Le,
    useFormState: Le,
    useActionState: Le,
    useOptimistic: Le,
    useMemoCache: Le,
    useCacheRefresh: Le,
  };
  rn.useEffectEvent = Le;
  var Hg = {
      readContext: Xe,
      use: ui,
      useCallback: function (e, t) {
        return (rt().memoizedState = [e, t === void 0 ? null : t]), e;
      },
      useContext: Xe,
      useEffect: vp,
      useImperativeHandle: function (e, t, a) {
        (a = a != null ? a.concat([e]) : null), gu(4194308, 4, Cg.bind(null, t, e), a);
      },
      useLayoutEffect: function (e, t) {
        return gu(4194308, 4, e, t);
      },
      useInsertionEffect: function (e, t) {
        gu(4, 2, e, t);
      },
      useMemo: function (e, t) {
        var a = rt();
        t = t === void 0 ? null : t;
        var l = e();
        if (Hl) {
          Ja(!0);
          try {
            e();
          } finally {
            Ja(!1);
          }
        }
        return (a.memoizedState = [l, t]), l;
      },
      useReducer: function (e, t, a) {
        var l = rt();
        if (a !== void 0) {
          var r = a(t);
          if (Hl) {
            Ja(!0);
            try {
              a(t);
            } finally {
              Ja(!1);
            }
          }
        } else r = t;
        return (
          (l.memoizedState = l.baseState = r),
          (e = {
            pending: null,
            lanes: 0,
            dispatch: null,
            lastRenderedReducer: e,
            lastRenderedState: r,
          }),
          (l.queue = e),
          (e = e.dispatch = xb.bind(null, q, e)),
          [l.memoizedState, e]
        );
      },
      useRef: function (e) {
        var t = rt();
        return (e = { current: e }), (t.memoizedState = e);
      },
      useState: function (e) {
        e = Sd(e);
        var t = e.queue,
          a = Bg.bind(null, q, t);
        return (t.dispatch = a), [e.memoizedState, a];
      },
      useDebugValue: Sf,
      useDeferredValue: function (e, t) {
        var a = rt();
        return bf(a, e, t);
      },
      useTransition: function () {
        var e = Sd(!1);
        return (e = Ag.bind(null, q, e.queue, !0, !1)), (rt().memoizedState = e), [!1, e];
      },
      useSyncExternalStore: function (e, t, a) {
        var l = q,
          r = rt();
        if ($) {
          if (a === void 0) throw Error(S(407));
          a = a();
        } else {
          if (((a = t()), se === null)) throw Error(S(349));
          (W & 127) !== 0 || sg(l, t, a);
        }
        r.memoizedState = a;
        var o = { value: a, getSnapshot: t };
        return (
          (r.queue = o),
          vp(fg.bind(null, l, o, e), [e]),
          (l.flags |= 2048),
          Pr(9, { destroy: void 0 }, dg.bind(null, l, o, a, t), null),
          a
        );
      },
      useId: function () {
        var e = rt(),
          t = se.identifierPrefix;
        if ($) {
          var a = oa,
            l = ra;
          (a = (l & ~(1 << (32 - Rt(l) - 1))).toString(32) + a),
            (t = '_' + t + 'R_' + a),
            (a = Nu++),
            0 < a && (t += 'H' + a.toString(32)),
            (t += '_');
        } else (a = fb++), (t = '_' + t + 'r_' + a.toString(32) + '_');
        return (e.memoizedState = t);
      },
      useHostTransitionStatus: Cf,
      useFormState: gp,
      useActionState: gp,
      useOptimistic: function (e) {
        var t = rt();
        t.memoizedState = t.baseState = e;
        var a = {
          pending: null,
          lanes: 0,
          dispatch: null,
          lastRenderedReducer: null,
          lastRenderedState: null,
        };
        return (t.queue = a), (t = wf.bind(null, q, !0, a)), (a.dispatch = t), [e, t];
      },
      useMemoCache: xf,
      useCacheRefresh: function () {
        return (rt().memoizedState = yb.bind(null, q));
      },
      useEffectEvent: function (e) {
        var t = rt(),
          a = { impl: e };
        return (
          (t.memoizedState = a),
          function () {
            if ((ae & 2) !== 0) throw Error(S(440));
            return a.impl.apply(void 0, arguments);
          }
        );
      },
    },
    Rf = {
      readContext: Xe,
      use: ui,
      useCallback: Rg,
      useContext: Xe,
      useEffect: Lf,
      useImperativeHandle: wg,
      useInsertionEffect: Sg,
      useLayoutEffect: bg,
      useMemo: Ig,
      useReducer: hu,
      useRef: vg,
      useState: function () {
        return hu(Ba);
      },
      useDebugValue: Sf,
      useDeferredValue: function (e, t) {
        var a = we();
        return Eg(a, ue.memoizedState, e, t);
      },
      useTransition: function () {
        var e = hu(Ba)[0],
          t = we().memoizedState;
        return [typeof e == 'boolean' ? e : xn(e), t];
      },
      useSyncExternalStore: ig,
      useId: Dg,
      useHostTransitionStatus: Cf,
      useFormState: yp,
      useActionState: yp,
      useOptimistic: function (e, t) {
        var a = we();
        return pg(a, ue, e, t);
      },
      useMemoCache: xf,
      useCacheRefresh: kg,
    };
  Rf.useEffectEvent = Lg;
  var Ng = {
    readContext: Xe,
    use: ui,
    useCallback: Rg,
    useContext: Xe,
    useEffect: Lf,
    useImperativeHandle: wg,
    useInsertionEffect: Sg,
    useLayoutEffect: bg,
    useMemo: Ig,
    useReducer: Ds,
    useRef: vg,
    useState: function () {
      return Ds(Ba);
    },
    useDebugValue: Sf,
    useDeferredValue: function (e, t) {
      var a = we();
      return ue === null ? bf(a, e, t) : Eg(a, ue.memoizedState, e, t);
    },
    useTransition: function () {
      var e = Ds(Ba)[0],
        t = we().memoizedState;
      return [typeof e == 'boolean' ? e : xn(e), t];
    },
    useSyncExternalStore: ig,
    useId: Dg,
    useHostTransitionStatus: Cf,
    useFormState: xp,
    useActionState: xp,
    useOptimistic: function (e, t) {
      var a = we();
      return ue !== null ? pg(a, ue, e, t) : ((a.baseState = e), [e, a.queue.dispatch]);
    },
    useMemoCache: xf,
    useCacheRefresh: kg,
  };
  Ng.useEffectEvent = Lg;
  function ks(e, t, a, l) {
    (t = e.memoizedState),
      (a = a(l, t)),
      (a = a == null ? t : ye({}, t, a)),
      (e.memoizedState = a),
      e.lanes === 0 && (e.updateQueue.baseState = a);
  }
  var wd = {
    enqueueSetState: function (e, t, a) {
      e = e._reactInternals;
      var l = It(),
        r = ol(l);
      (r.payload = t),
        a != null && (r.callback = a),
        (t = nl(e, r, l)),
        t !== null && (ft(t, e, l), Fo(t, e, l));
    },
    enqueueReplaceState: function (e, t, a) {
      e = e._reactInternals;
      var l = It(),
        r = ol(l);
      (r.tag = 1),
        (r.payload = t),
        a != null && (r.callback = a),
        (t = nl(e, r, l)),
        t !== null && (ft(t, e, l), Fo(t, e, l));
    },
    enqueueForceUpdate: function (e, t) {
      e = e._reactInternals;
      var a = It(),
        l = ol(a);
      (l.tag = 2),
        t != null && (l.callback = t),
        (t = nl(e, l, a)),
        t !== null && (ft(t, e, a), Fo(t, e, a));
    },
  };
  function Lp(e, t, a, l, r, o, n) {
    return (
      (e = e.stateNode),
      typeof e.shouldComponentUpdate == 'function'
        ? e.shouldComponentUpdate(l, o, n)
        : t.prototype && t.prototype.isPureReactComponent
          ? !$o(a, l) || !$o(r, o)
          : !0
    );
  }
  function Sp(e, t, a, l) {
    (e = t.state),
      typeof t.componentWillReceiveProps == 'function' && t.componentWillReceiveProps(a, l),
      typeof t.UNSAFE_componentWillReceiveProps == 'function' &&
        t.UNSAFE_componentWillReceiveProps(a, l),
      t.state !== e && wd.enqueueReplaceState(t, t.state, null);
  }
  function Nl(e, t) {
    var a = t;
    if ('ref' in t) {
      a = {};
      for (var l in t) l !== 'ref' && (a[l] = t[l]);
    }
    if ((e = e.defaultProps)) {
      a === t && (a = ye({}, a));
      for (var r in e) a[r] === void 0 && (a[r] = e[r]);
    }
    return a;
  }
  function Pg(e) {
    Tu(e);
  }
  function _g(e) {
    console.error(e);
  }
  function zg(e) {
    Tu(e);
  }
  function Pu(e, t) {
    try {
      var a = e.onUncaughtError;
      a(t.value, { componentStack: t.stack });
    } catch (l) {
      setTimeout(function () {
        throw l;
      });
    }
  }
  function bp(e, t, a) {
    try {
      var l = e.onCaughtError;
      l(a.value, { componentStack: a.stack, errorBoundary: t.tag === 1 ? t.stateNode : null });
    } catch (r) {
      setTimeout(function () {
        throw r;
      });
    }
  }
  function Rd(e, t, a) {
    return (
      (a = ol(a)),
      (a.tag = 3),
      (a.payload = { element: null }),
      (a.callback = function () {
        Pu(e, t);
      }),
      a
    );
  }
  function Fg(e) {
    return (e = ol(e)), (e.tag = 3), e;
  }
  function qg(e, t, a, l) {
    var r = a.type.getDerivedStateFromError;
    if (typeof r == 'function') {
      var o = l.value;
      (e.payload = function () {
        return r(o);
      }),
        (e.callback = function () {
          bp(t, a, l);
        });
    }
    var n = a.stateNode;
    n !== null &&
      typeof n.componentDidCatch == 'function' &&
      (e.callback = function () {
        bp(t, a, l),
          typeof r != 'function' && (ul === null ? (ul = new Set([this])) : ul.add(this));
        var u = l.stack;
        this.componentDidCatch(l.value, { componentStack: u !== null ? u : '' });
      });
  }
  function vb(e, t, a, l, r) {
    if (((a.flags |= 32768), l !== null && typeof l == 'object' && typeof l.then == 'function')) {
      if (((t = a.alternate), t !== null && Xr(t, a, r, !0), (a = At.current), a !== null)) {
        switch (a.tag) {
          case 31:
          case 13:
            return (
              zt === null ? Gu() : a.alternate === null && Se === 0 && (Se = 3),
              (a.flags &= -257),
              (a.flags |= 65536),
              (a.lanes = r),
              l === Bu
                ? (a.flags |= 16384)
                : ((t = a.updateQueue),
                  t === null ? (a.updateQueue = new Set([l])) : t.add(l),
                  Gs(e, l, r)),
              !1
            );
          case 22:
            return (
              (a.flags |= 65536),
              l === Bu
                ? (a.flags |= 16384)
                : ((t = a.updateQueue),
                  t === null
                    ? ((t = { transitions: null, markerInstances: null, retryQueue: new Set([l]) }),
                      (a.updateQueue = t))
                    : ((a = t.retryQueue), a === null ? (t.retryQueue = new Set([l])) : a.add(l)),
                  Gs(e, l, r)),
              !1
            );
        }
        throw Error(S(435, a.tag));
      }
      return Gs(e, l, r), Gu(), !1;
    }
    if ($)
      return (
        (t = At.current),
        t !== null
          ? ((t.flags & 65536) === 0 && (t.flags |= 256),
            (t.flags |= 65536),
            (t.lanes = r),
            l !== cd && ((e = Error(S(422), { cause: l })), tn(Pt(e, a))))
          : (l !== cd && ((t = Error(S(423), { cause: l })), tn(Pt(t, a))),
            (e = e.current.alternate),
            (e.flags |= 65536),
            (r &= -r),
            (e.lanes |= r),
            (l = Pt(l, a)),
            (r = Rd(e.stateNode, l, r)),
            Ms(e, r),
            Se !== 4 && (Se = 2)),
        !1
      );
    var o = Error(S(520), { cause: l });
    if (((o = Pt(o, a)), Yo === null ? (Yo = [o]) : Yo.push(o), Se !== 4 && (Se = 2), t === null))
      return !0;
    (l = Pt(l, a)), (a = t);
    do {
      switch (a.tag) {
        case 3:
          return (
            (a.flags |= 65536),
            (e = r & -r),
            (a.lanes |= e),
            (e = Rd(a.stateNode, l, e)),
            Ms(a, e),
            !1
          );
        case 1:
          if (
            ((t = a.type),
            (o = a.stateNode),
            (a.flags & 128) === 0 &&
              (typeof t.getDerivedStateFromError == 'function' ||
                (o !== null &&
                  typeof o.componentDidCatch == 'function' &&
                  (ul === null || !ul.has(o)))))
          )
            return (
              (a.flags |= 65536),
              (r &= -r),
              (a.lanes |= r),
              (r = Fg(r)),
              qg(r, e, a, l),
              Ms(a, r),
              !1
            );
      }
      a = a.return;
    } while (a !== null);
    return !1;
  }
  var If = Error(S(461)),
    Te = !1;
  function Ge(e, t, a, l) {
    t.child = e === null ? ag(t, null, a, l) : Ul(t, e.child, a, l);
  }
  function Cp(e, t, a, l, r) {
    a = a.render;
    var o = t.ref;
    if ('ref' in l) {
      var n = {};
      for (var u in l) u !== 'ref' && (n[u] = l[u]);
    } else n = l;
    return (
      Ol(t),
      (l = pf(e, t, a, n, o, r)),
      (u = hf()),
      e !== null && !Te
        ? (gf(e, t, r), Oa(e, t, r))
        : ($ && u && of(t), (t.flags |= 1), Ge(e, t, l, r), t.child)
    );
  }
  function wp(e, t, a, l, r) {
    if (e === null) {
      var o = a.type;
      return typeof o == 'function' && !rf(o) && o.defaultProps === void 0 && a.compare === null
        ? ((t.tag = 15), (t.type = o), Gg(e, t, o, l, r))
        : ((e = mu(a.type, null, l, t, t.mode, r)), (e.ref = t.ref), (e.return = t), (t.child = e));
    }
    if (((o = e.child), !Ef(e, r))) {
      var n = o.memoizedProps;
      if (((a = a.compare), (a = a !== null ? a : $o), a(n, l) && e.ref === t.ref))
        return Oa(e, t, r);
    }
    return (t.flags |= 1), (e = Aa(o, l)), (e.ref = t.ref), (e.return = t), (t.child = e);
  }
  function Gg(e, t, a, l, r) {
    if (e !== null) {
      var o = e.memoizedProps;
      if ($o(o, l) && e.ref === t.ref)
        if (((Te = !1), (t.pendingProps = l = o), Ef(e, r))) (e.flags & 131072) !== 0 && (Te = !0);
        else return (t.lanes = e.lanes), Oa(e, t, r);
    }
    return Id(e, t, a, l, r);
  }
  function Vg(e, t, a, l) {
    var r = l.children,
      o = e !== null ? e.memoizedState : null;
    if (
      (e === null &&
        t.stateNode === null &&
        (t.stateNode = {
          _visibility: 1,
          _pendingMarkers: null,
          _retryCache: null,
          _transitions: null,
        }),
      l.mode === 'hidden')
    ) {
      if ((t.flags & 128) !== 0) {
        if (((o = o !== null ? o.baseLanes | a : a), e !== null)) {
          for (l = t.child = e.child, r = 0; l !== null; )
            (r = r | l.lanes | l.childLanes), (l = l.sibling);
          l = r & ~o;
        } else (l = 0), (t.child = null);
        return Rp(e, t, o, a, l);
      }
      if ((a & 536870912) !== 0)
        (t.memoizedState = { baseLanes: 0, cachePool: null }),
          e !== null && pu(t, o !== null ? o.cachePool : null),
          o !== null ? mp(t, o) : vd(),
          og(t);
      else return (l = t.lanes = 536870912), Rp(e, t, o !== null ? o.baseLanes | a : a, a, l);
    } else
      o !== null
        ? (pu(t, o.cachePool), mp(t, o), Za(t), (t.memoizedState = null))
        : (e !== null && pu(t, null), vd(), Za(t));
    return Ge(e, t, r, a), t.child;
  }
  function Uo(e, t) {
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
  function Rp(e, t, a, l, r) {
    var o = sf();
    return (
      (o = o === null ? null : { parent: Ae._currentValue, pool: o }),
      (t.memoizedState = { baseLanes: a, cachePool: o }),
      e !== null && pu(t, null),
      vd(),
      og(t),
      e !== null && Xr(e, t, l, !0),
      (t.childLanes = r),
      null
    );
  }
  function yu(e, t) {
    return (
      (t = _u({ mode: t.mode, children: t.children }, e.mode)),
      (t.ref = e.ref),
      (e.child = t),
      (t.return = e),
      t
    );
  }
  function Ip(e, t, a) {
    return (
      Ul(t, e.child, null, a),
      (e = yu(t, t.pendingProps)),
      (e.flags |= 2),
      Lt(t),
      (t.memoizedState = null),
      e
    );
  }
  function Lb(e, t, a) {
    var l = t.pendingProps,
      r = (t.flags & 128) !== 0;
    if (((t.flags &= -129), e === null)) {
      if ($) {
        if (l.mode === 'hidden') return (e = yu(t, l)), (t.lanes = 536870912), Uo(null, e);
        if (
          (Ld(t),
          (e = ge)
            ? ((e = Hy(e, _t)),
              (e = e !== null && e.data === '&' ? e : null),
              e !== null &&
                ((t.memoizedState = {
                  dehydrated: e,
                  treeContext: cl !== null ? { id: ra, overflow: oa } : null,
                  retryLane: 536870912,
                  hydrationErrors: null,
                }),
                (a = Qh(e)),
                (a.return = t),
                (t.child = a),
                (je = t),
                (ge = null)))
            : (e = null),
          e === null)
        )
          throw ml(t);
        return (t.lanes = 536870912), null;
      }
      return yu(t, l);
    }
    var o = e.memoizedState;
    if (o !== null) {
      var n = o.dehydrated;
      if ((Ld(t), r))
        if (t.flags & 256) (t.flags &= -257), (t = Ip(e, t, a));
        else if (t.memoizedState !== null) (t.child = e.child), (t.flags |= 128), (t = null);
        else throw Error(S(558));
      else if ((Te || Xr(e, t, a, !1), (r = (a & e.childLanes) !== 0), Te || r)) {
        if (((l = se), l !== null && ((n = Sh(l, a)), n !== 0 && n !== o.retryLane)))
          throw ((o.retryLane = n), Fl(e, n), ft(l, e, n), If);
        Gu(), (t = Ip(e, t, a));
      } else
        (e = o.treeContext),
          (ge = Ft(n.nextSibling)),
          (je = t),
          ($ = !0),
          (rl = null),
          (_t = !1),
          e !== null && Wh(t, e),
          (t = yu(t, l)),
          (t.flags |= 4096);
      return t;
    }
    return (
      (e = Aa(e.child, { mode: l.mode, children: l.children })),
      (e.ref = t.ref),
      (t.child = e),
      (e.return = t),
      e
    );
  }
  function xu(e, t) {
    var a = t.ref;
    if (a === null) e !== null && e.ref !== null && (t.flags |= 4194816);
    else {
      if (typeof a != 'function' && typeof a != 'object') throw Error(S(284));
      (e === null || e.ref !== a) && (t.flags |= 4194816);
    }
  }
  function Id(e, t, a, l, r) {
    return (
      Ol(t),
      (a = pf(e, t, a, l, void 0, r)),
      (l = hf()),
      e !== null && !Te
        ? (gf(e, t, r), Oa(e, t, r))
        : ($ && l && of(t), (t.flags |= 1), Ge(e, t, a, r), t.child)
    );
  }
  function Ep(e, t, a, l, r, o) {
    return (
      Ol(t),
      (t.updateQueue = null),
      (a = ug(t, l, a, r)),
      ng(e),
      (l = hf()),
      e !== null && !Te
        ? (gf(e, t, o), Oa(e, t, o))
        : ($ && l && of(t), (t.flags |= 1), Ge(e, t, a, o), t.child)
    );
  }
  function Ap(e, t, a, l, r) {
    if ((Ol(t), t.stateNode === null)) {
      var o = br,
        n = a.contextType;
      typeof n == 'object' && n !== null && (o = Xe(n)),
        (o = new a(l, o)),
        (t.memoizedState = o.state !== null && o.state !== void 0 ? o.state : null),
        (o.updater = wd),
        (t.stateNode = o),
        (o._reactInternals = t),
        (o = t.stateNode),
        (o.props = l),
        (o.state = t.memoizedState),
        (o.refs = {}),
        ff(t),
        (n = a.contextType),
        (o.context = typeof n == 'object' && n !== null ? Xe(n) : br),
        (o.state = t.memoizedState),
        (n = a.getDerivedStateFromProps),
        typeof n == 'function' && (ks(t, a, n, l), (o.state = t.memoizedState)),
        typeof a.getDerivedStateFromProps == 'function' ||
          typeof o.getSnapshotBeforeUpdate == 'function' ||
          (typeof o.UNSAFE_componentWillMount != 'function' &&
            typeof o.componentWillMount != 'function') ||
          ((n = o.state),
          typeof o.componentWillMount == 'function' && o.componentWillMount(),
          typeof o.UNSAFE_componentWillMount == 'function' && o.UNSAFE_componentWillMount(),
          n !== o.state && wd.enqueueReplaceState(o, o.state, null),
          Go(t, l, o, r),
          qo(),
          (o.state = t.memoizedState)),
        typeof o.componentDidMount == 'function' && (t.flags |= 4194308),
        (l = !0);
    } else if (e === null) {
      o = t.stateNode;
      var u = t.memoizedProps,
        i = Nl(a, u);
      o.props = i;
      var s = o.context,
        f = a.contextType;
      (n = br), typeof f == 'object' && f !== null && (n = Xe(f));
      var d = a.getDerivedStateFromProps;
      (f = typeof d == 'function' || typeof o.getSnapshotBeforeUpdate == 'function'),
        (u = t.pendingProps !== u),
        f ||
          (typeof o.UNSAFE_componentWillReceiveProps != 'function' &&
            typeof o.componentWillReceiveProps != 'function') ||
          ((u || s !== n) && Sp(t, o, l, n)),
        (Ya = !1);
      var p = t.memoizedState;
      (o.state = p),
        Go(t, l, o, r),
        qo(),
        (s = t.memoizedState),
        u || p !== s || Ya
          ? (typeof d == 'function' && (ks(t, a, d, l), (s = t.memoizedState)),
            (i = Ya || Lp(t, a, i, l, p, s, n))
              ? (f ||
                  (typeof o.UNSAFE_componentWillMount != 'function' &&
                    typeof o.componentWillMount != 'function') ||
                  (typeof o.componentWillMount == 'function' && o.componentWillMount(),
                  typeof o.UNSAFE_componentWillMount == 'function' &&
                    o.UNSAFE_componentWillMount()),
                typeof o.componentDidMount == 'function' && (t.flags |= 4194308))
              : (typeof o.componentDidMount == 'function' && (t.flags |= 4194308),
                (t.memoizedProps = l),
                (t.memoizedState = s)),
            (o.props = l),
            (o.state = s),
            (o.context = n),
            (l = i))
          : (typeof o.componentDidMount == 'function' && (t.flags |= 4194308), (l = !1));
    } else {
      (o = t.stateNode),
        yd(e, t),
        (n = t.memoizedProps),
        (f = Nl(a, n)),
        (o.props = f),
        (d = t.pendingProps),
        (p = o.context),
        (s = a.contextType),
        (i = br),
        typeof s == 'object' && s !== null && (i = Xe(s)),
        (u = a.getDerivedStateFromProps),
        (s = typeof u == 'function' || typeof o.getSnapshotBeforeUpdate == 'function') ||
          (typeof o.UNSAFE_componentWillReceiveProps != 'function' &&
            typeof o.componentWillReceiveProps != 'function') ||
          ((n !== d || p !== i) && Sp(t, o, l, i)),
        (Ya = !1),
        (p = t.memoizedState),
        (o.state = p),
        Go(t, l, o, r),
        qo();
      var h = t.memoizedState;
      n !== d || p !== h || Ya || (e !== null && e.dependencies !== null && ku(e.dependencies))
        ? (typeof u == 'function' && (ks(t, a, u, l), (h = t.memoizedState)),
          (f =
            Ya ||
            Lp(t, a, f, l, p, h, i) ||
            (e !== null && e.dependencies !== null && ku(e.dependencies)))
            ? (s ||
                (typeof o.UNSAFE_componentWillUpdate != 'function' &&
                  typeof o.componentWillUpdate != 'function') ||
                (typeof o.componentWillUpdate == 'function' && o.componentWillUpdate(l, h, i),
                typeof o.UNSAFE_componentWillUpdate == 'function' &&
                  o.UNSAFE_componentWillUpdate(l, h, i)),
              typeof o.componentDidUpdate == 'function' && (t.flags |= 4),
              typeof o.getSnapshotBeforeUpdate == 'function' && (t.flags |= 1024))
            : (typeof o.componentDidUpdate != 'function' ||
                (n === e.memoizedProps && p === e.memoizedState) ||
                (t.flags |= 4),
              typeof o.getSnapshotBeforeUpdate != 'function' ||
                (n === e.memoizedProps && p === e.memoizedState) ||
                (t.flags |= 1024),
              (t.memoizedProps = l),
              (t.memoizedState = h)),
          (o.props = l),
          (o.state = h),
          (o.context = i),
          (l = f))
        : (typeof o.componentDidUpdate != 'function' ||
            (n === e.memoizedProps && p === e.memoizedState) ||
            (t.flags |= 4),
          typeof o.getSnapshotBeforeUpdate != 'function' ||
            (n === e.memoizedProps && p === e.memoizedState) ||
            (t.flags |= 1024),
          (l = !1));
    }
    return (
      (o = l),
      xu(e, t),
      (l = (t.flags & 128) !== 0),
      o || l
        ? ((o = t.stateNode),
          (a = l && typeof a.getDerivedStateFromError != 'function' ? null : o.render()),
          (t.flags |= 1),
          e !== null && l
            ? ((t.child = Ul(t, e.child, null, r)), (t.child = Ul(t, null, a, r)))
            : Ge(e, t, a, r),
          (t.memoizedState = o.state),
          (e = t.child))
        : (e = Oa(e, t, r)),
      e
    );
  }
  function Tp(e, t, a, l) {
    return Bl(), (t.flags |= 256), Ge(e, t, a, l), t.child;
  }
  var Bs = { dehydrated: null, treeContext: null, retryLane: 0, hydrationErrors: null };
  function Os(e) {
    return { baseLanes: e, cachePool: $h() };
  }
  function Us(e, t, a) {
    return (e = e !== null ? e.childLanes & ~a : 0), t && (e |= bt), e;
  }
  function jg(e, t, a) {
    var l = t.pendingProps,
      r = !1,
      o = (t.flags & 128) !== 0,
      n;
    if (
      ((n = o) || (n = e !== null && e.memoizedState === null ? !1 : (Ce.current & 2) !== 0),
      n && ((r = !0), (t.flags &= -129)),
      (n = (t.flags & 32) !== 0),
      (t.flags &= -33),
      e === null)
    ) {
      if ($) {
        if (
          (r ? Qa(t) : Za(t),
          (e = ge)
            ? ((e = Hy(e, _t)),
              (e = e !== null && e.data !== '&' ? e : null),
              e !== null &&
                ((t.memoizedState = {
                  dehydrated: e,
                  treeContext: cl !== null ? { id: ra, overflow: oa } : null,
                  retryLane: 536870912,
                  hydrationErrors: null,
                }),
                (a = Qh(e)),
                (a.return = t),
                (t.child = a),
                (je = t),
                (ge = null)))
            : (e = null),
          e === null)
        )
          throw ml(t);
        return zd(e) ? (t.lanes = 32) : (t.lanes = 536870912), null;
      }
      var u = l.children;
      return (
        (l = l.fallback),
        r
          ? (Za(t),
            (r = t.mode),
            (u = _u({ mode: 'hidden', children: u }, r)),
            (l = Tl(l, r, a, null)),
            (u.return = t),
            (l.return = t),
            (u.sibling = l),
            (t.child = u),
            (l = t.child),
            (l.memoizedState = Os(a)),
            (l.childLanes = Us(e, n, a)),
            (t.memoizedState = Bs),
            Uo(null, l))
          : (Qa(t), Ed(t, u))
      );
    }
    var i = e.memoizedState;
    if (i !== null && ((u = i.dehydrated), u !== null)) {
      if (o)
        t.flags & 256
          ? (Qa(t), (t.flags &= -257), (t = Hs(e, t, a)))
          : t.memoizedState !== null
            ? (Za(t), (t.child = e.child), (t.flags |= 128), (t = null))
            : (Za(t),
              (u = l.fallback),
              (r = t.mode),
              (l = _u({ mode: 'visible', children: l.children }, r)),
              (u = Tl(u, r, a, null)),
              (u.flags |= 2),
              (l.return = t),
              (u.return = t),
              (l.sibling = u),
              (t.child = l),
              Ul(t, e.child, null, a),
              (l = t.child),
              (l.memoizedState = Os(a)),
              (l.childLanes = Us(e, n, a)),
              (t.memoizedState = Bs),
              (t = Uo(null, l)));
      else if ((Qa(t), zd(u))) {
        if (((n = u.nextSibling && u.nextSibling.dataset), n)) var s = n.dgst;
        (n = s),
          (l = Error(S(419))),
          (l.stack = ''),
          (l.digest = n),
          tn({ value: l, source: null, stack: null }),
          (t = Hs(e, t, a));
      } else if ((Te || Xr(e, t, a, !1), (n = (a & e.childLanes) !== 0), Te || n)) {
        if (((n = se), n !== null && ((l = Sh(n, a)), l !== 0 && l !== i.retryLane)))
          throw ((i.retryLane = l), Fl(e, l), ft(n, e, l), If);
        _d(u) || Gu(), (t = Hs(e, t, a));
      } else
        _d(u)
          ? ((t.flags |= 192), (t.child = e.child), (t = null))
          : ((e = i.treeContext),
            (ge = Ft(u.nextSibling)),
            (je = t),
            ($ = !0),
            (rl = null),
            (_t = !1),
            e !== null && Wh(t, e),
            (t = Ed(t, l.children)),
            (t.flags |= 4096));
      return t;
    }
    return r
      ? (Za(t),
        (u = l.fallback),
        (r = t.mode),
        (i = e.child),
        (s = i.sibling),
        (l = Aa(i, { mode: 'hidden', children: l.children })),
        (l.subtreeFlags = i.subtreeFlags & 65011712),
        s !== null ? (u = Aa(s, u)) : ((u = Tl(u, r, a, null)), (u.flags |= 2)),
        (u.return = t),
        (l.return = t),
        (l.sibling = u),
        (t.child = l),
        Uo(null, l),
        (l = t.child),
        (u = e.child.memoizedState),
        u === null
          ? (u = Os(a))
          : ((r = u.cachePool),
            r !== null
              ? ((i = Ae._currentValue), (r = r.parent !== i ? { parent: i, pool: i } : r))
              : (r = $h()),
            (u = { baseLanes: u.baseLanes | a, cachePool: r })),
        (l.memoizedState = u),
        (l.childLanes = Us(e, n, a)),
        (t.memoizedState = Bs),
        Uo(e.child, l))
      : (Qa(t),
        (a = e.child),
        (e = a.sibling),
        (a = Aa(a, { mode: 'visible', children: l.children })),
        (a.return = t),
        (a.sibling = null),
        e !== null &&
          ((n = t.deletions), n === null ? ((t.deletions = [e]), (t.flags |= 16)) : n.push(e)),
        (t.child = a),
        (t.memoizedState = null),
        a);
  }
  function Ed(e, t) {
    return (t = _u({ mode: 'visible', children: t }, e.mode)), (t.return = e), (e.child = t);
  }
  function _u(e, t) {
    return (e = St(22, e, null, t)), (e.lanes = 0), e;
  }
  function Hs(e, t, a) {
    return (
      Ul(t, e.child, null, a),
      (e = Ed(t, t.pendingProps.children)),
      (e.flags |= 2),
      (t.memoizedState = null),
      e
    );
  }
  function Mp(e, t, a) {
    e.lanes |= t;
    var l = e.alternate;
    l !== null && (l.lanes |= t), pd(e.return, t, a);
  }
  function Ns(e, t, a, l, r, o) {
    var n = e.memoizedState;
    n === null
      ? (e.memoizedState = {
          isBackwards: t,
          rendering: null,
          renderingStartTime: 0,
          last: l,
          tail: a,
          tailMode: r,
          treeForkCount: o,
        })
      : ((n.isBackwards = t),
        (n.rendering = null),
        (n.renderingStartTime = 0),
        (n.last = l),
        (n.tail = a),
        (n.tailMode = r),
        (n.treeForkCount = o));
  }
  function Xg(e, t, a) {
    var l = t.pendingProps,
      r = l.revealOrder,
      o = l.tail;
    l = l.children;
    var n = Ce.current,
      u = (n & 2) !== 0;
    if (
      (u ? ((n = (n & 1) | 2), (t.flags |= 128)) : (n &= 1),
      ce(Ce, n),
      Ge(e, t, l, a),
      (l = $ ? en : 0),
      !u && e !== null && (e.flags & 128) !== 0)
    )
      e: for (e = t.child; e !== null; ) {
        if (e.tag === 13) e.memoizedState !== null && Mp(e, a, t);
        else if (e.tag === 19) Mp(e, a, t);
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
    switch (r) {
      case 'forwards':
        for (a = t.child, r = null; a !== null; )
          (e = a.alternate), e !== null && Uu(e) === null && (r = a), (a = a.sibling);
        (a = r),
          a === null ? ((r = t.child), (t.child = null)) : ((r = a.sibling), (a.sibling = null)),
          Ns(t, !1, r, a, o, l);
        break;
      case 'backwards':
      case 'unstable_legacy-backwards':
        for (a = null, r = t.child, t.child = null; r !== null; ) {
          if (((e = r.alternate), e !== null && Uu(e) === null)) {
            t.child = r;
            break;
          }
          (e = r.sibling), (r.sibling = a), (a = r), (r = e);
        }
        Ns(t, !0, a, null, o, l);
        break;
      case 'together':
        Ns(t, !1, null, null, void 0, l);
        break;
      default:
        t.memoizedState = null;
    }
    return t.child;
  }
  function Oa(e, t, a) {
    if (
      (e !== null && (t.dependencies = e.dependencies), (hl |= t.lanes), (a & t.childLanes) === 0)
    )
      if (e !== null) {
        if ((Xr(e, t, a, !1), (a & t.childLanes) === 0)) return null;
      } else return null;
    if (e !== null && t.child !== e.child) throw Error(S(153));
    if (t.child !== null) {
      for (e = t.child, a = Aa(e, e.pendingProps), t.child = a, a.return = t; e.sibling !== null; )
        (e = e.sibling), (a = a.sibling = Aa(e, e.pendingProps)), (a.return = t);
      a.sibling = null;
    }
    return t.child;
  }
  function Ef(e, t) {
    return (e.lanes & t) !== 0 ? !0 : ((e = e.dependencies), !!(e !== null && ku(e)));
  }
  function Sb(e, t, a) {
    switch (t.tag) {
      case 3:
        Ru(t, t.stateNode.containerInfo), Ka(t, Ae, e.memoizedState.cache), Bl();
        break;
      case 27:
      case 5:
        td(t);
        break;
      case 4:
        Ru(t, t.stateNode.containerInfo);
        break;
      case 10:
        Ka(t, t.type, t.memoizedProps.value);
        break;
      case 31:
        if (t.memoizedState !== null) return (t.flags |= 128), Ld(t), null;
        break;
      case 13:
        var l = t.memoizedState;
        if (l !== null)
          return l.dehydrated !== null
            ? (Qa(t), (t.flags |= 128), null)
            : (a & t.child.childLanes) !== 0
              ? jg(e, t, a)
              : (Qa(t), (e = Oa(e, t, a)), e !== null ? e.sibling : null);
        Qa(t);
        break;
      case 19:
        var r = (e.flags & 128) !== 0;
        if (
          ((l = (a & t.childLanes) !== 0),
          l || (Xr(e, t, a, !1), (l = (a & t.childLanes) !== 0)),
          r)
        ) {
          if (l) return Xg(e, t, a);
          t.flags |= 128;
        }
        if (
          ((r = t.memoizedState),
          r !== null && ((r.rendering = null), (r.tail = null), (r.lastEffect = null)),
          ce(Ce, Ce.current),
          l)
        )
          break;
        return null;
      case 22:
        return (t.lanes = 0), Vg(e, t, a, t.pendingProps);
      case 24:
        Ka(t, Ae, e.memoizedState.cache);
    }
    return Oa(e, t, a);
  }
  function Yg(e, t, a) {
    if (e !== null)
      if (e.memoizedProps !== t.pendingProps) Te = !0;
      else {
        if (!Ef(e, a) && (t.flags & 128) === 0) return (Te = !1), Sb(e, t, a);
        Te = (e.flags & 131072) !== 0;
      }
    else (Te = !1), $ && (t.flags & 1048576) !== 0 && Zh(t, en, t.index);
    switch (((t.lanes = 0), t.tag)) {
      case 16:
        e: {
          var l = t.pendingProps;
          if (((e = Il(t.elementType)), (t.type = e), typeof e == 'function'))
            rf(e)
              ? ((l = Nl(e, l)), (t.tag = 1), (t = Ap(null, t, e, l, a)))
              : ((t.tag = 0), (t = Id(null, t, e, l, a)));
          else {
            if (e != null) {
              var r = e.$$typeof;
              if (r === Vd) {
                (t.tag = 11), (t = Cp(null, t, e, l, a));
                break e;
              } else if (r === jd) {
                (t.tag = 14), (t = wp(null, t, e, l, a));
                break e;
              }
            }
            throw ((t = $s(e) || e), Error(S(306, t, '')));
          }
        }
        return t;
      case 0:
        return Id(e, t, t.type, t.pendingProps, a);
      case 1:
        return (l = t.type), (r = Nl(l, t.pendingProps)), Ap(e, t, l, r, a);
      case 3:
        e: {
          if ((Ru(t, t.stateNode.containerInfo), e === null)) throw Error(S(387));
          l = t.pendingProps;
          var o = t.memoizedState;
          (r = o.element), yd(e, t), Go(t, l, null, a);
          var n = t.memoizedState;
          if (
            ((l = n.cache),
            Ka(t, Ae, l),
            l !== o.cache && hd(t, [Ae], a, !0),
            qo(),
            (l = n.element),
            o.isDehydrated)
          )
            if (
              ((o = { element: l, isDehydrated: !1, cache: n.cache }),
              (t.updateQueue.baseState = o),
              (t.memoizedState = o),
              t.flags & 256)
            ) {
              t = Tp(e, t, l, a);
              break e;
            } else if (l !== r) {
              (r = Pt(Error(S(424)), t)), tn(r), (t = Tp(e, t, l, a));
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
                ge = Ft(e.firstChild),
                  je = t,
                  $ = !0,
                  rl = null,
                  _t = !0,
                  a = ag(t, null, l, a),
                  t.child = a;
                a;

              )
                (a.flags = (a.flags & -3) | 4096), (a = a.sibling);
            }
          else {
            if ((Bl(), l === r)) {
              t = Oa(e, t, a);
              break e;
            }
            Ge(e, t, l, a);
          }
          t = t.child;
        }
        return t;
      case 26:
        return (
          xu(e, t),
          e === null
            ? (a = Jp(t.type, null, t.pendingProps, null))
              ? (t.memoizedState = a)
              : $ ||
                ((a = t.type),
                (e = t.pendingProps),
                (l = Yu(ll.current).createElement(a)),
                (l[Ve] = t),
                (l[ct] = e),
                Ye(l, a, e),
                Pe(l),
                (t.stateNode = l))
            : (t.memoizedState = Jp(t.type, e.memoizedProps, t.pendingProps, e.memoizedState)),
          null
        );
      case 27:
        return (
          td(t),
          e === null &&
            $ &&
            ((l = t.stateNode = Ny(t.type, t.pendingProps, ll.current)),
            (je = t),
            (_t = !0),
            (r = ge),
            yl(t.type) ? ((Fd = r), (ge = Ft(l.firstChild))) : (ge = r)),
          Ge(e, t, t.pendingProps.children, a),
          xu(e, t),
          e === null && (t.flags |= 4194304),
          t.child
        );
      case 5:
        return (
          e === null &&
            $ &&
            ((r = l = ge) &&
              ((l = Qb(l, t.type, t.pendingProps, _t)),
              l !== null
                ? ((t.stateNode = l), (je = t), (ge = Ft(l.firstChild)), (_t = !1), (r = !0))
                : (r = !1)),
            r || ml(t)),
          td(t),
          (r = t.type),
          (o = t.pendingProps),
          (n = e !== null ? e.memoizedProps : null),
          (l = o.children),
          Nd(r, o) ? (l = null) : n !== null && Nd(r, n) && (t.flags |= 32),
          t.memoizedState !== null && ((r = pf(e, t, cb, null, null, a)), (un._currentValue = r)),
          xu(e, t),
          Ge(e, t, l, a),
          t.child
        );
      case 6:
        return (
          e === null &&
            $ &&
            ((e = a = ge) &&
              ((a = Zb(a, t.pendingProps, _t)),
              a !== null ? ((t.stateNode = a), (je = t), (ge = null), (e = !0)) : (e = !1)),
            e || ml(t)),
          null
        );
      case 13:
        return jg(e, t, a);
      case 4:
        return (
          Ru(t, t.stateNode.containerInfo),
          (l = t.pendingProps),
          e === null ? (t.child = Ul(t, null, l, a)) : Ge(e, t, l, a),
          t.child
        );
      case 11:
        return Cp(e, t, t.type, t.pendingProps, a);
      case 7:
        return Ge(e, t, t.pendingProps, a), t.child;
      case 8:
        return Ge(e, t, t.pendingProps.children, a), t.child;
      case 12:
        return Ge(e, t, t.pendingProps.children, a), t.child;
      case 10:
        return (l = t.pendingProps), Ka(t, t.type, l.value), Ge(e, t, l.children, a), t.child;
      case 9:
        return (
          (r = t.type._context),
          (l = t.pendingProps.children),
          Ol(t),
          (r = Xe(r)),
          (l = l(r)),
          (t.flags |= 1),
          Ge(e, t, l, a),
          t.child
        );
      case 14:
        return wp(e, t, t.type, t.pendingProps, a);
      case 15:
        return Gg(e, t, t.type, t.pendingProps, a);
      case 19:
        return Xg(e, t, a);
      case 31:
        return Lb(e, t, a);
      case 22:
        return Vg(e, t, a, t.pendingProps);
      case 24:
        return (
          Ol(t),
          (l = Xe(Ae)),
          e === null
            ? ((r = sf()),
              r === null &&
                ((r = se),
                (o = uf()),
                (r.pooledCache = o),
                o.refCount++,
                o !== null && (r.pooledCacheLanes |= a),
                (r = o)),
              (t.memoizedState = { parent: l, cache: r }),
              ff(t),
              Ka(t, Ae, r))
            : ((e.lanes & a) !== 0 && (yd(e, t), Go(t, null, null, a), qo()),
              (r = e.memoizedState),
              (o = t.memoizedState),
              r.parent !== l
                ? ((r = { parent: l, cache: l }),
                  (t.memoizedState = r),
                  t.lanes === 0 && (t.memoizedState = t.updateQueue.baseState = r),
                  Ka(t, Ae, l))
                : ((l = o.cache), Ka(t, Ae, l), l !== r.cache && hd(t, [Ae], a, !0))),
          Ge(e, t, t.pendingProps.children, a),
          t.child
        );
      case 29:
        throw t.pendingProps;
    }
    throw Error(S(156, t.tag));
  }
  function va(e) {
    e.flags |= 4;
  }
  function Ps(e, t, a, l, r) {
    if (((t = (e.mode & 32) !== 0) && (t = !1), t)) {
      if (((e.flags |= 16777216), (r & 335544128) === r))
        if (e.stateNode.complete) e.flags |= 8192;
        else if (yy()) e.flags |= 8192;
        else throw ((Dl = Bu), df);
    } else e.flags &= -16777217;
  }
  function Dp(e, t) {
    if (t.type !== 'stylesheet' || (t.state.loading & 4) !== 0) e.flags &= -16777217;
    else if (((e.flags |= 16777216), !zy(t)))
      if (yy()) e.flags |= 8192;
      else throw ((Dl = Bu), df);
  }
  function au(e, t) {
    t !== null && (e.flags |= 4),
      e.flags & 16384 && ((t = e.tag !== 22 ? xh() : 536870912), (e.lanes |= t), (_r |= t));
  }
  function Ao(e, t) {
    if (!$)
      switch (e.tailMode) {
        case 'hidden':
          t = e.tail;
          for (var a = null; t !== null; ) t.alternate !== null && (a = t), (t = t.sibling);
          a === null ? (e.tail = null) : (a.sibling = null);
          break;
        case 'collapsed':
          a = e.tail;
          for (var l = null; a !== null; ) a.alternate !== null && (l = a), (a = a.sibling);
          l === null
            ? t || e.tail === null
              ? (e.tail = null)
              : (e.tail.sibling = null)
            : (l.sibling = null);
      }
  }
  function he(e) {
    var t = e.alternate !== null && e.alternate.child === e.child,
      a = 0,
      l = 0;
    if (t)
      for (var r = e.child; r !== null; )
        (a |= r.lanes | r.childLanes),
          (l |= r.subtreeFlags & 65011712),
          (l |= r.flags & 65011712),
          (r.return = e),
          (r = r.sibling);
    else
      for (r = e.child; r !== null; )
        (a |= r.lanes | r.childLanes),
          (l |= r.subtreeFlags),
          (l |= r.flags),
          (r.return = e),
          (r = r.sibling);
    return (e.subtreeFlags |= l), (e.childLanes = a), t;
  }
  function bb(e, t, a) {
    var l = t.pendingProps;
    switch ((nf(t), t.tag)) {
      case 16:
      case 15:
      case 0:
      case 11:
      case 7:
      case 8:
      case 12:
      case 9:
      case 14:
        return he(t), null;
      case 1:
        return he(t), null;
      case 3:
        return (
          (a = t.stateNode),
          (l = null),
          e !== null && (l = e.memoizedState.cache),
          t.memoizedState.cache !== l && (t.flags |= 2048),
          Ta(Ae),
          Br(),
          a.pendingContext && ((a.context = a.pendingContext), (a.pendingContext = null)),
          (e === null || e.child === null) &&
            (dr(t)
              ? va(t)
              : e === null ||
                (e.memoizedState.isDehydrated && (t.flags & 256) === 0) ||
                ((t.flags |= 1024), Ts())),
          he(t),
          null
        );
      case 26:
        var r = t.type,
          o = t.memoizedState;
        return (
          e === null
            ? (va(t), o !== null ? (he(t), Dp(t, o)) : (he(t), Ps(t, r, null, l, a)))
            : o
              ? o !== e.memoizedState
                ? (va(t), he(t), Dp(t, o))
                : (he(t), (t.flags &= -16777217))
              : ((e = e.memoizedProps), e !== l && va(t), he(t), Ps(t, r, e, l, a)),
          null
        );
      case 27:
        if ((Iu(t), (a = ll.current), (r = t.type), e !== null && t.stateNode != null))
          e.memoizedProps !== l && va(t);
        else {
          if (!l) {
            if (t.stateNode === null) throw Error(S(166));
            return he(t), null;
          }
          (e = ua.current), dr(t) ? np(t, e) : ((e = Ny(r, l, a)), (t.stateNode = e), va(t));
        }
        return he(t), null;
      case 5:
        if ((Iu(t), (r = t.type), e !== null && t.stateNode != null))
          e.memoizedProps !== l && va(t);
        else {
          if (!l) {
            if (t.stateNode === null) throw Error(S(166));
            return he(t), null;
          }
          if (((o = ua.current), dr(t))) np(t, o);
          else {
            var n = Yu(ll.current);
            switch (o) {
              case 1:
                o = n.createElementNS('http://www.w3.org/2000/svg', r);
                break;
              case 2:
                o = n.createElementNS('http://www.w3.org/1998/Math/MathML', r);
                break;
              default:
                switch (r) {
                  case 'svg':
                    o = n.createElementNS('http://www.w3.org/2000/svg', r);
                    break;
                  case 'math':
                    o = n.createElementNS('http://www.w3.org/1998/Math/MathML', r);
                    break;
                  case 'script':
                    (o = n.createElement('div')),
                      (o.innerHTML = '<script><\/script>'),
                      (o = o.removeChild(o.firstChild));
                    break;
                  case 'select':
                    (o =
                      typeof l.is == 'string'
                        ? n.createElement('select', { is: l.is })
                        : n.createElement('select')),
                      l.multiple ? (o.multiple = !0) : l.size && (o.size = l.size);
                    break;
                  default:
                    o =
                      typeof l.is == 'string'
                        ? n.createElement(r, { is: l.is })
                        : n.createElement(r);
                }
            }
            (o[Ve] = t), (o[ct] = l);
            e: for (n = t.child; n !== null; ) {
              if (n.tag === 5 || n.tag === 6) o.appendChild(n.stateNode);
              else if (n.tag !== 4 && n.tag !== 27 && n.child !== null) {
                (n.child.return = n), (n = n.child);
                continue;
              }
              if (n === t) break e;
              for (; n.sibling === null; ) {
                if (n.return === null || n.return === t) break e;
                n = n.return;
              }
              (n.sibling.return = n.return), (n = n.sibling);
            }
            t.stateNode = o;
            e: switch ((Ye(o, r, l), r)) {
              case 'button':
              case 'input':
              case 'select':
              case 'textarea':
                l = !!l.autoFocus;
                break e;
              case 'img':
                l = !0;
                break e;
              default:
                l = !1;
            }
            l && va(t);
          }
        }
        return he(t), Ps(t, t.type, e === null ? null : e.memoizedProps, t.pendingProps, a), null;
      case 6:
        if (e && t.stateNode != null) e.memoizedProps !== l && va(t);
        else {
          if (typeof l != 'string' && t.stateNode === null) throw Error(S(166));
          if (((e = ll.current), dr(t))) {
            if (((e = t.stateNode), (a = t.memoizedProps), (l = null), (r = je), r !== null))
              switch (r.tag) {
                case 27:
                case 5:
                  l = r.memoizedProps;
              }
            (e[Ve] = t),
              (e = !!(
                e.nodeValue === a ||
                (l !== null && l.suppressHydrationWarning === !0) ||
                By(e.nodeValue, a)
              )),
              e || ml(t, !0);
          } else (e = Yu(e).createTextNode(l)), (e[Ve] = t), (t.stateNode = e);
        }
        return he(t), null;
      case 31:
        if (((a = t.memoizedState), e === null || e.memoizedState !== null)) {
          if (((l = dr(t)), a !== null)) {
            if (e === null) {
              if (!l) throw Error(S(318));
              if (((e = t.memoizedState), (e = e !== null ? e.dehydrated : null), !e))
                throw Error(S(557));
              e[Ve] = t;
            } else Bl(), (t.flags & 128) === 0 && (t.memoizedState = null), (t.flags |= 4);
            he(t), (e = !1);
          } else
            (a = Ts()),
              e !== null && e.memoizedState !== null && (e.memoizedState.hydrationErrors = a),
              (e = !0);
          if (!e) return t.flags & 256 ? (Lt(t), t) : (Lt(t), null);
          if ((t.flags & 128) !== 0) throw Error(S(558));
        }
        return he(t), null;
      case 13:
        if (
          ((l = t.memoizedState),
          e === null || (e.memoizedState !== null && e.memoizedState.dehydrated !== null))
        ) {
          if (((r = dr(t)), l !== null && l.dehydrated !== null)) {
            if (e === null) {
              if (!r) throw Error(S(318));
              if (((r = t.memoizedState), (r = r !== null ? r.dehydrated : null), !r))
                throw Error(S(317));
              r[Ve] = t;
            } else Bl(), (t.flags & 128) === 0 && (t.memoizedState = null), (t.flags |= 4);
            he(t), (r = !1);
          } else
            (r = Ts()),
              e !== null && e.memoizedState !== null && (e.memoizedState.hydrationErrors = r),
              (r = !0);
          if (!r) return t.flags & 256 ? (Lt(t), t) : (Lt(t), null);
        }
        return (
          Lt(t),
          (t.flags & 128) !== 0
            ? ((t.lanes = a), t)
            : ((a = l !== null),
              (e = e !== null && e.memoizedState !== null),
              a &&
                ((l = t.child),
                (r = null),
                l.alternate !== null &&
                  l.alternate.memoizedState !== null &&
                  l.alternate.memoizedState.cachePool !== null &&
                  (r = l.alternate.memoizedState.cachePool.pool),
                (o = null),
                l.memoizedState !== null &&
                  l.memoizedState.cachePool !== null &&
                  (o = l.memoizedState.cachePool.pool),
                o !== r && (l.flags |= 2048)),
              a !== e && a && (t.child.flags |= 8192),
              au(t, t.updateQueue),
              he(t),
              null)
        );
      case 4:
        return Br(), e === null && Of(t.stateNode.containerInfo), he(t), null;
      case 10:
        return Ta(t.type), he(t), null;
      case 19:
        if ((_e(Ce), (l = t.memoizedState), l === null)) return he(t), null;
        if (((r = (t.flags & 128) !== 0), (o = l.rendering), o === null))
          if (r) Ao(l, !1);
          else {
            if (Se !== 0 || (e !== null && (e.flags & 128) !== 0))
              for (e = t.child; e !== null; ) {
                if (((o = Uu(e)), o !== null)) {
                  for (
                    t.flags |= 128,
                      Ao(l, !1),
                      e = o.updateQueue,
                      t.updateQueue = e,
                      au(t, e),
                      t.subtreeFlags = 0,
                      e = a,
                      a = t.child;
                    a !== null;

                  )
                    Kh(a, e), (a = a.sibling);
                  return ce(Ce, (Ce.current & 1) | 2), $ && Ca(t, l.treeForkCount), t.child;
                }
                e = e.sibling;
              }
            l.tail !== null &&
              Ct() > Fu &&
              ((t.flags |= 128), (r = !0), Ao(l, !1), (t.lanes = 4194304));
          }
        else {
          if (!r)
            if (((e = Uu(o)), e !== null)) {
              if (
                ((t.flags |= 128),
                (r = !0),
                (e = e.updateQueue),
                (t.updateQueue = e),
                au(t, e),
                Ao(l, !0),
                l.tail === null && l.tailMode === 'hidden' && !o.alternate && !$)
              )
                return he(t), null;
            } else
              2 * Ct() - l.renderingStartTime > Fu &&
                a !== 536870912 &&
                ((t.flags |= 128), (r = !0), Ao(l, !1), (t.lanes = 4194304));
          l.isBackwards
            ? ((o.sibling = t.child), (t.child = o))
            : ((e = l.last), e !== null ? (e.sibling = o) : (t.child = o), (l.last = o));
        }
        return l.tail !== null
          ? ((e = l.tail),
            (l.rendering = e),
            (l.tail = e.sibling),
            (l.renderingStartTime = Ct()),
            (e.sibling = null),
            (a = Ce.current),
            ce(Ce, r ? (a & 1) | 2 : a & 1),
            $ && Ca(t, l.treeForkCount),
            e)
          : (he(t), null);
      case 22:
      case 23:
        return (
          Lt(t),
          cf(),
          (l = t.memoizedState !== null),
          e !== null
            ? (e.memoizedState !== null) !== l && (t.flags |= 8192)
            : l && (t.flags |= 8192),
          l
            ? (a & 536870912) !== 0 &&
              (t.flags & 128) === 0 &&
              (he(t), t.subtreeFlags & 6 && (t.flags |= 8192))
            : he(t),
          (a = t.updateQueue),
          a !== null && au(t, a.retryQueue),
          (a = null),
          e !== null &&
            e.memoizedState !== null &&
            e.memoizedState.cachePool !== null &&
            (a = e.memoizedState.cachePool.pool),
          (l = null),
          t.memoizedState !== null &&
            t.memoizedState.cachePool !== null &&
            (l = t.memoizedState.cachePool.pool),
          l !== a && (t.flags |= 2048),
          e !== null && _e(Ml),
          null
        );
      case 24:
        return (
          (a = null),
          e !== null && (a = e.memoizedState.cache),
          t.memoizedState.cache !== a && (t.flags |= 2048),
          Ta(Ae),
          he(t),
          null
        );
      case 25:
        return null;
      case 30:
        return null;
    }
    throw Error(S(156, t.tag));
  }
  function Cb(e, t) {
    switch ((nf(t), t.tag)) {
      case 1:
        return (e = t.flags), e & 65536 ? ((t.flags = (e & -65537) | 128), t) : null;
      case 3:
        return (
          Ta(Ae),
          Br(),
          (e = t.flags),
          (e & 65536) !== 0 && (e & 128) === 0 ? ((t.flags = (e & -65537) | 128), t) : null
        );
      case 26:
      case 27:
      case 5:
        return Iu(t), null;
      case 31:
        if (t.memoizedState !== null) {
          if ((Lt(t), t.alternate === null)) throw Error(S(340));
          Bl();
        }
        return (e = t.flags), e & 65536 ? ((t.flags = (e & -65537) | 128), t) : null;
      case 13:
        if ((Lt(t), (e = t.memoizedState), e !== null && e.dehydrated !== null)) {
          if (t.alternate === null) throw Error(S(340));
          Bl();
        }
        return (e = t.flags), e & 65536 ? ((t.flags = (e & -65537) | 128), t) : null;
      case 19:
        return _e(Ce), null;
      case 4:
        return Br(), null;
      case 10:
        return Ta(t.type), null;
      case 22:
      case 23:
        return (
          Lt(t),
          cf(),
          e !== null && _e(Ml),
          (e = t.flags),
          e & 65536 ? ((t.flags = (e & -65537) | 128), t) : null
        );
      case 24:
        return Ta(Ae), null;
      case 25:
        return null;
      default:
        return null;
    }
  }
  function Kg(e, t) {
    switch ((nf(t), t.tag)) {
      case 3:
        Ta(Ae), Br();
        break;
      case 26:
      case 27:
      case 5:
        Iu(t);
        break;
      case 4:
        Br();
        break;
      case 31:
        t.memoizedState !== null && Lt(t);
        break;
      case 13:
        Lt(t);
        break;
      case 19:
        _e(Ce);
        break;
      case 10:
        Ta(t.type);
        break;
      case 22:
      case 23:
        Lt(t), cf(), e !== null && _e(Ml);
        break;
      case 24:
        Ta(Ae);
    }
  }
  function vn(e, t) {
    try {
      var a = t.updateQueue,
        l = a !== null ? a.lastEffect : null;
      if (l !== null) {
        var r = l.next;
        a = r;
        do {
          if ((a.tag & e) === e) {
            l = void 0;
            var o = a.create,
              n = a.inst;
            (l = o()), (n.destroy = l);
          }
          a = a.next;
        } while (a !== r);
      }
    } catch (u) {
      oe(t, t.return, u);
    }
  }
  function pl(e, t, a) {
    try {
      var l = t.updateQueue,
        r = l !== null ? l.lastEffect : null;
      if (r !== null) {
        var o = r.next;
        l = o;
        do {
          if ((l.tag & e) === e) {
            var n = l.inst,
              u = n.destroy;
            if (u !== void 0) {
              (n.destroy = void 0), (r = t);
              var i = a,
                s = u;
              try {
                s();
              } catch (f) {
                oe(r, i, f);
              }
            }
          }
          l = l.next;
        } while (l !== o);
      }
    } catch (f) {
      oe(t, t.return, f);
    }
  }
  function Qg(e) {
    var t = e.updateQueue;
    if (t !== null) {
      var a = e.stateNode;
      try {
        rg(t, a);
      } catch (l) {
        oe(e, e.return, l);
      }
    }
  }
  function Zg(e, t, a) {
    (a.props = Nl(e.type, e.memoizedProps)), (a.state = e.memoizedState);
    try {
      a.componentWillUnmount();
    } catch (l) {
      oe(e, t, l);
    }
  }
  function jo(e, t) {
    try {
      var a = e.ref;
      if (a !== null) {
        switch (e.tag) {
          case 26:
          case 27:
          case 5:
            var l = e.stateNode;
            break;
          case 30:
            l = e.stateNode;
            break;
          default:
            l = e.stateNode;
        }
        typeof a == 'function' ? (e.refCleanup = a(l)) : (a.current = l);
      }
    } catch (r) {
      oe(e, t, r);
    }
  }
  function na(e, t) {
    var a = e.ref,
      l = e.refCleanup;
    if (a !== null)
      if (typeof l == 'function')
        try {
          l();
        } catch (r) {
          oe(e, t, r);
        } finally {
          (e.refCleanup = null), (e = e.alternate), e != null && (e.refCleanup = null);
        }
      else if (typeof a == 'function')
        try {
          a(null);
        } catch (r) {
          oe(e, t, r);
        }
      else a.current = null;
  }
  function Wg(e) {
    var t = e.type,
      a = e.memoizedProps,
      l = e.stateNode;
    try {
      e: switch (t) {
        case 'button':
        case 'input':
        case 'select':
        case 'textarea':
          a.autoFocus && l.focus();
          break e;
        case 'img':
          a.src ? (l.src = a.src) : a.srcSet && (l.srcset = a.srcSet);
      }
    } catch (r) {
      oe(e, e.return, r);
    }
  }
  function _s(e, t, a) {
    try {
      var l = e.stateNode;
      Gb(l, e.type, a, t), (l[ct] = t);
    } catch (r) {
      oe(e, e.return, r);
    }
  }
  function Jg(e) {
    return (
      e.tag === 5 || e.tag === 3 || e.tag === 26 || (e.tag === 27 && yl(e.type)) || e.tag === 4
    );
  }
  function zs(e) {
    e: for (;;) {
      for (; e.sibling === null; ) {
        if (e.return === null || Jg(e.return)) return null;
        e = e.return;
      }
      for (
        e.sibling.return = e.return, e = e.sibling;
        e.tag !== 5 && e.tag !== 6 && e.tag !== 18;

      ) {
        if ((e.tag === 27 && yl(e.type)) || e.flags & 2 || e.child === null || e.tag === 4)
          continue e;
        (e.child.return = e), (e = e.child);
      }
      if (!(e.flags & 2)) return e.stateNode;
    }
  }
  function Ad(e, t, a) {
    var l = e.tag;
    if (l === 5 || l === 6)
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
            a != null || t.onclick !== null || (t.onclick = Ia));
    else if (
      l !== 4 &&
      (l === 27 && yl(e.type) && ((a = e.stateNode), (t = null)), (e = e.child), e !== null)
    )
      for (Ad(e, t, a), e = e.sibling; e !== null; ) Ad(e, t, a), (e = e.sibling);
  }
  function zu(e, t, a) {
    var l = e.tag;
    if (l === 5 || l === 6) (e = e.stateNode), t ? a.insertBefore(e, t) : a.appendChild(e);
    else if (l !== 4 && (l === 27 && yl(e.type) && (a = e.stateNode), (e = e.child), e !== null))
      for (zu(e, t, a), e = e.sibling; e !== null; ) zu(e, t, a), (e = e.sibling);
  }
  function $g(e) {
    var t = e.stateNode,
      a = e.memoizedProps;
    try {
      for (var l = e.type, r = t.attributes; r.length; ) t.removeAttributeNode(r[0]);
      Ye(t, l, a), (t[Ve] = e), (t[ct] = a);
    } catch (o) {
      oe(e, e.return, o);
    }
  }
  var wa = !1,
    Ee = !1,
    Fs = !1,
    kp = typeof WeakSet == 'function' ? WeakSet : Set,
    Ne = null;
  function wb(e, t) {
    if (((e = e.containerInfo), (Ud = Wu), (e = zh(e)), tf(e))) {
      if ('selectionStart' in e) var a = { start: e.selectionStart, end: e.selectionEnd };
      else
        e: {
          a = ((a = e.ownerDocument) && a.defaultView) || window;
          var l = a.getSelection && a.getSelection();
          if (l && l.rangeCount !== 0) {
            a = l.anchorNode;
            var r = l.anchorOffset,
              o = l.focusNode;
            l = l.focusOffset;
            try {
              a.nodeType, o.nodeType;
            } catch {
              a = null;
              break e;
            }
            var n = 0,
              u = -1,
              i = -1,
              s = 0,
              f = 0,
              d = e,
              p = null;
            t: for (;;) {
              for (
                var h;
                d !== a || (r !== 0 && d.nodeType !== 3) || (u = n + r),
                  d !== o || (l !== 0 && d.nodeType !== 3) || (i = n + l),
                  d.nodeType === 3 && (n += d.nodeValue.length),
                  (h = d.firstChild) !== null;

              )
                (p = d), (d = h);
              for (;;) {
                if (d === e) break t;
                if (
                  (p === a && ++s === r && (u = n),
                  p === o && ++f === l && (i = n),
                  (h = d.nextSibling) !== null)
                )
                  break;
                (d = p), (p = d.parentNode);
              }
              d = h;
            }
            a = u === -1 || i === -1 ? null : { start: u, end: i };
          } else a = null;
        }
      a = a || { start: 0, end: 0 };
    } else a = null;
    for (Hd = { focusedElem: e, selectionRange: a }, Wu = !1, Ne = t; Ne !== null; )
      if (((t = Ne), (e = t.child), (t.subtreeFlags & 1028) !== 0 && e !== null))
        (e.return = t), (Ne = e);
      else
        for (; Ne !== null; ) {
          switch (((t = Ne), (o = t.alternate), (e = t.flags), t.tag)) {
            case 0:
              if (
                (e & 4) !== 0 &&
                ((e = t.updateQueue), (e = e !== null ? e.events : null), e !== null)
              )
                for (a = 0; a < e.length; a++) (r = e[a]), (r.ref.impl = r.nextImpl);
              break;
            case 11:
            case 15:
              break;
            case 1:
              if ((e & 1024) !== 0 && o !== null) {
                (e = void 0),
                  (a = t),
                  (r = o.memoizedProps),
                  (o = o.memoizedState),
                  (l = a.stateNode);
                try {
                  var x = Nl(a.type, r);
                  (e = l.getSnapshotBeforeUpdate(x, o)),
                    (l.__reactInternalSnapshotBeforeUpdate = e);
                } catch (v) {
                  oe(a, a.return, v);
                }
              }
              break;
            case 3:
              if ((e & 1024) !== 0) {
                if (((e = t.stateNode.containerInfo), (a = e.nodeType), a === 9)) Pd(e);
                else if (a === 1)
                  switch (e.nodeName) {
                    case 'HEAD':
                    case 'HTML':
                    case 'BODY':
                      Pd(e);
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
              if ((e & 1024) !== 0) throw Error(S(163));
          }
          if (((e = t.sibling), e !== null)) {
            (e.return = t.return), (Ne = e);
            break;
          }
          Ne = t.return;
        }
  }
  function ey(e, t, a) {
    var l = a.flags;
    switch (a.tag) {
      case 0:
      case 11:
      case 15:
        Sa(e, a), l & 4 && vn(5, a);
        break;
      case 1:
        if ((Sa(e, a), l & 4))
          if (((e = a.stateNode), t === null))
            try {
              e.componentDidMount();
            } catch (n) {
              oe(a, a.return, n);
            }
          else {
            var r = Nl(a.type, t.memoizedProps);
            t = t.memoizedState;
            try {
              e.componentDidUpdate(r, t, e.__reactInternalSnapshotBeforeUpdate);
            } catch (n) {
              oe(a, a.return, n);
            }
          }
        l & 64 && Qg(a), l & 512 && jo(a, a.return);
        break;
      case 3:
        if ((Sa(e, a), l & 64 && ((e = a.updateQueue), e !== null))) {
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
            rg(e, t);
          } catch (n) {
            oe(a, a.return, n);
          }
        }
        break;
      case 27:
        t === null && l & 4 && $g(a);
      case 26:
      case 5:
        Sa(e, a), t === null && l & 4 && Wg(a), l & 512 && jo(a, a.return);
        break;
      case 12:
        Sa(e, a);
        break;
      case 31:
        Sa(e, a), l & 4 && ly(e, a);
        break;
      case 13:
        Sa(e, a),
          l & 4 && ry(e, a),
          l & 64 &&
            ((e = a.memoizedState),
            e !== null && ((e = e.dehydrated), e !== null && ((a = Bb.bind(null, a)), Wb(e, a))));
        break;
      case 22:
        if (((l = a.memoizedState !== null || wa), !l)) {
          (t = (t !== null && t.memoizedState !== null) || Ee), (r = wa);
          var o = Ee;
          (wa = l),
            (Ee = t) && !o ? ba(e, a, (a.subtreeFlags & 8772) !== 0) : Sa(e, a),
            (wa = r),
            (Ee = o);
        }
        break;
      case 30:
        break;
      default:
        Sa(e, a);
    }
  }
  function ty(e) {
    var t = e.alternate;
    t !== null && ((e.alternate = null), ty(t)),
      (e.child = null),
      (e.deletions = null),
      (e.sibling = null),
      e.tag === 5 && ((t = e.stateNode), t !== null && Qd(t)),
      (e.stateNode = null),
      (e.return = null),
      (e.dependencies = null),
      (e.memoizedProps = null),
      (e.memoizedState = null),
      (e.pendingProps = null),
      (e.stateNode = null),
      (e.updateQueue = null);
  }
  var ve = null,
    st = !1;
  function La(e, t, a) {
    for (a = a.child; a !== null; ) ay(e, t, a), (a = a.sibling);
  }
  function ay(e, t, a) {
    if (wt && typeof wt.onCommitFiberUnmount == 'function')
      try {
        wt.onCommitFiberUnmount(cn, a);
      } catch {}
    switch (a.tag) {
      case 26:
        Ee || na(a, t),
          La(e, t, a),
          a.memoizedState
            ? a.memoizedState.count--
            : a.stateNode && ((a = a.stateNode), a.parentNode.removeChild(a));
        break;
      case 27:
        Ee || na(a, t);
        var l = ve,
          r = st;
        yl(a.type) && ((ve = a.stateNode), (st = !1)),
          La(e, t, a),
          Qo(a.stateNode),
          (ve = l),
          (st = r);
        break;
      case 5:
        Ee || na(a, t);
      case 6:
        if (((l = ve), (r = st), (ve = null), La(e, t, a), (ve = l), (st = r), ve !== null))
          if (st)
            try {
              (ve.nodeType === 9
                ? ve.body
                : ve.nodeName === 'HTML'
                  ? ve.ownerDocument.body
                  : ve
              ).removeChild(a.stateNode);
            } catch (o) {
              oe(a, t, o);
            }
          else
            try {
              ve.removeChild(a.stateNode);
            } catch (o) {
              oe(a, t, o);
            }
        break;
      case 18:
        ve !== null &&
          (st
            ? ((e = ve),
              Yp(
                e.nodeType === 9 ? e.body : e.nodeName === 'HTML' ? e.ownerDocument.body : e,
                a.stateNode
              ),
              Gr(e))
            : Yp(ve, a.stateNode));
        break;
      case 4:
        (l = ve),
          (r = st),
          (ve = a.stateNode.containerInfo),
          (st = !0),
          La(e, t, a),
          (ve = l),
          (st = r);
        break;
      case 0:
      case 11:
      case 14:
      case 15:
        pl(2, a, t), Ee || pl(4, a, t), La(e, t, a);
        break;
      case 1:
        Ee ||
          (na(a, t), (l = a.stateNode), typeof l.componentWillUnmount == 'function' && Zg(a, t, l)),
          La(e, t, a);
        break;
      case 21:
        La(e, t, a);
        break;
      case 22:
        (Ee = (l = Ee) || a.memoizedState !== null), La(e, t, a), (Ee = l);
        break;
      default:
        La(e, t, a);
    }
  }
  function ly(e, t) {
    if (
      t.memoizedState === null &&
      ((e = t.alternate), e !== null && ((e = e.memoizedState), e !== null))
    ) {
      e = e.dehydrated;
      try {
        Gr(e);
      } catch (a) {
        oe(t, t.return, a);
      }
    }
  }
  function ry(e, t) {
    if (
      t.memoizedState === null &&
      ((e = t.alternate),
      e !== null && ((e = e.memoizedState), e !== null && ((e = e.dehydrated), e !== null)))
    )
      try {
        Gr(e);
      } catch (a) {
        oe(t, t.return, a);
      }
  }
  function Rb(e) {
    switch (e.tag) {
      case 31:
      case 13:
      case 19:
        var t = e.stateNode;
        return t === null && (t = e.stateNode = new kp()), t;
      case 22:
        return (
          (e = e.stateNode), (t = e._retryCache), t === null && (t = e._retryCache = new kp()), t
        );
      default:
        throw Error(S(435, e.tag));
    }
  }
  function lu(e, t) {
    var a = Rb(e);
    t.forEach(function (l) {
      if (!a.has(l)) {
        a.add(l);
        var r = Ob.bind(null, e, l);
        l.then(r, r);
      }
    });
  }
  function ut(e, t) {
    var a = t.deletions;
    if (a !== null)
      for (var l = 0; l < a.length; l++) {
        var r = a[l],
          o = e,
          n = t,
          u = n;
        e: for (; u !== null; ) {
          switch (u.tag) {
            case 27:
              if (yl(u.type)) {
                (ve = u.stateNode), (st = !1);
                break e;
              }
              break;
            case 5:
              (ve = u.stateNode), (st = !1);
              break e;
            case 3:
            case 4:
              (ve = u.stateNode.containerInfo), (st = !0);
              break e;
          }
          u = u.return;
        }
        if (ve === null) throw Error(S(160));
        ay(o, n, r),
          (ve = null),
          (st = !1),
          (o = r.alternate),
          o !== null && (o.return = null),
          (r.return = null);
      }
    if (t.subtreeFlags & 13886) for (t = t.child; t !== null; ) oy(t, e), (t = t.sibling);
  }
  var Yt = null;
  function oy(e, t) {
    var a = e.alternate,
      l = e.flags;
    switch (e.tag) {
      case 0:
      case 11:
      case 14:
      case 15:
        ut(t, e), it(e), l & 4 && (pl(3, e, e.return), vn(3, e), pl(5, e, e.return));
        break;
      case 1:
        ut(t, e),
          it(e),
          l & 512 && (Ee || a === null || na(a, a.return)),
          l & 64 &&
            wa &&
            ((e = e.updateQueue),
            e !== null &&
              ((l = e.callbacks),
              l !== null &&
                ((a = e.shared.hiddenCallbacks),
                (e.shared.hiddenCallbacks = a === null ? l : a.concat(l)))));
        break;
      case 26:
        var r = Yt;
        if ((ut(t, e), it(e), l & 512 && (Ee || a === null || na(a, a.return)), l & 4)) {
          var o = a !== null ? a.memoizedState : null;
          if (((l = e.memoizedState), a === null))
            if (l === null)
              if (e.stateNode === null) {
                e: {
                  (l = e.type), (a = e.memoizedProps), (r = r.ownerDocument || r);
                  t: switch (l) {
                    case 'title':
                      (o = r.getElementsByTagName('title')[0]),
                        (!o ||
                          o[hn] ||
                          o[Ve] ||
                          o.namespaceURI === 'http://www.w3.org/2000/svg' ||
                          o.hasAttribute('itemprop')) &&
                          ((o = r.createElement(l)),
                          r.head.insertBefore(o, r.querySelector('head > title'))),
                        Ye(o, l, a),
                        (o[Ve] = e),
                        Pe(o),
                        (l = o);
                      break e;
                    case 'link':
                      var n = eh('link', 'href', r).get(l + (a.href || ''));
                      if (n) {
                        for (var u = 0; u < n.length; u++)
                          if (
                            ((o = n[u]),
                            o.getAttribute('href') ===
                              (a.href == null || a.href === '' ? null : a.href) &&
                              o.getAttribute('rel') === (a.rel == null ? null : a.rel) &&
                              o.getAttribute('title') === (a.title == null ? null : a.title) &&
                              o.getAttribute('crossorigin') ===
                                (a.crossOrigin == null ? null : a.crossOrigin))
                          ) {
                            n.splice(u, 1);
                            break t;
                          }
                      }
                      (o = r.createElement(l)), Ye(o, l, a), r.head.appendChild(o);
                      break;
                    case 'meta':
                      if ((n = eh('meta', 'content', r).get(l + (a.content || '')))) {
                        for (u = 0; u < n.length; u++)
                          if (
                            ((o = n[u]),
                            o.getAttribute('content') ===
                              (a.content == null ? null : '' + a.content) &&
                              o.getAttribute('name') === (a.name == null ? null : a.name) &&
                              o.getAttribute('property') ===
                                (a.property == null ? null : a.property) &&
                              o.getAttribute('http-equiv') ===
                                (a.httpEquiv == null ? null : a.httpEquiv) &&
                              o.getAttribute('charset') === (a.charSet == null ? null : a.charSet))
                          ) {
                            n.splice(u, 1);
                            break t;
                          }
                      }
                      (o = r.createElement(l)), Ye(o, l, a), r.head.appendChild(o);
                      break;
                    default:
                      throw Error(S(468, l));
                  }
                  (o[Ve] = e), Pe(o), (l = o);
                }
                e.stateNode = l;
              } else th(r, e.type, e.stateNode);
            else e.stateNode = $p(r, l, e.memoizedProps);
          else
            o !== l
              ? (o === null
                  ? a.stateNode !== null && ((a = a.stateNode), a.parentNode.removeChild(a))
                  : o.count--,
                l === null ? th(r, e.type, e.stateNode) : $p(r, l, e.memoizedProps))
              : l === null && e.stateNode !== null && _s(e, e.memoizedProps, a.memoizedProps);
        }
        break;
      case 27:
        ut(t, e),
          it(e),
          l & 512 && (Ee || a === null || na(a, a.return)),
          a !== null && l & 4 && _s(e, e.memoizedProps, a.memoizedProps);
        break;
      case 5:
        if ((ut(t, e), it(e), l & 512 && (Ee || a === null || na(a, a.return)), e.flags & 32)) {
          r = e.stateNode;
          try {
            Ur(r, '');
          } catch (x) {
            oe(e, e.return, x);
          }
        }
        l & 4 &&
          e.stateNode != null &&
          ((r = e.memoizedProps), _s(e, r, a !== null ? a.memoizedProps : r)),
          l & 1024 && (Fs = !0);
        break;
      case 6:
        if ((ut(t, e), it(e), l & 4)) {
          if (e.stateNode === null) throw Error(S(162));
          (l = e.memoizedProps), (a = e.stateNode);
          try {
            a.nodeValue = l;
          } catch (x) {
            oe(e, e.return, x);
          }
        }
        break;
      case 3:
        if (
          ((Su = null),
          (r = Yt),
          (Yt = Ku(t.containerInfo)),
          ut(t, e),
          (Yt = r),
          it(e),
          l & 4 && a !== null && a.memoizedState.isDehydrated)
        )
          try {
            Gr(t.containerInfo);
          } catch (x) {
            oe(e, e.return, x);
          }
        Fs && ((Fs = !1), ny(e));
        break;
      case 4:
        (l = Yt), (Yt = Ku(e.stateNode.containerInfo)), ut(t, e), it(e), (Yt = l);
        break;
      case 12:
        ut(t, e), it(e);
        break;
      case 31:
        ut(t, e),
          it(e),
          l & 4 && ((l = e.updateQueue), l !== null && ((e.updateQueue = null), lu(e, l)));
        break;
      case 13:
        ut(t, e),
          it(e),
          e.child.flags & 8192 &&
            (e.memoizedState !== null) != (a !== null && a.memoizedState !== null) &&
            (di = Ct()),
          l & 4 && ((l = e.updateQueue), l !== null && ((e.updateQueue = null), lu(e, l)));
        break;
      case 22:
        r = e.memoizedState !== null;
        var i = a !== null && a.memoizedState !== null,
          s = wa,
          f = Ee;
        if (((wa = s || r), (Ee = f || i), ut(t, e), (Ee = f), (wa = s), it(e), l & 8192))
          e: for (
            t = e.stateNode,
              t._visibility = r ? t._visibility & -2 : t._visibility | 1,
              r && (a === null || i || wa || Ee || El(e)),
              a = null,
              t = e;
            ;

          ) {
            if (t.tag === 5 || t.tag === 26) {
              if (a === null) {
                i = a = t;
                try {
                  if (((o = i.stateNode), r))
                    (n = o.style),
                      typeof n.setProperty == 'function'
                        ? n.setProperty('display', 'none', 'important')
                        : (n.display = 'none');
                  else {
                    u = i.stateNode;
                    var d = i.memoizedProps.style,
                      p = d != null && d.hasOwnProperty('display') ? d.display : null;
                    u.style.display = p == null || typeof p == 'boolean' ? '' : ('' + p).trim();
                  }
                } catch (x) {
                  oe(i, i.return, x);
                }
              }
            } else if (t.tag === 6) {
              if (a === null) {
                i = t;
                try {
                  i.stateNode.nodeValue = r ? '' : i.memoizedProps;
                } catch (x) {
                  oe(i, i.return, x);
                }
              }
            } else if (t.tag === 18) {
              if (a === null) {
                i = t;
                try {
                  var h = i.stateNode;
                  r ? Kp(h, !0) : Kp(i.stateNode, !1);
                } catch (x) {
                  oe(i, i.return, x);
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
        l & 4 &&
          ((l = e.updateQueue),
          l !== null && ((a = l.retryQueue), a !== null && ((l.retryQueue = null), lu(e, a))));
        break;
      case 19:
        ut(t, e),
          it(e),
          l & 4 && ((l = e.updateQueue), l !== null && ((e.updateQueue = null), lu(e, l)));
        break;
      case 30:
        break;
      case 21:
        break;
      default:
        ut(t, e), it(e);
    }
  }
  function it(e) {
    var t = e.flags;
    if (t & 2) {
      try {
        for (var a, l = e.return; l !== null; ) {
          if (Jg(l)) {
            a = l;
            break;
          }
          l = l.return;
        }
        if (a == null) throw Error(S(160));
        switch (a.tag) {
          case 27:
            var r = a.stateNode,
              o = zs(e);
            zu(e, o, r);
            break;
          case 5:
            var n = a.stateNode;
            a.flags & 32 && (Ur(n, ''), (a.flags &= -33));
            var u = zs(e);
            zu(e, u, n);
            break;
          case 3:
          case 4:
            var i = a.stateNode.containerInfo,
              s = zs(e);
            Ad(e, s, i);
            break;
          default:
            throw Error(S(161));
        }
      } catch (f) {
        oe(e, e.return, f);
      }
      e.flags &= -3;
    }
    t & 4096 && (e.flags &= -4097);
  }
  function ny(e) {
    if (e.subtreeFlags & 1024)
      for (e = e.child; e !== null; ) {
        var t = e;
        ny(t), t.tag === 5 && t.flags & 1024 && t.stateNode.reset(), (e = e.sibling);
      }
  }
  function Sa(e, t) {
    if (t.subtreeFlags & 8772)
      for (t = t.child; t !== null; ) ey(e, t.alternate, t), (t = t.sibling);
  }
  function El(e) {
    for (e = e.child; e !== null; ) {
      var t = e;
      switch (t.tag) {
        case 0:
        case 11:
        case 14:
        case 15:
          pl(4, t, t.return), El(t);
          break;
        case 1:
          na(t, t.return);
          var a = t.stateNode;
          typeof a.componentWillUnmount == 'function' && Zg(t, t.return, a), El(t);
          break;
        case 27:
          Qo(t.stateNode);
        case 26:
        case 5:
          na(t, t.return), El(t);
          break;
        case 22:
          t.memoizedState === null && El(t);
          break;
        case 30:
          El(t);
          break;
        default:
          El(t);
      }
      e = e.sibling;
    }
  }
  function ba(e, t, a) {
    for (a = a && (t.subtreeFlags & 8772) !== 0, t = t.child; t !== null; ) {
      var l = t.alternate,
        r = e,
        o = t,
        n = o.flags;
      switch (o.tag) {
        case 0:
        case 11:
        case 15:
          ba(r, o, a), vn(4, o);
          break;
        case 1:
          if ((ba(r, o, a), (l = o), (r = l.stateNode), typeof r.componentDidMount == 'function'))
            try {
              r.componentDidMount();
            } catch (s) {
              oe(l, l.return, s);
            }
          if (((l = o), (r = l.updateQueue), r !== null)) {
            var u = l.stateNode;
            try {
              var i = r.shared.hiddenCallbacks;
              if (i !== null)
                for (r.shared.hiddenCallbacks = null, r = 0; r < i.length; r++) lg(i[r], u);
            } catch (s) {
              oe(l, l.return, s);
            }
          }
          a && n & 64 && Qg(o), jo(o, o.return);
          break;
        case 27:
          $g(o);
        case 26:
        case 5:
          ba(r, o, a), a && l === null && n & 4 && Wg(o), jo(o, o.return);
          break;
        case 12:
          ba(r, o, a);
          break;
        case 31:
          ba(r, o, a), a && n & 4 && ly(r, o);
          break;
        case 13:
          ba(r, o, a), a && n & 4 && ry(r, o);
          break;
        case 22:
          o.memoizedState === null && ba(r, o, a), jo(o, o.return);
          break;
        case 30:
          break;
        default:
          ba(r, o, a);
      }
      t = t.sibling;
    }
  }
  function Af(e, t) {
    var a = null;
    e !== null &&
      e.memoizedState !== null &&
      e.memoizedState.cachePool !== null &&
      (a = e.memoizedState.cachePool.pool),
      (e = null),
      t.memoizedState !== null &&
        t.memoizedState.cachePool !== null &&
        (e = t.memoizedState.cachePool.pool),
      e !== a && (e != null && e.refCount++, a != null && yn(a));
  }
  function Tf(e, t) {
    (e = null),
      t.alternate !== null && (e = t.alternate.memoizedState.cache),
      (t = t.memoizedState.cache),
      t !== e && (t.refCount++, e != null && yn(e));
  }
  function Xt(e, t, a, l) {
    if (t.subtreeFlags & 10256) for (t = t.child; t !== null; ) uy(e, t, a, l), (t = t.sibling);
  }
  function uy(e, t, a, l) {
    var r = t.flags;
    switch (t.tag) {
      case 0:
      case 11:
      case 15:
        Xt(e, t, a, l), r & 2048 && vn(9, t);
        break;
      case 1:
        Xt(e, t, a, l);
        break;
      case 3:
        Xt(e, t, a, l),
          r & 2048 &&
            ((e = null),
            t.alternate !== null && (e = t.alternate.memoizedState.cache),
            (t = t.memoizedState.cache),
            t !== e && (t.refCount++, e != null && yn(e)));
        break;
      case 12:
        if (r & 2048) {
          Xt(e, t, a, l), (e = t.stateNode);
          try {
            var o = t.memoizedProps,
              n = o.id,
              u = o.onPostCommit;
            typeof u == 'function' &&
              u(n, t.alternate === null ? 'mount' : 'update', e.passiveEffectDuration, -0);
          } catch (i) {
            oe(t, t.return, i);
          }
        } else Xt(e, t, a, l);
        break;
      case 31:
        Xt(e, t, a, l);
        break;
      case 13:
        Xt(e, t, a, l);
        break;
      case 23:
        break;
      case 22:
        (o = t.stateNode),
          (n = t.alternate),
          t.memoizedState !== null
            ? o._visibility & 2
              ? Xt(e, t, a, l)
              : Xo(e, t)
            : o._visibility & 2
              ? Xt(e, t, a, l)
              : ((o._visibility |= 2), cr(e, t, a, l, (t.subtreeFlags & 10256) !== 0 || !1)),
          r & 2048 && Af(n, t);
        break;
      case 24:
        Xt(e, t, a, l), r & 2048 && Tf(t.alternate, t);
        break;
      default:
        Xt(e, t, a, l);
    }
  }
  function cr(e, t, a, l, r) {
    for (r = r && ((t.subtreeFlags & 10256) !== 0 || !1), t = t.child; t !== null; ) {
      var o = e,
        n = t,
        u = a,
        i = l,
        s = n.flags;
      switch (n.tag) {
        case 0:
        case 11:
        case 15:
          cr(o, n, u, i, r), vn(8, n);
          break;
        case 23:
          break;
        case 22:
          var f = n.stateNode;
          n.memoizedState !== null
            ? f._visibility & 2
              ? cr(o, n, u, i, r)
              : Xo(o, n)
            : ((f._visibility |= 2), cr(o, n, u, i, r)),
            r && s & 2048 && Af(n.alternate, n);
          break;
        case 24:
          cr(o, n, u, i, r), r && s & 2048 && Tf(n.alternate, n);
          break;
        default:
          cr(o, n, u, i, r);
      }
      t = t.sibling;
    }
  }
  function Xo(e, t) {
    if (t.subtreeFlags & 10256)
      for (t = t.child; t !== null; ) {
        var a = e,
          l = t,
          r = l.flags;
        switch (l.tag) {
          case 22:
            Xo(a, l), r & 2048 && Af(l.alternate, l);
            break;
          case 24:
            Xo(a, l), r & 2048 && Tf(l.alternate, l);
            break;
          default:
            Xo(a, l);
        }
        t = t.sibling;
      }
  }
  var Ho = 8192;
  function fr(e, t, a) {
    if (e.subtreeFlags & Ho) for (e = e.child; e !== null; ) iy(e, t, a), (e = e.sibling);
  }
  function iy(e, t, a) {
    switch (e.tag) {
      case 26:
        fr(e, t, a),
          e.flags & Ho && e.memoizedState !== null && s0(a, Yt, e.memoizedState, e.memoizedProps);
        break;
      case 5:
        fr(e, t, a);
        break;
      case 3:
      case 4:
        var l = Yt;
        (Yt = Ku(e.stateNode.containerInfo)), fr(e, t, a), (Yt = l);
        break;
      case 22:
        e.memoizedState === null &&
          ((l = e.alternate),
          l !== null && l.memoizedState !== null
            ? ((l = Ho), (Ho = 16777216), fr(e, t, a), (Ho = l))
            : fr(e, t, a));
        break;
      default:
        fr(e, t, a);
    }
  }
  function sy(e) {
    var t = e.alternate;
    if (t !== null && ((e = t.child), e !== null)) {
      t.child = null;
      do (t = e.sibling), (e.sibling = null), (e = t);
      while (e !== null);
    }
  }
  function To(e) {
    var t = e.deletions;
    if ((e.flags & 16) !== 0) {
      if (t !== null)
        for (var a = 0; a < t.length; a++) {
          var l = t[a];
          (Ne = l), fy(l, e);
        }
      sy(e);
    }
    if (e.subtreeFlags & 10256) for (e = e.child; e !== null; ) dy(e), (e = e.sibling);
  }
  function dy(e) {
    switch (e.tag) {
      case 0:
      case 11:
      case 15:
        To(e), e.flags & 2048 && pl(9, e, e.return);
        break;
      case 3:
        To(e);
        break;
      case 12:
        To(e);
        break;
      case 22:
        var t = e.stateNode;
        e.memoizedState !== null && t._visibility & 2 && (e.return === null || e.return.tag !== 13)
          ? ((t._visibility &= -3), vu(e))
          : To(e);
        break;
      default:
        To(e);
    }
  }
  function vu(e) {
    var t = e.deletions;
    if ((e.flags & 16) !== 0) {
      if (t !== null)
        for (var a = 0; a < t.length; a++) {
          var l = t[a];
          (Ne = l), fy(l, e);
        }
      sy(e);
    }
    for (e = e.child; e !== null; ) {
      switch (((t = e), t.tag)) {
        case 0:
        case 11:
        case 15:
          pl(8, t, t.return), vu(t);
          break;
        case 22:
          (a = t.stateNode), a._visibility & 2 && ((a._visibility &= -3), vu(t));
          break;
        default:
          vu(t);
      }
      e = e.sibling;
    }
  }
  function fy(e, t) {
    for (; Ne !== null; ) {
      var a = Ne;
      switch (a.tag) {
        case 0:
        case 11:
        case 15:
          pl(8, a, t);
          break;
        case 23:
        case 22:
          if (a.memoizedState !== null && a.memoizedState.cachePool !== null) {
            var l = a.memoizedState.cachePool.pool;
            l != null && l.refCount++;
          }
          break;
        case 24:
          yn(a.memoizedState.cache);
      }
      if (((l = a.child), l !== null)) (l.return = a), (Ne = l);
      else
        e: for (a = e; Ne !== null; ) {
          l = Ne;
          var r = l.sibling,
            o = l.return;
          if ((ty(l), l === a)) {
            Ne = null;
            break e;
          }
          if (r !== null) {
            (r.return = o), (Ne = r);
            break e;
          }
          Ne = o;
        }
    }
  }
  var Ib = {
      getCacheForType: function (e) {
        var t = Xe(Ae),
          a = t.data.get(e);
        return a === void 0 && ((a = e()), t.data.set(e, a)), a;
      },
      cacheSignal: function () {
        return Xe(Ae).controller.signal;
      },
    },
    Eb = typeof WeakMap == 'function' ? WeakMap : Map,
    ae = 0,
    se = null,
    K = null,
    W = 0,
    re = 0,
    vt = null,
    el = !1,
    Kr = !1,
    Mf = !1,
    Ua = 0,
    Se = 0,
    hl = 0,
    kl = 0,
    Df = 0,
    bt = 0,
    _r = 0,
    Yo = null,
    dt = null,
    Td = !1,
    di = 0,
    cy = 0,
    Fu = 1 / 0,
    qu = null,
    ul = null,
    De = 0,
    il = null,
    zr = null,
    Ma = 0,
    Md = 0,
    Dd = null,
    my = null,
    Ko = 0,
    kd = null;
  function It() {
    return (ae & 2) !== 0 && W !== 0 ? W & -W : N.T !== null ? Bf() : bh();
  }
  function py() {
    if (bt === 0)
      if ((W & 536870912) === 0 || $) {
        var e = Yn;
        (Yn <<= 1), (Yn & 3932160) === 0 && (Yn = 262144), (bt = e);
      } else bt = 536870912;
    return (e = At.current), e !== null && (e.flags |= 32), bt;
  }
  function ft(e, t, a) {
    ((e === se && (re === 2 || re === 9)) || e.cancelPendingCommit !== null) &&
      (Fr(e, 0), tl(e, W, bt, !1)),
      pn(e, a),
      ((ae & 2) === 0 || e !== se) &&
        (e === se && ((ae & 2) === 0 && (kl |= a), Se === 4 && tl(e, W, bt, !1)), sa(e));
  }
  function hy(e, t, a) {
    if ((ae & 6) !== 0) throw Error(S(327));
    var l = (!a && (t & 127) === 0 && (t & e.expiredLanes) === 0) || mn(e, t),
      r = l ? Mb(e, t) : qs(e, t, !0),
      o = l;
    do {
      if (r === 0) {
        Kr && !l && tl(e, t, 0, !1);
        break;
      } else {
        if (((a = e.current.alternate), o && !Ab(a))) {
          (r = qs(e, t, !1)), (o = !1);
          continue;
        }
        if (r === 2) {
          if (((o = t), e.errorRecoveryDisabledLanes & o)) var n = 0;
          else (n = e.pendingLanes & -536870913), (n = n !== 0 ? n : n & 536870912 ? 536870912 : 0);
          if (n !== 0) {
            t = n;
            e: {
              var u = e;
              r = Yo;
              var i = u.current.memoizedState.isDehydrated;
              if ((i && (Fr(u, n).flags |= 256), (n = qs(u, n, !1)), n !== 2)) {
                if (Mf && !i) {
                  (u.errorRecoveryDisabledLanes |= o), (kl |= o), (r = 4);
                  break e;
                }
                (o = dt), (dt = r), o !== null && (dt === null ? (dt = o) : dt.push.apply(dt, o));
              }
              r = n;
            }
            if (((o = !1), r !== 2)) continue;
          }
        }
        if (r === 1) {
          Fr(e, 0), tl(e, t, 0, !0);
          break;
        }
        e: {
          switch (((l = e), (o = r), o)) {
            case 0:
            case 1:
              throw Error(S(345));
            case 4:
              if ((t & 4194048) !== t) break;
            case 6:
              tl(l, t, bt, !el);
              break e;
            case 2:
              dt = null;
              break;
            case 3:
            case 5:
              break;
            default:
              throw Error(S(329));
          }
          if ((t & 62914560) === t && ((r = di + 300 - Ct()), 10 < r)) {
            if ((tl(l, t, bt, !el), $u(l, 0, !0) !== 0)) break e;
            (Ma = t),
              (l.timeoutHandle = Uy(
                Bp.bind(null, l, a, dt, qu, Td, t, bt, kl, _r, el, o, 'Throttled', -0, 0),
                r
              ));
            break e;
          }
          Bp(l, a, dt, qu, Td, t, bt, kl, _r, el, o, null, -0, 0);
        }
      }
      break;
    } while (!0);
    sa(e);
  }
  function Bp(e, t, a, l, r, o, n, u, i, s, f, d, p, h) {
    if (((e.timeoutHandle = -1), (d = t.subtreeFlags), d & 8192 || (d & 16785408) === 16785408)) {
      (d = {
        stylesheets: null,
        count: 0,
        imgCount: 0,
        imgBytes: 0,
        suspenseyImages: [],
        waitingForImages: !0,
        waitingForViewTransition: !1,
        unsuspend: Ia,
      }),
        iy(t, o, d);
      var x = (o & 62914560) === o ? di - Ct() : (o & 4194048) === o ? cy - Ct() : 0;
      if (((x = d0(d, x)), x !== null)) {
        (Ma = o),
          (e.cancelPendingCommit = x(Up.bind(null, e, t, o, a, l, r, n, u, i, f, d, null, p, h))),
          tl(e, o, n, !s);
        return;
      }
    }
    Up(e, t, o, a, l, r, n, u, i);
  }
  function Ab(e) {
    for (var t = e; ; ) {
      var a = t.tag;
      if (
        (a === 0 || a === 11 || a === 15) &&
        t.flags & 16384 &&
        ((a = t.updateQueue), a !== null && ((a = a.stores), a !== null))
      )
        for (var l = 0; l < a.length; l++) {
          var r = a[l],
            o = r.getSnapshot;
          r = r.value;
          try {
            if (!Et(o(), r)) return !1;
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
  function tl(e, t, a, l) {
    (t &= ~Df),
      (t &= ~kl),
      (e.suspendedLanes |= t),
      (e.pingedLanes &= ~t),
      l && (e.warmLanes |= t),
      (l = e.expirationTimes);
    for (var r = t; 0 < r; ) {
      var o = 31 - Rt(r),
        n = 1 << o;
      (l[o] = -1), (r &= ~n);
    }
    a !== 0 && vh(e, a, t);
  }
  function fi() {
    return (ae & 6) === 0 ? (Ln(0, !1), !1) : !0;
  }
  function kf() {
    if (K !== null) {
      if (re === 0) var e = K.return;
      else (e = K), (Ea = ql = null), yf(e), (Mr = null), (an = 0), (e = K);
      for (; e !== null; ) Kg(e.alternate, e), (e = e.return);
      K = null;
    }
  }
  function Fr(e, t) {
    var a = e.timeoutHandle;
    a !== -1 && ((e.timeoutHandle = -1), Xb(a)),
      (a = e.cancelPendingCommit),
      a !== null && ((e.cancelPendingCommit = null), a()),
      (Ma = 0),
      kf(),
      (se = e),
      (K = a = Aa(e.current, null)),
      (W = t),
      (re = 0),
      (vt = null),
      (el = !1),
      (Kr = mn(e, t)),
      (Mf = !1),
      (_r = bt = Df = kl = hl = Se = 0),
      (dt = Yo = null),
      (Td = !1),
      (t & 8) !== 0 && (t |= t & 32);
    var l = e.entangledLanes;
    if (l !== 0)
      for (e = e.entanglements, l &= t; 0 < l; ) {
        var r = 31 - Rt(l),
          o = 1 << r;
        (t |= e[r]), (l &= ~o);
      }
    return (Ua = t), li(), a;
  }
  function gy(e, t) {
    (q = null),
      (N.H = rn),
      t === Yr || t === oi
        ? ((t = fp()), (re = 3))
        : t === df
          ? ((t = fp()), (re = 4))
          : (re =
              t === If
                ? 8
                : t !== null && typeof t == 'object' && typeof t.then == 'function'
                  ? 6
                  : 1),
      (vt = t),
      K === null && ((Se = 1), Pu(e, Pt(t, e.current)));
  }
  function yy() {
    var e = At.current;
    return e === null
      ? !0
      : (W & 4194048) === W
        ? zt === null
        : (W & 62914560) === W || (W & 536870912) !== 0
          ? e === zt
          : !1;
  }
  function xy() {
    var e = N.H;
    return (N.H = rn), e === null ? rn : e;
  }
  function vy() {
    var e = N.A;
    return (N.A = Ib), e;
  }
  function Gu() {
    (Se = 4),
      el || ((W & 4194048) !== W && At.current !== null) || (Kr = !0),
      ((hl & 134217727) === 0 && (kl & 134217727) === 0) || se === null || tl(se, W, bt, !1);
  }
  function qs(e, t, a) {
    var l = ae;
    ae |= 2;
    var r = xy(),
      o = vy();
    (se !== e || W !== t) && ((qu = null), Fr(e, t)), (t = !1);
    var n = Se;
    e: do
      try {
        if (re !== 0 && K !== null) {
          var u = K,
            i = vt;
          switch (re) {
            case 8:
              kf(), (n = 6);
              break e;
            case 3:
            case 2:
            case 9:
            case 6:
              At.current === null && (t = !0);
              var s = re;
              if (((re = 0), (vt = null), Rr(e, u, i, s), a && Kr)) {
                n = 0;
                break e;
              }
              break;
            default:
              (s = re), (re = 0), (vt = null), Rr(e, u, i, s);
          }
        }
        Tb(), (n = Se);
        break;
      } catch (f) {
        gy(e, f);
      }
    while (!0);
    return (
      t && e.shellSuspendCounter++,
      (Ea = ql = null),
      (ae = l),
      (N.H = r),
      (N.A = o),
      K === null && ((se = null), (W = 0), li()),
      n
    );
  }
  function Tb() {
    for (; K !== null; ) Ly(K);
  }
  function Mb(e, t) {
    var a = ae;
    ae |= 2;
    var l = xy(),
      r = vy();
    se !== e || W !== t ? ((qu = null), (Fu = Ct() + 500), Fr(e, t)) : (Kr = mn(e, t));
    e: do
      try {
        if (re !== 0 && K !== null) {
          t = K;
          var o = vt;
          t: switch (re) {
            case 1:
              (re = 0), (vt = null), Rr(e, t, o, 1);
              break;
            case 2:
            case 9:
              if (dp(o)) {
                (re = 0), (vt = null), Op(t);
                break;
              }
              (t = function () {
                (re !== 2 && re !== 9) || se !== e || (re = 7), sa(e);
              }),
                o.then(t, t);
              break e;
            case 3:
              re = 7;
              break e;
            case 4:
              re = 5;
              break e;
            case 7:
              dp(o) ? ((re = 0), (vt = null), Op(t)) : ((re = 0), (vt = null), Rr(e, t, o, 7));
              break;
            case 5:
              var n = null;
              switch (K.tag) {
                case 26:
                  n = K.memoizedState;
                case 5:
                case 27:
                  var u = K;
                  if (n ? zy(n) : u.stateNode.complete) {
                    (re = 0), (vt = null);
                    var i = u.sibling;
                    if (i !== null) K = i;
                    else {
                      var s = u.return;
                      s !== null ? ((K = s), ci(s)) : (K = null);
                    }
                    break t;
                  }
              }
              (re = 0), (vt = null), Rr(e, t, o, 5);
              break;
            case 6:
              (re = 0), (vt = null), Rr(e, t, o, 6);
              break;
            case 8:
              kf(), (Se = 6);
              break e;
            default:
              throw Error(S(462));
          }
        }
        Db();
        break;
      } catch (f) {
        gy(e, f);
      }
    while (!0);
    return (
      (Ea = ql = null),
      (N.H = l),
      (N.A = r),
      (ae = a),
      K !== null ? 0 : ((se = null), (W = 0), li(), Se)
    );
  }
  function Db() {
    for (; K !== null && !eS(); ) Ly(K);
  }
  function Ly(e) {
    var t = Yg(e.alternate, e, Ua);
    (e.memoizedProps = e.pendingProps), t === null ? ci(e) : (K = t);
  }
  function Op(e) {
    var t = e,
      a = t.alternate;
    switch (t.tag) {
      case 15:
      case 0:
        t = Ep(a, t, t.pendingProps, t.type, void 0, W);
        break;
      case 11:
        t = Ep(a, t, t.pendingProps, t.type.render, t.ref, W);
        break;
      case 5:
        yf(t);
      default:
        Kg(a, t), (t = K = Kh(t, Ua)), (t = Yg(a, t, Ua));
    }
    (e.memoizedProps = e.pendingProps), t === null ? ci(e) : (K = t);
  }
  function Rr(e, t, a, l) {
    (Ea = ql = null), yf(t), (Mr = null), (an = 0);
    var r = t.return;
    try {
      if (vb(e, r, t, a, W)) {
        (Se = 1), Pu(e, Pt(a, e.current)), (K = null);
        return;
      }
    } catch (o) {
      if (r !== null) throw ((K = r), o);
      (Se = 1), Pu(e, Pt(a, e.current)), (K = null);
      return;
    }
    t.flags & 32768
      ? ($ || l === 1
          ? (e = !0)
          : Kr || (W & 536870912) !== 0
            ? (e = !1)
            : ((el = e = !0),
              (l === 2 || l === 9 || l === 3 || l === 6) &&
                ((l = At.current), l !== null && l.tag === 13 && (l.flags |= 16384))),
        Sy(t, e))
      : ci(t);
  }
  function ci(e) {
    var t = e;
    do {
      if ((t.flags & 32768) !== 0) {
        Sy(t, el);
        return;
      }
      e = t.return;
      var a = bb(t.alternate, t, Ua);
      if (a !== null) {
        K = a;
        return;
      }
      if (((t = t.sibling), t !== null)) {
        K = t;
        return;
      }
      K = t = e;
    } while (t !== null);
    Se === 0 && (Se = 5);
  }
  function Sy(e, t) {
    do {
      var a = Cb(e.alternate, e);
      if (a !== null) {
        (a.flags &= 32767), (K = a);
        return;
      }
      if (
        ((a = e.return),
        a !== null && ((a.flags |= 32768), (a.subtreeFlags = 0), (a.deletions = null)),
        !t && ((e = e.sibling), e !== null))
      ) {
        K = e;
        return;
      }
      K = e = a;
    } while (e !== null);
    (Se = 6), (K = null);
  }
  function Up(e, t, a, l, r, o, n, u, i) {
    e.cancelPendingCommit = null;
    do mi();
    while (De !== 0);
    if ((ae & 6) !== 0) throw Error(S(327));
    if (t !== null) {
      if (t === e.current) throw Error(S(177));
      if (
        ((o = t.lanes | t.childLanes),
        (o |= af),
        dS(e, a, o, n, u, i),
        e === se && ((K = se = null), (W = 0)),
        (zr = t),
        (il = e),
        (Ma = a),
        (Md = o),
        (Dd = r),
        (my = l),
        (t.subtreeFlags & 10256) !== 0 || (t.flags & 10256) !== 0
          ? ((e.callbackNode = null),
            (e.callbackPriority = 0),
            Ub(Eu, function () {
              return Iy(), null;
            }))
          : ((e.callbackNode = null), (e.callbackPriority = 0)),
        (l = (t.flags & 13878) !== 0),
        (t.subtreeFlags & 13878) !== 0 || l)
      ) {
        (l = N.T), (N.T = null), (r = le.p), (le.p = 2), (n = ae), (ae |= 4);
        try {
          wb(e, t, a);
        } finally {
          (ae = n), (le.p = r), (N.T = l);
        }
      }
      (De = 1), by(), Cy(), wy();
    }
  }
  function by() {
    if (De === 1) {
      De = 0;
      var e = il,
        t = zr,
        a = (t.flags & 13878) !== 0;
      if ((t.subtreeFlags & 13878) !== 0 || a) {
        (a = N.T), (N.T = null);
        var l = le.p;
        le.p = 2;
        var r = ae;
        ae |= 4;
        try {
          oy(t, e);
          var o = Hd,
            n = zh(e.containerInfo),
            u = o.focusedElem,
            i = o.selectionRange;
          if (n !== u && u && u.ownerDocument && _h(u.ownerDocument.documentElement, u)) {
            if (i !== null && tf(u)) {
              var s = i.start,
                f = i.end;
              if ((f === void 0 && (f = s), 'selectionStart' in u))
                (u.selectionStart = s), (u.selectionEnd = Math.min(f, u.value.length));
              else {
                var d = u.ownerDocument || document,
                  p = (d && d.defaultView) || window;
                if (p.getSelection) {
                  var h = p.getSelection(),
                    x = u.textContent.length,
                    v = Math.min(i.start, x),
                    L = i.end === void 0 ? v : Math.min(i.end, x);
                  !h.extend && v > L && ((n = L), (L = v), (v = n));
                  var c = lp(u, v),
                    m = lp(u, L);
                  if (
                    c &&
                    m &&
                    (h.rangeCount !== 1 ||
                      h.anchorNode !== c.node ||
                      h.anchorOffset !== c.offset ||
                      h.focusNode !== m.node ||
                      h.focusOffset !== m.offset)
                  ) {
                    var g = d.createRange();
                    g.setStart(c.node, c.offset),
                      h.removeAllRanges(),
                      v > L
                        ? (h.addRange(g), h.extend(m.node, m.offset))
                        : (g.setEnd(m.node, m.offset), h.addRange(g));
                  }
                }
              }
            }
            for (d = [], h = u; (h = h.parentNode); )
              h.nodeType === 1 && d.push({ element: h, left: h.scrollLeft, top: h.scrollTop });
            for (typeof u.focus == 'function' && u.focus(), u = 0; u < d.length; u++) {
              var y = d[u];
              (y.element.scrollLeft = y.left), (y.element.scrollTop = y.top);
            }
          }
          (Wu = !!Ud), (Hd = Ud = null);
        } finally {
          (ae = r), (le.p = l), (N.T = a);
        }
      }
      (e.current = t), (De = 2);
    }
  }
  function Cy() {
    if (De === 2) {
      De = 0;
      var e = il,
        t = zr,
        a = (t.flags & 8772) !== 0;
      if ((t.subtreeFlags & 8772) !== 0 || a) {
        (a = N.T), (N.T = null);
        var l = le.p;
        le.p = 2;
        var r = ae;
        ae |= 4;
        try {
          ey(e, t.alternate, t);
        } finally {
          (ae = r), (le.p = l), (N.T = a);
        }
      }
      De = 3;
    }
  }
  function wy() {
    if (De === 4 || De === 3) {
      (De = 0), tS();
      var e = il,
        t = zr,
        a = Ma,
        l = my;
      (t.subtreeFlags & 10256) !== 0 || (t.flags & 10256) !== 0
        ? (De = 5)
        : ((De = 0), (zr = il = null), Ry(e, e.pendingLanes));
      var r = e.pendingLanes;
      if (
        (r === 0 && (ul = null),
        Kd(a),
        (t = t.stateNode),
        wt && typeof wt.onCommitFiberRoot == 'function')
      )
        try {
          wt.onCommitFiberRoot(cn, t, void 0, (t.current.flags & 128) === 128);
        } catch {}
      if (l !== null) {
        (t = N.T), (r = le.p), (le.p = 2), (N.T = null);
        try {
          for (var o = e.onRecoverableError, n = 0; n < l.length; n++) {
            var u = l[n];
            o(u.value, { componentStack: u.stack });
          }
        } finally {
          (N.T = t), (le.p = r);
        }
      }
      (Ma & 3) !== 0 && mi(),
        sa(e),
        (r = e.pendingLanes),
        (a & 261930) !== 0 && (r & 42) !== 0 ? (e === kd ? Ko++ : ((Ko = 0), (kd = e))) : (Ko = 0),
        Ln(0, !1);
    }
  }
  function Ry(e, t) {
    (e.pooledCacheLanes &= t) === 0 &&
      ((t = e.pooledCache), t != null && ((e.pooledCache = null), yn(t)));
  }
  function mi() {
    return by(), Cy(), wy(), Iy();
  }
  function Iy() {
    if (De !== 5) return !1;
    var e = il,
      t = Md;
    Md = 0;
    var a = Kd(Ma),
      l = N.T,
      r = le.p;
    try {
      (le.p = 32 > a ? 32 : a), (N.T = null), (a = Dd), (Dd = null);
      var o = il,
        n = Ma;
      if (((De = 0), (zr = il = null), (Ma = 0), (ae & 6) !== 0)) throw Error(S(331));
      var u = ae;
      if (
        ((ae |= 4),
        dy(o.current),
        uy(o, o.current, n, a),
        (ae = u),
        Ln(0, !1),
        wt && typeof wt.onPostCommitFiberRoot == 'function')
      )
        try {
          wt.onPostCommitFiberRoot(cn, o);
        } catch {}
      return !0;
    } finally {
      (le.p = r), (N.T = l), Ry(e, t);
    }
  }
  function Hp(e, t, a) {
    (t = Pt(a, t)), (t = Rd(e.stateNode, t, 2)), (e = nl(e, t, 2)), e !== null && (pn(e, 2), sa(e));
  }
  function oe(e, t, a) {
    if (e.tag === 3) Hp(e, e, a);
    else
      for (; t !== null; ) {
        if (t.tag === 3) {
          Hp(t, e, a);
          break;
        } else if (t.tag === 1) {
          var l = t.stateNode;
          if (
            typeof t.type.getDerivedStateFromError == 'function' ||
            (typeof l.componentDidCatch == 'function' && (ul === null || !ul.has(l)))
          ) {
            (e = Pt(a, e)),
              (a = Fg(2)),
              (l = nl(t, a, 2)),
              l !== null && (qg(a, l, t, e), pn(l, 2), sa(l));
            break;
          }
        }
        t = t.return;
      }
  }
  function Gs(e, t, a) {
    var l = e.pingCache;
    if (l === null) {
      l = e.pingCache = new Eb();
      var r = new Set();
      l.set(t, r);
    } else (r = l.get(t)), r === void 0 && ((r = new Set()), l.set(t, r));
    r.has(a) || ((Mf = !0), r.add(a), (e = kb.bind(null, e, t, a)), t.then(e, e));
  }
  function kb(e, t, a) {
    var l = e.pingCache;
    l !== null && l.delete(t),
      (e.pingedLanes |= e.suspendedLanes & a),
      (e.warmLanes &= ~a),
      se === e &&
        (W & a) === a &&
        (Se === 4 || (Se === 3 && (W & 62914560) === W && 300 > Ct() - di)
          ? (ae & 2) === 0 && Fr(e, 0)
          : (Df |= a),
        _r === W && (_r = 0)),
      sa(e);
  }
  function Ey(e, t) {
    t === 0 && (t = xh()), (e = Fl(e, t)), e !== null && (pn(e, t), sa(e));
  }
  function Bb(e) {
    var t = e.memoizedState,
      a = 0;
    t !== null && (a = t.retryLane), Ey(e, a);
  }
  function Ob(e, t) {
    var a = 0;
    switch (e.tag) {
      case 31:
      case 13:
        var l = e.stateNode,
          r = e.memoizedState;
        r !== null && (a = r.retryLane);
        break;
      case 19:
        l = e.stateNode;
        break;
      case 22:
        l = e.stateNode._retryCache;
        break;
      default:
        throw Error(S(314));
    }
    l !== null && l.delete(t), Ey(e, a);
  }
  function Ub(e, t) {
    return Xd(e, t);
  }
  var Vu = null,
    mr = null,
    Bd = !1,
    ju = !1,
    Vs = !1,
    al = 0;
  function sa(e) {
    e !== mr && e.next === null && (mr === null ? (Vu = mr = e) : (mr = mr.next = e)),
      (ju = !0),
      Bd || ((Bd = !0), Nb());
  }
  function Ln(e, t) {
    if (!Vs && ju) {
      Vs = !0;
      do
        for (var a = !1, l = Vu; l !== null; ) {
          if (!t)
            if (e !== 0) {
              var r = l.pendingLanes;
              if (r === 0) var o = 0;
              else {
                var n = l.suspendedLanes,
                  u = l.pingedLanes;
                (o = (1 << (31 - Rt(42 | e) + 1)) - 1),
                  (o &= r & ~(n & ~u)),
                  (o = o & 201326741 ? (o & 201326741) | 1 : o ? o | 2 : 0);
              }
              o !== 0 && ((a = !0), Np(l, o));
            } else
              (o = W),
                (o = $u(
                  l,
                  l === se ? o : 0,
                  l.cancelPendingCommit !== null || l.timeoutHandle !== -1
                )),
                (o & 3) === 0 || mn(l, o) || ((a = !0), Np(l, o));
          l = l.next;
        }
      while (a);
      Vs = !1;
    }
  }
  function Hb() {
    Ay();
  }
  function Ay() {
    ju = Bd = !1;
    var e = 0;
    al !== 0 && jb() && (e = al);
    for (var t = Ct(), a = null, l = Vu; l !== null; ) {
      var r = l.next,
        o = Ty(l, t);
      o === 0
        ? ((l.next = null), a === null ? (Vu = r) : (a.next = r), r === null && (mr = a))
        : ((a = l), (e !== 0 || (o & 3) !== 0) && (ju = !0)),
        (l = r);
    }
    (De !== 0 && De !== 5) || Ln(e, !1), al !== 0 && (al = 0);
  }
  function Ty(e, t) {
    for (
      var a = e.suspendedLanes,
        l = e.pingedLanes,
        r = e.expirationTimes,
        o = e.pendingLanes & -62914561;
      0 < o;

    ) {
      var n = 31 - Rt(o),
        u = 1 << n,
        i = r[n];
      i === -1
        ? ((u & a) === 0 || (u & l) !== 0) && (r[n] = sS(u, t))
        : i <= t && (e.expiredLanes |= u),
        (o &= ~u);
    }
    if (
      ((t = se),
      (a = W),
      (a = $u(e, e === t ? a : 0, e.cancelPendingCommit !== null || e.timeoutHandle !== -1)),
      (l = e.callbackNode),
      a === 0 || (e === t && (re === 2 || re === 9)) || e.cancelPendingCommit !== null)
    )
      return l !== null && l !== null && vs(l), (e.callbackNode = null), (e.callbackPriority = 0);
    if ((a & 3) === 0 || mn(e, a)) {
      if (((t = a & -a), t === e.callbackPriority)) return t;
      switch ((l !== null && vs(l), Kd(a))) {
        case 2:
        case 8:
          a = gh;
          break;
        case 32:
          a = Eu;
          break;
        case 268435456:
          a = yh;
          break;
        default:
          a = Eu;
      }
      return (
        (l = My.bind(null, e)), (a = Xd(a, l)), (e.callbackPriority = t), (e.callbackNode = a), t
      );
    }
    return l !== null && l !== null && vs(l), (e.callbackPriority = 2), (e.callbackNode = null), 2;
  }
  function My(e, t) {
    if (De !== 0 && De !== 5) return (e.callbackNode = null), (e.callbackPriority = 0), null;
    var a = e.callbackNode;
    if (mi() && e.callbackNode !== a) return null;
    var l = W;
    return (
      (l = $u(e, e === se ? l : 0, e.cancelPendingCommit !== null || e.timeoutHandle !== -1)),
      l === 0
        ? null
        : (hy(e, l, t),
          Ty(e, Ct()),
          e.callbackNode != null && e.callbackNode === a ? My.bind(null, e) : null)
    );
  }
  function Np(e, t) {
    if (mi()) return null;
    hy(e, t, !0);
  }
  function Nb() {
    Yb(function () {
      (ae & 6) !== 0 ? Xd(hh, Hb) : Ay();
    });
  }
  function Bf() {
    if (al === 0) {
      var e = Hr;
      e === 0 && ((e = Xn), (Xn <<= 1), (Xn & 261888) === 0 && (Xn = 256)), (al = e);
    }
    return al;
  }
  function Pp(e) {
    return e == null || typeof e == 'symbol' || typeof e == 'boolean'
      ? null
      : typeof e == 'function'
        ? e
        : du('' + e);
  }
  function _p(e, t) {
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
  function Pb(e, t, a, l, r) {
    if (t === 'submit' && a && a.stateNode === r) {
      var o = Pp((r[ct] || null).action),
        n = l.submitter;
      n &&
        ((t = (t = n[ct] || null) ? Pp(t.formAction) : n.getAttribute('formAction')),
        t !== null && ((o = t), (n = null)));
      var u = new ei('action', 'action', null, l, r);
      e.push({
        event: u,
        listeners: [
          {
            instance: null,
            listener: function () {
              if (l.defaultPrevented) {
                if (al !== 0) {
                  var i = n ? _p(r, n) : new FormData(r);
                  Cd(a, { pending: !0, data: i, method: r.method, action: o }, null, i);
                }
              } else
                typeof o == 'function' &&
                  (u.preventDefault(),
                  (i = n ? _p(r, n) : new FormData(r)),
                  Cd(a, { pending: !0, data: i, method: r.method, action: o }, o, i));
            },
            currentTarget: r,
          },
        ],
      });
    }
  }
  for (ru = 0; ru < fd.length; ru++)
    (ou = fd[ru]),
      (zp = ou.toLowerCase()),
      (Fp = ou[0].toUpperCase() + ou.slice(1)),
      Kt(zp, 'on' + Fp);
  var ou, zp, Fp, ru;
  Kt(qh, 'onAnimationEnd');
  Kt(Gh, 'onAnimationIteration');
  Kt(Vh, 'onAnimationStart');
  Kt('dblclick', 'onDoubleClick');
  Kt('focusin', 'onFocus');
  Kt('focusout', 'onBlur');
  Kt(ab, 'onTransitionRun');
  Kt(lb, 'onTransitionStart');
  Kt(rb, 'onTransitionCancel');
  Kt(jh, 'onTransitionEnd');
  Or('onMouseEnter', ['mouseout', 'mouseover']);
  Or('onMouseLeave', ['mouseout', 'mouseover']);
  Or('onPointerEnter', ['pointerout', 'pointerover']);
  Or('onPointerLeave', ['pointerout', 'pointerover']);
  Pl('onChange', 'change click focusin focusout input keydown keyup selectionchange'.split(' '));
  Pl(
    'onSelect',
    'focusout contextmenu dragend focusin keydown keyup mousedown mouseup selectionchange'.split(
      ' '
    )
  );
  Pl('onBeforeInput', ['compositionend', 'keypress', 'textInput', 'paste']);
  Pl('onCompositionEnd', 'compositionend focusout keydown keypress keyup mousedown'.split(' '));
  Pl('onCompositionStart', 'compositionstart focusout keydown keypress keyup mousedown'.split(' '));
  Pl(
    'onCompositionUpdate',
    'compositionupdate focusout keydown keypress keyup mousedown'.split(' ')
  );
  var on =
      'abort canplay canplaythrough durationchange emptied encrypted ended error loadeddata loadedmetadata loadstart pause play playing progress ratechange resize seeked seeking stalled suspend timeupdate volumechange waiting'.split(
        ' '
      ),
    _b = new Set(
      'beforetoggle cancel close invalid load scroll scrollend toggle'.split(' ').concat(on)
    );
  function Dy(e, t) {
    t = (t & 4) !== 0;
    for (var a = 0; a < e.length; a++) {
      var l = e[a],
        r = l.event;
      l = l.listeners;
      e: {
        var o = void 0;
        if (t)
          for (var n = l.length - 1; 0 <= n; n--) {
            var u = l[n],
              i = u.instance,
              s = u.currentTarget;
            if (((u = u.listener), i !== o && r.isPropagationStopped())) break e;
            (o = u), (r.currentTarget = s);
            try {
              o(r);
            } catch (f) {
              Tu(f);
            }
            (r.currentTarget = null), (o = i);
          }
        else
          for (n = 0; n < l.length; n++) {
            if (
              ((u = l[n]),
              (i = u.instance),
              (s = u.currentTarget),
              (u = u.listener),
              i !== o && r.isPropagationStopped())
            )
              break e;
            (o = u), (r.currentTarget = s);
            try {
              o(r);
            } catch (f) {
              Tu(f);
            }
            (r.currentTarget = null), (o = i);
          }
      }
    }
  }
  function Y(e, t) {
    var a = t[ld];
    a === void 0 && (a = t[ld] = new Set());
    var l = e + '__bubble';
    a.has(l) || (ky(t, e, 2, !1), a.add(l));
  }
  function js(e, t, a) {
    var l = 0;
    t && (l |= 4), ky(a, e, l, t);
  }
  var nu = '_reactListening' + Math.random().toString(36).slice(2);
  function Of(e) {
    if (!e[nu]) {
      (e[nu] = !0),
        Ch.forEach(function (a) {
          a !== 'selectionchange' && (_b.has(a) || js(a, !1, e), js(a, !0, e));
        });
      var t = e.nodeType === 9 ? e : e.ownerDocument;
      t === null || t[nu] || ((t[nu] = !0), js('selectionchange', !1, t));
    }
  }
  function ky(e, t, a, l) {
    switch (jy(t)) {
      case 2:
        var r = m0;
        break;
      case 8:
        r = p0;
        break;
      default:
        r = Pf;
    }
    (a = r.bind(null, t, a, e)),
      (r = void 0),
      !id || (t !== 'touchstart' && t !== 'touchmove' && t !== 'wheel') || (r = !0),
      l
        ? r !== void 0
          ? e.addEventListener(t, a, { capture: !0, passive: r })
          : e.addEventListener(t, a, !0)
        : r !== void 0
          ? e.addEventListener(t, a, { passive: r })
          : e.addEventListener(t, a, !1);
  }
  function Xs(e, t, a, l, r) {
    var o = l;
    if ((t & 1) === 0 && (t & 2) === 0 && l !== null)
      e: for (;;) {
        if (l === null) return;
        var n = l.tag;
        if (n === 3 || n === 4) {
          var u = l.stateNode.containerInfo;
          if (u === r) break;
          if (n === 4)
            for (n = l.return; n !== null; ) {
              var i = n.tag;
              if ((i === 3 || i === 4) && n.stateNode.containerInfo === r) return;
              n = n.return;
            }
          for (; u !== null; ) {
            if (((n = gr(u)), n === null)) return;
            if (((i = n.tag), i === 5 || i === 6 || i === 26 || i === 27)) {
              l = o = n;
              continue e;
            }
            u = u.parentNode;
          }
        }
        l = l.return;
      }
    Dh(function () {
      var s = o,
        f = Wd(a),
        d = [];
      e: {
        var p = Xh.get(e);
        if (p !== void 0) {
          var h = ei,
            x = e;
          switch (e) {
            case 'keypress':
              if (cu(a) === 0) break e;
            case 'keydown':
            case 'keyup':
              h = OS;
              break;
            case 'focusin':
              (x = 'focus'), (h = ws);
              break;
            case 'focusout':
              (x = 'blur'), (h = ws);
              break;
            case 'beforeblur':
            case 'afterblur':
              h = ws;
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
              h = Km;
              break;
            case 'drag':
            case 'dragend':
            case 'dragenter':
            case 'dragexit':
            case 'dragleave':
            case 'dragover':
            case 'dragstart':
            case 'drop':
              h = bS;
              break;
            case 'touchcancel':
            case 'touchend':
            case 'touchmove':
            case 'touchstart':
              h = NS;
              break;
            case qh:
            case Gh:
            case Vh:
              h = RS;
              break;
            case jh:
              h = _S;
              break;
            case 'scroll':
            case 'scrollend':
              h = LS;
              break;
            case 'wheel':
              h = FS;
              break;
            case 'copy':
            case 'cut':
            case 'paste':
              h = ES;
              break;
            case 'gotpointercapture':
            case 'lostpointercapture':
            case 'pointercancel':
            case 'pointerdown':
            case 'pointermove':
            case 'pointerout':
            case 'pointerover':
            case 'pointerup':
              h = Zm;
              break;
            case 'toggle':
            case 'beforetoggle':
              h = GS;
          }
          var v = (t & 4) !== 0,
            L = !v && (e === 'scroll' || e === 'scrollend'),
            c = v ? (p !== null ? p + 'Capture' : null) : p;
          v = [];
          for (var m = s, g; m !== null; ) {
            var y = m;
            if (
              ((g = y.stateNode),
              (y = y.tag),
              (y !== 5 && y !== 26 && y !== 27) ||
                g === null ||
                c === null ||
                ((y = Wo(m, c)), y != null && v.push(nn(m, y, g))),
              L)
            )
              break;
            m = m.return;
          }
          0 < v.length && ((p = new h(p, x, null, a, f)), d.push({ event: p, listeners: v }));
        }
      }
      if ((t & 7) === 0) {
        e: {
          if (
            ((p = e === 'mouseover' || e === 'pointerover'),
            (h = e === 'mouseout' || e === 'pointerout'),
            p && a !== ud && (x = a.relatedTarget || a.fromElement) && (gr(x) || x[Vr]))
          )
            break e;
          if (
            (h || p) &&
            ((p =
              f.window === f
                ? f
                : (p = f.ownerDocument)
                  ? p.defaultView || p.parentWindow
                  : window),
            h
              ? ((x = a.relatedTarget || a.toElement),
                (h = s),
                (x = x ? gr(x) : null),
                x !== null &&
                  ((L = fn(x)), (v = x.tag), x !== L || (v !== 5 && v !== 27 && v !== 6)) &&
                  (x = null))
              : ((h = null), (x = s)),
            h !== x)
          ) {
            if (
              ((v = Km),
              (y = 'onMouseLeave'),
              (c = 'onMouseEnter'),
              (m = 'mouse'),
              (e === 'pointerout' || e === 'pointerover') &&
                ((v = Zm), (y = 'onPointerLeave'), (c = 'onPointerEnter'), (m = 'pointer')),
              (L = h == null ? p : Oo(h)),
              (g = x == null ? p : Oo(x)),
              (p = new v(y, m + 'leave', h, a, f)),
              (p.target = L),
              (p.relatedTarget = g),
              (y = null),
              gr(f) === s &&
                ((v = new v(c, m + 'enter', x, a, f)),
                (v.target = g),
                (v.relatedTarget = L),
                (y = v)),
              (L = y),
              h && x)
            )
              t: {
                for (v = zb, c = h, m = x, g = 0, y = c; y; y = v(y)) g++;
                y = 0;
                for (var w = m; w; w = v(w)) y++;
                for (; 0 < g - y; ) (c = v(c)), g--;
                for (; 0 < y - g; ) (m = v(m)), y--;
                for (; g--; ) {
                  if (c === m || (m !== null && c === m.alternate)) {
                    v = c;
                    break t;
                  }
                  (c = v(c)), (m = v(m));
                }
                v = null;
              }
            else v = null;
            h !== null && qp(d, p, h, v, !1), x !== null && L !== null && qp(d, L, x, v, !0);
          }
        }
        e: {
          if (
            ((p = s ? Oo(s) : window),
            (h = p.nodeName && p.nodeName.toLowerCase()),
            h === 'select' || (h === 'input' && p.type === 'file'))
          )
            var U = ep;
          else if ($m(p))
            if (Nh) U = $S;
            else {
              U = WS;
              var C = ZS;
            }
          else
            (h = p.nodeName),
              !h || h.toLowerCase() !== 'input' || (p.type !== 'checkbox' && p.type !== 'radio')
                ? s && Zd(s.elementType) && (U = ep)
                : (U = JS);
          if (U && (U = U(e, s))) {
            Hh(d, U, a, f);
            break e;
          }
          C && C(e, p, s),
            e === 'focusout' &&
              s &&
              p.type === 'number' &&
              s.memoizedProps.value != null &&
              nd(p, 'number', p.value);
        }
        switch (((C = s ? Oo(s) : window), e)) {
          case 'focusin':
            ($m(C) || C.contentEditable === 'true') && ((vr = C), (sd = s), (_o = null));
            break;
          case 'focusout':
            _o = sd = vr = null;
            break;
          case 'mousedown':
            dd = !0;
            break;
          case 'contextmenu':
          case 'mouseup':
          case 'dragend':
            (dd = !1), rp(d, a, f);
            break;
          case 'selectionchange':
            if (tb) break;
          case 'keydown':
          case 'keyup':
            rp(d, a, f);
        }
        var b;
        if (ef)
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
          xr
            ? Oh(e, a) && (M = 'onCompositionEnd')
            : e === 'keydown' && a.keyCode === 229 && (M = 'onCompositionStart');
        M &&
          (Bh &&
            a.locale !== 'ko' &&
            (xr || M !== 'onCompositionStart'
              ? M === 'onCompositionEnd' && xr && (b = kh())
              : (($a = f), (Jd = 'value' in $a ? $a.value : $a.textContent), (xr = !0))),
          (C = Xu(s, M)),
          0 < C.length &&
            ((M = new Qm(M, e, null, a, f)),
            d.push({ event: M, listeners: C }),
            b ? (M.data = b) : ((b = Uh(a)), b !== null && (M.data = b)))),
          (b = jS ? XS(e, a) : YS(e, a)) &&
            ((M = Xu(s, 'onBeforeInput')),
            0 < M.length &&
              ((C = new Qm('onBeforeInput', 'beforeinput', null, a, f)),
              d.push({ event: C, listeners: M }),
              (C.data = b))),
          Pb(d, e, s, a, f);
      }
      Dy(d, t);
    });
  }
  function nn(e, t, a) {
    return { instance: e, listener: t, currentTarget: a };
  }
  function Xu(e, t) {
    for (var a = t + 'Capture', l = []; e !== null; ) {
      var r = e,
        o = r.stateNode;
      if (
        ((r = r.tag),
        (r !== 5 && r !== 26 && r !== 27) ||
          o === null ||
          ((r = Wo(e, a)),
          r != null && l.unshift(nn(e, r, o)),
          (r = Wo(e, t)),
          r != null && l.push(nn(e, r, o))),
        e.tag === 3)
      )
        return l;
      e = e.return;
    }
    return [];
  }
  function zb(e) {
    if (e === null) return null;
    do e = e.return;
    while (e && e.tag !== 5 && e.tag !== 27);
    return e || null;
  }
  function qp(e, t, a, l, r) {
    for (var o = t._reactName, n = []; a !== null && a !== l; ) {
      var u = a,
        i = u.alternate,
        s = u.stateNode;
      if (((u = u.tag), i !== null && i === l)) break;
      (u !== 5 && u !== 26 && u !== 27) ||
        s === null ||
        ((i = s),
        r
          ? ((s = Wo(a, o)), s != null && n.unshift(nn(a, s, i)))
          : r || ((s = Wo(a, o)), s != null && n.push(nn(a, s, i)))),
        (a = a.return);
    }
    n.length !== 0 && e.push({ event: t, listeners: n });
  }
  var Fb = /\r\n?/g,
    qb = /\u0000|\uFFFD/g;
  function Gp(e) {
    return (typeof e == 'string' ? e : '' + e)
      .replace(
        Fb,
        `
`
      )
      .replace(qb, '');
  }
  function By(e, t) {
    return (t = Gp(t)), Gp(e) === t;
  }
  function ne(e, t, a, l, r, o) {
    switch (a) {
      case 'children':
        typeof l == 'string'
          ? t === 'body' || (t === 'textarea' && l === '') || Ur(e, l)
          : (typeof l == 'number' || typeof l == 'bigint') && t !== 'body' && Ur(e, '' + l);
        break;
      case 'className':
        Qn(e, 'class', l);
        break;
      case 'tabIndex':
        Qn(e, 'tabindex', l);
        break;
      case 'dir':
      case 'role':
      case 'viewBox':
      case 'width':
      case 'height':
        Qn(e, a, l);
        break;
      case 'style':
        Mh(e, l, o);
        break;
      case 'data':
        if (t !== 'object') {
          Qn(e, 'data', l);
          break;
        }
      case 'src':
      case 'href':
        if (l === '' && (t !== 'a' || a !== 'href')) {
          e.removeAttribute(a);
          break;
        }
        if (l == null || typeof l == 'function' || typeof l == 'symbol' || typeof l == 'boolean') {
          e.removeAttribute(a);
          break;
        }
        (l = du('' + l)), e.setAttribute(a, l);
        break;
      case 'action':
      case 'formAction':
        if (typeof l == 'function') {
          e.setAttribute(
            a,
            "javascript:throw new Error('A React form was unexpectedly submitted. If you called form.submit() manually, consider using form.requestSubmit() instead. If you\\'re trying to use event.stopPropagation() in a submit event handler, consider also calling event.preventDefault().')"
          );
          break;
        } else
          typeof o == 'function' &&
            (a === 'formAction'
              ? (t !== 'input' && ne(e, t, 'name', r.name, r, null),
                ne(e, t, 'formEncType', r.formEncType, r, null),
                ne(e, t, 'formMethod', r.formMethod, r, null),
                ne(e, t, 'formTarget', r.formTarget, r, null))
              : (ne(e, t, 'encType', r.encType, r, null),
                ne(e, t, 'method', r.method, r, null),
                ne(e, t, 'target', r.target, r, null)));
        if (l == null || typeof l == 'symbol' || typeof l == 'boolean') {
          e.removeAttribute(a);
          break;
        }
        (l = du('' + l)), e.setAttribute(a, l);
        break;
      case 'onClick':
        l != null && (e.onclick = Ia);
        break;
      case 'onScroll':
        l != null && Y('scroll', e);
        break;
      case 'onScrollEnd':
        l != null && Y('scrollend', e);
        break;
      case 'dangerouslySetInnerHTML':
        if (l != null) {
          if (typeof l != 'object' || !('__html' in l)) throw Error(S(61));
          if (((a = l.__html), a != null)) {
            if (r.children != null) throw Error(S(60));
            e.innerHTML = a;
          }
        }
        break;
      case 'multiple':
        e.multiple = l && typeof l != 'function' && typeof l != 'symbol';
        break;
      case 'muted':
        e.muted = l && typeof l != 'function' && typeof l != 'symbol';
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
        if (l == null || typeof l == 'function' || typeof l == 'boolean' || typeof l == 'symbol') {
          e.removeAttribute('xlink:href');
          break;
        }
        (a = du('' + l)), e.setAttributeNS('http://www.w3.org/1999/xlink', 'xlink:href', a);
        break;
      case 'contentEditable':
      case 'spellCheck':
      case 'draggable':
      case 'value':
      case 'autoReverse':
      case 'externalResourcesRequired':
      case 'focusable':
      case 'preserveAlpha':
        l != null && typeof l != 'function' && typeof l != 'symbol'
          ? e.setAttribute(a, '' + l)
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
        l && typeof l != 'function' && typeof l != 'symbol'
          ? e.setAttribute(a, '')
          : e.removeAttribute(a);
        break;
      case 'capture':
      case 'download':
        l === !0
          ? e.setAttribute(a, '')
          : l !== !1 && l != null && typeof l != 'function' && typeof l != 'symbol'
            ? e.setAttribute(a, l)
            : e.removeAttribute(a);
        break;
      case 'cols':
      case 'rows':
      case 'size':
      case 'span':
        l != null && typeof l != 'function' && typeof l != 'symbol' && !isNaN(l) && 1 <= l
          ? e.setAttribute(a, l)
          : e.removeAttribute(a);
        break;
      case 'rowSpan':
      case 'start':
        l == null || typeof l == 'function' || typeof l == 'symbol' || isNaN(l)
          ? e.removeAttribute(a)
          : e.setAttribute(a, l);
        break;
      case 'popover':
        Y('beforetoggle', e), Y('toggle', e), su(e, 'popover', l);
        break;
      case 'xlinkActuate':
        xa(e, 'http://www.w3.org/1999/xlink', 'xlink:actuate', l);
        break;
      case 'xlinkArcrole':
        xa(e, 'http://www.w3.org/1999/xlink', 'xlink:arcrole', l);
        break;
      case 'xlinkRole':
        xa(e, 'http://www.w3.org/1999/xlink', 'xlink:role', l);
        break;
      case 'xlinkShow':
        xa(e, 'http://www.w3.org/1999/xlink', 'xlink:show', l);
        break;
      case 'xlinkTitle':
        xa(e, 'http://www.w3.org/1999/xlink', 'xlink:title', l);
        break;
      case 'xlinkType':
        xa(e, 'http://www.w3.org/1999/xlink', 'xlink:type', l);
        break;
      case 'xmlBase':
        xa(e, 'http://www.w3.org/XML/1998/namespace', 'xml:base', l);
        break;
      case 'xmlLang':
        xa(e, 'http://www.w3.org/XML/1998/namespace', 'xml:lang', l);
        break;
      case 'xmlSpace':
        xa(e, 'http://www.w3.org/XML/1998/namespace', 'xml:space', l);
        break;
      case 'is':
        su(e, 'is', l);
        break;
      case 'innerText':
      case 'textContent':
        break;
      default:
        (!(2 < a.length) || (a[0] !== 'o' && a[0] !== 'O') || (a[1] !== 'n' && a[1] !== 'N')) &&
          ((a = xS.get(a) || a), su(e, a, l));
    }
  }
  function Od(e, t, a, l, r, o) {
    switch (a) {
      case 'style':
        Mh(e, l, o);
        break;
      case 'dangerouslySetInnerHTML':
        if (l != null) {
          if (typeof l != 'object' || !('__html' in l)) throw Error(S(61));
          if (((a = l.__html), a != null)) {
            if (r.children != null) throw Error(S(60));
            e.innerHTML = a;
          }
        }
        break;
      case 'children':
        typeof l == 'string'
          ? Ur(e, l)
          : (typeof l == 'number' || typeof l == 'bigint') && Ur(e, '' + l);
        break;
      case 'onScroll':
        l != null && Y('scroll', e);
        break;
      case 'onScrollEnd':
        l != null && Y('scrollend', e);
        break;
      case 'onClick':
        l != null && (e.onclick = Ia);
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
        if (!wh.hasOwnProperty(a))
          e: {
            if (
              a[0] === 'o' &&
              a[1] === 'n' &&
              ((r = a.endsWith('Capture')),
              (t = a.slice(2, r ? a.length - 7 : void 0)),
              (o = e[ct] || null),
              (o = o != null ? o[a] : null),
              typeof o == 'function' && e.removeEventListener(t, o, r),
              typeof l == 'function')
            ) {
              typeof o != 'function' &&
                o !== null &&
                (a in e ? (e[a] = null) : e.hasAttribute(a) && e.removeAttribute(a)),
                e.addEventListener(t, l, r);
              break e;
            }
            a in e ? (e[a] = l) : l === !0 ? e.setAttribute(a, '') : su(e, a, l);
          }
    }
  }
  function Ye(e, t, a) {
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
        Y('error', e), Y('load', e);
        var l = !1,
          r = !1,
          o;
        for (o in a)
          if (a.hasOwnProperty(o)) {
            var n = a[o];
            if (n != null)
              switch (o) {
                case 'src':
                  l = !0;
                  break;
                case 'srcSet':
                  r = !0;
                  break;
                case 'children':
                case 'dangerouslySetInnerHTML':
                  throw Error(S(137, t));
                default:
                  ne(e, t, o, n, a, null);
              }
          }
        r && ne(e, t, 'srcSet', a.srcSet, a, null), l && ne(e, t, 'src', a.src, a, null);
        return;
      case 'input':
        Y('invalid', e);
        var u = (o = n = r = null),
          i = null,
          s = null;
        for (l in a)
          if (a.hasOwnProperty(l)) {
            var f = a[l];
            if (f != null)
              switch (l) {
                case 'name':
                  r = f;
                  break;
                case 'type':
                  n = f;
                  break;
                case 'checked':
                  i = f;
                  break;
                case 'defaultChecked':
                  s = f;
                  break;
                case 'value':
                  o = f;
                  break;
                case 'defaultValue':
                  u = f;
                  break;
                case 'children':
                case 'dangerouslySetInnerHTML':
                  if (f != null) throw Error(S(137, t));
                  break;
                default:
                  ne(e, t, l, f, a, null);
              }
          }
        Eh(e, o, u, i, s, n, r, !1);
        return;
      case 'select':
        Y('invalid', e), (l = n = o = null);
        for (r in a)
          if (a.hasOwnProperty(r) && ((u = a[r]), u != null))
            switch (r) {
              case 'value':
                o = u;
                break;
              case 'defaultValue':
                n = u;
                break;
              case 'multiple':
                l = u;
              default:
                ne(e, t, r, u, a, null);
            }
        (t = o),
          (a = n),
          (e.multiple = !!l),
          t != null ? Er(e, !!l, t, !1) : a != null && Er(e, !!l, a, !0);
        return;
      case 'textarea':
        Y('invalid', e), (o = r = l = null);
        for (n in a)
          if (a.hasOwnProperty(n) && ((u = a[n]), u != null))
            switch (n) {
              case 'value':
                l = u;
                break;
              case 'defaultValue':
                r = u;
                break;
              case 'children':
                o = u;
                break;
              case 'dangerouslySetInnerHTML':
                if (u != null) throw Error(S(91));
                break;
              default:
                ne(e, t, n, u, a, null);
            }
        Th(e, l, r, o);
        return;
      case 'option':
        for (i in a)
          if (a.hasOwnProperty(i) && ((l = a[i]), l != null))
            switch (i) {
              case 'selected':
                e.selected = l && typeof l != 'function' && typeof l != 'symbol';
                break;
              default:
                ne(e, t, i, l, a, null);
            }
        return;
      case 'dialog':
        Y('beforetoggle', e), Y('toggle', e), Y('cancel', e), Y('close', e);
        break;
      case 'iframe':
      case 'object':
        Y('load', e);
        break;
      case 'video':
      case 'audio':
        for (l = 0; l < on.length; l++) Y(on[l], e);
        break;
      case 'image':
        Y('error', e), Y('load', e);
        break;
      case 'details':
        Y('toggle', e);
        break;
      case 'embed':
      case 'source':
      case 'link':
        Y('error', e), Y('load', e);
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
        for (s in a)
          if (a.hasOwnProperty(s) && ((l = a[s]), l != null))
            switch (s) {
              case 'children':
              case 'dangerouslySetInnerHTML':
                throw Error(S(137, t));
              default:
                ne(e, t, s, l, a, null);
            }
        return;
      default:
        if (Zd(t)) {
          for (f in a)
            a.hasOwnProperty(f) && ((l = a[f]), l !== void 0 && Od(e, t, f, l, a, void 0));
          return;
        }
    }
    for (u in a) a.hasOwnProperty(u) && ((l = a[u]), l != null && ne(e, t, u, l, a, null));
  }
  function Gb(e, t, a, l) {
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
        var r = null,
          o = null,
          n = null,
          u = null,
          i = null,
          s = null,
          f = null;
        for (h in a) {
          var d = a[h];
          if (a.hasOwnProperty(h) && d != null)
            switch (h) {
              case 'checked':
                break;
              case 'value':
                break;
              case 'defaultValue':
                i = d;
              default:
                l.hasOwnProperty(h) || ne(e, t, h, null, l, d);
            }
        }
        for (var p in l) {
          var h = l[p];
          if (((d = a[p]), l.hasOwnProperty(p) && (h != null || d != null)))
            switch (p) {
              case 'type':
                o = h;
                break;
              case 'name':
                r = h;
                break;
              case 'checked':
                s = h;
                break;
              case 'defaultChecked':
                f = h;
                break;
              case 'value':
                n = h;
                break;
              case 'defaultValue':
                u = h;
                break;
              case 'children':
              case 'dangerouslySetInnerHTML':
                if (h != null) throw Error(S(137, t));
                break;
              default:
                h !== d && ne(e, t, p, h, l, d);
            }
        }
        od(e, n, u, i, s, f, o, r);
        return;
      case 'select':
        h = n = u = p = null;
        for (o in a)
          if (((i = a[o]), a.hasOwnProperty(o) && i != null))
            switch (o) {
              case 'value':
                break;
              case 'multiple':
                h = i;
              default:
                l.hasOwnProperty(o) || ne(e, t, o, null, l, i);
            }
        for (r in l)
          if (((o = l[r]), (i = a[r]), l.hasOwnProperty(r) && (o != null || i != null)))
            switch (r) {
              case 'value':
                p = o;
                break;
              case 'defaultValue':
                u = o;
                break;
              case 'multiple':
                n = o;
              default:
                o !== i && ne(e, t, r, o, l, i);
            }
        (t = u),
          (a = n),
          (l = h),
          p != null
            ? Er(e, !!a, p, !1)
            : !!l != !!a && (t != null ? Er(e, !!a, t, !0) : Er(e, !!a, a ? [] : '', !1));
        return;
      case 'textarea':
        h = p = null;
        for (u in a)
          if (((r = a[u]), a.hasOwnProperty(u) && r != null && !l.hasOwnProperty(u)))
            switch (u) {
              case 'value':
                break;
              case 'children':
                break;
              default:
                ne(e, t, u, null, l, r);
            }
        for (n in l)
          if (((r = l[n]), (o = a[n]), l.hasOwnProperty(n) && (r != null || o != null)))
            switch (n) {
              case 'value':
                p = r;
                break;
              case 'defaultValue':
                h = r;
                break;
              case 'children':
                break;
              case 'dangerouslySetInnerHTML':
                if (r != null) throw Error(S(91));
                break;
              default:
                r !== o && ne(e, t, n, r, l, o);
            }
        Ah(e, p, h);
        return;
      case 'option':
        for (var x in a)
          if (((p = a[x]), a.hasOwnProperty(x) && p != null && !l.hasOwnProperty(x)))
            switch (x) {
              case 'selected':
                e.selected = !1;
                break;
              default:
                ne(e, t, x, null, l, p);
            }
        for (i in l)
          if (((p = l[i]), (h = a[i]), l.hasOwnProperty(i) && p !== h && (p != null || h != null)))
            switch (i) {
              case 'selected':
                e.selected = p && typeof p != 'function' && typeof p != 'symbol';
                break;
              default:
                ne(e, t, i, p, l, h);
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
        for (var v in a)
          (p = a[v]),
            a.hasOwnProperty(v) && p != null && !l.hasOwnProperty(v) && ne(e, t, v, null, l, p);
        for (s in l)
          if (((p = l[s]), (h = a[s]), l.hasOwnProperty(s) && p !== h && (p != null || h != null)))
            switch (s) {
              case 'children':
              case 'dangerouslySetInnerHTML':
                if (p != null) throw Error(S(137, t));
                break;
              default:
                ne(e, t, s, p, l, h);
            }
        return;
      default:
        if (Zd(t)) {
          for (var L in a)
            (p = a[L]),
              a.hasOwnProperty(L) &&
                p !== void 0 &&
                !l.hasOwnProperty(L) &&
                Od(e, t, L, void 0, l, p);
          for (f in l)
            (p = l[f]),
              (h = a[f]),
              !l.hasOwnProperty(f) ||
                p === h ||
                (p === void 0 && h === void 0) ||
                Od(e, t, f, p, l, h);
          return;
        }
    }
    for (var c in a)
      (p = a[c]),
        a.hasOwnProperty(c) && p != null && !l.hasOwnProperty(c) && ne(e, t, c, null, l, p);
    for (d in l)
      (p = l[d]),
        (h = a[d]),
        !l.hasOwnProperty(d) || p === h || (p == null && h == null) || ne(e, t, d, p, l, h);
  }
  function Vp(e) {
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
  function Vb() {
    if (typeof performance.getEntriesByType == 'function') {
      for (
        var e = 0, t = 0, a = performance.getEntriesByType('resource'), l = 0;
        l < a.length;
        l++
      ) {
        var r = a[l],
          o = r.transferSize,
          n = r.initiatorType,
          u = r.duration;
        if (o && u && Vp(n)) {
          for (n = 0, u = r.responseEnd, l += 1; l < a.length; l++) {
            var i = a[l],
              s = i.startTime;
            if (s > u) break;
            var f = i.transferSize,
              d = i.initiatorType;
            f && Vp(d) && ((i = i.responseEnd), (n += f * (i < u ? 1 : (u - s) / (i - s))));
          }
          if ((--l, (t += (8 * (o + n)) / (r.duration / 1e3)), e++, 10 < e)) break;
        }
      }
      if (0 < e) return t / e / 1e6;
    }
    return navigator.connection && ((e = navigator.connection.downlink), typeof e == 'number')
      ? e
      : 5;
  }
  var Ud = null,
    Hd = null;
  function Yu(e) {
    return e.nodeType === 9 ? e : e.ownerDocument;
  }
  function jp(e) {
    switch (e) {
      case 'http://www.w3.org/2000/svg':
        return 1;
      case 'http://www.w3.org/1998/Math/MathML':
        return 2;
      default:
        return 0;
    }
  }
  function Oy(e, t) {
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
  function Nd(e, t) {
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
  var Ys = null;
  function jb() {
    var e = window.event;
    return e && e.type === 'popstate' ? (e === Ys ? !1 : ((Ys = e), !0)) : ((Ys = null), !1);
  }
  var Uy = typeof setTimeout == 'function' ? setTimeout : void 0,
    Xb = typeof clearTimeout == 'function' ? clearTimeout : void 0,
    Xp = typeof Promise == 'function' ? Promise : void 0,
    Yb =
      typeof queueMicrotask == 'function'
        ? queueMicrotask
        : typeof Xp < 'u'
          ? function (e) {
              return Xp.resolve(null).then(e).catch(Kb);
            }
          : Uy;
  function Kb(e) {
    setTimeout(function () {
      throw e;
    });
  }
  function yl(e) {
    return e === 'head';
  }
  function Yp(e, t) {
    var a = t,
      l = 0;
    do {
      var r = a.nextSibling;
      if ((e.removeChild(a), r && r.nodeType === 8))
        if (((a = r.data), a === '/$' || a === '/&')) {
          if (l === 0) {
            e.removeChild(r), Gr(t);
            return;
          }
          l--;
        } else if (a === '$' || a === '$?' || a === '$~' || a === '$!' || a === '&') l++;
        else if (a === 'html') Qo(e.ownerDocument.documentElement);
        else if (a === 'head') {
          (a = e.ownerDocument.head), Qo(a);
          for (var o = a.firstChild; o; ) {
            var n = o.nextSibling,
              u = o.nodeName;
            o[hn] ||
              u === 'SCRIPT' ||
              u === 'STYLE' ||
              (u === 'LINK' && o.rel.toLowerCase() === 'stylesheet') ||
              a.removeChild(o),
              (o = n);
          }
        } else a === 'body' && Qo(e.ownerDocument.body);
      a = r;
    } while (a);
    Gr(t);
  }
  function Kp(e, t) {
    var a = e;
    e = 0;
    do {
      var l = a.nextSibling;
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
        l && l.nodeType === 8)
      )
        if (((a = l.data), a === '/$')) {
          if (e === 0) break;
          e--;
        } else (a !== '$' && a !== '$?' && a !== '$~' && a !== '$!') || e++;
      a = l;
    } while (a);
  }
  function Pd(e) {
    var t = e.firstChild;
    for (t && t.nodeType === 10 && (t = t.nextSibling); t; ) {
      var a = t;
      switch (((t = t.nextSibling), a.nodeName)) {
        case 'HTML':
        case 'HEAD':
        case 'BODY':
          Pd(a), Qd(a);
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
  function Qb(e, t, a, l) {
    for (; e.nodeType === 1; ) {
      var r = a;
      if (e.nodeName.toLowerCase() !== t.toLowerCase()) {
        if (!l && (e.nodeName !== 'INPUT' || e.type !== 'hidden')) break;
      } else if (l) {
        if (!e[hn])
          switch (t) {
            case 'meta':
              if (!e.hasAttribute('itemprop')) break;
              return e;
            case 'link':
              if (
                ((o = e.getAttribute('rel')),
                o === 'stylesheet' && e.hasAttribute('data-precedence'))
              )
                break;
              if (
                o !== r.rel ||
                e.getAttribute('href') !== (r.href == null || r.href === '' ? null : r.href) ||
                e.getAttribute('crossorigin') !== (r.crossOrigin == null ? null : r.crossOrigin) ||
                e.getAttribute('title') !== (r.title == null ? null : r.title)
              )
                break;
              return e;
            case 'style':
              if (e.hasAttribute('data-precedence')) break;
              return e;
            case 'script':
              if (
                ((o = e.getAttribute('src')),
                (o !== (r.src == null ? null : r.src) ||
                  e.getAttribute('type') !== (r.type == null ? null : r.type) ||
                  e.getAttribute('crossorigin') !==
                    (r.crossOrigin == null ? null : r.crossOrigin)) &&
                  o &&
                  e.hasAttribute('async') &&
                  !e.hasAttribute('itemprop'))
              )
                break;
              return e;
            default:
              return e;
          }
      } else if (t === 'input' && e.type === 'hidden') {
        var o = r.name == null ? null : '' + r.name;
        if (r.type === 'hidden' && e.getAttribute('name') === o) return e;
      } else return e;
      if (((e = Ft(e.nextSibling)), e === null)) break;
    }
    return null;
  }
  function Zb(e, t, a) {
    if (t === '') return null;
    for (; e.nodeType !== 3; )
      if (
        ((e.nodeType !== 1 || e.nodeName !== 'INPUT' || e.type !== 'hidden') && !a) ||
        ((e = Ft(e.nextSibling)), e === null)
      )
        return null;
    return e;
  }
  function Hy(e, t) {
    for (; e.nodeType !== 8; )
      if (
        ((e.nodeType !== 1 || e.nodeName !== 'INPUT' || e.type !== 'hidden') && !t) ||
        ((e = Ft(e.nextSibling)), e === null)
      )
        return null;
    return e;
  }
  function _d(e) {
    return e.data === '$?' || e.data === '$~';
  }
  function zd(e) {
    return e.data === '$!' || (e.data === '$?' && e.ownerDocument.readyState !== 'loading');
  }
  function Wb(e, t) {
    var a = e.ownerDocument;
    if (e.data === '$~') e._reactRetry = t;
    else if (e.data !== '$?' || a.readyState !== 'loading') t();
    else {
      var l = function () {
        t(), a.removeEventListener('DOMContentLoaded', l);
      };
      a.addEventListener('DOMContentLoaded', l), (e._reactRetry = l);
    }
  }
  function Ft(e) {
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
  var Fd = null;
  function Qp(e) {
    e = e.nextSibling;
    for (var t = 0; e; ) {
      if (e.nodeType === 8) {
        var a = e.data;
        if (a === '/$' || a === '/&') {
          if (t === 0) return Ft(e.nextSibling);
          t--;
        } else (a !== '$' && a !== '$!' && a !== '$?' && a !== '$~' && a !== '&') || t++;
      }
      e = e.nextSibling;
    }
    return null;
  }
  function Zp(e) {
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
  function Ny(e, t, a) {
    switch (((t = Yu(a)), e)) {
      case 'html':
        if (((e = t.documentElement), !e)) throw Error(S(452));
        return e;
      case 'head':
        if (((e = t.head), !e)) throw Error(S(453));
        return e;
      case 'body':
        if (((e = t.body), !e)) throw Error(S(454));
        return e;
      default:
        throw Error(S(451));
    }
  }
  function Qo(e) {
    for (var t = e.attributes; t.length; ) e.removeAttributeNode(t[0]);
    Qd(e);
  }
  var qt = new Map(),
    Wp = new Set();
  function Ku(e) {
    return typeof e.getRootNode == 'function'
      ? e.getRootNode()
      : e.nodeType === 9
        ? e
        : e.ownerDocument;
  }
  var Ha = le.d;
  le.d = { f: Jb, r: $b, D: e0, C: t0, L: a0, m: l0, X: o0, S: r0, M: n0 };
  function Jb() {
    var e = Ha.f(),
      t = fi();
    return e || t;
  }
  function $b(e) {
    var t = jr(e);
    t !== null && t.tag === 5 && t.type === 'form' ? Mg(t) : Ha.r(e);
  }
  var Qr = typeof document > 'u' ? null : document;
  function Py(e, t, a) {
    var l = Qr;
    if (l && typeof t == 'string' && t) {
      var r = Nt(t);
      (r = 'link[rel="' + e + '"][href="' + r + '"]'),
        typeof a == 'string' && (r += '[crossorigin="' + a + '"]'),
        Wp.has(r) ||
          (Wp.add(r),
          (e = { rel: e, crossOrigin: a, href: t }),
          l.querySelector(r) === null &&
            ((t = l.createElement('link')), Ye(t, 'link', e), Pe(t), l.head.appendChild(t)));
    }
  }
  function e0(e) {
    Ha.D(e), Py('dns-prefetch', e, null);
  }
  function t0(e, t) {
    Ha.C(e, t), Py('preconnect', e, t);
  }
  function a0(e, t, a) {
    Ha.L(e, t, a);
    var l = Qr;
    if (l && e && t) {
      var r = 'link[rel="preload"][as="' + Nt(t) + '"]';
      t === 'image' && a && a.imageSrcSet
        ? ((r += '[imagesrcset="' + Nt(a.imageSrcSet) + '"]'),
          typeof a.imageSizes == 'string' && (r += '[imagesizes="' + Nt(a.imageSizes) + '"]'))
        : (r += '[href="' + Nt(e) + '"]');
      var o = r;
      switch (t) {
        case 'style':
          o = qr(e);
          break;
        case 'script':
          o = Zr(e);
      }
      qt.has(o) ||
        ((e = ye(
          { rel: 'preload', href: t === 'image' && a && a.imageSrcSet ? void 0 : e, as: t },
          a
        )),
        qt.set(o, e),
        l.querySelector(r) !== null ||
          (t === 'style' && l.querySelector(Sn(o))) ||
          (t === 'script' && l.querySelector(bn(o))) ||
          ((t = l.createElement('link')), Ye(t, 'link', e), Pe(t), l.head.appendChild(t)));
    }
  }
  function l0(e, t) {
    Ha.m(e, t);
    var a = Qr;
    if (a && e) {
      var l = t && typeof t.as == 'string' ? t.as : 'script',
        r = 'link[rel="modulepreload"][as="' + Nt(l) + '"][href="' + Nt(e) + '"]',
        o = r;
      switch (l) {
        case 'audioworklet':
        case 'paintworklet':
        case 'serviceworker':
        case 'sharedworker':
        case 'worker':
        case 'script':
          o = Zr(e);
      }
      if (
        !qt.has(o) &&
        ((e = ye({ rel: 'modulepreload', href: e }, t)), qt.set(o, e), a.querySelector(r) === null)
      ) {
        switch (l) {
          case 'audioworklet':
          case 'paintworklet':
          case 'serviceworker':
          case 'sharedworker':
          case 'worker':
          case 'script':
            if (a.querySelector(bn(o))) return;
        }
        (l = a.createElement('link')), Ye(l, 'link', e), Pe(l), a.head.appendChild(l);
      }
    }
  }
  function r0(e, t, a) {
    Ha.S(e, t, a);
    var l = Qr;
    if (l && e) {
      var r = Ir(l).hoistableStyles,
        o = qr(e);
      t = t || 'default';
      var n = r.get(o);
      if (!n) {
        var u = { loading: 0, preload: null };
        if ((n = l.querySelector(Sn(o)))) u.loading = 5;
        else {
          (e = ye({ rel: 'stylesheet', href: e, 'data-precedence': t }, a)),
            (a = qt.get(o)) && Uf(e, a);
          var i = (n = l.createElement('link'));
          Pe(i),
            Ye(i, 'link', e),
            (i._p = new Promise(function (s, f) {
              (i.onload = s), (i.onerror = f);
            })),
            i.addEventListener('load', function () {
              u.loading |= 1;
            }),
            i.addEventListener('error', function () {
              u.loading |= 2;
            }),
            (u.loading |= 4),
            Lu(n, t, l);
        }
        (n = { type: 'stylesheet', instance: n, count: 1, state: u }), r.set(o, n);
      }
    }
  }
  function o0(e, t) {
    Ha.X(e, t);
    var a = Qr;
    if (a && e) {
      var l = Ir(a).hoistableScripts,
        r = Zr(e),
        o = l.get(r);
      o ||
        ((o = a.querySelector(bn(r))),
        o ||
          ((e = ye({ src: e, async: !0 }, t)),
          (t = qt.get(r)) && Hf(e, t),
          (o = a.createElement('script')),
          Pe(o),
          Ye(o, 'link', e),
          a.head.appendChild(o)),
        (o = { type: 'script', instance: o, count: 1, state: null }),
        l.set(r, o));
    }
  }
  function n0(e, t) {
    Ha.M(e, t);
    var a = Qr;
    if (a && e) {
      var l = Ir(a).hoistableScripts,
        r = Zr(e),
        o = l.get(r);
      o ||
        ((o = a.querySelector(bn(r))),
        o ||
          ((e = ye({ src: e, async: !0, type: 'module' }, t)),
          (t = qt.get(r)) && Hf(e, t),
          (o = a.createElement('script')),
          Pe(o),
          Ye(o, 'link', e),
          a.head.appendChild(o)),
        (o = { type: 'script', instance: o, count: 1, state: null }),
        l.set(r, o));
    }
  }
  function Jp(e, t, a, l) {
    var r = (r = ll.current) ? Ku(r) : null;
    if (!r) throw Error(S(446));
    switch (e) {
      case 'meta':
      case 'title':
        return null;
      case 'style':
        return typeof a.precedence == 'string' && typeof a.href == 'string'
          ? ((t = qr(a.href)),
            (a = Ir(r).hoistableStyles),
            (l = a.get(t)),
            l || ((l = { type: 'style', instance: null, count: 0, state: null }), a.set(t, l)),
            l)
          : { type: 'void', instance: null, count: 0, state: null };
      case 'link':
        if (
          a.rel === 'stylesheet' &&
          typeof a.href == 'string' &&
          typeof a.precedence == 'string'
        ) {
          e = qr(a.href);
          var o = Ir(r).hoistableStyles,
            n = o.get(e);
          if (
            (n ||
              ((r = r.ownerDocument || r),
              (n = {
                type: 'stylesheet',
                instance: null,
                count: 0,
                state: { loading: 0, preload: null },
              }),
              o.set(e, n),
              (o = r.querySelector(Sn(e))) && !o._p && ((n.instance = o), (n.state.loading = 5)),
              qt.has(e) ||
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
                qt.set(e, a),
                o || u0(r, e, a, n.state))),
            t && l === null)
          )
            throw Error(S(528, ''));
          return n;
        }
        if (t && l !== null) throw Error(S(529, ''));
        return null;
      case 'script':
        return (
          (t = a.async),
          (a = a.src),
          typeof a == 'string' && t && typeof t != 'function' && typeof t != 'symbol'
            ? ((t = Zr(a)),
              (a = Ir(r).hoistableScripts),
              (l = a.get(t)),
              l || ((l = { type: 'script', instance: null, count: 0, state: null }), a.set(t, l)),
              l)
            : { type: 'void', instance: null, count: 0, state: null }
        );
      default:
        throw Error(S(444, e));
    }
  }
  function qr(e) {
    return 'href="' + Nt(e) + '"';
  }
  function Sn(e) {
    return 'link[rel="stylesheet"][' + e + ']';
  }
  function _y(e) {
    return ye({}, e, { 'data-precedence': e.precedence, precedence: null });
  }
  function u0(e, t, a, l) {
    e.querySelector('link[rel="preload"][as="style"][' + t + ']')
      ? (l.loading = 1)
      : ((t = e.createElement('link')),
        (l.preload = t),
        t.addEventListener('load', function () {
          return (l.loading |= 1);
        }),
        t.addEventListener('error', function () {
          return (l.loading |= 2);
        }),
        Ye(t, 'link', a),
        Pe(t),
        e.head.appendChild(t));
  }
  function Zr(e) {
    return '[src="' + Nt(e) + '"]';
  }
  function bn(e) {
    return 'script[async]' + e;
  }
  function $p(e, t, a) {
    if ((t.count++, t.instance === null))
      switch (t.type) {
        case 'style':
          var l = e.querySelector('style[data-href~="' + Nt(a.href) + '"]');
          if (l) return (t.instance = l), Pe(l), l;
          var r = ye({}, a, {
            'data-href': a.href,
            'data-precedence': a.precedence,
            href: null,
            precedence: null,
          });
          return (
            (l = (e.ownerDocument || e).createElement('style')),
            Pe(l),
            Ye(l, 'style', r),
            Lu(l, a.precedence, e),
            (t.instance = l)
          );
        case 'stylesheet':
          r = qr(a.href);
          var o = e.querySelector(Sn(r));
          if (o) return (t.state.loading |= 4), (t.instance = o), Pe(o), o;
          (l = _y(a)),
            (r = qt.get(r)) && Uf(l, r),
            (o = (e.ownerDocument || e).createElement('link')),
            Pe(o);
          var n = o;
          return (
            (n._p = new Promise(function (u, i) {
              (n.onload = u), (n.onerror = i);
            })),
            Ye(o, 'link', l),
            (t.state.loading |= 4),
            Lu(o, a.precedence, e),
            (t.instance = o)
          );
        case 'script':
          return (
            (o = Zr(a.src)),
            (r = e.querySelector(bn(o)))
              ? ((t.instance = r), Pe(r), r)
              : ((l = a),
                (r = qt.get(o)) && ((l = ye({}, a)), Hf(l, r)),
                (e = e.ownerDocument || e),
                (r = e.createElement('script')),
                Pe(r),
                Ye(r, 'link', l),
                e.head.appendChild(r),
                (t.instance = r))
          );
        case 'void':
          return null;
        default:
          throw Error(S(443, t.type));
      }
    else
      t.type === 'stylesheet' &&
        (t.state.loading & 4) === 0 &&
        ((l = t.instance), (t.state.loading |= 4), Lu(l, a.precedence, e));
    return t.instance;
  }
  function Lu(e, t, a) {
    for (
      var l = a.querySelectorAll('link[rel="stylesheet"][data-precedence],style[data-precedence]'),
        r = l.length ? l[l.length - 1] : null,
        o = r,
        n = 0;
      n < l.length;
      n++
    ) {
      var u = l[n];
      if (u.dataset.precedence === t) o = u;
      else if (o !== r) break;
    }
    o
      ? o.parentNode.insertBefore(e, o.nextSibling)
      : ((t = a.nodeType === 9 ? a.head : a), t.insertBefore(e, t.firstChild));
  }
  function Uf(e, t) {
    e.crossOrigin == null && (e.crossOrigin = t.crossOrigin),
      e.referrerPolicy == null && (e.referrerPolicy = t.referrerPolicy),
      e.title == null && (e.title = t.title);
  }
  function Hf(e, t) {
    e.crossOrigin == null && (e.crossOrigin = t.crossOrigin),
      e.referrerPolicy == null && (e.referrerPolicy = t.referrerPolicy),
      e.integrity == null && (e.integrity = t.integrity);
  }
  var Su = null;
  function eh(e, t, a) {
    if (Su === null) {
      var l = new Map(),
        r = (Su = new Map());
      r.set(a, l);
    } else (r = Su), (l = r.get(a)), l || ((l = new Map()), r.set(a, l));
    if (l.has(e)) return l;
    for (l.set(e, null), a = a.getElementsByTagName(e), r = 0; r < a.length; r++) {
      var o = a[r];
      if (
        !(o[hn] || o[Ve] || (e === 'link' && o.getAttribute('rel') === 'stylesheet')) &&
        o.namespaceURI !== 'http://www.w3.org/2000/svg'
      ) {
        var n = o.getAttribute(t) || '';
        n = e + n;
        var u = l.get(n);
        u ? u.push(o) : l.set(n, [o]);
      }
    }
    return l;
  }
  function th(e, t, a) {
    (e = e.ownerDocument || e),
      e.head.insertBefore(a, t === 'title' ? e.querySelector('head > title') : null);
  }
  function i0(e, t, a) {
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
  function zy(e) {
    return !(e.type === 'stylesheet' && (e.state.loading & 3) === 0);
  }
  function s0(e, t, a, l) {
    if (
      a.type === 'stylesheet' &&
      (typeof l.media != 'string' || matchMedia(l.media).matches !== !1) &&
      (a.state.loading & 4) === 0
    ) {
      if (a.instance === null) {
        var r = qr(l.href),
          o = t.querySelector(Sn(r));
        if (o) {
          (t = o._p),
            t !== null &&
              typeof t == 'object' &&
              typeof t.then == 'function' &&
              (e.count++, (e = Qu.bind(e)), t.then(e, e)),
            (a.state.loading |= 4),
            (a.instance = o),
            Pe(o);
          return;
        }
        (o = t.ownerDocument || t),
          (l = _y(l)),
          (r = qt.get(r)) && Uf(l, r),
          (o = o.createElement('link')),
          Pe(o);
        var n = o;
        (n._p = new Promise(function (u, i) {
          (n.onload = u), (n.onerror = i);
        })),
          Ye(o, 'link', l),
          (a.instance = o);
      }
      e.stylesheets === null && (e.stylesheets = new Map()),
        e.stylesheets.set(a, t),
        (t = a.state.preload) &&
          (a.state.loading & 3) === 0 &&
          (e.count++,
          (a = Qu.bind(e)),
          t.addEventListener('load', a),
          t.addEventListener('error', a));
    }
  }
  var Ks = 0;
  function d0(e, t) {
    return (
      e.stylesheets && e.count === 0 && bu(e, e.stylesheets),
      0 < e.count || 0 < e.imgCount
        ? function (a) {
            var l = setTimeout(function () {
              if ((e.stylesheets && bu(e, e.stylesheets), e.unsuspend)) {
                var o = e.unsuspend;
                (e.unsuspend = null), o();
              }
            }, 6e4 + t);
            0 < e.imgBytes && Ks === 0 && (Ks = 62500 * Vb());
            var r = setTimeout(
              function () {
                if (
                  ((e.waitingForImages = !1),
                  e.count === 0 && (e.stylesheets && bu(e, e.stylesheets), e.unsuspend))
                ) {
                  var o = e.unsuspend;
                  (e.unsuspend = null), o();
                }
              },
              (e.imgBytes > Ks ? 50 : 800) + t
            );
            return (
              (e.unsuspend = a),
              function () {
                (e.unsuspend = null), clearTimeout(l), clearTimeout(r);
              }
            );
          }
        : null
    );
  }
  function Qu() {
    if ((this.count--, this.count === 0 && (this.imgCount === 0 || !this.waitingForImages))) {
      if (this.stylesheets) bu(this, this.stylesheets);
      else if (this.unsuspend) {
        var e = this.unsuspend;
        (this.unsuspend = null), e();
      }
    }
  }
  var Zu = null;
  function bu(e, t) {
    (e.stylesheets = null),
      e.unsuspend !== null &&
        (e.count++, (Zu = new Map()), t.forEach(f0, e), (Zu = null), Qu.call(e));
  }
  function f0(e, t) {
    if (!(t.state.loading & 4)) {
      var a = Zu.get(e);
      if (a) var l = a.get(null);
      else {
        (a = new Map()), Zu.set(e, a);
        for (
          var r = e.querySelectorAll('link[data-precedence],style[data-precedence]'), o = 0;
          o < r.length;
          o++
        ) {
          var n = r[o];
          (n.nodeName === 'LINK' || n.getAttribute('media') !== 'not all') &&
            (a.set(n.dataset.precedence, n), (l = n));
        }
        l && a.set(null, l);
      }
      (r = t.instance),
        (n = r.getAttribute('data-precedence')),
        (o = a.get(n) || l),
        o === l && a.set(null, r),
        a.set(n, r),
        this.count++,
        (l = Qu.bind(this)),
        r.addEventListener('load', l),
        r.addEventListener('error', l),
        o
          ? o.parentNode.insertBefore(r, o.nextSibling)
          : ((e = e.nodeType === 9 ? e.head : e), e.insertBefore(r, e.firstChild)),
        (t.state.loading |= 4);
    }
  }
  var un = {
    $$typeof: Ra,
    Provider: null,
    Consumer: null,
    _currentValue: Al,
    _currentValue2: Al,
    _threadCount: 0,
  };
  function c0(e, t, a, l, r, o, n, u, i) {
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
      (this.expirationTimes = Ls(-1)),
      (this.entangledLanes =
        this.shellSuspendCounter =
        this.errorRecoveryDisabledLanes =
        this.expiredLanes =
        this.warmLanes =
        this.pingedLanes =
        this.suspendedLanes =
        this.pendingLanes =
          0),
      (this.entanglements = Ls(0)),
      (this.hiddenUpdates = Ls(null)),
      (this.identifierPrefix = l),
      (this.onUncaughtError = r),
      (this.onCaughtError = o),
      (this.onRecoverableError = n),
      (this.pooledCache = null),
      (this.pooledCacheLanes = 0),
      (this.formState = i),
      (this.incompleteTransitions = new Map());
  }
  function Fy(e, t, a, l, r, o, n, u, i, s, f, d) {
    return (
      (e = new c0(e, t, a, n, i, s, f, d, u)),
      (t = 1),
      o === !0 && (t |= 24),
      (o = St(3, null, null, t)),
      (e.current = o),
      (o.stateNode = e),
      (t = uf()),
      t.refCount++,
      (e.pooledCache = t),
      t.refCount++,
      (o.memoizedState = { element: l, isDehydrated: a, cache: t }),
      ff(o),
      e
    );
  }
  function qy(e) {
    return e ? ((e = br), e) : br;
  }
  function Gy(e, t, a, l, r, o) {
    (r = qy(r)),
      l.context === null ? (l.context = r) : (l.pendingContext = r),
      (l = ol(t)),
      (l.payload = { element: a }),
      (o = o === void 0 ? null : o),
      o !== null && (l.callback = o),
      (a = nl(e, l, t)),
      a !== null && (ft(a, e, t), Fo(a, e, t));
  }
  function ah(e, t) {
    if (((e = e.memoizedState), e !== null && e.dehydrated !== null)) {
      var a = e.retryLane;
      e.retryLane = a !== 0 && a < t ? a : t;
    }
  }
  function Nf(e, t) {
    ah(e, t), (e = e.alternate) && ah(e, t);
  }
  function Vy(e) {
    if (e.tag === 13 || e.tag === 31) {
      var t = Fl(e, 67108864);
      t !== null && ft(t, e, 67108864), Nf(e, 67108864);
    }
  }
  function lh(e) {
    if (e.tag === 13 || e.tag === 31) {
      var t = It();
      t = Yd(t);
      var a = Fl(e, t);
      a !== null && ft(a, e, t), Nf(e, t);
    }
  }
  var Wu = !0;
  function m0(e, t, a, l) {
    var r = N.T;
    N.T = null;
    var o = le.p;
    try {
      (le.p = 2), Pf(e, t, a, l);
    } finally {
      (le.p = o), (N.T = r);
    }
  }
  function p0(e, t, a, l) {
    var r = N.T;
    N.T = null;
    var o = le.p;
    try {
      (le.p = 8), Pf(e, t, a, l);
    } finally {
      (le.p = o), (N.T = r);
    }
  }
  function Pf(e, t, a, l) {
    if (Wu) {
      var r = qd(l);
      if (r === null) Xs(e, t, l, Ju, a), rh(e, l);
      else if (g0(r, e, t, a, l)) l.stopPropagation();
      else if ((rh(e, l), t & 4 && -1 < h0.indexOf(e))) {
        for (; r !== null; ) {
          var o = jr(r);
          if (o !== null)
            switch (o.tag) {
              case 3:
                if (((o = o.stateNode), o.current.memoizedState.isDehydrated)) {
                  var n = Rl(o.pendingLanes);
                  if (n !== 0) {
                    var u = o;
                    for (u.pendingLanes |= 2, u.entangledLanes |= 2; n; ) {
                      var i = 1 << (31 - Rt(n));
                      (u.entanglements[1] |= i), (n &= ~i);
                    }
                    sa(o), (ae & 6) === 0 && ((Fu = Ct() + 500), Ln(0, !1));
                  }
                }
                break;
              case 31:
              case 13:
                (u = Fl(o, 2)), u !== null && ft(u, o, 2), fi(), Nf(o, 2);
            }
          if (((o = qd(l)), o === null && Xs(e, t, l, Ju, a), o === r)) break;
          r = o;
        }
        r !== null && l.stopPropagation();
      } else Xs(e, t, l, null, a);
    }
  }
  function qd(e) {
    return (e = Wd(e)), _f(e);
  }
  var Ju = null;
  function _f(e) {
    if (((Ju = null), (e = gr(e)), e !== null)) {
      var t = fn(e);
      if (t === null) e = null;
      else {
        var a = t.tag;
        if (a === 13) {
          if (((e = dh(t)), e !== null)) return e;
          e = null;
        } else if (a === 31) {
          if (((e = fh(t)), e !== null)) return e;
          e = null;
        } else if (a === 3) {
          if (t.stateNode.current.memoizedState.isDehydrated)
            return t.tag === 3 ? t.stateNode.containerInfo : null;
          e = null;
        } else t !== e && (e = null);
      }
    }
    return (Ju = e), null;
  }
  function jy(e) {
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
        switch (aS()) {
          case hh:
            return 2;
          case gh:
            return 8;
          case Eu:
          case lS:
            return 32;
          case yh:
            return 268435456;
          default:
            return 32;
        }
      default:
        return 32;
    }
  }
  var Gd = !1,
    sl = null,
    dl = null,
    fl = null,
    sn = new Map(),
    dn = new Map(),
    Wa = [],
    h0 =
      'mousedown mouseup touchcancel touchend touchstart auxclick dblclick pointercancel pointerdown pointerup dragend dragstart drop compositionend compositionstart keydown keypress keyup input textInput copy cut paste click change contextmenu reset'.split(
        ' '
      );
  function rh(e, t) {
    switch (e) {
      case 'focusin':
      case 'focusout':
        sl = null;
        break;
      case 'dragenter':
      case 'dragleave':
        dl = null;
        break;
      case 'mouseover':
      case 'mouseout':
        fl = null;
        break;
      case 'pointerover':
      case 'pointerout':
        sn.delete(t.pointerId);
        break;
      case 'gotpointercapture':
      case 'lostpointercapture':
        dn.delete(t.pointerId);
    }
  }
  function Mo(e, t, a, l, r, o) {
    return e === null || e.nativeEvent !== o
      ? ((e = {
          blockedOn: t,
          domEventName: a,
          eventSystemFlags: l,
          nativeEvent: o,
          targetContainers: [r],
        }),
        t !== null && ((t = jr(t)), t !== null && Vy(t)),
        e)
      : ((e.eventSystemFlags |= l),
        (t = e.targetContainers),
        r !== null && t.indexOf(r) === -1 && t.push(r),
        e);
  }
  function g0(e, t, a, l, r) {
    switch (t) {
      case 'focusin':
        return (sl = Mo(sl, e, t, a, l, r)), !0;
      case 'dragenter':
        return (dl = Mo(dl, e, t, a, l, r)), !0;
      case 'mouseover':
        return (fl = Mo(fl, e, t, a, l, r)), !0;
      case 'pointerover':
        var o = r.pointerId;
        return sn.set(o, Mo(sn.get(o) || null, e, t, a, l, r)), !0;
      case 'gotpointercapture':
        return (o = r.pointerId), dn.set(o, Mo(dn.get(o) || null, e, t, a, l, r)), !0;
    }
    return !1;
  }
  function Xy(e) {
    var t = gr(e.target);
    if (t !== null) {
      var a = fn(t);
      if (a !== null) {
        if (((t = a.tag), t === 13)) {
          if (((t = dh(a)), t !== null)) {
            (e.blockedOn = t),
              Fm(e.priority, function () {
                lh(a);
              });
            return;
          }
        } else if (t === 31) {
          if (((t = fh(a)), t !== null)) {
            (e.blockedOn = t),
              Fm(e.priority, function () {
                lh(a);
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
  function Cu(e) {
    if (e.blockedOn !== null) return !1;
    for (var t = e.targetContainers; 0 < t.length; ) {
      var a = qd(e.nativeEvent);
      if (a === null) {
        a = e.nativeEvent;
        var l = new a.constructor(a.type, a);
        (ud = l), a.target.dispatchEvent(l), (ud = null);
      } else return (t = jr(a)), t !== null && Vy(t), (e.blockedOn = a), !1;
      t.shift();
    }
    return !0;
  }
  function oh(e, t, a) {
    Cu(e) && a.delete(t);
  }
  function y0() {
    (Gd = !1),
      sl !== null && Cu(sl) && (sl = null),
      dl !== null && Cu(dl) && (dl = null),
      fl !== null && Cu(fl) && (fl = null),
      sn.forEach(oh),
      dn.forEach(oh);
  }
  function uu(e, t) {
    e.blockedOn === t &&
      ((e.blockedOn = null),
      Gd || ((Gd = !0), ke.unstable_scheduleCallback(ke.unstable_NormalPriority, y0)));
  }
  var iu = null;
  function nh(e) {
    iu !== e &&
      ((iu = e),
      ke.unstable_scheduleCallback(ke.unstable_NormalPriority, function () {
        iu === e && (iu = null);
        for (var t = 0; t < e.length; t += 3) {
          var a = e[t],
            l = e[t + 1],
            r = e[t + 2];
          if (typeof l != 'function') {
            if (_f(l || a) === null) continue;
            break;
          }
          var o = jr(a);
          o !== null &&
            (e.splice(t, 3),
            (t -= 3),
            Cd(o, { pending: !0, data: r, method: a.method, action: l }, l, r));
        }
      }));
  }
  function Gr(e) {
    function t(i) {
      return uu(i, e);
    }
    sl !== null && uu(sl, e),
      dl !== null && uu(dl, e),
      fl !== null && uu(fl, e),
      sn.forEach(t),
      dn.forEach(t);
    for (var a = 0; a < Wa.length; a++) {
      var l = Wa[a];
      l.blockedOn === e && (l.blockedOn = null);
    }
    for (; 0 < Wa.length && ((a = Wa[0]), a.blockedOn === null); )
      Xy(a), a.blockedOn === null && Wa.shift();
    if (((a = (e.ownerDocument || e).$$reactFormReplay), a != null))
      for (l = 0; l < a.length; l += 3) {
        var r = a[l],
          o = a[l + 1],
          n = r[ct] || null;
        if (typeof o == 'function') n || nh(a);
        else if (n) {
          var u = null;
          if (o && o.hasAttribute('formAction')) {
            if (((r = o), (n = o[ct] || null))) u = n.formAction;
            else if (_f(r) !== null) continue;
          } else u = n.action;
          typeof u == 'function' ? (a[l + 1] = u) : (a.splice(l, 3), (l -= 3)), nh(a);
        }
      }
  }
  function Yy() {
    function e(o) {
      o.canIntercept &&
        o.info === 'react-transition' &&
        o.intercept({
          handler: function () {
            return new Promise(function (n) {
              return (r = n);
            });
          },
          focusReset: 'manual',
          scroll: 'manual',
        });
    }
    function t() {
      r !== null && (r(), (r = null)), l || setTimeout(a, 20);
    }
    function a() {
      if (!l && !navigation.transition) {
        var o = navigation.currentEntry;
        o &&
          o.url != null &&
          navigation.navigate(o.url, {
            state: o.getState(),
            info: 'react-transition',
            history: 'replace',
          });
      }
    }
    if (typeof navigation == 'object') {
      var l = !1,
        r = null;
      return (
        navigation.addEventListener('navigate', e),
        navigation.addEventListener('navigatesuccess', t),
        navigation.addEventListener('navigateerror', t),
        setTimeout(a, 100),
        function () {
          (l = !0),
            navigation.removeEventListener('navigate', e),
            navigation.removeEventListener('navigatesuccess', t),
            navigation.removeEventListener('navigateerror', t),
            r !== null && (r(), (r = null));
        }
      );
    }
  }
  function zf(e) {
    this._internalRoot = e;
  }
  pi.prototype.render = zf.prototype.render = function (e) {
    var t = this._internalRoot;
    if (t === null) throw Error(S(409));
    var a = t.current,
      l = It();
    Gy(a, l, e, t, null, null);
  };
  pi.prototype.unmount = zf.prototype.unmount = function () {
    var e = this._internalRoot;
    if (e !== null) {
      this._internalRoot = null;
      var t = e.containerInfo;
      Gy(e.current, 2, null, e, null, null), fi(), (t[Vr] = null);
    }
  };
  function pi(e) {
    this._internalRoot = e;
  }
  pi.prototype.unstable_scheduleHydration = function (e) {
    if (e) {
      var t = bh();
      e = { blockedOn: null, target: e, priority: t };
      for (var a = 0; a < Wa.length && t !== 0 && t < Wa[a].priority; a++);
      Wa.splice(a, 0, e), a === 0 && Xy(e);
    }
  };
  var uh = ih.version;
  if (uh !== '19.2.8') throw Error(S(527, uh, '19.2.8'));
  le.findDOMNode = function (e) {
    var t = e._reactInternals;
    if (t === void 0)
      throw typeof e.render == 'function'
        ? Error(S(188))
        : ((e = Object.keys(e).join(',')), Error(S(268, e)));
    return (e = QL(t)), (e = e !== null ? ch(e) : null), (e = e === null ? null : e.stateNode), e;
  };
  var x0 = {
    bundleType: 0,
    version: '19.2.8',
    rendererPackageName: 'react-dom',
    currentDispatcherRef: N,
    reconcilerVersion: '19.2.8',
  };
  if (
    typeof __REACT_DEVTOOLS_GLOBAL_HOOK__ < 'u' &&
    ((Do = __REACT_DEVTOOLS_GLOBAL_HOOK__), !Do.isDisabled && Do.supportsFiber)
  )
    try {
      (cn = Do.inject(x0)), (wt = Do);
    } catch {}
  var Do;
  hi.createRoot = function (e, t) {
    if (!sh(e)) throw Error(S(299));
    var a = !1,
      l = '',
      r = Pg,
      o = _g,
      n = zg;
    return (
      t != null &&
        (t.unstable_strictMode === !0 && (a = !0),
        t.identifierPrefix !== void 0 && (l = t.identifierPrefix),
        t.onUncaughtError !== void 0 && (r = t.onUncaughtError),
        t.onCaughtError !== void 0 && (o = t.onCaughtError),
        t.onRecoverableError !== void 0 && (n = t.onRecoverableError)),
      (t = Fy(e, 1, !1, null, null, a, l, null, r, o, n, Yy)),
      (e[Vr] = t.current),
      Of(e),
      new zf(t)
    );
  };
  hi.hydrateRoot = function (e, t, a) {
    if (!sh(e)) throw Error(S(299));
    var l = !1,
      r = '',
      o = Pg,
      n = _g,
      u = zg,
      i = null;
    return (
      a != null &&
        (a.unstable_strictMode === !0 && (l = !0),
        a.identifierPrefix !== void 0 && (r = a.identifierPrefix),
        a.onUncaughtError !== void 0 && (o = a.onUncaughtError),
        a.onCaughtError !== void 0 && (n = a.onCaughtError),
        a.onRecoverableError !== void 0 && (u = a.onRecoverableError),
        a.formState !== void 0 && (i = a.formState)),
      (t = Fy(e, 1, !0, t, a ?? null, l, r, i, o, n, u, Yy)),
      (t.context = qy(null)),
      (a = t.current),
      (l = It()),
      (l = Yd(l)),
      (r = ol(l)),
      (r.callback = null),
      nl(a, r, l),
      (a = l),
      (t.current.lanes = a),
      pn(t, a),
      sa(t),
      (e[Vr] = t.current),
      Of(e),
      new pi(t)
    );
  };
  hi.version = '19.2.8';
});
var v0 = aa((PR, Zy) => {
  'use strict';
  function Qy() {
    if (
      !(
        typeof __REACT_DEVTOOLS_GLOBAL_HOOK__ > 'u' ||
        typeof __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE != 'function'
      )
    )
      try {
        __REACT_DEVTOOLS_GLOBAL_HOOK__.checkDCE(Qy);
      } catch (e) {
        console.error(e);
      }
  }
  Qy(), (Zy.exports = Ky());
});
var Kx = aa((Ii) => {
  'use strict';
  var aw = Symbol.for('react.transitional.element'),
    lw = Symbol.for('react.fragment');
  function Yx(e, t, a) {
    var l = null;
    if ((a !== void 0 && (l = '' + a), t.key !== void 0 && (l = '' + t.key), 'key' in t)) {
      a = {};
      for (var r in t) r !== 'key' && (a[r] = t[r]);
    } else a = t;
    return (t = a.ref), { $$typeof: aw, type: e, key: l, ref: t !== void 0 ? t : null, props: a };
  }
  Ii.Fragment = lw;
  Ii.jsx = Yx;
  Ii.jsxs = Yx;
});
var Q = aa((iE, Qx) => {
  'use strict';
  Qx.exports = Kx();
});
var ht = E(te(), 1),
  D = E(te(), 1),
  de = E(te(), 1),
  uc = E(te(), 1),
  Ox = E(te(), 1),
  J = E(te(), 1),
  zC = E(te(), 1),
  FC = E(te(), 1),
  qC = E(te(), 1),
  k = E(te(), 1),
  Xx = E(te(), 1);
var Qf = /^(?:[a-z][a-z0-9+.-]*:|[\\/]{2})/i,
  nx = /^[\\/]{2}/;
function L0(e, t) {
  return t + e.replace(/\\/g, '/');
}
var Wy = 'popstate';
function Jy(e) {
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
function ux(e = {}) {
  function t(l, r) {
    let o = r.state?.masked,
      { pathname: n, search: u, hash: i } = o || l.location;
    return jf(
      '',
      { pathname: n, search: u, hash: i },
      (r.state && r.state.usr) || null,
      (r.state && r.state.key) || 'default',
      o
        ? { pathname: l.location.pathname, search: l.location.search, hash: l.location.hash }
        : void 0
    );
  }
  function a(l, r) {
    return typeof r == 'string' ? r : xl(r);
  }
  return b0(t, a, null, e);
}
function me(e, t) {
  if (e === !1 || e === null || typeof e > 'u') throw new Error(t);
}
function pt(e, t) {
  if (!e) {
    typeof console < 'u' && console.warn(t);
    try {
      throw new Error(t);
    } catch {}
  }
}
function S0() {
  return Math.random().toString(36).substring(2, 10);
}
function $y(e, t) {
  return {
    usr: e.state,
    key: e.key,
    idx: t,
    masked: e.mask ? { pathname: e.pathname, search: e.search, hash: e.hash } : void 0,
  };
}
function jf(e, t, a = null, l, r) {
  return {
    pathname: typeof e == 'string' ? e : e.pathname,
    search: '',
    hash: '',
    ...(typeof t == 'string' ? Gl(t) : t),
    state: a,
    key: (t && t.key) || l || S0(),
    mask: r,
  };
}
function xl({ pathname: e = '/', search: t = '', hash: a = '' }) {
  return (
    t && t !== '?' && (e += t.charAt(0) === '?' ? t : '?' + t),
    a && a !== '#' && (e += a.charAt(0) === '#' ? a : '#' + a),
    e
  );
}
function Gl(e) {
  let t = {};
  if (e) {
    let a = e.indexOf('#');
    a >= 0 && ((t.hash = e.substring(a)), (e = e.substring(0, a)));
    let l = e.indexOf('?');
    l >= 0 && ((t.search = e.substring(l)), (e = e.substring(0, l))), e && (t.pathname = e);
  }
  return t;
}
function b0(e, t, a, l = {}) {
  let { window: r = document.defaultView, v5Compat: o = !1 } = l,
    n = r.history,
    u = 'POP',
    i = null,
    s = f();
  s == null && ((s = 0), n.replaceState({ ...n.state, idx: s }, ''));
  function f() {
    return (n.state || { idx: null }).idx;
  }
  function d() {
    u = 'POP';
    let L = f(),
      c = L == null ? null : L - s;
    (s = L), i && i({ action: u, location: v.location, delta: c });
  }
  function p(L, c) {
    u = 'PUSH';
    let m = Jy(L) ? L : jf(v.location, L, c);
    a && a(m, L), (s = f() + 1);
    let g = $y(m, s),
      y = v.createHref(m.mask || m);
    try {
      n.pushState(g, '', y);
    } catch (w) {
      if (w instanceof DOMException && w.name === 'DataCloneError') throw w;
      r.location.assign(y);
    }
    o && i && i({ action: u, location: v.location, delta: 1 });
  }
  function h(L, c) {
    u = 'REPLACE';
    let m = Jy(L) ? L : jf(v.location, L, c);
    a && a(m, L), (s = f());
    let g = $y(m, s),
      y = v.createHref(m.mask || m);
    n.replaceState(g, '', y), o && i && i({ action: u, location: v.location, delta: 0 });
  }
  function x(L) {
    return C0(r, L);
  }
  let v = {
    get action() {
      return u;
    },
    get location() {
      return e(r, n);
    },
    listen(L) {
      if (i) throw new Error('A history only accepts one active listener');
      return (
        r.addEventListener(Wy, d),
        (i = L),
        () => {
          r.removeEventListener(Wy, d), (i = null);
        }
      );
    },
    createHref(L) {
      return t(r, L);
    },
    createURL: x,
    encodeLocation(L) {
      let c = x(L);
      return { pathname: c.pathname, search: c.search, hash: c.hash };
    },
    push: p,
    replace: h,
    go(L) {
      return n.go(L);
    },
  };
  return v;
}
function C0(e, t, a = !1) {
  let l = 'http://localhost';
  e && (l = e.location.origin !== 'null' ? e.location.origin : e.location.href),
    me(l, 'No window.location.(origin|href) available to create URL');
  let r = typeof t == 'string' ? t : xl(t);
  return (r = r.replace(/ $/, '%20')), !a && nx.test(r) && (r = l + r), new URL(r, l);
}
var w0;
w0 = new WeakMap();
function Zf(e, t, a = '/') {
  return R0(e, t, a, !1);
}
function R0(e, t, a, l, r) {
  let o = typeof t == 'string' ? Gl(t) : t,
    n = da(o.pathname || '/', a);
  if (n == null) return null;
  let u = r ?? E0(e),
    i = null,
    s = P0(n);
  for (let f = 0; i == null && f < u.length; ++f) i = N0(u[f], s, l);
  return i;
}
function I0(e, t) {
  let { route: a, pathname: l, params: r } = e;
  return { id: a.id, pathname: l, params: r, data: t[a.id], loaderData: t[a.id], handle: a.handle };
}
function E0(e) {
  let t = ix(e);
  return A0(t), t;
}
function ix(e, t = [], a = [], l = '', r = !1) {
  let o = (n, u, i = r, s) => {
    let f = {
      relativePath: s === void 0 ? n.path || '' : s,
      caseSensitive: n.caseSensitive === !0,
      childrenIndex: u,
      route: n,
    };
    if (f.relativePath.startsWith('/')) {
      if (!f.relativePath.startsWith(l) && i) return;
      me(
        f.relativePath.startsWith(l),
        `Absolute route path "${f.relativePath}" nested under path "${l}" is not valid. An absolute child route path must start with the combined path of all its parent routes.`
      ),
        (f.relativePath = f.relativePath.slice(l.length));
    }
    let d = Qt([l, f.relativePath]),
      p = a.concat(f);
    n.children &&
      n.children.length > 0 &&
      (me(
        n.index !== !0,
        `Index routes must not have child routes. Please remove all child routes from route path "${d}".`
      ),
      ix(n.children, t, p, d, i)),
      !(n.path == null && !n.index) &&
        t.push({
          path: d,
          score: U0(d, n.index),
          routesMeta: p.map((h, x) => {
            let [v, L] = fx(h.relativePath, h.caseSensitive, x === p.length - 1);
            return { ...h, matcher: v, compiledParams: L };
          }),
        });
  };
  return (
    e.forEach((n, u) => {
      if (n.path === '' || !n.path?.includes('?')) o(n, u);
      else for (let i of sx(n.path)) o(n, u, !0, i);
    }),
    t
  );
}
function sx(e) {
  let t = e.split('/');
  if (t.length === 0) return [];
  let [a, ...l] = t,
    r = a.endsWith('?'),
    o = a.replace(/\?$/, '');
  if (l.length === 0) return r ? [o, ''] : [o];
  let n = sx(l.join('/')),
    u = [];
  return (
    u.push(...n.map((i) => (i === '' ? o : [o, i].join('/')))),
    r && u.push(...n),
    u.map((i) => (e.startsWith('/') && i === '' ? '/' : i))
  );
}
function A0(e) {
  e.sort((t, a) =>
    t.score !== a.score
      ? a.score - t.score
      : H0(
          t.routesMeta.map((l) => l.childrenIndex),
          a.routesMeta.map((l) => l.childrenIndex)
        )
  );
}
var T0 = /^:[\w-]+$/,
  M0 = 3,
  D0 = 2,
  k0 = 1,
  B0 = 10,
  O0 = -2,
  ex = (e) => e === '*';
function U0(e, t) {
  let a = e.split('/'),
    l = a.length;
  return (
    a.some(ex) && (l += O0),
    t && (l += D0),
    a.filter((r) => !ex(r)).reduce((r, o) => r + (T0.test(o) ? M0 : o === '' ? k0 : B0), l)
  );
}
function H0(e, t) {
  return e.length === t.length && e.slice(0, -1).every((l, r) => l === t[r])
    ? e[e.length - 1] - t[t.length - 1]
    : 0;
}
function N0(e, t, a = !1) {
  let { routesMeta: l } = e,
    r = {},
    o = '/',
    n = [];
  for (let u = 0; u < l.length; ++u) {
    let i = l[u],
      s = u === l.length - 1,
      f = o === '/' ? t : t.slice(o.length) || '/',
      d = { path: i.relativePath, caseSensitive: i.caseSensitive, end: s },
      p = i.matcher && i.compiledParams ? dx(d, f, i.matcher, i.compiledParams) : wn(d, f),
      h = i.route;
    if (
      (!p &&
        s &&
        a &&
        !l[l.length - 1].route.index &&
        (p = wn({ path: i.relativePath, caseSensitive: i.caseSensitive, end: !1 }, f)),
      !p)
    )
      return null;
    Object.assign(r, p.params),
      n.push({
        params: r,
        pathname: Qt([o, p.pathname]),
        pathnameBase: z0(Qt([o, p.pathnameBase])),
        route: h,
      }),
      p.pathnameBase !== '/' && (o = Qt([o, p.pathnameBase]));
  }
  return n;
}
function wn(e, t) {
  typeof e == 'string' && (e = { path: e, caseSensitive: !1, end: !0 });
  let [a, l] = fx(e.path, e.caseSensitive, e.end);
  return dx(e, t, a, l);
}
function dx(e, t, a, l) {
  let r = t.match(a);
  if (!r) return null;
  let o = r[0],
    n = Wr(o, 1),
    u = r.slice(1);
  return {
    params: l.reduce((s, { paramName: f, isOptional: d }, p) => {
      if (f === '*') {
        let x = u[p] || '';
        n = Wr(o.slice(0, o.length - x.length), 1);
      }
      let h = u[p];
      return d && !h ? (s[f] = void 0) : (s[f] = (h || '').replace(/%2F/g, '/')), s;
    }, {}),
    pathname: o,
    pathnameBase: n,
    pattern: e,
  };
}
function fx(e, t = !1, a = !0) {
  pt(
    e === '*' || !e.endsWith('*') || e.endsWith('/*'),
    `Route path "${e}" will be treated as if it were "${e.replace(/\*$/, '/*')}" because the \`*\` character must always follow a \`/\` in the pattern. To get rid of this warning, please change the route path to "${e.replace(/\*$/, '/*')}".`
  );
  let l = [],
    r =
      '^' +
      e
        .replace(/\/*\*?$/, '')
        .replace(/^\/*/, '/')
        .replace(/[\\.*+^${}|()[\]]/g, '\\$&')
        .replace(/\/:([\w-]+)(\?)?/g, (n, u, i, s, f) => {
          if ((l.push({ paramName: u, isOptional: i != null }), i)) {
            let d = f.charAt(s + n.length);
            return d && d !== '/' ? '/([^\\/]*)' : '(?:/([^\\/]*))?';
          }
          return '/([^\\/]+)';
        })
        .replace(/\/([\w-]+)\?(\/|$)/g, '(/$1)?$2');
  return (
    e.endsWith('*')
      ? (l.push({ paramName: '*' }), (r += e === '*' || e === '/*' ? '(.*)$' : '(?:\\/(.+)|\\/*)$'))
      : a
        ? (r += '\\/*$')
        : e !== '' && e !== '/' && (r += '(?:(?=\\/|$))'),
    [new RegExp(r, t ? void 0 : 'i'), l]
  );
}
function P0(e) {
  try {
    return e
      .split('/')
      .map((t) => decodeURIComponent(t).replace(/\//g, '%2F'))
      .join('/');
  } catch (t) {
    return (
      pt(
        !1,
        `The URL path "${e}" could not be decoded because it is a malformed URL segment. This is probably due to a bad percent encoding (${t}).`
      ),
      e
    );
  }
}
function da(e, t) {
  if (t === '/') return e;
  if (!e.toLowerCase().startsWith(t.toLowerCase())) return null;
  let a = t.endsWith('/') ? t.length - 1 : t.length,
    l = e.charAt(a);
  return l && l !== '/' ? null : e.slice(a) || '/';
}
function cx(e, t = '/') {
  let { pathname: a, search: l = '', hash: r = '' } = typeof e == 'string' ? Gl(e) : e,
    o;
  return (
    a
      ? ((a = mx(a)),
        a.startsWith('/') || a.startsWith('\\') ? (o = tx(a.substring(1), '/')) : (o = tx(a, t)))
      : (o = t),
    { pathname: o, search: F0(l), hash: q0(r) }
  );
}
function tx(e, t) {
  let a = Wr(t).split('/');
  return (
    e.split('/').forEach((r) => {
      r === '..' ? a.length > 1 && a.pop() : r !== '.' && a.push(r);
    }),
    a.length > 1 ? a.join('/') : '/'
  );
}
function Ff(e, t, a, l) {
  return `Cannot include a '${e}' character in a manually specified \`to.${t}\` field [${JSON.stringify(l)}].  Please separate it out to the \`to.${a}\` field. Alternatively you may provide the full path as a string in <Link to="..."> and the router will parse it for you.`;
}
function _0(e) {
  return e.filter((t, a) => a === 0 || (t.route.path && t.route.path.length > 0));
}
function Wf(e) {
  let t = _0(e);
  return t.map((a, l) => (l === t.length - 1 ? a.pathname : a.pathnameBase));
}
function bi(e, t, a, l = !1) {
  let r;
  typeof e == 'string'
    ? (r = Gl(e))
    : ((r = { ...e }),
      me(!r.pathname || !r.pathname.includes('?'), Ff('?', 'pathname', 'search', r)),
      me(!r.pathname || !r.pathname.includes('#'), Ff('#', 'pathname', 'hash', r)),
      me(!r.search || !r.search.includes('#'), Ff('#', 'search', 'hash', r)));
  let o = e === '' || r.pathname === '',
    n = o ? '/' : r.pathname,
    u;
  if (n == null) u = a;
  else {
    let d = t.length - 1;
    if (!l && n.startsWith('..')) {
      let p = n.split('/');
      for (; p[0] === '..'; ) p.shift(), (d -= 1);
      r.pathname = p.join('/');
    }
    u = d >= 0 ? t[d] : '/';
  }
  let i = cx(r, u),
    s = n && n !== '/' && n.endsWith('/'),
    f = (o || n === '.') && a.endsWith('/');
  return !i.pathname.endsWith('/') && (s || f) && (i.pathname += '/'), i;
}
var mx = (e) => e.replace(/[\\/]{2,}/g, '/'),
  Qt = (e) => mx(e.join('/'));
function Wr(e, t = 0) {
  let a = e.length;
  for (; a > t && e.charCodeAt(a - 1) === 47; ) a--;
  return a === e.length ? e : e.slice(0, a);
}
var z0 = (e) => Wr(e).replace(/^\/*/, '/'),
  F0 = (e) => (!e || e === '?' ? '' : e.startsWith('?') ? e : '?' + e),
  q0 = (e) => (!e || e === '#' ? '' : e.startsWith('#') ? e : '#' + e);
var px = class {
  constructor(e, t, a, l = !1) {
    (this.status = e),
      (this.statusText = t || ''),
      (this.internal = l),
      a instanceof Error ? ((this.data = a.toString()), (this.error = a)) : (this.data = a);
  }
};
function hx(e) {
  return (
    e != null &&
    typeof e.status == 'number' &&
    typeof e.statusText == 'string' &&
    typeof e.internal == 'boolean' &&
    'data' in e
  );
}
function G0(e) {
  let t = e.map((a) => a.route.path).filter(Boolean);
  return Qt(t) || '/';
}
var gx =
  typeof window < 'u' && typeof window.document < 'u' && typeof window.document.createElement < 'u';
function yx(e, t) {
  let a = e;
  if (typeof a != 'string' || !Qf.test(a)) return { absoluteURL: void 0, isExternal: !1, to: a };
  let l = a,
    r = !1;
  if (gx)
    try {
      let o = new URL(window.location.href),
        n = nx.test(a) ? new URL(L0(a, o.protocol)) : new URL(a),
        u = da(n.pathname, t);
      n.origin === o.origin && u != null ? (a = u + n.search + n.hash) : (r = !0);
    } catch {
      pt(
        !1,
        `<Link to="${a}"> contains an invalid URL which will probably break when clicked - please update to a valid URL path.`
      );
    }
  return { absoluteURL: l, isExternal: r, to: a };
}
var _R = Symbol('Uninstrumented');
var zR = Object.getOwnPropertyNames(Object.prototype).sort().join('\0');
var ax = new URL('http://localhost');
function Jf(e) {
  if (e.createURL) return e.createURL('/');
  try {
    return new URL(e.createHref('/'), ax);
  } catch {
    return ax;
  }
}
function qf(e, t) {
  return (
    e.origin === t.origin &&
    (e.origin !== 'null' || (e.protocol === t.protocol && e.host === t.host))
  );
}
function V0(e, t) {
  if (e.startsWith('//')) return !0;
  let a = t.protocol.toLowerCase();
  return e.toLowerCase().startsWith(a) ? t.host === '' || e.slice(a.length).startsWith('//') : !1;
}
function $f(e, t, a, l) {
  let r = null;
  try {
    r = e == null ? null : new URL(e, a);
  } catch {}
  let o = new URL(t, a),
    n = r != null && !qf(r, a),
    u = !qf(o, a);
  if (l === 'reject') {
    if (n || u) throw new Error('External navigation is not allowed');
  } else if (u && (r == null || !V0(e, r) || !qf(r, o)))
    throw new Error('External navigation is not allowed');
}
var xx = ['POST', 'PUT', 'PATCH', 'DELETE'],
  FR = new Set(xx),
  j0 = ['GET', ...xx],
  qR = new Set(j0);
var GR = Symbol('ResetLoaderData'),
  X0,
  Y0,
  K0,
  Q0;
X0 = new WeakMap();
Y0 = new WeakMap();
K0 = new WeakMap();
Q0 = new WeakMap();
var Z0 = [
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
function W0(e) {
  try {
    return Z0.includes(new URL(e).protocol);
  } catch {
    return !1;
  }
}
var Vl = ht.createContext(null);
Vl.displayName = 'DataRouter';
var Jr = ht.createContext(null);
Jr.displayName = 'DataRouterState';
var vx = ht.createContext(!1);
function J0() {
  return ht.useContext(vx);
}
var ec = ht.createContext({ isTransitioning: !1 });
ec.displayName = 'ViewTransition';
var Lx = ht.createContext(new Map());
Lx.displayName = 'Fetchers';
var $0 = ht.createContext(null);
$0.displayName = 'Await';
var Je = ht.createContext(null);
Je.displayName = 'Navigation';
var $r = ht.createContext(null);
$r.displayName = 'Location';
var Tt = ht.createContext({ outlet: null, matches: [], isDataRoute: !1 });
Tt.displayName = 'Route';
var tc = ht.createContext(null);
tc.displayName = 'RouteError';
var Xf = !0,
  Sx = 'REACT_ROUTER_ERROR',
  eC = 'REDIRECT',
  tC = 'ROUTE_ERROR_RESPONSE';
function aC(e) {
  if (e.startsWith(`${Sx}:${eC}:{`))
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
function lC(e) {
  if (e.startsWith(`${Sx}:${tC}:{`))
    try {
      let t = JSON.parse(e.slice(40));
      if (
        typeof t == 'object' &&
        t &&
        typeof t.status == 'number' &&
        typeof t.statusText == 'string'
      )
        return new px(t.status, t.statusText, t.data);
    } catch {}
}
function bx(e, { relative: t } = {}) {
  me(jl(), 'useHref() may be used only in the context of a <Router> component.');
  let { basename: a, navigator: l } = D.useContext(Je),
    { hash: r, pathname: o, search: n } = eo(e, { relative: t }),
    u = o;
  return (
    a !== '/' && (u = o === '/' ? a : Qt([a, o])), l.createHref({ pathname: u, search: n, hash: r })
  );
}
function jl() {
  return D.useContext($r) != null;
}
function gt() {
  return (
    me(jl(), 'useLocation() may be used only in the context of a <Router> component.'),
    D.useContext($r).location
  );
}
var Cx =
  'You should call navigate() in a React.useEffect(), not when your component is first rendered.';
function wx(e) {
  D.useContext(Je).static || D.useLayoutEffect(e);
}
function Ci() {
  let { isDataRoute: e } = D.useContext(Tt);
  return e ? hC() : rC();
}
function rC() {
  me(jl(), 'useNavigate() may be used only in the context of a <Router> component.');
  let e = D.useContext(Vl),
    { basename: t, navigator: a } = D.useContext(Je),
    { matches: l } = D.useContext(Tt),
    { pathname: r } = gt(),
    o = JSON.stringify(Wf(l)),
    n = D.useRef(!1);
  return (
    wx(() => {
      n.current = !0;
    }),
    D.useCallback(
      (i, s = {}) => {
        if ((pt(n.current, Cx), !n.current)) return;
        if (typeof i == 'number') {
          a.go(i);
          return;
        }
        let f = bi(i, JSON.parse(o), r, s.relative === 'path');
        e == null && t !== '/' && (f.pathname = f.pathname === '/' ? t : Qt([t, f.pathname])),
          $f(typeof i == 'string' ? i : xl(i), a.createHref(f), Jf(a), 'reject'),
          (s.replace ? a.replace : a.push)(f, s.state, s);
      },
      [t, a, o, r, e]
    )
  );
}
var oC = D.createContext(null);
function Rx(e) {
  let t = D.useContext(Tt).outlet;
  return D.useMemo(() => t && D.createElement(oC.Provider, { value: e }, t), [t, e]);
}
function nC() {
  let { matches: e } = D.useContext(Tt);
  return e[e.length - 1]?.params ?? {};
}
function eo(e, { relative: t } = {}) {
  let { matches: a } = D.useContext(Tt),
    { pathname: l } = gt(),
    r = JSON.stringify(Wf(a));
  return D.useMemo(() => bi(e, JSON.parse(r), l, t === 'path'), [e, r, l, t]);
}
function Ix(e, t) {
  return Ex(e, t);
}
function Ex(e, t, a) {
  me(jl(), 'useRoutes() may be used only in the context of a <Router> component.');
  let { navigator: l } = D.useContext(Je),
    { matches: r } = D.useContext(Tt),
    o = r[r.length - 1],
    n = o ? o.params : {},
    u = o ? o.pathname : '/',
    i = o ? o.pathnameBase : '/',
    s = o && o.route;
  if (Xf) {
    let L = (s && s.path) || '';
    Dx(
      u,
      !s || L.endsWith('*') || L.endsWith('*?'),
      `You rendered descendant <Routes> (or called \`useRoutes()\`) at "${u}" (under <Route path="${L}">) but the parent route path has no trailing "*". This means if you navigate deeper, the parent won't match anymore and therefore the child routes will never render.

Please change the parent <Route path="${L}"> to <Route path="${L === '/' ? '*' : `${L}/*`}">.`
    );
  }
  let f = gt(),
    d;
  if (t) {
    let L = typeof t == 'string' ? Gl(t) : t;
    me(
      i === '/' || L.pathname?.startsWith(i),
      `When overriding the location using \`<Routes location>\` or \`useRoutes(routes, location)\`, the location pathname must begin with the portion of the URL pathname that was matched by all parent routes. The current pathname base is "${i}" but pathname "${L.pathname}" was given in the \`location\` prop.`
    ),
      (d = L);
  } else d = f;
  let p = d.pathname || '/',
    h = p;
  if (i !== '/') {
    let L = i.replace(/^\//, '').split('/');
    h = '/' + p.replace(/^\//, '').split('/').slice(L.length).join('/');
  }
  let x =
    a && a.state.matches.length
      ? a.state.matches.map((L) => Object.assign(L, { route: a.manifest[L.route.id] || L.route }))
      : Zf(e, { pathname: h });
  Xf &&
    (pt(s || x != null, `No routes matched location "${d.pathname}${d.search}${d.hash}" `),
    pt(
      x == null ||
        x[x.length - 1].route.element !== void 0 ||
        x[x.length - 1].route.Component !== void 0 ||
        x[x.length - 1].route.lazy !== void 0,
      `Matched leaf route at location "${d.pathname}${d.search}${d.hash}" does not have an element or Component. This means it will render an <Outlet /> with a null value by default resulting in an "empty" page.`
    ));
  let v = fC(
    x &&
      x.map((L) =>
        Object.assign({}, L, {
          params: Object.assign({}, n, L.params),
          pathname: Qt([
            i,
            l.encodeLocation
              ? l.encodeLocation(
                  L.pathname.replace(/%/g, '%25').replace(/\?/g, '%3F').replace(/#/g, '%23')
                ).pathname
              : L.pathname,
          ]),
          pathnameBase:
            L.pathnameBase === '/'
              ? i
              : Qt([
                  i,
                  l.encodeLocation
                    ? l.encodeLocation(
                        L.pathnameBase
                          .replace(/%/g, '%25')
                          .replace(/\?/g, '%3F')
                          .replace(/#/g, '%23')
                      ).pathname
                    : L.pathnameBase,
                ]),
        })
      ),
    r,
    a
  );
  return t && v
    ? D.createElement(
        $r.Provider,
        {
          value: {
            location: {
              pathname: '/',
              search: '',
              hash: '',
              state: null,
              key: 'default',
              mask: void 0,
              ...d,
            },
            navigationType: 'POP',
          },
        },
        v
      )
    : v;
}
function uC() {
  let e = Mx(),
    t = hx(e) ? `${e.status} ${e.statusText}` : e instanceof Error ? e.message : JSON.stringify(e),
    a = e instanceof Error ? e.stack : null,
    l = 'rgba(200,200,200, 0.5)',
    r = { padding: '0.5rem', backgroundColor: l },
    o = { padding: '2px 4px', backgroundColor: l },
    n = null;
  return (
    Xf &&
      (console.error('Error handled by React Router default ErrorBoundary:', e),
      (n = D.createElement(
        D.Fragment,
        null,
        D.createElement('p', null, '\u{1F4BF} Hey developer \u{1F44B}'),
        D.createElement(
          'p',
          null,
          'You can provide a way better UX than this when your app throws errors by providing your own ',
          D.createElement('code', { style: o }, 'ErrorBoundary'),
          ' or',
          ' ',
          D.createElement('code', { style: o }, 'errorElement'),
          ' prop on your route.'
        )
      ))),
    D.createElement(
      D.Fragment,
      null,
      D.createElement('h2', null, 'Unexpected Application Error!'),
      D.createElement('h3', { style: { fontStyle: 'italic' } }, t),
      a ? D.createElement('pre', { style: r }, a) : null,
      n
    )
  );
}
var iC = D.createElement(uC, null),
  Ax = class extends D.Component {
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
        let a = lC(e.digest);
        a && (e = a);
      }
      let t =
        e !== void 0
          ? D.createElement(
              Tt.Provider,
              { value: this.props.routeContext },
              D.createElement(tc.Provider, { value: e, children: this.props.component })
            )
          : this.props.children;
      return this.context ? D.createElement(sC, { error: e }, t) : t;
    }
  };
Ax.contextType = vx;
var Gf = new WeakMap();
function sC({ children: e, error: t }) {
  let { basename: a, navigator: l } = D.useContext(Je);
  if (typeof t == 'object' && t && 'digest' in t && typeof t.digest == 'string') {
    let r = aC(t.digest);
    if (r) {
      let o = Gf.get(t);
      if (o) throw o;
      let n = yx(r.location, a),
        u = n.absoluteURL || n.to;
      if (($f(r.location, u, Jf(l), 'allow-explicit'), W0(u)))
        throw new Error('Invalid redirect location');
      if (gx && !Gf.get(t))
        if (n.isExternal || r.reloadDocument) window.location.href = u;
        else {
          let i = Promise.resolve().then(() =>
            window.__reactRouterDataRouter.navigate(n.to, { replace: r.replace })
          );
          throw (Gf.set(t, i), i);
        }
      return D.createElement('meta', { httpEquiv: 'refresh', content: `0;url=${u}` });
    }
  }
  return e;
}
function dC({ routeContext: e, match: t, children: a }) {
  let l = D.useContext(Vl);
  return (
    l &&
      l.static &&
      l.staticContext &&
      (t.route.errorElement || t.route.ErrorBoundary) &&
      (l.staticContext._deepestRenderedBoundaryId = t.route.id),
    D.createElement(Tt.Provider, { value: e }, a)
  );
}
function fC(e, t = [], a) {
  let l = a?.state;
  if (e == null) {
    if (!l) return null;
    if (l.errors) e = l.matches;
    else if (t.length === 0 && !l.initialized && l.matches.length > 0) e = l.matches;
    else return null;
  }
  let r = e,
    o = l?.errors;
  if (o != null) {
    let f = r.findIndex((d) => d.route.id && o?.[d.route.id] !== void 0);
    me(
      f >= 0,
      `Could not find a matching route for errors on route IDs: ${Object.keys(o).join(',')}`
    ),
      (r = r.slice(0, Math.min(r.length, f + 1)));
  }
  let n = !1,
    u = -1;
  if (a && l) {
    n = l.renderFallback;
    for (let f = 0; f < r.length; f++) {
      let d = r[f];
      if (((d.route.HydrateFallback || d.route.hydrateFallbackElement) && (u = f), d.route.id)) {
        let { loaderData: p, errors: h } = l,
          x = d.route.loader && !p.hasOwnProperty(d.route.id) && (!h || h[d.route.id] === void 0);
        if (d.route.lazy || x) {
          a.isStatic && (n = !0), u >= 0 ? (r = r.slice(0, u + 1)) : (r = [r[0]]);
          break;
        }
      }
    }
  }
  let i = a?.onError,
    s =
      l && i
        ? (f, d) => {
            i(f, {
              location: l.location,
              params: l.matches?.[0]?.params ?? {},
              pattern: G0(l.matches),
              errorInfo: d,
            });
          }
        : void 0;
  return r.reduceRight((f, d, p) => {
    let h,
      x = !1,
      v = null,
      L = null;
    l &&
      ((h = o && d.route.id ? o[d.route.id] : void 0),
      (v = d.route.errorElement || iC),
      n &&
        (u < 0 && p === 0
          ? (Dx(
              'route-fallback',
              !1,
              'No `HydrateFallback` element provided to render during initial hydration'
            ),
            (x = !0),
            (L = null))
          : u === p && ((x = !0), (L = d.route.hydrateFallbackElement || null))));
    let c = t.concat(r.slice(0, p + 1)),
      m = () => {
        let g;
        return (
          h
            ? (g = v)
            : x
              ? (g = L)
              : d.route.Component
                ? (g = D.createElement(d.route.Component, null))
                : d.route.element
                  ? (g = d.route.element)
                  : (g = f),
          D.createElement(dC, {
            match: d,
            routeContext: { outlet: f, matches: c, isDataRoute: l != null },
            children: g,
          })
        );
      };
    return l && (d.route.ErrorBoundary || d.route.errorElement || p === 0)
      ? D.createElement(Ax, {
          location: l.location,
          revalidation: l.revalidation,
          component: v,
          error: h,
          children: m(),
          routeContext: { outlet: null, matches: c, isDataRoute: !0 },
          onError: s,
        })
      : m();
  }, null);
}
function ac(e) {
  return `${e} must be used within a data router.  See https://reactrouter.com/en/main/routers/picking-a-router.`;
}
function cC(e) {
  let t = D.useContext(Vl);
  return me(t, ac(e)), t;
}
function lc(e) {
  let t = D.useContext(Jr);
  return me(t, ac(e)), t;
}
function mC(e) {
  let t = D.useContext(Tt);
  return me(t, ac(e)), t;
}
function rc(e) {
  let t = mC(e),
    a = t.matches[t.matches.length - 1];
  return me(a.route.id, `${e} can only be used on routes that contain a unique "id"`), a.route.id;
}
function pC() {
  return rc('useRouteId');
}
function Tx() {
  let e = lc('useNavigation');
  return D.useMemo(() => {
    let { matches: t, historyAction: a, ...l } = e.navigation;
    return l;
  }, [e.navigation]);
}
function oc() {
  let { matches: e, loaderData: t } = lc('useMatches');
  return D.useMemo(() => e.map((a) => I0(a, t)), [e, t]);
}
function Mx() {
  let e = D.useContext(tc),
    t = lc('useRouteError'),
    a = rc('useRouteError');
  return e !== void 0 ? e : t.errors?.[a];
}
function hC() {
  let { router: e } = cC('useNavigate'),
    t = rc('useNavigate'),
    a = D.useRef(!1);
  return (
    wx(() => {
      a.current = !0;
    }),
    D.useCallback(
      async (r, o = {}) => {
        pt(a.current, Cx),
          a.current &&
            (typeof r == 'number'
              ? await e.navigate(r)
              : await e.navigate(r, { fromRouteId: t, ...o }));
      },
      [e, t]
    )
  );
}
var lx = {};
function Dx(e, t, a) {
  !t && !lx[e] && ((lx[e] = !0), pt(!1, a));
}
var gC = 'useOptimistic',
  VR = de[gC];
var jR = de.memo(yC);
function yC({ routes: e, manifest: t, future: a, state: l, isStatic: r, onError: o }) {
  return Ex(e, void 0, { manifest: t, state: l, isStatic: r, onError: o, future: a });
}
function to({ to: e, replace: t, state: a, relative: l }) {
  me(jl(), '<Navigate> may be used only in the context of a <Router> component.');
  let { static: r, navigator: o } = de.useContext(Je);
  pt(
    !r,
    '<Navigate> must not be used on the initial render in a <StaticRouter>. This is a no-op, but you should modify your code so the <Navigate> is only ever rendered in response to some user interaction or state change.'
  );
  let { matches: n } = de.useContext(Tt),
    { pathname: u } = gt(),
    i = Ci(),
    s = bi(e, Wf(n), u, l === 'path');
  $f(typeof e == 'string' ? e : xl(e), o.createHref(s), Jf(o), 'reject');
  let f = JSON.stringify(s);
  return (
    de.useEffect(() => {
      i(JSON.parse(f), { replace: t, state: a, relative: l });
    }, [i, f, l, t, a]),
    null
  );
}
function xC(e) {
  return Rx(e.context);
}
function kx(e) {
  me(
    !1,
    'A <Route> is only ever to be used as the child of <Routes> element, never rendered directly. Please wrap your <Route> in a <Routes>.'
  );
}
function nc({
  basename: e = '/',
  children: t = null,
  location: a,
  navigationType: l = 'POP',
  navigator: r,
  static: o = !1,
  useTransitions: n,
}) {
  me(
    !jl(),
    'You cannot render a <Router> inside another <Router>. You should never have more than one in your app.'
  );
  let u = e.replace(/^\/*/, '/'),
    i = de.useMemo(
      () => ({ basename: u, navigator: r, static: o, useTransitions: n, future: {} }),
      [u, r, o, n]
    );
  typeof a == 'string' && (a = Gl(a));
  let {
      pathname: s = '/',
      search: f = '',
      hash: d = '',
      state: p = null,
      key: h = 'default',
      mask: x,
    } = a,
    v = de.useMemo(() => {
      let L = da(s, u);
      return L == null
        ? null
        : {
            location: { pathname: L, search: f, hash: d, state: p, key: h, mask: x },
            navigationType: l,
          };
    }, [u, s, f, d, p, h, l, x]);
  return (
    pt(
      v != null,
      `<Router basename="${u}"> is not able to match the URL "${s}${f}${d}" because it does not start with the basename, so the <Router> won't render anything.`
    ),
    v == null
      ? null
      : de.createElement(
          Je.Provider,
          { value: i },
          de.createElement($r.Provider, { children: t, value: v })
        )
  );
}
function vC({ children: e, location: t }) {
  return Ix(Li(e), t);
}
function Li(e, t = []) {
  let a = [];
  return (
    de.Children.forEach(e, (l, r) => {
      if (!de.isValidElement(l)) return;
      let o = [...t, r];
      if (l.type === de.Fragment) {
        a.push.apply(a, Li(l.props.children, o));
        return;
      }
      me(
        l.type === kx,
        `[${typeof l.type == 'string' ? l.type : l.type.name}] is not a <Route> component. All component children of <Routes> must be a <Route> or <React.Fragment>`
      ),
        me(!l.props.index || !l.props.children, 'An index route cannot have child routes.');
      let n = {
        id: l.props.id || o.join('-'),
        caseSensitive: l.props.caseSensitive,
        element: l.props.element,
        Component: l.props.Component,
        index: l.props.index,
        path: l.props.path,
        middleware: l.props.middleware,
        loader: l.props.loader,
        action: l.props.action,
        hydrateFallbackElement: l.props.hydrateFallbackElement,
        HydrateFallback: l.props.HydrateFallback,
        errorElement: l.props.errorElement,
        ErrorBoundary: l.props.ErrorBoundary,
        hasErrorBoundary:
          l.props.hasErrorBoundary === !0 ||
          l.props.ErrorBoundary != null ||
          l.props.errorElement != null,
        shouldRevalidate: l.props.shouldRevalidate,
        handle: l.props.handle,
        lazy: l.props.lazy,
      };
      l.props.children && (n.children = Li(l.props.children, o)), a.push(n);
    }),
    a
  );
}
var xi = 'get',
  vi = 'application/x-www-form-urlencoded';
function wi(e) {
  return typeof HTMLElement < 'u' && e instanceof HTMLElement;
}
function LC(e) {
  return wi(e) && e.tagName.toLowerCase() === 'button';
}
function SC(e) {
  return wi(e) && e.tagName.toLowerCase() === 'form';
}
function bC(e) {
  return wi(e) && e.tagName.toLowerCase() === 'input';
}
function CC(e) {
  return !!(e.metaKey || e.altKey || e.ctrlKey || e.shiftKey);
}
function wC(e, t) {
  return e.button === 0 && (!t || t === '_self') && !CC(e);
}
function Si(e = '') {
  return new URLSearchParams(
    typeof e == 'string' || Array.isArray(e) || e instanceof URLSearchParams
      ? e
      : Object.keys(e).reduce((t, a) => {
          let l = e[a];
          return t.concat(Array.isArray(l) ? l.map((r) => [a, r]) : [[a, l]]);
        }, [])
  );
}
function RC(e, t) {
  let a = Si(e);
  return (
    t &&
      t.forEach((l, r) => {
        a.has(r) ||
          t.getAll(r).forEach((o) => {
            a.append(r, o);
          });
      }),
    a
  );
}
var gi = null;
function IC() {
  if (gi === null)
    try {
      new FormData(document.createElement('form'), 0), (gi = !1);
    } catch {
      gi = !0;
    }
  return gi;
}
var EC = new Set(['application/x-www-form-urlencoded', 'multipart/form-data', 'text/plain']);
function Vf(e) {
  return e != null && !EC.has(e)
    ? (pt(
        !1,
        `"${e}" is not a valid \`encType\` for \`<Form>\`/\`<fetcher.Form>\` and will default to "${vi}"`
      ),
      null)
    : e;
}
function AC(e, t) {
  let a, l, r, o, n;
  if (SC(e)) {
    let u = e.getAttribute('action');
    (l = u ? da(u, t) : null),
      (a = e.getAttribute('method') || xi),
      (r = Vf(e.getAttribute('enctype')) || vi),
      (o = new FormData(e));
  } else if (LC(e) || (bC(e) && (e.type === 'submit' || e.type === 'image'))) {
    let u = e.form;
    if (u == null)
      throw new Error('Cannot submit a <button> or <input type="submit"> without a <form>');
    let i = e.getAttribute('formaction') || u.getAttribute('action');
    if (
      ((l = i ? da(i, t) : null),
      (a = e.getAttribute('formmethod') || u.getAttribute('method') || xi),
      (r = Vf(e.getAttribute('formenctype')) || Vf(u.getAttribute('enctype')) || vi),
      (o = new FormData(u, e)),
      !IC())
    ) {
      let { name: s, type: f, value: d } = e;
      if (f === 'image') {
        let p = s ? `${s}.` : '';
        o.append(`${p}x`, '0'), o.append(`${p}y`, '0');
      } else s && o.append(s, d);
    }
  } else {
    if (wi(e))
      throw new Error(
        'Cannot submit element that is not <form>, <button>, or <input type="submit|image">'
      );
    (a = xi), (l = null), (r = vi), (n = e);
  }
  return (
    o && r === 'text/plain' && ((n = o), (o = void 0)),
    { action: l, method: a.toLowerCase(), encType: r, formData: o, body: n }
  );
}
var XR = Object.getOwnPropertyNames(Object.prototype).sort().join('\0');
var TC = {
    '&': '\\u0026',
    '>': '\\u003e',
    '<': '\\u003c',
    '\u2028': '\\u2028',
    '\u2029': '\\u2029',
  },
  MC = /[&><\u2028\u2029]/g;
function rx(e) {
  return e.replace(MC, (t) => TC[t]);
}
function ic(e, t) {
  if (e === !1 || e === null || typeof e > 'u') throw new Error(t);
}
var DC = Symbol('SingleFetchRedirect');
function Bx(e, t, a, l) {
  let r =
    typeof e == 'string'
      ? new URL(e, typeof window > 'u' ? 'server://singlefetch/' : window.location.origin)
      : e;
  return (
    a
      ? r.pathname.endsWith('/')
        ? (r.pathname = `${r.pathname}_.${l}`)
        : (r.pathname = `${r.pathname}.${l}`)
      : r.pathname === '/'
        ? (r.pathname = `_root.${l}`)
        : t && da(r.pathname, t) === '/'
          ? (r.pathname = `${Wr(t)}/_root.${l}`)
          : (r.pathname = `${Wr(r.pathname)}.${l}`),
    r
  );
}
async function kC(e, t) {
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
function BC(e) {
  return e != null && typeof e.page == 'string';
}
function OC(e) {
  return e == null
    ? !1
    : e.href == null
      ? e.rel === 'preload' && typeof e.imageSrcSet == 'string' && typeof e.imageSizes == 'string'
      : typeof e.rel == 'string' && typeof e.href == 'string';
}
async function UC(e, t, a) {
  let l = await Promise.all(
    e.map(async (r) => {
      let o = t.routes[r.route.id];
      if (o) {
        let n = await kC(o, a);
        return n.links ? n.links() : [];
      }
      return [];
    })
  );
  return _C(
    l
      .flat(1)
      .filter(OC)
      .filter((r) => r.rel === 'stylesheet' || r.rel === 'preload')
      .map((r) =>
        r.rel === 'stylesheet' ? { ...r, rel: 'prefetch', as: 'style' } : { ...r, rel: 'prefetch' }
      )
  );
}
function ox(e, t, a, l, r, o) {
  let n = (i, s) => (a[s] ? i.route.id !== a[s].route.id : !0),
    u = (i, s) =>
      a[s].pathname !== i.pathname ||
      (a[s].route.path?.endsWith('*') && a[s].params['*'] !== i.params['*']);
  return o === 'assets'
    ? t.filter((i, s) => n(i, s) || u(i, s))
    : o === 'data'
      ? t.filter((i, s) => {
          let f = l.routes[i.route.id];
          if (!f || !f.hasLoader) return !1;
          if (n(i, s) || u(i, s)) return !0;
          if (i.route.shouldRevalidate) {
            let d = i.route.shouldRevalidate({
              currentUrl: new URL(r.pathname + r.search + r.hash, window.origin),
              currentParams: a[0]?.params || {},
              nextUrl: new URL(e, window.origin),
              nextParams: i.params,
              defaultShouldRevalidate: !0,
            });
            if (typeof d == 'boolean') return d;
          }
          return !0;
        })
      : [];
}
function HC(e, t, { includeHydrateFallback: a } = {}) {
  return NC(
    e
      .map((l) => {
        let r = t.routes[l.route.id];
        if (!r) return [];
        let o = [r.module];
        return (
          r.clientActionModule && (o = o.concat(r.clientActionModule)),
          r.clientLoaderModule && (o = o.concat(r.clientLoaderModule)),
          a && r.hydrateFallbackModule && (o = o.concat(r.hydrateFallbackModule)),
          r.imports && (o = o.concat(r.imports)),
          o
        );
      })
      .flat(1)
  );
}
function NC(e) {
  return [...new Set(e)];
}
function PC(e) {
  let t = {},
    a = Object.keys(e).sort();
  for (let l of a) t[l] = e[l];
  return t;
}
function _C(e, t) {
  let a = new Set(),
    l = new Set(t);
  return e.reduce((r, o) => {
    if (t && !BC(o) && o.as === 'script' && o.href && l.has(o.href)) return r;
    let u = JSON.stringify(PC(o));
    return a.has(u) || (a.add(u), r.push({ key: u, link: o })), r;
  }, []);
}
function sc() {
  let e = J.useContext(Vl);
  return ic(e, 'You must render this element inside a <DataRouterContext.Provider> element'), e;
}
function GC() {
  let e = J.useContext(Jr);
  return (
    ic(e, 'You must render this element inside a <DataRouterStateContext.Provider> element'), e
  );
}
var Rn = J.createContext(void 0);
Rn.displayName = 'FrameworkContext';
function Ri() {
  let e = J.useContext(Rn);
  return ic(e, 'You must render this element inside a <HydratedRouter> element'), e;
}
function VC(e, t) {
  let a = J.useContext(Rn),
    [l, r] = J.useState(!1),
    [o, n] = J.useState(!1),
    { onFocus: u, onBlur: i, onMouseEnter: s, onMouseLeave: f, onTouchStart: d } = t,
    p = J.useRef(null);
  J.useEffect(() => {
    if ((e === 'render' && n(!0), e === 'viewport')) {
      let v = (c) => {
          c.forEach((m) => {
            n(m.isIntersecting);
          });
        },
        L = new IntersectionObserver(v, { threshold: 0.5 });
      return (
        p.current && L.observe(p.current),
        () => {
          L.disconnect();
        }
      );
    }
  }, [e]),
    J.useEffect(() => {
      if (l) {
        let v = setTimeout(() => {
          n(!0);
        }, 100);
        return () => {
          clearTimeout(v);
        };
      }
    }, [l]);
  let h = () => {
      r(!0);
    },
    x = () => {
      r(!1), n(!1);
    };
  return a
    ? e !== 'intent'
      ? [o, p, {}]
      : [
          o,
          p,
          {
            onFocus: Cn(u, h),
            onBlur: Cn(i, x),
            onMouseEnter: Cn(s, h),
            onMouseLeave: Cn(f, x),
            onTouchStart: Cn(d, h),
          },
        ]
    : [!1, p, {}];
}
function Cn(e, t) {
  return (a) => {
    e && e(a), a.defaultPrevented || t(a);
  };
}
function Ux({ page: e, ...t }) {
  let a = J0(),
    { nonce: l } = Ri(),
    { router: r } = sc(),
    o = J.useMemo(() => Zf(r.routes, e, r.basename), [r.routes, e, r.basename]);
  return o
    ? (t.nonce == null && l && (t = { ...t, nonce: l }),
      a
        ? J.createElement(XC, { page: e, matches: o, ...t })
        : J.createElement(YC, { page: e, matches: o, ...t }))
    : null;
}
function jC(e) {
  let { manifest: t, routeModules: a } = Ri(),
    [l, r] = J.useState([]);
  return (
    J.useEffect(() => {
      let o = !1;
      return (
        UC(e, t, a).then((n) => {
          o || r(n);
        }),
        () => {
          o = !0;
        }
      );
    }, [e, t, a]),
    l
  );
}
function XC({ page: e, matches: t, ...a }) {
  let l = gt(),
    { future: r } = Ri(),
    { basename: o } = sc(),
    n = J.useMemo(() => {
      if (e === l.pathname + l.search + l.hash) return [];
      let u = Bx(e, o, r.v8_trailingSlashAwareDataRequests, 'rsc'),
        i = !1,
        s = [];
      for (let f of t)
        typeof f.route.shouldRevalidate == 'function' ? (i = !0) : s.push(f.route.id);
      return (
        i && s.length > 0 && u.searchParams.set('_routes', s.join(',')), [u.pathname + u.search]
      );
    }, [o, r.v8_trailingSlashAwareDataRequests, e, l, t]);
  return J.createElement(
    J.Fragment,
    null,
    n.map((u) => J.createElement('link', { key: u, rel: 'prefetch', as: 'fetch', href: u, ...a }))
  );
}
function YC({ page: e, matches: t, ...a }) {
  let l = gt(),
    { future: r, manifest: o, routeModules: n } = Ri(),
    { basename: u } = sc(),
    { loaderData: i, matches: s } = GC(),
    f = J.useMemo(() => ox(e, t, s, o, l, 'data'), [e, t, s, o, l]),
    d = J.useMemo(() => ox(e, t, s, o, l, 'assets'), [e, t, s, o, l]),
    p = J.useMemo(() => {
      if (e === l.pathname + l.search + l.hash) return [];
      let v = new Set(),
        L = !1;
      if (
        (t.forEach((m) => {
          let g = o.routes[m.route.id];
          !g ||
            !g.hasLoader ||
            ((!f.some((y) => y.route.id === m.route.id) &&
              m.route.id in i &&
              n[m.route.id]?.shouldRevalidate) ||
            g.hasClientLoader
              ? (L = !0)
              : v.add(m.route.id));
        }),
        v.size === 0)
      )
        return [];
      let c = Bx(e, u, r.v8_trailingSlashAwareDataRequests, 'data');
      return (
        L &&
          v.size > 0 &&
          c.searchParams.set(
            '_routes',
            t
              .filter((m) => v.has(m.route.id))
              .map((m) => m.route.id)
              .join(',')
          ),
        [c.pathname + c.search]
      );
    }, [u, r.v8_trailingSlashAwareDataRequests, i, l, o, f, t, e, n]),
    h = J.useMemo(() => HC(d, o), [d, o]),
    x = jC(d);
  return J.createElement(
    J.Fragment,
    null,
    p.map((v) => J.createElement('link', { key: v, rel: 'prefetch', as: 'fetch', href: v, ...a })),
    h.map((v) => J.createElement('link', { key: v, rel: 'modulepreload', href: v, ...a })),
    x.map(({ key: v, link: L }) =>
      J.createElement('link', {
        key: v,
        nonce: a.nonce,
        ...L,
        crossOrigin: L.crossOrigin ?? a.crossOrigin,
      })
    )
  );
}
function KC(...e) {
  return (t) => {
    e.forEach((a) => {
      typeof a == 'function' ? a(t) : a != null && (a.current = t);
    });
  };
}
var QC =
  typeof window < 'u' && typeof window.document < 'u' && typeof window.document.createElement < 'u';
try {
  QC && (window.__reactRouterVersion = '7.18.3');
} catch {}
function ZC({ basename: e, children: t, useTransitions: a, window: l }) {
  let r = k.useRef();
  r.current == null && (r.current = ux({ window: l, v5Compat: !0 }));
  let o = r.current,
    [n, u] = k.useState({ action: o.action, location: o.location }),
    i = k.useCallback(
      (s) => {
        a === !1 ? u(s) : k.startTransition(() => u(s));
      },
      [a]
    );
  return (
    k.useLayoutEffect(() => o.listen(i), [o, i]),
    k.createElement(nc, {
      basename: e,
      children: t,
      location: n.location,
      navigationType: n.action,
      navigator: o,
      useTransitions: a,
    })
  );
}
function Hx({ basename: e, children: t, history: a, useTransitions: l }) {
  let [r, o] = k.useState({ action: a.action, location: a.location }),
    n = k.useCallback(
      (u) => {
        l === !1 ? o(u) : k.startTransition(() => o(u));
      },
      [l]
    );
  return (
    k.useLayoutEffect(() => a.listen(n), [a, n]),
    k.createElement(nc, {
      basename: e,
      children: t,
      location: r.location,
      navigationType: r.action,
      navigator: a,
      useTransitions: l,
    })
  );
}
Hx.displayName = 'unstable_HistoryRouter';
var Xl = k.forwardRef(function (
  {
    onClick: t,
    discover: a = 'render',
    prefetch: l = 'none',
    relative: r,
    reloadDocument: o,
    replace: n,
    mask: u,
    state: i,
    target: s,
    to: f,
    preventScrollReset: d,
    viewTransition: p,
    defaultShouldRevalidate: h,
    ...x
  },
  v
) {
  let { basename: L, navigator: c, useTransitions: m } = k.useContext(Je),
    g = typeof f == 'string' && Qf.test(f),
    y = yx(f, L);
  f = y.to;
  let w = bx(f, { relative: r }),
    U = gt(),
    C = null;
  if (u) {
    let be = bi(u, [], U.mask ? U.mask.pathname : '/', !0);
    L !== '/' && (be.pathname = be.pathname === '/' ? L : Qt([L, be.pathname])),
      (C = c.createHref(be));
  }
  let [b, M, _] = VC(l, x),
    Ue = Fx(f, {
      replace: n,
      mask: u,
      state: i,
      target: s,
      preventScrollReset: d,
      relative: r,
      viewTransition: p,
      defaultShouldRevalidate: h,
      useTransitions: m,
    });
  function ot(be) {
    t && t(be), be.defaultPrevented || Ue(be);
  }
  let Gt = !(y.isExternal || o),
    Qe = k.createElement('a', {
      ...x,
      ..._,
      href: (Gt ? C : void 0) || y.absoluteURL || w,
      onClick: Gt ? ot : t,
      ref: KC(v, M),
      target: s,
      'data-discover': !g && a === 'render' ? 'true' : void 0,
    });
  return b && !g ? k.createElement(k.Fragment, null, Qe, k.createElement(Ux, { page: w })) : Qe;
});
Xl.displayName = 'Link';
var Nx = k.forwardRef(function (
  {
    'aria-current': t = 'page',
    caseSensitive: a = !1,
    className: l = '',
    end: r = !1,
    style: o,
    to: n,
    viewTransition: u,
    children: i,
    ...s
  },
  f
) {
  let d = eo(n, { relative: s.relative }),
    p = gt(),
    h = k.useContext(Jr),
    { navigator: x, basename: v } = k.useContext(Je),
    L = h != null && jx(d) && u === !0,
    c = x.encodeLocation ? x.encodeLocation(d).pathname : d.pathname,
    m = p.pathname,
    g = h && h.navigation && h.navigation.location ? h.navigation.location.pathname : null;
  a || ((m = m.toLowerCase()), (g = g ? g.toLowerCase() : null), (c = c.toLowerCase())),
    g && v && (g = da(g, v) || g);
  let y = c !== '/' && c.endsWith('/') ? c.length - 1 : c.length,
    w = m === c || (!r && m.startsWith(c) && m.charAt(y) === '/'),
    U = g != null && (g === c || (!r && g.startsWith(c) && g.charAt(c.length) === '/')),
    C = { isActive: w, isPending: U, isTransitioning: L },
    b = w ? t : void 0,
    M;
  typeof l == 'function'
    ? (M = l(C))
    : (M = [l, w ? 'active' : null, U ? 'pending' : null, L ? 'transitioning' : null]
        .filter(Boolean)
        .join(' '));
  let _ = typeof o == 'function' ? o(C) : o;
  return k.createElement(
    Xl,
    { ...s, 'aria-current': b, className: M, ref: f, style: _, to: n, viewTransition: u },
    typeof i == 'function' ? i(C) : i
  );
});
Nx.displayName = 'NavLink';
var Px = k.forwardRef(
  (
    {
      discover: e = 'render',
      fetcherKey: t,
      navigate: a,
      reloadDocument: l,
      replace: r,
      state: o,
      method: n = xi,
      action: u,
      onSubmit: i,
      relative: s,
      preventScrollReset: f,
      viewTransition: d,
      defaultShouldRevalidate: p,
      ...h
    },
    x
  ) => {
    let { useTransitions: v } = k.useContext(Je),
      L = qx(),
      c = Gx(u, { relative: s }),
      m = n.toLowerCase() === 'get' ? 'get' : 'post',
      g = typeof u == 'string' && Qf.test(u);
    return k.createElement('form', {
      ref: x,
      method: m,
      action: c,
      onSubmit: l
        ? i
        : (w) => {
            if ((i && i(w), w.defaultPrevented)) return;
            w.preventDefault();
            let U = w.nativeEvent.submitter,
              C = U?.getAttribute('formmethod') || n,
              b = () =>
                L(U || w.currentTarget, {
                  fetcherKey: t,
                  method: C,
                  navigate: a,
                  replace: r,
                  state: o,
                  relative: s,
                  preventScrollReset: f,
                  viewTransition: d,
                  defaultShouldRevalidate: p,
                });
            v && a !== !1 ? k.startTransition(() => b()) : b();
          },
      ...h,
      'data-discover': !g && e === 'render' ? 'true' : void 0,
    });
  }
);
Px.displayName = 'Form';
function _x({ getKey: e, storageKey: t, ...a }) {
  let l = k.useContext(Rn),
    { basename: r } = k.useContext(Je),
    o = gt(),
    n = oc();
  Vx({ getKey: e, storageKey: t });
  let u = k.useMemo(() => {
    if (!l || !e) return null;
    let s = Kf(o, n, r, e);
    return s !== o.key ? s : null;
  }, []);
  if (!l || l.isSpaMode) return null;
  let i = ((s, f) => {
    if (!window.history.state || !window.history.state.key) {
      let d = Math.random().toString(32).slice(2);
      window.history.replaceState({ key: d }, '');
    }
    try {
      let p = JSON.parse(sessionStorage.getItem(s) || '{}')[f || window.history.state.key];
      typeof p == 'number' && window.scrollTo(0, p);
    } catch (d) {
      console.error(d), sessionStorage.removeItem(s);
    }
  }).toString();
  return (
    a.nonce == null && l?.nonce && (a.nonce = l.nonce),
    k.createElement('script', {
      ...a,
      suppressHydrationWarning: !0,
      dangerouslySetInnerHTML: {
        __html: `(${i})(${rx(JSON.stringify(t || Yf))}, ${rx(JSON.stringify(u))})`,
      },
    })
  );
}
_x.displayName = 'ScrollRestoration';
function zx(e) {
  return `${e} must be used within a data router.  See https://reactrouter.com/en/main/routers/picking-a-router.`;
}
function dc(e) {
  let t = k.useContext(Vl);
  return me(t, zx(e)), t;
}
function WC(e) {
  let t = k.useContext(Jr);
  return me(t, zx(e)), t;
}
function Fx(
  e,
  {
    target: t,
    replace: a,
    mask: l,
    state: r,
    preventScrollReset: o,
    relative: n,
    viewTransition: u,
    defaultShouldRevalidate: i,
    useTransitions: s,
  } = {}
) {
  let f = Ci(),
    d = gt(),
    p = eo(e, { relative: n });
  return k.useCallback(
    (h) => {
      if (wC(h, t)) {
        h.preventDefault();
        let x = a !== void 0 ? a : xl(d) === xl(p),
          v = () =>
            f(e, {
              replace: x,
              mask: l,
              state: r,
              preventScrollReset: o,
              relative: n,
              viewTransition: u,
              defaultShouldRevalidate: i,
            });
        s ? k.startTransition(() => v()) : v();
      }
    },
    [d, f, p, a, l, r, t, e, o, n, u, i, s]
  );
}
function JC(e) {
  pt(
    typeof URLSearchParams < 'u',
    'You cannot use the `useSearchParams` hook in a browser that does not support the URLSearchParams API. If you need to support Internet Explorer 11, we recommend you load a polyfill such as https://github.com/ungap/url-search-params.'
  );
  let t = k.useRef(Si(e)),
    a = k.useRef(!1),
    l = gt(),
    r = k.useMemo(() => RC(l.search, a.current ? null : t.current), [l.search]),
    o = Ci(),
    n = k.useCallback(
      (u, i) => {
        let s = Si(typeof u == 'function' ? u(new URLSearchParams(r)) : u);
        (a.current = !0), o('?' + s, i);
      },
      [o, r]
    );
  return [r, n];
}
var $C = 0,
  ew = () => `__${String(++$C)}__`;
function qx() {
  let { router: e } = dc('useSubmit'),
    { basename: t } = k.useContext(Je),
    a = pC(),
    l = e.fetch,
    r = e.navigate;
  return k.useCallback(
    async (o, n = {}) => {
      let { action: u, method: i, encType: s, formData: f, body: d } = AC(o, t);
      if (n.navigate === !1) {
        let p = n.fetcherKey || ew();
        await l(p, a, n.action || u, {
          defaultShouldRevalidate: n.defaultShouldRevalidate,
          preventScrollReset: n.preventScrollReset,
          formData: f,
          body: d,
          formMethod: n.method || i,
          formEncType: n.encType || s,
          flushSync: n.flushSync,
        });
      } else
        await r(n.action || u, {
          defaultShouldRevalidate: n.defaultShouldRevalidate,
          preventScrollReset: n.preventScrollReset,
          formData: f,
          body: d,
          formMethod: n.method || i,
          formEncType: n.encType || s,
          replace: n.replace,
          state: n.state,
          fromRouteId: a,
          flushSync: n.flushSync,
          viewTransition: n.viewTransition,
        });
    },
    [l, r, t, a]
  );
}
function Gx(e, { relative: t } = {}) {
  let { basename: a } = k.useContext(Je),
    l = k.useContext(Tt);
  me(l, 'useFormAction must be used inside a RouteContext');
  let [r] = l.matches.slice(-1),
    o = { ...eo(e || '.', { relative: t }) },
    n = gt();
  if (e == null) {
    o.search = n.search;
    let u = new URLSearchParams(o.search),
      i = u.getAll('index');
    if (i.some((f) => f === '')) {
      u.delete('index'), i.filter((d) => d).forEach((d) => u.append('index', d));
      let f = u.toString();
      o.search = f ? `?${f}` : '';
    }
  }
  return (
    (!e || e === '.') &&
      r.route.index &&
      (o.search = o.search ? o.search.replace(/^\?/, '?index&') : '?index'),
    a !== '/' && (o.pathname = o.pathname === '/' ? a : Qt([a, o.pathname])),
    xl(o)
  );
}
var Yf = 'react-router-scroll-positions',
  yi = {};
function Kf(e, t, a, l) {
  let r = null;
  return (
    l &&
      (a !== '/' ? (r = l({ ...e, pathname: da(e.pathname, a) || e.pathname }, t)) : (r = l(e, t))),
    r == null && (r = e.key),
    r
  );
}
function Vx({ getKey: e, storageKey: t } = {}) {
  let { router: a } = dc('useScrollRestoration'),
    { restoreScrollPosition: l, preventScrollReset: r } = WC('useScrollRestoration'),
    { basename: o } = k.useContext(Je),
    n = gt(),
    u = oc(),
    i = Tx();
  k.useEffect(
    () => (
      (window.history.scrollRestoration = 'manual'),
      () => {
        window.history.scrollRestoration = 'auto';
      }
    ),
    []
  ),
    tw(
      k.useCallback(() => {
        if (i.state === 'idle') {
          let s = Kf(n, u, o, e);
          yi[s] = window.scrollY;
        }
        try {
          sessionStorage.setItem(t || Yf, JSON.stringify(yi));
        } catch (s) {
          pt(
            !1,
            `Failed to save scroll positions in sessionStorage, <ScrollRestoration /> will not work properly (${s}).`
          );
        }
        window.history.scrollRestoration = 'auto';
      }, [i.state, e, o, n, u, t])
    ),
    typeof document < 'u' &&
      (k.useLayoutEffect(() => {
        try {
          let s = sessionStorage.getItem(t || Yf);
          s && (yi = JSON.parse(s));
        } catch {}
      }, [t]),
      k.useLayoutEffect(() => {
        let s = a?.enableScrollRestoration(
          yi,
          () => window.scrollY,
          e ? (f, d) => Kf(f, d, o, e) : void 0
        );
        return () => s && s();
      }, [a, o, e]),
      k.useLayoutEffect(() => {
        if (l !== !1) {
          if (typeof l == 'number') {
            window.scrollTo(0, l);
            return;
          }
          try {
            if (n.hash) {
              let s = document.getElementById(decodeURIComponent(n.hash.slice(1)));
              if (s) {
                s.scrollIntoView();
                return;
              }
            }
          } catch {
            pt(
              !1,
              `"${n.hash.slice(1)}" is not a decodable element ID. The view will not scroll to it.`
            );
          }
          r !== !0 && window.scrollTo(0, 0);
        }
      }, [n, l, r]));
}
function tw(e, t) {
  let { capture: a } = t || {};
  k.useEffect(() => {
    let l = a != null ? { capture: a } : void 0;
    return (
      window.addEventListener('pagehide', e, l),
      () => {
        window.removeEventListener('pagehide', e, l);
      }
    );
  }, [e, a]);
}
function jx(e, { relative: t } = {}) {
  let a = k.useContext(ec);
  me(
    a != null,
    "`useViewTransitionState` must be used within `react-router-dom`'s `RouterProvider`.  Did you accidentally import `RouterProvider` from `react-router`?"
  );
  let { basename: l } = dc('useViewTransitionState'),
    r = eo(e, { relative: t });
  if (!a.isTransitioning) return !1;
  let o = da(a.currentLocation.pathname, l) || a.currentLocation.pathname,
    n = da(a.nextLocation.pathname, l) || a.nextLocation.pathname;
  return wn(r.pathname, n) != null || wn(r.pathname, o) != null;
}
var Zt = E(te());
function Ei() {
  try {
    return rw(), window.localStorage.getItem('') === 'light' ? 'light' : 'dark';
  } catch {
    return 'dark';
  }
}
function rw() {
  try {
    if (window.localStorage.getItem('') === '1') return;
    window.localStorage.setItem('', 'dark'), window.localStorage.setItem('', '1');
  } catch {}
}
function Zx() {
  let e = document.documentElement;
  return e.classList.contains('light') ? 'light' : e.classList.contains('dark') ? 'dark' : Ei();
}
function Ai(e) {
  let t = document.documentElement,
    a = e === 'dark' ? 'light' : 'dark';
  if (t.classList.contains(e) && !t.classList.contains(a)) {
    t.style.colorScheme !== e && (t.style.colorScheme = e);
    return;
  }
  t.classList.remove('light', 'dark'), t.classList.add(e), (t.style.colorScheme = e);
}
function Ti(e) {
  try {
    window.localStorage.setItem('', e);
  } catch {}
}
function sE(e) {
  return e === 'dark' ? 'Switch to light theme' : 'Switch to dark theme';
}
var Wx = E(Q()),
  nw = (0, Zt.createContext)(null);
function cE({ children: e }) {
  let [t, a] = (0, Zt.useState)(() => Zx()),
    l = (0, Zt.useCallback)((o) => {
      a((n) => (n === o ? n : (Ai(o), Ti(o), o)));
    }, []);
  (0, Zt.useEffect)(() => {
    let o = Ei();
    Ai(o), Ti(o), a((n) => (n === o ? n : o));
  }, []),
    (0, Zt.useEffect)(() => {
      let o = (n) => {
        if (n.key !== '') return;
        let u = Ei();
        a((i) => (i === u ? i : (Ai(u), Ti(u), u)));
      };
      return window.addEventListener('storage', o), () => window.removeEventListener('storage', o);
    }, []);
  let r = (0, Zt.useMemo)(() => ({ theme: t, setTheme: l }), [t, l]);
  return (0, Wx.jsx)(nw.Provider, { value: r, children: e });
}
var Oi = E(te());
var Wt = E(te());
var Ke = class extends Error {
  status;
  code;
  constructor(t, a, l) {
    super(l), (this.name = 'ApiError'), (this.status = t), (this.code = a);
  }
};
var Jx = 15e3;
function uw(e) {
  if (typeof document > 'u') return;
  let t = `${e}=`;
  for (let a of document.cookie.split(';')) {
    let l = a.trim();
    if (l.startsWith(t)) return decodeURIComponent(l.slice(t.length));
  }
}
function iw(e) {
  let t = new AbortController(),
    a = () => {
      t.signal.aborted || t.abort();
    };
  for (let l of e) {
    if (l.aborted) return a(), t.signal;
    l.addEventListener('abort', a, { once: !0 });
  }
  return t.signal;
}
function sw(e) {
  let t = e.toUpperCase();
  return t !== 'GET' && t !== 'HEAD' && t !== 'OPTIONS';
}
function $x(e) {
  let t =
      typeof e.feature_key == 'string'
        ? e.feature_key
        : typeof e.feature_required == 'string'
          ? e.feature_required
          : '',
    a = typeof e.plan_code == 'string' ? e.plan_code : '';
  return t && a ? `${t} requires ${a} plan` : t || '';
}
function dw(e) {
  return e === 'feature_required' ? 'FEATURE_REQUIRED' : e;
}
async function In(e) {
  let t = 'HTTP_ERROR',
    a = e.statusText || `HTTP ${e.status}`;
  try {
    let l = await e.json();
    if (l && typeof l == 'object') {
      let r = l,
        o = r.error;
      if (o && typeof o == 'object') {
        let n = o;
        if (
          (typeof n.code == 'string' && (t = dw(n.code)),
          typeof n.message == 'string' && (a = n.message),
          t === 'FEATURE_REQUIRED' && a === e.statusText)
        ) {
          let u = $x({ ...r, ...n });
          u !== '' && (a = u);
        }
      } else if (typeof o == 'string')
        if (o === 'feature_required') {
          t = 'FEATURE_REQUIRED';
          let n = $x(r);
          a = n !== '' ? n : o;
        } else a = o;
    }
  } catch {}
  return new Ke(e.status, t, a);
}
function fc(e) {
  return e instanceof DOMException && e.name === 'AbortError'
    ? !0
    : e instanceof Error && e.name === 'AbortError';
}
async function En(e, t = {}) {
  let a = new AbortController(),
    l = setTimeout(() => {
      a.abort();
    }, Jx),
    r = [a.signal];
  t.signal && r.push(t.signal);
  let o = (t.method ?? 'GET').toUpperCase(),
    n = new Headers(t.headers ?? void 0);
  if (sw(o)) {
    let u = uw('csrfToken');
    u && n.set('X-CSRF-Token', u);
  }
  t.body != null && !n.has('Content-Type') && n.set('Content-Type', 'application/json');
  try {
    return await fetch(e, { ...t, method: o, headers: n, credentials: 'include', signal: iw(r) });
  } catch (u) {
    throw fc(u) && a.signal.aborted && !t.signal?.aborted
      ? new Ke(0, 'TIMEOUT', `Request timed out after ${Jx}ms`)
      : u;
  } finally {
    clearTimeout(l);
  }
}
async function gE(e, t, a) {
  let l = await En(e, t);
  if (!l.ok) throw await In(l);
  if (l.status === 204 || l.headers.get('Content-Length') === '0') return;
  let o = await l.json();
  return a(o);
}
async function Be(e, t = {}) {
  let a = await En(e, t);
  if (!a.ok) throw await In(a);
  if (!(a.status === 204 || a.headers.get('Content-Length') === '0')) return await a.json();
}
async function yE(e, t = {}) {
  let a = await Be(e, t);
  if (!Array.isArray(a)) throw new Ke(502, 'INVALID_RESPONSE', 'Expected JSON array response');
  return a;
}
async function xE(e, t, a, l = {}) {
  let r = await En(e, t);
  if (!r.ok) throw await In(r);
  let o = await r.json();
  if (!Array.isArray(o)) throw new Ke(502, 'INVALID_RESPONSE', 'Expected JSON array response');
  let n = o.map(a),
    u = l.totalHeader ?? 'X-Total-Count',
    i = r.headers.get(u),
    s = i != null ? Number.parseInt(i, 10) : n.length,
    f = Number.isFinite(s) ? s : n.length;
  return { items: n, total: f };
}
function ev(e, t) {
  let [a, l] = (0, Wt.useState)(void 0),
    [r, o] = (0, Wt.useState)(void 0),
    [n, u] = (0, Wt.useState)(!0),
    [i, s] = (0, Wt.useState)(!1),
    [f, d] = (0, Wt.useState)(null),
    p = (0, Wt.useRef)(0),
    h = (0, Wt.useRef)(!1);
  return (
    (0, Wt.useEffect)(() => {
      let x = new AbortController(),
        v = ++p.current,
        L = h.current;
      return (
        L ? s(!0) : (u(!0), s(!1)),
        e(x.signal)
          .then((c) => {
            v === p.current && ((h.current = !0), l(c), o(void 0), d(new Date().toISOString()));
          })
          .catch((c) => {
            v === p.current &&
              (fc(c) || (L && (h.current = !0), o(c instanceof Error ? c : new Error(String(c)))));
          })
          .finally(() => {
            v === p.current && (u(!1), s(!1));
          }),
        () => {
          x.abort();
        }
      );
    }, t),
    { data: a, error: r, fetching: n, revalidating: i, updatedAt: f }
  );
}
var Di = E(te());
var Mi = E(te());
function tv(e) {
  return e.inFlightGuard && e.inFlight
    ? 'skip_in_flight'
    : e.nowMs - e.lastFiredAtMs < e.windowMs
      ? 'skip_window'
      : 'allow';
}
function av(e, t = {}) {
  let { windowMs: a = 500, inFlightGuard: l = !1, inFlight: r = !1 } = t,
    o = (0, Mi.useRef)(0);
  return (0, Mi.useCallback)(() => {
    tv({
      lastFiredAtMs: o.current,
      nowMs: Date.now(),
      windowMs: a,
      inFlight: r,
      inFlightGuard: l,
    }) === 'allow' && ((o.current = Date.now()), e());
  }, [e, r, l, a]);
}
function lv() {
  let [e, t] = (0, Di.useState)(0),
    a = (0, Di.useCallback)(() => {
      t((l) => l + 1);
    }, []);
  return { refreshToken: e, bumpRefresh: a };
}
function rv(e, t) {
  return av(e, { inFlightGuard: !0, inFlight: t });
}
async function ov(e) {
  return Be('/api/v1/meta', { signal: e });
}
async function TE(e) {
  return Be('/api/v1/eula', { signal: e });
}
async function ME(e, t) {
  return Be('/api/v1/eula/accept', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function DE(e) {
  return Be('/api/v1/license/status', { signal: e });
}
async function kE(e, t) {
  return Be('/api/v1/license/apply', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function BE(e = {}, t) {
  let a = new URLSearchParams();
  e.customer_id && a.set('customer_id', e.customer_id),
    e.limit != null && a.set('limit', String(e.limit)),
    e.offset != null && a.set('offset', String(e.offset));
  let l = a.toString(),
    r = l ? `/api/v1/disputes?${l}` : '/api/v1/disputes';
  return Be(r, { signal: t });
}
async function OE(e) {
  return Be('/api/v1/support/feedback/meta', { signal: e });
}
async function UE(e, t) {
  return Be('/api/v1/support/feedback', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function HE(e, t, a) {
  let l = await En('/api/v1/consent', {
    method: 'POST',
    headers: { 'X-Consent-Signature': t },
    body: e,
    signal: a,
  });
  if (!l.ok) throw await In(l);
}
var ki, ao;
function Bi(e) {
  return ki
    ? Promise.resolve(ki)
    : ao ||
        ((ao = ov(e)
          .then((t) => ((ki = t), t))
          .finally(() => {
            ao = void 0;
          })),
        ao);
}
function nv() {
  (ki = void 0), (ao = void 0);
}
var cw = new Set(['EXPIRED', 'REVOKED']);
function uv(e) {
  return e?.bootstrap_complete === !0;
}
function iv(e) {
  let t = e?.license?.state?.trim();
  return t ? cw.has(t) : !0;
}
function zE(e) {
  return e?.license?.state?.trim() || 'missing';
}
var sv = E(Q()),
  cc = (0, Oi.createContext)(void 0);
function XE({ children: e }) {
  let { refreshToken: t, bumpRefresh: a } = lv(),
    { data: l, error: r, fetching: o } = ev((i) => Bi(i), [t]),
    n = rv(() => {
      nv(), a();
    }, o),
    u = (0, Oi.useMemo)(
      () => ({
        meta: l,
        error: r,
        loading: o,
        bootstrapComplete: uv(l),
        licenseNeedsSetup: iv(l),
        refreshMeta: n,
      }),
      [l, r, o, n]
    );
  return (0, sv.jsx)(cc.Provider, { value: u, children: e });
}
async function dv(e, t) {
  return Be('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function ZE(e) {
  await Be('/api/v1/auth/logout', { method: 'POST', signal: e });
}
async function WE(e) {
  return Be('/api/v1/auth/refresh', { method: 'POST', signal: e });
}
async function mw(e) {
  return Be('/api/v1/auth/me', { signal: e });
}
async function pw(e) {
  return Be('/api/v1/session', { signal: e });
}
async function hw(e) {
  let [t, a, l] = await Promise.all([mw(e), pw(e), Bi(e).catch(() => {})]);
  return {
    user: t,
    session: a,
    eula_required: l?.eula_required,
    eula_accepted: l?.eula_accepted,
    eula_version: l?.eula_version,
  };
}
async function JE(e) {
  try {
    return await Be('/api/v1/session/bootstrap', { signal: e });
  } catch (t) {
    if (!(t instanceof Ke) || (t.status !== 401 && t.status !== 404)) throw t;
    try {
      return await hw(e);
    } catch (a) {
      if (a instanceof Ke && a.status === 401) return null;
      throw a;
    }
  }
}
async function fv(e, t) {
  return Be('/api/v1/public/activate', { method: 'POST', body: JSON.stringify(e), signal: t });
}
async function $E(e, t) {
  return Be('/api/v1/public/invite/accept', { method: 'POST', body: JSON.stringify(e), signal: t });
}
function cv(e) {
  var t,
    a,
    l = '';
  if (typeof e == 'string' || typeof e == 'number') l += e;
  else if (typeof e == 'object')
    if (Array.isArray(e)) {
      var r = e.length;
      for (t = 0; t < r; t++) e[t] && (a = cv(e[t])) && (l && (l += ' '), (l += a));
    } else for (a in e) e[a] && (l && (l += ' '), (l += a));
  return l;
}
function mv() {
  for (var e, t, a = 0, l = '', r = arguments.length; a < r; a++)
    (e = arguments[a]) && (t = cv(e)) && (l && (l += ' '), (l += t));
  return l;
}
var gw = (e, t) => {
    let a = new Array(e.length + t.length);
    for (let l = 0; l < e.length; l++) a[l] = e[l];
    for (let l = 0; l < t.length; l++) a[e.length + l] = t[l];
    return a;
  },
  yw = (e, t) => ({ classGroupId: e, validator: t }),
  vv = (e = new Map(), t = null, a) => ({ nextPart: e, validators: t, classGroupId: a });
var pv = [],
  xw = 'arbitrary..',
  vw = (e) => {
    let t = Sw(e),
      { conflictingClassGroups: a, conflictingClassGroupModifiers: l } = e;
    return {
      getClassGroupId: (n) => {
        if (n.startsWith('[') && n.endsWith(']')) return Lw(n);
        let u = n.split('-'),
          i = u[0] === '' && u.length > 1 ? 1 : 0;
        return Lv(u, i, t);
      },
      getConflictingClassGroupIds: (n, u) => {
        if (u) {
          let i = l[n],
            s = a[n];
          return i ? (s ? gw(s, i) : i) : s || pv;
        }
        return a[n] || pv;
      },
    };
  },
  Lv = (e, t, a) => {
    if (e.length - t === 0) return a.classGroupId;
    let r = e[t],
      o = a.nextPart.get(r);
    if (o) {
      let s = Lv(e, t + 1, o);
      if (s) return s;
    }
    let n = a.validators;
    if (n === null) return;
    let u = t === 0 ? e.join('-') : e.slice(t).join('-'),
      i = n.length;
    for (let s = 0; s < i; s++) {
      let f = n[s];
      if (f.validator(u)) return f.classGroupId;
    }
  },
  Lw = (e) =>
    e.slice(1, -1).indexOf(':') === -1
      ? void 0
      : (() => {
          let t = e.slice(1, -1),
            a = t.indexOf(':'),
            l = t.slice(0, a);
          return l ? xw + l : void 0;
        })(),
  Sw = (e) => {
    let { theme: t, classGroups: a } = e;
    return bw(a, t);
  },
  bw = (e, t) => {
    let a = vv();
    for (let l in e) {
      let r = e[l];
      pc(r, a, l, t);
    }
    return a;
  },
  pc = (e, t, a, l) => {
    let r = e.length;
    for (let o = 0; o < r; o++) {
      let n = e[o];
      Cw(n, t, a, l);
    }
  },
  Cw = (e, t, a, l) => {
    if (typeof e == 'string') {
      ww(e, t, a);
      return;
    }
    if (typeof e == 'function') {
      Rw(e, t, a, l);
      return;
    }
    Iw(e, t, a, l);
  },
  ww = (e, t, a) => {
    let l = e === '' ? t : Sv(t, e);
    l.classGroupId = a;
  },
  Rw = (e, t, a, l) => {
    if (Ew(e)) {
      pc(e(l), t, a, l);
      return;
    }
    t.validators === null && (t.validators = []), t.validators.push(yw(a, e));
  },
  Iw = (e, t, a, l) => {
    let r = Object.entries(e),
      o = r.length;
    for (let n = 0; n < o; n++) {
      let [u, i] = r[n];
      pc(i, Sv(t, u), a, l);
    }
  },
  Sv = (e, t) => {
    let a = e,
      l = t.split('-'),
      r = l.length;
    for (let o = 0; o < r; o++) {
      let n = l[o],
        u = a.nextPart.get(n);
      u || ((u = vv()), a.nextPart.set(n, u)), (a = u);
    }
    return a;
  },
  Ew = (e) => 'isThemeGetter' in e && e.isThemeGetter === !0,
  Aw = (e) => {
    if (e < 1) return { get: () => {}, set: () => {} };
    let t = 0,
      a = Object.create(null),
      l = Object.create(null),
      r = (o, n) => {
        (a[o] = n), t++, t > e && ((t = 0), (l = a), (a = Object.create(null)));
      };
    return {
      get(o) {
        let n = a[o];
        if (n !== void 0) return n;
        if ((n = l[o]) !== void 0) return r(o, n), n;
      },
      set(o, n) {
        o in a ? (a[o] = n) : r(o, n);
      },
    };
  };
var Tw = [],
  hv = (e, t, a, l, r) => ({
    modifiers: e,
    hasImportantModifier: t,
    baseClassName: a,
    maybePostfixModifierPosition: l,
    isExternal: r,
  }),
  Mw = (e) => {
    let { prefix: t, experimentalParseClassName: a } = e,
      l = (r) => {
        let o = [],
          n = 0,
          u = 0,
          i = 0,
          s,
          f = r.length;
        for (let v = 0; v < f; v++) {
          let L = r[v];
          if (n === 0 && u === 0) {
            if (L === ':') {
              o.push(r.slice(i, v)), (i = v + 1);
              continue;
            }
            if (L === '/') {
              s = v;
              continue;
            }
          }
          L === '[' ? n++ : L === ']' ? n-- : L === '(' ? u++ : L === ')' && u--;
        }
        let d = o.length === 0 ? r : r.slice(i),
          p = d,
          h = !1;
        d.endsWith('!')
          ? ((p = d.slice(0, -1)), (h = !0))
          : d.startsWith('!') && ((p = d.slice(1)), (h = !0));
        let x = s && s > i ? s - i : void 0;
        return hv(o, h, p, x);
      };
    if (t) {
      let r = t + ':',
        o = l;
      l = (n) => (n.startsWith(r) ? o(n.slice(r.length)) : hv(Tw, !1, n, void 0, !0));
    }
    if (a) {
      let r = l;
      l = (o) => a({ className: o, parseClassName: r });
    }
    return l;
  },
  Dw = (e) => {
    let t = new Map();
    return (
      e.orderSensitiveModifiers.forEach((a, l) => {
        t.set(a, 1e6 + l);
      }),
      (a) => {
        let l = [],
          r = [];
        for (let o = 0; o < a.length; o++) {
          let n = a[o],
            u = n[0] === '[',
            i = t.has(n);
          u || i ? (r.length > 0 && (r.sort(), l.push(...r), (r = [])), l.push(n)) : r.push(n);
        }
        return r.length > 0 && (r.sort(), l.push(...r)), l;
      }
    );
  },
  kw = (e) => ({
    cache: Aw(e.cacheSize),
    parseClassName: Mw(e),
    sortModifiers: Dw(e),
    postfixLookupClassGroupIds: Bw(e),
    ...vw(e),
  }),
  Bw = (e) => {
    let t = Object.create(null),
      a = e.postfixLookupClassGroups;
    if (a) for (let l = 0; l < a.length; l++) t[a[l]] = !0;
    return t;
  },
  Ow = /\s+/,
  Uw = (e, t) => {
    let {
        parseClassName: a,
        getClassGroupId: l,
        getConflictingClassGroupIds: r,
        sortModifiers: o,
        postfixLookupClassGroupIds: n,
      } = t,
      u = [],
      i = e.trim().split(Ow),
      s = '';
    for (let f = i.length - 1; f >= 0; f -= 1) {
      let d = i[f],
        {
          isExternal: p,
          modifiers: h,
          hasImportantModifier: x,
          baseClassName: v,
          maybePostfixModifierPosition: L,
        } = a(d);
      if (p) {
        s = d + (s.length > 0 ? ' ' + s : s);
        continue;
      }
      let c = !!L,
        m;
      if (c) {
        let C = v.substring(0, L);
        m = l(C);
        let b = m && n[m] ? l(v) : void 0;
        b && b !== m && ((m = b), (c = !1));
      } else m = l(v);
      if (!m) {
        if (!c) {
          s = d + (s.length > 0 ? ' ' + s : s);
          continue;
        }
        if (((m = l(v)), !m)) {
          s = d + (s.length > 0 ? ' ' + s : s);
          continue;
        }
        c = !1;
      }
      let g = h.length === 0 ? '' : h.length === 1 ? h[0] : o(h).join(':'),
        y = x ? g + '!' : g,
        w = y + m;
      if (u.indexOf(w) > -1) continue;
      u.push(w);
      let U = r(m, c);
      for (let C = 0; C < U.length; ++C) {
        let b = U[C];
        u.push(y + b);
      }
      s = d + (s.length > 0 ? ' ' + s : s);
    }
    return s;
  },
  Hw = (...e) => {
    let t = 0,
      a,
      l,
      r = '';
    for (; t < e.length; ) (a = e[t++]) && (l = bv(a)) && (r && (r += ' '), (r += l));
    return r;
  },
  bv = (e) => {
    if (typeof e == 'string') return e;
    let t,
      a = '';
    for (let l = 0; l < e.length; l++) e[l] && (t = bv(e[l])) && (a && (a += ' '), (a += t));
    return a;
  },
  Nw = (e, ...t) => {
    let a,
      l,
      r,
      o,
      n = (i) => {
        let s = t.reduce((f, d) => d(f), e());
        return (a = kw(s)), (l = a.cache.get), (r = a.cache.set), (o = u), u(i);
      },
      u = (i) => {
        let s = l(i);
        if (s) return s;
        let f = Uw(i, a);
        return r(i, f), f;
      };
    return (o = n), (...i) => o(Hw(...i));
  },
  Pw = [],
  Oe = (e) => {
    let t = (a) => a[e] || Pw;
    return (t.isThemeGetter = !0), t;
  },
  Cv = /^\[(?:(\w[\w-]*):)?(.+)\]$/i,
  wv = /^\((?:(\w[\w-]*):)?(.+)\)$/i,
  _w = /^\d+(?:\.\d+)?\/\d+(?:\.\d+)?$/,
  zw = /^(\d+(\.\d+)?)?(xs|sm|md|lg|xl)$/,
  Fw =
    /\d+(%|px|r?em|[sdl]?v([hwib]|min|max)|pt|pc|in|cm|mm|cap|ch|ex|r?lh|cq(w|h|i|b|min|max))|\b(calc|min|max|clamp)\(.+\)|^0$/,
  qw = /^(rgba?|hsla?|hwb|(ok)?(lab|lch)|color-mix)\(.+\)$/,
  Gw = /^(inset_)?-?((\d+)?\.?(\d+)[a-z]+|0)_-?((\d+)?\.?(\d+)[a-z]+|0)/,
  Vw =
    /^(url|image|image-set|cross-fade|element|(repeating-)?(linear|radial|conic)-gradient)\(.+\)$/,
  vl = (e) => _w.test(e),
  G = (e) => !!e && !Number.isNaN(Number(e)),
  fa = (e) => !!e && Number.isInteger(Number(e)),
  mc = (e) => e.endsWith('%') && G(e.slice(0, -1)),
  Na = (e) => zw.test(e),
  Rv = () => !0,
  jw = (e) => Fw.test(e) && !qw.test(e),
  hc = () => !1,
  Xw = (e) => Gw.test(e),
  Yw = (e) => Vw.test(e),
  Kw = (e) => !A(e) && !T(e),
  Qw = (e) =>
    e.startsWith('@container') &&
    ((e[10] === '/' && e[11] !== void 0) ||
      (e[11] === 's' && e[16] !== void 0 && e.startsWith('-size/', 10)) ||
      (e[11] === 'n' && e[18] !== void 0 && e.startsWith('-normal/', 10))),
  Zw = (e) => Ll(e, Av, hc),
  A = (e) => Cv.test(e),
  Yl = (e) => Ll(e, Tv, jw),
  gv = (e) => Ll(e, r1, G),
  Ww = (e) => Ll(e, Dv, Rv),
  Jw = (e) => Ll(e, Mv, hc),
  yv = (e) => Ll(e, Iv, hc),
  $w = (e) => Ll(e, Ev, Yw),
  Ui = (e) => Ll(e, kv, Xw),
  T = (e) => wv.test(e),
  An = (e) => Kl(e, Tv),
  e1 = (e) => Kl(e, Mv),
  xv = (e) => Kl(e, Iv),
  t1 = (e) => Kl(e, Av),
  a1 = (e) => Kl(e, Ev),
  Hi = (e) => Kl(e, kv, !0),
  l1 = (e) => Kl(e, Dv, !0),
  Ll = (e, t, a) => {
    let l = Cv.exec(e);
    return l ? (l[1] ? t(l[1]) : a(l[2])) : !1;
  },
  Kl = (e, t, a = !1) => {
    let l = wv.exec(e);
    return l ? (l[1] ? t(l[1]) : a) : !1;
  },
  Iv = (e) => e === 'position' || e === 'percentage',
  Ev = (e) => e === 'image' || e === 'url',
  Av = (e) => e === 'length' || e === 'size' || e === 'bg-size',
  Tv = (e) => e === 'length',
  r1 = (e) => e === 'number',
  Mv = (e) => e === 'family-name',
  Dv = (e) => e === 'number' || e === 'weight',
  kv = (e) => e === 'shadow';
var o1 = () => {
  let e = Oe('color'),
    t = Oe('font'),
    a = Oe('text'),
    l = Oe('font-weight'),
    r = Oe('tracking'),
    o = Oe('leading'),
    n = Oe('breakpoint'),
    u = Oe('container'),
    i = Oe('spacing'),
    s = Oe('radius'),
    f = Oe('shadow'),
    d = Oe('inset-shadow'),
    p = Oe('text-shadow'),
    h = Oe('drop-shadow'),
    x = Oe('blur'),
    v = Oe('perspective'),
    L = Oe('aspect'),
    c = Oe('ease'),
    m = Oe('animate'),
    g = () => ['auto', 'avoid', 'all', 'avoid-page', 'page', 'left', 'right', 'column'],
    y = () => [
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
    w = () => [...y(), T, A],
    U = () => ['auto', 'hidden', 'clip', 'visible', 'scroll'],
    C = () => ['auto', 'contain', 'none'],
    b = () => [T, A, i],
    M = () => [vl, 'full', 'auto', ...b()],
    _ = () => [fa, 'none', 'subgrid', T, A],
    Ue = () => ['auto', { span: ['full', fa, T, A] }, fa, T, A],
    ot = () => [fa, 'auto', T, A],
    Gt = () => ['auto', 'min', 'max', 'fr', T, A],
    Qe = () => [
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
    be = () => ['start', 'end', 'center', 'stretch', 'center-safe', 'end-safe'],
    j = () => ['auto', ...b()],
    Me = () => [
      vl,
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
      ...b(),
    ],
    xt = () => [vl, 'screen', 'full', 'dvw', 'lvw', 'svw', 'min', 'max', 'fit', ...b()],
    ze = () => [vl, 'screen', 'full', 'lh', 'dvh', 'lvh', 'svh', 'min', 'max', 'fit', ...b()],
    B = () => [e, T, A],
    Mt = () => [...y(), xv, yv, { position: [T, A] }],
    Vt = () => ['no-repeat', { repeat: ['', 'x', 'y', 'space', 'round'] }],
    ma = () => ['auto', 'cover', 'contain', t1, Zw, { size: [T, A] }],
    ee = () => [mc, An, Yl],
    V = () => ['', 'none', 'full', s, T, A],
    X = () => ['', G, An, Yl],
    Fe = () => ['solid', 'dashed', 'dotted', 'double'],
    jt = () => [
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
    F = () => [G, mc, xv, yv],
    Fa = () => ['', 'none', x, T, A],
    pa = () => ['none', G, T, A],
    Jt = () => ['none', G, T, A],
    ha = () => [G, T, A],
    qa = () => [vl, 'full', ...b()];
  return {
    cacheSize: 500,
    theme: {
      animate: ['spin', 'ping', 'pulse', 'bounce'],
      aspect: ['video'],
      blur: [Na],
      breakpoint: [Na],
      color: [Rv],
      container: [Na],
      'drop-shadow': [Na],
      ease: ['in', 'out', 'in-out'],
      font: [Kw],
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
      'inset-shadow': [Na],
      leading: ['none', 'tight', 'snug', 'normal', 'relaxed', 'loose'],
      perspective: ['dramatic', 'near', 'normal', 'midrange', 'distant', 'none'],
      radius: [Na],
      shadow: [Na],
      spacing: ['px', G],
      text: [Na],
      'text-shadow': [Na],
      tracking: ['tighter', 'tight', 'normal', 'wide', 'wider', 'widest'],
    },
    classGroups: {
      aspect: [{ aspect: ['auto', 'square', vl, A, T, L] }],
      container: ['container'],
      'container-type': [{ '@container': ['', 'normal', 'size', T, A] }],
      'container-named': [Qw],
      columns: [{ columns: [G, A, T, u] }],
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
      'object-position': [{ object: w() }],
      overflow: [{ overflow: U() }],
      'overflow-x': [{ 'overflow-x': U() }],
      'overflow-y': [{ 'overflow-y': U() }],
      overscroll: [{ overscroll: C() }],
      'overscroll-x': [{ 'overscroll-x': C() }],
      'overscroll-y': [{ 'overscroll-y': C() }],
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
      z: [{ z: [fa, 'auto', T, A] }],
      basis: [{ basis: [vl, 'full', 'auto', u, ...b()] }],
      'flex-direction': [{ flex: ['row', 'row-reverse', 'col', 'col-reverse'] }],
      'flex-wrap': [{ flex: ['nowrap', 'wrap', 'wrap-reverse'] }],
      flex: [{ flex: [G, vl, 'auto', 'initial', 'none', A] }],
      grow: [{ grow: ['', G, T, A] }],
      shrink: [{ shrink: ['', G, T, A] }],
      order: [{ order: [fa, 'first', 'last', 'none', T, A] }],
      'grid-cols': [{ 'grid-cols': _() }],
      'col-start-end': [{ col: Ue() }],
      'col-start': [{ 'col-start': ot() }],
      'col-end': [{ 'col-end': ot() }],
      'grid-rows': [{ 'grid-rows': _() }],
      'row-start-end': [{ row: Ue() }],
      'row-start': [{ 'row-start': ot() }],
      'row-end': [{ 'row-end': ot() }],
      'grid-flow': [{ 'grid-flow': ['row', 'col', 'dense', 'row-dense', 'col-dense'] }],
      'auto-cols': [{ 'auto-cols': Gt() }],
      'auto-rows': [{ 'auto-rows': Gt() }],
      gap: [{ gap: b() }],
      'gap-x': [{ 'gap-x': b() }],
      'gap-y': [{ 'gap-y': b() }],
      'justify-content': [{ justify: [...Qe(), 'normal'] }],
      'justify-items': [{ 'justify-items': [...be(), 'normal'] }],
      'justify-self': [{ 'justify-self': ['auto', ...be()] }],
      'align-content': [{ content: ['normal', ...Qe()] }],
      'align-items': [{ items: [...be(), { baseline: ['', 'last'] }] }],
      'align-self': [{ self: ['auto', ...be(), { baseline: ['', 'last'] }] }],
      'place-content': [{ 'place-content': Qe() }],
      'place-items': [{ 'place-items': [...be(), 'baseline'] }],
      'place-self': [{ 'place-self': ['auto', ...be()] }],
      p: [{ p: b() }],
      px: [{ px: b() }],
      py: [{ py: b() }],
      ps: [{ ps: b() }],
      pe: [{ pe: b() }],
      pbs: [{ pbs: b() }],
      pbe: [{ pbe: b() }],
      pt: [{ pt: b() }],
      pr: [{ pr: b() }],
      pb: [{ pb: b() }],
      pl: [{ pl: b() }],
      m: [{ m: j() }],
      mx: [{ mx: j() }],
      my: [{ my: j() }],
      ms: [{ ms: j() }],
      me: [{ me: j() }],
      mbs: [{ mbs: j() }],
      mbe: [{ mbe: j() }],
      mt: [{ mt: j() }],
      mr: [{ mr: j() }],
      mb: [{ mb: j() }],
      ml: [{ ml: j() }],
      'space-x': [{ 'space-x': b() }],
      'space-x-reverse': ['space-x-reverse'],
      'space-y': [{ 'space-y': b() }],
      'space-y-reverse': ['space-y-reverse'],
      size: [{ size: Me() }],
      'inline-size': [{ inline: ['auto', ...xt()] }],
      'min-inline-size': [{ 'min-inline': ['auto', ...xt()] }],
      'max-inline-size': [{ 'max-inline': ['none', ...xt()] }],
      'block-size': [{ block: ['auto', ...ze()] }],
      'min-block-size': [{ 'min-block': ['auto', ...ze()] }],
      'max-block-size': [{ 'max-block': ['none', ...ze()] }],
      w: [{ w: [u, 'screen', ...Me()] }],
      'min-w': [{ 'min-w': [u, 'screen', 'none', ...Me()] }],
      'max-w': [{ 'max-w': [u, 'screen', 'none', 'prose', { screen: [n] }, ...Me()] }],
      h: [{ h: ['screen', 'lh', ...Me()] }],
      'min-h': [{ 'min-h': ['screen', 'lh', 'none', ...Me()] }],
      'max-h': [{ 'max-h': ['screen', 'lh', ...Me()] }],
      'font-size': [{ text: ['base', a, An, Yl] }],
      'font-smoothing': ['antialiased', 'subpixel-antialiased'],
      'font-style': ['italic', 'not-italic'],
      'font-weight': [{ font: [l, l1, Ww] }],
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
            mc,
            A,
          ],
        },
      ],
      'font-family': [{ font: [e1, Jw, t] }],
      'font-features': [{ 'font-features': [A] }],
      'fvn-normal': ['normal-nums'],
      'fvn-ordinal': ['ordinal'],
      'fvn-slashed-zero': ['slashed-zero'],
      'fvn-figure': ['lining-nums', 'oldstyle-nums'],
      'fvn-spacing': ['proportional-nums', 'tabular-nums'],
      'fvn-fraction': ['diagonal-fractions', 'stacked-fractions'],
      tracking: [{ tracking: [r, T, A] }],
      'line-clamp': [{ 'line-clamp': [G, 'none', T, gv] }],
      leading: [{ leading: [o, ...b()] }],
      'list-image': [{ 'list-image': ['none', T, A] }],
      'list-style-position': [{ list: ['inside', 'outside'] }],
      'list-style-type': [{ list: ['disc', 'decimal', 'none', T, A] }],
      'text-alignment': [{ text: ['left', 'center', 'right', 'justify', 'start', 'end'] }],
      'placeholder-color': [{ placeholder: B() }],
      'text-color': [{ text: B() }],
      'text-decoration': ['underline', 'overline', 'line-through', 'no-underline'],
      'text-decoration-style': [{ decoration: [...Fe(), 'wavy'] }],
      'text-decoration-thickness': [{ decoration: [G, 'from-font', 'auto', T, Yl] }],
      'text-decoration-color': [{ decoration: B() }],
      'underline-offset': [{ 'underline-offset': [G, 'auto', T, A] }],
      'text-transform': ['uppercase', 'lowercase', 'capitalize', 'normal-case'],
      'text-overflow': ['truncate', 'text-ellipsis', 'text-clip'],
      'text-wrap': [{ text: ['wrap', 'nowrap', 'balance', 'pretty'] }],
      indent: [{ indent: b() }],
      'tab-size': [{ tab: [fa, T, A] }],
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
            A,
          ],
        },
      ],
      whitespace: [
        { whitespace: ['normal', 'nowrap', 'pre', 'pre-line', 'pre-wrap', 'break-spaces'] },
      ],
      break: [{ break: ['normal', 'words', 'all', 'keep'] }],
      wrap: [{ wrap: ['break-word', 'anywhere', 'normal'] }],
      hyphens: [{ hyphens: ['none', 'manual', 'auto'] }],
      content: [{ content: ['none', T, A] }],
      'bg-attachment': [{ bg: ['fixed', 'local', 'scroll'] }],
      'bg-clip': [{ 'bg-clip': ['border', 'padding', 'content', 'text'] }],
      'bg-origin': [{ 'bg-origin': ['border', 'padding', 'content'] }],
      'bg-position': [{ bg: Mt() }],
      'bg-repeat': [{ bg: Vt() }],
      'bg-size': [{ bg: ma() }],
      'bg-image': [
        {
          bg: [
            'none',
            {
              linear: [{ to: ['t', 'tr', 'r', 'br', 'b', 'bl', 'l', 'tl'] }, fa, T, A],
              radial: ['', T, A],
              conic: [fa, T, A],
            },
            a1,
            $w,
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
      rounded: [{ rounded: V() }],
      'rounded-s': [{ 'rounded-s': V() }],
      'rounded-e': [{ 'rounded-e': V() }],
      'rounded-t': [{ 'rounded-t': V() }],
      'rounded-r': [{ 'rounded-r': V() }],
      'rounded-b': [{ 'rounded-b': V() }],
      'rounded-l': [{ 'rounded-l': V() }],
      'rounded-ss': [{ 'rounded-ss': V() }],
      'rounded-se': [{ 'rounded-se': V() }],
      'rounded-ee': [{ 'rounded-ee': V() }],
      'rounded-es': [{ 'rounded-es': V() }],
      'rounded-tl': [{ 'rounded-tl': V() }],
      'rounded-tr': [{ 'rounded-tr': V() }],
      'rounded-br': [{ 'rounded-br': V() }],
      'rounded-bl': [{ 'rounded-bl': V() }],
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
      'border-style': [{ border: [...Fe(), 'hidden', 'none'] }],
      'divide-style': [{ divide: [...Fe(), 'hidden', 'none'] }],
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
      'outline-style': [{ outline: [...Fe(), 'none', 'hidden'] }],
      'outline-offset': [{ 'outline-offset': [G, T, A] }],
      'outline-w': [{ outline: ['', G, An, Yl] }],
      'outline-color': [{ outline: B() }],
      shadow: [{ shadow: ['', 'none', f, Hi, Ui] }],
      'shadow-color': [{ shadow: B() }],
      'inset-shadow': [{ 'inset-shadow': ['none', d, Hi, Ui] }],
      'inset-shadow-color': [{ 'inset-shadow': B() }],
      'ring-w': [{ ring: X() }],
      'ring-w-inset': ['ring-inset'],
      'ring-color': [{ ring: B() }],
      'ring-offset-w': [{ 'ring-offset': [G, Yl] }],
      'ring-offset-color': [{ 'ring-offset': B() }],
      'inset-ring-w': [{ 'inset-ring': X() }],
      'inset-ring-color': [{ 'inset-ring': B() }],
      'text-shadow': [{ 'text-shadow': ['none', p, Hi, Ui] }],
      'text-shadow-color': [{ 'text-shadow': B() }],
      opacity: [{ opacity: [G, T, A] }],
      'mix-blend': [{ 'mix-blend': [...jt(), 'plus-darker', 'plus-lighter'] }],
      'bg-blend': [{ 'bg-blend': jt() }],
      'mask-clip': [
        { 'mask-clip': ['border', 'padding', 'content', 'fill', 'stroke', 'view'] },
        'mask-no-clip',
      ],
      'mask-composite': [{ mask: ['add', 'subtract', 'intersect', 'exclude'] }],
      'mask-image-linear-pos': [{ 'mask-linear': [G] }],
      'mask-image-linear-from-pos': [{ 'mask-linear-from': F() }],
      'mask-image-linear-to-pos': [{ 'mask-linear-to': F() }],
      'mask-image-linear-from-color': [{ 'mask-linear-from': B() }],
      'mask-image-linear-to-color': [{ 'mask-linear-to': B() }],
      'mask-image-t-from-pos': [{ 'mask-t-from': F() }],
      'mask-image-t-to-pos': [{ 'mask-t-to': F() }],
      'mask-image-t-from-color': [{ 'mask-t-from': B() }],
      'mask-image-t-to-color': [{ 'mask-t-to': B() }],
      'mask-image-r-from-pos': [{ 'mask-r-from': F() }],
      'mask-image-r-to-pos': [{ 'mask-r-to': F() }],
      'mask-image-r-from-color': [{ 'mask-r-from': B() }],
      'mask-image-r-to-color': [{ 'mask-r-to': B() }],
      'mask-image-b-from-pos': [{ 'mask-b-from': F() }],
      'mask-image-b-to-pos': [{ 'mask-b-to': F() }],
      'mask-image-b-from-color': [{ 'mask-b-from': B() }],
      'mask-image-b-to-color': [{ 'mask-b-to': B() }],
      'mask-image-l-from-pos': [{ 'mask-l-from': F() }],
      'mask-image-l-to-pos': [{ 'mask-l-to': F() }],
      'mask-image-l-from-color': [{ 'mask-l-from': B() }],
      'mask-image-l-to-color': [{ 'mask-l-to': B() }],
      'mask-image-x-from-pos': [{ 'mask-x-from': F() }],
      'mask-image-x-to-pos': [{ 'mask-x-to': F() }],
      'mask-image-x-from-color': [{ 'mask-x-from': B() }],
      'mask-image-x-to-color': [{ 'mask-x-to': B() }],
      'mask-image-y-from-pos': [{ 'mask-y-from': F() }],
      'mask-image-y-to-pos': [{ 'mask-y-to': F() }],
      'mask-image-y-from-color': [{ 'mask-y-from': B() }],
      'mask-image-y-to-color': [{ 'mask-y-to': B() }],
      'mask-image-radial': [{ 'mask-radial': [T, A] }],
      'mask-image-radial-from-pos': [{ 'mask-radial-from': F() }],
      'mask-image-radial-to-pos': [{ 'mask-radial-to': F() }],
      'mask-image-radial-from-color': [{ 'mask-radial-from': B() }],
      'mask-image-radial-to-color': [{ 'mask-radial-to': B() }],
      'mask-image-radial-shape': [{ 'mask-radial': ['circle', 'ellipse'] }],
      'mask-image-radial-size': [
        { 'mask-radial': [{ closest: ['side', 'corner'], farthest: ['side', 'corner'] }] },
      ],
      'mask-image-radial-pos': [{ 'mask-radial-at': y() }],
      'mask-image-conic-pos': [{ 'mask-conic': [G] }],
      'mask-image-conic-from-pos': [{ 'mask-conic-from': F() }],
      'mask-image-conic-to-pos': [{ 'mask-conic-to': F() }],
      'mask-image-conic-from-color': [{ 'mask-conic-from': B() }],
      'mask-image-conic-to-color': [{ 'mask-conic-to': B() }],
      'mask-mode': [{ mask: ['alpha', 'luminance', 'match'] }],
      'mask-origin': [
        { 'mask-origin': ['border', 'padding', 'content', 'fill', 'stroke', 'view'] },
      ],
      'mask-position': [{ mask: Mt() }],
      'mask-repeat': [{ mask: Vt() }],
      'mask-size': [{ mask: ma() }],
      'mask-type': [{ 'mask-type': ['alpha', 'luminance'] }],
      'mask-image': [{ mask: ['none', T, A] }],
      filter: [{ filter: ['', 'none', T, A] }],
      blur: [{ blur: Fa() }],
      brightness: [{ brightness: [G, T, A] }],
      contrast: [{ contrast: [G, T, A] }],
      'drop-shadow': [{ 'drop-shadow': ['', 'none', h, Hi, Ui] }],
      'drop-shadow-color': [{ 'drop-shadow': B() }],
      grayscale: [{ grayscale: ['', G, T, A] }],
      'hue-rotate': [{ 'hue-rotate': [G, T, A] }],
      invert: [{ invert: ['', G, T, A] }],
      saturate: [{ saturate: [G, T, A] }],
      sepia: [{ sepia: ['', G, T, A] }],
      'backdrop-filter': [{ 'backdrop-filter': ['', 'none', T, A] }],
      'backdrop-blur': [{ 'backdrop-blur': Fa() }],
      'backdrop-brightness': [{ 'backdrop-brightness': [G, T, A] }],
      'backdrop-contrast': [{ 'backdrop-contrast': [G, T, A] }],
      'backdrop-grayscale': [{ 'backdrop-grayscale': ['', G, T, A] }],
      'backdrop-hue-rotate': [{ 'backdrop-hue-rotate': [G, T, A] }],
      'backdrop-invert': [{ 'backdrop-invert': ['', G, T, A] }],
      'backdrop-opacity': [{ 'backdrop-opacity': [G, T, A] }],
      'backdrop-saturate': [{ 'backdrop-saturate': [G, T, A] }],
      'backdrop-sepia': [{ 'backdrop-sepia': ['', G, T, A] }],
      'border-collapse': [{ border: ['collapse', 'separate'] }],
      'border-spacing': [{ 'border-spacing': b() }],
      'border-spacing-x': [{ 'border-spacing-x': b() }],
      'border-spacing-y': [{ 'border-spacing-y': b() }],
      'table-layout': [{ table: ['auto', 'fixed'] }],
      caption: [{ caption: ['top', 'bottom'] }],
      transition: [
        { transition: ['', 'all', 'colors', 'opacity', 'shadow', 'transform', 'none', T, A] },
      ],
      'transition-behavior': [{ transition: ['normal', 'discrete'] }],
      duration: [{ duration: [G, 'initial', T, A] }],
      ease: [{ ease: ['linear', 'initial', c, T, A] }],
      delay: [{ delay: [G, T, A] }],
      animate: [{ animate: ['none', m, T, A] }],
      backface: [{ backface: ['hidden', 'visible'] }],
      perspective: [{ perspective: [v, T, A] }],
      'perspective-origin': [{ 'perspective-origin': w() }],
      rotate: [{ rotate: pa() }],
      'rotate-x': [{ 'rotate-x': pa() }],
      'rotate-y': [{ 'rotate-y': pa() }],
      'rotate-z': [{ 'rotate-z': pa() }],
      scale: [{ scale: Jt() }],
      'scale-x': [{ 'scale-x': Jt() }],
      'scale-y': [{ 'scale-y': Jt() }],
      'scale-z': [{ 'scale-z': Jt() }],
      'scale-3d': ['scale-3d'],
      skew: [{ skew: ha() }],
      'skew-x': [{ 'skew-x': ha() }],
      'skew-y': [{ 'skew-y': ha() }],
      transform: [{ transform: [T, A, '', 'none', 'gpu', 'cpu'] }],
      'transform-origin': [{ origin: w() }],
      'transform-style': [{ transform: ['3d', 'flat'] }],
      translate: [{ translate: qa() }],
      'translate-x': [{ 'translate-x': qa() }],
      'translate-y': [{ 'translate-y': qa() }],
      'translate-z': [{ 'translate-z': qa() }],
      'translate-none': ['translate-none'],
      zoom: [{ zoom: [fa, T, A] }],
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
            A,
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
      'scroll-m': [{ 'scroll-m': b() }],
      'scroll-mx': [{ 'scroll-mx': b() }],
      'scroll-my': [{ 'scroll-my': b() }],
      'scroll-ms': [{ 'scroll-ms': b() }],
      'scroll-me': [{ 'scroll-me': b() }],
      'scroll-mbs': [{ 'scroll-mbs': b() }],
      'scroll-mbe': [{ 'scroll-mbe': b() }],
      'scroll-mt': [{ 'scroll-mt': b() }],
      'scroll-mr': [{ 'scroll-mr': b() }],
      'scroll-mb': [{ 'scroll-mb': b() }],
      'scroll-ml': [{ 'scroll-ml': b() }],
      'scroll-p': [{ 'scroll-p': b() }],
      'scroll-px': [{ 'scroll-px': b() }],
      'scroll-py': [{ 'scroll-py': b() }],
      'scroll-ps': [{ 'scroll-ps': b() }],
      'scroll-pe': [{ 'scroll-pe': b() }],
      'scroll-pbs': [{ 'scroll-pbs': b() }],
      'scroll-pbe': [{ 'scroll-pbe': b() }],
      'scroll-pt': [{ 'scroll-pt': b() }],
      'scroll-pr': [{ 'scroll-pr': b() }],
      'scroll-pb': [{ 'scroll-pb': b() }],
      'scroll-pl': [{ 'scroll-pl': b() }],
      'snap-align': [{ snap: ['start', 'end', 'center', 'align-none'] }],
      'snap-stop': [{ snap: ['normal', 'always'] }],
      'snap-type': [{ snap: ['none', 'x', 'y', 'both'] }],
      'snap-strictness': [{ snap: ['mandatory', 'proximity'] }],
      touch: [{ touch: ['auto', 'none', 'manipulation'] }],
      'touch-x': [{ 'touch-pan': ['x', 'left', 'right'] }],
      'touch-y': [{ 'touch-pan': ['y', 'up', 'down'] }],
      'touch-pz': ['touch-pinch-zoom'],
      select: [{ select: ['none', 'text', 'all', 'auto'] }],
      'will-change': [{ 'will-change': ['auto', 'scroll', 'contents', 'transform', T, A] }],
      fill: [{ fill: ['none', ...B()] }],
      'stroke-w': [{ stroke: [G, An, Yl, gv] }],
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
var Bv = Nw(o1);
function O(...e) {
  return Bv(mv(e));
}
var Pa = E(Q());
function Ov({ columns: e = 4, rows: t = 6, className: a }) {
  return (0, Pa.jsxs)('div', {
    'aria-busy': 'true',
    'aria-label': 'Loading table',
    className: O('motion-safe:animate-pulse overflow-hidden border border-border bg-card', a),
    children: [
      (0, Pa.jsx)('div', {
        className: 'flex h-[34px] border-b border-border/40',
        children: Array.from({ length: e }, (l, r) =>
          (0, Pa.jsx)(
            'div',
            {
              className: 'flex flex-1 items-center px-4',
              children: (0, Pa.jsx)('div', { className: 'h-3 w-20 rounded bg-muted' }),
            },
            `head-${r}`
          )
        ),
      }),
      Array.from({ length: t }, (l, r) =>
        (0, Pa.jsx)(
          'div',
          {
            className: 'flex h-[34px] border-b border-border/40 last:border-0',
            children: Array.from({ length: e }, (o, n) =>
              (0, Pa.jsx)(
                'div',
                {
                  className: 'flex flex-1 items-center px-4',
                  children: (0, Pa.jsx)('div', {
                    className: O(
                      'h-3 rounded bg-muted',
                      n === 0 ? 'w-32' : n === e - 1 ? 'w-16' : 'w-24'
                    ),
                  }),
                },
                `cell-${r}-${n}`
              )
            ),
          },
          `row-${r}`
        )
      ),
    ],
  });
}
var lo = E(Q());
function ro({ variant: e = 'page', columns: t = 4, rows: a = 6 }) {
  return e === 'directory'
    ? (0, lo.jsx)(Ov, { columns: t, rows: a })
    : (0, lo.jsxs)('div', {
        'aria-busy': 'true',
        'aria-label': 'Loading',
        children: [(0, lo.jsx)('div', {}), (0, lo.jsx)('div', {})],
      });
}
var Uv = E(te());
function oo() {
  let e = (0, Uv.useContext)(cc);
  if (!e) throw new Error('useMeta must be used within MetaProvider');
  return e;
}
var tr = E(te());
var Pi = E(te());
var Hv = (e) => e.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase(),
  n1 = (e) =>
    e.replace(/^([A-Z])|[\s-_]+(\w)/g, (t, a, l) => (l ? l.toUpperCase() : a.toLowerCase())),
  gc = (e) => {
    let t = n1(e);
    return t.charAt(0).toUpperCase() + t.slice(1);
  },
  Ni = (...e) =>
    e
      .filter((t, a, l) => !!t && t.trim() !== '' && l.indexOf(t) === a)
      .join(' ')
      .trim(),
  Nv = (e) => {
    for (let t in e) if (t.startsWith('aria-') || t === 'role' || t === 'title') return !0;
  };
var Tn = E(te());
var Pv = {
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
var _v = (0, Tn.forwardRef)(
  (
    {
      color: e = 'currentColor',
      size: t = 24,
      strokeWidth: a = 2,
      absoluteStrokeWidth: l,
      className: r = '',
      children: o,
      iconNode: n,
      ...u
    },
    i
  ) =>
    (0, Tn.createElement)(
      'svg',
      {
        ref: i,
        ...Pv,
        width: t,
        height: t,
        stroke: e,
        strokeWidth: l ? (Number(a) * 24) / Number(t) : a,
        className: Ni('lucide', r),
        ...(!o && !Nv(u) && { 'aria-hidden': 'true' }),
        ...u,
      },
      [...n.map(([s, f]) => (0, Tn.createElement)(s, f)), ...(Array.isArray(o) ? o : [o])]
    )
);
var I = (e, t) => {
  let a = (0, Pi.forwardRef)(({ className: l, ...r }, o) =>
    (0, Pi.createElement)(_v, {
      ref: o,
      iconNode: t,
      className: Ni(`lucide-${Hv(gc(e))}`, `lucide-${e}`, l),
      ...r,
    })
  );
  return (a.displayName = gc(e)), a;
};
var u1 = [
    ['path', { d: 'm8 2 1.88 1.88', key: 'fmnt4t' }],
    ['path', { d: 'M14.12 3.88 16 2', key: 'qol33r' }],
    ['path', { d: 'M9 7.13v-1a3.003 3.003 0 1 1 6 0v1', key: 'd7y7pr' }],
    [
      'path',
      {
        d: 'M12 20c-3.3 0-6-2.7-6-6v-3a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v3c0 3.3-2.7 6-6 6',
        key: 'xs1cw7',
      },
    ],
    ['path', { d: 'M12 20v-9', key: '1qisl0' }],
    ['path', { d: 'M6.53 9C4.6 8.8 3 7.1 3 5', key: '32zzws' }],
    ['path', { d: 'M6 13H2', key: '82j7cp' }],
    ['path', { d: 'M3 21c0-2.1 1.7-3.9 3.8-4', key: '4p0ekp' }],
    ['path', { d: 'M20.97 5c0 2.1-1.6 3.8-3.5 4', key: '18gb23' }],
    ['path', { d: 'M22 13h-4', key: '1jl80f' }],
    ['path', { d: 'M17.2 17c2.1.1 3.8 1.9 3.8 4', key: 'k3fwyw' }],
  ],
  yc = I('bug', u1);
var i1 = [
    ['path', { d: 'M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z', key: '1b4qmf' }],
    ['path', { d: 'M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2', key: 'i71pzd' }],
    ['path', { d: 'M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2', key: '10jefs' }],
    ['path', { d: 'M10 6h4', key: '1itunk' }],
    ['path', { d: 'M10 10h4', key: 'tcdvrf' }],
    ['path', { d: 'M10 14h4', key: 'kelpxr' }],
    ['path', { d: 'M10 18h4', key: '1ulq68' }],
  ],
  xc = I('building-2', i1);
var s1 = [
    ['path', { d: 'M8 2v4', key: '1cmpym' }],
    ['path', { d: 'M16 2v4', key: '4m81vk' }],
    ['rect', { width: '18', height: '18', x: '3', y: '4', rx: '2', key: '1hopcy' }],
    ['path', { d: 'M3 10h18', key: '8toen8' }],
  ],
  vc = I('calendar', s1);
var d1 = [['path', { d: 'M20 6 9 17l-5-5', key: '1gmf2c' }]],
  Mn = I('check', d1);
var f1 = [['path', { d: 'm6 9 6 6 6-6', key: 'qrunsl' }]],
  Lc = I('chevron-down', f1);
var c1 = [['path', { d: 'm15 18-6-6 6-6', key: '1wnfg3' }]],
  Sc = I('chevron-left', c1);
var m1 = [['path', { d: 'm9 18 6-6-6-6', key: 'mthhwq' }]],
  bc = I('chevron-right', m1);
var p1 = [
    ['rect', { width: '14', height: '14', x: '8', y: '8', rx: '2', ry: '2', key: '17jyea' }],
    ['path', { d: 'M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2', key: 'zix9uf' }],
  ],
  Cc = I('copy', p1);
var h1 = [
    ['path', { d: 'M12 15V3', key: 'm9g1x1' }],
    ['path', { d: 'M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4', key: 'ih7n3h' }],
    ['path', { d: 'm7 10 5 5 5-5', key: 'brsn70' }],
  ],
  wc = I('download', h1);
var g1 = [
    ['circle', { cx: '12', cy: '12', r: '1', key: '41hilf' }],
    ['circle', { cx: '19', cy: '12', r: '1', key: '1wjl8i' }],
    ['circle', { cx: '5', cy: '12', r: '1', key: '1pcz8c' }],
  ],
  no = I('ellipsis', g1);
var y1 = [
    [
      'path',
      {
        d: 'M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49',
        key: 'ct8e1f',
      },
    ],
    ['path', { d: 'M14.084 14.158a3 3 0 0 1-4.242-4.242', key: '151rxh' }],
    [
      'path',
      {
        d: 'M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143',
        key: '13bj9a',
      },
    ],
    ['path', { d: 'm2 2 20 20', key: '1ooewy' }],
  ],
  Dn = I('eye-off', y1);
var x1 = [
    [
      'path',
      {
        d: 'M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0',
        key: '1nclc0',
      },
    ],
    ['circle', { cx: '12', cy: '12', r: '3', key: '1v7zrd' }],
  ],
  kn = I('eye', x1);
var v1 = [
    ['path', { d: 'M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z', key: '1rqfz7' }],
    ['path', { d: 'M14 2v4a2 2 0 0 0 2 2h4', key: 'tnqrlb' }],
    ['path', { d: 'M8 13h2', key: 'yr2amv' }],
    ['path', { d: 'M14 13h2', key: 'un5t4a' }],
    ['path', { d: 'M8 17h2', key: '2yhykz' }],
    ['path', { d: 'M14 17h2', key: '10kma7' }],
  ],
  Rc = I('file-spreadsheet', v1);
var L1 = [
    ['polyline', { points: '22 12 16 12 14 15 10 15 8 12 2 12', key: 'o97t9d' }],
    [
      'path',
      {
        d: 'M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z',
        key: 'oot6mr',
      },
    ],
  ],
  Ic = I('inbox', L1);
var S1 = [
    ['circle', { cx: '12', cy: '12', r: '10', key: '1mglay' }],
    ['path', { d: 'M12 16v-4', key: '1dtifu' }],
    ['path', { d: 'M12 8h.01', key: 'e9boi3' }],
  ],
  Ec = I('info', S1);
var b1 = [
    [
      'path',
      { d: 'm15.5 7.5 2.3 2.3a1 1 0 0 0 1.4 0l2.1-2.1a1 1 0 0 0 0-1.4L19 4', key: 'g0fldk' },
    ],
    ['path', { d: 'm21 2-9.6 9.6', key: '1j0ho8' }],
    ['circle', { cx: '7.5', cy: '15.5', r: '5.5', key: 'yqb3hr' }],
  ],
  Ac = I('key', b1);
var C1 = [
    ['path', { d: 'M9 17H7A5 5 0 0 1 7 7h2', key: '8i5ue5' }],
    ['path', { d: 'M15 7h2a5 5 0 1 1 0 10h-2', key: '1b9ql8' }],
    ['line', { x1: '8', x2: '16', y1: '12', y2: '12', key: '1jonct' }],
  ],
  Tc = I('link-2', C1);
var w1 = [['path', { d: 'M21 12a9 9 0 1 1-6.219-8.56', key: '13zald' }]],
  Sl = I('loader-circle', w1);
var R1 = [
    ['path', { d: 'm16 17 5-5-5-5', key: '1bji2h' }],
    ['path', { d: 'M21 12H9', key: 'dn1m92' }],
    ['path', { d: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4', key: '1uf3rs' }],
  ],
  Mc = I('log-out', R1);
var I1 = [
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
  Dc = I('megaphone', I1);
var E1 = [
    ['path', { d: 'M4 12h16', key: '1lakjw' }],
    ['path', { d: 'M4 18h16', key: '19g7jn' }],
    ['path', { d: 'M4 6h16', key: '1o0s65' }],
  ],
  kc = I('menu', E1);
var A1 = [
    [
      'path',
      {
        d: 'M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401',
        key: 'kfwtm',
      },
    ],
  ],
  Bc = I('moon', A1);
var T1 = [
    ['path', { d: 'M12 22v-5', key: '1ega77' }],
    ['path', { d: 'M9 8V2', key: '14iosj' }],
    ['path', { d: 'M15 8V2', key: '18g5xt' }],
    ['path', { d: 'M18 8v5a4 4 0 0 1-4 4h-4a4 4 0 0 1-4-4V8Z', key: 'osxo6l' }],
  ],
  Oc = I('plug', T1);
var M1 = [
    ['path', { d: 'M5 12h14', key: '1ays0h' }],
    ['path', { d: 'M12 5v14', key: 's699le' }],
  ],
  Uc = I('plus', M1);
var D1 = [
    ['path', { d: 'M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8', key: 'v9h5vc' }],
    ['path', { d: 'M21 3v5h-5', key: '1q7to0' }],
    ['path', { d: 'M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16', key: '3uifl3' }],
    ['path', { d: 'M8 16H3v5', key: '1cv678' }],
  ],
  Hc = I('refresh-cw', D1);
var k1 = [
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
  Nc = I('scroll-text', k1);
var B1 = [
    ['path', { d: 'm21 21-4.34-4.34', key: '14j7rj' }],
    ['circle', { cx: '11', cy: '11', r: '8', key: '4ej97u' }],
  ],
  Pc = I('search', B1);
var O1 = [
    [
      'path',
      {
        d: 'M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915',
        key: '1i5ecw',
      },
    ],
    ['circle', { cx: '12', cy: '12', r: '3', key: '1v7zrd' }],
  ],
  _c = I('settings', O1);
var U1 = [
    ['circle', { cx: '18', cy: '5', r: '3', key: 'gq8acd' }],
    ['circle', { cx: '6', cy: '12', r: '3', key: 'w7nqdw' }],
    ['circle', { cx: '18', cy: '19', r: '3', key: '1xt0gg' }],
    ['line', { x1: '8.59', x2: '15.42', y1: '13.51', y2: '17.49', key: '47mynk' }],
    ['line', { x1: '15.41', x2: '8.59', y1: '6.51', y2: '10.49', key: '1n3mei' }],
  ],
  zc = I('share-2', U1);
var H1 = [
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
  Fc = I('sun', H1);
var N1 = [
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
  qc = I('tags', N1);
var P1 = [
    ['path', { d: 'M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2', key: '975kel' }],
    ['circle', { cx: '12', cy: '7', r: '4', key: '17ys0d' }],
  ],
  Gc = I('user', P1);
var _1 = [
    ['path', { d: 'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2', key: '1yyitq' }],
    ['path', { d: 'M16 3.128a4 4 0 0 1 0 7.744', key: '16gr8j' }],
    ['path', { d: 'M22 21v-2a4 4 0 0 0-3-3.87', key: 'kshegd' }],
    ['circle', { cx: '9', cy: '7', r: '4', key: 'nufk8' }],
  ],
  Vc = I('users', _1);
var z1 = [
    [
      'path',
      {
        d: 'M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.106-3.105c.32-.322.863-.22.983.218a6 6 0 0 1-8.259 7.057l-7.91 7.91a1 1 0 0 1-2.999-3l7.91-7.91a6 6 0 0 1 7.057-8.259c.438.12.54.662.219.984z',
        key: '1ngwbx',
      },
    ],
  ],
  jc = I('wrench', z1);
var F1 = [
    ['path', { d: 'M18 6 6 18', key: '1bl5f8' }],
    ['path', { d: 'm6 6 12 12', key: 'd8bk6v' }],
  ],
  Xc = I('x', F1);
var jv = E(te());
var H = {
    gap: { xs: 'gap-1', sm: 'gap-1.5', md: 'gap-2', lg: 'gap-3', xl: 'gap-4' },
    gapX: { filterRow: 'gap-x-3', formColumns: 'gap-x-4' },
    gapY: { filterRow: 'gap-y-4', formColumns: 'gap-y-4' },
    inset: {
      canvas: 'px-6 py-4',
      panel: 'p-4',
      band: 'px-4 py-2',
      bandCompact: 'px-3 py-2',
      bandLg: 'px-4 py-3',
      tableCellX: 'px-4',
      footer: 'px-6 pt-3',
      sectionTop: 'pt-3',
      emptyState: 'px-6 py-16',
      navItemX: 'px-2.5',
      navItemY: 'py-1',
      navGroupLabel: 'px-2.5 py-1.5',
      sidebarX: 'px-2.5',
      headerX: 'px-3 md:px-4',
      listMessage: 'px-3 py-2',
      listHeading: 'px-3 py-1',
      listSectionY: 'py-1',
    },
    stack: { titleBlock: 'grid gap-1' },
    flex: {
      buttonGroup: 'flex flex-wrap items-center gap-2',
      actionsRowEnd: 'flex flex-wrap items-center justify-end gap-2',
      headerActions: 'flex shrink-0 flex-wrap items-center gap-2',
      pageHeader: 'flex flex-wrap items-start justify-between gap-3',
      columnMd: 'flex flex-col gap-2',
      columnLg: 'flex flex-col gap-3',
      workspaceFlat: 'flex min-h-0 flex-1 flex-col gap-4',
      sectionStack: 'flex flex-col gap-3 border-t border-border pt-3 first:border-t-0 first:pt-0',
      footer:
        'flex shrink-0 flex-wrap items-center gap-3 border-0 border-t border-border bg-transparent',
      headerBar: 'relative flex h-full items-center gap-3',
      headerStart: 'relative z-[1] flex min-w-0 items-center',
      headerEnd: 'relative z-[1] flex min-w-0 items-center justify-end gap-2',
      headerSearchOverlay:
        'pointer-events-none absolute inset-x-0 top-0 flex h-12 items-center justify-center px-14 md:px-20',
      filterField: 'grid min-w-0 gap-2',
      statusBanner: 'flex items-center gap-2',
    },
    grid: {
      mainAside: 'grid min-h-0 min-w-0 w-full flex-1 gap-4',
      filterMatrix: 'grid gap-4 md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] md:items-end',
      campaignsFilterRow:
        'grid w-full grid-cols-2 gap-x-3 gap-y-2 sm:grid-cols-3 md:grid-cols-4 md:items-end',
      filterFormStack: 'grid w-full justify-items-start gap-4',
      sectionPanel:
        'grid gap-3 rounded-[8px] border border-border bg-card p-4 text-card-foreground',
    },
    opsDirectoryTable:
      'w-full border-collapse text-[13px] leading-[18px] [&_th]:sticky [&_th]:top-0 [&_th]:z-[2] [&_th]:h-[34px] [&_th]:max-h-[34px] [&_th]:bg-card [&_th]:px-4 [&_th]:py-0 [&_th]:text-[11px] [&_th]:font-semibold [&_th]:uppercase [&_th]:leading-[14px] [&_th]:tracking-normal [&_th]:text-muted-foreground [&_th]:shadow-sm [&_td]:h-[34px] [&_td]:max-h-[34px] [&_td]:px-4 [&_td]:py-0 [&_tbody_tr:nth-child(even)_td]:bg-muted/30 [&_tbody_tr:last-child_td]:border-b-0',
  },
  ie = {
    pageTitle: 'text-lg font-bold tracking-tight text-foreground',
    sectionTitle: 'text-[13px] font-semibold leading-[18px] text-foreground',
    panelTitle: 'text-[13px] font-semibold leading-[18px] text-foreground',
    body: 'text-[13px] leading-[18px] text-foreground',
    bodyMuted: 'text-[13px] leading-[18px] text-muted-foreground',
    label: 'text-[13px] font-medium leading-[18px] text-foreground',
    labelMuted: 'text-[13px] leading-[18px] text-muted-foreground',
    caption:
      'text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
    captionPlain: 'text-[11px] leading-[14px] text-muted-foreground',
    tableHeader:
      'text-[11px] font-semibold uppercase leading-[14px] tracking-normal text-muted-foreground',
    tableBody: 'text-[13px] leading-[18px] text-foreground',
    tooltip: 'text-[12px] leading-5',
    badge: 'text-xs leading-4',
    monoData: 'font-mono text-[12px] leading-5',
  },
  xT = H.inset.canvas,
  vT = H.flex.workspaceFlat,
  LT = H.flex.sectionStack,
  ST = `${H.flex.footer} ${H.inset.footer}`,
  bT = `${H.inset.sectionTop} ${ie.sectionTitle}`,
  q1 = `grid ${H.gap.xl}`,
  CT = H.stack.titleBlock,
  wT = `grid ${H.gap.xl}`,
  RT = H.flex.filterField;
var G1 = 'scrollbar-admin',
  Bn =
    'grid gap-2 rounded-[8px] border p-4 text-[13px] leading-[18px] shadow-none backdrop-blur-none',
  $e = {
    message: Bn,
    messageError: `${Bn} border-destructive/40 bg-destructive/10 text-destructive`,
    messageMuted: `${Bn} border-border/40 bg-muted/30 text-muted-foreground`,
    messageSuccess: `${Bn} border-admin-status-active/25 bg-admin-status-active/10 text-admin-positive`,
    messageWarning: `${Bn} border-admin-warn-border bg-admin-warn-bg text-admin-warn`,
    panel:
      'grid gap-3 rounded-[8px] border border-border bg-card p-4 text-card-foreground shadow-none',
    control:
      'inline-flex h-7 min-h-7 shrink-0 items-center justify-center gap-2 border border-border/40 px-3 text-[13px] leading-none [&_svg]:size-4 [&_svg]:shrink-0',
    toolbarBand: 'flex min-w-0 flex-wrap items-center gap-2',
    toolbarBandSplit: 'flex w-full flex-wrap items-center justify-between gap-2',
    toolbarBandActions: 'flex min-w-0 flex-1 flex-wrap items-center gap-2',
    statusMetricsBand: 'flex flex-wrap items-center gap-4',
    chipRow: 'flex flex-wrap gap-2',
    chip: 'inline-flex min-h-7 max-w-full shrink-0 items-center justify-center gap-1.5 whitespace-nowrap border border-border px-3 py-1 text-xs font-semibold leading-[18px] shadow-none transition-colors',
    chipCount: 'text-[11px] font-semibold',
    summaryBand:
      'inline-flex h-7 max-w-full flex-nowrap items-center gap-2.5 overflow-x-auto border border-primary/20 bg-primary/5 px-3 text-card-foreground',
    summaryBandDivider: 'mx-1 h-3 w-px shrink-0 bg-border',
    tableHost:
      'w-full min-w-0 overflow-hidden rounded-[8px] border border-border bg-card text-card-foreground',
    tableHostFill: 'flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden',
    tableHostScroll: `${G1} min-w-0 overflow-x-auto bg-card`,
    directoryStack: 'flex w-full flex-col gap-3',
    metaLinksBand: `flex flex-wrap items-center ${H.gap.lg} ${ie.bodyMuted} [&_a]:text-primary [&_a:hover]:underline`,
    actionLinksBand: 'flex flex-wrap gap-2',
    filterPanel: 'grid gap-4 rounded-[8px] border border-border bg-card p-4 text-muted-foreground',
  },
  V1 = {
    error: $e.messageError,
    muted: $e.messageMuted,
    success: $e.messageSuccess,
    warning: $e.messageWarning,
  };
function zv(e) {
  return V1[e];
}
var fe = {
    controlRadius: 'rounded-[8px]',
    panelRadius: 'rounded-[8px]',
    nestedRadius: 'rounded-[4px]',
    pillRadius: 'rounded-full',
    controlPaddingX: 'px-3',
    fieldLabelGap: H.gap.md,
    chipPaddingX: 'px-3',
    chipInnerGap: H.gap.sm,
    compactInsetX: 'px-3',
    focusRing: 'focus-visible:outline-none focus-visible:ring-0 focus-visible:ring-offset-0',
    controlHeight: 'h-7 min-h-7',
    controlBorder: 'border border-border/40',
    controlText: ie.body,
    fieldLabelClass: ie.label,
    directoryTableHeadInnerClass: `flex w-full items-center ${H.gap.sm} ${H.inset.tableCellX} ${ie.tableHeader}`,
    buttonShell: $e.control,
    labelCaps: ie.caption,
    tableHeader: ie.tableHeader,
    tableRowHeight: 'h-[34px]',
    errorSurface: $e.messageError,
    toastSurface:
      'border-border/60 bg-card/90 text-card-foreground shadow-md shadow-black/10 backdrop-blur-none',
  },
  DT = {
    active: 'border-transparent bg-admin-status-active/10 text-admin-status-active',
    paused: 'border-transparent bg-admin-status-paused/10 text-admin-status-paused',
    archived: 'border-border bg-muted text-muted-foreground',
    error: 'border-transparent bg-destructive/10 text-destructive',
    draft: 'border-transparent bg-admin-status-draft/10 text-admin-status-draft',
    scheduled: 'border-transparent bg-admin-status-scheduled/10 text-admin-status-scheduled',
    muted: 'border-border bg-muted/50 text-muted-foreground',
  },
  kT = `inline-flex max-w-full shrink-0 items-center whitespace-nowrap rounded-md border border-border px-2.5 py-0.5 ${ie.badge} font-normal`,
  BT = { success: $e.messageSuccess, error: $e.messageError, warning: $e.messageWarning };
function OT(e, t) {
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
var Ql = {
  control: Fv(),
  controlFieldGroup: O(
    Fv(),
    `flex items-center ${H.gap.md} focus-within:border-border/40 focus-within:ring-0`
  ),
  controlFieldInset:
    'min-w-0 flex-1 border-0 bg-transparent p-0 text-foreground shadow-none outline-none placeholder:text-muted-foreground focus-visible:ring-0 focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50',
  controlGhost: O(
    fe.buttonShell,
    fe.controlRadius,
    `border border-transparent bg-transparent ${fe.controlPaddingX} text-foreground transition-colors hover:border-border hover:bg-accent hover:text-accent-foreground`
  ),
  panel: O(fe.panelRadius, 'border border-border bg-card text-card-foreground'),
  panelMuted: O(fe.panelRadius, 'bg-muted text-muted-foreground'),
  overlayBackdrop: 'fixed inset-0 z-50 bg-foreground/20 dark:bg-background/75',
  floating: O(
    'z-50 border border-border bg-popover p-1 text-popover-foreground shadow-md shadow-black/10',
    fe.controlRadius
  ),
  menuList: `flex flex-col ${H.gap.xs} p-0.5`,
  menuItem: O(
    `relative flex w-full cursor-pointer select-none items-center whitespace-nowrap ${fe.controlPaddingX} py-1.5 ${ie.body} outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent disabled:pointer-events-none disabled:opacity-60`,
    fe.nestedRadius
  ),
  menuItemSelected: 'bg-admin-selection text-foreground hover:bg-admin-selection',
  tableHead: `h-[34px] bg-muted/50 ${H.inset.tableCellX} text-left align-middle ${ie.tableHeader}`,
  tableCell: `${H.inset.tableCellX} py-0 align-middle ${ie.tableBody}`,
  muted: 'text-muted-foreground',
  pageTitle: ie.pageTitle,
};
function Fv() {
  return [
    'min-h-7 h-auto',
    fe.controlRadius,
    fe.controlText,
    `${fe.controlBorder} bg-admin-control ${fe.controlPaddingX} py-1 text-foreground shadow-none transition-colors`,
    'placeholder:text-muted-foreground',
    'hover:border-border/40 focus-visible:border-border/40 focus-visible:outline-none focus-visible:ring-0',
    'disabled:cursor-not-allowed disabled:border-border/40 disabled:bg-admin-input-disabled disabled:text-muted-foreground disabled:opacity-100',
    'aria-[invalid=true]:border-destructive/50 aria-[invalid=true]:ring-0',
  ].join(' ');
}
var qv = {
    default:
      'border-admin-brand bg-admin-brand text-admin-brand-foreground shadow-none hover:border-admin-brand-hover hover:bg-admin-brand-hover',
    brand:
      'border-admin-brand bg-admin-brand text-admin-brand-foreground shadow-none hover:border-admin-brand-hover hover:bg-admin-brand-hover',
    accent:
      'border-primary/35 bg-admin-selection text-foreground shadow-none hover:border-primary/50 hover:bg-admin-selection',
    secondary:
      'border-border bg-accent text-foreground shadow-none hover:border-border hover:bg-muted',
    outline:
      'border-border/40 bg-admin-control text-foreground shadow-none hover:border-border/40 hover:bg-admin-control hover:text-foreground focus-visible:ring-0 focus-visible:border-border/40',
    ghost:
      'border-transparent bg-transparent text-muted-foreground shadow-none hover:border-transparent hover:bg-accent hover:text-foreground',
    destructive:
      'border-destructive bg-destructive text-destructive-foreground shadow-none hover:border-destructive hover:bg-destructive/90',
    link: 'border-0 bg-transparent text-primary underline-offset-4 shadow-none hover:underline',
    headerHelp:
      'border-transparent bg-admin-header-faq text-primary-foreground shadow-none hover:border-transparent hover:bg-admin-header-faq-hover',
  },
  zT = {
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
var uo = E(te());
function j1(...e) {
  return (t) => {
    for (let a of e) typeof a == 'function' ? a(t) : a != null && (a.current = t);
  };
}
function Gv(...e) {
  return (t) => {
    for (let a of e) a?.(t);
  };
}
var Vv = uo.forwardRef(function ({ children: t, ...a }, l) {
  if (!uo.isValidElement(t)) return t ?? null;
  let r = t,
    o = r.ref;
  return uo.cloneElement(r, {
    ...a,
    ...r.props,
    ref: j1(l, o),
    onClick: Gv(a.onClick, r.props.onClick),
    onKeyDown: Gv(a.onKeyDown, r.props.onKeyDown),
  });
});
var On = E(Q()),
  X1 = { default: '', sm: 'text-xs', lg: 'px-5', icon: 'size-7 p-0' },
  Y1 = { default: '', pill: fe.pillRadius, square: fe.controlRadius },
  _a = jv.forwardRef(
    (
      {
        className: e,
        variant: t = 'default',
        size: a = 'default',
        shape: l = 'default',
        asChild: r = !1,
        loading: o = !1,
        disabled: n,
        children: u,
        type: i = 'button',
        ...s
      },
      f
    ) => {
      let d = O(
        fe.buttonShell,
        fe.controlRadius,
        'font-normal transition-colors focus-visible:outline-none focus-visible:ring-0 focus-visible:ring-offset-0 disabled:pointer-events-none disabled:opacity-50',
        qv[t],
        X1[a],
        Y1[l],
        e
      );
      return r
        ? (0, On.jsx)(Vv, {
            ref: f,
            'aria-busy': o || void 0,
            'aria-disabled': n || o || void 0,
            className: d,
            ...s,
            children: u,
          })
        : (0, On.jsxs)('button', {
            className: d,
            ref: f,
            disabled: n || o,
            'aria-busy': o || void 0,
            type: i,
            ...s,
            children: [
              o
                ? (0, On.jsx)(Sl, { className: 'h-4 w-4 animate-spin', 'aria-hidden': 'true' })
                : null,
              u,
            ],
          });
    }
  );
_a.displayName = 'Button';
var zi = E(Q());
function _i({ shape: e = 'default', variant: t = 'brand', ...a }) {
  return (0, zi.jsx)(_a, { shape: e, variant: t, ...a });
}
function ZT({ shape: e = 'pill', variant: t = 'outline', ...a }) {
  return (0, zi.jsx)(_a, { shape: e, variant: t, ...a });
}
function WT({
  shape: e = 'pill',
  type: t = 'submit',
  variant: a = 'brand',
  children: l = 'Apply',
  ...r
}) {
  return (0, zi.jsx)(_a, { shape: e, type: t, variant: a, ...r, children: l });
}
var Fi = 'AEP Admin';
function Xv() {
  return 'Ad Event Processor operator console';
}
var Yc = E(Q()),
  K1 = '/src/assets/product_avatar.svg',
  Q1 = { sm: 'h-6 w-6', md: 'h-8 w-8', lg: 'h-12 w-12' };
function Yv({ size: e = 'md', className: t, framed: a = !1 }) {
  return (0, Yc.jsx)('span', {
    className: O(
      'inline-flex shrink-0 items-center justify-center overflow-hidden',
      Q1[e],
      a && O('bg-primary text-primary-foreground', fe.controlRadius),
      t
    ),
    children: (0, Yc.jsx)('img', {
      alt: '',
      'aria-hidden': !0,
      className: O('h-full w-full object-cover', a && fe.controlRadius),
      decoding: 'async',
      draggable: !1,
      src: K1,
    }),
  });
}
var Zl = E(Q());
function Un({ children: e }) {
  return (0, Zl.jsxs)('div', {
    className: 'flex min-h-screen flex-col items-center justify-center bg-background p-4 sm:p-6',
    children: [
      (0, Zl.jsxs)('div', {
        className: O('mb-6 flex flex-col items-center', H.gap.md),
        children: [
          (0, Zl.jsx)(Yv, { framed: !0, size: 'lg' }),
          (0, Zl.jsx)('span', {
            className: O('whitespace-nowrap tracking-tight', ie.sectionTitle),
            children: Fi,
          }),
        ],
      }),
      (0, Zl.jsx)('div', { className: 'w-full max-w-sm', children: e }),
    ],
  });
}
var oL = E(te());
var R = E(te(), 1),
  Wv = E(hs(), 1);
function Z1(e) {
  if (!e || typeof document > 'u') return;
  let t = document.head || document.getElementsByTagName('head')[0],
    a = document.createElement('style');
  (a.type = 'text/css'),
    t.appendChild(a),
    a.styleSheet ? (a.styleSheet.cssText = e) : a.appendChild(document.createTextNode(e));
}
var W1 = (e) => {
    switch (e) {
      case 'success':
        return eR;
      case 'info':
        return aR;
      case 'warning':
        return tR;
      case 'error':
        return lR;
      default:
        return null;
    }
  },
  J1 = Array(12).fill(0),
  $1 = ({ visible: e, className: t }) =>
    R.default.createElement(
      'div',
      { className: ['sonner-loading-wrapper', t].filter(Boolean).join(' '), 'data-visible': e },
      R.default.createElement(
        'div',
        { className: 'sonner-spinner' },
        J1.map((a, l) =>
          R.default.createElement('div', {
            className: 'sonner-loading-bar',
            key: `spinner-bar-${l}`,
          })
        )
      )
    ),
  eR = R.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      viewBox: '0 0 20 20',
      fill: 'currentColor',
      height: '20',
      width: '20',
      'aria-hidden': 'true',
    },
    R.default.createElement('path', {
      fillRule: 'evenodd',
      d: 'M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z',
      clipRule: 'evenodd',
    })
  ),
  tR = R.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      viewBox: '0 0 24 24',
      fill: 'currentColor',
      height: '20',
      width: '20',
      'aria-hidden': 'true',
    },
    R.default.createElement('path', {
      fillRule: 'evenodd',
      d: 'M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003zM12 8.25a.75.75 0 01.75.75v3.75a.75.75 0 01-1.5 0V9a.75.75 0 01.75-.75zm0 8.25a.75.75 0 100-1.5.75.75 0 000 1.5z',
      clipRule: 'evenodd',
    })
  ),
  aR = R.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      viewBox: '0 0 20 20',
      fill: 'currentColor',
      height: '20',
      width: '20',
      'aria-hidden': 'true',
    },
    R.default.createElement('path', {
      fillRule: 'evenodd',
      d: 'M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z',
      clipRule: 'evenodd',
    })
  ),
  lR = R.default.createElement(
    'svg',
    {
      xmlns: 'http://www.w3.org/2000/svg',
      viewBox: '0 0 20 20',
      fill: 'currentColor',
      height: '20',
      width: '20',
      'aria-hidden': 'true',
    },
    R.default.createElement('path', {
      fillRule: 'evenodd',
      d: 'M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-5a.75.75 0 01.75.75v4.5a.75.75 0 01-1.5 0v-4.5A.75.75 0 0110 5zm0 10a1 1 0 100-2 1 1 0 000 2z',
      clipRule: 'evenodd',
    })
  ),
  rR = R.default.createElement(
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
    R.default.createElement('line', { x1: '18', y1: '6', x2: '6', y2: '18' }),
    R.default.createElement('line', { x1: '6', y1: '6', x2: '18', y2: '18' })
  ),
  oR = () => {
    let [e, t] = R.default.useState(document.hidden);
    return (
      R.default.useEffect(() => {
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
  nR = 1,
  uR = 100,
  Kv = (e) => {
    var t;
    return typeof e?.id == 'number' || (e == null || (t = e.id) == null ? void 0 : t.length) > 0
      ? e.id
      : nR++;
  },
  Kc = class {
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
          let t = this.toasts.length - uR;
          t <= 0 ||
            (this.toasts = this.toasts.filter((a) =>
              t > 0 && this.dismissedToasts.has(a.id)
                ? (this.dismissedToasts.delete(a.id), t--, !1)
                : !0
            ));
        }),
        (this.create = (t) => {
          let { message: a, ...l } = t,
            r = Kv(t),
            o = this.pendingDismissals.get(r);
          o !== void 0 &&
            (cancelAnimationFrame(o),
            this.pendingDismissals.delete(r),
            this.dismissedToasts.delete(r));
          let n = this.dismissedToasts.has(r),
            u = t.dismissible === void 0 ? !0 : t.dismissible;
          return (
            n &&
              (this.dismissedToasts.delete(r),
              (this.toasts = this.toasts.filter((s) => s.id !== r))),
            (n ? void 0 : this.toasts.find((s) => s.id === r))
              ? (this.toasts = this.toasts.map((s) =>
                  s.id === r
                    ? (this.publish({ ...s, ...t, id: r, title: a }),
                      { ...s, ...t, id: r, dismissible: u, title: a })
                    : s
                ))
              : this.addToast({ title: a, ...l, dismissible: u, id: r }),
            r
          );
        }),
        (this.dismiss = (t) => {
          if (t == null)
            return (
              this.getActiveToasts().forEach((l) => {
                this.dismissedToasts.add(l.id),
                  this.subscribers.forEach((r) => r({ id: l.id, dismiss: !0 }));
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
                  this.subscribers.forEach((l) => l({ id: t, dismiss: !0 }));
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
          let l;
          a.loading !== void 0 &&
            (l = this.create({
              ...a,
              promise: t,
              type: 'loading',
              message: a.loading,
              description: typeof a.description != 'function' ? a.description : void 0,
            }));
          let r = Promise.resolve(t instanceof Function ? t() : t),
            o = l !== void 0,
            n,
            u = r
              .then(async (s) => {
                if (((n = ['resolve', s]), R.default.isValidElement(s)))
                  (o = !1), this.create({ id: l, type: 'default', message: s });
                else if (sR(s) && !s.ok) {
                  o = !1;
                  let d =
                      typeof a.error == 'function'
                        ? await a.error(`HTTP error! status: ${s.status}`)
                        : a.error,
                    p =
                      typeof a.description == 'function'
                        ? await a.description(`HTTP error! status: ${s.status}`)
                        : a.description,
                    x = typeof d == 'object' && !R.default.isValidElement(d) ? d : { message: d };
                  this.create({ id: l, type: 'error', description: p, ...x });
                } else if (s instanceof Error) {
                  o = !1;
                  let d = typeof a.error == 'function' ? await a.error(s) : a.error,
                    p = typeof a.description == 'function' ? await a.description(s) : a.description,
                    x = typeof d == 'object' && !R.default.isValidElement(d) ? d : { message: d };
                  this.create({ id: l, type: 'error', description: p, ...x });
                } else if (a.success !== void 0) {
                  o = !1;
                  let d = typeof a.success == 'function' ? await a.success(s) : a.success,
                    p = typeof a.description == 'function' ? await a.description(s) : a.description,
                    x = typeof d == 'object' && !R.default.isValidElement(d) ? d : { message: d };
                  this.create({ id: l, type: 'success', description: p, ...x });
                }
              })
              .catch(async (s) => {
                if (((n = ['reject', s]), a.error !== void 0)) {
                  o = !1;
                  let f = typeof a.error == 'function' ? await a.error(s) : a.error,
                    d = typeof a.description == 'function' ? await a.description(s) : a.description,
                    h = typeof f == 'object' && !R.default.isValidElement(f) ? f : { message: f };
                  this.create({ id: l, type: 'error', description: d, ...h });
                }
              })
              .finally(() => {
                o && (this.dismiss(l), (l = void 0)), a.finally == null || a.finally.call(a);
              }),
            i = () =>
              new Promise((s, f) => u.then(() => (n[0] === 'reject' ? f(n[1]) : s(n[1]))).catch(f));
          return typeof l != 'string' && typeof l != 'number'
            ? { unwrap: i }
            : Object.assign(l, { unwrap: i });
        }),
        (this.custom = (t, a) => {
          let l = Kv(a);
          return this.create({ ...a, jsx: t(l), id: l, type: void 0 }), l;
        }),
        (this.getActiveToasts = () => this.toasts.filter((t) => !this.dismissedToasts.has(t.id))),
        (this.subscribers = []),
        (this.toasts = []),
        (this.dismissedToasts = new Set()),
        (this.pendingDismissals = new Map());
    }
  },
  yt = new Kc(),
  iR = (e, t) => yt.message(e, t),
  sR = (e) =>
    e &&
    typeof e == 'object' &&
    'ok' in e &&
    typeof e.ok == 'boolean' &&
    'status' in e &&
    typeof e.status == 'number',
  dR = iR,
  fR = () => yt.toasts,
  cR = () => yt.getActiveToasts(),
  Jv = Object.assign(
    dR,
    {
      success: yt.success,
      info: yt.info,
      warning: yt.warning,
      error: yt.error,
      custom: yt.custom,
      message: yt.message,
      promise: yt.promise,
      dismiss: yt.dismiss,
      loading: yt.loading,
    },
    { getHistory: fR, getToasts: cR }
  );
Z1(
  "[data-sonner-toaster][dir=ltr],html[dir=ltr]{--toast-icon-margin-start:-3px;--toast-icon-margin-end:4px;--toast-svg-margin-start:-1px;--toast-svg-margin-end:0px;--toast-button-margin-start:auto;--toast-button-margin-end:0;--toast-close-button-start:0;--toast-close-button-end:unset;--toast-close-button-transform:translate(-35%, -35%)}[data-sonner-toaster][dir=rtl],html[dir=rtl]{--toast-icon-margin-start:4px;--toast-icon-margin-end:-3px;--toast-svg-margin-start:0px;--toast-svg-margin-end:-1px;--toast-button-margin-start:0;--toast-button-margin-end:auto;--toast-close-button-start:unset;--toast-close-button-end:0;--toast-close-button-transform:translate(35%, -35%)}[data-sonner-toaster]{position:fixed;width:var(--width);font-family:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Arial,Noto Sans,sans-serif,Apple Color Emoji,Segoe UI Emoji,Segoe UI Symbol,Noto Color Emoji;--gray1:hsl(0, 0%, 99%);--gray2:hsl(0, 0%, 97.3%);--gray3:hsl(0, 0%, 95.1%);--gray4:hsl(0, 0%, 93%);--gray5:hsl(0, 0%, 90.9%);--gray6:hsl(0, 0%, 88.7%);--gray7:hsl(0, 0%, 85.8%);--gray8:hsl(0, 0%, 78%);--gray9:hsl(0, 0%, 56.1%);--gray10:hsl(0, 0%, 52.3%);--gray11:hsl(0, 0%, 43.5%);--gray12:hsl(0, 0%, 9%);--border-radius:8px;box-sizing:border-box;padding:0;margin:0;list-style:none;outline:0;z-index:999999999;transition:transform .4s ease}@media (hover:none) and (pointer:coarse){[data-sonner-toaster][data-lifted=true]{transform:none}}[data-sonner-toaster][data-x-position=right]{right:var(--offset-right)}[data-sonner-toaster][data-x-position=left]{left:var(--offset-left)}[data-sonner-toaster][data-x-position=center]{left:50%;transform:translateX(-50%)}[data-sonner-toaster][data-y-position=top]{top:var(--offset-top)}[data-sonner-toaster][data-y-position=bottom]{bottom:var(--offset-bottom)}[data-sonner-toast]{--y:translateY(100%);--lift-amount:calc(var(--lift) * var(--gap));z-index:var(--z-index);position:absolute;opacity:0;transform:var(--y);touch-action:none;transition:transform .4s,opacity .4s,height .4s,box-shadow .2s;box-sizing:border-box;outline:0;overflow-wrap:anywhere}[data-sonner-toast][data-styled=true]{padding:16px;background:var(--normal-bg);border:1px solid var(--normal-border);color:var(--normal-text);border-radius:var(--border-radius);box-shadow:0 4px 12px rgba(0,0,0,.1);width:var(--width);font-size:13px;display:flex;align-items:center;gap:6px}[data-sonner-toast]:focus-visible{box-shadow:0 4px 12px rgba(0,0,0,.1),0 0 0 2px rgba(0,0,0,.2)}[data-sonner-toast][data-y-position=top]{top:0;--y:translateY(-100%);--lift:1;--lift-amount:calc(1 * var(--gap))}[data-sonner-toast][data-y-position=bottom]{bottom:0;--y:translateY(100%);--lift:-1;--lift-amount:calc(var(--lift) * var(--gap))}[data-sonner-toast][data-styled=true] [data-description]{font-weight:400;line-height:1.4;color:#3f3f3f}[data-rich-colors=true][data-sonner-toast][data-styled=true] [data-description]{color:inherit}[data-sonner-toaster][data-sonner-theme=dark] [data-description]{color:#e8e8e8}[data-sonner-toast][data-styled=true] [data-title]{font-weight:500;line-height:1.5;color:inherit}[data-sonner-toast][data-styled=true] [data-icon]{display:flex;height:16px;width:16px;position:relative;justify-content:flex-start;align-items:center;flex-shrink:0;margin-left:var(--toast-icon-margin-start);margin-right:var(--toast-icon-margin-end)}[data-sonner-toast][data-promise=true] [data-icon]>svg{opacity:0;transform:scale(.8);transform-origin:center;animation:sonner-fade-in .3s ease forwards}[data-sonner-toast][data-styled=true] [data-icon]>*{flex-shrink:0}[data-sonner-toast][data-styled=true] [data-icon] svg{margin-left:var(--toast-svg-margin-start);margin-right:var(--toast-svg-margin-end)}[data-sonner-toast][data-styled=true] [data-content]{display:flex;flex-direction:column;gap:2px;flex:1;min-width:0}[data-sonner-toast][data-styled=true] [data-button]{border-radius:4px;padding-left:8px;padding-right:8px;height:24px;font-size:12px;color:var(--normal-bg);background:var(--normal-text);margin-left:var(--toast-button-margin-start);margin-right:var(--toast-button-margin-end);border:none;font-weight:500;cursor:pointer;outline:0;display:flex;align-items:center;flex-shrink:0;transition:opacity .4s,box-shadow .2s}[data-sonner-toast][data-styled=true] [data-button]:focus-visible{box-shadow:0 0 0 2px rgba(0,0,0,.4)}[data-sonner-toast][data-styled=true] [data-button]:first-of-type{margin-left:var(--toast-button-margin-start);margin-right:var(--toast-button-margin-end)}[data-sonner-toast][data-styled=true] [data-cancel]{color:var(--normal-text);background:rgba(0,0,0,.08)}[data-sonner-toaster][data-sonner-theme=dark] [data-sonner-toast][data-styled=true] [data-cancel]{background:rgba(255,255,255,.3)}[data-sonner-toast][data-styled=true] [data-close-button]{position:absolute;left:var(--toast-close-button-start);right:var(--toast-close-button-end);top:0;height:20px;width:20px;display:flex;justify-content:center;align-items:center;padding:0;color:var(--normal-text);background:var(--normal-bg);border:1px solid var(--normal-border);transform:var(--toast-close-button-transform);border-radius:50%;cursor:pointer;z-index:1;transition:opacity .1s,background .2s,border-color .2s}[data-sonner-toast][data-styled=true] [data-close-button]:focus-visible{box-shadow:0 4px 12px rgba(0,0,0,.1),0 0 0 2px rgba(0,0,0,.2)}[data-sonner-toast][data-styled=true] [data-disabled=true]{cursor:not-allowed}[data-sonner-toast][data-styled=true]:hover [data-close-button]:hover{background:var(--gray2);border-color:var(--gray5)}[data-sonner-toast][data-swiping=true]::before{content:'';position:absolute;left:-100%;right:-100%;height:100%;z-index:-1}[data-sonner-toast][data-y-position=top][data-swiping=true]::before{bottom:50%;transform:scaleY(3) translateY(50%)}[data-sonner-toast][data-y-position=bottom][data-swiping=true]::before{top:50%;transform:scaleY(3) translateY(-50%)}[data-sonner-toast][data-swiping=false][data-removed=true]::before{content:'';position:absolute;inset:0;transform:scaleY(2)}[data-sonner-toast][data-expanded=true]::after{content:'';position:absolute;left:0;height:calc(var(--gap) + 1px);bottom:100%;width:100%}[data-sonner-toast][data-mounted=true]{--y:translateY(0);opacity:1}[data-sonner-toast][data-expanded=false][data-front=false]{--scale:var(--toasts-before) * 0.05 + 1;--y:translateY(calc(var(--lift-amount) * var(--toasts-before))) scale(calc(-1 * var(--scale)));height:var(--front-toast-height)}[data-sonner-toast]>*{transition:opacity .4s}[data-sonner-toast][data-x-position=right]{right:0}[data-sonner-toast][data-x-position=left]{left:0}[data-sonner-toast][data-expanded=false][data-front=false][data-styled=true]>*{opacity:0}[data-sonner-toast][data-visible=false]{opacity:0;pointer-events:none}[data-sonner-toast][data-mounted=true][data-expanded=true]{--y:translateY(calc(var(--lift) * var(--offset)));height:var(--initial-height)}[data-sonner-toast][data-removed=true][data-front=true][data-swipe-out=false]{--y:translateY(calc(var(--lift) * -100%));opacity:0}[data-sonner-toast][data-removed=true][data-front=false][data-swipe-out=false][data-expanded=true]{--y:translateY(calc(var(--lift) * var(--offset) + var(--lift) * -100%));opacity:0}[data-sonner-toast][data-removed=true][data-front=false][data-swipe-out=false][data-expanded=false]{--y:translateY(40%);opacity:0;transition:transform .5s,opacity .2s}[data-sonner-toast][data-removed=true][data-front=false]::before{height:calc(var(--initial-height) + 20%)}[data-sonner-toast][data-swiping=true]{transform:var(--y) translateY(var(--swipe-amount-y,0)) translateX(var(--swipe-amount-x,0));transition:none}[data-sonner-toast][data-swiped=true]{-webkit-user-select:none;user-select:none}[data-sonner-toast][data-swipe-out=true][data-y-position=bottom],[data-sonner-toast][data-swipe-out=true][data-y-position=top]{animation-duration:.2s;animation-timing-function:ease-out;animation-fill-mode:forwards}[data-sonner-toast][data-swipe-out=true][data-swipe-direction=left]{animation-name:swipe-out-left}[data-sonner-toast][data-swipe-out=true][data-swipe-direction=right]{animation-name:swipe-out-right}[data-sonner-toast][data-swipe-out=true][data-swipe-direction=up]{animation-name:swipe-out-up}[data-sonner-toast][data-swipe-out=true][data-swipe-direction=down]{animation-name:swipe-out-down}@keyframes swipe-out-left{from{transform:var(--y) translateX(var(--swipe-amount-x));opacity:1}to{transform:var(--y) translateX(calc(var(--swipe-amount-x) - 100%));opacity:0}}@keyframes swipe-out-right{from{transform:var(--y) translateX(var(--swipe-amount-x));opacity:1}to{transform:var(--y) translateX(calc(var(--swipe-amount-x) + 100%));opacity:0}}@keyframes swipe-out-up{from{transform:var(--y) translateY(var(--swipe-amount-y));opacity:1}to{transform:var(--y) translateY(calc(var(--swipe-amount-y) - 100%));opacity:0}}@keyframes swipe-out-down{from{transform:var(--y) translateY(var(--swipe-amount-y));opacity:1}to{transform:var(--y) translateY(calc(var(--swipe-amount-y) + 100%));opacity:0}}@media (max-width:600px){[data-sonner-toaster]{position:fixed;right:var(--mobile-offset-right);left:var(--mobile-offset-left);width:100%}[data-sonner-toaster][dir=rtl]{left:calc(var(--mobile-offset-left) * -1)}[data-sonner-toaster] [data-sonner-toast]{left:0;right:0;width:calc(100% - var(--mobile-offset-left) * 2)}[data-sonner-toaster][data-x-position=left]{left:var(--mobile-offset-left)}[data-sonner-toaster][data-y-position=bottom]{bottom:var(--mobile-offset-bottom)}[data-sonner-toaster][data-y-position=top]{top:var(--mobile-offset-top)}[data-sonner-toaster][data-x-position=center]{left:var(--mobile-offset-left);right:var(--mobile-offset-right);transform:none}}[data-sonner-toaster][data-sonner-theme=light]{--normal-bg:#fff;--normal-border:var(--gray4);--normal-text:var(--gray12);--success-bg:hsl(143, 85%, 96%);--success-border:hsl(145, 92%, 87%);--success-text:hsl(140, 100%, 27%);--info-bg:hsl(208, 100%, 97%);--info-border:hsl(221, 91%, 93%);--info-text:hsl(210, 92%, 45%);--warning-bg:hsl(49, 100%, 97%);--warning-border:hsl(49, 91%, 84%);--warning-text:hsl(31, 92%, 45%);--error-bg:hsl(359, 100%, 97%);--error-border:hsl(359, 100%, 94%);--error-text:hsl(360, 100%, 45%)}[data-sonner-toaster][data-sonner-theme=light] [data-sonner-toast][data-invert=true]{--normal-bg:#000;--normal-border:hsl(0, 0%, 20%);--normal-text:var(--gray1)}[data-sonner-toaster][data-sonner-theme=dark] [data-sonner-toast][data-invert=true]{--normal-bg:#fff;--normal-border:var(--gray3);--normal-text:var(--gray12)}[data-sonner-toaster][data-sonner-theme=dark]{--normal-bg:#000;--normal-bg-hover:hsl(0, 0%, 12%);--normal-border:hsl(0, 0%, 20%);--normal-border-hover:hsl(0, 0%, 25%);--normal-text:var(--gray1);--success-bg:hsl(150, 100%, 6%);--success-border:hsl(147, 100%, 12%);--success-text:hsl(150, 86%, 65%);--info-bg:hsl(215, 100%, 6%);--info-border:hsl(223, 43%, 17%);--info-text:hsl(216, 87%, 65%);--warning-bg:hsl(64, 100%, 6%);--warning-border:hsl(60, 100%, 9%);--warning-text:hsl(46, 87%, 65%);--error-bg:hsl(358, 76%, 10%);--error-border:hsl(357, 89%, 16%);--error-text:hsl(358, 100%, 81%)}[data-sonner-toaster][data-sonner-theme=dark] [data-sonner-toast] [data-close-button]{background:var(--normal-bg);border-color:var(--normal-border);color:var(--normal-text)}[data-sonner-toaster][data-sonner-theme=dark] [data-sonner-toast] [data-close-button]:hover{background:var(--normal-bg-hover);border-color:var(--normal-border-hover)}[data-rich-colors=true][data-sonner-toast][data-type=success]{background:var(--success-bg);border-color:var(--success-border);color:var(--success-text)}[data-rich-colors=true][data-sonner-toast][data-type=success] [data-close-button]{background:var(--success-bg);border-color:var(--success-border);color:var(--success-text)}[data-rich-colors=true][data-sonner-toast][data-type=info]{background:var(--info-bg);border-color:var(--info-border);color:var(--info-text)}[data-rich-colors=true][data-sonner-toast][data-type=info] [data-close-button]{background:var(--info-bg);border-color:var(--info-border);color:var(--info-text)}[data-rich-colors=true][data-sonner-toast][data-type=warning]{background:var(--warning-bg);border-color:var(--warning-border);color:var(--warning-text)}[data-rich-colors=true][data-sonner-toast][data-type=warning] [data-close-button]{background:var(--warning-bg);border-color:var(--warning-border);color:var(--warning-text)}[data-rich-colors=true][data-sonner-toast][data-type=error]{background:var(--error-bg);border-color:var(--error-border);color:var(--error-text)}[data-rich-colors=true][data-sonner-toast][data-type=error] [data-close-button]{background:var(--error-bg);border-color:var(--error-border);color:var(--error-text)}.sonner-loading-wrapper{--size:16px;height:var(--size);width:var(--size);position:absolute;inset:0;z-index:10}.sonner-loading-wrapper[data-visible=false]{transform-origin:center;animation:sonner-fade-out .2s ease forwards}.sonner-spinner{position:relative;top:50%;left:50%;height:var(--size);width:var(--size)}.sonner-loading-bar{animation:sonner-spin 1.2s linear infinite;background:var(--gray11);border-radius:6px;height:8%;left:-10%;position:absolute;top:-3.9%;width:24%}.sonner-loading-bar:first-child{animation-delay:-1.2s;transform:rotate(.0001deg) translate(146%)}.sonner-loading-bar:nth-child(2){animation-delay:-1.1s;transform:rotate(30deg) translate(146%)}.sonner-loading-bar:nth-child(3){animation-delay:-1s;transform:rotate(60deg) translate(146%)}.sonner-loading-bar:nth-child(4){animation-delay:-.9s;transform:rotate(90deg) translate(146%)}.sonner-loading-bar:nth-child(5){animation-delay:-.8s;transform:rotate(120deg) translate(146%)}.sonner-loading-bar:nth-child(6){animation-delay:-.7s;transform:rotate(150deg) translate(146%)}.sonner-loading-bar:nth-child(7){animation-delay:-.6s;transform:rotate(180deg) translate(146%)}.sonner-loading-bar:nth-child(8){animation-delay:-.5s;transform:rotate(210deg) translate(146%)}.sonner-loading-bar:nth-child(9){animation-delay:-.4s;transform:rotate(240deg) translate(146%)}.sonner-loading-bar:nth-child(10){animation-delay:-.3s;transform:rotate(270deg) translate(146%)}.sonner-loading-bar:nth-child(11){animation-delay:-.2s;transform:rotate(300deg) translate(146%)}.sonner-loading-bar:nth-child(12){animation-delay:-.1s;transform:rotate(330deg) translate(146%)}@keyframes sonner-fade-in{0%{opacity:0;transform:scale(.8)}100%{opacity:1;transform:scale(1)}}@keyframes sonner-fade-out{0%{opacity:1;transform:scale(1)}100%{opacity:0;transform:scale(.8)}}@keyframes sonner-spin{0%{opacity:1}100%{opacity:.15}}@media (prefers-reduced-motion){.sonner-loading-bar,[data-sonner-toast],[data-sonner-toast]>*{transition:none!important;animation:none!important}}.sonner-loader{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);transform-origin:center;transition:opacity .2s,transform .2s}.sonner-loader[data-visible=false]{opacity:0;transform:scale(.8) translate(-50%,-50%)}"
);
function qi(e) {
  return e.label !== void 0;
}
var mR = 3,
  pR = '24px',
  hR = '16px',
  Qv = 4e3,
  gR = 356,
  yR = 14,
  xR = 45,
  vR = 200;
function ca(...e) {
  return e.filter(Boolean).join(' ');
}
function LR(e) {
  let [t, a] = e.split('-'),
    l = [];
  return t && l.push(t), a && l.push(a), l;
}
var SR = (e) => {
  var t, a, l, r, o, n, u, i, s;
  let {
      invert: f,
      toast: d,
      unstyled: p,
      interacting: h,
      setHeights: x,
      visibleToasts: v,
      heights: L,
      index: c,
      toasts: m,
      expanded: g,
      removeToast: y,
      defaultRichColors: w,
      closeButton: U,
      style: C,
      cancelButtonStyle: b,
      actionButtonStyle: M,
      className: _ = '',
      descriptionClassName: Ue = '',
      duration: ot,
      position: Gt,
      gap: Qe,
      expandByDefault: be,
      classNames: j,
      icons: Me,
      closeButtonAriaLabel: xt = 'Close toast',
    } = e,
    [ze, B] = R.default.useState(null),
    [Mt, Vt] = R.default.useState(null),
    [ma, ee] = R.default.useState(!1),
    [V, X] = R.default.useState(!1),
    [Fe, jt] = R.default.useState(!1),
    [F, Fa] = R.default.useState(!1),
    [pa, Jt] = R.default.useState(!1),
    [ha, qa] = R.default.useState(0),
    [cL, $c] = R.default.useState(0),
    vo = R.default.useRef(d.duration || ot || Qv),
    em = R.default.useRef(null),
    ga = R.default.useRef(null),
    mL = c === 0,
    pL = c + 1 <= v,
    tt = d.type,
    tm = tt ?? 'default',
    ar = d.dismissible !== !1,
    hL = d.className || '',
    gL = d.descriptionClassName || '',
    _n = R.default.useMemo(() => L.findIndex((P) => P.toastId === d.id) || 0, [L, d.id]),
    yL = R.default.useMemo(() => {
      var P;
      return (P = d.closeButton) != null ? P : U;
    }, [d.closeButton, U]),
    am = R.default.useMemo(() => d.duration || ot || Qv, [d.duration, ot]),
    Qi = R.default.useRef(0),
    lr = R.default.useRef(0),
    lm = R.default.useRef(0),
    rr = R.default.useRef(null),
    [xL, vL] = Gt.split('-'),
    rm = R.default.useMemo(
      () => L.reduce((P, He, Ze) => (Ze >= _n ? P : P + He.height), 0),
      [L, _n]
    ),
    om = oR(),
    $t = R.default.useMemo(() => {
      var P;
      return (P = e.swipeDirections) != null ? P : LR(Gt);
    }, [e.swipeDirections, Gt]),
    LL = d.invert || f,
    Zi = tt === 'loading';
  (lr.current = R.default.useMemo(() => _n * Qe + rm, [_n, rm])),
    R.default.useEffect(() => {
      vo.current = am;
    }, [am]),
    R.default.useEffect(() => {
      ee(!0);
    }, []),
    R.default.useEffect(() => {
      let P = ga.current;
      if (P) {
        let He = P.getBoundingClientRect().height;
        return (
          $c(He),
          x((Ze) => [{ toastId: d.id, height: He, position: d.position }, ...Ze]),
          () => x((Ze) => Ze.filter((nt) => nt.toastId !== d.id))
        );
      }
    }, [x, d.id]),
    R.default.useLayoutEffect(() => {
      if (!ma) return;
      let P = ga.current,
        He = P.style.height;
      P.style.height = 'auto';
      let Ze = P.getBoundingClientRect().height;
      (P.style.height = He),
        $c(Ze),
        x((nt) =>
          nt.find((qe) => qe.toastId === d.id)
            ? nt.map((qe) => (qe.toastId === d.id ? { ...qe, height: Ze } : qe))
            : [{ toastId: d.id, height: Ze, position: d.position }, ...nt]
        );
    }, [ma, d.title, d.description, x, d.id, d.jsx, d.action, d.cancel]);
  let Ga = R.default.useCallback(() => {
    X(!0),
      qa(lr.current),
      x((P) => P.filter((He) => He.toastId !== d.id)),
      setTimeout(() => {
        y(d);
      }, vR);
  }, [d, y, x, lr]);
  R.default.useEffect(() => {
    if ((d.promise && tt === 'loading') || d.duration === 1 / 0 || d.type === 'loading') return;
    let P;
    return (
      g || h || om
        ? (() => {
            if (lm.current < Qi.current) {
              let nt = new Date().getTime() - Qi.current;
              vo.current = vo.current - nt;
            }
            lm.current = new Date().getTime();
          })()
        : (() => {
            vo.current !== 1 / 0 &&
              ((Qi.current = new Date().getTime()),
              (P = setTimeout(() => {
                d.onAutoClose == null || d.onAutoClose.call(d, d), Ga();
              }, vo.current)));
          })(),
      () => clearTimeout(P)
    );
  }, [g, h, d, tt, om, Ga]),
    R.default.useEffect(() => {
      d.delete && (Ga(), d.onDismiss == null || d.onDismiss.call(d, d));
    }, [Ga, d.delete]);
  function nm() {
    var P;
    if (Me?.loading) {
      var He;
      return R.default.createElement(
        'div',
        {
          className: ca(
            j?.loader,
            d == null || (He = d.classNames) == null ? void 0 : He.loader,
            'sonner-loader'
          ),
          'data-visible': tt === 'loading',
        },
        Me.loading
      );
    }
    return R.default.createElement($1, {
      className: ca(j?.loader, d == null || (P = d.classNames) == null ? void 0 : P.loader),
      visible: tt === 'loading',
    });
  }
  let SL = d.icon || Me?.[tt] || W1(tt);
  var um, im;
  return R.default.createElement(
    'li',
    {
      tabIndex: 0,
      ref: ga,
      className: ca(
        _,
        hL,
        j?.toast,
        d == null || (t = d.classNames) == null ? void 0 : t.toast,
        j?.[tm],
        d == null || (a = d.classNames) == null ? void 0 : a[tm]
      ),
      'data-sonner-toast': '',
      'data-rich-colors': (um = d.richColors) != null ? um : w,
      'data-styled': !(d.jsx || d.unstyled || p),
      'data-mounted': ma,
      'data-promise': !!d.promise,
      'data-swiped': pa,
      'data-removed': V,
      'data-visible': pL,
      'data-y-position': xL,
      'data-x-position': vL,
      'data-index': c,
      'data-front': mL,
      'data-swiping': Fe,
      'data-dismissible': ar,
      'data-type': tt,
      'data-invert': LL,
      'data-swipe-out': F,
      'data-swipe-direction': Mt,
      'data-expanded': !!(g || (be && ma)),
      'data-testid': d.testId,
      style: {
        '--index': c,
        '--toasts-before': c,
        '--z-index': m.length - c,
        '--offset': `${V ? ha : lr.current}px`,
        '--initial-height': be ? 'auto' : `${cL}px`,
        ...C,
        ...d.style,
      },
      onDragEnd: () => {
        jt(!1), B(null), (rr.current = null);
      },
      onPointerDown: (P) => {
        P.button !== 2 &&
          (Zi ||
            !ar ||
            ((em.current = new Date()),
            qa(lr.current),
            P.target.setPointerCapture(P.pointerId),
            P.target.tagName !== 'BUTTON' &&
              (jt(!0), (rr.current = { x: P.clientX, y: P.clientY }))));
      },
      onPointerUp: () => {
        var P, He, Ze;
        if (F || !ar) return;
        rr.current = null;
        let nt = Number(
            ((P = ga.current) == null
              ? void 0
              : P.style.getPropertyValue('--swipe-amount-x').replace('px', '')) || 0
          ),
          Lo = Number(
            ((He = ga.current) == null
              ? void 0
              : He.style.getPropertyValue('--swipe-amount-y').replace('px', '')) || 0
          ),
          qe = new Date().getTime() - ((Ze = em.current) == null ? void 0 : Ze.getTime()),
          Dt = ze === 'x' ? nt : Lo,
          ea = Math.abs(Dt) / qe;
        if (
          (ze === 'x'
            ? $t.includes(nt > 0 ? 'right' : 'left')
            : $t.includes(Lo > 0 ? 'bottom' : 'top')) &&
          (Math.abs(Dt) >= xR || ea > 0.11)
        ) {
          qa(lr.current),
            d.onDismiss == null || d.onDismiss.call(d, d),
            Vt(ze === 'x' ? (nt > 0 ? 'right' : 'left') : Lo > 0 ? 'down' : 'up'),
            Ga(),
            Fa(!0);
          return;
        } else {
          var ta, Ji;
          (ta = ga.current) == null || ta.style.setProperty('--swipe-amount-x', '0px'),
            (Ji = ga.current) == null || Ji.style.setProperty('--swipe-amount-y', '0px');
        }
        Jt(!1), jt(!1), B(null);
      },
      onPointerMove: (P) => {
        var He, Ze, nt;
        if (
          !rr.current ||
          !ar ||
          ((He = window.getSelection()) == null ? void 0 : He.toString().length) > 0
        )
          return;
        let qe = P.clientY - rr.current.y,
          Dt = P.clientX - rr.current.x;
        !ze && (Math.abs(Dt) > 1 || Math.abs(qe) > 1) && B(Math.abs(Dt) > Math.abs(qe) ? 'x' : 'y');
        let ea = { x: 0, y: 0 },
          Wi = (ta) => 1 / (1.5 + Math.abs(ta) / 20);
        if (ze === 'y') {
          if ($t.includes('top') || $t.includes('bottom'))
            if (($t.includes('top') && qe < 0) || ($t.includes('bottom') && qe > 0)) ea.y = qe;
            else {
              let ta = qe * Wi(qe);
              ea.y = Math.abs(ta) < Math.abs(qe) ? ta : qe;
            }
        } else if (ze === 'x' && ($t.includes('left') || $t.includes('right')))
          if (($t.includes('left') && Dt < 0) || ($t.includes('right') && Dt > 0)) ea.x = Dt;
          else {
            let ta = Dt * Wi(Dt);
            ea.x = Math.abs(ta) < Math.abs(Dt) ? ta : Dt;
          }
        (Math.abs(ea.x) > 0 || Math.abs(ea.y) > 0) && Jt(!0),
          (Ze = ga.current) == null || Ze.style.setProperty('--swipe-amount-x', `${ea.x}px`),
          (nt = ga.current) == null || nt.style.setProperty('--swipe-amount-y', `${ea.y}px`);
      },
    },
    yL && !d.jsx && tt !== 'loading'
      ? R.default.createElement(
          'button',
          {
            'aria-label': xt,
            'data-disabled': Zi,
            'data-close-button': !0,
            onClick:
              Zi || !ar
                ? () => {}
                : () => {
                    Ga(), d.onDismiss == null || d.onDismiss.call(d, d);
                  },
            className: ca(
              j?.closeButton,
              d == null || (l = d.classNames) == null ? void 0 : l.closeButton
            ),
          },
          (im = Me?.close) != null ? im : rR
        )
      : null,
    (tt || d.icon || d.promise) && d.icon !== null && (Me?.[tt] !== null || d.icon)
      ? R.default.createElement(
          'div',
          {
            'data-icon': '',
            className: ca(j?.icon, d == null || (r = d.classNames) == null ? void 0 : r.icon),
          },
          tt === 'loading' ? d.icon || nm() : d.promise ? nm() : null,
          tt !== 'loading' ? SL : null
        )
      : null,
    R.default.createElement(
      'div',
      {
        'data-content': '',
        className: ca(j?.content, d == null || (o = d.classNames) == null ? void 0 : o.content),
      },
      R.default.createElement(
        'div',
        {
          'data-title': '',
          className: ca(j?.title, d == null || (n = d.classNames) == null ? void 0 : n.title),
        },
        d.jsx ? d.jsx : typeof d.title == 'function' ? d.title() : d.title
      ),
      d.description
        ? R.default.createElement(
            'div',
            {
              'data-description': '',
              className: ca(
                Ue,
                gL,
                j?.description,
                d == null || (u = d.classNames) == null ? void 0 : u.description
              ),
            },
            typeof d.description == 'function' ? d.description() : d.description
          )
        : null
    ),
    R.default.isValidElement(d.cancel)
      ? d.cancel
      : d.cancel && qi(d.cancel)
        ? R.default.createElement(
            'button',
            {
              'data-button': !0,
              'data-cancel': !0,
              style: d.cancelButtonStyle || b,
              onClick: (P) => {
                qi(d.cancel) &&
                  ar &&
                  (d.cancel.onClick == null || d.cancel.onClick.call(d.cancel, P), Ga());
              },
              className: ca(
                j?.cancelButton,
                d == null || (i = d.classNames) == null ? void 0 : i.cancelButton
              ),
            },
            d.cancel.label
          )
        : null,
    R.default.isValidElement(d.action)
      ? d.action
      : d.action && qi(d.action)
        ? R.default.createElement(
            'button',
            {
              'data-button': !0,
              'data-action': !0,
              style: d.actionButtonStyle || M,
              onClick: (P) => {
                qi(d.action) &&
                  (d.action.onClick == null || d.action.onClick.call(d.action, P),
                  !P.defaultPrevented && Ga());
              },
              className: ca(
                j?.actionButton,
                d == null || (s = d.classNames) == null ? void 0 : s.actionButton
              ),
            },
            d.action.label
          )
        : null
  );
};
function Zv() {
  if (typeof window > 'u' || typeof document > 'u') return 'ltr';
  let e = document.documentElement.getAttribute('dir');
  return e === 'auto' || !e ? window.getComputedStyle(document.documentElement).direction : e;
}
function bR(e, t) {
  let a = {};
  return (
    [e, t].forEach((l, r) => {
      let o = r === 1,
        n = o ? '--mobile-offset' : '--offset',
        u = o ? hR : pR;
      function i(s) {
        ['top', 'right', 'bottom', 'left'].forEach((f) => {
          a[`${n}-${f}`] = typeof s == 'number' ? `${s}px` : s;
        });
      }
      typeof l == 'number' || typeof l == 'string'
        ? i(l)
        : typeof l == 'object'
          ? ['top', 'right', 'bottom', 'left'].forEach((s) => {
              l[s] === void 0
                ? (a[`${n}-${s}`] = u)
                : (a[`${n}-${s}`] = typeof l[s] == 'number' ? `${l[s]}px` : l[s]);
            })
          : i(u);
    }),
    a
  );
}
var iM = R.default.forwardRef(function (t, a) {
  let {
      id: l,
      invert: r,
      position: o = 'bottom-right',
      hotkey: n = ['altKey', 'KeyT'],
      expand: u,
      closeButton: i,
      className: s,
      offset: f,
      mobileOffset: d,
      theme: p = 'light',
      richColors: h,
      duration: x,
      style: v,
      visibleToasts: L = mR,
      toastOptions: c,
      dir: m = Zv(),
      gap: g = yR,
      icons: y,
      customAriaLabel: w,
      containerAriaLabel: U = 'Notifications',
    } = t,
    [C, b] = R.default.useState([]),
    M = R.default.useMemo(
      () => (l ? C.filter((ee) => ee.toasterId === l) : C.filter((ee) => !ee.toasterId)),
      [C, l]
    ),
    _ = R.default.useMemo(
      () => Array.from(new Set([o].concat(M.filter((ee) => ee.position).map((ee) => ee.position)))),
      [M, o]
    ),
    [Ue, ot] = R.default.useState([]),
    [Gt, Qe] = R.default.useState(!1),
    [be, j] = R.default.useState(!1),
    [Me, xt] = R.default.useState(
      p !== 'system'
        ? p
        : typeof window < 'u' &&
            window.matchMedia &&
            window.matchMedia('(prefers-color-scheme: dark)').matches
          ? 'dark'
          : 'light'
    ),
    ze = R.default.useRef(null),
    B = n.join('+').replace(/Key/g, '').replace(/Digit/g, ''),
    Mt = R.default.useRef(null),
    Vt = R.default.useRef(!1),
    ma = R.default.useCallback((ee) => {
      b((V) => {
        var X;
        return (
          ((X = V.find((Fe) => Fe.id === ee.id)) != null && X.delete) || yt.dismiss(ee.id),
          V.filter(({ id: Fe }) => Fe !== ee.id)
        );
      });
    }, []);
  return (
    R.default.useEffect(
      () =>
        yt.subscribe((ee) => {
          if (ee.dismiss) {
            requestAnimationFrame(() => {
              b((V) => V.map((X) => (X.id === ee.id ? { ...X, delete: !0 } : X)));
            });
            return;
          }
          setTimeout(() => {
            Wv.default.flushSync(() => {
              b((V) => {
                let X = V.findIndex((Fe) => Fe.id === ee.id);
                return X !== -1
                  ? [...V.slice(0, X), { ...V[X], ...ee }, ...V.slice(X + 1)]
                  : [ee, ...V];
              });
            });
          });
        }),
      []
    ),
    R.default.useEffect(() => {
      if (p !== 'system') {
        xt(p);
        return;
      }
      if (
        (p === 'system' &&
          (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
            ? xt('dark')
            : xt('light')),
        typeof window > 'u')
      )
        return;
      let ee = window.matchMedia('(prefers-color-scheme: dark)');
      try {
        ee.addEventListener('change', ({ matches: V }) => {
          xt(V ? 'dark' : 'light');
        });
      } catch {
        ee.addListener(({ matches: X }) => {
          try {
            xt(X ? 'dark' : 'light');
          } catch (Fe) {
            console.error(Fe);
          }
        });
      }
    }, [p]),
    R.default.useEffect(() => {
      C.length <= 1 && Qe(!1);
    }, [C]),
    R.default.useEffect(() => {
      let ee = (V) => {
        var X;
        if (n.length > 0 && n.every((F) => V[F] || V.code === F)) {
          var jt;
          Qe(!0), (jt = ze.current) == null || jt.focus();
        }
        V.code === 'Escape' &&
          (document.activeElement === ze.current ||
            ((X = ze.current) != null && X.contains(document.activeElement))) &&
          Qe(!1);
      };
      return (
        document.addEventListener('keydown', ee), () => document.removeEventListener('keydown', ee)
      );
    }, [n]),
    R.default.useEffect(() => {
      if (ze.current)
        return () => {
          Mt.current &&
            (Mt.current.focus({ preventScroll: !0 }), (Mt.current = null), (Vt.current = !1));
        };
    }, [ze.current]),
    R.default.createElement(
      'section',
      {
        ref: a,
        'aria-label': w ?? `${U} ${B}`,
        tabIndex: -1,
        'aria-live': 'polite',
        'aria-relevant': 'additions text',
        'aria-atomic': 'false',
        suppressHydrationWarning: !0,
        'data-react-aria-top-layer': !0,
      },
      _.map((ee, V) => {
        var X;
        let [Fe, jt] = ee.split('-');
        return M.length
          ? R.default.createElement(
              'ol',
              {
                key: ee,
                dir: m === 'auto' ? Zv() : m,
                tabIndex: -1,
                ref: ze,
                className: s,
                'data-sonner-toaster': !0,
                'data-sonner-theme': Me,
                'data-y-position': Fe,
                'data-x-position': jt,
                style: {
                  '--front-toast-height': `${((X = Ue[0]) == null ? void 0 : X.height) || 0}px`,
                  '--width': `${gR}px`,
                  '--gap': `${g}px`,
                  ...v,
                  ...bR(f, d),
                },
                onBlur: (F) => {
                  Vt.current &&
                    !F.currentTarget.contains(F.relatedTarget) &&
                    ((Vt.current = !1),
                    Mt.current && (Mt.current.focus({ preventScroll: !0 }), (Mt.current = null)));
                },
                onFocus: (F) => {
                  (F.target instanceof HTMLElement && F.target.dataset.dismissible === 'false') ||
                    Vt.current ||
                    ((Vt.current = !0), (Mt.current = F.relatedTarget));
                },
                onMouseEnter: () => Qe(!0),
                onMouseMove: () => Qe(!0),
                onMouseLeave: () => {
                  be || Qe(!1);
                },
                onDragEnd: () => Qe(!1),
                onPointerDown: (F) => {
                  (F.target instanceof HTMLElement && F.target.dataset.dismissible === 'false') ||
                    j(!0);
                },
                onPointerUp: () => j(!1),
              },
              M.filter((F) => (!F.position && V === 0) || F.position === ee).map((F, Fa) => {
                var pa, Jt;
                return R.default.createElement(SR, {
                  key: F.id,
                  icons: y,
                  index: Fa,
                  toast: F,
                  defaultRichColors: h,
                  duration: (pa = c?.duration) != null ? pa : x,
                  className: c?.className,
                  descriptionClassName: c?.descriptionClassName,
                  invert: r,
                  visibleToasts: L,
                  closeButton: (Jt = c?.closeButton) != null ? Jt : i,
                  interacting: be,
                  position: ee,
                  style: c?.style,
                  unstyled: c?.unstyled,
                  classNames: c?.classNames,
                  cancelButtonStyle: c?.cancelButtonStyle,
                  actionButtonStyle: c?.actionButtonStyle,
                  closeButtonAriaLabel: c?.closeButtonAriaLabel,
                  removeToast: ma,
                  toasts: M.filter((ha) => ha.position == F.position),
                  heights: Ue.filter((ha) => ha.position == F.position),
                  setHeights: ot,
                  expandByDefault: u,
                  gap: g,
                  expanded: Gt,
                  swipeDirections: t.swipeDirections,
                });
              })
            )
          : null;
      })
    )
  );
});
var Hn = class extends Error {
  kind;
  field;
  code;
  constructor(t, a = {}) {
    super(t),
      (this.name = 'AdminValidationError'),
      (this.kind = a.kind ?? 'input'),
      a.field !== void 0 && (this.field = a.field),
      a.code !== void 0 && (this.code = a.code);
  }
};
function CR(e, t) {
  return new Hn(e, t);
}
function cM(e) {
  return new Hn(e, { kind: 'action', code: 'ACTION_GUARD' });
}
function io(e) {
  return e instanceof Hn;
}
function Qc(e, t = 'Check the highlighted fields and try again.') {
  return io(e) ||
    (e instanceof Ke && e.status === 400 && e.message.trim() !== '') ||
    (e instanceof Error && e.message.trim() !== '')
    ? e.message
    : typeof e == 'string' && e.trim() !== ''
      ? e
      : t;
}
function et(e, t) {
  return { ok: !1, error: CR(e, t ? { field: t } : void 0) };
}
function bl(e, t, a) {
  let l = e.trim();
  return l ? { ok: !0, value: l } : et(`${t} is required.`, a);
}
var wR = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
function mM(e, t = 'Email', a = 'email') {
  let l = e.trim();
  return l
    ? wR.test(l)
      ? { ok: !0, value: l }
      : et(`${t} must be a valid email address.`, a)
    : et(`${t} is required.`, a);
}
function $v(e, t, a = {}) {
  let l = e.trim();
  if (!l) return et(`${t} is required.`, a.field);
  if (!/^-?\d+$/.test(l)) return et(`${t} must be an integer.`, a.field);
  let r = Number(l);
  return Number.isSafeInteger(r)
    ? a.min != null && r < a.min
      ? et(`${t} must be at least ${a.min}.`, a.field)
      : a.max != null && r > a.max
        ? et(`${t} must be at most ${a.max}.`, a.field)
        : { ok: !0, value: r }
    : et(`${t} is out of range.`, a.field);
}
function pM(e, t, a) {
  let l = $v(e, t, { min: 1, field: a });
  return l.ok, l;
}
function hM(e, t, a) {
  return $v(e, t, { min: 0, field: a });
}
function gM(e, t, a) {
  let l = e.trim();
  if (!l) return et(`${t} is required.`, a);
  try {
    let r = JSON.parse(l);
    return r == null || typeof r != 'object' || Array.isArray(r)
      ? et(`${t} must be a JSON object.`, a)
      : { ok: !0, value: r };
  } catch {
    return et(`${t} must be valid JSON.`, a);
  }
}
function yM(e, t, a = {}) {
  let l = a.fromLabel ?? 'From date',
    r = a.toLabel ?? 'To date',
    o = e.trim(),
    n = t.trim();
  if (!o) return et(`${l} is required.`, 'from');
  if (!n) return et(`${r} is required.`, 'to');
  let u = Date.parse(o),
    i = Date.parse(n);
  return Number.isNaN(u)
    ? et(`${l} is invalid.`, 'from')
    : Number.isNaN(i)
      ? et(`${r} is invalid.`, 'to')
      : u >= i
        ? et(`${l} must be before ${r}.`, 'from')
        : { ok: !0, value: { from: o, to: n } };
}
function xM(e) {
  Jv.error(Qc(e));
}
var Zc = {
    load: 'This page could not be loaded. Try again or return to the home page.',
    render: 'This page failed to render. Try reloading or go back.',
    route: 'Navigation failed. The link may be invalid or the server returned an error.',
    'not-found': 'The page you requested does not exist in this console.',
    forbidden: 'You do not have permission to view this page.',
  },
  RR = {
    load: 'Could not load page',
    render: 'Page error',
    route: 'Navigation error',
    'not-found': 'Page not found',
    forbidden: 'Access denied',
  };
function bM(e) {
  return RR[e];
}
function CM(e) {
  return Zc[e];
}
function Gi() {
  return !1;
}
function Wc(e) {
  return e !== null && typeof e == 'object';
}
function Vi(e) {
  if (
    Wc(e) &&
    (typeof e.status == 'number' ||
      (typeof e.statusText == 'string' && typeof e.status == 'number'))
  )
    return e.status;
}
function eL(e) {
  if (Wc(e)) {
    if (typeof e.statusText == 'string' && e.statusText.trim() !== '') return e.statusText;
    if (typeof e.data == 'string' && e.data.trim() !== '') return e.data;
  }
}
function tL(e, t = 'Something went wrong. Try again or return to the home page.') {
  if (io(e)) return e.message;
  if (e instanceof Ke) {
    if (e.code === 'PAYMENT_UNAVAILABLE')
      return 'Payment history is not available in this deployment. Enable the payment module or use the ledger tab for balance activity.';
    if (e.code === 'BILLING_UNAVAILABLE') {
      let r = e.message.trim();
      return r !== '' ? r : 'Billing is not available in this deployment.';
    }
    if (e.code === 'APPROVAL_REQUIRED') {
      let r = e.message.trim();
      return r !== '' ? r : 'Budget approval is required before this change can take effect.';
    }
    if (e.code === 'FEATURE_REQUIRED') {
      let r = 'This feature is not included in your license plan.',
        o = e.message.trim();
      return o !== '' ? `${r} ${o}` : r;
    }
    if (e.code === 'CLICKHOUSE_UNAVAILABLE')
      return 'Analytics store is unavailable. Report data cannot be loaded right now. Reload this tab to retry.';
    if (e.code === 'FORECAST_UNAVAILABLE') {
      let r = e.message.trim();
      return r !== ''
        ? `${r} Reload this tab to retry when ClickHouse recovers.`
        : 'Billing forecast is unavailable right now. Reload this tab to retry when ClickHouse recovers.';
    }
    if (e.status === 404) return 'The requested resource was not found.';
    if (e.status === 403) return 'You do not have permission to view this resource.';
    if (e.status === 429 || e.code === 'TOO_MANY_REQUESTS') {
      let r = e.message.trim();
      return r !== '' ? r : 'Too many requests. Wait a moment and try again.';
    }
    if (e.status === 401) {
      let r = e.message.trim();
      return r !== '' && e.code === 'UNAUTHORIZED' ? r : 'Your session expired. Sign in again.';
    }
    return e.status === 0 && e.code === 'TIMEOUT'
      ? 'The request timed out. Check your connection and try again.'
      : e.status === 501
        ? e.message.trim() !== ''
          ? e.message
          : t
        : e.status >= 500
          ? 'The server encountered an error. Try again later.'
          : e.message.trim() !== ''
            ? e.message
            : t;
  }
  let a = Vi(e);
  if (a === 404) return Zc['not-found'];
  if (a === 403) return Zc.forbidden;
  if (a != null && a >= 500) return 'The server encountered an error. Try again later.';
  let l = eL(e);
  return (
    l ||
    (e instanceof Error && e.message.trim() !== ''
      ? Gi()
        ? e.message
        : t
      : typeof e == 'string' && e.trim() !== '' && Gi()
        ? e
        : t)
  );
}
function aL(e, t) {
  let a = [];
  if (e instanceof Ke)
    a.push(`name: ${e.name}`),
      a.push(`status: ${e.status}`),
      a.push(`code: ${e.code}`),
      a.push(`message: ${e.message}`),
      e.stack && a.push('', 'stack:', e.stack);
  else if (e instanceof Error)
    a.push(`name: ${e.name}`),
      a.push(`message: ${e.message}`),
      e.stack && a.push('', 'stack:', e.stack);
  else if (Wc(e)) {
    let l = Vi(e);
    l != null && a.push(`status: ${l}`);
    let r = eL(e);
    r && a.push(`statusText: ${r}`),
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
function wM(e) {
  return Vi(e) === 404
    ? 'not-found'
    : Vi(e) === 403
      ? 'forbidden'
      : e instanceof Ke
        ? e.status === 404
          ? 'not-found'
          : e.status === 403
            ? 'forbidden'
            : 'load'
        : 'route';
}
function RM(e) {
  return e instanceof Error ? e : new Error(String(e));
}
function ji(e, t) {
  return io(e) || e instanceof Error ? e : new Error(t);
}
async function lL(e) {
  let t = e.trim();
  if (t === '') throw new Error('copyTextToClipboard: empty value');
  if (typeof navigator < 'u' && navigator.clipboard?.writeText)
    try {
      await navigator.clipboard.writeText(t);
      return;
    } catch {}
  if (typeof document > 'u') throw new Error('copyTextToClipboard: document unavailable');
  let a = document.createElement('textarea');
  (a.value = t),
    a.setAttribute('readonly', ''),
    (a.style.position = 'fixed'),
    (a.style.left = '-9999px'),
    (a.style.top = '0'),
    document.body.appendChild(a),
    a.focus(),
    a.select(),
    a.setSelectionRange(0, t.length);
  let l = !1;
  try {
    l = document.execCommand('copy');
  } finally {
    document.body.removeChild(a);
  }
  if (!l) throw new Error('copyTextToClipboard: execCommand failed');
}
var rL = {
  trackerHeaderClass: `flex h-11 shrink-0 items-center ${H.gap.md} border-b border-border px-2`,
  sectionHeaderBandClass: `flex items-center justify-between ${H.gap.md} border-b border-border ${H.inset.band}`,
  sectionHeaderBandLgClass: `flex items-center justify-between ${H.gap.md} border-b border-border ${H.inset.bandLg}`,
  sectionFooterBandLgClass: `flex flex-wrap items-center justify-end ${H.gap.md} border-t border-border ${H.inset.bandLg}`,
  tableCaptionBandClass: `${H.inset.band} ${ie.caption}`,
  compactHeaderBandClass: `${H.inset.bandCompact} ${ie.bodyMuted}`,
  sectionPanelClass: H.grid.sectionPanel,
  surfaceRaisedClass: `rounded-[8px] border border-border bg-card ${H.inset.panel} text-card-foreground shadow-none`,
};
var Wl = E(Q());
function nL({ details: e }) {
  let [t, a] = (0, oL.useState)(!1);
  if (!Gi() || e.trim() === '') return null;
  async function l() {
    try {
      await lL(e), a(!0), window.setTimeout(() => a(!1), 2e3);
    } catch {
      a(!1);
    }
  }
  return (0, Wl.jsxs)('div', {
    className: O($e.panel, 'gap-0 p-0'),
    children: [
      (0, Wl.jsxs)('div', {
        className: O(rL.sectionHeaderBandClass, 'p-2'),
        children: [
          (0, Wl.jsx)('p', {
            className: 'm-0 text-xs font-semibold text-muted-foreground',
            children: 'Developer details',
          }),
          (0, Wl.jsx)(_a, {
            type: 'button',
            variant: 'outline',
            onClick: () => void l(),
            children: t ? 'Copied' : 'Copy',
          }),
        ],
      }),
      (0, Wl.jsx)('pre', {
        className: 'max-h-48 overflow-auto p-3 text-xs font-mono',
        children: e,
      }),
    ],
  });
}
var so = E(Q());
function Xi({ title: e = 'Error', message: t, error: a, componentStack: l }) {
  let r = t ?? (a != null && io(a) ? Qc(a) : tL(a, 'Request failed.')),
    o = a != null || l ? aL(a, l) : '';
  return (0, so.jsxs)('div', {
    className: O($e.messageError),
    role: 'alert',
    children: [
      (0, so.jsx)('p', { className: 'font-semibold', children: e }),
      (0, so.jsx)('p', { children: r }),
      (0, so.jsx)(nL, { details: o }),
    ],
  });
}
var Jl = E(te());
var $l = E(Q()),
  fo = Jl.forwardRef(({ className: e, ...t }, a) =>
    (0, $l.jsx)('div', {
      ref: a,
      'data-ui-card': '',
      className: O($e.panel, fe.panelRadius, e),
      ...t,
    })
  );
fo.displayName = 'Card';
var co = Jl.forwardRef(({ className: e, ...t }, a) =>
  (0, $l.jsx)('div', { ref: a, className: O(H.stack.titleBlock, e), ...t })
);
co.displayName = 'CardHeader';
var mo = Jl.forwardRef(({ className: e, ...t }, a) =>
  (0, $l.jsx)('div', { ref: a, className: O(ie.sectionTitle, e), ...t })
);
mo.displayName = 'CardTitle';
var po = Jl.forwardRef(({ className: e, ...t }, a) =>
  (0, $l.jsx)('div', { ref: a, className: O(ie.bodyMuted, e), ...t })
);
po.displayName = 'CardDescription';
var ho = Jl.forwardRef(({ className: e, ...t }, a) =>
  (0, $l.jsx)('div', { ref: a, className: O(`grid ${H.gap.xl}`, e), ...t })
);
ho.displayName = 'CardContent';
var IR = Jl.forwardRef(({ className: e, ...t }, a) =>
  (0, $l.jsx)('div', { ref: a, className: O(H.flex.buttonGroup, e), ...t })
);
IR.displayName = 'CardFooter';
var uL = E(te());
var iL = E(Q()),
  ER =
    '[appearance:textfield] [-moz-appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none',
  go = uL.forwardRef(({ className: e, type: t, ...a }, l) =>
    (0, iL.jsx)('input', {
      type: t,
      className: O(Ql.control, 'w-full', t === 'number' && ER, e),
      ref: l,
      ...a,
    })
  );
go.displayName = 'Input';
var Yi = E(te());
var er = E(Q()),
  Nn = Yi.forwardRef(({ className: e, disabled: t, 'aria-invalid': a, ...l }, r) => {
    let [o, n] = Yi.useState(!1);
    return (0, er.jsxs)('div', {
      'aria-invalid': a,
      className: O(Ql.controlFieldGroup, 'w-full pr-1', e),
      children: [
        (0, er.jsx)('input', {
          ref: r,
          type: o ? 'text' : 'password',
          className: Ql.controlFieldInset,
          disabled: t,
          'aria-invalid': a,
          ...l,
        }),
        (0, er.jsx)('button', {
          type: 'button',
          className: O(
            'inline-flex h-6 w-6 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-0 disabled:pointer-events-none disabled:opacity-50',
            fe.nestedRadius
          ),
          'aria-label': o ? 'Hide password' : 'Show password',
          'aria-pressed': o,
          disabled: t,
          onClick: () => n((u) => !u),
          children: o
            ? (0, er.jsx)(Dn, { 'aria-hidden': !0, className: 'h-4 w-4' })
            : (0, er.jsx)(kn, { 'aria-hidden': !0, className: 'h-4 w-4' }),
        }),
      ],
    });
  });
Nn.displayName = 'PasswordInput';
var sL = E(te());
var dL = E(Q()),
  za = sL.forwardRef(({ className: e, ...t }, a) =>
    (0, dL.jsx)('label', { ref: a, className: O(fe.fieldLabelClass, e), ...t })
  );
za.displayName = 'Label';
var fL = E(te()),
  xo = E(te());
var yo = E(Q());
function AR(...e) {
  return (t) => {
    for (let a of e) a && (typeof a == 'function' ? a(t) : (a.current = t));
  };
}
var Jc = fL.forwardRef(
  ({ className: e, maxLength: t, showCount: a, value: l, onChange: r, rows: o = 3, ...n }, u) => {
    let i = (0, xo.useRef)(null),
      s = a ?? t != null,
      f = typeof l == 'string' ? l : Array.isArray(l) ? l.join('') : l != null ? String(l) : '',
      d = f.length,
      p = (0, xo.useCallback)(() => {
        let h = i.current;
        h && ((h.style.height = 'auto'), (h.style.height = `${h.scrollHeight}px`));
      }, []);
    return (
      (0, xo.useLayoutEffect)(() => {
        p();
      }, [p, f]),
      (0, yo.jsxs)('div', {
        className: 'relative',
        children: [
          (0, yo.jsx)('div', {
            className: O(Ql.panel, 'overflow-hidden'),
            children: (0, yo.jsx)('textarea', {
              ref: AR(u, i),
              rows: o,
              className: O(
                'flex min-h-[5rem] w-full resize-none overflow-hidden border-0 bg-transparent px-3 py-2 text-sm text-foreground transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
                s && t != null && 'pb-7',
                e
              ),
              ...n,
              maxLength: t,
              value: f,
              onChange: (h) => {
                r?.(h), p();
              },
            }),
          }),
          s && t != null
            ? (0, yo.jsxs)('span', {
                className:
                  'pointer-events-none absolute bottom-2 right-3 text-xs text-muted-foreground',
                'aria-hidden': 'true',
                children: [d, '/', t.toLocaleString()],
              })
            : null,
        ],
      })
    );
  }
);
Jc.displayName = 'Textarea';
var Z = E(Q());
function ED() {
  let { bootstrapComplete: e, loading: t } = oo(),
    [a, l] = (0, tr.useState)(''),
    [r, o] = (0, tr.useState)(''),
    [n, u] = (0, tr.useState)(''),
    [i, s] = (0, tr.useState)(''),
    [f, d] = (0, tr.useState)(),
    [p, h] = (0, tr.useState)(!1);
  async function x(v) {
    v.preventDefault(), d(void 0);
    let L = bl(a, 'License key', 'license_token');
    if (!L.ok) {
      d(L.error);
      return;
    }
    let c = bl(r, 'Email', 'email');
    if (!c.ok) {
      d(c.error);
      return;
    }
    let m = bl(n, 'Password', 'password');
    if (!m.ok) {
      d(m.error);
      return;
    }
    let g = bl(i, 'Team name', 'team_name');
    if (!g.ok) {
      d(g.error);
      return;
    }
    h(!0);
    try {
      await fv({ license_token: L.value, email: c.value, password: m.value, team_name: g.value }),
        window.location.replace('/');
    } catch (y) {
      d(ji(y, 'Activation failed'));
    } finally {
      h(!1);
    }
  }
  return t
    ? (0, Z.jsx)(ro, {})
    : e
      ? (0, Z.jsx)(Un, {
          children: (0, Z.jsxs)(fo, {
            children: [
              (0, Z.jsxs)(co, {
                children: [
                  (0, Z.jsx)(mo, { children: 'License already activated' }),
                  (0, Z.jsxs)(po, {
                    children: [Fi, ' is set up on this host. Sign in with your operator account.'],
                  }),
                ],
              }),
              (0, Z.jsxs)(ho, {
                className: O('grid', H.gap.xl),
                children: [
                  (0, Z.jsxs)('div', {
                    className: zv('success'),
                    role: 'status',
                    children: [
                      (0, Z.jsxs)('div', {
                        className: O('flex items-center', H.gap.md),
                        children: [
                          (0, Z.jsx)(Mn, {
                            'aria-hidden': !0,
                            className: 'h-4 w-4 shrink-0 text-admin-positive',
                            strokeWidth: 2.5,
                          }),
                          (0, Z.jsx)('p', {
                            className: O(ie.sectionTitle, 'text-admin-positive'),
                            children: 'Activation complete',
                          }),
                        ],
                      }),
                      (0, Z.jsx)('p', {
                        children:
                          'The license for this installation has already been applied. Use the email and password you created during activation.',
                      }),
                    ],
                  }),
                  (0, Z.jsx)(_a, {
                    asChild: !0,
                    className: 'w-full',
                    type: 'button',
                    variant: 'brand',
                    children: (0, Z.jsx)(Xl, { to: '/login', children: 'Go to sign in' }),
                  }),
                ],
              }),
            ],
          }),
        })
      : (0, Z.jsx)(Un, {
          children: (0, Z.jsxs)(fo, {
            children: [
              (0, Z.jsxs)(co, {
                children: [
                  (0, Z.jsx)(mo, { children: 'Activate your deployment' }),
                  (0, Z.jsx)(po, {
                    children:
                      'Create the owner account and paste the license key from your vendor. One step, then you are signed in.',
                  }),
                ],
              }),
              (0, Z.jsxs)(ho, {
                children: [
                  f ? (0, Z.jsx)(Xi, { title: 'Activation failed', error: f }) : null,
                  (0, Z.jsxs)('form', {
                    className: O('grid', H.gap.xl),
                    onSubmit: x,
                    children: [
                      (0, Z.jsxs)('div', {
                        className: O('grid', H.gap.md),
                        children: [
                          (0, Z.jsx)(za, { htmlFor: 'activate-license', children: 'License key' }),
                          (0, Z.jsx)(Jc, {
                            id: 'activate-license',
                            className: 'min-h-[5rem] w-full',
                            placeholder: 'Paste JWT from vendor email',
                            required: !0,
                            value: a,
                            onChange: (v) => l(v.target.value),
                          }),
                        ],
                      }),
                      (0, Z.jsxs)('div', {
                        className: O('grid', H.gap.md),
                        children: [
                          (0, Z.jsx)(za, { htmlFor: 'activate-email', children: 'Your email' }),
                          (0, Z.jsx)(go, {
                            id: 'activate-email',
                            type: 'email',
                            autoComplete: 'username',
                            required: !0,
                            value: r,
                            onChange: (v) => o(v.target.value),
                          }),
                        ],
                      }),
                      (0, Z.jsxs)('div', {
                        className: O('grid', H.gap.md),
                        children: [
                          (0, Z.jsx)(za, { htmlFor: 'activate-password', children: 'Password' }),
                          (0, Z.jsx)(Nn, {
                            id: 'activate-password',
                            autoComplete: 'new-password',
                            required: !0,
                            value: n,
                            onChange: (v) => u(v.target.value),
                          }),
                        ],
                      }),
                      (0, Z.jsxs)('div', {
                        className: O('grid', H.gap.md),
                        children: [
                          (0, Z.jsx)(za, {
                            htmlFor: 'activate-team',
                            children: 'Team / company name',
                          }),
                          (0, Z.jsx)(go, {
                            id: 'activate-team',
                            required: !0,
                            value: i,
                            onChange: (v) => s(v.target.value),
                          }),
                        ],
                      }),
                      (0, Z.jsx)(_i, {
                        className: 'w-full',
                        loading: p,
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
var Pn = E(te());
var Re = E(Q());
function jD() {
  let { bootstrapComplete: e, loading: t } = oo(),
    [a, l] = (0, Pn.useState)(''),
    [r, o] = (0, Pn.useState)(''),
    [n, u] = (0, Pn.useState)(),
    [i, s] = (0, Pn.useState)(!1);
  async function f(d) {
    d.preventDefault(), u(void 0);
    let p = bl(a, 'Email', 'email');
    if (!p.ok) {
      u(p.error);
      return;
    }
    let h = bl(r, 'Password', 'password');
    if (!h.ok) {
      u(h.error);
      return;
    }
    s(!0);
    try {
      await dv({ email: p.value, password: h.value }), window.location.replace('/');
    } catch (x) {
      u(ji(x, 'Sign in failed'));
    } finally {
      s(!1);
    }
  }
  return t
    ? (0, Re.jsx)(ro, {})
    : e
      ? (0, Re.jsx)(Un, {
          children: (0, Re.jsxs)(fo, {
            children: [
              (0, Re.jsxs)(co, {
                children: [
                  (0, Re.jsx)(mo, { children: 'Sign in' }),
                  (0, Re.jsx)(po, { children: Xv() }),
                ],
              }),
              (0, Re.jsxs)(ho, {
                children: [
                  n ? (0, Re.jsx)(Xi, { title: 'Sign in failed', error: n }) : null,
                  (0, Re.jsxs)('form', {
                    className: O('grid', H.gap.xl),
                    onSubmit: f,
                    children: [
                      (0, Re.jsxs)('div', {
                        className: O('grid', H.gap.md),
                        children: [
                          (0, Re.jsx)(za, { htmlFor: 'email', children: 'Email' }),
                          (0, Re.jsx)(go, {
                            id: 'email',
                            type: 'email',
                            autoComplete: 'username',
                            required: !0,
                            value: a,
                            onChange: (d) => l(d.target.value),
                          }),
                        ],
                      }),
                      (0, Re.jsxs)('div', {
                        className: O('grid', H.gap.md),
                        children: [
                          (0, Re.jsx)(za, { htmlFor: 'password', children: 'Password' }),
                          (0, Re.jsx)(Nn, {
                            id: 'password',
                            autoComplete: 'current-password',
                            required: !0,
                            value: r,
                            onChange: (d) => o(d.target.value),
                          }),
                        ],
                      }),
                      (0, Re.jsx)(_i, {
                        className: 'w-full',
                        loading: i,
                        type: 'submit',
                        children: 'Sign in',
                      }),
                    ],
                  }),
                  (0, Re.jsx)('p', {
                    className: O('text-center', ie.bodyMuted),
                    children: (0, Re.jsx)(Xl, {
                      className: 'text-primary hover:underline',
                      to: '/activate',
                      children: 'Activate with license',
                    }),
                  }),
                ],
              }),
            ],
          }),
        })
      : (0, Re.jsx)(to, { replace: !0, to: '/activate' });
}
var Ki = E(Q());
function ZD() {
  let { bootstrapComplete: e, loading: t } = oo();
  return t
    ? (0, Ki.jsx)(ro, {})
    : e
      ? (0, Ki.jsx)(to, { replace: !0, to: '/login' })
      : (0, Ki.jsx)(to, { replace: !0, to: '/activate' });
}
export {
  TR as a,
  E as b,
  te as c,
  hs as d,
  v0 as e,
  hx as f,
  gt as g,
  Ci as h,
  nC as i,
  Mx as j,
  to as k,
  xC as l,
  kx as m,
  vC as n,
  ZC as o,
  Xl as p,
  Nx as q,
  JC as r,
  Ke as s,
  In as t,
  fc as u,
  En as v,
  gE as w,
  Be as x,
  yE as y,
  xE as z,
  TE as A,
  ME as B,
  DE as C,
  kE as D,
  BE as E,
  OE as F,
  UE as G,
  HE as H,
  Bi as I,
  ZE as J,
  WE as K,
  JE as L,
  $E as M,
  ev as N,
  H as O,
  ie as P,
  xT as Q,
  vT as R,
  LT as S,
  ST as T,
  bT as U,
  q1 as V,
  CT as W,
  wT as X,
  yc as Y,
  xc as Z,
  vc as _,
  Mn as $,
  Lc as aa,
  Sc as ba,
  bc as ca,
  Cc as da,
  wc as ea,
  no as fa,
  Rc as ga,
  Ic as ha,
  Ec as ia,
  Ac as ja,
  Tc as ka,
  Mc as la,
  Dc as ma,
  kc as na,
  Bc as oa,
  Oc as pa,
  Uc as qa,
  Hc as ra,
  Nc as sa,
  Pc as ta,
  _c as ua,
  zc as va,
  Fc as wa,
  qc as xa,
  Gc as ya,
  Vc as za,
  jc as Aa,
  Xc as Ba,
  G1 as Ca,
  $e as Da,
  fe as Ea,
  DT as Fa,
  kT as Ga,
  OT as Ha,
  O as Ia,
  Ql as Ja,
  qv as Ka,
  zT as La,
  j1 as Ma,
  Vv as Na,
  Q as Oa,
  _a as Pa,
  go as Qa,
  Jv as Ra,
  iM as Sa,
  CR as Ta,
  cM as Ua,
  io as Va,
  bl as Wa,
  mM as Xa,
  $v as Ya,
  pM as Za,
  hM as _a,
  gM as $a,
  yM as ab,
  xM as bb,
  bM as cb,
  CM as db,
  Gi as eb,
  tL as fb,
  aL as gb,
  wM as hb,
  RM as ib,
  ji as jb,
  lL as kb,
  rL as lb,
  nL as mb,
  Xi as nb,
  tv as ob,
  Yv as pb,
  sE as qb,
  nw as rb,
  cE as sb,
  Fi as tb,
  ro as ub,
  _i as vb,
  ZT as wb,
  WT as xb,
  av as yb,
  lv as zb,
  rv as Ab,
  zE as Bb,
  XE as Cb,
  oo as Db,
  fo as Eb,
  co as Fb,
  mo as Gb,
  po as Hb,
  ho as Ib,
  za as Jb,
  Jc as Kb,
  Nn as Lb,
  Un as Mb,
  ED as Nb,
  jD as Ob,
  ZD as Pb,
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
lucide-react/dist/esm/icons/bug.js:
lucide-react/dist/esm/icons/building-2.js:
lucide-react/dist/esm/icons/calendar.js:
lucide-react/dist/esm/icons/check.js:
lucide-react/dist/esm/icons/chevron-down.js:
lucide-react/dist/esm/icons/chevron-left.js:
lucide-react/dist/esm/icons/chevron-right.js:
lucide-react/dist/esm/icons/copy.js:
lucide-react/dist/esm/icons/download.js:
lucide-react/dist/esm/icons/ellipsis.js:
lucide-react/dist/esm/icons/eye-off.js:
lucide-react/dist/esm/icons/eye.js:
lucide-react/dist/esm/icons/file-spreadsheet.js:
lucide-react/dist/esm/icons/inbox.js:
lucide-react/dist/esm/icons/info.js:
lucide-react/dist/esm/icons/key.js:
lucide-react/dist/esm/icons/link-2.js:
lucide-react/dist/esm/icons/loader-circle.js:
lucide-react/dist/esm/icons/log-out.js:
lucide-react/dist/esm/icons/megaphone.js:
lucide-react/dist/esm/icons/menu.js:
lucide-react/dist/esm/icons/moon.js:
lucide-react/dist/esm/icons/plug.js:
lucide-react/dist/esm/icons/plus.js:
lucide-react/dist/esm/icons/refresh-cw.js:
lucide-react/dist/esm/icons/scroll-text.js:
lucide-react/dist/esm/icons/search.js:
lucide-react/dist/esm/icons/settings.js:
lucide-react/dist/esm/icons/share-2.js:
lucide-react/dist/esm/icons/sun.js:
lucide-react/dist/esm/icons/tags.js:
lucide-react/dist/esm/icons/user.js:
lucide-react/dist/esm/icons/users.js:
lucide-react/dist/esm/icons/wrench.js:
lucide-react/dist/esm/icons/x.js:
lucide-react/dist/esm/lucide-react.js:
  (**
   * @license lucide-react v0.542.0 - ISC
   *
   * This source code is licensed under the ISC license.
   * See the LICENSE file in the root directory of this source tree.
   *)
*/
//# sourceMappingURL=chunk-732AU34H.js.map
