package modules

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

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

func init() { QueueHandlerRegistration(registerAIGen3Handlers) }

func registerAIGen3Handlers() {
	c := Client
	for _, spec := range aiGenSpecs {
		c.On("cmd:"+spec.cmd, makeAIGenHandler(spec))
	}
}
