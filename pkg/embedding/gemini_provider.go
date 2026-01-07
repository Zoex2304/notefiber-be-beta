package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GeminiProvider struct {
	ApiKey string
}

func NewGeminiProvider(apiKey string) EmbeddingProvider {
	return &GeminiProvider{
		ApiKey: apiKey,
	}
}

func (p *GeminiProvider) Generate(text string, taskType string) (*EmbeddingResponse, error) {
	// Gemini Text-Embedding-004 logic (Ported from original GetGeminiEmbedding)
	modelName := "text-embedding-004"

	geminiReq := EmbeddingRequest{
		Model: modelName,
		Content: EmbeddingRequestContent{
			Parts: []EmbeddingRequestContentPart{
				{
					Text: text,
				},
			},
		},
		TaskType: taskType,
	}
	geminiReqJson, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1/models/%s:embedContent",
		modelName,
	)

	req, err := http.NewRequest(
		"POST",
		endpoint,
		bytes.NewBuffer(geminiReqJson),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-goog-api-key", p.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resByte, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from gemini response, code %d, body %s", res.StatusCode, string(resByte))
	}

	var resEmbedding EmbeddingResponse
	err = json.Unmarshal(resByte, &resEmbedding)
	if err != nil {
		return nil, err
	}

	return &resEmbedding, nil
}
