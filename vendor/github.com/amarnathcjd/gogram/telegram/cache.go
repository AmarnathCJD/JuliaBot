// Copyright (c) 2024 RoseLoverX

package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/amarnathcjd/gogram/internal/utils"
)

type CACHE struct {
	*sync.RWMutex
	fileN      string
	chats      map[int64]*ChatObj
	users      map[int64]*UserObj
	channels   map[int64]*Channel
	writeFile  bool
	InputPeers *InputPeerCache `json:"input_peers,omitempty"`
	logger     *utils.Logger
}

type InputPeerCache struct {
	InputChannels map[int64]int64 `json:"channels,omitempty"`
	InputUsers    map[int64]int64 `json:"users,omitempty"`
	InputChats    map[int64]int64 `json:"chats,omitempty"`
}

func (c *CACHE) SetWriteFile(write bool) {
	c.writeFile = write
}

func (c *CACHE) Clear() {
	c.Lock()
	defer c.Unlock()

	c.chats = make(map[int64]*ChatObj)
	c.users = make(map[int64]*UserObj)
	c.channels = make(map[int64]*Channel)
	c.InputPeers = &InputPeerCache{
		InputChannels: make(map[int64]int64),
		InputUsers:    make(map[int64]int64),
		InputChats:    make(map[int64]int64),
	}
}

func (c *CACHE) ExportJSON() ([]byte, error) {
	c.RLock()
	defer c.RUnlock()

	return json.Marshal(c.InputPeers)
}

func (c *CACHE) ImportJSON(data []byte) error {
	c.Lock()
	defer c.Unlock()

	return json.Unmarshal(data, c.InputPeers)
}

func NewCache(logLevel string, fileN string) *CACHE {
	c := &CACHE{
		RWMutex:  &sync.RWMutex{},
		fileN:    fileN + ".db",
		chats:    make(map[int64]*ChatObj),
		users:    make(map[int64]*UserObj),
		channels: make(map[int64]*Channel),
		InputPeers: &InputPeerCache{
			InputChannels: make(map[int64]int64),
			InputUsers:    make(map[int64]int64),
			InputChats:    make(map[int64]int64),
		},
		logger: utils.NewLogger("gogram [cache]").SetLevel(logLevel),
	}

	c.logger.Debug("initialized cache (" + c.fileN + ") successfully")

	if _, err := os.Stat(c.fileN); err == nil && c.writeFile {
		c.ReadFile()
	}

	return c
}

// --------- Cache file Functions ---------
func (c *CACHE) WriteFile() {
	file, err := os.OpenFile(c.fileN, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		c.logger.Error("error opening cache file: ", err)
		return
	}
	defer file.Close()

	var buffer strings.Builder

	c.Lock()
	for id, accessHash := range c.InputPeers.InputUsers {
		buffer.WriteString(fmt.Sprintf("1:%d:%d,", id, accessHash))
	}

	for id, accessHash := range c.InputPeers.InputChats {
		buffer.WriteString(fmt.Sprintf("2:%d:%d,", id, accessHash))
	}

	for id, accessHash := range c.InputPeers.InputChannels {
		buffer.WriteString(fmt.Sprintf("3:%d:%d,", id, accessHash))
	}
	c.Unlock()

	if _, err := file.WriteString(buffer.String()); err != nil {
		c.logger.Error("error writing to cache file: ", err)
	}
}

func (c *CACHE) ReadFile() {
	file, err := os.Open(c.fileN)
	if err != nil && !os.IsNotExist(err) {
		c.logger.Error("error opening cache file: ", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, 1)
	var data []byte
	totalLoaded := 0
	for {
		_, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				c.logger.Debug("error reading from cache file: ", err)
			}
			break
		}
		if buffer[0] == ',' {
			data = append(data, buffer[0])
			if processed := c.processData(data); processed {
				totalLoaded++
			}
			data = nil
		} else {
			data = append(data, buffer[0])
		}
	}

	if totalLoaded != 0 {
		c.logger.Debug("loaded ", totalLoaded, " peers from cacheFile")
	}
}

func (c *CACHE) processData(data []byte) bool {
	// data format: 'type:id:access_hash'
	// type: 1 for user, 2 for chat, 3 for channel
	// split data
	splitData := strings.Split(string(data), ":")
	if len(splitData) != 3 {
		return false
	}
	// convert to int
	id, err := strconv.Atoi(splitData[1])
	if err != nil {
		return false
	}
	accessHash, err := strconv.Atoi(strings.TrimSuffix(splitData[2], ","))
	if err != nil {
		return false
	}

	// process data
	c.Lock()
	defer c.Unlock()
	switch splitData[0] {
	case "1":
		c.InputPeers.InputUsers[int64(id)] = int64(accessHash)
	case "2":
		c.InputPeers.InputChats[int64(id)] = int64(accessHash)
	case "3":
		c.InputPeers.InputChannels[int64(id)] = int64(accessHash)
	default:
		return false
	}

	return true
}

func (c *CACHE) getUserPeer(userID int64) (InputUser, error) {
	c.RLock()
	defer c.RUnlock()

	if userHash, ok := c.InputPeers.InputUsers[userID]; ok {
		return &InputUserObj{UserID: userID, AccessHash: userHash}, nil
	}

	return nil, fmt.Errorf("no user with id %d or missing from cache", userID)
}

func (c *CACHE) getChannelPeer(channelID int64) (InputChannel, error) {
	c.RLock()
	defer c.RUnlock()

	if channelHash, ok := c.InputPeers.InputChannels[channelID]; ok {
		return &InputChannelObj{ChannelID: channelID, AccessHash: channelHash}, nil
	}

	return nil, fmt.Errorf("no channel with id %d or missing from cache", channelID)
}

func (c *CACHE) GetInputPeer(peerID int64) (InputPeer, error) {
	// if peerID is negative, it is a channel or a chat
	peerIdStr := strconv.Itoa(int(peerID))
	if strings.HasPrefix(peerIdStr, "-100") {
		peerIdStr = strings.TrimPrefix(peerIdStr, "-100")

		if peerIdInt, err := strconv.Atoi(peerIdStr); err == nil {
			peerID = int64(peerIdInt)
		} else {
			return nil, err
		}
	}
	c.RLock()
	defer c.RUnlock()

	if userHash, ok := c.InputPeers.InputUsers[peerID]; ok {
		return &InputPeerUser{peerID, userHash}, nil
	}

	if _, ok := c.InputPeers.InputChats[peerID]; ok {
		return &InputPeerChat{ChatID: peerID}, nil
	}

	if channelHash, ok := c.InputPeers.InputChannels[peerID]; ok {
		return &InputPeerChannel{peerID, channelHash}, nil
	}

	return nil, fmt.Errorf("there is no peer with id %d or missing from cache", peerID)
}

// ------------------ Get Chat/Channel/User From Cache/Telgram ------------------

func (c *Client) getUserFromCache(userID int64) (*UserObj, error) {
	c.Cache.RLock()
	if user, found := c.Cache.users[userID]; found {
		c.Cache.RUnlock()
		return user, nil
	}
	c.Cache.RUnlock()

	userPeer, err := c.Cache.getUserPeer(userID)
	if err != nil {
		return nil, err
	}

	users, err := c.UsersGetUsers([]InputUser{userPeer})
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("no user with id %d", userID)
	}

	user, ok := users[0].(*UserObj)
	if !ok {
		return nil, fmt.Errorf("expected UserObj for id %d, but got different type", userID)
	}

	return user, nil
}

func (c *Client) getChannelFromCache(channelID int64) (*Channel, error) {
	c.Cache.RLock()
	if channel, found := c.Cache.channels[channelID]; found {
		c.Cache.RUnlock()
		return channel, nil
	}
	c.Cache.RUnlock()

	channelPeer, err := c.Cache.getChannelPeer(channelID)
	if err != nil {
		return nil, err
	}

	channels, err := c.ChannelsGetChannels([]InputChannel{channelPeer})
	if err != nil {
		return nil, err
	}

	channelsObj, ok := channels.(*MessagesChatsObj)
	if !ok {
		return nil, fmt.Errorf("expected MessagesChatsObj for channel id %d, but got different type", channelID)
	}

	if len(channelsObj.Chats) == 0 {
		return nil, fmt.Errorf("no channel with id %d", channelID)
	}

	channel, ok := channelsObj.Chats[0].(*Channel)
	if !ok {
		return nil, fmt.Errorf("expected Channel for id %d, but got different type", channelID)
	}

	return channel, nil
}

func (c *Client) getChatFromCache(chatID int64) (*ChatObj, error) {
	c.Cache.RLock()
	if chat, found := c.Cache.chats[chatID]; found {
		c.Cache.RUnlock()
		return chat, nil
	}
	c.Cache.RUnlock()

	chat, err := c.MessagesGetChats([]int64{chatID})
	if err != nil {
		return nil, err
	}

	chatsObj, ok := chat.(*MessagesChatsObj)
	if !ok {
		return nil, fmt.Errorf("expected MessagesChatsObj for chat id %d, but got different type", chatID)
	}

	if len(chatsObj.Chats) == 0 {
		return nil, fmt.Errorf("no chat with id %d", chatID)
	}

	chatObj, ok := chatsObj.Chats[0].(*ChatObj)
	if !ok {
		return nil, fmt.Errorf("expected ChatObj for id %d, but got different type", chatID)
	}

	return chatObj, nil
}

// ----------------- Get User/Channel/Chat from cache -----------------

func (c *Client) GetUser(userID int64) (*UserObj, error) {
	user, err := c.getUserFromCache(userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (c *Client) GetChannel(channelID int64) (*Channel, error) {
	channel, err := c.getChannelFromCache(channelID)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func (c *Client) GetChat(chatID int64) (*ChatObj, error) {
	chat, err := c.getChatFromCache(chatID)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

// mux function to getChat/getChannel/getUser
func (c *Client) GetPeer(peerID int64) (any, error) {
	if chat, err := c.GetChat(peerID); err == nil {
		return chat, nil
	} else if channel, err := c.GetChannel(peerID); err == nil {
		return channel, nil
	} else if user, err := c.GetUser(peerID); err == nil {
		return user, nil
	} else {
		return nil, err
	}
}

// ----------------- Update User/Channel/Chat in cache -----------------

func (c *CACHE) UpdateUser(user *UserObj) bool {
	c.Lock()
	defer c.Unlock()

	if user.Min {
		if userFromCache, ok := c.users[user.ID]; ok {
			if userFromCache.Min {
				c.users[user.ID] = user
			}
			return false
		}
		c.users[user.ID] = user
		return false
	}

	if currAccessHash, ok := c.InputPeers.InputUsers[user.ID]; ok {
		if currAccessHash != user.AccessHash {
			c.InputPeers.InputUsers[user.ID] = user.AccessHash
			c.users[user.ID] = user
			return true
		}
		return false
	}

	c.InputPeers.InputUsers[user.ID] = user.AccessHash
	c.users[user.ID] = user
	return true
}

func (c *CACHE) UpdateChannel(channel *Channel) bool {
	c.Lock()
	defer c.Unlock()

	if currAccessHash, ok := c.InputPeers.InputChannels[channel.ID]; ok {
		if activeCh, ok := c.channels[channel.ID]; ok {
			if activeCh.Min {
				c.InputPeers.InputChannels[channel.ID] = channel.AccessHash
				c.channels[channel.ID] = channel
				return true
			}

			if !activeCh.Min && channel.Min {
				return false
			}
		}

		if currAccessHash != channel.AccessHash && !channel.Min {
			c.InputPeers.InputChannels[channel.ID] = channel.AccessHash
			c.channels[channel.ID] = channel
			return true
		}

		return false
	}

	c.channels[channel.ID] = channel
	c.InputPeers.InputChannels[channel.ID] = channel.AccessHash

	return true
}

func (c *CACHE) UpdateChat(chat *ChatObj) bool {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.InputPeers.InputChats[chat.ID]; ok {
		return false
	}

	c.chats[chat.ID] = chat
	c.InputPeers.InputChats[chat.ID] = chat.ID

	return true
}

func (cache *CACHE) UpdatePeersToCache(users []User, chats []Chat) {
	totalUpdates := [2]int{0, 0}

	for _, user := range users {
		if us, ok := user.(*UserObj); ok {
			if updated := cache.UpdateUser(us); updated {
				totalUpdates[0]++
			}
		}
	}

	for _, chat := range chats {
		if ch, ok := chat.(*ChatObj); ok {
			if updated := cache.UpdateChat(ch); updated {
				totalUpdates[1]++
			}
		} else if channel, ok := chat.(*Channel); ok {
			if updated := cache.UpdateChannel(channel); updated {
				totalUpdates[1]++
			}
		}
	}

	if totalUpdates[0] > 0 || totalUpdates[1] > 0 {
		if cache.writeFile {
			go cache.WriteFile() // Write to file asynchronously
		}
		cache.logger.Debug(
			fmt.Sprintf("updated %d users and %d chats to %s (users: %d, chats: %d)",
				totalUpdates[0], totalUpdates[1], cache.fileN,
				len(cache.InputPeers.InputUsers), len(cache.InputPeers.InputChats),
			),
		)
	}
}

func (c *Client) GetPeerUser(userID int64) (*InputPeerUser, error) {
	c.Cache.RLock()
	defer c.Cache.RUnlock()

	if peer, ok := c.Cache.InputPeers.InputUsers[userID]; ok {
		return &InputPeerUser{UserID: userID, AccessHash: peer}, nil
	}
	return nil, fmt.Errorf("no user with id %d or missing from cache", userID)
}

func (c *Client) GetPeerChannel(channelID int64) (*InputPeerChannel, error) {
	c.Cache.RLock()
	defer c.Cache.RUnlock()

	if peer, ok := c.Cache.InputPeers.InputChannels[channelID]; ok {
		return &InputPeerChannel{ChannelID: channelID, AccessHash: peer}, nil
	}
	return nil, fmt.Errorf("no channel with id %d or missing from cache", channelID)
}

func (c *Client) IdInCache(id int64) bool {
	c.Cache.RLock()
	defer c.Cache.RUnlock()

	_, ok := c.Cache.InputPeers.InputUsers[id]
	if ok {
		return true
	}
	_, ok = c.Cache.InputPeers.InputChats[id]
	if ok {
		return true
	}
	_, ok = c.Cache.InputPeers.InputChannels[id]
	return ok
}
