(function () {
  "use strict";

  var NAV_TARGETS = {
    Features: "FEATURES",
    Pricing: "PRICING",
    Install: "HOW-IT-WORKS",
    FAQ: "FAQ",
    Contacts: "CONTACTS",
    TCO: "TCO",
  };

  var THEME_STORAGE_KEY = "bidshard_theme";

  var KILLER_LUCIDE = {
    shield: "shield-check",
    funnel: "filter",
    ghost: "ghost",
    route: "route",
    scan: "scan-line",
    evidence: "file-check",
    migrate: "replace",
    auction: "gavel",
    gauge: "trending-up",
    layers: "layers",
    reconcile: "refresh-cw",
    capi: "share-2",
  };

  function siteIcon(lucideName, extraClass) {
    var cls = "site-icon" + (extraClass ? " " + extraClass : "");
    return '<i data-lucide="' + lucideName + '" class="' + cls + '" aria-hidden="true"></i>';
  }

  function refreshSiteIcons() {
    if (window.lucide && typeof window.lucide.createIcons === "function") {
      window.lucide.createIcons();
    }
  }

  function applyTheme(theme) {
    var next = theme === "light" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", next);
    try {
      localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch (e) {}
    document.querySelectorAll("[data-site-theme-toggle]").forEach(function (btn) {
      var isLight = next === "light";
      var ui = window.__siteUi || {};
      var label = isLight
        ? ui.theme_dark_label || "Switch to dark theme"
        : ui.theme_light_label || "Switch to light theme";
      var title = isLight
        ? ui.theme_dark_title || "Dark theme"
        : ui.theme_light_title || "Light theme";
      btn.setAttribute("aria-label", label);
      btn.setAttribute("title", title);
    });
    if (document.body.classList.contains("docs-page")) {
      renderDocsMermaid();
    }
  }

  function initTheme() {
    var saved = null;
    try {
      saved = localStorage.getItem(THEME_STORAGE_KEY);
    } catch (e) {}
    if (saved === "light" || saved === "dark") {
      applyTheme(saved);
      return;
    }
    applyTheme("dark");
  }

  function wireThemeToggle() {
    document.addEventListener("click", function (event) {
      var btn = event.target.closest("[data-site-theme-toggle]");
      if (!btn) {
        return;
      }
      var current = document.documentElement.getAttribute("data-theme") || "dark";
      applyTheme(current === "light" ? "dark" : "light");
      if (typeof btn.blur === "function") {
        btn.blur();
      }
    });
  }

  function siteLocale() {
    var lang = (document.documentElement.getAttribute("lang") || "en").toLowerCase();
    return lang === "uk" ? "uk" : "en";
  }

  function localeHref(path) {
    var target = String(path || "");
    if (!target) {
      return siteLocale() === "uk" ? "/uk/" : "/";
    }
    if (/^https?:\/\//i.test(target)) {
      return target;
    }
    if (target.charAt(0) !== "/") {
      target = "/" + target;
    }
    if (siteLocale() === "uk" && target.indexOf("/uk/") !== 0 && target !== "/uk") {
      return "/uk" + target;
    }
    return target;
  }

  function offerPageHref(config) {
    if (config && config.offer && config.offer.url) {
      return config.offer.url;
    }
    return siteLocale() === "uk" ? "../offer.html" : "offer.html";
  }

  function docsPageHref(config) {
    var ui = config && config.ui ? config.ui : {};
    if (ui.docs_href) {
      return ui.docs_href;
    }
    return siteLocale() === "uk" ? "/uk/docs.html" : "/docs.html";
  }

  function demoPageHref(config) {
    var ui = config && config.ui ? config.ui : {};
    if (ui.demo_href) {
      return ui.demo_href;
    }
    return siteLocale() === "uk" ? "/uk/demo.html" : "/demo.html";
  }

  function operatorsPageHref(config) {
    var ui = config && config.ui ? config.ui : {};
    if (ui.operators_href) {
      return ui.operators_href;
    }
    return siteLocale() === "uk" ? "/uk/operators.html" : "/operators.html";
  }

  function formatTemplate(template, vars) {
    var out = String(template || "");
    if (!vars) {
      return out;
    }
    Object.keys(vars).forEach(function (key) {
      out = out.split("{" + key + "}").join(String(vars[key]));
    });
    return out;
  }

  function uiText(config, key, fallback) {
    var ui = config.ui || {};
    return ui[key] || fallback || "";
  }

  function loadConfigForLocale(locale) {
    var file = locale === "uk" ? "/site.config.uk.json" : "/site.config.json";
    return fetch(file, { cache: "no-store" })
      .then(function (res) {
        return res.ok ? res.json() : {};
      })
      .catch(function () {
        return {};
      });
  }

  var localeConfigs = null;

  function preloadLocaleConfigs() {
    return Promise.all([loadConfigForLocale("en"), loadConfigForLocale("uk")]).then(function (rows) {
      localeConfigs = {
        en: normalizeConfig(rows[0]),
        uk: normalizeConfig(rows[1]),
      };
      return localeConfigs[siteLocale()] || localeConfigs.en;
    });
  }

  function escapeHtml(value) {
    return String(value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function copyLineMeta(line) {
    if (line == null) {
      return { text: "", highlight: false };
    }
    if (typeof line === "object") {
      return {
        text: line.text != null ? String(line.text) : "",
        highlight: line.highlight === true,
      };
    }
    return { text: String(line), highlight: false };
  }

  var LAYER_TO_ID = {
    FEATURES: "features",
    PRICING: "pricing",
    "HOW-IT-WORKS": "install",
    FAQ: "faq",
    CONTACTS: "contacts",
    TCO: "tco",
    ARCHITECTURE: "architecture",
  };

  var HASH_TO_LAYER = {
    features: "FEATURES",
    pricing: "PRICING",
    install: "HOW-IT-WORKS",
    faq: "FAQ",
    contacts: "CONTACTS",
    tco: "TCO",
    architecture: "ARCHITECTURE",
  };

  function scrollToLayer(layer, behavior) {
    var id = LAYER_TO_ID[layer] || String(layer).toLowerCase();
    var target =
      document.getElementById(id) ||
      document.querySelector('.Landing > [data-layer="' + layer + '"]') ||
      document.querySelector('.Landing > section[data-layer="' + layer + '"]');
    if (target) {
      target.scrollIntoView({ behavior: behavior || "smooth", block: "start" });
    }
  }

  function langSwitchHref(config) {
    if (document.body.classList.contains("demo-page")) {
      return siteLocale() === "uk" ? "/demo.html" : "/uk/demo.html";
    }
    if (document.body.classList.contains("operators-page")) {
      return siteLocale() === "uk" ? "/operators.html" : "/uk/operators.html";
    }
    if (document.body.classList.contains("docs-page")) {
      return siteLocale() === "uk" ? "/docs.html" : "/uk/docs.html";
    }
    if (document.body.classList.contains("offer-page")) {
      return siteLocale() === "uk" ? "/offer.html" : "/uk/";
    }
    var ui = config.ui || {};
    return ui.lang_switch_href || (siteLocale() === "uk" ? "/" : "/uk/");
  }

  function normalizeConfig(raw) {
    return {
      telegram_url: raw.telegram_url || "https://t.me/bidshardsupportbot",
      telegram_handle: raw.telegram_handle || "@bidshardsupportbot",
      telegram_manager_url: raw.telegram_manager_url || "https://t.me/bidshard_manager",
      telegram_manager_handle: raw.telegram_manager_handle || "@bidshard_manager",
      support_email: raw.support_email || "support@bidshard.pro",
      install_script_url: raw.install_script_url || "https://bidshard.com/get.sh",
      contacts: raw.contacts || null,
      pilot_days: raw.pilot_days || 10,
      tco: raw.tco || null,
      architecture: raw.architecture || null,
      offer: raw.offer || null,
      plans: raw.plans || [],
      hardware_sizing: raw.hardware_sizing || null,
      instant_sell: raw.instant_sell || null,
      killers: raw.killers || [],
      capabilities_compact: raw.capabilities_compact || [],
      appliance: raw.appliance || null,
      ui: raw.ui || null,
      locale: raw.locale || "en",
      copy: raw.copy || null,
      page: raw.page || null,
    };
  }

  function applyPageMeta(config) {
    var page = config.page || {};
    if (page.title) {
      document.title = page.title;
    }
    if (page.description) {
      var meta = document.querySelector('meta[name="description"]');
      if (meta) {
        meta.setAttribute("content", page.description);
      }
    }
    if (page.menu_aria) {
      var toggle = document.querySelector(".site-menu-toggle");
      if (toggle) {
        toggle.setAttribute("aria-label", page.menu_aria);
      }
    }
  }

  function copySelector(key) {
    if (/^[0-9]/.test(key)) {
      return '[class="' + key.replace(/\\/g, "\\\\").replace(/"/g, '\\"') + '"]';
    }
    return "." + key;
  }

  function applySiteCopy(config) {
    var copy = config.copy;
    if (!copy) {
      return;
    }
    Object.keys(copy).forEach(function (key) {
      var entry = copy[key];
      var nodes;
      try {
        nodes = document.querySelectorAll(copySelector(key));
      } catch (e) {
        return;
      }
      nodes.forEach(function (node) {
        if (entry && typeof entry === "object" && entry.html != null) {
          node.innerHTML = entry.html;
        } else if (entry != null) {
          node.textContent = entry;
        }
      });
    });
    applyPageMeta(config);
  }

  function refreshLanding(config) {
    applySiteCopy(config);
    renderFeatures(config);
    renderKillers(config);
    renderApplianceProof(config);
    renderPricing(config);
    renderHardwareSizing(config);
    renderTco(config);
    renderArchitecture(config);
    renderContacts(config);
    wireHeroCopy(config);
    wirePilotCopy(config);
    wireInstallCommand(config);
    var mobileNav = document.querySelector(".site-mobile-nav");
    if (mobileNav) {
      mobileNav.remove();
    }
    wireMobileMenu(config);
    updateLangSwitch(config);
    document.querySelectorAll(".NavArchitecture, .FooterArchitecture").forEach(function (link) {
      link.setAttribute("href", docsPageHref(config));
    });
    applyTheme(document.documentElement.getAttribute("data-theme") || "dark");
    refreshSiteIcons();
  }

  function switchLocale(href) {
    var targetLocale = href.indexOf("/uk") !== -1 ? "uk" : "en";
    var config = localeConfigs && localeConfigs[targetLocale];
    if (!config) {
      window.location.assign(href);
      return;
    }
    document.documentElement.setAttribute("lang", targetLocale);
    window.__siteConfig = config;
    window.__siteUi = config.ui || {};
    if (document.querySelector(".Landing")) {
      refreshLanding(config);
    } else {
      applyPageMeta(config);
      updateLangSwitch(config);
    }
    history.pushState({ bidshardLocale: targetLocale }, "", href);
  }

  function scrollFromHash() {
    var hash = (window.location.hash || "").replace(/^#/, "");
    if (!hash) {
      return;
    }
    var layer = HASH_TO_LAYER[hash.toLowerCase()] || hash.toUpperCase();
    window.requestAnimationFrame(function () {
      scrollToLayer(layer, "auto");
    });
  }

  function planPeriod(plan, config) {
    if (plan.pilot) {
      return formatTemplate(uiText(config, "plan_period_days", " / {days} days"), {
        days: config.pilot_days || 10,
      });
    }
    return plan.period || "";
  }

  function renderPricingCard(plan, config) {
    var classes = ["site-pricing-card"];
    if (plan.featured) {
      classes.push("site-pricing-card--featured");
    }
    if (plan.pilot) {
      classes.push("site-pricing-card--pilot");
    }

    var badge = plan.badge
      ? '<span class="site-pricing-badge">' +
        escapeHtml(plan.badge === "RECOMMENDED" ? uiText(config, "recommended_badge", plan.badge) : plan.badge) +
        "</span>"
      : "";

    var features = (plan.features || [])
      .map(function (line) {
        var meta = copyLineMeta(line);
        var rowClass = "site-pricing-feature";
        if (meta.highlight) {
          rowClass += " site-pricing-feature--highlight";
        }
        return (
          '<div class="' +
          rowClass +
          '">' +
          '<span class="site-pricing-feature__icon">' +
          siteIcon("check", "site-icon--sm") +
          "</span><span>" +
          escapeHtml(meta.text) +
          "</span></div>"
        );
      })
      .join("");

    var ctaClass = plan.featured || plan.pilot ? "site-pricing-cta site-pricing-cta--primary" : "site-pricing-cta";

    return (
      '<article class="' +
      classes.join(" ") +
      '" data-site-plan="' +
      escapeHtml(plan.code) +
      '">' +
      '<div class="site-pricing-card__head"><h3 class="site-pricing-card__title">' +
      escapeHtml(plan.name) +
      "</h3>" +
      badge +
      "</div>" +
      '<div class="site-pricing-card__price-row"><span class="site-pricing-card__price">' +
      escapeHtml(plan.price) +
      '</span><span class="site-pricing-card__period">' +
      escapeHtml(planPeriod(plan, config)) +
      "</span></div>" +
      '<p class="site-pricing-card__tagline">' +
      escapeHtml(plan.tagline) +
      "</p>" +
      '<div class="site-pricing-card__divider"></div>' +
      '<div class="site-pricing-card__spec-label">' +
      escapeHtml(plan.specs_label || uiText(config, "plan_specs_default", "Specifications")) +
      "</div>" +
      '<div class="site-pricing-card__features">' +
      features +
      "</div>" +
      '<button type="button" class="' +
      ctaClass +
      '" data-site-cta="telegram" data-site-plan="' +
      escapeHtml(plan.code) +
      '">' +
      escapeHtml(plan.cta || uiText(config, "plan_cta_default", "Get started")) +
      "</button></article>"
    );
  }

  function architectureFlowLevels(flow) {
    if (!flow) {
      return [];
    }
    if (flow.levels && flow.levels.length) {
      return flow.levels;
    }
    if (!flow.lanes || !flow.lanes.length) {
      return [];
    }
    var flat = [];
    flow.lanes.forEach(function (lane) {
      (lane.nodes || []).forEach(function (node) {
        flat.push(node);
      });
    });
    return flat;
  }

  function renderArchitectureFlow(flow) {
    var levels = architectureFlowLevels(flow);
    if (!levels.length) {
      return "";
    }
    var steps = [];
    levels.forEach(function (level, index) {
      steps.push('<span class="site-architecture__flow-step">' + escapeHtml(level) + "</span>");
      if (index < levels.length - 1) {
        steps.push('<span class="site-architecture__divider" aria-hidden="true">→</span>');
      }
    });
    return (
      '<div class="site-architecture__flow-strip" role="img" aria-label="' +
      escapeHtml(flow.aria_label || "Architecture flow") +
      '">' +
      '<p class="site-architecture__flow-line">' +
      steps.join("") +
      "</p></div>"
    );
  }

  function renderArchitecture(config) {
    var root = document.querySelector("[data-site-architecture]");
    if (!root || !config.architecture) {
      return;
    }
    var block = config.architecture;
    var points = (block.points || [])
      .map(function (line) {
        return (
          '<li class="site-architecture__point">' +
          '<span class="site-architecture__bullet" aria-hidden="true"></span>' +
          "<span>" +
          escapeHtml(copyLineMeta(line).text) +
          "</span></li>"
        );
      })
      .join("");
    var highlights = (block.highlights || [])
      .map(function (item) {
        return (
          '<article class="site-architecture__highlight">' +
          '<h3 class="site-architecture__highlight-title">' +
          escapeHtml(item.title || "") +
          "</h3>" +
          '<p class="site-architecture__highlight-body">' +
          escapeHtml(item.body || "") +
          "</p></article>"
        );
      })
      .join("");
    var aside =
      highlights.length > 0
        ? '<div class="site-architecture__highlights">' + highlights + "</div>"
        : "";
    root.innerHTML =
      '<div class="site-architecture__inner">' +
      '<div class="site-architecture__copy">' +
      '<div class="site-architecture__eyebrow">' +
      escapeHtml(block.eyebrow || "Architecture") +
      "</div>" +
      '<h2 id="site-architecture-title" class="site-architecture__title">' +
      escapeHtml(block.title || "") +
      "</h2>" +
      '<p class="site-architecture__subtitle">' +
      escapeHtml(block.subtitle || "") +
      "</p>" +
      '<ul class="site-architecture__list">' +
      points +
      "</ul>" +
      '<a class="site-architecture__link" href="' +
      escapeHtml(localeHref(block.url || "/docs.html")) +
      '">' +
      escapeHtml(block.cta || "Read architecture docs") +
      " →</a>" +
      "</div>" +
      aside +
      "</div>" +
      renderArchitectureFlow(block.flow);
  }

  function renderContacts(config) {
    var root = document.querySelector("[data-site-contacts]");
    if (!root) {
      return;
    }
    var block = config.contacts || {};
    var handle = config.telegram_handle || "@bidshardsupportbot";
    var url = config.telegram_url || "https://t.me/bidshardsupportbot";
    var managerHandle = config.telegram_manager_handle || "@bidshard_manager";
    var managerUrl = config.telegram_manager_url || "https://t.me/bidshard_manager";
    var supportEmail = config.support_email || "support@bidshard.pro";
    root.innerHTML =
      '<div class="site-contacts__inner">' +
      '<div class="site-contacts__copy">' +
      '<div class="site-contacts__eyebrow">' +
      escapeHtml(block.eyebrow || "Contacts") +
      "</div>" +
      '<h2 id="site-contacts-title" class="site-contacts__title">' +
      escapeHtml(block.title || "Support and sales on Telegram") +
      "</h2>" +
      '<p class="site-contacts__subtitle">' +
      escapeHtml(block.subtitle || "") +
      "</p>" +
      (block.relocation_note
        ? '<p class="site-contacts__relocation">' + escapeHtml(block.relocation_note) + "</p>"
        : "") +
      '<div class="site-contacts__handles">' +
      '<a class="site-contacts__handle" href="#" data-site-cta="telegram" data-site-telegram-url="' +
      escapeHtml(url) +
      '">' +
      escapeHtml(handle) +
      "</a>" +
      '<span class="site-contacts__handle-sep" aria-hidden="true">·</span>' +
      '<a class="site-contacts__handle" href="#" data-site-cta="telegram" data-site-telegram-url="' +
      escapeHtml(managerUrl) +
      '">' +
      escapeHtml(managerHandle) +
      "</a>" +
      '<span class="site-contacts__handle-sep" aria-hidden="true">·</span>' +
      '<a class="site-contacts__handle" href="mailto:' +
      escapeHtml(supportEmail) +
      '">' +
      escapeHtml(supportEmail) +
      "</a></div></div>" +
      '<a class="site-contacts__cta BtnPrimary" href="#" data-site-cta="telegram" data-site-telegram-url="' +
      escapeHtml(url) +
      '">' +
      escapeHtml(block.cta || "Message on Telegram") +
      "</a>" +
      "</div>";
  }

  function renderTcoImpacts(tco) {
    var impacts = tco.impacts;
    if (!impacts || !impacts.rows || !impacts.rows.length) {
      return "";
    }
    var headers = impacts.headers || {};
    var rowsHtml = impacts.rows
      .map(function (row) {
        return (
          '<tr class="site-tco-impact__row">' +
          '<th scope="row" class="site-tco-impact__label">' +
          escapeHtml(row.label || "") +
          "</th>" +
          '<td class="site-tco-impact__patchwork">' +
          escapeHtml(row.patchwork || "") +
          "</td>" +
          '<td class="site-tco-impact__bidshard">' +
          escapeHtml(row.bidshard || "") +
          "</td>" +
          '<td class="site-tco-impact__delta">' +
          '<span class="site-tco-impact__delta-value">' +
          escapeHtml(row.delta || "") +
          "</span>" +
          (row.delta_note
            ? '<span class="site-tco-impact__delta-note">' + escapeHtml(row.delta_note) + "</span>"
            : "") +
          "</td></tr>"
        );
      })
      .join("");

    var totalHtml = impacts.total
      ? '<div class="site-tco-impact__total">' +
        '<div class="site-tco-impact__total-value">' +
        escapeHtml(impacts.total.value || "") +
        "</div>" +
        '<div class="site-tco-impact__total-label">' +
        escapeHtml(impacts.total.label || "") +
        "</div>" +
        (impacts.total.body
          ? '<p class="site-tco-impact__total-body">' + escapeHtml(impacts.total.body) + "</p>"
          : "") +
        "</div>"
      : "";

    return (
      '<div class="site-tco__impacts">' +
      '<div class="site-tco-impact__head">' +
      '<h3 class="site-tco-impact__title">' +
      escapeHtml(impacts.title || "") +
      "</h3>" +
      (impacts.subtitle
        ? '<p class="site-tco-impact__subtitle">' + escapeHtml(impacts.subtitle) + "</p>"
        : "") +
      "</div>" +
      '<div class="site-tco-impact__table-wrap">' +
      '<table class="site-tco-impact__table">' +
      "<thead><tr>" +
      '<th scope="col">' +
      escapeHtml(headers.driver || "Driver") +
      "</th>" +
      '<th scope="col">' +
      escapeHtml(headers.patchwork || "Patchwork") +
      "</th>" +
      '<th scope="col">' +
      escapeHtml(headers.bidshard || "BidShard") +
      "</th>" +
      '<th scope="col">' +
      escapeHtml(headers.impact || "TCO impact") +
      "</th>" +
      "</tr></thead><tbody>" +
      rowsHtml +
      "</tbody></table></div>" +
      totalHtml +
      "</div>"
    );
  }

  function renderTco(config) {
    var root = document.querySelector("[data-site-tco]");
    if (!root || !config.tco) {
      return;
    }
    var tco = config.tco;
    var columns = (tco.columns || [])
      .map(function (col) {
        var items = (col.items || [])
          .map(function (line) {
            return (
              '<li class="site-tco-card__item">' +
              '<span class="site-tco-card__bullet" aria-hidden="true"></span>' +
              "<span>" +
              escapeHtml(copyLineMeta(line).text) +
              "</span></li>"
            );
          })
          .join("");
        var cardClass = "site-tco-card";
        if (col.highlight) {
          cardClass += " site-tco-card--highlight";
        }
        return (
          '<article class="' +
          cardClass +
          '">' +
          '<h3 class="site-tco-card__label">' +
          escapeHtml(col.label || "") +
          "</h3>" +
          '<ul class="site-tco-card__list">' +
          items +
          "</ul>" +
          (col.summary
            ? '<p class="site-tco-card__summary">' + escapeHtml(col.summary) + "</p>"
            : "") +
          (col.footnote
            ? '<p class="site-tco-card__footnote">' + escapeHtml(col.footnote) + "</p>"
            : "") +
          "</article>"
        );
      })
      .join("");

    root.innerHTML =
      '<div class="site-tco__inner">' +
      '<div class="site-tco__head">' +
      '<div class="site-tco__eyebrow">' +
      escapeHtml(tco.eyebrow || "Total cost") +
      "</div>" +
      '<h2 id="site-tco-title" class="site-tco__title">' +
      escapeHtml(tco.title || "") +
      "</h2>" +
      '<p class="site-tco__subtitle">' +
      escapeHtml(tco.subtitle || "") +
      "</p>" +
      "</div>" +
      '<div class="site-tco__grid">' +
      columns +
      "</div>" +
      renderTcoImpacts(tco) +
      "</div>";
  }

  function renderPricing(config) {
    var grid = document.querySelector("[data-site-pricing-grid]");
    if (!grid || !config.plans || !config.plans.length) {
      return;
    }
    grid.innerHTML = config.plans.map(function (plan) {
      return renderPricingCard(plan, config);
    }).join("");
  }

  function planMetaByCode(config) {
    var map = {};
    (config.plans || []).forEach(function (plan) {
      if (plan.code) {
        map[plan.code] = plan;
      }
    });
    return map;
  }

  function renderHardwareSpecRow(label, value) {
    if (!value) {
      return "";
    }
    return (
      '<div class="site-hardware-card__spec-row">' +
      '<dt class="site-hardware-card__spec-label">' +
      escapeHtml(label) +
      "</dt>" +
      '<dd class="site-hardware-card__spec-value">' +
      escapeHtml(value) +
      "</dd></div>"
    );
  }

  function renderHardwareSizing(config) {
    var root = document.querySelector("[data-site-hardware-sizing]");
    var block = config.hardware_sizing;
    if (!root || !block || !block.tiers || !block.tiers.length) {
      if (root) {
        root.innerHTML = "";
      }
      return;
    }

    var labels = block.labels || {};
    var planMeta = planMetaByCode(config);
    var cards = block.tiers
      .map(function (tier) {
        var meta = planMeta[tier.code] || {};
        var cardClass = "site-hardware-card";
        if (meta.featured) {
          cardClass += " site-hardware-card--featured";
        }
        if (meta.pilot) {
          cardClass += " site-hardware-card--pilot";
        }
        if (meta.enterprise) {
          cardClass += " site-hardware-card--enterprise";
        }
        var priceLine = meta.price
          ? '<div class="site-hardware-card__price">' +
            escapeHtml(meta.price) +
            escapeHtml(meta.period || "") +
            "</div>"
          : "";

        return (
          '<article class="' +
          cardClass +
          '" data-site-hardware-tier="' +
          escapeHtml(tier.code || "") +
          '">' +
          '<div class="site-hardware-card__head">' +
          '<h3 class="site-hardware-card__plan">' +
          escapeHtml(tier.plan || meta.name || "") +
          "</h3>" +
          priceLine +
          "</div>" +
          '<dl class="site-hardware-card__specs">' +
          renderHardwareSpecRow(labels.hosts || "Hosts", tier.hosts) +
          renderHardwareSpecRow(labels.cpu || "CPU", tier.cpu) +
          renderHardwareSpecRow(labels.ram || "RAM", tier.ram) +
          renderHardwareSpecRow(labels.disk || "Storage", tier.disk) +
          "</dl>" +
          (tier.profile
            ? '<p class="site-hardware-card__profile">' + escapeHtml(tier.profile) + "</p>"
            : "") +
          "</article>"
        );
      })
      .join("");

    root.innerHTML =
      '<div class="site-hardware-sizing__inner">' +
      '<div class="site-hardware-sizing__head">' +
      '<div class="site-hardware-sizing__eyebrow">' +
      escapeHtml(block.eyebrow || "Hardware sizing") +
      "</div>" +
      '<h2 id="site-hardware-title" class="site-hardware-sizing__title">' +
      escapeHtml(block.title || "") +
      "</h2>" +
      '<p class="site-hardware-sizing__subtitle">' +
      escapeHtml(block.subtitle || "") +
      "</p></div>" +
      '<div class="site-hardware-sizing__grid">' +
      cards +
      "</div>" +
      (block.footnote
        ? '<p class="site-hardware-sizing__footnote">' + escapeHtml(block.footnote) + "</p>"
        : "") +
      "</div>";
  }

  function formatUsd(amount) {
    return "$" + Math.round(Number(amount) || 0).toLocaleString("en-US");
  }

  function killerIcon(name) {
    return siteIcon(KILLER_LUCIDE[name] || KILLER_LUCIDE.shield, "site-icon--killer");
  }

  function renderMarginGuard(block) {
    var root = document.querySelector("[data-site-margin-guard-root]");
    if (!root || !block) {
      return;
    }
    root.innerHTML =
      '<div class="site-margin-guard">' +
      '<div class="site-margin-guard__eyebrow">' +
      escapeHtml(block.eyebrow || "Margin Guard") +
      "</div>" +
      '<h3 class="site-margin-guard__title">' +
      escapeHtml(block.title || "") +
      "</h3>" +
      (block.metric
        ? '<div class="site-margin-guard__metric">' + escapeHtml(block.metric) + "</div>"
        : "") +
      '<p class="site-margin-guard__body">' +
      escapeHtml(block.body || "") +
      "</p>" +
      (block.footnote
        ? '<p class="site-margin-guard__footnote">' + escapeHtml(block.footnote) + "</p>"
        : "") +
      "</div>";
  }

  function renderFeatures(config) {
    var instant = config.instant_sell;
    if (instant && instant.margin_guard) {
      renderMarginGuard(instant.margin_guard);
    }
    renderCapabilitiesCompact(config);
  }

  function renderKillers(config) {
    var root = document.querySelector("[data-site-killers-root]");
    if (!root || !config.killers || !config.killers.length) {
      return;
    }
    root.innerHTML =
      '<div class="site-killers__grid">' +
      config.killers
        .map(function (item) {
          return (
            '<article class="site-killer-card" data-site-killer="' +
            escapeHtml(item.id) +
            '">' +
            '<div class="site-killer-card__head">' +
            '<div class="site-killer-card__icon">' +
            killerIcon(item.icon) +
            "</div>" +
            '<span class="site-killer-tier">' +
            escapeHtml(item.tier || "") +
            "</span></div>" +
            '<h3 class="site-killer-card__title">' +
            escapeHtml(item.title || "") +
            "</h3>" +
            '<p class="site-killer-card__text">' +
            escapeHtml(item.body || "") +
            "</p></article>"
          );
        })
        .join("") +
      "</div>";
  }

  function renderCapabilitiesCompact(config) {
    var root = document.querySelector("[data-site-capabilities-root]");
    if (!root || !config.capabilities_compact || !config.capabilities_compact.length) {
      return;
    }
    root.innerHTML =
      '<ul class="site-capabilities-compact">' +
      config.capabilities_compact
        .map(function (line) {
          return (
            '<li class="site-capabilities-compact__item">' +
            '<span class="site-capabilities-compact__icon">' +
            siteIcon("check", "site-icon--sm") +
            '</span><span class="site-capabilities-compact__text">' +
            escapeHtml(copyLineMeta(line).text) +
            "</span></li>"
          );
        })
        .join("") +
      "</ul>";
  }

  function renderApplianceProof(config) {
    var root = document.querySelector("[data-site-appliance-proof]");
    var block = config.appliance;
    if (!root || !block) {
      return;
    }
    root.innerHTML =
      '<div class="site-appliance-proof__copy">' +
      (block.eyebrow
        ? '<p class="site-appliance-proof__eyebrow">' + escapeHtml(block.eyebrow) + "</p>"
        : "") +
      '<h2 class="site-appliance-proof__title">' +
      escapeHtml(block.title || "") +
      "</h2>" +
      '<p class="site-appliance-proof__subtitle">' +
      escapeHtml(block.subtitle || "") +
      "</p></div>";
    var installTitle = document.querySelector("[data-site-install-cli-title]");
    if (installTitle && block.install_cli_title) {
      installTitle.textContent = block.install_cli_title;
    }
  }

  function wireInstallCopy() {
    if (window.__siteInstallCopyWired) {
      return;
    }
    window.__siteInstallCopyWired = true;
    document.addEventListener("click", function (event) {
      var btn = event.target.closest("[data-site-copy-install]");
      if (!btn) {
        return;
      }
      var line = document.querySelector(".CurlFsslHttpsReleasesExampleComGetShBash");
      if (!line) {
        return;
      }
      var text = line.textContent.replace(/^\$\s*/, "").trim();
      if (!text || !navigator.clipboard || !navigator.clipboard.writeText) {
        return;
      }
      event.preventDefault();
      navigator.clipboard.writeText(text).then(function () {
        btn.setAttribute("data-copied", "1");
        window.setTimeout(function () {
          btn.removeAttribute("data-copied");
        }, 1600);
      });
    });
  }

  function wireHeroCopy(config) {
    var block = config.hero || {};
    var eyebrow = document.querySelector("[data-site-hero-eyebrow]");
    if (eyebrow && block.eyebrow) {
      eyebrow.textContent = block.eyebrow;
    }
    document.querySelectorAll("[data-site-hero-docs]").forEach(function (link) {
      link.setAttribute("href", docsPageHref(config));
    });
    document.querySelectorAll("[data-site-hero-demo]").forEach(function (link) {
      link.setAttribute("href", demoPageHref(config));
      var demoLabel = uiText(config, "hero_demo_cta", "View demo");
      if (demoLabel) {
        link.textContent = demoLabel;
      }
    });
  }

  function wirePilotCopy(config) {
    var days = config.pilot_days || 10;
    var offerLinkText = uiText(config, "offer_link_text", "public offer");
    var offerHref = offerPageHref(config);
    var badge = document.querySelector("[data-site-pilot-badge]");
    if (badge) {
      badge.textContent = formatTemplate(uiText(config, "pilot_badge", "Self-hosted · Free {days}-day pilot"), {
        days: days,
      });
    }
    var headline = document.querySelector("[data-site-pilot-cta-headline]");
    if (headline) {
      headline.textContent = formatTemplate(uiText(config, "pilot_headline", "Try free for {days} days"), {
        days: days,
      });
    }
    var disclaimer = document.querySelector("[data-site-pilot-disclaimer]");
    if (disclaimer) {
      var disclaimerText = formatTemplate(
        uiText(
          config,
          "pilot_disclaimer",
          "No credit card. Free {days}-day pilot. By requesting a pilot you accept the {offer_link}."
        ),
        { days: days, offer_link: offerLinkText }
      );
      disclaimer.innerHTML = disclaimerText.replace(
        offerLinkText,
        '<a href="' + escapeHtml(offerHref) + '">' + escapeHtml(offerLinkText) + "</a>"
      );
    }
    document.querySelectorAll(".KRps1ServerRulesOnlyAntifraudMessageUsOnTelegramWithExpectedTrafficVpsSpec").forEach(function (el) {
      el.textContent = uiText(
        config,
        "pilot_cta_subline",
        "1 server · full click routing · message us on Telegram with expected traffic and VPS spec"
      );
    });
  }

  function wireOfferDisclaimer() {
    document.querySelectorAll("[data-site-offer-disclaimer]").forEach(function (el) {
      if (el.querySelector("a[href*='offer']")) {
        return;
      }
    });
  }

  function wireInstallCommand(config) {
    document.querySelectorAll(".CurlFsslHttpsReleasesExampleComGetShBash").forEach(function (el) {
      el.textContent = "$ curl -fsSL " + config.install_script_url + " | bash";
    });
  }

  function wireTelegramLinks(config) {
    document.querySelectorAll("[data-site-telegram]").forEach(function (el) {
      if (el.tagName === "A") {
        el.href = config.telegram_url;
        el.target = "_blank";
        el.rel = "noopener noreferrer";
      }
    });
  }

  function wireAccordions() {
    document.querySelectorAll(".Landing .Faq .AccordionItem").forEach(function (item) {
      if (item.children.length < 2) {
        return;
      }

      var trigger = item.children[0];
      if (trigger.classList.contains("AccordionItem__trigger")) {
        return;
      }
      trigger.classList.add("AccordionItem__trigger");
      trigger.setAttribute("role", "button");
      trigger.setAttribute("tabindex", "0");
      trigger.setAttribute("aria-expanded", "false");

      function toggle() {
        var open = item.classList.toggle("is-open");
        trigger.setAttribute("aria-expanded", open ? "true" : "false");
      }

      trigger.addEventListener("click", toggle);
      trigger.addEventListener("keydown", function (event) {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          toggle();
        }
      });
    });
  }

  function wireMobileMenu(config) {
    var toggle = document.querySelector(".site-menu-toggle");
    if (!toggle) {
      return;
    }

    var docsHref = docsPageHref(config);
    var nav = document.createElement("nav");
    nav.className = "site-mobile-nav";
    nav.setAttribute("aria-label", "Mobile");
    nav.innerHTML =
      '<a href="#" data-site-scroll="FEATURES">' +
      escapeHtml(uiText(config, "mobile_nav_features", "Features")) +
      "</a>" +
      '<a href="#" data-site-scroll="PRICING">' +
      escapeHtml(uiText(config, "mobile_nav_pricing", "Pricing")) +
      "</a>" +
      '<a href="' +
      escapeHtml(docsHref) +
      '">' +
      escapeHtml(uiText(config, "mobile_nav_architecture", "Architecture")) +
      "</a>" +
      '<a href="#" data-site-scroll="HOW-IT-WORKS">' +
      escapeHtml(uiText(config, "mobile_nav_install", "Install")) +
      "</a>" +
      '<a href="#" data-site-scroll="FAQ">' +
      escapeHtml(uiText(config, "mobile_nav_faq", "FAQ")) +
      "</a>" +
      '<a href="#" data-site-scroll="CONTACTS">' +
      escapeHtml(uiText(config, "mobile_nav_contacts", "Contacts")) +
      "</a>" +
      '<a href="#" class="site-mobile-nav__cta" data-site-cta="telegram">' +
      escapeHtml(uiText(config, "mobile_nav_cta", "Get free pilot")) +
      "</a>";
    document.body.appendChild(nav);

    function closeNav() {
      nav.classList.remove("is-open");
      toggle.setAttribute("aria-expanded", "false");
    }

    toggle.addEventListener("click", function () {
      var open = nav.classList.toggle("is-open");
      toggle.setAttribute("aria-expanded", open ? "true" : "false");
    });

    nav.querySelectorAll("[data-site-scroll]").forEach(function (link) {
      link.addEventListener("click", function (event) {
        event.preventDefault();
        closeNav();
        scrollToLayer(link.getAttribute("data-site-scroll"));
      });
    });
  }

  var OFFER_STORAGE_KEY = "bidshard_offer_accept";

  function readOfferAcceptance(version) {
    try {
      var raw = window.localStorage.getItem(OFFER_STORAGE_KEY);
      if (!raw) {
        return null;
      }
      var data = JSON.parse(raw);
      if (!data || data.version !== version) {
        return null;
      }
      return data;
    } catch (err) {
      return null;
    }
  }

  function saveOfferAcceptance(version) {
    window.localStorage.setItem(
      OFFER_STORAGE_KEY,
      JSON.stringify({
        version: version,
        accepted_at: new Date().toISOString(),
      })
    );
  }

  function openTelegram(config, url) {
    var target = url || config.telegram_url || "https://t.me/bidshardsupportbot";
    window.open(target, "_blank", "noopener,noreferrer");
  }

  function ensureOfferAccepted(config, onAllowed) {
    var offer = config.offer || {};
    var version = offer.version || "2026-09-09";
    if (readOfferAcceptance(version)) {
      onAllowed();
      return;
    }
    showOfferGate(config, onAllowed);
  }

  function showOfferGate(config, onAllowed) {
    var root = document.querySelector("[data-site-offer-gate]");
    if (!root) {
      onAllowed();
      return;
    }
    var offer = config.offer || {};
    var version = offer.version || "2026-09-09";
    var points = (offer.points || [])
      .map(function (line) {
        return "<li>" + escapeHtml(copyLineMeta(line).text) + "</li>";
      })
      .join("");

    root.innerHTML =
      '<div class="site-offer-gate__backdrop" data-site-offer-close></div>' +
      '<div class="site-offer-gate__dialog" role="dialog" aria-modal="true" aria-labelledby="site-offer-gate-title">' +
      '<div class="site-offer-gate__head">' +
      '<h2 id="site-offer-gate-title" class="site-offer-gate__title">' +
      escapeHtml(offer.title || "Accept the public offer") +
      "</h2>" +
      '<p class="site-offer-gate__intro">' +
      escapeHtml(offer.intro || "") +
      "</p></div>" +
      '<div class="site-offer-gate__body">' +
      '<div class="site-offer-gate__scroll" data-site-offer-scroll tabindex="0">' +
      '<p class="site-offer-gate__version">' +
      escapeHtml(uiText(config, "offer_gate_version", "Offer version:")) +
      " <strong>" +
      escapeHtml(version) +
      "</strong></p>" +
      "<ul class=\"site-offer-gate__list\">" +
      points +
      "</ul>" +
      '<p class="site-offer-gate__hint">' +
      escapeHtml(uiText(config, "offer_gate_scroll_hint", "Scroll to the end to enable acceptance.")) +
      "</p>" +
      "</div></div>" +
      '<div class="site-offer-gate__footer">' +
      '<label class="site-offer-gate__check is-locked">' +
      '<input type="checkbox" class="site-offer-gate__checkbox-input" disabled data-site-offer-checkbox />' +
      '<span class="site-offer-gate__check-ui" aria-hidden="true">' +
      siteIcon("check", "site-icon--xs") +
      "</span>" +
      '<span class="site-offer-gate__check-text">' +
      escapeHtml(offer.checkbox || "I accept the public offer.") +
      "</span></label>" +
      '<div class="site-offer-gate__actions">' +
      '<button type="button" class="site-offer-gate__accept BtnPrimary" disabled data-site-offer-accept>' +
      escapeHtml(offer.accept_cta || "I accept") +
      "</button>" +
      '<a class="site-offer-gate__read" href="' +
      escapeHtml(offerPageHref(config)) +
      '" target="_blank" rel="noopener noreferrer">' +
      escapeHtml(offer.read_cta || "Read full public offer") +
      "</a>" +
      "</div></div></div>";

    root.hidden = false;
    root.setAttribute("aria-hidden", "false");
    document.body.classList.add("site-offer-gate-open");
    refreshSiteIcons();

    var scrollBox = root.querySelector("[data-site-offer-scroll]");
    var checkbox = root.querySelector("[data-site-offer-checkbox]");
    var checkLabel = root.querySelector(".site-offer-gate__check");
    var acceptBtn = root.querySelector("[data-site-offer-accept]");

    function scrolledToEnd() {
      if (!scrollBox) {
        return true;
      }
      return scrollBox.scrollTop + scrollBox.clientHeight >= scrollBox.scrollHeight - 8;
    }

    function syncControls() {
      var enabled = scrolledToEnd();
      if (checkbox) {
        checkbox.disabled = !enabled;
        if (!enabled) {
          checkbox.checked = false;
        }
      }
      if (checkLabel) {
        checkLabel.classList.toggle("is-locked", !enabled);
        checkLabel.classList.toggle("is-checked", Boolean(checkbox && checkbox.checked));
      }
      if (acceptBtn) {
        acceptBtn.disabled = !enabled || !checkbox || !checkbox.checked;
      }
    }

    if (scrollBox) {
      scrollBox.addEventListener("scroll", syncControls);
      syncControls();
    }

    if (checkbox) {
      checkbox.addEventListener("change", syncControls);
    }

    function closeGate() {
      root.hidden = true;
      root.setAttribute("aria-hidden", "true");
      document.body.classList.remove("site-offer-gate-open");
    }

    root.querySelectorAll("[data-site-offer-close]").forEach(function (el) {
      el.addEventListener("click", closeGate);
    });

    if (acceptBtn) {
      acceptBtn.addEventListener("click", function () {
        if (acceptBtn.disabled) {
          return;
        }
        saveOfferAcceptance(version);
        closeGate();
        onAllowed();
      });
    }
  }

  function wireScrollTargets() {
    if (window.__siteScrollTargetsWired) {
      return;
    }
    window.__siteScrollTargetsWired = true;
    document.addEventListener("click", function (event) {
      var el = event.target.closest("[data-site-scroll]");
      if (!el) {
        return;
      }
      event.preventDefault();
      scrollToLayer(el.getAttribute("data-site-scroll"));
    });
  }

  function wireInteractions() {
    if (window.__siteInteractionsWired) {
      return;
    }
    window.__siteInteractionsWired = true;

    document.addEventListener("click", function (event) {
      var config = window.__siteConfig || {};
      var nav = event.target.closest("[data-site-nav]");
      if (nav) {
        event.preventDefault();
        scrollToLayer(NAV_TARGETS[nav.getAttribute("data-site-nav")]);
        return;
      }

      if (event.target.closest('[data-site-cta="telegram"]')) {
        event.preventDefault();
        var telegramEl = event.target.closest('[data-site-cta="telegram"]');
        var telegramUrl = telegramEl ? telegramEl.getAttribute("data-site-telegram-url") : "";
        ensureOfferAccepted(config, function () {
          openTelegram(config, telegramUrl || undefined);
        });
        return;
      }

      if (event.target.closest('[data-site-cta="install"]')) {
        event.preventDefault();
        scrollToLayer("install-cli");
        return;
      }

      if (event.target.closest('[data-site-link="offer"]')) {
        event.preventDefault();
        window.location.href = offerPageHref(config);
      }
    });
  }

  function updateLangSwitch(config) {
    var ui = config.ui || {};
    var href = langSwitchHref(config);
    document.querySelectorAll(".site-lang-switch, [data-site-lang-switch]").forEach(function (el) {
      el.setAttribute("href", href);
      if (ui.lang_switch_label) {
        el.textContent = ui.lang_switch_label;
      }
      if (ui.lang_switch_title) {
        el.setAttribute("title", ui.lang_switch_title);
      }
    });
  }

  function wireLangSwitch() {
    if (window.__siteLangSwitchWired) {
      return;
    }
    window.__siteLangSwitchWired = true;

    document.addEventListener("click", function (event) {
      var link = event.target.closest(".site-lang-switch, [data-site-lang-switch]");
      if (!link) {
        return;
      }
      if (document.body.classList.contains("docs-page") || document.body.classList.contains("offer-page")) {
        return;
      }
      event.preventDefault();
      var href = link.getAttribute("href");
      if (!href) {
        return;
      }
      switchLocale(href);
    });
  }

  function initLanding(config) {
    wireAccordions();
    applySiteCopy(config);
    renderFeatures(config);
    renderKillers(config);
    renderApplianceProof(config);
    renderPricing(config);
    renderHardwareSizing(config);
    renderTco(config);
    renderArchitecture(config);
    renderContacts(config);
    wireHeroCopy(config);
    wirePilotCopy(config);
    wireOfferDisclaimer();
    wireInstallCommand(config);
    wireInstallCopy();
    wireMobileMenu(config);
    wireScrollTargets();
    updateLangSwitch(config);
    refreshSiteIcons();
  }

  function initDocs(config) {
    updateLangSwitch(config);
    applyTheme(document.documentElement.getAttribute("data-theme") || "dark");
    var homeHref = siteLocale() === "uk" ? "/uk/" : "/";
    document.querySelectorAll(".site-doc-header a[href='index.html'], .site-doc-back").forEach(function (link) {
      link.setAttribute("href", homeHref);
    });
    document.querySelectorAll(".site-doc-header__nav a[href='offer.html']").forEach(function (link) {
      link.setAttribute("href", offerPageHref(config));
    });
    document.querySelectorAll(".site-doc-header__nav a[href='docs.html']").forEach(function (link) {
      link.setAttribute("href", docsPageHref(config));
    });
    document.querySelectorAll("a[href='operators.html']").forEach(function (link) {
      link.setAttribute("href", operatorsPageHref(config));
    });
    document.querySelectorAll(".site-doc-actions a[href='docs.html']").forEach(function (link) {
      link.setAttribute("href", docsPageHref(config));
    });
    document.querySelectorAll("a[href='index.html#pricing'], a[href='index.html#tco']").forEach(function (link) {
      var hash = link.getAttribute("href").split("#")[1] || "";
      link.setAttribute("href", homeHref + "#" + hash);
    });
  }

  function mermaidThemeVariables(isLight) {
    if (isLight) {
      return {
        background: "transparent",
        primaryColor: "#ffffff",
        primaryTextColor: "#2c2c2c",
        primaryBorderColor: "#c8cacf",
        secondaryColor: "#fafbfc",
        tertiaryColor: "#ebeef3",
        lineColor: "#6b7280",
        textColor: "#2c2c2c",
        mainBkg: "#ffffff",
        nodeBorder: "#c8cacf",
        clusterBkg: "#fafbfc",
        clusterBorder: "#d1d5db",
        titleColor: "#234d6e",
        edgeLabelBackground: "#ffffff",
        nodeTextColor: "#2c2c2c",
        fontFamily: "Google Sans, system-ui, sans-serif",
      };
    }
    return {
      darkMode: true,
      background: "transparent",
      primaryColor: "#333333",
      primaryTextColor: "#d4d4d4",
      primaryBorderColor: "#616161",
      secondaryColor: "#2e2e2e",
      tertiaryColor: "#282828",
      lineColor: "#8a8a8a",
      textColor: "#d4d4d4",
      mainBkg: "#333333",
      nodeBorder: "#616161",
      clusterBkg: "#282828",
      clusterBorder: "#4a4a4a",
      titleColor: "#8ec8f0",
      edgeLabelBackground: "#333333",
      nodeTextColor: "#d4d4d4",
      fontFamily: "Google Sans, system-ui, sans-serif",
    };
  }

  function renderDocsMermaid() {
    var blocks = document.querySelectorAll(".site-doc-mermaid");
    if (!blocks.length || typeof mermaid === "undefined") {
      return;
    }
    var isLight = document.documentElement.getAttribute("data-theme") === "light";
    mermaid.initialize({
      startOnLoad: false,
      theme: "base",
      themeVariables: mermaidThemeVariables(isLight),
      securityLevel: "strict",
      flowchart: { curve: "basis", htmlLabels: true, padding: 14 },
    });
    blocks.forEach(function (el) {
      if (!el.dataset.mermaidSource) {
        el.dataset.mermaidSource = el.textContent.trim();
      }
      el.removeAttribute("data-processed");
      el.innerHTML = el.dataset.mermaidSource;
    });
    mermaid.run({ nodes: Array.from(blocks) }).catch(function () {});
  }

  function initOffer(config) {
    updateLangSwitch(config);
    applyTheme(document.documentElement.getAttribute("data-theme") || "dark");
    document.querySelectorAll(".offer-header a[href='index.html'], .offer-back").forEach(function (link) {
      link.setAttribute("href", siteLocale() === "uk" ? "/uk/" : "/");
    });
    document.querySelectorAll(".offer-header .site-doc-header__nav a[href='docs.html']").forEach(function (link) {
      link.setAttribute("href", docsPageHref(config));
    });
  }

  initTheme();
  wireThemeToggle();
  wireInteractions();
  wireLangSwitch();

  if (!window.__siteHashWired) {
    window.__siteHashWired = true;
    window.addEventListener("hashchange", scrollFromHash);
  }

  window.addEventListener("popstate", function () {
    if (!window.history.state || !window.history.state.bidshardLocale) {
      return;
    }
    var locale = window.history.state.bidshardLocale;
    switchLocale(locale === "uk" ? "/uk/" : "/");
  });

  preloadLocaleConfigs().then(function (config) {
    window.__siteConfig = config;
    if (document.querySelector(".Landing")) {
      initLanding(config);
    } else if (document.body.classList.contains("docs-page")) {
      initDocs(config);
    } else {
      initOffer(config);
    }
    scrollFromHash();
  });
})();
