package extensions

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	tabsOpenRe  = regexp.MustCompile(`^:{3}\s*tabs\s*$`)
	tabsCloseRe = regexp.MustCompile(`^:{3}\s*$`)
	tabDelimRe  = regexp.MustCompile(`^===\s+(.+)$`)
)

var tabGroupCounter int

// renderTabContent renders markdown content for a tab panel.
func renderTabContent(lines []string) string {
	md := goldmark.New(goldmark.WithRendererOptions(html.WithUnsafe()))
	var buf strings.Builder
	source := strings.Join(lines, "\n")
	if err := md.Convert([]byte(source), &buf); err != nil {
		return source
	}
	return buf.String()
}

// PreprocessTabs converts :::tabs / === label syntax to HTML.
func PreprocessTabs(source string) string {
	lines := strings.Split(source, "\n")
	var newLines []string
	i := 0

	for i < len(lines) {
		if tabsOpenRe.MatchString(strings.TrimSpace(lines[i])) {
			i++
			tabGroupCounter++
			groupID := fmt.Sprintf("code-tabs-%d", tabGroupCounter)

			type tab struct {
				label string
				lines []string
			}

			var tabs []tab
			var currentLabel string
			var currentLines []string
			started := false

			for i < len(lines) && !tabsCloseRe.MatchString(strings.TrimSpace(lines[i])) {
				tabMatch := tabDelimRe.FindStringSubmatch(lines[i])
				if tabMatch != nil {
					if started {
						tabs = append(tabs, tab{label: currentLabel, lines: currentLines})
					}
					currentLabel = strings.TrimSpace(tabMatch[1])
					currentLines = nil
					started = true
				} else {
					currentLines = append(currentLines, lines[i])
				}
				i++
			}

			// Save final tab
			if started {
				tabs = append(tabs, tab{label: currentLabel, lines: currentLines})
			}

			if i < len(lines) {
				i++ // skip closing :::
			}

			// Generate HTML
			if len(tabs) > 0 {
				newLines = append(newLines, "")
				newLines = append(newLines, fmt.Sprintf(`<div class="code-tabs" id="%s">`, groupID))

				// Tab navigation
				newLines = append(newLines, `<div class="code-tabs-nav" role="tablist">`)
				for idx, t := range tabs {
					active := ""
					if idx == 0 {
						active = " active"
					}
					tabID := fmt.Sprintf("%s-%d", groupID, idx)
					selected := "false"
					if idx == 0 {
						selected = "true"
					}
					newLines = append(newLines, fmt.Sprintf(
						`<button class="code-tab%s" role="tab" aria-selected="%s" data-tab="%s">%s</button>`,
						active, selected, tabID, t.label))
				}
				newLines = append(newLines, `</div>`)

				// Tab panels
				for idx, t := range tabs {
					active := ""
					if idx == 0 {
						active = " active"
					}
					tabID := fmt.Sprintf("%s-%d", groupID, idx)
					rendered := renderTabContent(t.lines)
					newLines = append(newLines, fmt.Sprintf(
						`<div class="code-tab-panel%s" id="%s" role="tabpanel">`, active, tabID))
					newLines = append(newLines, rendered)
					newLines = append(newLines, `</div>`)
				}

				newLines = append(newLines, `</div>`)
				newLines = append(newLines, "")
			}
		} else {
			newLines = append(newLines, lines[i])
			i++
		}
	}

	return strings.Join(newLines, "\n")
}
