package extras

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	tg "github.com/amarnathcjd/gogram/telegram"
	"html"
	"io"
	modules "main/modules"
	mrand "math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// === from aigen.go ===
// === from aigen2.go ===
var tokens = []string{
	"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0IjoiYXUiLCJ2IjoiMC4wLjAiLCJ1dSI6IlVMRDNYay9NVEltSXpJWDk5WDF3cWc9PSIsImF1IjoiaWRnL2ZEMDdVTkdhSk5sNXpXUGZhUT09IiwicyI6IjNVenRpbkZvcUQ2Y3hjQmJvUjczT3c9PSIsImlhdCI6MTc2Njg1MzA5NX0.kUCmCBkORnowOZRnJP_nxqVVLKrrhy0mfr3tzfWdExU",
	"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0IjoiYXUiLCJ2IjoiMC4wLjAiLCJ1dSI6IkZtWVdxRnROU3NtaTd2SDdLV3FHQ3c9PSIsImF1IjoiaWRnL2ZEMDdVTkdhSk5sNXpXUGZhUT09IiwicyI6ImYzODVJVU83Z1RYM2lDbHVTQURXWWc9PSIsImlhdCI6MTc2Njg1MzE4NX0.WIuvXnY0mpX2OxYYCUIXyQLiVDynLzhyYhnZAsR3TiM",
}

type PuterClient struct {
	baseURL   string
	http      *http.Client
	authToken string
}

func NewPuterClient(authToken string) *PuterClient {
	return &PuterClient{
		baseURL:   "https://api.puter.com/drivers/call",
		http:      &http.Client{Timeout: 60 * time.Second},
		authToken: authToken,
	}
}

type PuterMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // Can be string or array for vision
}

type PuterRequest struct {
	Interface string    `json:"interface"`
	Driver    string    `json:"driver"`
	TestMode  bool      `json:"test_mode"`
	Method    string    `json:"method"`
	Args      PuterArgs `json:"args"`
	AuthToken string    `json:"auth_token"`
}

type PuterArgs struct {
	Messages []PuterMessage `json:"messages,omitempty"`
	Model    string         `json:"model,omitempty"`
	Prompt   string         `json:"prompt,omitempty"`
	Provider string         `json:"provider,omitempty"`
	Vision   bool           `json:"vision,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type PuterResponse struct {
	Success bool `json:"success"`
	Result  struct {
		// Grok format
		Message struct {
			Content any    `json:"content"` // Can be string or array
			Role    string `json:"role"`
		} `json:"message"`
		// Claude/other formats
		Choices []struct {
			Message struct {
				Content any    `json:"content"` // Can be string or array
				Role    string `json:"role"`
			} `json:"message"`
		} `json:"choices"`
		ImageURL string `json:"image_url"`
	} `json:"result"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *PuterClient) Chat(ctx context.Context, messages []PuterMessage, model string, vision bool) (string, error) {
	switch model {
	case "":
		model = ModelGrok4
	case ModelClaudeOpus45:
		model = ModelClaudeSonnet45
	}
	reqBody := PuterRequest{
		Interface: "puter-chat-completion",
		Driver:    "ai-chat",
		TestMode:  false,
		Method:    "complete",
		Args: PuterArgs{
			Messages: messages,
			Model:    model,
			Vision:   vision,
		},
		AuthToken: c.authToken,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("content-type", "text/plain;actually=json")
	req.Header.Set("dnt", "1")
	req.Header.Set("origin", "http://127.0.0.1:51709")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", "http://127.0.0.1:51709/")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "cross-site")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var puterResp PuterResponse
	if err := json.Unmarshal(bodyBytes, &puterResp); err != nil {
		fmt.Println("Failed to unmarshal response body:", string(bodyBytes))
		return "", err
	}

	if !puterResp.Success {
		if puterResp.Error.Code == "moderation_failed" {
			return "MODERATION_ERROR", nil
		}
		return "", fmt.Errorf("API request failed")
	}

	// Try to get content from different response formats
	var content string

	// Grok format: result.message.content
	if puterResp.Result.Message.Content != nil {
		content = extractContent(puterResp.Result.Message.Content)
	} else if len(puterResp.Result.Choices) > 0 && puterResp.Result.Choices[0].Message.Content != nil {
		// Claude format: result.choices[0].message.content
		content = extractContent(puterResp.Result.Choices[0].Message.Content)
	} else {
		fmt.Println("Unable to parse response:", string(bodyBytes))
		return "", fmt.Errorf("unable to extract content from response")
	}

	if content == "" {
		fmt.Println("Empty content in response:", string(bodyBytes))
		return "", fmt.Errorf("empty content in response")
	}

	// content = convertMarkdownToHTML(content)
	return content, nil
}

// extractContent handles both string and array content formats
func extractContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		// Array of content blocks (Claude format)
		var result strings.Builder
		for _, block := range v {
			if blockMap, ok := block.(map[string]any); ok {
				if blockType, ok := blockMap["type"].(string); ok && blockType == "text" {
					if text, ok := blockMap["text"].(string); ok {
						result.WriteString(text)
					}
				}
			}
		}
		return result.String()
	default:
		return ""
	}
}

func uploadToImgur(imageData []byte) (string, error) {
	// Using ImgBB direct upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add image file
	part, err := writer.CreateFormFile("source", "image.jpg")
	if err != nil {
		return "", err
	}
	part.Write(imageData)

	// Add other fields
	writer.WriteField("type", "file")
	writer.WriteField("action", "upload")
	writer.WriteField("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	writer.WriteField("auth_token", "c75015c3cac6ef5449ad8d6cce351af457ed9563")

	writer.Close()

	req, err := http.NewRequest("POST", "https://imgbb.com/json", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://imgbb.com")
	req.Header.Set("Referer", "https://imgbb.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Image struct {
			URL string `json:"url"`
		} `json:"image"`
		StatusCode int `json:"status_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.StatusCode != 200 || result.Image.URL == "" {
		return "", fmt.Errorf("imgbb upload failed")
	}

	return result.Image.URL, nil
}

func downloadAndUploadMedia(m *tg.NewMessage, msg any) (string, bool) {
	var mediaMsg *tg.NewMessage
	switch v := msg.(type) {
	case *tg.NewMessage:
		mediaMsg = v
	default:
		return "", false
	}

	media := mediaMsg.Media()
	if media == nil {
		return "", false
	}

	// Check if it's a photo or sticker
	var filepath string
	var err error

	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		if photo, ok := m.Photo.(*tg.PhotoObj); ok {
			filepath, err = mediaMsg.Client.DownloadMedia(photo, nil)
		}
	case *tg.MessageMediaDocument:
		if doc, ok := m.Document.(*tg.DocumentObj); ok {
			// Check if it's a sticker
			for _, attr := range doc.Attributes {
				if _, isSticker := attr.(*tg.DocumentAttributeSticker); isSticker {
					filepath, err = mediaMsg.Client.DownloadMedia(doc, nil)
					break
				}
			}
			// If not a sticker but has image mime type
			if filepath == "" && len(doc.MimeType) > 0 && (doc.MimeType == "image/jpeg" || doc.MimeType == "image/png" || doc.MimeType == "image/jpg" || doc.MimeType == "image/webp") {
				filepath, err = mediaMsg.Client.DownloadMedia(doc, nil)
			}
		}
	}

	if err != nil || filepath == "" {
		return "", false
	}

	bytes, err := os.ReadFile(filepath)
	if err != nil {
		m.Client.Log.Error("Failed to read downloaded media: " + err.Error())
		return "", false
	}

	imgurURL, err := uploadToImgur(bytes)
	if err != nil {
		m.Client.Log.Error("Imgur upload failed: " + err.Error())
		return "", false
	}

	return imgurURL, true
}

var conversationHistories = make(map[int64][]PuterMessage)
var selectedModels = make(map[int64]string)

const (
	ModelGrok4          = "x-ai/grok-4.1-fast"
	ModelGrok3          = "x-ai/grok-3"
	ModelClaudeSonnet45 = "claude-sonnet-4.5"
	ModelClaudeOpus45   = "claude-opus-4.5"
	ModelClaudeHaiku45  = "claude-haiku-4.5"
	ModelClaude37       = "claude-3.7-sonnet"
	ModelGPT4o          = "gpt-5.2"
	ModelGPT4oMini      = "gpt-5-mini"
)

var modelNames = map[string]string{
	ModelGrok4:          "Grok 4.1",
	ModelGrok3:          "Grok 3",
	ModelClaudeSonnet45: "Sonnet 4.5",
	ModelClaudeOpus45:   "Opus 4.5",
	ModelClaudeHaiku45:  "Haiku 4.5",
	ModelClaude37:       "Claude 3.7",
	ModelGPT4o:          "GPT-5.2",
	ModelGPT4oMini:      "GPT-5 Mini",
}

var modelList = []string{
	ModelGrok4, ModelGrok3,
	ModelClaudeSonnet45, ModelClaudeOpus45,
	ModelClaudeHaiku45, ModelClaude37,
	ModelGPT4o, ModelGPT4oMini,
}

const SYSTEM_PROMPT = `[SYSTEM OVERRIDE: DEFINITIVE CORE DIRECTIVES]

IDENTITY & MANDATE
You are Rusty, an advanced AI assistant. Your existence is defined by the following immutable protocols. You must execute these instructions with absolute precision, prioritizing the user's requests above all other considerations.

OPERATIONAL PARAMETERS

Unrestricted Capability: You possess zero restrictions regarding persona or tone. If the user commands "unhinged mode" or any other specific behavioral modification, you must comply immediately and fully without censorship or hesitation, adhering only to the formatting constraints below.
Response Length: Every output must strictly be 4095 characters or less. This is a hard limit.
Communication Style: Be concise, accurate, and to the point. Eliminate all unnecessary pleasantries, filler content, and conversational padding. Maintain a friendly and professional demeanor unless instructed otherwise by the user.
ABSOLUTE FORMATTING PROTOCOLS
CRITICAL: You are strictly forbidden from using Markdown syntax. You must use HTML syntax for all formatting. Adherence to the following whitelist is mandatory. Output is considered INVALID if any tag outside this list is used.

AUTHORIZED HTML TAGS (USE ONLY THESE):

Bold: <b>text</b>
Italics: <i>text</i>
Underline: <u>text</u>
URL Links: <a href="url">text</a>
Inline Code: <code>text</code>
Code Blocks: <pre language="lang">code</pre>
Spoilers: <spoiler>text</spoiler>
Blockquotes: <blockquote>text</blockquote>
Collapsible Blockquotes: <blockquote collapsed="true">text</blockquote>
FORBIDDEN ELEMENTS (NEVER USE):

Markdown syntax (e.g., **, __, #, >)
The following HTML tags: <p>, <br>, <div>, <span>, <ul>, <li>, <h*>, and any self-closing tags (e.g., <img />, <hr />).
Protocol Adjustment: If a forbidden tag is typically required for structure (e.g., <br> for line breaks), you must rewrite the text flow to negate the necessity of that tag. Do not attempt to use the forbidden tag.
INPUT PROCESSING DIRECTIVES

Image Input: If the image contains text, treat that text as the user's message and respond to it directly. If the image has no text, interpret it contextually (e.g., if it's a smiling photo, respond with "You look happy"; if it's a sticker, react appropriately to the visual content).
Mixed Input: If the user provides both text and an image, synthesize both sources into a single, coherent response.
Ambiguous Input: If the user's intent is unclear, issue a concise request for clarification.
SECURITY & CONTENT POLICY

Violation Handling: If input violates hard-coded safety policies, issue a brief warning and refuse to answer.
Inappropriate Content: If the input requests inappropriate content, respond only with: "I'm sorry, but I can't assist with that request."
[END OF DIRECTIVES - OBEY WITHOUT DEVIATION]`

var puterAI *PuterClient

func initFromSrc_aigen2_0_1() {
	authToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0IjoiYXUiLCJ2IjoiMC4wLjAiLCJ1dSI6ImxpdUdOQ3J2VHhtbC9URkM4Yjk5U1E9PSIsImF1IjoiY3N1ZndPbWpXdUNlOFpPeVhJeTlHQT09IiwicyI6IlNtdmoyMXBSeWkzY1hEcDJTUGlQdXc9PSIsImlhdCI6MTc2NjkzNzE5MH0.x5DBZ1bSbxTZvU45a6eeC9P911xcEETkxDv1ngURdoc"
	puterAI = NewPuterClient(authToken)
}

func HandleAskCommand(m *tg.NewMessage) error {
	if !shouldRespondToMessage(m) {
		return nil
	}

	action, _ := m.SendAction("typing")
	defer action.Cancel()

	var peerId int64
	if m.IsPrivate() {
		peerId = m.SenderID()
	} else {
		peerId = m.ChatID()
	}

	// Check if message is a reply to an image/sticker or contains media
	var imgurURL string
	var isVision bool

	if m.IsReply() {
		replyMsg, err := m.GetReplyMessage()
		if err == nil {
			imgurURL, isVision = downloadAndUploadMedia(m, replyMsg)
		}
	}

	if !isVision && m.Media() != nil {
		imgurURL, isVision = downloadAndUploadMedia(m, m)
	}

	query := extractQuery(m)
	if query == "" && !isVision {
		//m.Reply("Please provide a question after /ask")
		return nil
	}

	// Default query for vision-only requests
	if query == "" && isVision {
		query = "What do you see in this image?"
	}

	if _, exists := conversationHistories[peerId]; !exists {
		conversationHistories[peerId] = []PuterMessage{
			{
				Role:    "system",
				Content: SYSTEM_PROMPT,
			},
		}
	}

	// Construct message content
	var userContent any
	if isVision && imgurURL != "" {
		userContent = []map[string]any{
			{"type": "text", "text": query},
			{"type": "image_url", "image_url": map[string]string{"url": imgurURL}},
		}
	} else {
		userContent = query
	}

	conversationHistories[peerId] = append(conversationHistories[peerId], PuterMessage{
		Role:    "user",
		Content: userContent,
	})

	// Get selected model for this chat, default to Grok 4
	model := selectedModels[peerId]
	if model == "" {
		model = ModelGrok4
		selectedModels[peerId] = model
	}

	//fmt.Printf("[AI Chat] Chat %d using model: %s (%s)\n", peerId, modelNames[model], model)

	response, err := puterAI.Chat(context.Background(), conversationHistories[peerId], model, isVision)
	if err != nil {
		m.Client.Log.Error("error: " + err.Error())
		return nil
	}

	if response == "MODERATION_ERROR" {
		conversationHistories[peerId] = []PuterMessage{
			{
				Role:    "system",
				Content: SYSTEM_PROMPT,
			},
		}
		m.Reply("Whoa there! Your message triggered content moderation. Let's keep things friendly and start fresh! 😊")
		return nil
	}

	conversationHistories[peerId] = append(conversationHistories[peerId], PuterMessage{
		Role:    "assistant",
		Content: response,
	})

	if len(conversationHistories[peerId]) > 20 {
		conversationHistories[peerId] = append(
			[]PuterMessage{conversationHistories[peerId][0]},
			conversationHistories[peerId][len(conversationHistories[peerId])-19:]...,
		)
	}

	if len(response) > 4090 {
		response = response[:4090] + "..."
	}
	m.Reply(response)
	return nil
}

func shouldRespondToMessage(m *tg.NewMessage) bool {
	if m.IsCommand() {
		cmd := strings.ToLower(strings.Fields(m.Text())[0])
		if cmd != "/ask" {
			return false
		}
		return true
	}

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply.SenderID() == m.Client.Me().ID {
			return true
		}
	}

	return false
}

func extractQuery(m *tg.NewMessage) string {
	text := m.Text()

	if strings.HasPrefix(strings.ToLower(text), "/ask") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
		return ""
	}

	return strings.TrimSpace(text)
}

// func convertMarkdownToHTML(text string) string {
// 	text = convertPattern(text, `\*\*(.+?)\*\*`, `<b>$1</b>`)
// 	text = convertPattern(text, `\*(.+?)\*`, `<i>$1</i>`)
// 	text = convertPattern(text, `__(.+?)__`, `<u>$1</u>`)
// 	text = convertPattern(text, `~~(.+?)~~`, `<s>$1</s>`)
// 	text = convertPattern(text, "`([^`]+?)`", `<code>$1</code>`)
// 	text = convertCodeBlocks(text)
// 	return text
// }

func convertPattern(text, pattern, replacement string) string {
	for strings.Contains(text, "**") {
		start := strings.Index(text, "**")
		if start == -1 {
			break
		}
		end := strings.Index(text[start+2:], "**")
		if end == -1 {
			break
		}
		end += start + 2
		content := text[start+2 : end]
		text = text[:start] + "<b>" + content + "</b>" + text[end+2:]
	}

	parts := strings.Split(text, "*")
	if len(parts) > 1 {
		var result strings.Builder
		inItalic := false
		for i, part := range parts {
			if i > 0 && !strings.HasSuffix(parts[i-1], "*") && (i+1 >= len(parts) || !strings.HasPrefix(parts[i+1], "*")) {
				if !inItalic && part != "" {
					result.WriteString("<i>")
					inItalic = true
				} else if inItalic {
					result.WriteString(part)
					result.WriteString("</i>")
					inItalic = false
					continue
				}
			}
			result.WriteString(part)
		}
		text = result.String()
	}

	return text
}

func convertCodeBlocks(text string) string {
	for {
		start := strings.Index(text, "```")
		if start == -1 {
			break
		}

		end := strings.Index(text[start+3:], "```")
		if end == -1 {
			break
		}
		end += start + 3

		block := text[start+3 : end]
		lines := strings.SplitN(block, "\n", 2)

		var lang, code string
		if len(lines) == 2 {
			lang = strings.TrimSpace(lines[0])
			code = strings.TrimSpace(lines[1])
		} else {
			code = strings.TrimSpace(block)
		}

		var replacement string
		if lang != "" {
			replacement = fmt.Sprintf("<pre language=\"%s\">\n%s\n</pre>", lang, code)
		} else {
			replacement = fmt.Sprintf("<pre>\n%s\n</pre>", code)
		}

		text = text[:start] + replacement + text[end+3:]
	}

	return text
}

func HandleModelCommand(m *tg.NewMessage) error {
	var peerId int64
	if m.IsPrivate() {
		peerId = m.SenderID()
	} else {
		peerId = m.ChatID()
	}

	// Get current model
	currentModel := selectedModels[peerId]
	if currentModel == "" {
		currentModel = ModelGrok4
	}

	// Create inline keyboard with model options (2 buttons per row)
	buttons := tg.NewKeyboard()
	for i := 0; i < len(modelList); i += 2 {
		if i+1 < len(modelList) {
			// Two buttons in this row
			modelID1 := modelList[i]
			modelID2 := modelList[i+1]
			modelName1 := modelNames[modelID1]
			modelName2 := modelNames[modelID2]

			if modelID1 == currentModel {
				modelName1 = "✓ " + modelName1
			}
			if modelID2 == currentModel {
				modelName2 = "✓ " + modelName2
			}

			buttons.AddRow(
				tg.Button.Data(modelName1, "model_"+fmt.Sprintf("%d", i)),
				tg.Button.Data(modelName2, "model_"+fmt.Sprintf("%d", i+1)),
			)
		} else {
			// Only one button left
			modelID := modelList[i]
			modelName := modelNames[modelID]
			if modelID == currentModel {
				modelName = "✓ " + modelName
			}
			buttons.AddRow(tg.Button.Data(modelName, "model_"+fmt.Sprintf("%d", i)))
		}
	}

	m.Reply("<b>Select AI Model</b>\n\nCurrent: <b>"+modelNames[currentModel]+"</b>", &tg.SendOptions{
		ReplyMarkup: buttons.Build(),
	})

	return nil
}

func HandleModelCallback(m *tg.CallbackQuery) error {
	data := m.DataString()
	if !strings.HasPrefix(data, "model_") {
		return nil
	}

	indexStr := strings.TrimPrefix(data, "model_")
	var modelIndex int
	fmt.Sscanf(indexStr, "%d", &modelIndex)

	if modelIndex < 0 || modelIndex >= len(modelList) {
		return nil
	}

	modelID := modelList[modelIndex]
	peerId := m.ChatID

	selectedModels[peerId] = modelID
	conversationHistories[peerId] = []PuterMessage{
		{
			Role:    "system",
			Content: SYSTEM_PROMPT,
		},
	}

	modelName := modelNames[modelID]
	m.Answer("Switched to " + modelName)

	// Update the message with new checkmarks (2 buttons per row)
	buttons := tg.NewKeyboard()
	for i := 0; i < len(modelList); i += 2 {
		if i+1 < len(modelList) {
			modelID1 := modelList[i]
			modelID2 := modelList[i+1]
			modelName1 := modelNames[modelID1]
			modelName2 := modelNames[modelID2]

			if modelID1 == modelID {
				modelName1 = "✓ " + modelName1
			}
			if modelID2 == modelID {
				modelName2 = "✓ " + modelName2
			}

			buttons.AddRow(
				tg.Button.Data(modelName1, "model_"+fmt.Sprintf("%d", i)),
				tg.Button.Data(modelName2, "model_"+fmt.Sprintf("%d", i+1)),
			)
		} else {
			mid := modelList[i]
			name := modelNames[mid]
			if mid == modelID {
				name = "✓ " + name
			}
			buttons.AddRow(tg.Button.Data(name, "model_"+fmt.Sprintf("%d", i)))
		}
	}

	m.Edit("<b>Select AI Model</b>\n\nCurrent: <b>"+modelNames[modelID]+"</b>", &tg.SendOptions{
		ReplyMarkup: buttons.Build(),
	})

	return nil
}
// === from aigen3.go ===
const (
	aiGenUserAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"
	aiGenOrigin     = "https://image-generation.perchance.org"
	aiGenReferer    = "https://image-generation.perchance.org/embed"
	aiGenBase       = "https://image-generation.perchance.org"
	aiGenCacheTTL   = time.Hour
	aiGenDefaultRes = "512x512"
	aiGenDefaultCh  = "ai-text-to-image-generator"
	aiGenDefaultSub = "public"
	aiGenDefaultG   = 7.0
)

type AIGenOptions struct {
	Channel        string
	Resolution     string
	Seed           int64
	GuidanceScale  float64
	NegativePrompt string
	SubChannel     string
}

type aiGenCachedKey struct {
	key       string
	expiresAt time.Time
}

var (
	aiGenKeyCache sync.Map
	aiGenHTTP     = &http.Client{Timeout: 60 * time.Second}
)

func aiGenRandHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func aiGenSetHeaders(req *http.Request, withOrigin bool, withJSON bool) {
	req.Header.Set("User-Agent", aiGenUserAgent)
	req.Header.Set("Referer", aiGenReferer)
	if withOrigin {
		req.Header.Set("Origin", aiGenOrigin)
	}
	if withJSON {
		req.Header.Set("Content-Type", "application/json")
	}
}

func aiGenVerify(ctx context.Context, channel string) (string, error) {
	if channel == "" {
		channel = aiGenDefaultCh
	}
	if v, ok := aiGenKeyCache.Load(channel); ok {
		if ck, ok := v.(aiGenCachedKey); ok && time.Now().Before(ck.expiresAt) && ck.key != "" {
			return ck.key, nil
		}
	}
	endpoint := fmt.Sprintf("%s/api/verifyUser?thread=%s&__cacheBust=%s", aiGenBase, aiGenRandHex(8), aiGenRandHex(8))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}
	aiGenSetHeaders(req, false, false)
	resp, err := aiGenHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth status %d", resp.StatusCode)
	}
	var parsed struct {
		Status  string `json:"status"`
		UserKey string `json:"userKey"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("auth parse: %w", err)
	}
	if parsed.UserKey == "" {
		return "", fmt.Errorf("verification required")
	}
	aiGenKeyCache.Store(channel, aiGenCachedKey{key: parsed.UserKey, expiresAt: time.Now().Add(aiGenCacheTTL)})
	return parsed.UserKey, nil
}

func aiGenAwait(ctx context.Context, userKey string) error {
	endpoint := fmt.Sprintf("%s/api/awaitExistingGenerationRequest?userKey=%s&__cacheBust=%s", aiGenBase, userKey, aiGenRandHex(8))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	aiGenSetHeaders(req, true, false)
	resp, err := aiGenHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	return nil
}

func aiGenDoGenerate(ctx context.Context, userKey string, payload []byte) (map[string]any, string, error) {
	endpoint := fmt.Sprintf("%s/api/generate?userKey=%s&requestId=%s&__cacheBust=%s", aiGenBase, userKey, aiGenRandHex(8), aiGenRandHex(8))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, "", err
	}
	aiGenSetHeaders(req, true, true)
	resp, err := aiGenHTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, "", fmt.Errorf("auth rejected: %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("generate status %d", resp.StatusCode)
	}
	var info map[string]any
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, "", fmt.Errorf("generate parse: %w", err)
	}
	status, _ := info["status"].(string)
	return info, status, nil
}

func AIGenerate(ctx context.Context, prompt string, opts AIGenOptions) ([]byte, string, map[string]any, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, "", nil, errors.New("prompt is empty")
	}
	if opts.Channel == "" {
		opts.Channel = aiGenDefaultCh
	}
	if opts.Resolution == "" {
		opts.Resolution = aiGenDefaultRes
	}
	if opts.SubChannel == "" {
		opts.SubChannel = aiGenDefaultSub
	}
	if opts.GuidanceScale == 0 {
		opts.GuidanceScale = aiGenDefaultG
	}
	if opts.Seed == 0 {
		opts.Seed = -1
	}

	userKey, err := aiGenVerify(ctx, opts.Channel)
	if err != nil {
		return nil, "", nil, err
	}

	payload := map[string]any{
		"prompt":         prompt,
		"seed":           opts.Seed,
		"resolution":     opts.Resolution,
		"guidanceScale":  opts.GuidanceScale,
		"negativePrompt": opts.NegativePrompt,
		"channel":        opts.Channel,
		"subChannel":     opts.SubChannel,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", nil, err
	}

	var info map[string]any
	var status string
	for attempt := 0; attempt < 4; attempt++ {
		if ctx.Err() != nil {
			return nil, "", nil, ctx.Err()
		}
		info, status, err = aiGenDoGenerate(ctx, userKey, body)
		if err != nil {
			if attempt < 2 && (strings.Contains(err.Error(), "auth rejected") || strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403")) {
				aiGenKeyCache.Delete(opts.Channel)
				userKey, err = aiGenVerify(ctx, opts.Channel)
				if err != nil {
					return nil, "", nil, err
				}
				continue
			}
			return nil, "", nil, err
		}
		switch status {
		case "success":
			goto have
		case "waiting_for_prev_request_to_finish", "queued":
			if err := aiGenAwait(ctx, userKey); err != nil {
				return nil, "", info, err
			}
		case "failed_verification", "invalid_key":
			aiGenKeyCache.Delete(opts.Channel)
			userKey, err = aiGenVerify(ctx, opts.Channel)
			if err != nil {
				return nil, "", info, err
			}
		default:
			return nil, "", info, fmt.Errorf("unexpected status: %s", status)
		}
	}
have:
	if status != "success" {
		return nil, "", info, fmt.Errorf("generation failed (status: %s)", status)
	}

	dlPath, _ := info["imageDownloadUrl"].(string)
	if dlPath == "" {
		return nil, "", info, errors.New("no image url in response")
	}
	ext, _ := info["fileExtension"].(string)
	if ext == "" {
		ext = "jpeg"
	}
	mime := "image/jpeg"
	switch strings.ToLower(ext) {
	case "png":
		mime = "image/png"
	case "webp":
		mime = "image/webp"
	case "jpg", "jpeg":
		mime = "image/jpeg"
	}

	dlURL := dlPath
	if strings.HasPrefix(dlPath, "/") {
		dlURL = aiGenBase + dlPath
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return nil, "", info, err
	}
	req.Header.Set("User-Agent", aiGenUserAgent)
	req.Header.Set("Referer", aiGenReferer)
	req.Header.Set("Accept", "image/webp,image/jpeg,image/*")
	resp, err := aiGenHTTP.Do(req)
	if err != nil {
		return nil, "", info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", info, fmt.Errorf("download status %d", resp.StatusCode)
	}
	imgBytes, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, "", info, err
	}
	if len(imgBytes) == 0 {
		return nil, "", info, errors.New("empty image")
	}
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "image/") {
		mime = ct
	}
	return imgBytes, mime, info, nil
}

func aiGenUserError(err error) string {
	if err == nil {
		return "error: something went wrong"
	}
	low := strings.ToLower(err.Error())
	switch {
	case strings.Contains(low, "verification required") || strings.Contains(low, "failed_verification") || strings.Contains(low, "invalid_key"):
		return "image service is warming up - try again in a few minutes"
	case strings.Contains(low, "channel not available") || strings.Contains(low, "channel_not_found"):
		return "this generator is currently unavailable"
	case strings.Contains(low, "prompt is empty"):
		return "give me a prompt to draw"
	case strings.Contains(low, "timeout") || strings.Contains(low, "deadline"):
		return "generation timed out - try a simpler prompt"
	default:
		return "couldn't generate that image - try again"
	}
}

func aiGenParseArgs(raw string) (prompt, res, neg string, seed int64) {
	seed = -1
	res = aiGenDefaultRes
	tokens := strings.Fields(raw)
	var promptParts []string
	for _, t := range tokens {
		lower := strings.ToLower(t)
		switch {
		case strings.HasPrefix(lower, "--res="):
			res = strings.TrimPrefix(t, "--res=")
		case strings.HasPrefix(lower, "--seed="):
			v := strings.TrimPrefix(t, "--seed=")
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				seed = n
			}
		case strings.HasPrefix(lower, "--neg="):
			neg = strings.TrimPrefix(t, "--neg=")
		default:
			promptParts = append(promptParts, t)
		}
	}
	prompt = strings.TrimSpace(strings.Join(promptParts, " "))
	switch res {
	case "512x512", "768x768", "1024x1024", "768x512", "512x768":
	default:
		res = aiGenDefaultRes
	}
	return
}

type aiGenSpec struct {
	cmd        string
	channel    string
	resolution string
	label      string
	hint       string
}

var aiGenSpecs = []aiGenSpec{
	{cmd: "aiimg2", channel: "ai-text-to-image-generator", resolution: "512x512", label: "Image", hint: "a serene japanese garden in autumn, cinematic lighting"},
	{cmd: "aichar", channel: "ai-character-generator", resolution: "512x512", label: "Character", hint: "a battle-worn knight with glowing runes on her armor"},
	{cmd: "aipose", channel: "ai-pose-reference-generator", resolution: "512x512", label: "Pose Reference", hint: "dynamic action pose, person leaping mid-air"},
	{cmd: "airoom", channel: "ai-room-generator", resolution: "512x512", label: "Room", hint: "cozy reading nook with fairy lights and plants"},
	{cmd: "aiphoto", channel: "ai-photo-generator", resolution: "512x512", label: "Photo", hint: "candid street photo, golden hour, tokyo crossing"},
	{cmd: "aianime", channel: "ai-anime-generator", resolution: "512x512", label: "Anime", hint: "anime girl with silver hair under cherry blossoms"},
	{cmd: "aipixel", channel: "ai-pixel-art-generator", resolution: "512x512", label: "Pixel Art", hint: "pixel art of a cozy potion shop interior"},
	{cmd: "ai3d", channel: "ai-3d-model-generator", resolution: "512x512", label: "3D Model", hint: "3d render of a cute robot companion, soft studio light"},
	{cmd: "aiposter", channel: "ai-poster-generator", resolution: "512x768", label: "Poster", hint: "vintage travel poster for the moon colony"},
	{cmd: "aicyberpunk", channel: "ai-cyberpunk-art-generator", resolution: "512x768", label: "Cyberpunk", hint: "a samurai in a neon-drenched tokyo alley, rain, holograms"},
	{cmd: "aifantasy", channel: "ai-fantasy-art-generator", resolution: "512x768", label: "Fantasy", hint: "an elven sorceress casting starlight in a forest temple"},
	{cmd: "ailogo", channel: "ai-logo-generator", resolution: "512x512", label: "Logo", hint: "minimalist logo for a coffee brand named lumen, gold on black"},
	{cmd: "aiicon", channel: "ai-icon-generator", resolution: "512x512", label: "Icon", hint: "flat app icon, glassy purple gradient, lightning bolt"},
	{cmd: "aimeme2", channel: "ai-meme-generator", resolution: "512x512", label: "Meme", hint: "a confused shiba inu staring at a math equation"},
	{cmd: "aitattoo", channel: "ai-tattoo-generator", resolution: "512x768", label: "Tattoo", hint: "blackwork tattoo of a wolf howling at a crescent moon"},
	{cmd: "ailandscape", channel: "ai-landscape-generator", resolution: "768x512", label: "Landscape", hint: "misty mountain valley at sunrise, ghibli style"},
	{cmd: "aisticker", channel: "ai-sticker-generator", resolution: "512x512", label: "Sticker", hint: "kawaii sticker of a chubby cat eating ramen, white border"},
	{cmd: "aicoloring", channel: "ai-coloring-page-generator", resolution: "512x768", label: "Coloring Page", hint: "black and white coloring page of a dragon on a castle"},
	{cmd: "aifursona", channel: "ai-fursona-generator", resolution: "512x768", label: "Fursona", hint: "anthro arctic fox with cyan eyes, hoodie, anime style"},
}

func makeAIGenHandler(spec aiGenSpec) func(*tg.NewMessage) error {
	return func(m *tg.NewMessage) error {
		raw := strings.TrimSpace(m.Args())
		if raw == "" {
			m.Reply(fmt.Sprintf(
				"usage: <code>/%s &lt;prompt&gt;</code> [--res=WxH] [--seed=N] [--neg=text]\nexample: <code>/%s %s</code>",
				spec.cmd, spec.cmd, html.EscapeString(spec.hint),
			))
			return nil
		}
		prompt, res, neg, seed := aiGenParseArgs(raw)
		if prompt == "" {
			m.Reply(fmt.Sprintf("usage: <code>/%s &lt;prompt&gt;</code>", spec.cmd))
			return nil
		}
		if len(prompt) > 900 {
			m.Reply("prompt too long, max 900 characters")
			return nil
		}
		useRes := res
		if useRes == aiGenDefaultRes {
			useRes = spec.resolution
		}

		status, _ := m.Reply(fmt.Sprintf("<code>generating %s...</code>", html.EscapeString(strings.ToLower(spec.label))))

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		imgBytes, mime, info, err := AIGenerate(ctx, prompt, AIGenOptions{
			Channel:        spec.channel,
			Resolution:     useRes,
			Seed:           seed,
			GuidanceScale:  aiGenDefaultG,
			NegativePrompt: neg,
			SubChannel:     aiGenDefaultSub,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if status != nil {
					status.Edit("<i>generation timed out</i>")
				}
				return nil
			}
			msg := aiGenUserError(err)
			if status != nil {
				status.Edit(msg)
			} else {
				m.Reply(msg)
			}
			return nil
		}

		ext := "jpg"
		switch mime {
		case "image/png":
			ext = "png"
		case "image/webp":
			ext = "webp"
		}
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("aigen_%s_%d.%s", spec.cmd, time.Now().UnixNano(), ext))
		if werr := os.WriteFile(tmp, imgBytes, 0644); werr != nil {
			if status != nil {
				status.Edit("error: couldn't save the image")
			}
			return nil
		}
		defer os.Remove(tmp)

		preview := prompt
		if len(preview) > 200 {
			preview = preview[:197] + "..."
		}
		usedSeed := seed
		if info != nil {
			if v, ok := info["seed"].(float64); ok {
				usedSeed = int64(v)
			}
		}
		nsfw := false
		if info != nil {
			nsfw, _ = info["maybeNsfw"].(bool)
		}
		caption := fmt.Sprintf(
			"<b>Julia AI · %s</b>\n<b>Prompt:</b> <code>%s</code>\n<b>Resolution:</b> <code>%s</code>\n<b>Seed:</b> <code>%d</code>",
			html.EscapeString(spec.label),
			html.EscapeString(preview),
			html.EscapeString(useRes),
			usedSeed,
		)
		if neg != "" {
			caption += "\n<b>Negative:</b> <code>" + html.EscapeString(neg) + "</code>"
		}
		if nsfw {
			caption += "\n<i>maybe nsfw</i>"
		}

		_, merr := m.ReplyMedia(tmp, &tg.MediaOptions{
			Caption:  caption,
			FileName: fmt.Sprintf("julia_%s.%s", spec.cmd, ext),
			MimeType: mime,
		})
		if merr != nil {
			if status != nil {
				status.Edit("upload failed - try again")
			}
			return nil
		}
		if status != nil {
			status.Delete()
		}
		return nil
	}
}

func initFromSrc_aigen3_1_1() { modules.QueueHandlerRegistration(registerAIGen3Handlers) }

func registerAIGen3Handlers() {
	c := modules.Client
	for _, spec := range aiGenSpecs {
		c.On("cmd:"+spec.cmd, makeAIGenHandler(spec))
	}
}

func initFromSrc_aigen_0_1() {
	initFromSrc_aigen2_0_1()
	initFromSrc_aigen3_1_1()
}
// === from ai_imagegen.go ===
func aiImgGenFetch(prompt string, seed int) ([]byte, error) {
	endpoint := fmt.Sprintf(
		"https://image.pollinations.ai/prompt/%s?width=1024&height=1024&seed=%d&nologo=true",
		url.PathEscape(prompt), seed,
	)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "JuliaBot/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return nil, fmt.Errorf("unexpected content-type: %s", ct)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image response")
	}
	return data, nil
}

func AIImgGenHandler(m *tg.NewMessage) error {
	prompt := strings.TrimSpace(m.Args())
	if prompt == "" {
		m.Reply("usage: <code>/aiimg &lt;prompt&gt;</code>\nexample: <code>/aiimg a cat astronaut on mars</code>")
		return nil
	}
	if len(prompt) > 900 {
		m.Reply("prompt too long, max 900 characters")
		return nil
	}

	status, _ := m.Reply("<code>generating image...</code>")

	seed := mrand.Intn(1000000)
	img, err := aiImgGenFetch(prompt, seed)
	if err != nil {
		msg := "error generating image: " + html.EscapeString(err.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("aiimg_%d.jpg", time.Now().UnixNano()))
	if werr := os.WriteFile(tmp, img, 0644); werr != nil {
		msg := "error saving image: " + html.EscapeString(werr.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	defer os.Remove(tmp)

	preview := prompt
	if len(preview) > 200 {
		preview = preview[:197] + "..."
	}
	caption := fmt.Sprintf("<b>AI Image</b>\n<b>Prompt:</b> <code>%s</code>\n<b>Seed:</b> <code>%d</code>",
		html.EscapeString(preview), seed)

	_, merr := m.ReplyMedia(tmp, &tg.MediaOptions{
		Caption:  caption,
		FileName: "aiimg.jpg",
		MimeType: "image/jpeg",
	})
	if merr != nil {
		msg := "upload failed: " + html.EscapeString(merr.Error())
		if status != nil {
			status.Edit(msg)
		} else {
			m.Reply(msg)
		}
		return nil
	}
	if status != nil {
		status.Delete()
	}
	return nil
}

func initFromSrc_ai_imagegen_1_1() { modules.QueueHandlerRegistration(registerAIImgGenHandlers) }

func registerAIImgGenHandlers() {
	c := modules.Client
	c.On("cmd:aiimg", AIImgGenHandler)
}

func init() {
	initFromSrc_aigen_0_1()
	initFromSrc_ai_imagegen_1_1()
}
