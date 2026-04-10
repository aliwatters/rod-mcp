// stealth.js — Injected via Page.EvalOnNewDocument when stealth mode is enabled.
// Patches browser APIs that bot-detection scripts probe to identify automation.
// Must run before any page JavaScript executes.

(function () {
  'use strict';

  // 1. navigator.webdriver → undefined
  // Chromium sets this to true when --enable-automation is present or when
  // controlled by DevTools protocol. Delete the property so typeof returns
  // "undefined", matching a normal browser.
  Object.defineProperty(navigator, 'webdriver', {
    get: function () { return undefined; },
    configurable: true,
  });

  // 2. navigator.plugins — inject a realistic plugin array.
  // Headless/automated Chrome reports an empty PluginArray. Real Chrome on
  // desktop typically has at least the PDF plugins.
  var fakePlugins = [
    { name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
    { name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
    { name: 'Chromium PDF Viewer', filename: 'internal-pdf-viewer', description: '' },
  ];

  var pluginArray = Object.create(PluginArray.prototype);
  fakePlugins.forEach(function (p, i) {
    var plugin = Object.create(Plugin.prototype);
    Object.defineProperties(plugin, {
      name:        { get: function () { return p.name; } },
      filename:    { get: function () { return p.filename; } },
      description: { get: function () { return p.description; } },
      length:      { get: function () { return 0; } },
    });
    pluginArray[i] = plugin;
  });
  Object.defineProperty(pluginArray, 'length', { get: function () { return fakePlugins.length; } });
  pluginArray.item = function (i) { return this[i] || null; };
  pluginArray.namedItem = function (name) {
    for (var i = 0; i < this.length; i++) {
      if (this[i].name === name) return this[i];
    }
    return null;
  };
  pluginArray.refresh = function () {};

  Object.defineProperty(navigator, 'plugins', {
    get: function () { return pluginArray; },
    configurable: true,
  });

  // 3. navigator.languages — ensure a realistic value.
  Object.defineProperty(navigator, 'languages', {
    get: function () { return ['en-US', 'en']; },
    configurable: true,
  });

  // 4. window.chrome — inject a minimal Chrome runtime object.
  // Bot detectors check for window.chrome existence and its runtime property.
  if (!window.chrome) {
    Object.defineProperty(window, 'chrome', {
      value: {},
      writable: true,
      configurable: true,
    });
  }
  if (!window.chrome.runtime) {
    window.chrome.runtime = {
      connect: function () { return {}; },
      sendMessage: function () {},
    };
  }

  // 5. Remove cdc_ prefixed globals injected by ChromeDriver.
  // These leak automation presence even when other patches are applied.
  var cdcKeys = Object.getOwnPropertyNames(window).filter(function (k) {
    return k.startsWith('cdc_') || k.startsWith('$cdc_');
  });
  cdcKeys.forEach(function (k) {
    try { delete window[k]; } catch (e) { /* non-configurable — ignore */ }
  });

  // 6. Patch Permissions.query to report "prompt" for notifications.
  // Automated browsers sometimes report "denied" which bot detectors flag.
  var origQuery = window.Permissions && window.Permissions.prototype.query;
  if (origQuery) {
    window.Permissions.prototype.query = function (parameters) {
      if (parameters && parameters.name === 'notifications') {
        return Promise.resolve({ state: Notification.permission || 'prompt' });
      }
      return origQuery.call(this, parameters);
    };
  }
})();
