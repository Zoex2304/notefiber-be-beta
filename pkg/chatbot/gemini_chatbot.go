package chatbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"ai-notetaking-be/internal/constant"
)

type GeminiChatParts struct {
	Text string `json:"text"`
}

type GeminiChatContent struct {
	Parts []*GeminiChatParts `json:"parts"`
	Role  string             `json:"role"`
}

type GeminiChatRequest struct {
	Contents []*GeminiChatContent `json:"contents"`
}

type ChatHistory struct {
	Chat string
	Role string
}

type GeminiChatCandidate struct {
	Content *GeminiChatContent `json:"content"`
}

type GeminiChatResponse struct {
	Candidates []*GeminiChatCandidate `json:"candidates"`
}

type GeminiResponseAppSchema struct {
	AnswerDirectly bool `json:"answer_directly"`
}

const (
	ChatMessageRoleUser  = "user"
	ChatMessageRoleModel = "model"
)

func GetGeminiResponse(
	ctx context.Context,
	apiKey string,
	chatHistories []*ChatHistory,
) (string, error) {
	chatContents := make([]*GeminiChatContent, 0)
	for _, chatHistory := range chatHistories {
		chatContents = append(chatContents, &GeminiChatContent{
			Parts: []*GeminiChatParts{
				{
					Text: chatHistory.Chat,
				},
			},
			Role: chatHistory.Role,
		})
	}
	payload := GeminiChatRequest{
		Contents: chatContents,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(
		"POST",
		"https://generativelanguage.googleapis.com/v1/models/gemini-1.5-flash:generateContent",
		bytes.NewBuffer(payloadJson),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"status error, got status %d. with response body %s",
			res.StatusCode,
			string(resBody),
		)
	}

	var geminiRes GeminiChatResponse
	err = json.Unmarshal(resBody, &geminiRes)
	if err != nil {
		return "", err
	}

	return geminiRes.Candidates[0].Content.Parts[0].Text, nil
}

func DecideToUseRAG(
	ctx context.Context,
	apiKey string,
	chatHistories []*ChatHistory,
) (bool, error) {
	chatContents := make([]*GeminiChatContent, 0)

	// Gunakan prompt constant yang sudah ada
	chatContents = append(chatContents, &GeminiChatContent{
		Parts: []*GeminiChatParts{{Text: constant.DecideUseRAGMessageRawInitialUserPromptV1}},
		Role:  ChatMessageRoleUser,
	})
	chatContents = append(chatContents, &GeminiChatContent{
		Parts: []*GeminiChatParts{{Text: constant.DecideUseRAGMessageRawInitialModelPromptV1}},
		Role:  ChatMessageRoleModel,
	})

	// Masukkan chat histories
	for _, chatHistory := range chatHistories {
		chatContents = append(chatContents, &GeminiChatContent{
			Parts: []*GeminiChatParts{{Text: chatHistory.Chat}},
			Role:  chatHistory.Role,
		})
	}

	// Tambahkan instruksi terakhir untuk enforce JSON format
	chatContents = append(chatContents, &GeminiChatContent{
		Parts: []*GeminiChatParts{{
			Text: `Respond with ONLY this JSON format: {"answer_directly": true} or {"answer_directly": false}. No other text.`,
		}},
		Role: ChatMessageRoleUser,
	})

	// Payload TANPA GenerationConfig
	payload := GeminiChatRequest{
		Contents: chatContents,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest(
		"POST",
		"https://generativelanguage.googleapis.com/v1/models/gemini-1.5-flash:generateContent",
		bytes.NewBuffer(payloadJson),
	)
	if err != nil {
		return false, err
	}

	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf(
			"status error, got status %d. with response body %s",
			res.StatusCode,
			string(resBody),
		)
	}

	var geminiRes GeminiChatResponse
	err = json.Unmarshal(resBody, &geminiRes)
	if err != nil {
		return false, err
	}

	// Parse response dan clean dari markdown wrapper
	responseText := geminiRes.Candidates[0].Content.Parts[0].Text
	responseBytes := []byte(responseText)
	responseBytes = bytes.TrimSpace(responseBytes)
	responseBytes = bytes.TrimPrefix(responseBytes, []byte("```json"))
	responseBytes = bytes.TrimPrefix(responseBytes, []byte("```"))
	responseBytes = bytes.TrimSuffix(responseBytes, []byte("```"))
	responseBytes = bytes.TrimSpace(responseBytes)

	var appSchema GeminiResponseAppSchema
	err = json.Unmarshal(responseBytes, &appSchema)
	if err != nil {
		return false, fmt.Errorf("parse error: %w | raw: %s", err, string(responseBytes))
	}

	return !appSchema.AnswerDirectly, nil
}
