// Package extensions implements OpenDoc markdown preprocessors.
package extensions

import (
	"fmt"
	"regexp"
	"strings"
)

// ── Regex patterns ──────────────────────────────────────────

var (
	inlineMathRe    = regexp.MustCompile(`(?:^|[^$])\$([^$]+?)\$(?:[^$]|$)`)
	blockMathOpenRe = regexp.MustCompile(`^\$\$\s*$`)
	blockMathCloseRe = regexp.MustCompile(`^\$\$\s*(?:\{#(eq:\S+)\})?\s*$`)
	latexEnvOpenRe  = regexp.MustCompile(`^\\begin\{(equation|align|alignat|gather|multline)\*?\}`)
	theoremOpenRe   = regexp.MustCompile(`^:::(theorem|definition|lemma|proposition|corollary|remark|proof)(?:\s+(.+))?\s*$`)
	theoremCloseRe  = regexp.MustCompile(`^:::\s*$`)
)

var theoremLabels = map[string]string{
	"theorem":     "Theorem",
	"definition":  "Definition",
	"lemma":       "Lemma",
	"proposition": "Proposition",
	"corollary":   "Corollary",
	"remark":      "Remark",
	"proof":       "Proof",
}

// PreprocessMath processes LaTeX math expressions in markdown source.
// Supports inline $...$, display $$...$$, LaTeX environments,
// equation numbering, and theorem blocks.
func PreprocessMath(source string) string {
	lines := strings.Split(source, "\n")
	var newLines []string
	equationCounter := 0
	theoremCounters := make(map[string]int)
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// --- Theorem-like environments: :::theorem ... ::: ---
		if thmMatch := theoremOpenRe.FindStringSubmatch(trimmed); thmMatch != nil {
			envType := thmMatch[1]
			customTitle := ""
			if len(thmMatch) > 2 {
				customTitle = strings.TrimSpace(thmMatch[2])
			}
			i++

			var innerLines []string
			for i < len(lines) && !theoremCloseRe.MatchString(strings.TrimSpace(lines[i])) {
				innerLines = append(innerLines, lines[i])
				i++
			}
			if i < len(lines) {
				i++ // skip closing :::
			}

			// Build label
			var label string
			if envType != "proof" {
				theoremCounters[envType]++
				num := theoremCounters[envType]
				label = fmt.Sprintf("%s %d", theoremLabels[envType], num)
				if customTitle != "" {
					label += fmt.Sprintf(" (%s)", customTitle)
				}
			} else {
				if customTitle != "" {
					label = fmt.Sprintf("Proof (%s)", customTitle)
				} else {
					label = "Proof"
				}
			}

			// Recursively process inner content for math
			processedInner := strings.Split(PreprocessMath(strings.Join(innerLines, "\n")), "\n")

			newLines = append(newLines, "")
			newLines = append(newLines, fmt.Sprintf(`<div class="%s" markdown="1">`, envType))
			newLines = append(newLines, fmt.Sprintf(`<div class="%s-head">%s</div>`, envType, label))
			newLines = append(newLines, "")
			newLines = append(newLines, processedInner...)
			newLines = append(newLines, "")
			if envType == "proof" {
				newLines = append(newLines, `<div class="proof-qed"></div>`)
			}
			newLines = append(newLines, "</div>")
			newLines = append(newLines, "")
			continue
		}

		// --- LaTeX environments: \begin{equation} ... \end{equation} ---
		if envMatch := latexEnvOpenRe.FindStringSubmatch(trimmed); envMatch != nil {
			envName := envMatch[1]
			closeRe := regexp.MustCompile(`^\\end\{` + regexp.QuoteMeta(envName) + `\*?\}`)
			var mathLines []string
			mathLines = append(mathLines, line)
			i++
			for i < len(lines) && !closeRe.MatchString(strings.TrimSpace(lines[i])) {
				mathLines = append(mathLines, lines[i])
				i++
			}
			if i < len(lines) {
				mathLines = append(mathLines, lines[i])
				i++
			}

			latex := strings.Join(mathLines, "\n")
			newLines = append(newLines, "")
			newLines = append(newLines, fmt.Sprintf(`<div class="math-display" data-math-display>$$%s$$</div>`, latex))
			newLines = append(newLines, "")
			continue
		}

		// --- Block math: $$ ... $$ ---
		if blockMathOpenRe.MatchString(trimmed) {
			i++
			var mathBlock []string
			var eqLabel string
			for i < len(lines) {
				closeMatch := blockMathCloseRe.FindStringSubmatch(strings.TrimSpace(lines[i]))
				if closeMatch != nil {
					if len(closeMatch) > 1 {
						eqLabel = closeMatch[1]
					}
					i++
					break
				}
				mathBlock = append(mathBlock, lines[i])
				i++
			}

			latex := strings.Join(mathBlock, "\n")
			attrs := `class="math-display" data-math-display`
			if eqLabel != "" {
				equationCounter++
				attrs += fmt.Sprintf(` data-equation-number="%d"`, equationCounter)
				attrs += fmt.Sprintf(` id="%s"`, eqLabel)
			}

			newLines = append(newLines, "")
			newLines = append(newLines, fmt.Sprintf(`<div %s>$$%s$$</div>`, attrs, latex))
			newLines = append(newLines, "")
			continue
		}

		// --- Inline math on this line ---
		newLines = append(newLines, protectInlineMath(line))
		i++
	}

	return strings.Join(newLines, "\n")
}

// protectInlineMath escapes markdown-sensitive characters inside inline math.
func protectInlineMath(line string) string {
	// Use a manual approach since Go's regexp doesn't support lookbehind.
	// Find $...$ patterns that are not $$ and protect them.
	result := []byte(line)
	var out []byte
	idx := 0
	for idx < len(result) {
		// Skip $$ (display math marker)
		if idx < len(result)-1 && result[idx] == '$' && result[idx+1] == '$' {
			out = append(out, '$', '$')
			idx += 2
			continue
		}

		if result[idx] == '$' {
			// Look for closing $
			end := idx + 1
			for end < len(result) {
				if result[end] == '$' && (end+1 >= len(result) || result[end+1] != '$') {
					break
				}
				if result[end] == '$' && end+1 < len(result) && result[end+1] == '$' {
					end = -1
					break
				}
				end++
			}
			if end > idx+1 && end < len(result) {
				// Found inline math: $content$
				inner := string(result[idx+1 : end])
				// Escape markdown-sensitive chars
				inner = strings.ReplaceAll(inner, "_", "\\_")
				inner = strings.ReplaceAll(inner, "*", "\\*")
				out = append(out, '$')
				out = append(out, []byte(inner)...)
				out = append(out, '$')
				idx = end + 1
				continue
			}
		}

		out = append(out, result[idx])
		idx++
	}
	return string(out)
}
