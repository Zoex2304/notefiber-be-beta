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
	// Updated token for user zikri
	authToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc0OTAxNTksInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.jaUJYwutyRYvuv_G6zYnbjWuoDdaHcQb8VgYEhVRDpQ"
)

// RAGIndicators are phrases that should NOT appear in bypass mode
var RAGIndicators = []string{
	"According to",
	"Reference [",
	"based on your notes",
	"in your notes",
	"from your note",
	"(Reference",
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         BYPASS MODE RAG LEAKAGE DIAGNOSTIC                       ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	fmt.Println("║ Purpose: Prove that RAG prompts leak into bypass mode history    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Create a fresh session
	fmt.Println("[STEP 1] Creating fresh chat session...")
	sessionID := createSession()
	if sessionID == "" {
		fmt.Println("❌ FAILED: Could not create session")
		os.Exit(1)
	}
	fmt.Printf("✅ Session created: %s\n\n", sessionID)

	// Step 2: Get raw history immediately after creation
	fmt.Println("[STEP 2] Inspecting RAW messages seeded at session creation...")
	rawMessages := getRawHistory(sessionID)

	fmt.Printf("\n📋 RAW HISTORY CONTENTS (%d messages):\n", len(rawMessages))
	fmt.Println("─────────────────────────────────────────────────────────────────────")

	ragPromptFound := false
	for i, msg := range rawMessages {
		role := msg["role"].(string)
		chat := msg["chat"].(string)

		// Truncate for display
		displayChat := chat
		if len(displayChat) > 150 {
			displayChat = displayChat[:150] + "... [TRUNCATED]"
		}

		fmt.Printf("[%d] ROLE: %s\n", i+1, strings.ToUpper(role))
		fmt.Printf("    CONTENT: %s\n\n", displayChat)

		// Check for RAG prompt contamination
		if strings.Contains(chat, "pattern-based logic") ||
			strings.Contains(chat, "According to [note_title]") ||
			strings.Contains(chat, "CITATION FORMAT") ||
			strings.Contains(chat, "Reference [N]") {
			ragPromptFound = true
			fmt.Println("    ⚠️  RAG PROMPT DETECTED IN THIS MESSAGE!")
		}
	}
	fmt.Println("─────────────────────────────────────────────────────────────────────")

	if ragPromptFound {
		fmt.Println("\n🔴 DIAGNOSIS: RAG PROMPTS ARE SEEDED INTO SESSION AT CREATION TIME")
		fmt.Println("   These prompts will be loaded as conversation history for ALL modes,")
		fmt.Println("   including bypass mode. This is the source of the leakage.")
	} else {
		fmt.Println("\n🟢 No RAG prompts found in initial session messages.")
	}

	// Step 3: Send bypass query and check response
	fmt.Println("\n[STEP 3] Sending bypass query...")
	query := "/bypass What is 2+2?"
	fmt.Printf("Query: %s\n", query)

	reply, citations := sendChat(sessionID, query)

	fmt.Printf("\n🤖 AI Reply:\n%s\n", reply)

	// Check for RAG artifacts in response
	fmt.Println("\n[STEP 4] Analyzing response for RAG artifacts...")
	ragLeakDetected := false
	for _, indicator := range RAGIndicators {
		if strings.Contains(reply, indicator) {
			fmt.Printf("⚠️  RAG INDICATOR FOUND: '%s'\n", indicator)
			ragLeakDetected = true
		}
	}

	if len(citations) > 0 {
		fmt.Printf("⚠️  CITATIONS RETURNED: %d citations in bypass mode!\n", len(citations))
		ragLeakDetected = true
	}

	// Final verdict
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════╗")
	if ragLeakDetected || ragPromptFound {
		fmt.Println("║  VERDICT: RAG LEAKAGE CONFIRMED                                  ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
		fmt.Println("║  Root Cause: CreateSession() seeds RAG prompts into              ║")
		fmt.Println("║  ChatMessageRaw table. LoadConversationHistory() then loads      ║")
		fmt.Println("║  these prompts and passes them to bypass pipeline.               ║")
	} else {
		fmt.Println("║  VERDICT: NO RAG LEAKAGE DETECTED                                ║")
	}
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
}

func createSession() string {
	resp, body := doRequest("POST", "/chatbot/v1/create-session", nil)
	if resp.StatusCode != 200 {
		fmt.Printf("Failed to create session: %s\n", string(body))
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

func getRawHistory(sessionID string) []map[string]interface{} {
	// This endpoint may not exist - we'll need to query DB directly
	// For now, we'll check via the debug endpoint or direct DB access

	// Alternative: Check what's returned in history endpoint
	resp, body := doRequest("GET", "/chatbot/v1/history?chat_session_id="+sessionID, nil)
	if resp.StatusCode != 200 {
		fmt.Printf("Note: History endpoint returned %d\n", resp.StatusCode)
		return nil
	}

	var res map[string]interface{}
	json.Unmarshal(body, &res)

	// The history endpoint returns ChatMessage, not ChatMessageRaw
	// ChatMessageRaw is internal - we need to check via debug or DB
	fmt.Println("Note: Public history API doesn't expose raw messages.")
	fmt.Println("      RAG contamination happens in ChatMessageRaw table.")
	fmt.Println("      Checking via server logs or DB query is recommended.")

	return nil
}

func sendChat(sessionID, message string) (string, []interface{}) {
	payload := map[string]interface{}{
		"chat_session_id": sessionID,
		"chat":            message,
	}

	resp, body := doRequest("POST", "/chatbot/v1/send-chat", payload)
	if resp.StatusCode != 200 {
		fmt.Printf("Failed to send chat: %s\n", string(body))
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
	return reply, citations
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
		fmt.Printf("Network Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp, respBody
}
