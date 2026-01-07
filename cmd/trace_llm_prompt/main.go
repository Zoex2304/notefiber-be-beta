package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	apiBaseURL = "http://localhost:3000/api"
	authToken  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc0OTAxNTksInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.jaUJYwutyRYvuv_G6zYnbjWuoDdaHcQb8VgYEhVRDpQ"
)

// RAG prompt fingerprints - should NEVER appear in bypass mode
var RAGFingerprints = []string{
	"pattern-based logic",
	"According to [note_title]",
	"CITATION FORMAT",
	"Reference [N]",
	"personal notes",
	"NOTES DATABASE",
	"knowledge assistant",
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    LLM PROMPT TRACE TEST SCRIPT                              ║")
	fmt.Println("║   Purpose: Observe exact prompts sent to LLM in BYPASS vs RAG mode          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// ==========================
	// TEST 1: BYPASS MODE
	// ==========================
	fmt.Println("┌──────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  TEST 1: BYPASS MODE - Fresh Session                                        │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────────────┘")

	bypassSessionID := createSession()
	fmt.Printf("✅ Bypass Session Created: %s\n\n", bypassSessionID)

	// Send bypass query
	bypassQuery := "/bypass What is the capital of France?"
	fmt.Printf("📤 Sending: %s\n\n", bypassQuery)

	bypassReply, bypassCitations := sendChatWithTrace(bypassSessionID, bypassQuery, "BYPASS")

	analyzeResponse("BYPASS", bypassReply, bypassCitations)

	// ==========================
	// TEST 2: RAG MODE
	// ==========================
	fmt.Println("\n┌──────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  TEST 2: RAG MODE - Fresh Session (English Exam Notes)                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────────────┘")

	ragSessionID := createSession()
	fmt.Printf("✅ RAG Session Created: %s\n\n", ragSessionID)

	// Send RAG query about English exam
	ragQuery := "What do my notes say about the English exam?"
	fmt.Printf("📤 Sending: %s\n\n", ragQuery)

	ragReply, ragCitations := sendChatWithTrace(ragSessionID, ragQuery, "RAG")

	analyzeResponse("RAG", ragReply, ragCitations)

	// ==========================
	// COMPARISON SUMMARY
	// ==========================
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                          COMPARISON SUMMARY                                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")

	fmt.Println("\n📊 BYPASS MODE:")
	fmt.Printf("   - Citations: %d\n", len(bypassCitations))
	fmt.Printf("   - Has RAG artifacts: %v\n", hasRAGArtifacts(bypassReply))

	fmt.Println("\n📊 RAG MODE:")
	fmt.Printf("   - Citations: %d\n", len(ragCitations))
	fmt.Printf("   - Has RAG artifacts: %v\n", hasRAGArtifacts(ragReply))

	fmt.Println("\n" + strings.Repeat("─", 80))
	if len(bypassCitations) == 0 && !hasRAGArtifacts(bypassReply) {
		fmt.Println("✅ BYPASS MODE: Working correctly (no RAG artifacts in response)")
	} else {
		fmt.Println("❌ BYPASS MODE: RAG LEAKAGE DETECTED")
		fmt.Println("   The response contains RAG-specific patterns that should not be present.")
		fmt.Println("   This confirms the issue: RAG prompts are contaminating bypass mode.")
	}
}

func createSession() string {
	resp, body := doRequest("POST", "/chatbot/v1/create-session", nil)
	if resp.StatusCode != 200 {
		fmt.Printf("❌ Failed to create session: %s\n", string(body))
		os.Exit(1)
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

func sendChatWithTrace(sessionID, message, mode string) (string, []interface{}) {
	payload := map[string]interface{}{
		"chat_session_id": sessionID,
		"chat":            message,
	}

	fmt.Println("┌─── REQUEST TO BACKEND ─────────────────────────────────────────────────────")
	fmt.Printf("│ Mode: %s\n", mode)
	fmt.Printf("│ Session: %s\n", sessionID)
	fmt.Printf("│ Message: %s\n", message)
	fmt.Println("└────────────────────────────────────────────────────────────────────────────")

	resp, body := doRequest("POST", "/chatbot/v1/send-chat", payload)
	if resp.StatusCode != 200 {
		fmt.Printf("❌ Failed to send chat: %s\n", string(body))
		return "", nil
	}

	var res map[string]interface{}
	json.Unmarshal(body, &res)

	reply := ""
	var citations []interface{}

	if data, ok := res["data"].(map[string]interface{}); ok {
		if replyObj, ok := data["reply"].(map[string]interface{}); ok {
			if content, ok := replyObj["chat"].(string); ok {
				reply = content
			}
			if c, ok := replyObj["citations"].([]interface{}); ok {
				citations = c
			}
		}
	}

	fmt.Println("\n┌─── RESPONSE FROM BACKEND ──────────────────────────────────────────────────")
	fmt.Printf("│ Reply Length: %d characters\n", len(reply))
	fmt.Printf("│ Citations Count: %d\n", len(citations))
	fmt.Println("│")
	fmt.Println("│ 📨 REPLY CONTENT:")
	printBoxed(reply, "│   ")
	fmt.Println("└────────────────────────────────────────────────────────────────────────────")

	return reply, citations
}

func analyzeResponse(mode, reply string, citations []interface{}) {
	fmt.Printf("\n🔍 ANALYSIS FOR %s MODE:\n", mode)
	fmt.Println(strings.Repeat("─", 40))

	// Check for RAG fingerprints
	ragFound := []string{}
	for _, fp := range RAGFingerprints {
		if strings.Contains(strings.ToLower(reply), strings.ToLower(fp)) {
			ragFound = append(ragFound, fp)
		}
	}

	if len(ragFound) > 0 {
		fmt.Println("⚠️  RAG Fingerprints FOUND in response:")
		for _, f := range ragFound {
			fmt.Printf("   - \"%s\"\n", f)
		}
	} else {
		if mode == "BYPASS" {
			fmt.Println("✅ No RAG fingerprints in response (expected for bypass)")
		} else {
			fmt.Println("ℹ️  No explicit RAG fingerprints (LLM may have paraphrased)")
		}
	}

	if len(citations) > 0 {
		fmt.Printf("📎 Citations returned: %d\n", len(citations))
		for i, c := range citations {
			if cMap, ok := c.(map[string]interface{}); ok {
				fmt.Printf("   [%d] %v\n", i+1, cMap["title"])
			}
		}
		if mode == "BYPASS" {
			fmt.Println("   ⚠️  BYPASS mode should NOT have citations!")
		}
	} else {
		if mode == "BYPASS" {
			fmt.Println("✅ No citations (correct for bypass mode)")
		} else {
			fmt.Println("⚠️  No citations returned (RAG might not have found relevant notes)")
		}
	}
}

func hasRAGArtifacts(reply string) bool {
	lowerReply := strings.ToLower(reply)
	ragPatterns := []string{
		"according to your notes",
		"based on your notes",
		"in your notes",
		"reference [",
		"(reference",
		"from my notes",
		"according to my notes",
	}
	for _, p := range ragPatterns {
		if strings.Contains(lowerReply, p) {
			return true
		}
	}
	return false
}

func printBoxed(text string, prefix string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		// Wrap long lines
		for len(line) > 70 {
			fmt.Printf("%s%s\n", prefix, line[:70])
			line = line[70:]
		}
		fmt.Printf("%s%s\n", prefix, line)
	}
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Network Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp, respBody
}
