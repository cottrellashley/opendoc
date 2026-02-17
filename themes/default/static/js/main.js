/* OpenDoc — Professional Theme JS */

(function () {
    "use strict";

    /* =============================================
       Dark Mode Toggle
       ============================================= */

    function initDarkMode() {
        var toggle = document.getElementById("theme-toggle");
        if (!toggle) return;

        toggle.addEventListener("click", function () {
            var html = document.documentElement;
            var current = html.getAttribute("data-theme");
            var next = current === "dark" ? "light" : "dark";
            html.setAttribute("data-theme", next);
            localStorage.setItem("opendoc-theme", next);
        });
    }

    /* =============================================
       Reading Progress Bar
       ============================================= */

    function initReadingProgress() {
        var bar = document.getElementById("reading-progress-bar");
        if (!bar) return;

        var ticking = false;

        function update() {
            var scrollTop = window.scrollY;
            var docHeight = document.documentElement.scrollHeight - window.innerHeight;
            if (docHeight <= 0) {
                bar.style.width = "0%";
                ticking = false;
                return;
            }
            bar.style.width = Math.min((scrollTop / docHeight) * 100, 100) + "%";
            ticking = false;
        }

        window.addEventListener("scroll", function () {
            if (!ticking) {
                requestAnimationFrame(update);
                ticking = true;
            }
        }, { passive: true });

        update();
    }

    /* =============================================
       Post Hero Line — Scroll Animation
       The decorative line under the hero fades out
       as the user scrolls into the article body.
       ============================================= */

    function initHeroScroll() {
        var hero = document.querySelector(".post-hero");
        if (!hero) return;

        var heroBottom = hero.offsetTop + hero.offsetHeight;
        var triggered = false;

        function check() {
            var scrollY = window.scrollY;
            var pastHero = scrollY > heroBottom - 100;

            if (pastHero && !triggered) {
                hero.classList.add("scrolled-past");
                triggered = true;
            } else if (!pastHero && triggered) {
                hero.classList.remove("scrolled-past");
                triggered = false;
            }
        }

        window.addEventListener("scroll", function () {
            requestAnimationFrame(check);
        }, { passive: true });

        check();
    }

    /* =============================================
       Scroll TOC — Active Section Tracking
       Fixed: properly updates after smooth scroll
       by using IntersectionObserver for accuracy.
       ============================================= */

    function initScrollTOC() {
        var toc = document.getElementById("scroll-toc");
        if (!toc) return;

        var links = toc.querySelectorAll("a");
        if (links.length === 0) return;

        // Build a map of heading IDs to TOC links
        var sections = [];
        links.forEach(function (link) {
            var href = link.getAttribute("href");
            if (href && href.startsWith("#")) {
                var el = document.getElementById(href.slice(1));
                if (el) sections.push({ el: el, link: link });
            }
        });

        if (sections.length === 0) return;

        // Use IntersectionObserver for accurate tracking
        var currentActive = null;

        var observer = new IntersectionObserver(function (entries) {
            // Find the topmost visible heading
            entries.forEach(function (entry) {
                if (entry.isIntersecting) {
                    var match = sections.find(function (s) {
                        return s.el === entry.target;
                    });
                    if (match) {
                        setActive(match.link);
                    }
                }
            });
        }, {
            rootMargin: "-80px 0px -70% 0px",
            threshold: 0
        });

        sections.forEach(function (s) {
            observer.observe(s.el);
        });

        function setActive(link) {
            if (currentActive === link) return;
            links.forEach(function (l) { l.classList.remove("active"); });
            link.classList.add("active");
            currentActive = link;
        }

        // Also do a manual scroll-position check for initial state and edge cases
        function updateFromScroll() {
            var scrollY = window.scrollY;
            var offset = 100;
            var found = null;

            for (var i = sections.length - 1; i >= 0; i--) {
                var rect = sections[i].el.getBoundingClientRect();
                if (rect.top <= offset) {
                    found = sections[i];
                    break;
                }
            }

            if (found) {
                setActive(found.link);
            }
        }

        // On scroll, do the manual check too (catches cases the observer misses)
        var scrollTicking = false;
        window.addEventListener("scroll", function () {
            if (!scrollTicking) {
                requestAnimationFrame(function () {
                    updateFromScroll();
                    scrollTicking = false;
                });
                scrollTicking = true;
            }
        }, { passive: true });

        // Smooth scroll on click — then update active after scroll settles
        links.forEach(function (link) {
            link.addEventListener("click", function (e) {
                var href = link.getAttribute("href");
                if (href && href.startsWith("#")) {
                    var target = document.getElementById(href.slice(1));
                    if (target) {
                        e.preventDefault();

                        // Immediately set this link as active
                        setActive(link);

                        var top = target.getBoundingClientRect().top + window.scrollY - 80;
                        window.scrollTo({ top: top, behavior: "smooth" });
                    }
                }
            });
        });

        // Initial state
        updateFromScroll();
    }

    /* =============================================
       Margin Notes — Reposition Under Section Headings
       Moves sidenote blocks to sit right after their
       section's h2 so they align below the divider line.
       ============================================= */

    function initSidenotePositioning() {
        var content = document.querySelector(".tufte-layout .content");
        if (!content) return;

        // Find all h2s and sidenote blocks inside the content area
        var allNodes = content.querySelectorAll("h2, .sidenote-block");
        if (allNodes.length === 0) return;

        // Group sidenote blocks by their preceding h2
        var currentH2 = null;
        var groups = []; // { h2: Element, notes: Element[] }

        allNodes.forEach(function (node) {
            if (node.tagName === "H2") {
                currentH2 = node;
                groups.push({ h2: node, notes: [] });
            } else if (node.classList.contains("sidenote-block") && currentH2) {
                groups[groups.length - 1].notes.push(node);
            }
        });

        // Move each group's notes to right after their h2
        groups.forEach(function (group) {
            if (group.notes.length === 0) return;
            // Insert notes right after the h2
            var ref = group.h2.nextSibling;
            group.notes.forEach(function (note) {
                content.insertBefore(note, ref);
            });
        });
    }

    /* =============================================
       Margin Notes — Hybrid Breakout Card / Bottom Sheet
       Desktop: floating card grows from the margin
       Mobile: bottom sheet slides up
       ============================================= */

    function initMarginNotes() {
        var blocks = document.querySelectorAll(".sidenote-block");
        if (blocks.length === 0) return;

        // Create shared backdrop (once)
        var backdrop = document.createElement("div");
        backdrop.className = "sidenote-backdrop";
        document.body.appendChild(backdrop);

        // (Close button removed — backdrop click and Escape handle closing)

        // Track which block is currently open
        var currentOpen = null;

        // Determine if we're on desktop layout
        function isDesktop() {
            return window.innerWidth >= 1101;
        }

        // Save original position data so we can restore after closing
        function savePosition(block) {
            if (!block._savedPosition) {
                block._savedPosition = {
                    parent: block.parentNode,
                    next: block.nextSibling
                };
            }
        }

        // --- Open a sidenote ---
        function openNote(block) {
            if (currentOpen === block) return;
            if (currentOpen) closeNote(currentOpen, true);

            savePosition(block);

            var header = block.querySelector(".marginnote-header");

            if (isDesktop()) {
                // Position the breakout card.
                // Vertically: centered in the viewport (clamped to stay on screen)
                // Horizontally: right-aligned, overlapping content + margin area
                var headerHeight = 56;
                var cardWidth = Math.min(540, window.innerWidth - 40);
                var maxCardHeight = window.innerHeight - headerHeight - 48;

                // Center vertically in available space below header
                var cardTop = headerHeight + 24;

                // Right edge: align with the right side of the content area
                var rightEdge = Math.max(16, (window.innerWidth - 940) / 2);

                block.classList.add("is-open");
                block.style.position = "fixed";
                block.style.top = cardTop + "px";
                block.style.right = rightEdge + "px";
                block.style.left = "auto";
                block.style.width = cardWidth + "px";
                block.style.bottom = "auto";
                block.style.maxHeight = maxCardHeight + "px";
                block.style.float = "none";
                block.style.margin = "0";
            } else {
                // Mobile bottom sheet — just add the class, CSS handles positioning
                block.classList.add("is-open");
            }

            if (header) {
                header.setAttribute("aria-expanded", "true");
                // Disable header click while open (close button handles it)
                header.style.pointerEvents = "none";
            }

            // Show backdrop
            backdrop.classList.add("is-visible");

            currentOpen = block;
        }

        // --- Close a sidenote ---
        function closeNote(block, instant) {
            if (!block) return;

            block.classList.remove("is-open");

            // Clear inline positioning styles
            block.style.position = "";
            block.style.top = "";
            block.style.right = "";
            block.style.left = "";
            block.style.width = "";
            block.style.bottom = "";
            block.style.maxHeight = "";
            block.style.float = "";
            block.style.margin = "";

            var header = block.querySelector(".marginnote-header");
            if (header) {
                header.setAttribute("aria-expanded", "false");
                header.style.pointerEvents = "";
            }

            // Hide backdrop
            backdrop.classList.remove("is-visible");

            if (currentOpen === block) currentOpen = null;
        }

        // --- Setup each block ---
        blocks.forEach(function (block) {
            var header = block.querySelector(".marginnote-header");

            // Header click: open the note (when collapsed)
            if (header) {
                header.addEventListener("click", function () {
                    if (!block.classList.contains("is-open")) {
                        openNote(block);
                    }
                });

                header.addEventListener("keydown", function (e) {
                    if ((e.key === "Enter" || e.key === " ") && !block.classList.contains("is-open")) {
                        e.preventDefault();
                        openNote(block);
                    }
                });
            }
        });

        // Backdrop click closes current note
        backdrop.addEventListener("click", function () {
            if (currentOpen) closeNote(currentOpen);
        });

        // Escape key closes current note
        document.addEventListener("keydown", function (e) {
            if (e.key === "Escape" && currentOpen) {
                closeNote(currentOpen);
            }
        });

        // On resize, reposition the card if one is open on desktop
        var resizeTimer;
        window.addEventListener("resize", function () {
            clearTimeout(resizeTimer);
            resizeTimer = setTimeout(function () {
                if (currentOpen && currentOpen.classList.contains("is-open")) {
                    // Just close and let user reopen — simplest approach for resize
                    closeNote(currentOpen);
                }
            }, 200);
        });
    }

    /* =============================================
       Code Copy Button
       Adds a copy-to-clipboard button on every <pre> block.
       ============================================= */

    function initCopyButtons() {
        var blocks = document.querySelectorAll(".content pre, .code-tab-panel pre");
        blocks.forEach(function (pre) {
            // Don't double-add
            if (pre.querySelector(".copy-btn")) return;

            var wrapper = document.createElement("div");
            wrapper.className = "code-block-wrapper";
            pre.parentNode.insertBefore(wrapper, pre);
            wrapper.appendChild(pre);

            var btn = document.createElement("button");
            btn.className = "copy-btn";
            btn.setAttribute("aria-label", "Copy code");
            btn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>';
            wrapper.appendChild(btn);

            btn.addEventListener("click", function () {
                var code = pre.querySelector("code");
                var text = code ? code.textContent : pre.textContent;
                navigator.clipboard.writeText(text).then(function () {
                    btn.classList.add("copied");
                    btn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>';
                    setTimeout(function () {
                        btn.classList.remove("copied");
                        btn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>';
                    }, 2000);
                });
            });
        });
    }

    /* =============================================
       Tabbed Code Blocks
       Switches between tab panels on click.
       ============================================= */

    function initCodeTabs() {
        var tabGroups = document.querySelectorAll(".code-tabs");
        tabGroups.forEach(function (group) {
            var buttons = group.querySelectorAll(".code-tab");
            var panels = group.querySelectorAll(".code-tab-panel");

            buttons.forEach(function (btn) {
                btn.addEventListener("click", function () {
                    var targetId = btn.getAttribute("data-tab");

                    // Deactivate all
                    buttons.forEach(function (b) {
                        b.classList.remove("active");
                        b.setAttribute("aria-selected", "false");
                    });
                    panels.forEach(function (p) {
                        p.classList.remove("active");
                    });

                    // Activate clicked tab
                    btn.classList.add("active");
                    btn.setAttribute("aria-selected", "true");
                    var panel = document.getElementById(targetId);
                    if (panel) panel.classList.add("active");
                });
            });
        });
    }

    /* =============================================
       KaTeX Math Rendering
       Renders LaTeX equations using KaTeX auto-render.
       ============================================= */

    function initMath() {
        if (typeof renderMathInElement !== "function") return;

        renderMathInElement(document.body, {
            delimiters: [
                { left: "$$", right: "$$", display: true },
                { left: "$", right: "$", display: false },
                { left: "\\begin{equation}", right: "\\end{equation}", display: true },
                { left: "\\begin{align}", right: "\\end{align}", display: true },
                { left: "\\begin{gather}", right: "\\end{gather}", display: true },
                { left: "\\(", right: "\\)", display: false },
                { left: "\\[", right: "\\]", display: true },
            ],
            ignoredTags: ["script", "noscript", "style", "textarea", "pre", "code"],
            throwOnError: false,
        });
    }

    /* =============================================
       Init
       ============================================= */

    document.addEventListener("DOMContentLoaded", function () {
        initDarkMode();
        initReadingProgress();
        initHeroScroll();
        initScrollTOC();
        initSidenotePositioning(); // Move notes under their section h2
        initMarginNotes();
        initCodeTabs();
        initCopyButtons();
        initMath();
    });
})();
