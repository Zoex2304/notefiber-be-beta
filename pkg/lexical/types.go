package lexical

// LexicalRoot represents the top-level structure
type LexicalRoot struct {
	Root Node `json:"root"`
}

// Node represents any node in the Lexical tree
// Using pointers for nullable fields to save space/detect absence
type Node struct {
	Type     string `json:"type"`
	Version  int    `json:"version"`
	Children []Node `json:"children,omitempty"`

	// Text specific
	Text   string      `json:"text,omitempty"`
	Format interface{} `json:"format,omitempty"` // Can be int (bitmask) or string (alignment)
	Style  string      `json:"style,omitempty"`
	Mode   string      `json:"mode,omitempty"`
	Detail int         `json:"detail,omitempty"`

	// Paragraph specific
	Direction  string `json:"direction,omitempty"`
	Indent     int    `json:"indent,omitempty"`
	TextFormat int    `json:"textFormat,omitempty"`

	// Link specific
	URL    string `json:"url,omitempty"`
	Rel    string `json:"rel,omitempty"`
	Target string `json:"target,omitempty"`
	Title  string `json:"title,omitempty"`

	// List specific
	ListType string `json:"listType,omitempty"` // check, bullet, number
	Start    int    `json:"start,omitempty"`
	Tag      string `json:"tag,omitempty"`

	// ListItem specific
	Checked bool `json:"checked,omitempty"`
	Value   int  `json:"value,omitempty"`

	// Table specific
	ColSpan     int `json:"colSpan,omitempty"`
	RowSpan     int `json:"rowSpan,omitempty"`
	HeaderState int `json:"headerState,omitempty"` // 1 = header, 0 = normal
}

// Constants for Text Format Bitmask
const (
	FormatBold          = 1
	FormatItalic        = 2
	FormatStrikethrough = 4
	FormatUnderline     = 8
	FormatCode          = 16
	FormatSubscript     = 32
	FormatSuperscript   = 64
	FormatHighlight     = 1 << 7 // ? verify
)
