package runtime

import (
	"regexp"
	"strings"
	"unicode"
)

// ANSI escape code regex - enhanced to cover more sequences
var (
	// Basic CSI sequences: ESC[...letter
	csiRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)
	// OSC sequences: ESC]...BEL or ESC]...ST
	oscRegex = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	// DCS/APC/PM/SOS sequences
	dcsRegex = regexp.MustCompile(`\x1b[PX^_][^\x1b]*\x1b\\`)
	// Single char escapes: ESC followed by one char
	singleEscRegex = regexp.MustCompile(`\x1b[()#%][A-Za-z0-9]?`)
	// Mouse tracking sequences
	mouseRegex = regexp.MustCompile(`\x1b\[M...|\x1b\[<[0-9;]*[mM]`)
	// Braille pattern characters (spinners)
	brailleRegex = regexp.MustCompile(`[\x{2800}-\x{28FF}]`)

	// Dynamic content patterns to normalize (percentage, time, etc.)
	// These are NOT removed but normalized for comparison purposes
	percentRegex = regexp.MustCompile(`\d+%`)
	timeRegex    = regexp.MustCompile(`\d+:\d+:\d+|\d+:\d+|\d+s|\d+ms`)
	numberRegex  = regexp.MustCompile(`\b\d+\b`)
)

// CleanOutput removes all ANSI escape codes and control characters.
func CleanOutput(input string) string {
	clean := input
	clean = csiRegex.ReplaceAllString(clean, "")
	clean = oscRegex.ReplaceAllString(clean, "")
	clean = dcsRegex.ReplaceAllString(clean, "")
	clean = singleEscRegex.ReplaceAllString(clean, "")
	clean = mouseRegex.ReplaceAllString(clean, "")
	
	// Remove remaining raw ESC chars
	clean = strings.ReplaceAll(clean, "\x1b", "")
	// Remove other control chars except newline/tab
	var b strings.Builder
	for _, r := range clean {
		if r == '\n' || r == '\t' || r == '\r' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// ExtractConclusion extracts meaningful content from terminal output.
// It uses a tiered strategy for robustness:
// 1. Explicit Marker (`:::VIBE_OUTPUT:::`) - 100% reliability if present.
// 2. Dynamic Frame Detection (`isolateFinalFrame`) - Heuristic for TUI tools without markers.
// 3. Fallback - Returns cleaned full content.
func ExtractConclusion(input string) string {
	// 1. Clean ANSI first
	// We do this early so both strategies work on clean text.
	clean := CleanOutput(input)

	var content string
	
	// Strategy A: Explicit Delimiter (Priority 1)
	if extracted, found := extractFromDelimiter(clean); found {
		content = extracted
	} else {
		// Strategy B: Dynamic Frame Isolatation (Priority 2)
		// If no marker found, we try to detect the TUI frame structure.
		content = isolateFinalFrame(clean) 
	}

	// 4. Common Cleanup
	// Regardless of strategy, we remove generic TUI noise (help tips, status bars)
	// that might remain at the bottom of the output.
	content = removeTUINoise(content)

	// 5. Standard Cleanup
	lines := strings.Split(content, "\n")
	filtered := filterNoiseLines(lines)
	result := deduplicateConsecutive(filtered)
	
	return strings.Join(result, "\n")
}

// isolateFinalFrame uses a heuristic to detect the "Frame Separator" automatically.
// It assumes that in a TUI recording, the "Status Bar" or "Header" appears repeatedly between frames.
// We find the most frequent "complex" line and use it as the delimiter.
func isolateFinalFrame(input string) string {
	lines := strings.Split(input, "\n")
	if len(lines) < 10 {
		return input
	}

	// 1. Frequency Analysis
	lineCounts := make(map[string]int)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 10 { // Only consider "substantial" lines as potential separators
			lineCounts[trimmed]++
		}
	}

	// 2. Identify the likely Separator
	// We look for the most frequent line that appears at least 3 times.
	var bestSeparator string
	maxCount := 0

	for line, count := range lineCounts {
		if count > 2 && count > maxCount {
			maxCount = count
			bestSeparator = line
		}
	}

	// If no recurring separator found, return original input (Priority 3: Fallback)
	if bestSeparator == "" {
		return input
	}

	// 3. Split and take the last frame
	// We use the original input to preserve newlines structure, relying on matching the trimmed content
	var finalBlocks []string
	var currentBlock strings.Builder
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if trimmed == bestSeparator {
			if currentBlock.Len() > 0 {
				finalBlocks = append(finalBlocks, currentBlock.String())
				currentBlock.Reset()
			}
			continue 
		}
		
		currentBlock.WriteString(line + "\n")
	}
	if currentBlock.Len() > 0 {
		finalBlocks = append(finalBlocks, currentBlock.String())
	}

	// Return the last substantial block
	for i := len(finalBlocks) - 1; i >= 0; i-- {
		block := strings.TrimSpace(finalBlocks[i])
		if len(block) > 20 { 
			return block
		}
	}

	return input
}

// extractFromDelimiter looks for the magic token ":::VIBE_OUTPUT:::"
// Returns the content and true if found, or original and false if not.
func extractFromDelimiter(input string) (string, bool) {
	token := ":::VIBE_OUTPUT:::"
	idx := strings.LastIndex(input, token)
	if idx != -1 {
		// Return everything after the token
		return strings.TrimSpace(input[idx+len(token):]), true
	}
	return input, false
}

// removeTUINoise removes generic TUI noise using heuristics rather than hardcoded prompts.
func removeTUINoise(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Heuristic 1: Filter out lines that look like "Help" tips
		// e.g., "1. Type /help...", "Tips: Press...", "入门提示..."
		if regexp.MustCompile(`(?i)^(tips|hint|usage|入门|提示)[:：]`).MatchString(trimmed) {
			continue
		}
		if regexp.MustCompile(`(?i)(ctrl|alt|esc)\s*\+`).MatchString(trimmed) {
			continue
		}

		// Heuristic 2: Filter out "Loaded files" or "Context" info lines
		// e.g., "Loaded: file.go", "Context: 2 files"
		if regexp.MustCompile(`(?i)^(loaded|context|已加载)[:：]`).MatchString(trimmed) {
			continue
		}
		
		// Heuristic 3: Filter box drawings that are likely borders (only if line is mostly borders)
		// e.g. "──────", "══════"
		borderChars := 0
		for _, r := range trimmed {
			if strings.ContainsRune("─═-_│┃║╔╗╚╝", r) {
				borderChars++
			}
		}
		if len(trimmed) > 0 && float64(borderChars)/float64(len(trimmed)) > 0.8 {
			continue
		}

		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

// truncateBottomStagnation scans from the bottom and truncates when
// we detect repeated lines (ignoring dynamic content like %, time, numbers).
// This removes TUI refresh noise where the same frame is output repeatedly.
func truncateBottomStagnation(lines []string, threshold int) []string {
	if len(lines) < threshold {
		return lines
	}

	// Scan from bottom
	repeatCount := 1
	cutoffIndex := len(lines)

	for i := len(lines) - 2; i >= 0; i-- {
		current := normalizeDynamicContent(strings.TrimSpace(lines[i]))
		next := normalizeDynamicContent(strings.TrimSpace(lines[i+1]))

		if current == "" || next == "" {
			// Skip empty lines in comparison
			continue
		}

		if current == next {
			repeatCount++
			if repeatCount >= threshold {
				// Found stagnation point - cut here
				cutoffIndex = i + 1
			}
		} else {
			// Reset counter and update cutoff if we had enough repeats
			if repeatCount >= threshold {
				break
			}
			repeatCount = 1
		}
	}

	if cutoffIndex < len(lines) {
		return lines[:cutoffIndex]
	}
	return lines
}

// normalizeDynamicContent replaces dynamic content (%, time, numbers) with placeholders
// so that lines differing only in these values compare as equal.
func normalizeDynamicContent(line string) string {
	normalized := percentRegex.ReplaceAllString(line, "<PCT>")
	normalized = timeRegex.ReplaceAllString(normalized, "<TIME>")
	// Don't normalize all numbers - just the ones in specific patterns
	return normalized
}

// filterNoiseLines removes known TUI noise patterns.
func filterNoiseLines(lines []string) []string {
	noisePatterns := []*regexp.Regexp{
		// Common AI CLI prompts and input areas
		regexp.MustCompile(`(?i)^>\s*(输入您的消息|Type your message|Enter your)`),
		regexp.MustCompile(`(?i)^\?\s*Select`),
		// Status indicators
		regexp.MustCompile(`(?i)(Thinking\.{0,3}|Smart mode|esc to cancel)`),
		regexp.MustCompile(`(?i)(loading\.{0,3}\d*s?\)?)`),
		// VibeMux/Tool specific status bars
		regexp.MustCompile(`(?i)(vibmux|glm-\d+\.\d+|qwen.*code)`),
		// Braille spinners (backup if regex didn't catch as runes)
		brailleRegex,
		// Horizontal rules/separators
		regexp.MustCompile(`^[\s]*[─═\-_]{5,}[\s]*$`),
		// Empty prompts
		regexp.MustCompile(`^\s*>\s*$`),
		// Claude Code cost summary (keep for context) - actually let's keep this
		// Gemini CLI specific
		regexp.MustCompile(`(?i)^Gemini\s*>`),
		// Common empty box drawing
		regexp.MustCompile(`^[\s│┃|]*$`),
		
		// --- VibeMux Analysis Specific Noise ---
		// Status bar: "vibemux (main*) ... sandbox (99%)"
		regexp.MustCompile(`(?i)vibemux\s*\(.*`),
		regexp.MustCompile(`(?i)sandbox\s*\(\d+%`),
		// Key hints: "m to toggle)", "Enter to select"
		regexp.MustCompile(`(?i).*\s+to\s+(toggle|select|switch|cancel)\)?\s*$`),
		// Truncated paths in status bars: ".../path/to/file" or "...C:\path"
		regexp.MustCompile(`^\.{3}.*[\\/].*`),
		// Specific TUI status words
		regexp.MustCompile(`(?i)\s+coder-model\s*$`),
		regexp.MustCompile(`(?i)\s+sandbox\s*$`),
		// Context counters
		regexp.MustCompile(`(?i)context\s+left\s+\d+`),
		// Loose "1 file" indicators if they appear alone
		regexp.MustCompile(`(?i)^\s*-\s*\d+\s*个\s*.*文件\s*$`),
		regexp.MustCompile(`(?i)^\s*-\s*\d+\s*file\(s\)\s*$`),

		// --- Chain Context Injection Boilerplate ---
		// We use QuoteMeta to ensure the constants defined in chain.go are matched exactly
		regexp.MustCompile(`^` + regexp.QuoteMeta(ChainPromptHeader)),
		regexp.MustCompile(`^` + regexp.QuoteMeta(ChainPromptInstruction)),
	}

	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		isNoise := false
		for _, p := range noisePatterns {
			if p.MatchString(trimmed) {
				isNoise = true
				break
			}
		}
		if isNoise {
			continue
		}

		filtered = append(filtered, line)
	}
	return filtered
}

// deduplicateConsecutive removes consecutive duplicate lines.
func deduplicateConsecutive(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}

	result := make([]string, 0, len(lines))
	result = append(result, lines[0])

	for i := 1; i < len(lines); i++ {
		// Use normalized comparison for dedup too
		currentNorm := normalizeDynamicContent(strings.TrimSpace(lines[i]))
		prevNorm := normalizeDynamicContent(strings.TrimSpace(lines[i-1]))
		if currentNorm != prevNorm {
			result = append(result, lines[i])
		}
	}
	return result
}

// deduplicateIncrementalOutput removes incremental streaming output patterns.
// This detects when a line is a prefix of the next line (or vice versa),
// which happens during AI streaming output where characters are added incrementally.
// Example: "君" -> "君不见" -> "君不见黄河" should only keep "君不见黄河"
func deduplicateIncrementalOutput(lines []string) []string {
	if len(lines) < 2 {
		return lines
	}

	result := make([]string, 0, len(lines))
	
	for i := 0; i < len(lines); i++ {
		current := strings.TrimSpace(lines[i])
		if current == "" {
			continue
		}
		
		// Look ahead to see if current line is a prefix of any upcoming line
		isPrefix := false
		for j := i + 1; j < len(lines) && j <= i+10; j++ { // Look ahead max 10 lines
			next := strings.TrimSpace(lines[j])
			if next == "" {
				continue
			}
			// Check if current is a prefix of next (incremental output pattern)
			if strings.HasPrefix(next, current) && len(next) > len(current) {
				isPrefix = true
				break
			}
			// Also check if they share a common starting pattern (same content being built)
			if len(current) > 5 && len(next) > 5 {
				prefix := current
				if len(prefix) > 20 {
					prefix = prefix[:20]
				}
				if strings.HasPrefix(next, prefix) {
					isPrefix = true
					break
				}
			}
		}
		
		if !isPrefix {
			result = append(result, lines[i])
		}
	}
	
	return result
}
