package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var MIDJ_PORT = "8092"
var SELF_PORT = "8091"

// {'type': 'end', 'id': 1346573492694286527, 'content': '**<#4523711711#>half fish half dragon half cat hybrid, retro screencap --ar 2:3 --niji 5** - <@1346561481885356103> (relaxed)', 'attachments': [{'filename': 'amarnathcjd_4523711711half_fish_half_dragon_half_cat_hybrid_ret_1b8d6c03-3b4d-4652-840e-1d3d7cbb32ed.png', 'id': 1346573491775864872, 'proxy_url': 'https://media.discordapp.net/attachments/1346562415915171913/1346573491775864872/amarnathcjd_4523711711half_fish_half_dragon_half_cat_hybrid_ret_1b8d6c03-3b4d-4652-840e-1d3d7cbb32ed.png?ex=67c8adca&is=67c75c4a&hm=478844ce13f6d79ac3bcc72de4c321df3cd9c03c83274929304b79e8121c4171&', 'size': 8228789, 'url': 'https://cdn.discordapp.com/attachments/1346562415915171913/1346573491775864872/amarnathcjd_4523711711half_fish_half_dragon_half_cat_hybrid_ret_1b8d6c03-3b4d-4652-840e-1d3d7cbb32ed.png?ex=67c8adca&is=67c75c4a&hm=478844ce13f6d79ac3bcc72de4c321df3cd9c03c83274929304b79e8121c4171&', 'spoiler': False, 'height': 2688, 'width': 1792, 'content_type': 'image/png'}], 'embeds': [], 'trigger_id': '4523711711'}

func init() {
	http.HandleFunc("/midj", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var resp struct {
			Type        string `json:"type"`
			TriggerID   string `json:"trigger_id"`
			Content     string `json:"content"`
			Attachments []struct {
				ProxyURL string `json:"proxy_url"`
			} `json:"attachments"`
		}

		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			panic(err)
		} else {
			fmt.Println(resp)
		}
		if resp.Type == "end" {
			if j, ok := trigs[resp.TriggerID]; ok {
				picbytes, _ := http.Get(resp.Attachments[0].ProxyURL)
				rd, _ := io.ReadAll(picbytes.Body)
				os.WriteFile("midj.png", rd, 0644)

				j.msg.ReplyMedia("midj.png", tg.MediaOptions{Caption: resp.Content})
				delete(trigs, resp.TriggerID)
			}
		} else {
			if j, ok := trigs[resp.TriggerID]; ok {
				j.msg.Edit(resp.Content)
			}
		}
	})

	go http.ListenAndServe(":"+SELF_PORT, nil)
}

type JT struct {
	TriggerID string
	msg       *tg.NewMessage
}

var trigs = make(map[string]JT)

type RequestData struct {
	Prompt string `json:"prompt"`
}

func MidjHandler(m *tg.NewMessage) error {
	m.Reply("Please wait... (soon™)")
	return nil
	if m.Args() == "" {
		m.Reply("Usage: /midj <trigger> <message>")
		return nil
	}
	url := "http://127.0.0.1:8092/v1/api/trigger/imagine"
	//
	//args := m.Args()

	data := RequestData{
		Prompt: m.Args(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)

	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	respx, err := client.Do(req)
	if err != nil {
		m.Reply("Failed to fetch the URL")
		return nil
	}

	if respx.StatusCode != 200 {
		b, _ := io.ReadAll(respx.Body)
		fmt.Println(string(b))
	}

	var resp struct {
		TriggerID string `json:"trigger_id"`
	}

	if err := json.NewDecoder(respx.Body).Decode(&resp); err != nil {
		fmt.Println("Error decoding JSON:", err)
		b, _ := io.ReadAll(respx.Body)
		fmt.Println(string(b))
	}
	msg, _ := m.Reply("Generating...")

	trigs[resp.TriggerID] = JT{TriggerID: resp.TriggerID, msg: msg}

	return nil
}
