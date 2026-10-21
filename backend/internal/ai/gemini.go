package ai

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

//go:embed schemas/*.json
var schemaFS embed.FS

// Gemini calls the Google Gemini API in JSON mode with a response schema.
type Gemini struct {
	APIKey  string
	Model   string
	BaseURL string // default https://generativelanguage.googleapis.com
	HTTP    *http.Client
}

type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig struct {
		ResponseMimeType   string          `json:"responseMimeType"`
		ResponseJSONSchema json.RawMessage `json:"responseJsonSchema"`
		Temperature        float64         `json:"temperature"`
	} `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (g Gemini) generate(ctx context.Context, kind, prompt string, out any) error {
	schema, err := schemaFS.ReadFile("schemas/" + kind + ".json")
	if err != nil {
		return err
	}
	var req geminiRequest
	req.Contents = []geminiContent{{Role: "user", Parts: []geminiPart{{Text: prompt}}}}
	req.GenerationConfig.ResponseMimeType = "application/json"
	req.GenerationConfig.ResponseJSONSchema = schema
	req.GenerationConfig.Temperature = 0.4
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	base := g.BaseURL
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", strings.TrimRight(base, "/"), g.Model)
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("x-goog-api-key", g.APIKey)
	client := g.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(hreq)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("gemini: HTTP %d", res.StatusCode)
	}
	var gr geminiResponse
	if err := json.Unmarshal(raw, &gr); err != nil {
		return fmt.Errorf("gemini: bad envelope: %w", err)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("gemini: empty response")
	}
	return decodeStrict(gr.Candidates[0].Content.Parts[0].Text, out)
}

// decodeStrict parses model JSON, rejecting unknown fields.
func decodeStrict(text string, out any) error {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return &ValidationError{Problems: []string{"the answer was not valid JSON for the schema: " + err.Error()}}
	}
	return nil
}

// GenerateRolePlan implements Provider.
func (g Gemini) GenerateRolePlan(ctx context.Context, in RolePlanInput) (RolePlanOutput, error) {
	var out RolePlanOutput
	p, err := renderRolePlan(in)
	if err == nil {
		err = g.generate(ctx, KindRolePlan, p, &out)
	}
	return out, err
}

// MatchRoles implements Provider.
func (g Gemini) MatchRoles(ctx context.Context, in MatchInput) (MatchOutput, error) {
	var out MatchOutput
	p, err := renderMatching(in)
	if err == nil {
		err = g.generate(ctx, KindMatching, p, &out)
	}
	return out, err
}

// DraftRACI implements Provider.
func (g Gemini) DraftRACI(ctx context.Context, in RACIInput) (RACIOutput, error) {
	var out RACIOutput
	p, err := renderRACI(in)
	if err == nil {
		err = g.generate(ctx, KindRACI, p, &out)
	}
	return out, err
}
