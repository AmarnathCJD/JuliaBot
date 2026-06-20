package modules

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var epicFreeHTTPClient = &http.Client{Timeout: 30 * time.Second}

type epicFreeKeyImage struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type epicFreePageMapping struct {
	PageSlug string `json:"pageSlug"`
	PageType string `json:"pageType"`
}

type epicFreeCatalogNs struct {
	Mappings []epicFreePageMapping `json:"mappings"`
}

type epicFreeDiscountSetting struct {
	DiscountType       string `json:"discountType"`
	DiscountPercentage *int   `json:"discountPercentage"`
}

type epicFreePromotionalOffer struct {
	StartDate       string                  `json:"startDate"`
	EndDate         string                  `json:"endDate"`
	DiscountSetting epicFreeDiscountSetting `json:"discountSetting"`
}

type epicFreePromotionalOfferGroup struct {
	PromotionalOffers []epicFreePromotionalOffer `json:"promotionalOffers"`
}

type epicFreePromotions struct {
	PromotionalOffers         []epicFreePromotionalOfferGroup `json:"promotionalOffers"`
	UpcomingPromotionalOffers []epicFreePromotionalOfferGroup `json:"upcomingPromotionalOffers"`
}

type epicFreePriceFmt struct {
	OriginalPrice  string `json:"originalPrice"`
	DiscountPrice  string `json:"discountPrice"`
}

type epicFreeTotalPrice struct {
	FmtPrice epicFreePriceFmt `json:"fmtPrice"`
}

type epicFreePrice struct {
	TotalPrice epicFreeTotalPrice `json:"totalPrice"`
}

type epicFreeElement struct {
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	OfferType      string                `json:"offerType"`
	ProductSlug    *string               `json:"productSlug"`
	URLSlug        string                `json:"urlSlug"`
	KeyImages      []epicFreeKeyImage    `json:"keyImages"`
	CatalogNs      epicFreeCatalogNs     `json:"catalogNs"`
	OfferMappings  []epicFreePageMapping `json:"offerMappings"`
	Promotions     *epicFreePromotions   `json:"promotions"`
	Price          epicFreePrice         `json:"price"`
}

type epicFreeSearchStore struct {
	Elements []epicFreeElement `json:"elements"`
}

type epicFreeCatalog struct {
	SearchStore epicFreeSearchStore `json:"searchStore"`
}

type epicFreeDataBlock struct {
	Catalog epicFreeCatalog `json:"Catalog"`
}

type epicFreeResponse struct {
	Data epicFreeDataBlock `json:"data"`
}

func epicFreeStoreURL(el epicFreeElement) string {
	if len(el.OfferMappings) > 0 && strings.TrimSpace(el.OfferMappings[0].PageSlug) != "" {
		return "https://store.epicgames.com/en-US/p/" + el.OfferMappings[0].PageSlug
	}
	for _, m := range el.CatalogNs.Mappings {
		if strings.TrimSpace(m.PageSlug) != "" {
			return "https://store.epicgames.com/en-US/p/" + m.PageSlug
		}
	}
	if el.ProductSlug != nil {
		slug := strings.TrimSuffix(strings.TrimSpace(*el.ProductSlug), "/home")
		if slug != "" {
			return "https://store.epicgames.com/en-US/p/" + slug
		}
	}
	return ""
}

func epicFreePickImage(images []epicFreeKeyImage) string {
	priority := []string{"OfferImageWide", "DieselStoreFrontWide", "featuredMedia", "VaultClosed", "OfferImageTall", "Thumbnail"}
	for _, p := range priority {
		for _, k := range images {
			if k.Type == p && strings.TrimSpace(k.URL) != "" {
				return k.URL
			}
		}
	}
	for _, k := range images {
		if strings.TrimSpace(k.URL) != "" {
			return k.URL
		}
	}
	return ""
}

func epicFreeFormatDate(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.UTC().Format("Jan 02, 2006 15:04 UTC")
}

func epicFreeCategorize(el epicFreeElement) (bool, bool, string, string) {
	if el.Promotions == nil {
		return false, false, "", ""
	}
	for _, grp := range el.Promotions.PromotionalOffers {
		for _, o := range grp.PromotionalOffers {
			if o.DiscountSetting.DiscountPercentage != nil && *o.DiscountSetting.DiscountPercentage == 0 {
				return true, false, o.StartDate, o.EndDate
			}
		}
	}
	for _, grp := range el.Promotions.UpcomingPromotionalOffers {
		for _, o := range grp.PromotionalOffers {
			if o.DiscountSetting.DiscountPercentage != nil && *o.DiscountSetting.DiscountPercentage == 0 {
				return false, true, o.StartDate, o.EndDate
			}
		}
	}
	return false, false, "", ""
}

func EpicFreeHandler(m *tg.NewMessage) error {
	status, _ := m.Reply("Fetching Epic Games free promotions...")
	req, err := http.NewRequest(http.MethodGet, "https://store-site-backend-static.ak.epicgames.com/freeGamesPromotions?locale=en-US&country=US", nil)
	if err != nil {
		status.Edit("couldn't build request: " + html.EscapeString(err.Error()))
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 JuliaBot/1.0")
	resp, err := epicFreeHTTPClient.Do(req)
	if err != nil {
		status.Edit("couldn't reach Epic Games: " + html.EscapeString(err.Error()))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		status.Edit(fmt.Sprintf("HTTP %d from Epic Games", resp.StatusCode))
		return nil
	}
	var data epicFreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		status.Edit("couldn't parse Epic response: " + html.EscapeString(err.Error()))
		return nil
	}
	var freeNow []epicFreeElement
	var upcoming []epicFreeElement
	var freeNowDates [][2]string
	var upcomingDates [][2]string
	for _, el := range data.Data.Catalog.SearchStore.Elements {
		isFree, isUpcoming, start, end := epicFreeCategorize(el)
		if isFree {
			freeNow = append(freeNow, el)
			freeNowDates = append(freeNowDates, [2]string{start, end})
		} else if isUpcoming {
			upcoming = append(upcoming, el)
			upcomingDates = append(upcomingDates, [2]string{start, end})
		}
	}
	if len(freeNow) == 0 && len(upcoming) == 0 {
		status.Edit("<b>Epic Games Free Promotions</b>\nNo current or upcoming free games found.")
		return nil
	}
	var b strings.Builder
	b.WriteString("<b>Epic Games - Free Promotions</b>\n")
	if len(freeNow) > 0 {
		b.WriteString("\n<b>Free Now:</b>\n")
		for i, el := range freeNow {
			url := epicFreeStoreURL(el)
			b.WriteString(fmt.Sprintf("\n<b>%d.</b> ", i+1))
			if url != "" {
				b.WriteString("<a href=\"")
				b.WriteString(url)
				b.WriteString("\">")
				b.WriteString(html.EscapeString(strings.TrimSpace(el.Title)))
				b.WriteString("</a>")
			} else {
				b.WriteString(html.EscapeString(strings.TrimSpace(el.Title)))
			}
			desc := strings.TrimSpace(el.Description)
			if desc != "" {
				if len(desc) > 180 {
					desc = desc[:180] + "..."
				}
				b.WriteString("\n  <i>")
				b.WriteString(html.EscapeString(desc))
				b.WriteString("</i>")
			}
			orig := strings.TrimSpace(el.Price.TotalPrice.FmtPrice.OriginalPrice)
			if orig != "" && orig != "0" {
				b.WriteString("\n  Was: <s>")
				b.WriteString(html.EscapeString(orig))
				b.WriteString("</s> - Now: Free")
			}
			end := freeNowDates[i][1]
			if end != "" {
				b.WriteString("\n  Ends: <code>")
				b.WriteString(html.EscapeString(epicFreeFormatDate(end)))
				b.WriteString("</code>")
			}
			b.WriteString("\n")
		}
	}
	if len(upcoming) > 0 {
		b.WriteString("\n<b>Coming Soon:</b>\n")
		for i, el := range upcoming {
			url := epicFreeStoreURL(el)
			b.WriteString(fmt.Sprintf("\n<b>%d.</b> ", i+1))
			if url != "" {
				b.WriteString("<a href=\"")
				b.WriteString(url)
				b.WriteString("\">")
				b.WriteString(html.EscapeString(strings.TrimSpace(el.Title)))
				b.WriteString("</a>")
			} else {
				b.WriteString(html.EscapeString(strings.TrimSpace(el.Title)))
			}
			start := upcomingDates[i][0]
			end := upcomingDates[i][1]
			if start != "" {
				b.WriteString("\n  Starts: <code>")
				b.WriteString(html.EscapeString(epicFreeFormatDate(start)))
				b.WriteString("</code>")
			}
			if end != "" {
				b.WriteString("\n  Ends: <code>")
				b.WriteString(html.EscapeString(epicFreeFormatDate(end)))
				b.WriteString("</code>")
			}
			b.WriteString("\n")
		}
	}
	caption := strings.TrimRight(b.String(), "\n")
	var topImage string
	if len(freeNow) > 0 {
		topImage = epicFreePickImage(freeNow[0].KeyImages)
	} else if len(upcoming) > 0 {
		topImage = epicFreePickImage(upcoming[0].KeyImages)
	}
	if topImage != "" {
		if _, err := m.ReplyMedia(topImage, &tg.MediaOptions{Caption: caption}); err != nil {
			status.Edit(caption, &tg.SendOptions{LinkPreview: false})
			return nil
		}
		status.Delete()
		return nil
	}
	status.Edit(caption, &tg.SendOptions{LinkPreview: false})
	return nil
}

func registerEpicFreeHandlers() {
	c := Client
	c.On("cmd:epicfree", EpicFreeHandler)
}

func init() {
	QueueHandlerRegistration(registerEpicFreeHandlers)
}
