package extras

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	modules "main/modules"
)

type randomUserName struct {
	Title string `json:"title"`
	First string `json:"first"`
	Last  string `json:"last"`
}

type randomUserStreet struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

type randomUserLocation struct {
	Street  randomUserStreet `json:"street"`
	City    string           `json:"city"`
	State   string           `json:"state"`
	Country string           `json:"country"`
}

type randomUserDob struct {
	Date string `json:"date"`
	Age  int    `json:"age"`
}

type randomUserPicture struct {
	Large     string `json:"large"`
	Medium    string `json:"medium"`
	Thumbnail string `json:"thumbnail"`
}

type randomUserLogin struct {
	Username string `json:"username"`
}

type randomUserResult struct {
	Gender   string             `json:"gender"`
	Name     randomUserName     `json:"name"`
	Location randomUserLocation `json:"location"`
	Email    string             `json:"email"`
	Login    randomUserLogin    `json:"login"`
	Dob      randomUserDob      `json:"dob"`
	Phone    string             `json:"phone"`
	Cell     string             `json:"cell"`
	Picture  randomUserPicture  `json:"picture"`
	Nat      string             `json:"nat"`
}

type randomUserResponse struct {
	Results []randomUserResult `json:"results"`
}

func RandUserHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("generating random user...")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://randomuser.me/api/")
	if err != nil {
		if status != nil {
			status.Edit("failed to fetch: " + html.EscapeString(err.Error()))
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if status != nil {
			status.Edit(fmt.Sprintf("api returned status %d", resp.StatusCode))
		}
		return nil
	}

	var data randomUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		if status != nil {
			status.Edit("failed to parse response")
		}
		return nil
	}

	if len(data.Results) == 0 {
		if status != nil {
			status.Edit("no user returned")
		}
		return nil
	}

	u := data.Results[0]

	fullName := strings.TrimSpace(fmt.Sprintf("%s %s %s", u.Name.Title, u.Name.First, u.Name.Last))
	if fullName == "" {
		fullName = "Unknown"
	}

	locParts := []string{}
	street := strings.TrimSpace(fmt.Sprintf("%d %s", u.Location.Street.Number, u.Location.Street.Name))
	if street != "" {
		locParts = append(locParts, street)
	}
	if u.Location.City != "" {
		locParts = append(locParts, u.Location.City)
	}
	if u.Location.State != "" {
		locParts = append(locParts, u.Location.State)
	}
	if u.Location.Country != "" {
		locParts = append(locParts, u.Location.Country)
	}
	location := strings.Join(locParts, ", ")
	if location == "" {
		location = "Unknown"
	}

	gender := u.Gender
	if gender == "" {
		gender = "n/a"
	}

	phone := u.Phone
	if phone == "" {
		phone = u.Cell
	}
	if phone == "" {
		phone = "n/a"
	}

	username := u.Login.Username
	if username == "" {
		username = "n/a"
	}

	email := u.Email
	if email == "" {
		email = "n/a"
	}

	nat := u.Nat
	if nat == "" {
		nat = "n/a"
	}

	caption := fmt.Sprintf(
		"<b>Random User</b>\n\n"+
			"<b>Name:</b> %s\n"+
			"<b>Gender:</b> %s\n"+
			"<b>Age:</b> %d\n"+
			"<b>Email:</b> <code>%s</code>\n"+
			"<b>Username:</b> <code>%s</code>\n"+
			"<b>Phone:</b> <code>%s</code>\n"+
			"<b>Nationality:</b> %s\n"+
			"<b>Location:</b> %s\n\n"+
			"<i>fake profile from randomuser.me</i>",
		html.EscapeString(fullName),
		html.EscapeString(gender),
		u.Dob.Age,
		html.EscapeString(email),
		html.EscapeString(username),
		html.EscapeString(phone),
		html.EscapeString(strings.ToUpper(nat)),
		html.EscapeString(location),
	)

	photoURL := u.Picture.Large
	if photoURL == "" {
		photoURL = u.Picture.Medium
	}
	if photoURL == "" {
		photoURL = u.Picture.Thumbnail
	}

	if photoURL != "" {
		if _, err := m.ReplyMedia(photoURL, &tg.MediaOptions{Caption: caption}); err != nil {
			m.Reply(caption + "\n\n<a href=\"" + html.EscapeString(photoURL) + "\">photo</a>")
		}
	} else {
		m.Reply(caption)
	}

	if status != nil {
		status.Delete()
	}
	return nil
}

func registerRandomUserHandlers() {
	c := modules.Client
	c.On("cmd:randuser", RandUserHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerRandomUserHandlers)
}
