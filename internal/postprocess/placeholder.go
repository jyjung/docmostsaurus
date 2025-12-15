package postprocess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WrapPlaceholdersWithBackticks searches for {placeholder} patterns in markdown files
// and wraps them with backticks: {text} -> `{text}`
// It skips patterns that are already wrapped with backticks.
func WrapPlaceholdersWithBackticks(spaceDir string) error {
	return filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .md files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", path, err)
			return nil
		}

		// Process content
		newContent := wrapPlaceholders(string(content))

		// Only write if content changed
		if newContent != string(content) {
			if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
				fmt.Printf("Warning: failed to write file %s: %v\n", path, err)
				return nil
			}
			fmt.Printf("  Updated placeholders in: %s\n", path)
		}

		return nil
	})
}

// WrapAngleBracketsWithBackticks searches for <> patterns in markdown files
// and wraps them with backticks: <> -> `<>`, </> -> `</>`
// It skips patterns that are already wrapped with backticks or inside code blocks.
func WrapAngleBracketsWithBackticks(spaceDir string) error {
	return filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .md files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", path, err)
			return nil
		}

		// Process content
		newContent := wrapAngleBrackets(string(content))

		// Only write if content changed
		if newContent != string(content) {
			if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
				fmt.Printf("Warning: failed to write file %s: %v\n", path, err)
				return nil
			}
			fmt.Printf("  Updated angle brackets in: %s\n", path)
		}

		return nil
	})
}

// wrapAngleBrackets wraps <> and </> patterns with backticks
// Skips patterns already wrapped with backticks
// Skips patterns inside code blocks (triple backticks) or inline code (single backticks)
func wrapAngleBrackets(content string) string {
	var result strings.Builder
	i := 0
	inCodeBlock := false
	inInlineCode := false

	for i < len(content) {
		// Check for triple backticks (code block start/end)
		if i+2 < len(content) && content[i] == '`' && content[i+1] == '`' && content[i+2] == '`' {
			inCodeBlock = !inCodeBlock
			result.WriteString("```")
			i += 3
			continue
		}

		// Check for single backtick (inline code start/end) - only when not in code block
		if !inCodeBlock && content[i] == '`' {
			inInlineCode = !inInlineCode
			result.WriteByte('`')
			i++
			continue
		}

		// If inside code block or inline code, just copy content as-is
		if inCodeBlock || inInlineCode {
			result.WriteByte(content[i])
			i++
			continue
		}

		// Check for <> or </> patterns
		if content[i] == '<' {
			// Check for </> pattern
			if i+2 < len(content) && content[i+1] == '/' && content[i+2] == '>' {
				// Check if already wrapped with backticks
				alreadyWrapped := false
				if i > 0 && content[i-1] == '`' {
					if i+3 < len(content) && content[i+3] == '`' {
						alreadyWrapped = true
					}
				}

				if alreadyWrapped {
					result.WriteString("</>")
				} else {
					result.WriteString("`</>`")
				}
				i += 3
				continue
			}

			// Check for <> pattern
			if i+1 < len(content) && content[i+1] == '>' {
				// Check if already wrapped with backticks
				alreadyWrapped := false
				if i > 0 && content[i-1] == '`' {
					if i+2 < len(content) && content[i+2] == '`' {
						alreadyWrapped = true
					}
				}

				if alreadyWrapped {
					result.WriteString("<>")
				} else {
					result.WriteString("`<>`")
				}
				i += 2
				continue
			}
		}

		result.WriteByte(content[i])
		i++
	}

	return result.String()
}

// WrapRawHTMLWithCodeBlock searches for raw HTML (like <table>, <tbody>, etc.) in markdown files
// that are not already inside code blocks and wraps them with triple backticks.
func WrapRawHTMLWithCodeBlock(spaceDir string) error {
	return filepath.Walk(spaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .md files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", path, err)
			return nil
		}

		// Process content
		newContent := wrapRawHTML(string(content))

		// Only write if content changed
		if newContent != string(content) {
			if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
				fmt.Printf("Warning: failed to write file %s: %v\n", path, err)
				return nil
			}
			fmt.Printf("  Wrapped raw HTML in: %s\n", path)
		}

		return nil
	})
}

// wrapRawHTML wraps raw HTML blocks with triple backticks
// It detects HTML tags like <table>, <tbody>, <tr>, <th>, <td>, <div>, etc.
// that are not already inside code blocks and wraps them with ```html ... ```
func wrapRawHTML(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inCodeBlock := false
	htmlStartIdx := -1
	htmlLines := []string{}

	// HTML tags that indicate raw HTML content that should be wrapped
	htmlTagPattern := []string{"<table", "<tbody", "<thead", "<tr>", "<th", "<td", "</table>", "</tbody>", "</thead>", "</tr>", "</th>", "</td>"}

	isHTMLLine := func(line string) bool {
		trimmed := strings.TrimSpace(line)
		for _, pattern := range htmlTagPattern {
			if strings.Contains(strings.ToLower(trimmed), pattern) {
				return true
			}
		}
		return false
	}

	for i, line := range lines {
		// Check for triple backticks (code block start/end)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			// If we were collecting HTML and hit a code block, flush HTML first
			if htmlStartIdx >= 0 {
				result = append(result, "```html")
				result = append(result, htmlLines...)
				result = append(result, "```")
				htmlStartIdx = -1
				htmlLines = []string{}
			}
			result = append(result, line)
			continue
		}

		// If inside code block, just copy content as-is
		if inCodeBlock {
			result = append(result, line)
			continue
		}

		// Check if this line contains HTML
		if isHTMLLine(line) {
			if htmlStartIdx < 0 {
				htmlStartIdx = i
			}
			htmlLines = append(htmlLines, line)
		} else {
			// If we were collecting HTML lines, wrap them now
			if htmlStartIdx >= 0 {
				result = append(result, "```html")
				result = append(result, htmlLines...)
				result = append(result, "```")
				htmlStartIdx = -1
				htmlLines = []string{}
			}
			result = append(result, line)
		}
	}

	// Handle any remaining HTML lines at end of file
	if htmlStartIdx >= 0 {
		result = append(result, "```html")
		result = append(result, htmlLines...)
		result = append(result, "```")
	}

	return strings.Join(result, "\n")
}

// wrapPlaceholders wraps {placeholder} patterns with backticks
// Skips patterns already wrapped with backticks
// Skips patterns inside code blocks (triple backticks)
// Skips patterns inside inline code (single backticks)
// Skips patterns inside markdown link paths: [text](path) or ![alt](path)
// Skips JSON-like patterns: "key": { or "key" : {
func wrapPlaceholders(content string) string {
	var result strings.Builder
	i := 0
	inCodeBlock := false
	inInlineCode := false
	inLinkPath := false

	for i < len(content) {
		// Check for triple backticks (code block start/end)
		if i+2 < len(content) && content[i] == '`' && content[i+1] == '`' && content[i+2] == '`' {
			inCodeBlock = !inCodeBlock
			result.WriteString("```")
			i += 3
			continue
		}

		// If inside code block, just copy content as-is
		if inCodeBlock {
			result.WriteByte(content[i])
			i++
			continue
		}

		// Check for single backtick (inline code start/end) - only when not in code block
		if content[i] == '`' {
			inInlineCode = !inInlineCode
			result.WriteByte('`')
			i++
			continue
		}

		// If inside inline code, just copy content as-is
		if inInlineCode {
			result.WriteByte(content[i])
			i++
			continue
		}

		// Check for markdown link start: ]( - this marks the start of a link path
		// Handles both [text](path) and ![alt](path)
		if i+1 < len(content) && content[i] == ']' && content[i+1] == '(' {
			inLinkPath = true
			result.WriteString("](")
			i += 2
			continue
		}

		// Check for link path end: ) - this marks the end of a link path
		if inLinkPath && content[i] == ')' {
			inLinkPath = false
			result.WriteByte(')')
			i++
			continue
		}

		// Also handle newline ending the link path (malformed link)
		if inLinkPath && content[i] == '\n' {
			inLinkPath = false
		}

		// If inside link path, just copy content as-is
		if inLinkPath {
			result.WriteByte(content[i])
			i++
			continue
		}

		// Check if we're at a potential placeholder
		if content[i] == '{' {
			// Find the closing brace
			j := i + 1
			for j < len(content) && content[j] != '}' && content[j] != '\n' {
				j++
			}

			// If we found a closing brace on the same line
			if j < len(content) && content[j] == '}' {
				placeholder := content[i : j+1]

				// Check if already wrapped with backticks
				alreadyWrapped := false
				if i > 0 && content[i-1] == '`' {
					// Check if there's a closing backtick after the placeholder
					if j+1 < len(content) && content[j+1] == '`' {
						alreadyWrapped = true
					}
				}

				// Check if this is a JSON-like pattern: "key": { or "key" : {
				// Look backwards for colon followed by optional spaces
				isJSONPattern := false
				if i > 0 {
					// Look backwards, skipping spaces
					k := i - 1
					for k >= 0 && (content[k] == ' ' || content[k] == '\t') {
						k--
					}
					// Check for colon
					if k >= 0 && content[k] == ':' {
						k--
						// Skip spaces after quote
						for k >= 0 && (content[k] == ' ' || content[k] == '\t') {
							k--
						}
						// Check for closing quote (indicating JSON key)
						if k >= 0 && (content[k] == '"' || content[k] == '\'') {
							isJSONPattern = true
						}
					}
				}

				if alreadyWrapped || isJSONPattern {
					// Already wrapped or JSON pattern, just write the placeholder
					result.WriteString(placeholder)
				} else {
					// Wrap with backticks
					result.WriteByte('`')
					result.WriteString(placeholder)
					result.WriteByte('`')
				}
				i = j + 1
				continue
			}
		}

		result.WriteByte(content[i])
		i++
	}

	return result.String()
}
