/**
 * OpenDoc Chat — Cursor-inspired chat UX with tool call cards.
 *
 * Chat panel resizes via drag. Snaps to 0%, 25%, 50%, 100%.
 * The "<" button toggles between 0% and 25%.
 *
 * Architecture:
 *   .chat-msg-assistant
 *     .chat-msg-avatar
 *     .chat-msg-body
 *       .chat-msg-role
 *       .chat-msg-content          (markdown — segment 0)
 *       .chat-tool-card            (tool card)
 *       .chat-msg-content          (markdown — segment 1)
 *       .chat-tool-card            (tool card)
 *       .chat-msg-content          (markdown — segment 2)
 *       ...
 *
 * Tool cards live OUTSIDE markdown content divs, so re-rendering
 * markdown never destroys them.
 */
(function () {
  "use strict";

  var sessionId = null;
  var isStreaming = false;

  // Snap points as percentages of container width
  var SNAPS = [0, 25, 50, 100];

  // Render debounce interval (ms)
  var RENDER_DEBOUNCE = 50;

  // Configure marked
  if (typeof marked !== "undefined") {
    marked.setOptions({ breaks: true, gfm: true });
  }

  // ── State ──────────────────────────────────────────────

  var comp, frame, divider, panel, messages, input, sendBtn, toggleBtn, hintEl;
  var chatPct = 0;

  // SVG icons (reused)
  var ICONS = {
    user: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>',
    bot: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 8V4H8"/><rect x="2" y="8" width="20" height="14" rx="2"/><path d="M6 18h.01"/><path d="M18 18h.01"/></svg>',
    eye: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>',
    pencil: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.85 2.85 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>',
    edit: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>',
    folder: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>',
    hammer: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 12-8.5 8.5c-.83.83-2.17.83-3 0 0 0 0 0 0 0a2.12 2.12 0 0 1 0-3L12 9"/><path d="M17.64 15 22 10.64"/><path d="m20.91 11.7-1.25-1.25c-.6-.6-.93-1.4-.93-2.25v-.86L16.01 4.6a5.56 5.56 0 0 0-3.94-1.64H9l.92.82A6.18 6.18 0 0 1 12 8.4v1.56l2 2h2.47l2.26 1.91"/></svg>',
    settings: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>',
    nav: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="18" x2="21" y2="18"/></svg>',
    check: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>',
    chevron: '<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>',
    x: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>',
    sparkle: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3l1.5 5.5L19 10l-5.5 1.5L12 17l-1.5-5.5L5 10l5.5-1.5L12 3z"/></svg>',
    file: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/></svg>',
    copy: '<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>',
    copied: '<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>',
  };

  // Tool name -> { icon, label, detailFn }
  var TOOL_META = {
    read_file:  { icon: "eye",     label: "Read file",       detailKey: "path" },
    write_file: { icon: "pencil",  label: "Write file",      detailKey: "path" },
    edit_file:  { icon: "edit",    label: "Edit file",       detailKey: "path" },
    list_files: { icon: "folder",  label: "List files",      detailKey: "path" },
    build:      { icon: "hammer",  label: "Build site",      detailKey: null },
    get_config: { icon: "settings",label: "Read config",     detailKey: null },
    update_nav: { icon: "nav",     label: "Update navigation", detailKey: null },
  };

  function init() {
    comp = document.querySelector(".user-mode-component");
    if (!comp) return;

    frame = comp.querySelector(".user-site-frame");
    divider = comp.querySelector(".chat-divider");
    panel = comp.querySelector(".chat-panel");
    messages = comp.querySelector(".chat-messages");
    input = comp.querySelector(".chat-input");
    sendBtn = comp.querySelector(".chat-send-btn");
    toggleBtn = comp.querySelector(".chat-toggle-btn");
    hintEl = comp.querySelector(".chat-input-hint");

    setChatPct(0);
    renderWelcome();
    initDividerDrag();

    toggleBtn.addEventListener("click", function () {
      setChatPct(chatPct === 0 ? 25 : 0, true);
    });

    sendBtn.addEventListener("click", function () { sendMessage(); });
    input.addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendMessage(); }
    });

    input.addEventListener("input", function () {
      this.style.height = "auto";
      this.style.height = Math.min(this.scrollHeight, 120) + "px";
    });
  }

  // ── Set chat panel width ───────────────────────────────

  function setChatPct(pct, animate) {
    chatPct = pct;
    var w = comp.offsetWidth;
    var chatW = Math.round(w * pct / 100);

    if (animate) {
      panel.style.transition = "width 0.25s ease";
      frame.style.transition = "width 0.25s ease";
      setTimeout(function () {
        panel.style.transition = "";
        frame.style.transition = "";
      }, 260);
    }

    panel.style.width = chatW + "px";
    comp.setAttribute("data-chat-pct", pct === 100 ? "100" : pct === 0 ? "0" : "");
    comp.classList.toggle("chat-open", pct > 0);
  }

  // ── Draggable divider ──────────────────────────────────

  function initDividerDrag() {
    var dragging = false;

    divider.addEventListener("mousedown", function (e) {
      e.preventDefault();
      dragging = true;
      divider.classList.add("active");
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
      frame.style.pointerEvents = "none";

      function onMove(e) {
        if (!dragging) return;
        var rect = comp.getBoundingClientRect();
        var x = e.clientX - rect.left;
        var w = rect.width;
        var rawPct = ((w - x) / w) * 100;
        rawPct = Math.max(0, Math.min(100, rawPct));
        panel.style.width = Math.round(w * rawPct / 100) + "px";
        chatPct = rawPct;
        comp.classList.toggle("chat-open", rawPct > 2);
      }

      function onUp() {
        dragging = false;
        divider.classList.remove("active");
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        frame.style.pointerEvents = "";
        document.removeEventListener("mousemove", onMove);
        document.removeEventListener("mouseup", onUp);

        var nearest = findNearestSnap(chatPct);
        setChatPct(nearest, true);
        if (nearest > 0) setTimeout(function () { input.focus(); }, 280);
      }

      document.addEventListener("mousemove", onMove);
      document.addEventListener("mouseup", onUp);
    });

    divider.addEventListener("dblclick", function () {
      var currentSnap = findNearestSnap(chatPct);
      var idx = SNAPS.indexOf(currentSnap);
      var next = SNAPS[(idx + 1) % SNAPS.length];
      setChatPct(next, true);
      if (next > 0) setTimeout(function () { input.focus(); }, 280);
    });
  }

  function findNearestSnap(pct) {
    var best = SNAPS[0], bestDist = Math.abs(pct - best);
    for (var i = 1; i < SNAPS.length; i++) {
      var d = Math.abs(pct - SNAPS[i]);
      if (d < bestDist) { best = SNAPS[i]; bestDist = d; }
    }
    return best;
  }

  // ── Send message ────────────────────────────────────────

  function sendMessage() {
    var text = (input.value || "").trim();
    if (!text || isStreaming) return;

    if (chatPct === 0) setChatPct(25, true);

    // Remove welcome
    var welcome = messages.querySelector(".chat-welcome");
    if (welcome) welcome.remove();

    appendUserMessage(text);
    input.value = "";
    input.style.height = "auto";
    isStreaming = true;
    sendBtn.disabled = true;
    setHint("AI is responding...", true);

    // Build assistant message shell
    var assistantEl = appendAssistantMessage();
    var bodyEl = assistantEl.querySelector(".chat-msg-body");

    // Create initial content segment + thinking indicator
    var thinkingEl = createThinking();
    bodyEl.appendChild(thinkingEl);

    var activeContent = createContentSegment();
    bodyEl.appendChild(activeContent);
    activeContent.style.display = "none"; // hidden until first text

    var fullText = "";
    var renderTimer = null;
    var toolArgsAccum = {}; // toolId -> accumulated args string
    var currentToolCard = null;

    fetch("/api/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ message: text, sessionId: sessionId, provider: "anthropic" }),
    })
      .then(function (response) {
        if (!response.ok) {
          return response.json().then(function (err) { throw new Error(err.error || "Chat failed"); });
        }

        var reader = response.body.getReader();
        var decoder = new TextDecoder();
        var buffer = "";

        function read() {
          return reader.read().then(function (result) {
            if (result.done) {
              flushRender();
              finishStreaming(assistantEl, activeContent, fullText);
              return;
            }

            buffer += decoder.decode(result.value, { stream: true });
            var lines = buffer.split("\n");
            buffer = lines.pop() || "";

            var eventType = "";
            for (var i = 0; i < lines.length; i++) {
              var line = lines[i];
              if (line.startsWith("event: ")) {
                eventType = line.slice(7);
              } else if (line.startsWith("data: ")) {
                var data;
                try { data = JSON.parse(line.slice(6)); } catch (e) { continue; }

                if (eventType === "session") {
                  sessionId = data.sessionId;

                } else if (eventType === "chunk") {
                  if (data.type === "text" && data.content) {
                    // Remove thinking indicator on first text
                    if (thinkingEl && thinkingEl.parentNode) {
                      thinkingEl.remove();
                      thinkingEl = null;
                    }
                    activeContent.style.display = "";
                    activeContent.classList.add("chat-msg-content-active");

                    fullText += data.content;
                    debouncedRender(activeContent, fullText);

                  } else if (data.type === "tool_call_start") {
                    // Remove thinking if still showing
                    if (thinkingEl && thinkingEl.parentNode) {
                      thinkingEl.remove();
                      thinkingEl = null;
                    }

                    // Flush current render
                    flushRender();
                    activeContent.classList.remove("chat-msg-content-active");

                    var tc = data.toolCall || {};
                    var fn = (tc.function || tc.Function || {});
                    var toolName = fn.name || fn.Name || "tool";
                    var toolId = tc.id || tc.ID || "";
                    var toolArgs = fn.arguments || fn.Arguments || "";

                    toolArgsAccum[toolId] = toolArgs;
                    currentToolCard = appendToolCard(bodyEl, toolName, toolArgs, toolId);
                    scrollBottom();

                  } else if (data.type === "tool_call_args") {
                    // Accumulate args for current tool
                    if (currentToolCard) {
                      var tcId = currentToolCard.getAttribute("data-tool-id");
                      if (tcId && toolArgsAccum[tcId] !== undefined) {
                        toolArgsAccum[tcId] += (data.content || "");
                        updateToolCardDetail(currentToolCard, toolArgsAccum[tcId]);
                      }
                    }

                  } else if (data.type === "tool_call_end") {
                    // Mark current tool as done
                    if (currentToolCard) {
                      var tcId2 = currentToolCard.getAttribute("data-tool-id");
                      var finalArgs = "";
                      if (tcId2 && toolArgsAccum[tcId2]) {
                        finalArgs = toolArgsAccum[tcId2];
                      }
                      // Also get args from the end event
                      if (data.toolCall) {
                        var endFn = (data.toolCall.function || data.toolCall.Function || {});
                        var endArgs = endFn.arguments || endFn.Arguments || "";
                        if (endArgs) finalArgs = endArgs;
                      }
                      markToolCardDone(currentToolCard, finalArgs);
                      currentToolCard = null;
                    }

                    // Create new content segment for text after tool
                    activeContent = createContentSegment();
                    bodyEl.appendChild(activeContent);
                    fullText = ""; // reset for next segment
                    scrollBottom();
                  }

                } else if (eventType === "error") {
                  if (thinkingEl && thinkingEl.parentNode) thinkingEl.remove();
                  activeContent.style.display = "";
                  activeContent.innerHTML = "";
                  activeContent.textContent = "Error: " + (data.error || "Unknown");
                  activeContent.classList.add("chat-error");
                }
                eventType = "";
              }
            }
            return read();
          });
        }

        return read();
      })
      .catch(function (err) {
        if (thinkingEl && thinkingEl.parentNode) thinkingEl.remove();
        activeContent.style.display = "";
        activeContent.innerHTML = "";
        activeContent.textContent = "Error: " + err.message;
        activeContent.classList.add("chat-error");
        finishStreaming(assistantEl, activeContent, "");
      });

    // ── Debounced markdown rendering ──────────────────────

    function debouncedRender(el, md) {
      if (renderTimer) clearTimeout(renderTimer);
      renderTimer = setTimeout(function () {
        renderMarkdown(el, md);
        scrollBottom();
        renderTimer = null;
      }, RENDER_DEBOUNCE);
    }

    function flushRender() {
      if (renderTimer) {
        clearTimeout(renderTimer);
        renderTimer = null;
      }
      if (fullText && activeContent) {
        renderMarkdown(activeContent, fullText);
      }
    }
  }

  function finishStreaming(el, contentEl, text) {
    isStreaming = false;
    sendBtn.disabled = false;
    el.classList.remove("chat-msg-streaming");
    setHint("", false);
    input.focus();

    // Final render + copy buttons for ALL content segments in this message
    var segments = el.querySelectorAll(".chat-msg-content");
    segments.forEach(function (seg) {
      addCopyButtons(seg);
    });
    if (text && contentEl) {
      renderMarkdown(contentEl, text);
      addCopyButtons(contentEl);
    }

    // Clean up empty content segments
    segments.forEach(function (seg) {
      if (!seg.textContent.trim() && !seg.querySelector("*")) seg.remove();
    });

    // Refresh preview with smooth overlay
    if (window.reloadFrame) {
      window.reloadFrame(frame);
    } else {
      try { frame.contentWindow.location.reload(); } catch (e) {}
    }
  }

  // ── Markdown rendering ──────────────────────────────────

  function renderMarkdown(el, md) {
    if (typeof marked === "undefined" || typeof DOMPurify === "undefined") {
      el.textContent = md; return;
    }

    el.innerHTML = DOMPurify.sanitize(marked.parse(md), {
      ALLOWED_TAGS: [
        "p", "br", "strong", "em", "del", "code", "pre", "blockquote",
        "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li",
        "a", "img", "table", "thead", "tbody", "tr", "th", "td",
        "hr", "div", "span", "sup", "sub",
      ],
      ALLOWED_ATTR: ["href", "src", "alt", "title", "class", "id", "target", "rel"],
    });
  }

  function addCopyButtons(container) {
    container.querySelectorAll("pre").forEach(function (pre) {
      if (pre.querySelector(".chat-code-header")) return;
      var code = pre.querySelector("code");
      if (!code) return;

      var lang = "";
      var m = (code.className || "").match(/language-(\w+)/);
      if (m) lang = m[1];

      var header = document.createElement("div");
      header.className = "chat-code-header";

      var langEl = document.createElement("span");
      langEl.className = "chat-code-lang";
      langEl.textContent = lang || "code";
      header.appendChild(langEl);

      var copyBtn = document.createElement("button");
      copyBtn.className = "chat-code-copy";
      copyBtn.innerHTML = ICONS.copy + " Copy";
      copyBtn.addEventListener("click", function () {
        navigator.clipboard.writeText(code.textContent || "").then(function () {
          copyBtn.classList.add("copied");
          copyBtn.innerHTML = ICONS.copied + " Copied";
          setTimeout(function () {
            copyBtn.classList.remove("copied");
            copyBtn.innerHTML = ICONS.copy + " Copy";
          }, 2000);
        });
      });
      header.appendChild(copyBtn);
      pre.insertBefore(header, pre.firstChild);
    });
  }

  // ── UI helpers ──────────────────────────────────────────

  function appendUserMessage(text) {
    var el = document.createElement("div");
    el.className = "chat-msg chat-msg-user";

    var avatar = document.createElement("div");
    avatar.className = "chat-msg-avatar";
    avatar.innerHTML = ICONS.user;
    el.appendChild(avatar);

    var body = document.createElement("div");
    body.className = "chat-msg-body";

    var role = document.createElement("div");
    role.className = "chat-msg-role";
    role.textContent = "You";
    body.appendChild(role);

    var content = document.createElement("div");
    content.className = "chat-msg-content";
    content.textContent = text;
    body.appendChild(content);

    el.appendChild(body);
    messages.appendChild(el);
    scrollBottom();
  }

  function appendAssistantMessage() {
    var el = document.createElement("div");
    el.className = "chat-msg chat-msg-assistant chat-msg-streaming";

    var avatar = document.createElement("div");
    avatar.className = "chat-msg-avatar";
    avatar.innerHTML = ICONS.bot;
    el.appendChild(avatar);

    var body = document.createElement("div");
    body.className = "chat-msg-body";

    var role = document.createElement("div");
    role.className = "chat-msg-role";
    role.textContent = "OpenDoc";
    body.appendChild(role);

    el.appendChild(body);
    messages.appendChild(el);
    scrollBottom();
    return el;
  }

  function createContentSegment() {
    var el = document.createElement("div");
    el.className = "chat-msg-content";
    return el;
  }

  function createThinking() {
    var el = document.createElement("div");
    el.className = "chat-thinking";
    el.innerHTML =
      '<div class="chat-thinking-dot"></div>' +
      '<div class="chat-thinking-dot"></div>' +
      '<div class="chat-thinking-dot"></div>';
    return el;
  }

  // ── Tool cards ─────────────────────────────────────────

  function getToolMeta(toolName) {
    return TOOL_META[toolName] || { icon: "settings", label: toolName, detailKey: null };
  }

  function extractDetail(toolName, argsStr) {
    var meta = getToolMeta(toolName);
    if (!meta.detailKey) {
      // Static label
      if (toolName === "build") return "Building site...";
      if (toolName === "get_config") return "Reading opendoc.yml";
      if (toolName === "update_nav") return "Updating navigation";
      return "";
    }

    try {
      var args = JSON.parse(argsStr);
      var val = args[meta.detailKey];
      if (val) return val;
    } catch (e) {}

    return "";
  }

  function appendToolCard(parentEl, toolName, argsStr, toolId) {
    var meta = getToolMeta(toolName);
    var detail = extractDetail(toolName, argsStr);

    var card = document.createElement("div");
    card.className = "chat-tool-card running";
    card.setAttribute("data-tool-id", toolId || "");
    card.setAttribute("data-tool-name", toolName);

    var header = document.createElement("div");
    header.className = "chat-tool-card-header";

    // Icon
    var iconEl = document.createElement("div");
    iconEl.className = "chat-tool-card-icon";
    iconEl.innerHTML = ICONS[meta.icon] || ICONS.settings;
    header.appendChild(iconEl);

    // Info
    var infoEl = document.createElement("div");
    infoEl.className = "chat-tool-card-info";

    var nameEl = document.createElement("div");
    nameEl.className = "chat-tool-card-name";
    nameEl.textContent = meta.label;
    infoEl.appendChild(nameEl);

    var detailEl = document.createElement("div");
    detailEl.className = "chat-tool-card-detail";
    detailEl.textContent = detail;
    infoEl.appendChild(detailEl);

    header.appendChild(infoEl);

    // Status: spinner
    var statusEl = document.createElement("div");
    statusEl.className = "chat-tool-card-status";
    statusEl.innerHTML = '<div class="chat-tool-card-spinner"></div>';
    header.appendChild(statusEl);

    // Chevron
    var chevronEl = document.createElement("div");
    chevronEl.className = "chat-tool-card-chevron";
    chevronEl.innerHTML = ICONS.chevron;
    header.appendChild(chevronEl);

    card.appendChild(header);

    // Expandable body
    var body = document.createElement("div");
    body.className = "chat-tool-card-body";
    card.appendChild(body);

    // Click to expand/collapse
    header.addEventListener("click", function () {
      card.classList.toggle("expanded");
    });

    parentEl.appendChild(card);
    return card;
  }

  function updateToolCardDetail(card, argsStr) {
    var toolName = card.getAttribute("data-tool-name") || "";
    var detail = extractDetail(toolName, argsStr);
    var detailEl = card.querySelector(".chat-tool-card-detail");
    if (detailEl && detail) detailEl.textContent = detail;
  }

  function markToolCardDone(card, finalArgs) {
    card.classList.remove("running");
    card.classList.add("done");

    // Update status icon
    var statusEl = card.querySelector(".chat-tool-card-status");
    if (statusEl) statusEl.innerHTML = '<div class="chat-tool-card-check">' + ICONS.check + "</div>";

    // Update detail if we got final args
    if (finalArgs) updateToolCardDetail(card, finalArgs);

    // Populate expandable body
    var body = card.querySelector(".chat-tool-card-body");
    if (body && finalArgs) {
      try {
        var parsed = JSON.parse(finalArgs);
        var argLabel = document.createElement("div");
        argLabel.className = "chat-tool-card-body-label";
        argLabel.textContent = "Arguments";
        body.appendChild(argLabel);

        var pre = document.createElement("pre");
        pre.textContent = JSON.stringify(parsed, null, 2);
        body.appendChild(pre);
      } catch (e) {
        if (finalArgs.trim()) {
          var pre2 = document.createElement("pre");
          pre2.textContent = finalArgs;
          body.appendChild(pre2);
        }
      }
    }
  }

  // ── Welcome screen ─────────────────────────────────────

  function renderWelcome() {
    var el = document.createElement("div");
    el.className = "chat-welcome";

    el.innerHTML =
      '<div class="chat-welcome-icon">' + ICONS.sparkle + "</div>" +
      '<div class="chat-welcome-title">OpenDoc AI</div>' +
      '<div class="chat-welcome-sub">Create pages, edit content, and manage your site with natural language.</div>' +
      '<div class="chat-welcome-actions">' +
        '<button class="chat-welcome-action" data-prompt="Create a new blog post">' +
          ICONS.file + '<span>Create a new page</span></button>' +
        '<button class="chat-welcome-action" data-prompt="What pages exist on my site?">' +
          ICONS.folder + '<span>Browse my pages</span></button>' +
        '<button class="chat-welcome-action" data-prompt="Show me my site configuration">' +
          ICONS.settings + '<span>View site config</span></button>' +
      "</div>";

    messages.appendChild(el);

    el.querySelectorAll(".chat-welcome-action").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var prompt = btn.getAttribute("data-prompt");
        if (prompt) {
          if (chatPct === 0) setChatPct(25, true);
          input.value = prompt;
          setTimeout(function () { sendMessage(); }, 300);
        }
      });
    });
  }

  // ── Utilities ──────────────────────────────────────────

  function setHint(text, streaming) {
    if (!hintEl) return;
    hintEl.textContent = text;
    hintEl.classList.toggle("streaming", !!streaming);
  }

  function scrollBottom() {
    requestAnimationFrame(function () {
      if (messages) {
        messages.scrollTo({ top: messages.scrollHeight, behavior: "smooth" });
      }
    });
  }

  function esc(str) {
    var d = document.createElement("div");
    d.textContent = str;
    return d.innerHTML;
  }

  // ── Init ────────────────────────────────────────────────

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
