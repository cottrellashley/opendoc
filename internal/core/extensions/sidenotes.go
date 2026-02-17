package extensions

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	sidenoteOpenRe  = regexp.MustCompile(`^:{3}\s+(sidenote|widget|deepdive|aside)\s+(.+)$`)
	sidenoteCloseRe = regexp.MustCompile(`^:{3}\s*$`)
)

var variantLabels = map[string]string{
	"sidenote": "Note",
	"widget":   "Widget",
	"deepdive": "Deep dive",
	"aside":    "Aside",
}

// renderInnerMarkdown renders markdown content for sidenote bodies.
func renderInnerMarkdown(lines []string) string {
	md := goldmark.New(goldmark.WithRendererOptions(html.WithUnsafe()))
	var buf strings.Builder
	source := strings.Join(lines, "\n")
	if err := md.Convert([]byte(source), &buf); err != nil {
		return source
	}
	return buf.String()
}

// PreprocessSidenotes converts :::sidenote/widget/deepdive/aside syntax to HTML.
func PreprocessSidenotes(source string) string {
	lines := strings.Split(source, "\n")
	var newLines []string
	noteCounter := 0
	i := 0

	for i < len(lines) {
		match := sidenoteOpenRe.FindStringSubmatch(lines[i])
		if match != nil {
			variant := match[1]
			title := strings.TrimSpace(match[2])
			label := variantLabels[variant]
			if label == "" {
				label = "Note"
			}
			noteCounter++
			noteID := fmt.Sprintf("mn-%d", noteCounter)
			i++

			// Collect inner content until closing :::
			var innerLines []string
			for i < len(lines) && !sidenoteCloseRe.MatchString(lines[i]) {
				innerLines = append(innerLines, lines[i])
				i++
			}
			if i < len(lines) {
				i++ // skip closing :::
			}

			innerHTML := renderInnerMarkdown(innerLines)

			newLines = append(newLines, "")
			newLines = append(newLines, fmt.Sprintf(
				`<div class="sidenote-block sidenote-block--%s" id="%s">`, variant, noteID))
			newLines = append(newLines, fmt.Sprintf(
				`<div class="marginnote marginnote--%s">`, variant))
			newLines = append(newLines, fmt.Sprintf(
				`<div class="marginnote-header" role="button" tabindex="0" aria-expanded="false">`+
					`<span class="marginnote-label">%s</span>`+
					`<span class="marginnote-title">%s</span>`+
					`</div>`, label, title))
			newLines = append(newLines, `<div class="marginnote-body">`)
			newLines = append(newLines, innerHTML)
			newLines = append(newLines, `</div>`)
			newLines = append(newLines, `</div>`)
			newLines = append(newLines, `</div>`)
			newLines = append(newLines, "")
		} else {
			newLines = append(newLines, lines[i])
			i++
		}
	}

	return strings.Join(newLines, "\n")
}
