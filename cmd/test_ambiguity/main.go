package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Configuration holds application configuration
type Configuration struct {
	APIBaseURL string
	AuthToken  string
	Timeout    time.Duration
}

// HTTPClient interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ChatClient handles communication with chat API
type ChatClient struct {
	config Configuration
	client HTTPClient
}

// NewChatClient creates a new chat client instance
func NewChatClient(config Configuration) *ChatClient {
	return &ChatClient{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

// CreateSession creates a new chat session
func (c *ChatClient) CreateSession() (string, error) {
	resp, body, err := c.doRequest("POST", "/chatbot/v1/create-session", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create session: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return c.extractSessionID(result)
}

// SendMessage sends a message to the chat API
func (c *ChatClient) SendMessage(sessionID, message string) (*ChatResponse, error) {
	payload := map[string]interface{}{
		"chat_session_id": sessionID,
		"chat":            message,
	}

	resp, body, err := c.doRequest("POST", "/chatbot/v1/send-chat", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chat error: %s", string(body))
	}

	return c.parseChatResponse(body)
}

func (c *ChatClient) doRequest(method, endpoint string, body interface{}) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, c.config.APIBaseURL+endpoint, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.AuthToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}

	return resp, respBody, nil
}

func (c *ChatClient) extractSessionID(result map[string]interface{}) (string, error) {
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	id, ok := data["id"].(string)
	if !ok {
		return "", fmt.Errorf("session ID not found")
	}

	return id, nil
}

func (c *ChatClient) parseChatResponse(body []byte) (*ChatResponse, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	replyObj, ok := data["reply"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("reply not found")
	}

	response := &ChatResponse{}
	
	if content, ok := replyObj["chat"].(string); ok {
		response.Content = content
	}

	if citations, ok := replyObj["citations"].([]interface{}); ok {
		response.CitationCount = len(citations)
	}

	return response, nil
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	Content       string
	CitationCount int
	Duration      time.Duration
}

// Session represents an active chat session
type Session struct {
	ID        string
	client    *ChatClient
	startTime time.Time
}

// NewSession creates a new session instance
func NewSession(client *ChatClient) (*Session, error) {
	sessionID, err := client.CreateSession()
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        sessionID,
		client:    client,
		startTime: time.Now(),
	}, nil
}

// SendMessage sends a message within this session
func (s *Session) SendMessage(message string) (*ChatResponse, error) {
	start := time.Now()
	response, err := s.client.SendMessage(s.ID, message)
	if err != nil {
		return nil, err
	}

	response.Duration = time.Since(start)
	return response, nil
}

// ChatApplication orchestrates the interactive chat experience
type ChatApplication struct {
	session *Session
	output  OutputFormatter
	input   InputReader
}

// OutputFormatter handles output formatting
type OutputFormatter struct{}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter() *OutputFormatter {
	return &OutputFormatter{}
}

func (o *OutputFormatter) PrintBanner() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    INTERACTIVE CHAT CLIENT                                   ║")
	fmt.Println("║               Type your messages and press Enter                             ║")
	fmt.Println("║               Press Ctrl+C to exit                                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func (o *OutputFormatter) PrintSessionCreated(sessionID string) {
	fmt.Printf("✅ Session Created: %s\n\n", sessionID)
}

func (o *OutputFormatter) PrintResponse(response *ChatResponse) {
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("🤖 AI (%v):\n%s\n", response.Duration, response.Content)
	
	if response.CitationCount > 0 {
		fmt.Printf("\n📎 Citations: %d\n", response.CitationCount)
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
}

func (o *OutputFormatter) PrintError(err error) {
	fmt.Printf("❌ Error: %v\n\n", err)
}

func (o *OutputFormatter) PrintPrompt() {
	fmt.Print("💬 You: ")
}

// InputReader handles user input
type InputReader struct {
	scanner *bufio.Scanner
}

// NewInputReader creates a new input reader
func NewInputReader() *InputReader {
	return &InputReader{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (i *InputReader) ReadLine() (string, error) {
	if i.scanner.Scan() {
		return strings.TrimSpace(i.scanner.Text()), nil
	}
	
	if err := i.scanner.Err(); err != nil {
		return "", err
	}
	
	return "", io.EOF
}

// NewChatApplication creates a new chat application instance
func NewChatApplication(session *Session) *ChatApplication {
	return &ChatApplication{
		session: session,
		output:  *NewOutputFormatter(),
		input:   *NewInputReader(),
	}
}

// Run starts the interactive chat loop
func (app *ChatApplication) Run() error {
	app.output.PrintBanner()
	app.output.PrintSessionCreated(app.session.ID)

	for {
		app.output.PrintPrompt()
		
		message, err := app.input.ReadLine()
		if err != nil {
			if err == io.EOF {
				fmt.Println("\n👋 Goodbye!")
				return nil
			}
			return err
		}

		if message == "" {
			continue
		}

		response, err := app.session.SendMessage(message)
		if err != nil {
			app.output.PrintError(err)
			continue
		}

		app.output.PrintResponse(response)
	}
}

func main() {
	config := Configuration{
		APIBaseURL: "http://localhost:3000/api",
		AuthToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Njc3MTExNDUsInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.1cg8aPo9XSLgtePD4ayrXkqt6xbToLJrabZKnoK1Res",
		Timeout:    120 * time.Second,
	}

	client := NewChatClient(config)
	
	session, err := NewSession(client)
	if err != nil {
		fmt.Printf("❌ Failed to create session: %v\n", err)
		os.Exit(1)
	}

	app := NewChatApplication(session)
	
	if err := app.Run(); err != nil {
		fmt.Printf("❌ Application error: %v\n", err)
		os.Exit(1)
	}
}