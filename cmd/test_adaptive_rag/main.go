package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	apiBaseURL = "http://localhost:3000/api"
	authToken  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc0OTAxNTksInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.jaUJYwutyRYvuv_G6zYnbjWuoDdaHcQb8VgYEhVRDpQ"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ADAPTIVE AMBIGUITY DETECTION TEST                              ║")
	fmt.Println("║   Testing: Dynamic language adaptation & specificity detection              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Test 1: Ambiguous query (should show menu)
	testAmbiguousQuery()

	// Test 2: Specific query (should auto-focus)
	testSpecificQuery()

	// Test 3: Language adaptation
	testLanguageAdaptation()
}

func testAmbiguousQuery() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  TEST 1: AMBIGUOUS QUERY                                                    ║")
	fmt.Println("║  Query: 'English exam' (expected: multiple matches, should show menu)      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		return
	}
	fmt.Printf("Session: %s\n", sessionID)

	query := "Tell me about my English exam notes"
	fmt.Printf("\n📤 USER: %s\n", query)

	reply, citations := sendChat(sessionID, query)
	fmt.Printf("\n🤖 AI: %s\n", reply)

	// Analysis
	fmt.Println("\n📊 ANALYSIS:")
	fmt.Println(strings.Repeat("─", 60))

	// Check for menu-like response (numbered list)
	if strings.Contains(reply, "1.") && strings.Contains(reply, "2.") {
		fmt.Println("   ✅ Menu presented (numbered options)")
	} else {
		fmt.Println("   ⚠️  No numbered menu detected")
	}

	// Check language
	if !strings.Contains(strings.ToLower(reply), "saya") && !strings.Contains(strings.ToLower(reply), "maaf") {
		fmt.Println("   ✅ English response (not hardcoded Indonesian)")
	} else {
		fmt.Println("   ⚠️  Contains Indonesian words")
	}

	if citations > 0 {
		fmt.Printf("   📎 Citations: %d\n", citations)
	}
	fmt.Println(strings.Repeat("─", 60))
}

func testSpecificQuery() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  TEST 2: SPECIFIC QUERY                                                     ║")
	fmt.Println("║  Query: 'English Final Examination' (expected: direct answer)              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ Failed to create session")
		return
	}
	fmt.Printf("Session: %s\n", sessionID)

	query := "Tell me about the English Final Examination"
	fmt.Printf("\n📤 USER: %s\n", query)

	reply, citations := sendChat(sessionID, query)
	fmt.Printf("\n🤖 AI: %s\n", truncate(reply, 300))

	// Analysis
	fmt.Println("\n📊 ANALYSIS:")
	fmt.Println(strings.Repeat("─", 60))

	// Check if it directly answered (not just showing menu)
	if !strings.Contains(reply, "Which one") && !strings.Contains(reply, "focus on") {
		fmt.Println("   ✅ Direct answer (no menu)")
	} else {
		fmt.Println("   ⚠️  Menu shown instead of direct answer")
	}

	if citations > 0 {
		fmt.Printf("   📎 Citations: %d (focused on specific note)\n", citations)
	}
	fmt.Println(strings.Repeat("─", 60))
}

func testLanguageAdaptation() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  TEST 3: LANGUAGE ADAPTATION                                                ║")
	fmt.Println("║  Testing different language queries                                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	tests := []struct {
		Lang  string
		Query string
	}{
		{"English", "Tell me about exams"},
		{"Indonesian", "Ceritakan tentang ujian"},
	}

	for _, test := range tests {
		sessionID := createSession()
		if sessionID == "" {
			continue
		}

		fmt.Printf("\n[%s] Query: %s\n", test.Lang, test.Query)
		reply, _ := sendChat(sessionID, test.Query)
		fmt.Printf("AI: %s\n", truncate(reply, 150))

		// Basic language detection
		hasIndonesian := strings.Contains(strings.ToLower(reply), "saya") ||
			strings.Contains(strings.ToLower(reply), "catatan") ||
			strings.Contains(strings.ToLower(reply), "yang")

		if test.Lang == "Indonesian" && hasIndonesian {
			fmt.Println("   ✅ Responded in Indonesian")
		} else if test.Lang == "English" && !hasIndonesian {
			fmt.Println("   ✅ Responded in English")
		} else {
			fmt.Printf("   ℹ️  Language may be mixed or adaptive\n")
		}
	}

	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("✅ ADAPTIVE AMBIGUITY TEST COMPLETE")
}

func createSession() string {
	resp, body := doRequest("POST", "/chatbot/v1/create-session", nil)
	if resp.StatusCode != 200 {
		return ""
	}

	var res map[string]interface{}
	json.Unmarshal(body, &res)

	if data, ok := res["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok {
			return id
		}
	}
	return ""
}

func sendChat(sessionID, message string) (string, int) {
	payload := map[string]interface{}{
		"chat_session_id": sessionID,
		"chat":            message,
	}

	resp, body := doRequest("POST", "/chatbot/v1/send-chat", payload)
	if resp.StatusCode != 200 {
		return fmt.Sprintf("[ERROR: %s]", string(body)), 0
	}

	var res map[string]interface{}
	json.Unmarshal(body, &res)

	reply := ""
	citations := 0

	if data, ok := res["data"].(map[string]interface{}); ok {
		if replyObj, ok := data["reply"].(map[string]interface{}); ok {
			if content, ok := replyObj["chat"].(string); ok {
				reply = content
			}
			if cits, ok := replyObj["citations"].([]interface{}); ok {
				citations = len(cits)
			}
		}
	}
	return reply, citations
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func doRequest(method, url string, body interface{}) (*http.Response, []byte) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, _ := http.NewRequest(method, apiBaseURL+url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Network Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp, respBody
}
