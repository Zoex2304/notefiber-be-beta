package lexical

import (
	"strings"
)

// StyleMap represents parsed CSS styles
type StyleMap map[string]string

// ParseStyle parses a CSS style string into a map
// Example: "color: #F97316; background-color: #BFDBFE;"
func ParseStyle(styleStr string) StyleMap {
	styles := make(StyleMap)
	if styleStr == "" {
		return styles
	}

	parts := strings.Split(styleStr, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])
			if k != "" && v != "" {
				styles[k] = v
			}
		}
	}
	return styles
}

// BuildStyleTag creates an HTML span with specific meaningful styles
// Returns empty string if no relevant styles found
func (s StyleMap) BuildAnnotatedOpenTag() string {
	var relevant []string

	// Whitelist of styles to preserve for LLM context
	whitelist := []string{"color", "background-color", "text-transform"}

	for _, k := range whitelist {
		if v, ok := s[k]; ok {
			// Skip empty or default values if needed, though usually not present in Lexical output unless set
			relevant = append(relevant, k+":"+v)
		}
	}

	if len(relevant) == 0 {
		return ""
	}

	return "<span style=\"" + strings.Join(relevant, "; ") + "\">"
}
