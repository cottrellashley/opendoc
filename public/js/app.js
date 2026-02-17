/**
 * Smooth iframe reload — shows a "Rebuilding..." overlay instead of a white flash.
 *
 * The overlay stays visible for at least MIN_DISPLAY_MS so the user perceives
 * a calm "rebuilding" state rather than a jarring flash.
 *
 * @param {HTMLIFrameElement} iframe
 * @param {HTMLElement} [overlay] - optional overlay element (must have .rebuild-overlay class)
 */
window.reloadFrame = function (iframe, overlay) {
  if (!iframe) return;

  var MIN_DISPLAY_MS = 600; // minimum time the overlay stays visible

  // Find the closest overlay if not provided
  if (!overlay) {
    var parent = iframe.parentElement;
    if (parent) overlay = parent.querySelector(".rebuild-overlay");
  }

  // Show overlay
  if (overlay) overlay.classList.add("active");
  var showTime = Date.now();

  function hideOverlay() {
    if (!overlay) return;
    var elapsed = Date.now() - showTime;
    var remaining = Math.max(0, MIN_DISPLAY_MS - elapsed);
    setTimeout(function () {
      overlay.classList.remove("active");
    }, remaining);
  }

  // Listen for load event to hide overlay (after min display)
  function onLoad() {
    iframe.removeEventListener("load", onLoad);
    hideOverlay();
  }
  iframe.addEventListener("load", onLoad);

  // Safety timeout in case load never fires
  setTimeout(function () {
    iframe.removeEventListener("load", onLoad);
    if (overlay) overlay.classList.remove("active");
  }, 8000);

  try { iframe.contentWindow.location.reload(); } catch (e) {
    iframe.src = iframe.src;
  }
};

/**
 * OpenDoc Workbench — Main Application
 *
 * Modes: Editor, User (default), Publish (placeholder), Settings
 * Navigation via hamburger menu sidebar.
 */
(function () {
  "use strict";

  // ── State ───────────────────────────────────────────────

  var state = {
    mode: "user",
    openFiles: [],
    activeFile: null,
    consoleCollapsed: false,
    menuOpen: false,
    settings: null,
  };

  // ── DOM refs ────────────────────────────────────────────

  var $menuBtn = document.getElementById("menu-btn");
  var $menuBtnEditor = document.getElementById("menu-btn-editor");
  var $menuSidebar = document.getElementById("menu-sidebar");
  var $menuBackdrop = document.getElementById("menu-backdrop");
  var $menuClose = document.getElementById("menu-close");
  var $menuBrand = $menuSidebar.querySelector(".menu-brand");
  var $fileTree = document.getElementById("file-tree");
  var $monacoContainer = document.getElementById("monaco-container");
  var $openTabs = document.getElementById("open-tabs");
  var $buildStatus = document.getElementById("build-status");
  var $fileStatus = document.getElementById("file-status");
  var $terminalContainer = document.getElementById("terminal-container");
  var $terminalStatus = document.getElementById("terminal-status");
  var $dialogOverlay = document.getElementById("dialog-overlay");
  var $newFileInput = document.getElementById("new-file-input");

  // ── Menu open/close ─────────────────────────────────────

  function openMenu() {
    state.menuOpen = true;
    $menuSidebar.classList.add("open");
    $menuBackdrop.classList.add("open");
  }

  function closeMenu() {
    state.menuOpen = false;
    $menuSidebar.classList.remove("open");
    $menuBackdrop.classList.remove("open");
  }

  $menuBtn.addEventListener("click", openMenu);
  $menuBtnEditor.addEventListener("click", openMenu);
  $menuClose.addEventListener("click", closeMenu);
  $menuBackdrop.addEventListener("click", closeMenu);

  // ── Mode switching ─────────────────────────────────────

  function switchMode(mode) {
    state.mode = mode;

    // Update menu items
    document.querySelectorAll(".menu-item[data-mode]").forEach(function (item) {
      item.classList.toggle("active", item.getAttribute("data-mode") === mode);
    });

    // Update views
    document.querySelectorAll(".view").forEach(function (v) { v.classList.remove("active"); });
    var target = document.getElementById("view-" + mode);
    if (target) target.classList.add("active");

    // Set body class for CSS to hide/show floating hamburger
    document.body.className = "mode-" + mode;

    // Refit editor/terminal when switching to editor
    if (mode === "editor") {
      if (editor && editor.layout) setTimeout(function () { editor.layout(); }, 50);
      if (fitAddon && !state.consoleCollapsed) setTimeout(function () { fitAddon.fit(); }, 50);
    }

    // Populate settings fields when entering settings mode
    if (mode === "settings") {
      loadSettingsIntoForm();
    }

    // Auto-build publish preview when entering publish mode
    if (mode === "publish") {
      triggerPublishBuild();
    }

    closeMenu();
  }

  document.querySelectorAll(".menu-item[data-mode]").forEach(function (item) {
    if (item.disabled) return;
    item.addEventListener("click", function () {
      switchMode(item.getAttribute("data-mode"));
    });
  });

  // ── Settings ────────────────────────────────────────────

  function loadSettingsFromServer(cb) {
    fetch("/api/settings").then(function (r) { return r.json(); })
      .then(function (s) { state.settings = s; applyBranding(s); if (cb) cb(s); })
      .catch(function () {});
  }

  function loadSettingsIntoForm() {
    if (!state.settings) {
      loadSettingsFromServer(function (s) { fillSettingsForm(s); });
    } else {
      fillSettingsForm(state.settings);
    }
    // Refresh integration and API key status
    loadIntegrationStatus();
    loadAPIKeyStatus();
  }

  function fillSettingsForm(s) {
    document.getElementById("setting-project-name").value = s.project_name || "";
    document.getElementById("setting-user-name").value = s.user_name || "";
    document.getElementById("setting-github-account").value = s.github_account || "";
    document.getElementById("setting-github-repo").value = s.github_repo || "";
    document.getElementById("settings-status").textContent = "";
  }

  function applyBranding(s) {
    var name = s.project_name || "OpenDoc";
    $menuBrand.textContent = name;
    document.title = name;
  }

  document.getElementById("btn-settings-save").addEventListener("click", function () {
    var data = {
      project_name: document.getElementById("setting-project-name").value.trim(),
      user_name: document.getElementById("setting-user-name").value.trim(),
      github_account: document.getElementById("setting-github-account").value.trim(),
      github_repo: document.getElementById("setting-github-repo").value.trim(),
    };

    var statusEl = document.getElementById("settings-status");
    statusEl.textContent = "Saving...";
    statusEl.style.color = "var(--warning)";

    fetch("/api/settings", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    })
      .then(function (r) { return r.json(); })
      .then(function (s) {
        state.settings = s;
        applyBranding(s);
        statusEl.textContent = "Saved";
        statusEl.style.color = "var(--success)";
        setTimeout(function () { statusEl.textContent = ""; }, 3000);
      })
      .catch(function () {
        statusEl.textContent = "Error saving";
        statusEl.style.color = "var(--danger)";
      });
  });

  // ── Integration status ────────────────────────────────────

  function loadIntegrationStatus() {
    fetch("/api/integrations/status").then(function (r) { return r.json(); })
      .then(function (data) { renderGHStatus(data.github); renderClaudeStatus(data.claude); })
      .catch(function () {
        renderGHStatus({ installed: false, authenticated: false });
        renderClaudeStatus({ installed: false, authenticated: false });
      });
  }

  function renderGHStatus(gh) {
    var badge = document.getElementById("gh-badge");
    var badgeLabel = document.getElementById("gh-badge-label");
    var statusText = document.getElementById("gh-status-text");
    var versionEl = document.getElementById("gh-version");
    var accountEl = document.getElementById("gh-account");
    var actionsEl = document.getElementById("gh-actions");
    actionsEl.innerHTML = "";

    if (!gh.installed) {
      badge.setAttribute("data-state", "disconnected");
      badgeLabel.textContent = "Not installed";
      statusText.textContent = "gh CLI is not installed";
      versionEl.textContent = "";
      accountEl.textContent = "";
      var link = document.createElement("a");
      link.href = "https://cli.github.com/";
      link.target = "_blank";
      link.className = "integration-btn primary";
      link.textContent = "Install GitHub CLI";
      actionsEl.appendChild(link);
    } else if (!gh.authenticated) {
      badge.setAttribute("data-state", "partial");
      badgeLabel.textContent = "Not connected";
      statusText.textContent = "Installed but not authenticated";
      versionEl.textContent = gh.version || "";
      accountEl.textContent = "";
      var btn = document.createElement("button");
      btn.className = "integration-btn primary";
      btn.textContent = "Connect GitHub";
      btn.addEventListener("click", startGHLogin);
      actionsEl.appendChild(btn);
    } else {
      badge.setAttribute("data-state", "connected");
      badgeLabel.textContent = "Connected";
      statusText.textContent = gh.account ? "Logged in as " + gh.account : "Authenticated";
      versionEl.textContent = gh.version || "";
      accountEl.textContent = gh.account ? "Account: " + gh.account : "";
      // Auto-fill GitHub account field if empty
      if (gh.account) {
        var field = document.getElementById("setting-github-account");
        if (!field.value) field.value = gh.account;
      }
      var btn = document.createElement("button");
      btn.className = "integration-btn";
      btn.textContent = "Reconnect";
      btn.addEventListener("click", startGHLogin);
      actionsEl.appendChild(btn);
    }
  }

  function renderClaudeStatus(claude) {
    var badge = document.getElementById("claude-badge");
    var badgeLabel = document.getElementById("claude-badge-label");
    var statusText = document.getElementById("claude-status-text");
    var versionEl = document.getElementById("claude-version");
    var actionsEl = document.getElementById("claude-actions");
    actionsEl.innerHTML = "";

    if (!claude.installed) {
      badge.setAttribute("data-state", "disconnected");
      badgeLabel.textContent = "Not installed";
      statusText.textContent = "Claude CLI is not installed";
      versionEl.textContent = "";
      var link = document.createElement("a");
      link.href = "https://docs.anthropic.com/en/docs/claude-code";
      link.target = "_blank";
      link.className = "integration-btn primary";
      link.textContent = "Install Claude CLI";
      actionsEl.appendChild(link);
    } else {
      badge.setAttribute("data-state", "connected");
      badgeLabel.textContent = "Available";
      statusText.textContent = "Ready to use in the terminal";
      versionEl.textContent = claude.version || "";
      var hint = document.createElement("span");
      hint.className = "integration-detail";
      hint.textContent = "Use 'claude' in the editor console to interact.";
      actionsEl.appendChild(hint);
    }
  }

  // ── gh auth login flow ────────────────────────────────────

  // ── API Keys ──────────────────────────────────────────────

  function loadAPIKeyStatus() {
    fetch("/api/integrations/api-keys").then(function (r) { return r.json(); })
      .then(function (data) {
        renderKeyStatus("anthropic", data.anthropic);
        renderKeyStatus("openai", data.openai);
      })
      .catch(function () {});
  }

  function renderKeyStatus(provider, info) {
    var badge = document.getElementById(provider + "-key-badge");
    var source = document.getElementById(provider + "-key-source");
    var input = document.getElementById("input-" + provider + "-key");

    if (info.configured) {
      badge.textContent = "Configured";
      badge.className = "api-key-badge configured";
      input.placeholder = info.masked || "••••••••";
      if (info.source === "environment") {
        source.textContent = "Set via environment variable";
      } else if (info.source === "settings") {
        source.textContent = "Saved in settings";
      }
    } else {
      badge.textContent = "Not set";
      badge.className = "api-key-badge not-configured";
      source.textContent = "";
    }
  }

  function saveAPIKey(provider) {
    var input = document.getElementById("input-" + provider + "-key");
    var key = input.value.trim();
    if (!key) return;

    var body = {};
    body[provider + "_key"] = key;

    fetch("/api/integrations/api-keys", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    })
      .then(function (r) { return r.json(); })
      .then(function (data) {
        if (data.success) {
          input.value = "";
          renderKeyStatus("anthropic", data.anthropic);
          renderKeyStatus("openai", data.openai);
        }
      })
      .catch(function () {});
  }

  function removeAPIKey(provider) {
    var body = {};
    body[provider + "_key"] = "__REMOVE__";

    fetch("/api/integrations/api-keys", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    })
      .then(function (r) { return r.json(); })
      .then(function (data) {
        if (data.success) {
          renderKeyStatus("anthropic", data.anthropic);
          renderKeyStatus("openai", data.openai);
        }
      })
      .catch(function () {});
  }

  // Wire up save/remove/toggle buttons
  document.getElementById("save-anthropic-key").addEventListener("click", function () { saveAPIKey("anthropic"); });
  document.getElementById("save-openai-key").addEventListener("click", function () { saveAPIKey("openai"); });
  document.getElementById("remove-anthropic-key").addEventListener("click", function () { removeAPIKey("anthropic"); });
  document.getElementById("remove-openai-key").addEventListener("click", function () { removeAPIKey("openai"); });

  document.getElementById("toggle-anthropic-key").addEventListener("click", function () {
    var input = document.getElementById("input-anthropic-key");
    input.type = input.type === "password" ? "text" : "password";
  });
  document.getElementById("toggle-openai-key").addEventListener("click", function () {
    var input = document.getElementById("input-openai-key");
    input.type = input.type === "password" ? "text" : "password";
  });

  function startGHLogin() {
    var statusText = document.getElementById("gh-status-text");
    var actionsEl = document.getElementById("gh-actions");
    var badge = document.getElementById("gh-badge");
    var badgeLabel = document.getElementById("gh-badge-label");

    statusText.textContent = "Authenticating... Check your browser for the GitHub login page.";
    badge.setAttribute("data-state", "partial");
    badgeLabel.textContent = "Connecting...";
    actionsEl.innerHTML = '<span class="integration-detail">Waiting for browser authentication...</span>';

    var source = new EventSource("/api/integrations/gh-login");

    source.addEventListener("output", function (e) {
      var data = JSON.parse(e.data);
      if (data.message) {
        statusText.textContent = data.message;
        // Check for the "one-time code" line — show it prominently
        if (data.message.indexOf("one-time code") !== -1 || data.message.indexOf("code:") !== -1) {
          actionsEl.innerHTML = '<div class="integration-detail" style="font-size:13px;font-weight:600;color:var(--accent);">' + esc(data.message) + '</div>';
        }
      }
    });

    source.addEventListener("complete", function (e) {
      source.close();
      var data = JSON.parse(e.data);
      if (data.success && data.status) {
        renderGHStatus(data.status);
      }
      // Refresh full status
      setTimeout(loadIntegrationStatus, 500);
    });

    source.addEventListener("error", function (e) {
      if (e.data) {
        var data = JSON.parse(e.data);
        statusText.textContent = data.message || "Login failed";
      }
      source.close();
      setTimeout(loadIntegrationStatus, 1000);
    });

    source.onerror = function () {
      source.close();
      setTimeout(loadIntegrationStatus, 1000);
    };
  }

  // ── Theme toggle + sync ─────────────────────────────────

  function setTheme(theme) {
    var html = document.documentElement;
    html.setAttribute("data-theme", theme);
    localStorage.setItem("opendoc-workbench-theme", theme);
    localStorage.setItem("opendoc-theme", theme);
    if (editor) monaco.editor.setTheme(theme === "light" ? "vs" : "vs-dark");

    // Sync to all iframes (user + publish)
    document.querySelectorAll(".user-site-frame, .publish-frame").forEach(function (frame) {
      try {
        frame.contentWindow.postMessage({ type: "opendoc-theme-sync", theme: theme }, "*");
      } catch (e) {}
    });
  }

  document.getElementById("theme-toggle").addEventListener("click", function () {
    var current = document.documentElement.getAttribute("data-theme") || "dark";
    setTheme(current === "dark" ? "light" : "dark");
  });

  // Listen for theme changes from iframe (when user clicks toggle inside the OpenDoc site)
  window.addEventListener("message", function (e) {
    if (e.data && e.data.type === "opendoc-theme-sync") {
      var theme = e.data.theme;
      document.documentElement.setAttribute("data-theme", theme);
      localStorage.setItem("opendoc-workbench-theme", theme);
      localStorage.setItem("opendoc-theme", theme);
      if (editor) monaco.editor.setTheme(theme === "light" ? "vs" : "vs-dark");
    }
  });

  // Sync theme to iframe on load
  function syncThemeToIframe(frame) {
    var theme = document.documentElement.getAttribute("data-theme") || "dark";
    // Wait for iframe to load, then send theme
    frame.addEventListener("load", function () {
      try {
        frame.contentWindow.postMessage({ type: "opendoc-theme-sync", theme: theme }, "*");
      } catch (e) {}
    });
  }

  // ── Monaco Editor ───────────────────────────────────────

  var editor = null;
  var editorModels = {};

  function initEditor() {
    showWelcome();

    function create() {
      var theme = document.documentElement.getAttribute("data-theme");
      editor = monaco.editor.create($monacoContainer, {
        value: "", language: "markdown",
        theme: theme === "light" ? "vs" : "vs-dark",
        fontSize: 13,
        fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
        minimap: { enabled: false },
        lineNumbers: "on", renderWhitespace: "selection",
        wordWrap: "on", tabSize: 2,
        scrollBeyondLastLine: false,
        padding: { top: 8 },
        automaticLayout: true,
      });

      editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, function () {
        saveActiveFile();
      });
    }

    if (window.monacoReady) create();
    else document.addEventListener("monaco-ready", create);
  }

  function showWelcome() {
    if ($monacoContainer.querySelector("#editor-welcome")) return;
    $monacoContainer.innerHTML = "";
    var w = document.createElement("div");
    w.id = "editor-welcome";
    w.innerHTML =
      "<h2>Editor</h2>" +
      "<p>Select a file from the sidebar to edit.</p>" +
      "<p><kbd>Cmd+S</kbd> save &middot; <kbd>Cmd+B</kbd> build</p>";
    $monacoContainer.appendChild(w);
  }

  function getLanguage(path) {
    if (!path) return "plaintext";
    if (path.endsWith(".md")) return "markdown";
    if (path.endsWith(".yml") || path.endsWith(".yaml")) return "yaml";
    return "plaintext";
  }

  function openFileInEditor(filePath, content) {
    if (!editor) return;
    var welcome = $monacoContainer.querySelector("#editor-welcome");
    if (welcome) welcome.remove();

    var lang = getLanguage(filePath);
    var uri = monaco.Uri.parse("file:///" + filePath);

    if (!editorModels[filePath]) {
      editorModels[filePath] = monaco.editor.createModel(content, lang, uri);
      editorModels[filePath].onDidChangeContent(function () {
        var f = state.openFiles.find(function (f) { return f.path === filePath; });
        if (f && f.content !== editorModels[filePath].getValue()) {
          f.modified = true;
          renderOpenTabs();
        }
      });
    }

    editor.setModel(editorModels[filePath]);
    state.activeFile = filePath;
    renderOpenTabs();
    highlightActiveTreeItem(filePath);
    $fileStatus.textContent = filePath;
  }

  function saveActiveFile() {
    if (!state.activeFile || !editor) return;
    var f = state.openFiles.find(function (f) { return f.path === state.activeFile; });
    if (!f) return;

    var content = editor.getValue();
    f.content = content; f.modified = false;

    fetch("/api/files/" + f.path, { method: "PUT", headers: { "Content-Type": "text/plain" }, body: content })
      .then(function (r) { if (!r.ok) throw new Error("Save failed"); $fileStatus.textContent = f.path + " — saved"; renderOpenTabs(); })
      .catch(function (e) { $fileStatus.textContent = "Error: " + e.message; });
  }

  // ── Open tabs ───────────────────────────────────────────

  function addOpenFile(filePath, content) {
    var existing = state.openFiles.find(function (f) { return f.path === filePath; });
    if (existing) { openFileInEditor(filePath, existing.content); return; }
    state.openFiles.push({ path: filePath, content: content, modified: false });
    openFileInEditor(filePath, content);
  }

  function closeFile(filePath) {
    var idx = state.openFiles.findIndex(function (f) { return f.path === filePath; });
    if (idx === -1) return;
    if (editorModels[filePath]) { editorModels[filePath].dispose(); delete editorModels[filePath]; }
    state.openFiles.splice(idx, 1);
    if (state.activeFile === filePath) {
      if (state.openFiles.length > 0) {
        var next = state.openFiles[Math.min(idx, state.openFiles.length - 1)];
        openFileInEditor(next.path, next.content);
      } else { state.activeFile = null; showWelcome(); $fileStatus.textContent = ""; }
    }
    renderOpenTabs();
  }

  function renderOpenTabs() {
    $openTabs.innerHTML = "";
    state.openFiles.forEach(function (file) {
      var tab = document.createElement("button");
      tab.className = "open-tab" + (file.path === state.activeFile ? " active" : "");
      var label = document.createElement("span");
      label.textContent = file.path.split("/").pop();
      tab.appendChild(label);
      if (file.modified) { var dot = document.createElement("span"); dot.className = "tab-modified"; tab.appendChild(dot); }
      var close = document.createElement("span");
      close.className = "tab-close"; close.textContent = "\u00d7";
      close.addEventListener("click", function (e) { e.stopPropagation(); closeFile(file.path); });
      tab.appendChild(close);
      tab.addEventListener("click", function () { openFileInEditor(file.path, file.content); });
      $openTabs.appendChild(tab);
    });
  }

  // ── File tree (.md, .yml only) ──────────────────────────

  var ALLOWED = [".md", ".yml", ".yaml"];
  function isAllowed(name) { for (var i = 0; i < ALLOWED.length; i++) { if (name.endsWith(ALLOWED[i])) return true; } return false; }

  function filterTree(items) {
    var out = [];
    for (var i = 0; i < items.length; i++) {
      var item = items[i];
      if (item.type === "directory") {
        var ch = filterTree(item.children || []);
        if (ch.length > 0) out.push({ type: "directory", name: item.name, path: item.path, children: ch });
      } else if (isAllowed(item.name)) out.push(item);
    }
    return out;
  }

  function loadFileTree() {
    fetch("/api/files").then(function (r) { return r.json(); })
      .then(function (tree) { $fileTree.innerHTML = ""; renderTree(filterTree(tree), $fileTree, 0); })
      .catch(function () { $fileTree.innerHTML = '<div class="tree-item" style="color:var(--danger)">Error</div>'; });
  }

  function renderTree(items, container, depth) {
    items.forEach(function (item) {
      if (item.type === "directory") {
        var dirEl = document.createElement("div");
        var dirItem = document.createElement("div");
        dirItem.className = "tree-item dir";
        dirItem.style.paddingLeft = (8 + depth * 14) + "px";
        dirItem.innerHTML = '<span class="tree-icon">\u25BE</span><span class="tree-label">' + esc(item.name) + "</span>";
        var children = document.createElement("div"); children.className = "tree-children";
        renderTree(item.children, children, depth + 1);
        dirItem.addEventListener("click", function () {
          children.classList.toggle("collapsed");
          dirItem.querySelector(".tree-icon").textContent = children.classList.contains("collapsed") ? "\u25B8" : "\u25BE";
        });
        dirEl.appendChild(dirItem); dirEl.appendChild(children); container.appendChild(dirEl);
      } else {
        var fileItem = document.createElement("div");
        fileItem.className = "tree-item file";
        fileItem.style.paddingLeft = (8 + depth * 14) + "px";
        fileItem.setAttribute("data-path", item.path);
        var icon = item.name.endsWith(".md") ? "\uD83D\uDCC4" : "\u2699";
        fileItem.innerHTML = '<span class="tree-icon">' + icon + '</span><span class="tree-label">' + esc(item.name) + "</span>";
        fileItem.addEventListener("click", function () {
          fetch("/api/files/" + item.path).then(function (r) { return r.json(); })
            .then(function (d) { addOpenFile(item.path, d.content); });
        });
        container.appendChild(fileItem);
      }
    });
  }

  function highlightActiveTreeItem(filePath) {
    $fileTree.querySelectorAll(".tree-item.file").forEach(function (el) {
      el.classList.toggle("active", el.getAttribute("data-path") === filePath);
    });
  }

  function esc(str) { var d = document.createElement("div"); d.textContent = str; return d.innerHTML; }

  // ── Terminal / Console ──────────────────────────────────

  var term = null, fitAddon = null, ws = null, reconnectTimer = null, reconnectDelay = 1000;

  function initTerminal() {
    if (!$terminalContainer) return;
    term = new Terminal({
      cursorBlink: true, cursorStyle: "bar",
      fontSize: 12, fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      lineHeight: 1.2,
      theme: { background: "#010409", foreground: "#e6edf3", cursor: "#79a5f2",
        black: "#0d1117", red: "#f85149", green: "#3fb950", yellow: "#d29922",
        blue: "#79a5f2", magenta: "#bc8cff", cyan: "#76e3ea", white: "#e6edf3" },
      scrollback: 5000,
    });
    fitAddon = new FitAddon.FitAddon(); term.loadAddon(fitAddon);
    if (typeof WebLinksAddon !== "undefined") term.loadAddon(new WebLinksAddon.WebLinksAddon());
    term.open($terminalContainer); fitAddon.fit();
    term.onData(function (data) { if (ws && ws.readyState === WebSocket.OPEN) ws.send(new TextEncoder().encode(data)); });
    new ResizeObserver(function () { if (fitAddon) setTimeout(function () { fitAddon.fit(); sendResize(); }, 80); }).observe($terminalContainer);
    connectTerminal();
  }

  function connectTerminal() {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;
    setTermStatus("connecting");
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    try { ws = new WebSocket(proto + "//" + location.host + "/console/ws?mode=shell"); ws.binaryType = "arraybuffer"; }
    catch (e) { setTermStatus("disconnected"); scheduleReconnect(); return; }
    ws.onopen = function () { setTermStatus("connected"); reconnectDelay = 1000; sendResize(); };
    ws.onmessage = function (e) { term.write(e.data instanceof ArrayBuffer ? new Uint8Array(e.data) : e.data); };
    ws.onclose = function () { setTermStatus("disconnected"); scheduleReconnect(); };
    ws.onerror = function () { setTermStatus("disconnected"); };
  }

  function scheduleReconnect() {
    if (reconnectTimer) return;
    reconnectTimer = setTimeout(function () { reconnectTimer = null; connectTerminal(); reconnectDelay = Math.min(reconnectDelay * 2, 16000); }, reconnectDelay);
  }

  function sendResize() {
    if (!ws || ws.readyState !== WebSocket.OPEN || !term) return;
    var p = JSON.stringify({ cols: term.cols, rows: term.rows });
    var buf = new Uint8Array(1 + p.length); buf[0] = 0x01;
    for (var i = 0; i < p.length; i++) buf[i + 1] = p.charCodeAt(i);
    ws.send(buf.buffer);
  }

  function setTermStatus(s) { if ($terminalStatus) $terminalStatus.setAttribute("data-status", s); }

  var consoleToggle = document.getElementById("console-toggle");
  if (consoleToggle) {
    consoleToggle.addEventListener("click", function () {
      var area = document.getElementById("console-area");
      state.consoleCollapsed = !state.consoleCollapsed;
      area.classList.toggle("collapsed", state.consoleCollapsed);
      if (!state.consoleCollapsed && fitAddon) setTimeout(function () { fitAddon.fit(); }, 50);
    });
  }

  var termReconnect = document.getElementById("terminal-reconnect");
  if (termReconnect) {
    termReconnect.addEventListener("click", function () {
      clearTimeout(reconnectTimer); reconnectTimer = null;
      if (ws) { ws.onclose = null; ws.close(); ws = null; }
      if (term) term.clear(); reconnectDelay = 1000; connectTerminal();
    });
  }

  // ── SSE ─────────────────────────────────────────────────

  function connectSSE() {
    var source = new EventSource("/api/events");
    source.addEventListener("build-start", function () {
      $buildStatus.textContent = "Building..."; $buildStatus.className = "building";
    });
    source.addEventListener("build-complete", function (e) {
      var data = JSON.parse(e.data);
      if (data.success) {
        $buildStatus.textContent = "Built"; $buildStatus.className = "success";
        document.querySelectorAll(".user-site-frame").forEach(function (f) { window.reloadFrame(f); });
        setTimeout(function () { $buildStatus.textContent = ""; $buildStatus.className = ""; }, 3000);
      } else { $buildStatus.textContent = "Failed"; $buildStatus.className = "error"; }
    });
    source.addEventListener("file-changed", function () { loadFileTree(); });
    source.addEventListener("file-deleted", function () { loadFileTree(); });
    source.addEventListener("settings-changed", function (e) {
      var s = JSON.parse(e.data);
      state.settings = s;
      applyBranding(s);
    });
    source.addEventListener("publish-build-start", function () {
      if ($publishStatus) { $publishStatus.textContent = "Building..."; $publishStatus.className = "building"; }
    });
    source.addEventListener("publish-build-complete", function (e) {
      var data = JSON.parse(e.data);
      if ($publishStatus) {
        if (data.success) {
          $publishStatus.textContent = "Ready"; $publishStatus.className = "success";
          window.reloadFrame($publishFrame, document.getElementById("publish-rebuild-overlay"));
          setTimeout(function () { $publishStatus.textContent = ""; $publishStatus.className = ""; }, 3000);
        } else {
          $publishStatus.textContent = "Failed"; $publishStatus.className = "error";
        }
      }
    });
  }

  // ── Resizers ────────────────────────────────────────────

  function initResizer() {
    // Sidebar horizontal resizer
    var sidebarResizer = document.getElementById("sidebar-resizer");
    if (sidebarResizer) {
      sidebarResizer.addEventListener("mousedown", function (e) {
        e.preventDefault(); var startX = e.clientX;
        sidebarResizer.classList.add("active");
        document.body.style.cursor = "col-resize"; document.body.style.userSelect = "none";
        function onMove(e) {
          var sidebar = document.getElementById("sidebar");
          sidebar.style.width = Math.max(160, Math.min(400, sidebar.offsetWidth + (e.clientX - startX))) + "px";
          startX = e.clientX; if (editor) editor.layout();
        }
        function onUp() {
          sidebarResizer.classList.remove("active"); document.body.style.cursor = ""; document.body.style.userSelect = "";
          document.removeEventListener("mousemove", onMove); document.removeEventListener("mouseup", onUp);
        }
        document.addEventListener("mousemove", onMove); document.addEventListener("mouseup", onUp);
      });
    }

    // Console vertical resizer (drag to resize console height)
    var consoleResizer = document.getElementById("console-resizer");
    if (consoleResizer) {
      consoleResizer.addEventListener("mousedown", function (e) {
        e.preventDefault(); var startY = e.clientY;
        consoleResizer.classList.add("active");
        document.body.style.cursor = "row-resize"; document.body.style.userSelect = "none";
        var consoleArea = document.getElementById("console-area");
        function onMove(e) {
          var deltaY = startY - e.clientY;
          var newHeight = Math.max(60, Math.min(500, consoleArea.offsetHeight + deltaY));
          consoleArea.style.height = newHeight + "px";
          startY = e.clientY;
          if (editor) editor.layout();
          if (fitAddon) setTimeout(function () { fitAddon.fit(); }, 20);
        }
        function onUp() {
          consoleResizer.classList.remove("active"); document.body.style.cursor = ""; document.body.style.userSelect = "";
          document.removeEventListener("mousemove", onMove); document.removeEventListener("mouseup", onUp);
          if (fitAddon) fitAddon.fit();
        }
        document.addEventListener("mousemove", onMove); document.addEventListener("mouseup", onUp);
      });
    }
  }

  // ── Build button ────────────────────────────────────────

  document.getElementById("btn-build").addEventListener("click", function () {
    fetch("/api/opendoc/build", { method: "POST" })
      .then(function (r) { return r.json(); })
      .then(function (d) { if (!d.success && d.error) { $buildStatus.textContent = d.error; $buildStatus.className = "error"; } })
      .catch(function () { $buildStatus.textContent = "Failed"; $buildStatus.className = "error"; });
  });

  // ── Publish build ──────────────────────────────────────

  var $publishStatus = document.getElementById("publish-status");
  var $publishFrame = document.getElementById("publish-frame");

  function triggerPublishBuild() {
    $publishStatus.textContent = "Building...";
    $publishStatus.className = "building";

    fetch("/api/opendoc/publish-build", { method: "POST" })
      .then(function (r) { return r.json(); })
      .then(function (d) {
        if (d.success) {
          $publishStatus.textContent = "Ready";
          $publishStatus.className = "success";
          // Reload the publish iframe
          window.reloadFrame($publishFrame, document.getElementById("publish-rebuild-overlay"));
          setTimeout(function () { $publishStatus.textContent = ""; $publishStatus.className = ""; }, 3000);
        } else {
          $publishStatus.textContent = d.error || "Failed";
          $publishStatus.className = "error";
        }
      })
      .catch(function () {
        $publishStatus.textContent = "Build failed";
        $publishStatus.className = "error";
      });
  }

  document.getElementById("btn-publish-build").addEventListener("click", triggerPublishBuild);

  // ── Publish deploy modal ──────────────────────────────────

  var $deployOverlay = document.getElementById("deploy-overlay");
  var $deployRepo = document.getElementById("deploy-repo");
  var $deployLog = document.getElementById("deploy-log");
  var $deployLogArea = document.getElementById("deploy-log-area");
  var $deployResult = document.getElementById("deploy-result");
  var $deployStatus = document.getElementById("deploy-status");
  var $btnDeployStart = document.getElementById("btn-deploy-start");

  document.getElementById("btn-publish-deploy").addEventListener("click", openDeployModal);
  document.getElementById("deploy-close").addEventListener("click", closeDeployModal);
  $deployOverlay.addEventListener("click", function (e) {
    if (e.target === $deployOverlay) closeDeployModal();
  });

  function openDeployModal() {
    $deployOverlay.classList.remove("hidden");
    $deployLogArea.classList.add("hidden");
    $deployResult.classList.add("hidden");
    $deployLog.textContent = "";
    $deployStatus.textContent = "";
    $btnDeployStart.disabled = true;
    $btnDeployStart.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 2L11 13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg> Deploy Now';

    // Pre-fill repo from settings
    if (state.settings && state.settings.github_repo) {
      $deployRepo.value = state.settings.github_repo;
    }

    // Run preflight checks
    runDeployChecks();
  }

  function closeDeployModal() {
    $deployOverlay.classList.add("hidden");
  }

  function runDeployChecks() {
    var checkGH = document.getElementById("deploy-check-gh");
    var checkAuth = document.getElementById("deploy-check-auth");
    var checkRepo = document.getElementById("deploy-check-repo");

    // Reset
    [checkGH, checkAuth, checkRepo].forEach(function (el) {
      el.className = "deploy-check";
      el.querySelector(".deploy-check-icon").textContent = "⏳";
    });

    fetch("/api/integrations/status").then(function (r) { return r.json(); })
      .then(function (data) {
        var allGood = true;

        // gh installed
        if (data.github.installed) {
          checkGH.classList.add("pass");
          checkGH.querySelector(".deploy-check-icon").textContent = "✓";
        } else {
          checkGH.classList.add("fail");
          checkGH.querySelector(".deploy-check-icon").textContent = "✗";
          allGood = false;
        }

        // gh authenticated
        if (data.github.authenticated) {
          checkAuth.classList.add("pass");
          checkAuth.querySelector(".deploy-check-icon").textContent = "✓";
        } else {
          checkAuth.classList.add("fail");
          checkAuth.querySelector(".deploy-check-icon").textContent = "✗";
          allGood = false;
        }

        // repo configured
        var repo = $deployRepo.value.trim();
        if (repo && repo.indexOf("/") !== -1) {
          checkRepo.classList.add("pass");
          checkRepo.querySelector(".deploy-check-icon").textContent = "✓";
        } else {
          checkRepo.classList.add("fail");
          checkRepo.querySelector(".deploy-check-icon").textContent = "✗";
          allGood = false;
        }

        $btnDeployStart.disabled = !allGood;
      })
      .catch(function () {
        [checkGH, checkAuth, checkRepo].forEach(function (el) {
          el.classList.add("fail");
          el.querySelector(".deploy-check-icon").textContent = "✗";
        });
      });
  }

  // Re-check when repo input changes
  $deployRepo.addEventListener("input", function () {
    var checkRepo = document.getElementById("deploy-check-repo");
    var val = $deployRepo.value.trim();
    if (val && val.indexOf("/") !== -1) {
      checkRepo.className = "deploy-check pass";
      checkRepo.querySelector(".deploy-check-icon").textContent = "✓";
      // Re-enable button if other checks passed
      var ghOk = document.getElementById("deploy-check-gh").classList.contains("pass");
      var authOk = document.getElementById("deploy-check-auth").classList.contains("pass");
      $btnDeployStart.disabled = !(ghOk && authOk);
    } else {
      checkRepo.className = "deploy-check fail";
      checkRepo.querySelector(".deploy-check-icon").textContent = "✗";
      $btnDeployStart.disabled = true;
    }
  });

  $btnDeployStart.addEventListener("click", function () {
    var repo = $deployRepo.value.trim();
    if (!repo) return;

    // Show spinner
    $btnDeployStart.disabled = true;
    $btnDeployStart.innerHTML = '<span class="deploy-spinner"></span> Deploying...';
    $deployStatus.textContent = "Building and deploying...";
    $deployLogArea.classList.remove("hidden");
    $deployResult.classList.add("hidden");
    $deployLog.textContent = "Starting publish build...\n";

    fetch("/api/integrations/publish-deploy", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ repo: repo }),
    })
      .then(function (r) { return r.json(); })
      .then(function (d) {
        if (d.log) $deployLog.textContent += d.log + "\n";

        if (d.success) {
          $deployStatus.textContent = "";
          $deployResult.classList.remove("hidden");
          document.getElementById("deploy-result-icon").textContent = "🎉";
          document.getElementById("deploy-result-text").textContent = "Successfully deployed to " + d.repo;
          var link = document.getElementById("deploy-result-link");
          if (d.url) { link.textContent = d.url; link.href = d.url; } else { link.textContent = ""; }
          $btnDeployStart.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 2L11 13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg> Deploy Again';
          $btnDeployStart.disabled = false;
        } else {
          $deployStatus.textContent = d.error || "Deploy failed";
          $deployStatus.style.color = "var(--danger)";
          $deployLog.textContent += "\nError: " + (d.error || "Unknown error") + "\n";
          $btnDeployStart.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 2L11 13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg> Retry';
          $btnDeployStart.disabled = false;
        }
      })
      .catch(function (e) {
        $deployStatus.textContent = "Network error: " + e.message;
        $deployStatus.style.color = "var(--danger)";
        $btnDeployStart.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 2L11 13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg> Retry';
        $btnDeployStart.disabled = false;
      });
  });

  // ── New file dialog ─────────────────────────────────────

  function openNewFileDialog() {
    $dialogOverlay.classList.remove("hidden");
    $newFileInput.value = "content/"; $newFileInput.focus();
    $newFileInput.setSelectionRange(8, 8);
  }

  document.getElementById("btn-new-file").addEventListener("click", openNewFileDialog);

  document.getElementById("dialog-cancel").addEventListener("click", function () { $dialogOverlay.classList.add("hidden"); });
  document.getElementById("dialog-create").addEventListener("click", createNewFile);
  $newFileInput.addEventListener("keydown", function (e) {
    if (e.key === "Enter") createNewFile();
    if (e.key === "Escape") $dialogOverlay.classList.add("hidden");
  });

  function createNewFile() {
    var fp = $newFileInput.value.trim(); if (!fp) return;
    var content = "";
    if (fp.endsWith(".md")) {
      var title = fp.split("/").pop().replace(".md", "").replace(/-/g, " ");
      title = title.charAt(0).toUpperCase() + title.slice(1);
      content = "---\ntitle: \"" + title + "\"\n---\n\n# " + title + "\n\n";
    }
    fetch("/api/files/" + fp, { method: "POST", headers: { "Content-Type": "text/plain" }, body: content })
      .then(function (r) { if (!r.ok) throw new Error("Failed"); return r.json(); })
      .then(function () { $dialogOverlay.classList.add("hidden"); loadFileTree(); switchMode("editor"); addOpenFile(fp, content); })
      .catch(function (e) { alert("Error: " + e.message); });
  }

  // ── Keyboard shortcuts ──────────────────────────────────

  document.addEventListener("keydown", function (e) {
    if ((e.metaKey || e.ctrlKey) && e.key === "b") { e.preventDefault(); document.getElementById("btn-build").click(); }
    if ((e.metaKey || e.ctrlKey) && e.key === "1") { e.preventDefault(); switchMode("editor"); }
    if ((e.metaKey || e.ctrlKey) && e.key === "2") { e.preventDefault(); switchMode("user"); }
    if ((e.metaKey || e.ctrlKey) && e.key === "3") { e.preventDefault(); switchMode("publish"); }
    if (e.key === "Escape") {
      if (state.menuOpen) closeMenu();
      $dialogOverlay.classList.add("hidden");
    }
  });

  // ── Init ────────────────────────────────────────────────

  function init() {
    loadFileTree();
    initEditor();
    initTerminal();
    initResizer();
    connectSSE();

    // Load settings and apply branding
    loadSettingsFromServer();

    // Set up theme sync for iframes
    document.querySelectorAll(".user-site-frame, .publish-frame").forEach(syncThemeToIframe);

    switchMode("user");
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
