package modules

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const tttCallbackPrefix = "ttt:"

const (
	tttEmpty = 0
	tttX     = 1
	tttO     = 2
)

type tttPlayer struct {
	ID   int64
	Name string
	Bot  bool
}

type tttGame struct {
	ChatID    int64
	MessageID int32
	Board     [9]int
	X         tttPlayer
	O         tttPlayer
	Turn      int
	Finished  bool
	Winner    int
	WinLine   [3]int
	SoloVsBot bool
}

var (
	tttGames sync.Map
	tttMu    sync.Mutex
)

func tttSymbol(v int) string {
	switch v {
	case tttX:
		return "X"
	case tttO:
		return "O"
	default:
		return " "
	}
}

func tttCellLabel(v int) string {
	switch v {
	case tttX:
		return "X"
	case tttO:
		return "O"
	default:
		return "·"
	}
}

func tttMention(p tttPlayer) string {
	if p.Bot {
		return "<b>Bot</b>"
	}
	name := p.Name
	if name == "" {
		name = "Player"
	}
	if p.ID == 0 {
		return html.EscapeString(name)
	}
	return fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", p.ID, html.EscapeString(name))
}

func tttPlayerName(m *tg.NewMessage) string {
	if m.Sender != nil {
		name := strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
		if name != "" {
			return name
		}
		if m.Sender.Username != "" {
			return m.Sender.Username
		}
	}
	return "Player"
}

func tttResolveOpponent(m *tg.NewMessage, raw string) (tttPlayer, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "@")
	if raw == "" {
		return tttPlayer{}, false
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if u, uerr := m.Client.GetUser(n); uerr == nil && u != nil {
			name := u.FirstName
			if name == "" {
				name = u.Username
			}
			if name == "" {
				name = "Player"
			}
			return tttPlayer{ID: n, Name: name}, true
		}
		return tttPlayer{ID: n, Name: "Player"}, true
	}
	peer, err := m.Client.ResolvePeer(raw)
	if err != nil || peer == nil {
		return tttPlayer{}, false
	}
	id := m.Client.GetPeerID(peer)
	name := raw
	if u, uerr := m.Client.GetUser(id); uerr == nil && u != nil {
		if u.FirstName != "" {
			name = u.FirstName
		} else if u.Username != "" {
			name = u.Username
		}
	}
	return tttPlayer{ID: id, Name: name}, true
}

var tttWinLines = [8][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
	{0, 4, 8}, {2, 4, 6},
}

func tttCheckWinner(board [9]int) (int, [3]int, bool) {
	for _, line := range tttWinLines {
		a, b, c := board[line[0]], board[line[1]], board[line[2]]
		if a != tttEmpty && a == b && b == c {
			return a, line, true
		}
	}
	return tttEmpty, [3]int{}, false
}

func tttIsDraw(board [9]int) bool {
	for _, v := range board {
		if v == tttEmpty {
			return false
		}
	}
	return true
}

func tttCurrentPlayer(g *tttGame) tttPlayer {
	if g.Turn == tttX {
		return g.X
	}
	return g.O
}

func tttRenderText(g *tttGame) string {
	var sb strings.Builder
	sb.WriteString("<b>Tic-Tac-Toe</b>\n\n")
	sb.WriteString("X: " + tttMention(g.X) + "\n")
	sb.WriteString("O: " + tttMention(g.O) + "\n\n")
	if g.Finished {
		if g.Winner == tttEmpty {
			sb.WriteString("<b>Draw!</b> Nobody won.")
		} else {
			winner := g.X
			if g.Winner == tttO {
				winner = g.O
			}
			sb.WriteString(fmt.Sprintf("<b>%s wins!</b> (%s)", tttMention(winner), tttSymbol(g.Winner)))
		}
	} else {
		cur := tttCurrentPlayer(g)
		sb.WriteString(fmt.Sprintf("Turn: %s (%s)", tttMention(cur), tttSymbol(g.Turn)))
	}
	return sb.String()
}

func tttCellHighlighted(g *tttGame, idx int) bool {
	if !g.Finished || g.Winner == tttEmpty {
		return false
	}
	for _, i := range g.WinLine {
		if i == idx {
			return true
		}
	}
	return false
}

func tttBuildKeyboard(g *tttGame) *tg.ReplyInlineMarkup {
	b := tg.Button
	kb := tg.NewKeyboard()
	for r := 0; r < 3; r++ {
		row := []tg.KeyboardButton{}
		for c := 0; c < 3; c++ {
			idx := r*3 + c
			label := tttCellLabel(g.Board[idx])
			if tttCellHighlighted(g, idx) {
				label = "[" + tttSymbol(g.Board[idx]) + "]"
			}
			data := fmt.Sprintf("%smove:%d:%d", tttCallbackPrefix, g.ChatID, idx)
			row = append(row, b.Data(label, data))
		}
		kb.AddRow(row...)
	}
	if g.Finished {
		kb.AddRow(b.Data("New Game", fmt.Sprintf("%snew:%d", tttCallbackPrefix, g.ChatID)))
	} else {
		kb.AddRow(b.Data("Resign", fmt.Sprintf("%sresign:%d", tttCallbackPrefix, g.ChatID)))
	}
	return kb.Build()
}

func tttMinimax(board [9]int, player int, depth int) (int, int) {
	if w, _, ok := tttCheckWinner(board); ok {
		if w == tttO {
			return 10 - depth, -1
		}
		return depth - 10, -1
	}
	if tttIsDraw(board) {
		return 0, -1
	}

	bestMove := -1
	if player == tttO {
		bestScore := -1000
		for i := 0; i < 9; i++ {
			if board[i] != tttEmpty {
				continue
			}
			board[i] = tttO
			score, _ := tttMinimax(board, tttX, depth+1)
			board[i] = tttEmpty
			if score > bestScore {
				bestScore = score
				bestMove = i
			}
		}
		return bestScore, bestMove
	}
	bestScore := 1000
	for i := 0; i < 9; i++ {
		if board[i] != tttEmpty {
			continue
		}
		board[i] = tttX
		score, _ := tttMinimax(board, tttO, depth+1)
		board[i] = tttEmpty
		if score < bestScore {
			bestScore = score
			bestMove = i
		}
	}
	return bestScore, bestMove
}

func tttBotMove(g *tttGame) int {
	_, move := tttMinimax(g.Board, tttO, 0)
	return move
}

func tttApplyMove(g *tttGame, idx int) {
	g.Board[idx] = g.Turn
	if w, line, ok := tttCheckWinner(g.Board); ok {
		g.Finished = true
		g.Winner = w
		g.WinLine = line
		return
	}
	if tttIsDraw(g.Board) {
		g.Finished = true
		g.Winner = tttEmpty
		return
	}
	if g.Turn == tttX {
		g.Turn = tttO
	} else {
		g.Turn = tttX
	}
}

func tttPlayerCanMove(g *tttGame, userID int64) bool {
	if g.Finished {
		return false
	}
	cur := tttCurrentPlayer(g)
	if cur.Bot {
		return false
	}
	return cur.ID == userID
}

func tttIsParticipant(g *tttGame, userID int64) bool {
	if g.X.ID == userID && !g.X.Bot {
		return true
	}
	if g.O.ID == userID && !g.O.Bot {
		return true
	}
	return false
}

func tttSendOrEdit(m *tg.NewMessage, g *tttGame) {
	text := tttRenderText(g)
	kb := tttBuildKeyboard(g)
	sent, err := m.Reply(text, &tg.SendOptions{ReplyMarkup: kb})
	if err != nil {
		return
	}
	if sent != nil {
		g.MessageID = int32(sent.ID)
	}
}

func TttHandler(m *tg.NewMessage) error {
	chatID := m.ChatID()

	tttMu.Lock()
	if _, exists := tttGames.Load(chatID); exists {
		tttMu.Unlock()
		m.Reply("A Tic-Tac-Toe game is already running in this chat. Resign or finish it first.")
		return nil
	}

	challenger := tttPlayer{ID: m.SenderID(), Name: tttPlayerName(m)}
	args := strings.TrimSpace(m.Args())

	var opponent tttPlayer
	soloVsBot := false

	if args == "" {
		opponent = tttPlayer{ID: 0, Name: "Bot", Bot: true}
		soloVsBot = true
	} else {
		opp, ok := tttResolveOpponent(m, args)
		if !ok {
			tttMu.Unlock()
			m.Reply("Could not resolve that user. Use <code>/ttt</code> for solo vs bot or <code>/ttt @user</code> to challenge someone.")
			return nil
		}
		if opp.ID == challenger.ID {
			tttMu.Unlock()
			m.Reply("You can't challenge yourself. Try <code>/ttt</code> for solo vs bot.")
			return nil
		}
		opponent = opp
	}

	g := &tttGame{
		ChatID:    chatID,
		X:         challenger,
		O:         opponent,
		Turn:      tttX,
		SoloVsBot: soloVsBot,
	}
	tttGames.Store(chatID, g)
	tttMu.Unlock()

	tttSendOrEdit(m, g)
	return nil
}

func tttEditFromCallback(c *tg.CallbackQuery, g *tttGame) {
	text := tttRenderText(g)
	kb := tttBuildKeyboard(g)
	c.Edit(text, &tg.SendOptions{ReplyMarkup: kb})
}

func TttCallbackHandler(c *tg.CallbackQuery) error {
	data := c.DataString()
	if !strings.HasPrefix(data, tttCallbackPrefix) {
		return nil
	}
	body := strings.TrimPrefix(data, tttCallbackPrefix)
	parts := strings.Split(body, ":")
	if len(parts) < 2 {
		c.Answer("Invalid data.", &tg.CallbackOptions{Alert: true})
		return nil
	}
	action := parts[0]
	chatID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		c.Answer("Invalid chat.", &tg.CallbackOptions{Alert: true})
		return nil
	}

	switch action {
	case "move":
		if len(parts) < 3 {
			c.Answer("Invalid move.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		idx, err := strconv.Atoi(parts[2])
		if err != nil || idx < 0 || idx > 8 {
			c.Answer("Invalid cell.", &tg.CallbackOptions{Alert: true})
			return nil
		}

		tttMu.Lock()
		v, ok := tttGames.Load(chatID)
		if !ok {
			tttMu.Unlock()
			c.Answer("No active game in this chat.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		g := v.(*tttGame)
		if g.Finished {
			tttMu.Unlock()
			c.Answer("Game is over.", &tg.CallbackOptions{Alert: false})
			return nil
		}
		if !tttIsParticipant(g, c.SenderID) {
			tttMu.Unlock()
			c.Answer("You're not a player in this game.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		if !tttPlayerCanMove(g, c.SenderID) {
			tttMu.Unlock()
			c.Answer("It's not your turn.", &tg.CallbackOptions{Alert: false})
			return nil
		}
		if g.Board[idx] != tttEmpty {
			tttMu.Unlock()
			c.Answer("Cell already taken.", &tg.CallbackOptions{Alert: false})
			return nil
		}

		tttApplyMove(g, idx)

		if !g.Finished && g.SoloVsBot && tttCurrentPlayer(g).Bot {
			move := tttBotMove(g)
			if move >= 0 {
				tttApplyMove(g, move)
			}
		}

		if g.Finished {
			tttGames.Delete(chatID)
		}
		tttMu.Unlock()

		tttEditFromCallback(c, g)
		c.Answer("", &tg.CallbackOptions{Alert: false})
	case "resign":
		tttMu.Lock()
		v, ok := tttGames.Load(chatID)
		if !ok {
			tttMu.Unlock()
			c.Answer("No active game.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		g := v.(*tttGame)
		if g.Finished {
			tttMu.Unlock()
			c.Answer("Game already over.", &tg.CallbackOptions{Alert: false})
			return nil
		}
		if !tttIsParticipant(g, c.SenderID) {
			tttMu.Unlock()
			c.Answer("You're not in this game.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		g.Finished = true
		if g.X.ID == c.SenderID {
			g.Winner = tttO
		} else {
			g.Winner = tttX
		}
		tttGames.Delete(chatID)
		tttMu.Unlock()

		tttEditFromCallback(c, g)
		c.Answer("Resigned.", &tg.CallbackOptions{Alert: false})
	case "new":
		tttMu.Lock()
		if _, ok := tttGames.Load(chatID); ok {
			tttMu.Unlock()
			c.Answer("A game is already running.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		tttMu.Unlock()
		c.Answer("Send /ttt to start a new game.", &tg.CallbackOptions{Alert: false})
	default:
		c.Answer("Unknown action.", &tg.CallbackOptions{Alert: true})
	}
	return nil
}

func registerTicTacToeHandlers() {
	c := Client
	c.On("cmd:ttt", TttHandler)
	c.On("callback:ttt:", TttCallbackHandler)
}

func init() {
	QueueHandlerRegistration(registerTicTacToeHandlers)

	Mods.AddModule("TicTacToe", `<b>Tic-Tac-Toe Module</b>

Play classic 3x3 tic-tac-toe on an inline keyboard grid.

<b>Commands:</b>
 - /ttt - start a solo game vs the bot (minimax, unbeatable)
 - /ttt @user - challenge another user to a match

<b>Notes:</b>
 - The challenger plays X and moves first; opponent plays O.
 - Tap any empty cell on the keyboard to make a move.
 - Resign at any time with the Resign button.
 - Only one active game per chat.`)
}
