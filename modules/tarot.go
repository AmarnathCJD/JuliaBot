package modules

import (
	"fmt"
	"html"
	"math/rand"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var tarotRng = rand.New(rand.NewSource(time.Now().UnixNano()))

type tarotCard struct {
	Name     string
	Upright  string
	Reversed string
}

var majorArcana = []tarotCard{
	{"The Fool", "New beginnings, innocence, spontaneity, a free spirit.", "Recklessness, taken advantage of, inconsideration."},
	{"The Magician", "Manifestation, resourcefulness, power, inspired action.", "Manipulation, poor planning, untapped talents."},
	{"The High Priestess", "Intuition, sacred knowledge, divine feminine, subconscious mind.", "Secrets, disconnected from intuition, withdrawal and silence."},
	{"The Empress", "Femininity, beauty, nature, nurturing, abundance.", "Creative block, dependence on others, smothering."},
	{"The Emperor", "Authority, establishment, structure, a father figure.", "Domination, excessive control, rigidity, inflexibility."},
	{"The Hierophant", "Spiritual wisdom, religious beliefs, conformity, tradition.", "Personal beliefs, freedom, challenging the status quo."},
	{"The Lovers", "Love, harmony, relationships, values alignment, choices.", "Disharmony, imbalance, misalignment of values."},
	{"The Chariot", "Control, willpower, success, action, determination.", "Self-discipline, opposition, lack of direction."},
	{"Strength", "Courage, persuasion, influence, compassion, inner strength.", "Inner strength, self-doubt, low energy, raw emotion."},
	{"The Hermit", "Soul-searching, introspection, being alone, inner guidance.", "Isolation, loneliness, withdrawal, paranoia."},
	{"Wheel of Fortune", "Good luck, karma, life cycles, destiny, a turning point.", "Bad luck, resistance to change, breaking cycles."},
	{"Justice", "Justice, fairness, truth, cause and effect, law.", "Unfairness, lack of accountability, dishonesty."},
	{"The Hanged Man", "Pause, surrender, letting go, new perspectives.", "Delays, resistance, stalling, indecision."},
	{"Death", "Endings, change, transformation, transition, rebirth.", "Resistance to change, personal transformation, inner purging."},
	{"Temperance", "Balance, moderation, patience, purpose, meaning.", "Imbalance, excess, self-healing, re-alignment."},
	{"The Devil", "Shadow self, attachment, addiction, restriction, sexuality.", "Releasing limiting beliefs, exploring dark thoughts, detachment."},
	{"The Tower", "Sudden change, upheaval, chaos, revelation, awakening.", "Personal transformation, fear of change, averting disaster."},
	{"The Star", "Hope, faith, purpose, renewal, spirituality.", "Lack of faith, despair, self-trust, disconnection."},
	{"The Moon", "Illusion, fear, anxiety, subconscious, intuition.", "Release of fear, repressed emotion, inner confusion."},
	{"The Sun", "Positivity, fun, warmth, success, vitality, joy.", "Inner child, feeling down, overly optimistic."},
	{"Judgement", "Judgement, rebirth, inner calling, absolution.", "Self-doubt, inner critic, ignoring the call."},
	{"The World", "Completion, integration, accomplishment, travel, fulfillment.", "Seeking personal closure, short-cuts, delays."},
}

var tarotPositions = []string{"Past", "Present", "Future"}

func TarotHandler(m *tg.NewMessage) error {
	indices := tarotRng.Perm(len(majorArcana))[:3]
	var sb strings.Builder
	sb.WriteString("<b>Tarot Reading - Past / Present / Future</b>\n\n")
	for i, idx := range indices {
		card := majorArcana[idx]
		reversed := tarotRng.Intn(100) < 30
		orientation := "Upright"
		meaning := card.Upright
		if reversed {
			orientation = "Reversed"
			meaning = card.Reversed
		}
		sb.WriteString(fmt.Sprintf("<b>%s:</b> <i>%s</i> <code>(%s)</code>\n<blockquote>%s</blockquote>\n\n",
			html.EscapeString(tarotPositions[i]),
			html.EscapeString(card.Name),
			orientation,
			html.EscapeString(meaning),
		))
	}
	sb.WriteString("<i>The cards have spoken. Reflect on their wisdom.</i>")
	m.Reply(sb.String())
	return nil
}

func registerTarotHandlers() {
	c := Client
	c.On("cmd:tarot", TarotHandler)
}

func init() {
	QueueHandlerRegistration(registerTarotHandlers)
}
