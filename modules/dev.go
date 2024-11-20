package modules

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func ShellHandle(m *telegram.NewMessage) error {
	cmd := m.Args()
	var cmd_args []string
	if cmd == "" {
		m.Reply("No command provided")
		return nil
	}

	if runtime.GOOS == "windows" {
		cmd = "cmd"
		cmd_args_b := strings.Split(m.Args(), " ")
		cmd_args = []string{"/C"}
		cmd_args = append(cmd_args, cmd_args_b...)
	} else {
		cmd = strings.Split(cmd, " ")[0]
		cmd_args = strings.Split(m.Args(), " ")
		cmd_args = append(cmd_args[:0], cmd_args[1:]...)
	}
	cmx := exec.Command(cmd, cmd_args...)
	var out bytes.Buffer
	cmx.Stdout = &out
	var errx bytes.Buffer
	cmx.Stderr = &errx
	_ = cmx.Run()

	if errx.String() == "" && out.String() == "" {
		m.Reply("<code>No Output</code>")
		return nil
	}

	if out.String() != "" {
		m.Reply(`<pre lang="bash">` + strings.TrimSpace(out.String()) + `</pre>`)
	} else {
		m.Reply(`<pre lang="bash">` + strings.TrimSpace(errx.String()) + `</pre>`)
	}
	return nil
}

// --------- Eval function ------------

const boiler_code_for_eval = `
package main

import "fmt"
import "github.com/amarnathcjd/gogram/telegram"
import "encoding/json"

var msg_id int32 = %d

var client *telegram.Client
var message *telegram.NewMessage
var m *telegram.NewMessage
` + "var msg = `%s`\nvar snd = `%s`\nvar cht = `%s`\nvar chn = `%s`" + `


func evalCode() {
	%s
}

func main() {
	var msg_o *telegram.MessageObj
	var snd_o *telegram.UserObj
	var cht_o *telegram.ChatObj
	var chn_o *telegram.Channel
	json.Unmarshal([]byte(msg), &msg_o)
	json.Unmarshal([]byte(snd), &snd_o)
	json.Unmarshal([]byte(cht), &cht_o)
	json.Unmarshal([]byte(chn), &chn_o)
	client, _ = telegram.NewClient(telegram.ClientConfig{
		StringSession: "%s",
	})

	client.Conn()

	x := []telegram.User{}
	y := []telegram.Chat{}
	x = append(x, snd_o)
	if chn_o != nil {
		y = append(y, chn_o)
	}
	if cht_o != nil {
		y = append(y, cht_o)
	}
	client.Cache.UpdatePeersToCache(x, y)
	idx := 0
	if cht_o != nil {
		idx = int(cht_o.ID)
	}
	if chn_o != nil {
		idx = int(chn_o.ID)
	}
	if snd_o != nil && idx == 0 {
		idx = int(snd_o.ID)
	}

	messageX, err := client.GetMessages(idx, &telegram.SearchOption{
		IDs: int(msg_id),
	})

	if err != nil {
		fmt.Println(err)
	}

	message = &messageX[0]
	m = message

	fmt.Println("output-start")
	evalCode()
}

func packMessage(c *telegram.Client, message telegram.Message, sender *telegram.UserObj, channel *telegram.Channel, chat *telegram.ChatObj) *telegram.NewMessage {
	var (
		m = &telegram.NewMessage{}
	)
	switch message := message.(type) {
	case *telegram.MessageObj:
		m.ID = message.ID
		m.OriginalUpdate = message
		m.Message = message
		m.Client = c
	default:
		return nil
	}
	m.Sender = sender
	m.Chat = chat
	m.Channel = channel
	if m.Channel != nil && (m.Sender.ID == m.Channel.ID) {
		m.SenderChat = channel
	} else {
		m.SenderChat = &telegram.Channel{}
	}
	m.Peer, _ = c.GetSendablePeer(message.(*telegram.MessageObj).PeerID)

	/*if m.IsMedia() {
		FileID := telegram.PackBotFileID(m.Media())
		m.File = &telegram.CustomFile{
			FileID: FileID,
			Name:   getFileName(m.Media()),
			Size:   getFileSize(m.Media()),
			Ext:    getFileExt(m.Media()),
		}
	}*/
	return m
}
`

func EvalHandle(m *telegram.NewMessage) error {
	code := m.Args()
	if code == "" {
		return nil
	}

	resp := perfomEval(code, m)
	resp = strings.TrimSpace(resp)
	resp = fmt.Sprintf("<pre lang='go'>%s</pre>", resp)

	if resp != "" {
		if _, err := m.Reply(resp); err != nil {
			fmt.Println(err)
		}
	}
	return nil
}

func perfomEval(code string, m *telegram.NewMessage) string {
	msg_b, _ := json.Marshal(m.Message)
	snd_b, _ := json.Marshal(m.Sender)
	cnt_b, _ := json.Marshal(m.Chat)
	chn_b, _ := json.Marshal(m.Channel)
	code_file := fmt.Sprintf(boiler_code_for_eval, m.ID, msg_b, snd_b, cnt_b, chn_b, code, m.Client.ExportSession())
	tmp_dir := "tmp"
	_, err := os.ReadDir(tmp_dir)
	if err != nil {
		err = os.Mkdir(tmp_dir, 0755)
		if err != nil {
			fmt.Println(err)
		}
	}

	defer os.Remove(tmp_dir + "/eval.go")
	defer os.Remove(tmp_dir + "/cache.db")
	defer os.Remove(tmp_dir)

	// copy cache.db file to tmp
	_, err = os.Stat("cache.db")
	if err == nil {
		f, _ := os.ReadFile("cache.db")
		_ = os.WriteFile(tmp_dir+"/cache.db", f, 0644)
	}

	os.WriteFile(tmp_dir+"/eval.go", []byte(code_file), 0644)
	cmd := exec.Command("go", "run", "tmp/eval.go")
	out, err := cmd.CombinedOutput()
	if err != nil {
		errx := fmt.Sprintf("Error: %s\nOutput: <code>%s</code>", err, strings.Split(string(out), "tmp\\eval.go:")[1])
		return strings.TrimSpace(errx)
	}
	outN := strings.Split(string(out), "output-start")

	if len(outN) > 1 {
		return strings.TrimSpace(outN[1])
	}

	return "No Output."
}

func JsonHandle(m *telegram.NewMessage) error {
	var jsonString []byte
	if !m.IsReply() {
		if strings.Contains(m.Args(), "-s") {
			jsonString, _ = json.MarshalIndent(m.Sender, "", "  ")
		} else if strings.Contains(m.Args(), "-m") {
			jsonString, _ = json.MarshalIndent(m.Media(), "", "  ")
		} else if strings.Contains(m.Args(), "-c") {
			jsonString, _ = json.MarshalIndent(m.Channel, "", "  ")
		} else {
			jsonString, _ = json.MarshalIndent(m.OriginalUpdate, "", "  ")
		}
	} else {
		r, err := m.GetReplyMessage()
		if err != nil {
			m.Reply("<code>Error:</code> <b>" + err.Error() + "</b>")
			return nil
		}
		if strings.Contains(m.Args(), "-s") {
			jsonString, _ = json.MarshalIndent(r.Sender, "", "  ")
		} else if strings.Contains(m.Args(), "-m") {
			jsonString, _ = json.MarshalIndent(r.Media(), "", "  ")
		} else if strings.Contains(m.Args(), "-c") {
			jsonString, _ = json.MarshalIndent(r.Channel, "", "  ")
		} else {
			jsonString, _ = json.MarshalIndent(r.OriginalUpdate, "", "  ")
		}
	}

	// find all "Data": "<base64>" and decode and replace with actual data
	dataFieldRegex := regexp.MustCompile(`"Data": "([a-zA-Z0-9+/]+={0,2})"`)
	dataFields := dataFieldRegex.FindAllStringSubmatch(string(jsonString), -1)
	for _, v := range dataFields {
		decoded, err := base64.StdEncoding.DecodeString(v[1])
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}
		jsonString = []byte(strings.ReplaceAll(string(jsonString), v[0], `"Data": "`+string(decoded)+`"`))
	}

	if len(jsonString) > 4095 {
		defer os.Remove("message.json")
		tmpFile, err := os.Create("message.json")
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		_, err = tmpFile.Write(jsonString)
		if err != nil {
			m.Reply("Error: " + err.Error())
			return nil
		}

		_, err = m.ReplyMedia(tmpFile.Name(), telegram.MediaOptions{Caption: "Message JSON"})
		if err != nil {
			m.Reply("Error: " + err.Error())
		}
	} else {
		m.Reply("<pre lang='json'>" + string(jsonString) + "</pre>")
	}

	return nil
}

func init() {
	Mods.AddModule("Dev", `<b>Here are the commands available in Dev module:</b>

- <code>/sh &lt;command&gt;</code> - Execute shell commands
- <code>/eval &lt;code&gt;</code> - Evaluate Go code
- <code>/json [-s | -m | -c] &lt;message&gt;</code> - Get JSON of a message`)
}
