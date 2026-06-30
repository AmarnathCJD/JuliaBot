package extras

import (
	"fmt"
	"html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

type avatarTarget struct {
	UserID    int64
	FirstName string
	LastName  string
	Username  string
	Photo     tg.UserProfilePhoto
}

func avatarResolveTarget(m *tg.NewMessage) (avatarTarget, error) {
	var info avatarTarget

	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply != nil && reply.SenderID() != 0 {
			u, uerr := m.Client.GetUser(reply.SenderID())
			if uerr == nil && u != nil {
				info.UserID = u.ID
				info.FirstName = u.FirstName
				info.LastName = u.LastName
				info.Username = u.Username
				info.Photo = u.Photo
				return info, nil
			}
			info.UserID = reply.SenderID()
			info.FirstName = "User"
			return info, nil
		}
	}

	args := strings.TrimSpace(m.Args())
	if args != "" {
		token := strings.Fields(args)[0]
		token = strings.TrimPrefix(token, "@")
		if n, err := strconv.ParseInt(token, 10, 64); err == nil {
			u, uerr := m.Client.GetUser(n)
			if uerr == nil && u != nil {
				info.UserID = u.ID
				info.FirstName = u.FirstName
				info.LastName = u.LastName
				info.Username = u.Username
				info.Photo = u.Photo
				return info, nil
			}
			return info, fmt.Errorf("could not resolve user %d", n)
		}
		peer, err := m.Client.ResolvePeer(token)
		if err != nil {
			return info, err
		}
		id := m.Client.GetPeerID(peer)
		u, uerr := m.Client.GetUser(id)
		if uerr == nil && u != nil {
			info.UserID = u.ID
			info.FirstName = u.FirstName
			info.LastName = u.LastName
			info.Username = u.Username
			info.Photo = u.Photo
			return info, nil
		}
		info.UserID = id
		info.FirstName = token
		return info, nil
	}

	if m.Sender != nil {
		info.UserID = m.Sender.ID
		info.FirstName = m.Sender.FirstName
		info.LastName = m.Sender.LastName
		info.Username = m.Sender.Username
		info.Photo = m.Sender.Photo
		return info, nil
	}

	info.UserID = m.SenderID()
	info.FirstName = "User"
	return info, nil
}

func avatarGetAccessHash(m *tg.NewMessage, userID int64) int64 {
	peer, err := m.Client.ResolvePeer(userID)
	if err != nil {
		return 0
	}
	if pu, ok := peer.(*tg.InputPeerUser); ok {
		return pu.AccessHash
	}
	return 0
}

func avatarDownload(m *tg.NewMessage, info avatarTarget) (string, error) {
	if info.Photo == nil {
		return "", fmt.Errorf("user has no profile photo")
	}
	full, err := m.Client.UsersGetFullUser(&tg.InputUserObj{
		UserID:     info.UserID,
		AccessHash: avatarGetAccessHash(m, info.UserID),
	})
	if err != nil || full == nil {
		return "", fmt.Errorf("could not fetch full user")
	}
	uf := full.FullUser
	var photo tg.Photo
	if uf.ProfilePhoto != nil {
		photo = uf.ProfilePhoto
	} else if uf.PersonalPhoto != nil {
		photo = uf.PersonalPhoto
	} else if uf.FallbackPhoto != nil {
		photo = uf.FallbackPhoto
	}
	if photo == nil {
		return "", fmt.Errorf("no profile photo available")
	}
	p, ok := photo.(*tg.PhotoObj)
	if !ok || p == nil {
		return "", fmt.Errorf("invalid photo object")
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("avatar_dl_%d_%d.jpg", info.UserID, time.Now().UnixNano()))
	_, err = m.Client.DownloadMedia(p, &tg.DownloadOptions{
		FileName: tmp,
	})
	if err != nil {
		os.Remove(tmp)
		return "", err
	}
	return tmp, nil
}

func avatarFormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func avatarImageDimensions(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func AvatarHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("fetching avatar...")

	info, err := avatarResolveTarget(m)
	if err != nil {
		if status != nil {
			status.Edit("failed: " + html.EscapeString(err.Error()))
		}
		return nil
	}

	if info.UserID == 0 {
		if status != nil {
			status.Edit("could not resolve user")
		}
		return nil
	}

	path, err := avatarDownload(m, info)
	if err != nil {
		if status != nil {
			status.Edit("failed: " + html.EscapeString(err.Error()))
		}
		return nil
	}
	defer os.Remove(path)

	st, serr := os.Stat(path)
	var size int64
	if serr == nil {
		size = st.Size()
	}
	w, h := avatarImageDimensions(path)

	displayName := strings.TrimSpace(info.FirstName + " " + info.LastName)
	if displayName == "" {
		displayName = "User"
	}

	caption := fmt.Sprintf("<b>%s</b>", html.EscapeString(displayName))
	if info.Username != "" {
		caption += fmt.Sprintf("\n@%s", html.EscapeString(info.Username))
	}
	caption += fmt.Sprintf("\n<code>%d</code>", info.UserID)
	if w > 0 && h > 0 {
		caption += fmt.Sprintf("\n<b>Size:</b> <code>%s</code>  <b>Dim:</b> <code>%dx%d</code>", avatarFormatBytes(size), w, h)
	} else {
		caption += fmt.Sprintf("\n<b>Size:</b> <code>%s</code>", avatarFormatBytes(size))
	}

	_, merr := m.ReplyMedia(path, &tg.MediaOptions{
		Caption:  caption,
		FileName: fmt.Sprintf("avatar_%d.jpg", info.UserID),
		MimeType: "image/jpeg",
	})
	if merr != nil {
		if status != nil {
			status.Edit("upload failed: " + html.EscapeString(merr.Error()))
		}
		return nil
	}
	if status != nil {
		status.Delete()
	}
	return nil
}

func registerAvatarHandlers() {
	c := modules.Client
	c.On("cmd:avatar", AvatarHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerAvatarHandlers)
}
