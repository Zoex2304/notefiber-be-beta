package main

import (
	"fmt"
	"log"
	"strings"

	"ai-notetaking-be/pkg/lexical"
)

func main() {
	// this is the content string from the user's sample
	// heavily escaped as it was inside a JSON response
	rawLexicalJSON := `{
    "root": {
        "children": [
            {
                "children": [
                    {
                        "detail": 0,
                        "format": 1,
                        "mode": "normal",
                        "style": "",
                        "text": "asdasdasdasd",
                        "type": "text",
                        "version": 1
                    }
                ],
                "direction": null,
                "format": "",
                "indent": 0,
                "type": "paragraph",
                "version": 1,
                "textFormat": 1,
                "textStyle": ""
            },
            {
                "children": [
                    {
                        "detail": 0,
                        "format": 66,
                        "mode": "normal",
                        "style": "color: #F97316;",
                        "text": "adasdsdadaads",
                        "type": "text",
                        "version": 1
                    }
                ],
                "direction": null,
                "format": "",
                "indent": 0,
                "type": "paragraph",
                "version": 1,
                "textFormat": 66,
                "textStyle": "color: #F97316;"
            },
            {
                "children": [
                    {
                        "children": [
                            {
                                "children": [
                                    {
                                        "children": [
                                            {
                                                "detail": 0,
                                                "format": 0,
                                                "mode": "normal",
                                                "style": "",
                                                "text": "Cell 1",
                                                "type": "text",
                                                "version": 1
                                            }
                                        ],
                                        "direction": null,
                                        "format": "",
                                        "indent": 0,
                                        "type": "paragraph",
                                        "version": 1,
                                        "textFormat": 0,
                                        "textStyle": ""
                                    }
                                ],
                                "direction": null,
                                "format": "",
                                "indent": 0,
                                "type": "tablecell",
                                "version": 1,
                                "backgroundColor": null,
                                "colSpan": 1,
                                "headerState": 1,
                                "rowSpan": 1
                            },
                            {
                                "children": [
                                    {
                                        "children": [
                                            {
                                                "detail": 0,
                                                "format": 0,
                                                "mode": "normal",
                                                "style": "",
                                                "text": "Cell 2",
                                                "type": "text",
                                                "version": 1
                                            }
                                        ],
                                        "direction": null,
                                        "format": "",
                                        "indent": 0,
                                        "type": "paragraph",
                                        "version": 1,
                                        "textFormat": 0,
                                        "textStyle": ""
                                    }
                                ],
                                "direction": null,
                                "format": "",
                                "indent": 0,
                                "type": "tablecell",
                                "version": 1,
                                "backgroundColor": null,
                                "colSpan": 1,
                                "headerState": 1,
                                "rowSpan": 1
                            }
                        ],
                        "direction": null,
                        "format": "",
                        "indent": 0,
                        "type": "tablerow",
                        "version": 1
                    },
                    {
                         "children": [
                            {
                                "children": [
                                    {
                                        "children": [
                                            {
                                                "detail": 0,
                                                "format": 0,
                                                "mode": "normal",
                                                "style": "",
                                                "text": "Row 2 Col 1",
                                                "type": "text",
                                                "version": 1
                                            }
                                        ],
                                        "direction": null,
                                        "format": "",
                                        "indent": 0,
                                        "type": "paragraph",
                                        "version": 1,
                                        "textFormat": 0,
                                        "textStyle": ""
                                    }
                                ],
                                "direction": null,
                                "format": "",
                                "indent": 0,
                                "type": "tablecell",
                                "version": 1,
                                "backgroundColor": null,
                                "colSpan": 1,
                                "headerState": 0,
                                "rowSpan": 1
                            },
                            {
                                "children": [
                                    {
                                        "children": [
                                            {
                                                "detail": 0,
                                                "format": 0,
                                                "mode": "normal",
                                                "style": "",
                                                "text": "Row 2 Col 2",
                                                "type": "text",
                                                "version": 1
                                            }
                                        ],
                                        "direction": null,
                                        "format": "",
                                        "indent": 0,
                                        "type": "paragraph",
                                        "version": 1,
                                        "textFormat": 0,
                                        "textStyle": ""
                                    }
                                ],
                                "direction": null,
                                "format": "",
                                "indent": 0,
                                "type": "tablecell",
                                "version": 1,
                                "backgroundColor": null,
                                "colSpan": 1,
                                "headerState": 0,
                                "rowSpan": 1
                            }
                         ],
                        "direction": null,
                        "format": "",
                        "indent": 0,
                        "type": "tablerow",
                        "version": 1
                    }
                ],
                "direction": null,
                "format": "",
                "indent": 0,
                "type": "table",
                "version": 1
            },
            {
                "children": [
                    {
                        "children": [
                            {
                                "detail": 0,
                                "format": 0,
                                "mode": "normal",
                                "style": "",
                                "text": "Unchecked Item",
                                "type": "text",
                                "version": 1
                            }
                        ],
                        "direction": null,
                        "format": "",
                        "indent": 0,
                        "type": "listitem",
                        "version": 1,
                        "textStyle": "",
                        "checked": false,
                        "value": 1
                    },
                    {
                        "children": [
                            {
                                "detail": 0,
                                "format": 0,
                                "mode": "normal",
                                "style": "",
                                "text": "Checked Item",
                                "type": "text",
                                "version": 1
                            }
                        ],
                        "direction": null,
                        "format": "",
                        "indent": 0,
                        "type": "listitem",
                        "version": 1,
                        "textStyle": "",
                        "checked": true,
                        "value": 2
                    }
                ],
                "direction": null,
                "format": "",
                "indent": 0,
                "type": "list",
                "version": 1,
                "textStyle": "",
                "listType": "check",
                "start": 1,
                "tag": "ul"
            }
        ],
        "direction": null,
        "format": "",
        "indent": 0,
        "type": "root",
        "version": 1,
        "textStyle": ""
    }
}`

	fmt.Println("parsing...")
	parser := lexical.NewParser()
	md, err := parser.Parse(rawLexicalJSON)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Println("--- OUTPUT MARKDOWN START ---")
	fmt.Println(md)
	fmt.Println("--- OUTPUT MARKDOWN END ---")

	// Validations
	if strings.Contains(md, "**asdasdasdasd**") {
		fmt.Println("✅ Bold text detected")
	} else {
		fmt.Println("❌ Bold text MISSING (format: 1)")
	}

	if strings.Contains(md, "color: #F97316") {
		fmt.Println("✅ Color annotation detected")
	} else {
		fmt.Println("❌ Color annotation MISSING")
	}

	if strings.Contains(md, "| Cell 1 |") && strings.Contains(md, "|---|") {
		fmt.Println("✅ Table detected")
	} else {
		fmt.Println("❌ Table structure MISSING")
	}

	if strings.Contains(md, "- [ ] Unchecked Item") {
		fmt.Println("✅ Unchecked item detected")
	} else {
		fmt.Println("❌ Unchecked item MISSING")
	}

	if strings.Contains(md, "- [x] Checked Item") {
		fmt.Println("✅ Checked item detected")
	} else {
		fmt.Println("❌ Checked item MISSING")
	}
}
