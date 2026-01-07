package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/fatih/color"
)

const (
	baseURL    = "http://localhost:3000/api"
	userToken  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc0MjQzMTYsInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6IjY2YTMyMDE1LTQzYjctNGYzMC1hNGM5LTZmNGM3NGEwZDNjMyJ9.lZCHNAJ-CGFiKVdw9SzQoEr9Hk3IZjbiLwbUVJnlpQg"
	adminToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc0MjExMDUsInJvbGUiOiJhZG1pbiIsInVzZXJfaWQiOiJmNmMwYzM1Yi0zYTQyLTRkYTktODgyZi0yNTM0MmZhNmZlNGMifQ.Pc2njNI0Tv4qhwshBdPwxM6dZx_5B2voB4FKGIIgUDg"
)

// Pretty print JSON helper
func prettyPrint(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("%v\n", v)
		return
	}
	fmt.Println(string(b))
}

// Request helper
func sendRequest(method, url, token string, body interface{}) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+url, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// client := &http.Client{Timeout: 10 * time.Second}
	client := &http.Client{} // No timeout
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	return resp, respBody, err
}

func main() {
	color.Cyan("🚀 Starting AI Pipeline & Nuance API Test\n")

	// 1. Test Admin: Get All Configurations
	color.Yellow("\n[ADMIN] 1. Get All AI Configurations")
	resp, body, err := sendRequest("GET", "/admin/ai/configurations", adminToken, nil)
	if err != nil {
		color.Red("Failed: %v", err)
		os.Exit(1)
	}
	color.Green("Status: %s", resp.Status)
	var configResp map[string]interface{}
	json.Unmarshal(body, &configResp)
	prettyPrint(configResp)

	// 2. Test Admin: Create New Nuance
	color.Yellow("\n[ADMIN] 2. Create New 'Tester' Nuance")
	nuanceReq := map[string]interface{}{
		"key":            "tester_mode",
		"name":           "Tester Mode",
		"description":    "A mode for testing API flow",
		"system_prompt":  "You are a test bot.",
		"model_override": "qwen2.5",
		"sort_order":     10,
	}
	resp, body, err = sendRequest("POST", "/admin/ai/nuances", adminToken, nuanceReq)
	if err != nil {
		color.Red("Failed: %v", err)
		os.Exit(1)
	}
	color.Green("Status: %s", resp.Status)
	var createNuanceResp map[string]interface{}
	json.Unmarshal(body, &createNuanceResp)
	prettyPrint(createNuanceResp)

	// Extract new ID if created
	var nuanceID string
	if data, ok := createNuanceResp["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok {
			nuanceID = id
		}
	}

	// 3. Test User: Get Available Nuances (Public Endpoint)
	color.Yellow("\n[USER] 3. Get Available Nuances (Public Endpoint)")
	resp, body, err = sendRequest("GET", "/chatbot/v1/nuances", userToken, nil)
	if err != nil {
		color.Red("Failed: %v", err)
		os.Exit(1)
	}
	color.Green("Status: %s", resp.Status)
	var publicNuancesResp map[string]interface{}
	json.Unmarshal(body, &publicNuancesResp)
	prettyPrint(publicNuancesResp)

	// 3a. Test User: Create Chat Session (Required for RAG)
	color.Yellow("\n[USER] 3a. Create Chat Session")
	resp, body, err = sendRequest("POST", "/chatbot/v1/create-session", userToken, nil)
	var sessionID string
	if err != nil {
		color.Red("Failed: %v", err)
		os.Exit(1)
	}
	color.Green("Status: %s", resp.Status)
	var createSessResp map[string]interface{}
	json.Unmarshal(body, &createSessResp)
	if data, ok := createSessResp["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok {
			sessionID = id
			fmt.Printf("Created Session ID: %s\n", sessionID)
		}
	}

	// 4. Test RAG: Default Chat (English Exam)
	color.Yellow("\n[USER] 4. Test RAG: 'English Exam' (Default Mode)")
	if sessionID == "" {
		color.Red("Skipping RAG test: Failed to create session")
	} else {
		ragReq := map[string]interface{}{
			"chat_session_id": sessionID,
			"chat":            "What are the key topics for my English exam?",
		}
		resp, body, err = sendRequest("POST", "/chatbot/v1/send-chat", userToken, ragReq)
		if err != nil {
			color.Red("Failed: %v", err)
		} else {
			color.Green("Status: %s", resp.Status)
			var ragResp map[string]interface{}
			json.Unmarshal(body, &ragResp)
			// Concise printing for chat response to avoid huge citation dump
			if data, ok := ragResp["data"].(map[string]interface{}); ok {
				fmt.Printf("Reply: %s\n", data["reply"])
				if citations, ok := data["citations"].([]interface{}); ok {
					fmt.Printf("Citations: %d\n", len(citations))
				}
			} else {
				prettyPrint(ragResp)
			}
		}
	}

	// 5. Test RAG: Chat about English exam without prefixes
	color.Yellow("\n[USER] 5. Test RAG: 'English Exam' (Without Prefixes)")
	if sessionID == "" {
		color.Red("Skipping test: Session ID missing")
	} else {
		ragReq := map[string]interface{}{
			"chat_session_id": sessionID,
			"chat":            "Tell me about the key topics for my English exam.",
		}
		resp, body, err = sendRequest("POST", "/chatbot/v1/send-chat", userToken, ragReq)
		if err != nil {
			color.Red("Failed: %v", err)
		} else {
			color.Green("Status: %s", resp.Status)
			var ragResp map[string]interface{}
			json.Unmarshal(body, &ragResp)
			if data, ok := ragResp["data"].(map[string]interface{}); ok {
				fmt.Printf("Reply: %s\n", data["reply"])
				if citations, ok := data["citations"].([]interface{}); ok {
					fmt.Printf("Citations: %d\n", len(citations))
				}
			} else {
				prettyPrint(ragResp)
			}
		}
	}

	// 6. Test Cleanup (Delete created nuance)
	if nuanceID != "" {
		color.Yellow("\n[ADMIN] 6. Cleanup: Delete 'Tester' Nuance")
		resp, body, err = sendRequest("DELETE", "/admin/ai/nuances/"+nuanceID, adminToken, nil)
		if err != nil {
			color.Red("Failed: %v", err)
		} else {
			color.Green("Status: %s", resp.Status)
			var deleteResp map[string]interface{}
			json.Unmarshal(body, &deleteResp)
			prettyPrint(deleteResp)
		}
	} else {
		color.Red("\n[SKIP] Cleanup skipped (no ID returned from create)")
	}

	color.Cyan("\n✅ Test Sequence Complete")
}
