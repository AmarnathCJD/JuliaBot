package modules

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"html"
	"io"
	"os"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func hashComputeText(algo, text string) (string, error) {
	var h hash.Hash
	switch strings.ToLower(algo) {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", fmt.Errorf("unsupported algorithm")
	}
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashComputeFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hMD5 := md5.New()
	hSHA1 := sha1.New()
	hSHA256 := sha256.New()
	hSHA512 := sha512.New()

	mw := io.MultiWriter(hMD5, hSHA1, hSHA256, hSHA512)
	if _, err := io.Copy(mw, f); err != nil {
		return nil, err
	}

	return map[string]string{
		"md5":    hex.EncodeToString(hMD5.Sum(nil)),
		"sha1":   hex.EncodeToString(hSHA1.Sum(nil)),
		"sha256": hex.EncodeToString(hSHA256.Sum(nil)),
		"sha512": hex.EncodeToString(hSHA512.Sum(nil)),
	}, nil
}

func HashHandler(m *tg.NewMessage) error {
	if m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil && r.IsMedia() {
			status, _ := m.Reply("<code>downloading file...</code>")

			path, err := m.Client.DownloadMedia(r.Media())
			if err != nil {
				msg := "error downloading: " + html.EscapeString(err.Error())
				if status != nil {
					status.Edit(msg)
				} else {
					m.Reply(msg)
				}
				return nil
			}
			defer os.Remove(path)

			if status != nil {
				status.Edit("<code>computing hashes...</code>")
			}

			hashes, err := hashComputeFile(path)
			if err != nil {
				msg := "error computing: " + html.EscapeString(err.Error())
				if status != nil {
					status.Edit(msg)
				} else {
					m.Reply(msg)
				}
				return nil
			}

			var sb strings.Builder
			sb.WriteString("<b>File Hashes</b>\n\n")
			sb.WriteString("<b>MD5:</b>\n<code>" + html.EscapeString(hashes["md5"]) + "</code>\n\n")
			sb.WriteString("<b>SHA1:</b>\n<code>" + html.EscapeString(hashes["sha1"]) + "</code>\n\n")
			sb.WriteString("<b>SHA256:</b>\n<code>" + html.EscapeString(hashes["sha256"]) + "</code>\n\n")
			sb.WriteString("<b>SHA512:</b>\n<code>" + html.EscapeString(hashes["sha512"]) + "</code>")

			out := sb.String()
			if status != nil {
				status.Edit(out)
			} else {
				m.Reply(out)
			}
			return nil
		}
	}

	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply("usage: <code>/hash &lt;md5|sha1|sha256|sha512&gt; &lt;text&gt;</code>\nor reply to a file with <code>/hash</code>")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		m.Reply("usage: <code>/hash &lt;md5|sha1|sha256|sha512&gt; &lt;text&gt;</code>")
		return nil
	}

	algo := strings.ToLower(strings.TrimSpace(parts[0]))
	text := parts[1]

	digest, err := hashComputeText(algo, text)
	if err != nil {
		m.Reply("error: unsupported algorithm. use md5, sha1, sha256, or sha512")
		return nil
	}

	out := "<b>" + strings.ToUpper(algo) + ":</b>\n<code>" + html.EscapeString(digest) + "</code>"
	m.Reply(out)
	return nil
}

func init() { QueueHandlerRegistration(registerHashHandlers) }

func registerHashHandlers() {
	c := Client
	c.On("cmd:hash", HashHandler)
}
