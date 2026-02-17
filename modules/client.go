package modules

import (
	"os"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	Client       *tg.Client
	OwnerId      int64
	LoadModules  bool
	handlerQueue []func()
)

func InitClient(c *tg.Client) {
	Client = c
}

func QueueHandlerRegistration(fn func()) {
	handlerQueue = append(handlerQueue, fn)
}

func RegisterHandlers() {
	if Client == nil {
		panic("Client not initialized")
	}

	_, _ = Client.UpdatesGetState()
	Client.SetCommandPrefixes("./!-?")

	if !LoadModules {
		return
	}

	Client.Logger.Info("Loading Modules...")

	for _, registerFn := range handlerQueue {
		registerFn()
	}

	Mods.Init(Client)
}

func SetupFilters(ownerId int64, loadModules bool) {
	OwnerId = ownerId
	LoadModules = loadModules
}

func FilterOwner(m *tg.NewMessage) bool {
	if m.SenderID() == OwnerId {
		return true
	}
	m.Reply("You are not allowed to use this command")
	return false
}

func FilterOwnerAndAuth(m *tg.NewMessage) bool {
	auths := os.Getenv("AUTH_USERS")
	if auths == "" {
		return FilterOwner(m)
	} else {
		if m.SenderID() == OwnerId {
			return true
		}
		au := strings.SplitSeq(auths, ",")
		for user := range au {
			if strings.TrimSpace(user) == strconv.Itoa(int(m.SenderID())) {
				return true
			}
		}
	}

	m.Reply("You are not allowed to use this command")
	return false
}

func FilterOwnerNoReply(m *tg.NewMessage) bool {
	return m.SenderID() == OwnerId
}
