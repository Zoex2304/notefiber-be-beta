package lexical

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Parser handles Lexical JSON to Markdown conversion
type Parser struct{}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{}
}

// Parse converts a Lexical JSON string to Semantic Markdown
func (p *Parser) Parse(jsonContent string) (string, error) {
	var root LexicalRoot
	if err := json.Unmarshal([]byte(jsonContent), &root); err != nil {
		return "", fmt.Errorf("failed to parse lexical json: %w", err)
	}

	var sb strings.Builder
	p.walkNode(root.Root, &sb, 0)
	return sb.String(), nil
}

// ParseContent is a convenience function to parse a raw string
// It attempts to parse as Lexical JSON; if it fails (not JSON or error), it returns the original string
func ParseContent(content string) string {
	// Optimization: Quick check if it looks like Lexical
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, `{"root":`) {
		return content
	}

	p := NewParser()
	md, err := p.Parse(trimmed)
	if err != nil {
		// Fallback to original content if parsing fails
		return content
	}
	return md
}

// walkNode traverses the tree and writers markdown
func (p *Parser) walkNode(node Node, sb *strings.Builder, depth int) {
	switch node.Type {
	case "root":
		for _, child := range node.Children {
			p.walkNode(child, sb, depth)
			sb.WriteString("\n")
		}

	case "paragraph":
		p.handleParagraph(node, sb, depth)

	case "text":
		p.handleText(node, sb)

	case "list":
		p.handleList(node, sb, depth)

	// ListItems are handled by handleList to ensure correct marking (bullet/number/check)
	case "listitem":
		// Fallback if encountered loose
		for _, child := range node.Children {
			p.walkNode(child, sb, depth)
		}

	case "table":
		p.handleTable(node, sb)

	case "link":
		p.handleLink(node, sb)

	case "horizontalrule":
		sb.WriteString("---\n")

	default:
		// Generic recursion
		for _, child := range node.Children {
			p.walkNode(child, sb, depth)
		}
	}
}

func (p *Parser) handleParagraph(node Node, sb *strings.Builder, depth int) {
	align := ""
	if fmtStr, ok := node.Format.(string); ok && fmtStr != "" && fmtStr != "left" {
		align = fmtStr
	}

	if align != "" {
		sb.WriteString(fmt.Sprintf("<div align=\"%s\">", align))
	}

	for _, child := range node.Children {
		p.walkNode(child, sb, depth)
	}

	if align != "" {
		sb.WriteString("</div>")
	}
	sb.WriteString("\n")
}

func (p *Parser) handleText(node Node, sb *strings.Builder) {
	text := node.Text

	// Annotations
	styleStyles := ParseStyle(node.Style)
	openTag := styleStyles.BuildAnnotatedOpenTag()
	if openTag != "" {
		sb.WriteString(openTag)
	}

	// Format
	fmtInt := 0
	if f, ok := node.Format.(float64); ok {
		fmtInt = int(f)
	} else if f, ok := node.Format.(int); ok {
		fmtInt = f
	}

	isBold := (fmtInt & FormatBold) != 0
	isItalic := (fmtInt & FormatItalic) != 0
	isUnderline := (fmtInt & FormatUnderline) != 0
	isCode := (fmtInt & FormatCode) != 0
	isStrike := (fmtInt & FormatStrikethrough) != 0

	// Apply wrappers (Code > Bold > Italic > Underline > Strike)
	// Markdown doesn't support underline natively everywhere, using HTML <u>
	if isCode {
		sb.WriteString("`")
	}
	if isBold {
		sb.WriteString("**")
	}
	if isItalic {
		sb.WriteString("_")
	}
	if isUnderline {
		sb.WriteString("<u>")
	}
	if isStrike {
		sb.WriteString("~~")
	}

	sb.WriteString(text)

	if isStrike {
		sb.WriteString("~~")
	}
	if isUnderline {
		sb.WriteString("</u>")
	}
	if isItalic {
		sb.WriteString("_")
	}
	if isBold {
		sb.WriteString("**")
	}
	if isCode {
		sb.WriteString("`")
	}

	if openTag != "" {
		sb.WriteString("</span>")
	}
}

func (p *Parser) handleLink(node Node, sb *strings.Builder) {
	// Standard MD link: [text](url)
	sb.WriteString("[")
	for _, child := range node.Children {
		p.walkNode(child, sb, 0) // depth 0 for inline
	}
	sb.WriteString(fmt.Sprintf("](%s)", node.URL))
}

func (p *Parser) handleList(node Node, sb *strings.Builder, depth int) {
	listType := node.ListType
	index := 1
	if node.Start > 0 {
		index = node.Start
	}

	for _, child := range node.Children {
		// Only process listitems
		if child.Type != "listitem" {
			continue
		}

		// Indentation for nested lists (2 spaces per depth level)
		sb.WriteString(strings.Repeat("  ", depth))

		// List Marker
		switch listType {
		case "number":
			sb.WriteString(fmt.Sprintf("%d. ", index))
			index++
		case "check":
			// Check Lexical's 'checked' status
			// Note: In Lexical JSON, the 'checked' bool is on the listItem node (child)
			if child.Checked {
				sb.WriteString("- [x] ")
			} else {
				sb.WriteString("- [ ] ")
			}
		case "bullet":
			sb.WriteString("- ")
		default:
			sb.WriteString("- ")
		}

		// List Item Content
		// Recursively walk children of list item
		// Warning: If list item contains a NESTED LIST, it usually appears as a child of the listitem
		for _, grandChild := range child.Children {
			if grandChild.Type == "list" {
				sb.WriteString("\n")
				// Increase depth for nested list
				p.handleList(grandChild, sb, depth+1)
			} else {
				p.walkNode(grandChild, sb, depth)
			}
		}
		sb.WriteString("\n")
	}
	// Extra newline after list
	if depth == 0 {
		sb.WriteString("\n")
	}
}

func (p *Parser) handleTable(node Node, sb *strings.Builder) {
	// 1. Extract grid data
	var rows [][]string
	maxCols := 0

	for _, row := range node.Children {
		if row.Type != "tablerow" {
			continue
		}

		var rowData []string
		for _, cell := range row.Children {
			// Render cell content to string
			var cellSb strings.Builder
			for _, content := range cell.Children {
				p.walkNode(content, &cellSb, 0)
			}
			// Clean newlines in cells as they break MD tables
			cleanContent := strings.ReplaceAll(cellSb.String(), "\n", " ")
			rowData = append(rowData, cleanContent)
		}
		rows = append(rows, rowData)
		if len(rowData) > maxCols {
			maxCols = len(rowData)
		}
	}

	if len(rows) == 0 {
		return
	}

	// 2. Render Markdown Table
	// Header
	sb.WriteString("|")
	for i := 0; i < maxCols; i++ {
		if i < len(rows[0]) {
			sb.WriteString(" " + rows[0][i] + " |")
		} else {
			sb.WriteString("  |")
		}
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("|")
	for i := 0; i < maxCols; i++ {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	// Body (skip row 0 as it's used as header)
	for i := 1; i < len(rows); i++ {
		sb.WriteString("|")
		for j := 0; j < maxCols; j++ {
			if j < len(rows[i]) {
				sb.WriteString(" " + rows[i][j] + " |")
			} else {
				sb.WriteString("  |")
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}
