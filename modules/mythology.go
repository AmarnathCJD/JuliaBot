package modules

import (
	"html"
	"sort"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type mythDeity struct {
	Name       string
	Pantheon   string
	Domain     string
	Parents    string
	Symbols    string
	Attributes string
	Aliases    []string
}

var mythDeities = []mythDeity{
	{Name: "Zeus", Pantheon: "Greek", Domain: "Sky, thunder, king of the gods", Parents: "Cronus and Rhea", Symbols: "Thunderbolt, eagle, oak, scepter", Attributes: "Supreme ruler of Mount Olympus, wielder of the thunderbolt and enforcer of divine order.", Aliases: []string{"jupiter"}},
	{Name: "Hera", Pantheon: "Greek", Domain: "Marriage, women, family, childbirth", Parents: "Cronus and Rhea", Symbols: "Peacock, cow, diadem, pomegranate", Attributes: "Queen of the gods and protector of marriage, known for her jealousy toward Zeus' lovers.", Aliases: []string{"juno"}},
	{Name: "Poseidon", Pantheon: "Greek", Domain: "Sea, earthquakes, horses", Parents: "Cronus and Rhea", Symbols: "Trident, horse, dolphin, bull", Attributes: "God of the sea who shakes the earth, brother of Zeus and Hades.", Aliases: []string{"neptune"}},
	{Name: "Hades", Pantheon: "Greek", Domain: "Underworld, wealth, the dead", Parents: "Cronus and Rhea", Symbols: "Cerberus, bident, cypress, helm of darkness", Attributes: "Lord of the underworld and ruler of the dead, husband of Persephone.", Aliases: []string{"pluto", "aides"}},
	{Name: "Demeter", Pantheon: "Greek", Domain: "Harvest, agriculture, grain, fertility", Parents: "Cronus and Rhea", Symbols: "Wheat, torch, cornucopia, poppy", Attributes: "Goddess of the harvest who governs the seasons through her grief for Persephone.", Aliases: []string{"ceres"}},
	{Name: "Athena", Pantheon: "Greek", Domain: "Wisdom, warfare, crafts, strategy", Parents: "Zeus (born from his head) and Metis", Symbols: "Owl, olive tree, aegis, spear", Attributes: "Virgin goddess of wisdom and just war, patron of Athens.", Aliases: []string{"minerva", "pallas"}},
	{Name: "Apollo", Pantheon: "Greek", Domain: "Sun, music, prophecy, healing, archery", Parents: "Zeus and Leto", Symbols: "Lyre, laurel, bow, sun chariot", Attributes: "Radiant god of light and the arts, twin brother of Artemis and oracle of Delphi.", Aliases: []string{"phoebus"}},
	{Name: "Artemis", Pantheon: "Greek", Domain: "Hunt, wilderness, moon, chastity", Parents: "Zeus and Leto", Symbols: "Bow, deer, moon, cypress", Attributes: "Virgin huntress of the wild and protector of young women and animals.", Aliases: []string{"diana"}},
	{Name: "Ares", Pantheon: "Greek", Domain: "War, violence, bloodlust", Parents: "Zeus and Hera", Symbols: "Spear, helmet, vulture, dog", Attributes: "Brutal god of war, lover of Aphrodite and feared by mortals and gods alike.", Aliases: []string{"mars"}},
	{Name: "Aphrodite", Pantheon: "Greek", Domain: "Love, beauty, desire, pleasure", Parents: "Born from sea foam near Cyprus (from Uranus)", Symbols: "Dove, rose, myrtle, scallop shell", Attributes: "Goddess of love and beauty whose charms enchant even the gods.", Aliases: []string{"venus", "cytherea"}},
	{Name: "Hephaestus", Pantheon: "Greek", Domain: "Fire, forge, metalwork, craftsmen", Parents: "Zeus and Hera (or Hera alone)", Symbols: "Hammer, anvil, tongs, volcano", Attributes: "Lame smith of the gods who crafts divine weapons and armor.", Aliases: []string{"vulcan"}},
	{Name: "Hermes", Pantheon: "Greek", Domain: "Travel, messengers, trade, thieves, boundaries", Parents: "Zeus and Maia", Symbols: "Caduceus, winged sandals, petasos, tortoise", Attributes: "Swift herald of the gods and guide of souls to the underworld.", Aliases: []string{"mercury"}},
	{Name: "Dionysus", Pantheon: "Greek", Domain: "Wine, ecstasy, theater, fertility", Parents: "Zeus and Semele", Symbols: "Grapevine, thyrsus, leopard, ivy", Attributes: "God of wine and ritual madness who inspires both joy and frenzy.", Aliases: []string{"bacchus", "liber"}},
	{Name: "Hestia", Pantheon: "Greek", Domain: "Hearth, home, family, state", Parents: "Cronus and Rhea", Symbols: "Hearth, flame, kettle", Attributes: "Gentle virgin goddess of the hearth, guardian of the household flame.", Aliases: []string{"vesta"}},
	{Name: "Persephone", Pantheon: "Greek", Domain: "Spring, vegetation, queen of the underworld", Parents: "Zeus and Demeter", Symbols: "Pomegranate, narcissus, torch, sheaf of grain", Attributes: "Maiden of spring and dread queen of the underworld, wife of Hades.", Aliases: []string{"kore", "proserpina"}},
	{Name: "Cronus", Pantheon: "Greek", Domain: "Time, harvest, the Golden Age", Parents: "Uranus and Gaia", Symbols: "Sickle, scythe, serpent", Attributes: "Titan king who devoured his children before being overthrown by Zeus.", Aliases: []string{"kronos", "saturn"}},
	{Name: "Rhea", Pantheon: "Greek", Domain: "Motherhood, fertility, generation", Parents: "Uranus and Gaia", Symbols: "Lion, cornucopia, drum", Attributes: "Titaness mother of the Olympians who saved Zeus from Cronus.", Aliases: []string{"opis", "ops"}},
	{Name: "Gaia", Pantheon: "Greek", Domain: "Earth, fertility, creation", Parents: "Primordial, born from Chaos", Symbols: "Globe, fruits, grain, snake", Attributes: "Primordial Earth Mother and ancestor of all life.", Aliases: []string{"gaea", "tellus"}},
	{Name: "Uranus", Pantheon: "Greek", Domain: "Sky, heavens", Parents: "Born from Gaia", Symbols: "Stars, sky, sickle", Attributes: "Primordial sky god and first ruler of the cosmos, castrated by his son Cronus.", Aliases: []string{"ouranos"}},
	{Name: "Helios", Pantheon: "Greek", Domain: "Sun, sight", Parents: "Hyperion and Theia", Symbols: "Sun chariot, golden crown, horses", Attributes: "Titan who drives the sun across the sky each day in a fiery chariot.", Aliases: []string{"sol"}},
	{Name: "Selene", Pantheon: "Greek", Domain: "Moon", Parents: "Hyperion and Theia", Symbols: "Crescent moon, chariot, bull, torch", Attributes: "Titan goddess of the moon who drives her silver chariot through the night.", Aliases: []string{"luna", "mene"}},
	{Name: "Eros", Pantheon: "Greek", Domain: "Love, desire, attraction", Parents: "Aphrodite and Ares (or primordial)", Symbols: "Bow and arrow, wings, torch", Attributes: "Mischievous winged god whose arrows ignite love in mortals and gods.", Aliases: []string{"cupid", "amor"}},
	{Name: "Nike", Pantheon: "Greek", Domain: "Victory, speed, strength", Parents: "Pallas and Styx", Symbols: "Wings, laurel wreath, palm branch", Attributes: "Winged goddess of victory who crowns champions in war and games.", Aliases: []string{"victoria"}},
	{Name: "Hecate", Pantheon: "Greek", Domain: "Magic, witchcraft, crossroads, ghosts", Parents: "Perses and Asteria", Symbols: "Torch, key, dog, serpent", Attributes: "Triple goddess of magic and the night who haunts crossroads with her hounds.", Aliases: []string{"trivia"}},
	{Name: "Pan", Pantheon: "Greek", Domain: "Wilderness, shepherds, flocks, rustic music", Parents: "Hermes and a nymph", Symbols: "Pan flute, goat legs, horns, pine", Attributes: "Half-goat god of the wild whose sudden cry inspires panic.", Aliases: []string{"faunus"}},
	{Name: "Asclepius", Pantheon: "Greek", Domain: "Medicine, healing", Parents: "Apollo and Coronis", Symbols: "Serpent-entwined staff, dog, rooster", Attributes: "Mortal hero turned god of medicine, his staff is the symbol of healers.", Aliases: []string{"aesculapius"}},
	{Name: "Nemesis", Pantheon: "Greek", Domain: "Retribution, vengeance, balance", Parents: "Nyx (alone) or Erebus and Nyx", Symbols: "Sword, scales, whip, griffin", Attributes: "Goddess who delivers swift retribution to those guilty of hubris.", Aliases: []string{"adrastea", "rhamnousia"}},
	{Name: "Tyche", Pantheon: "Greek", Domain: "Fortune, prosperity, fate of cities", Parents: "Oceanus and Tethys", Symbols: "Cornucopia, wheel, ship's rudder", Attributes: "Goddess of luck whose wheel raises and topples mortals at whim.", Aliases: []string{"fortuna"}},
	{Name: "Odin", Pantheon: "Norse", Domain: "Wisdom, war, poetry, death, magic", Parents: "Borr and Bestla", Symbols: "Ravens Huginn and Muninn, spear Gungnir, eight-legged Sleipnir, runes", Attributes: "All-father of the Aesir who hung from Yggdrasil to win the runes of wisdom.", Aliases: []string{"woden", "wotan"}},
	{Name: "Thor", Pantheon: "Norse", Domain: "Thunder, lightning, storms, strength, protection", Parents: "Odin and Jord", Symbols: "Hammer Mjolnir, oak, goats, iron gloves", Attributes: "Red-bearded thunder god who defends Asgard with his hammer against giants.", Aliases: []string{"donar"}},
	{Name: "Loki", Pantheon: "Norse", Domain: "Trickery, mischief, shapeshifting, fire", Parents: "Farbauti and Laufey", Symbols: "Snake, salmon, fly, fire", Attributes: "Cunning shape-shifter and blood-brother of Odin whose schemes bring Ragnarok.", Aliases: []string{"loptr"}},
	{Name: "Freyja", Pantheon: "Norse", Domain: "Love, beauty, fertility, war, seidr magic", Parents: "Njord and Nerthus", Symbols: "Falcon cloak, necklace Brisingamen, cats, boar", Attributes: "Vanir goddess of love and war who rides a chariot drawn by cats and claims half the slain.", Aliases: []string{"freya", "vanadis"}},
	{Name: "Freyr", Pantheon: "Norse", Domain: "Fertility, sunshine, prosperity, peace", Parents: "Njord and Nerthus", Symbols: "Boar Gullinbursti, ship Skidbladnir, sword, antlers", Attributes: "Vanir god of bountiful harvests and king of the elves of Alfheim.", Aliases: []string{"frey", "yngvi"}},
	{Name: "Frigg", Pantheon: "Norse", Domain: "Marriage, motherhood, household, foresight", Parents: "Fjorgynn", Symbols: "Distaff, falcon cloak, keys, mistletoe", Attributes: "Queen of Asgard and Odin's wife who knows all fates but speaks them not.", Aliases: []string{"frigga"}},
	{Name: "Baldr", Pantheon: "Norse", Domain: "Light, beauty, purity, joy", Parents: "Odin and Frigg", Symbols: "Mistletoe, sun, ship Hringhorni", Attributes: "Shining god whose death by mistletoe foretells the end of the world.", Aliases: []string{"balder", "baldur"}},
	{Name: "Tyr", Pantheon: "Norse", Domain: "War, law, justice, oaths", Parents: "Odin (or the giant Hymir)", Symbols: "Sword, one hand, spear", Attributes: "One-handed god of justice who sacrificed his hand to bind the wolf Fenrir.", Aliases: []string{"tiw", "ziu"}},
	{Name: "Heimdall", Pantheon: "Norse", Domain: "Watchfulness, guardianship, foresight", Parents: "Odin and the nine mothers", Symbols: "Horn Gjallarhorn, rainbow bridge Bifrost, ram", Attributes: "Ever-vigilant watchman of the gods whose horn will herald Ragnarok.", Aliases: []string{"heimdallr", "rig"}},
	{Name: "Hel", Pantheon: "Norse", Domain: "Underworld, dead", Parents: "Loki and Angrboda", Symbols: "Half-blue face, throne Eljudnir, raven", Attributes: "Half-living half-corpse queen of Helheim who rules those who die of sickness and age.", Aliases: []string{"hela"}},
	{Name: "Njord", Pantheon: "Norse", Domain: "Sea, wind, seafaring, wealth, fishing", Parents: "Vanir lineage", Symbols: "Ship, foot, fish, seashell", Attributes: "Vanir god of the sea and prosperity, father of Freyr and Freyja.", Aliases: []string{"njordr", "nerthus-consort"}},
	{Name: "Idunn", Pantheon: "Norse", Domain: "Youth, immortality, spring", Parents: "Ivaldi", Symbols: "Golden apples, ash tree", Attributes: "Keeper of the golden apples that grant the gods eternal youth.", Aliases: []string{"iduna", "idun"}},
	{Name: "Bragi", Pantheon: "Norse", Domain: "Poetry, eloquence, bards", Parents: "Odin (and Gunnlod in some tales)", Symbols: "Harp, runes on tongue", Attributes: "Skaldic god of poetry and husband of Idunn who greets warriors in Valhalla.", Aliases: []string{"brage"}},
	{Name: "Vidar", Pantheon: "Norse", Domain: "Vengeance, silence, primal force", Parents: "Odin and the giantess Grid", Symbols: "Iron shoe, forest, sword", Attributes: "Silent god of vengeance fated to kill Fenrir and survive Ragnarok.", Aliases: []string{"vithar", "vidarr"}},
	{Name: "Skadi", Pantheon: "Norse", Domain: "Winter, mountains, skiing, hunting", Parents: "The giant Thjazi", Symbols: "Bow, skis, snowshoes, wolves", Attributes: "Giantess huntress of the snowy peaks who joined the gods to avenge her father.", Aliases: []string{"skade", "skadhi"}},
	{Name: "Ra", Pantheon: "Egyptian", Domain: "Sun, creation, kingship", Parents: "Self-created (or Nun)", Symbols: "Sun disk, falcon, scarab, ankh", Attributes: "Falcon-headed sun god who sails his solar barque across the sky each day.", Aliases: []string{"re", "amun-ra"}},
	{Name: "Osiris", Pantheon: "Egyptian", Domain: "Afterlife, resurrection, vegetation, fertility", Parents: "Geb and Nut", Symbols: "Crook, flail, atef crown, green skin", Attributes: "Slain and resurrected lord of the dead who judges souls in the Hall of Ma'at.", Aliases: []string{"wesir", "asar"}},
	{Name: "Isis", Pantheon: "Egyptian", Domain: "Magic, motherhood, healing, protection", Parents: "Geb and Nut", Symbols: "Throne hieroglyph, wings, ankh, sun disk with horns", Attributes: "Great mother sorceress who resurrected Osiris and protects the pharaoh.", Aliases: []string{"aset", "auset"}},
	{Name: "Horus", Pantheon: "Egyptian", Domain: "Sky, kingship, protection, war", Parents: "Osiris and Isis", Symbols: "Falcon, Eye of Horus, double crown, was scepter", Attributes: "Falcon-headed sky god whose Eye protects pharaohs and the dead.", Aliases: []string{"heru", "har"}},
	{Name: "Set", Pantheon: "Egyptian", Domain: "Chaos, storms, desert, violence, foreigners", Parents: "Geb and Nut", Symbols: "Set animal, was scepter, red color, donkey", Attributes: "Red-skinned god of storms and disorder who murdered his brother Osiris.", Aliases: []string{"seth", "sutekh"}},
	{Name: "Anubis", Pantheon: "Egyptian", Domain: "Mummification, embalming, the dead, judgment", Parents: "Osiris and Nephthys (or Set)", Symbols: "Jackal, scales, flail, black skin", Attributes: "Jackal-headed guide of souls who weighs the heart against Ma'at's feather.", Aliases: []string{"anpu", "inpu"}},
	{Name: "Thoth", Pantheon: "Egyptian", Domain: "Writing, wisdom, magic, the moon, knowledge", Parents: "Self-created (or born from Set's forehead)", Symbols: "Ibis, baboon, scribe's palette, moon disk", Attributes: "Ibis-headed scribe of the gods and inventor of writing and mathematics.", Aliases: []string{"djehuty", "tehuti"}},
	{Name: "Bastet", Pantheon: "Egyptian", Domain: "Cats, home, fertility, protection of pharaoh", Parents: "Ra", Symbols: "Cat, lioness, sistrum, sun disk", Attributes: "Cat goddess of home and joy who fiercely defends pharaoh and family.", Aliases: []string{"bast", "ubasti"}},
	{Name: "Sekhmet", Pantheon: "Egyptian", Domain: "War, plague, healing, fire", Parents: "Ra", Symbols: "Lioness head, sun disk, ankh, arrows", Attributes: "Lion-headed warrior goddess whose breath formed the desert and whose wrath nearly destroyed humanity.", Aliases: []string{"sakhmet"}},
	{Name: "Hathor", Pantheon: "Egyptian", Domain: "Love, beauty, music, motherhood, joy, the sky", Parents: "Ra (or Nut and Geb)", Symbols: "Cow horns, sun disk, sistrum, mirror", Attributes: "Cow goddess of love and music who welcomes the dead and nurses pharaohs.", Aliases: []string{"het-heru"}},
	{Name: "Ptah", Pantheon: "Egyptian", Domain: "Crafts, creation, architecture, artisans", Parents: "Self-created", Symbols: "Was scepter, ankh, djed pillar, skullcap", Attributes: "Demiurge of Memphis who created the world by speaking it into being.", Aliases: []string{"peteh"}},
	{Name: "Geb", Pantheon: "Egyptian", Domain: "Earth, vegetation, snakes, fertility", Parents: "Shu and Tefnut", Symbols: "Goose, barley, green or black skin", Attributes: "Reclining earth god whose laughter causes earthquakes, husband and brother of Nut.", Aliases: []string{"keb", "seb"}},
	{Name: "Nut", Pantheon: "Egyptian", Domain: "Sky, stars, cosmos, mothers, astronomy", Parents: "Shu and Tefnut", Symbols: "Arched body covered in stars, sycamore, cow", Attributes: "Arched sky goddess who swallows the sun each night and births it at dawn.", Aliases: []string{"nuit"}},
	{Name: "Ma'at", Pantheon: "Egyptian", Domain: "Truth, balance, order, justice, morality", Parents: "Ra", Symbols: "Ostrich feather, scales, ankh, wings", Attributes: "Goddess of cosmic order whose feather weighs each soul in the afterlife.", Aliases: []string{"maat"}},
	{Name: "Sobek", Pantheon: "Egyptian", Domain: "Nile, crocodiles, fertility, military prowess", Parents: "Senuy or Neith", Symbols: "Crocodile, sun disk, water, was scepter", Attributes: "Crocodile-headed god of the Nile whose strength shields pharaoh in battle.", Aliases: []string{"sebek", "suchos"}},
	{Name: "Khonsu", Pantheon: "Egyptian", Domain: "Moon, time, healing, youth", Parents: "Amun and Mut", Symbols: "Moon disk and crescent, sidelock, falcon head, ankh", Attributes: "Lunar god whose journey across the night sky measures time and heals the sick.", Aliases: []string{"chonsu", "khons"}},
}

var mythDeityIndex map[string]*mythDeity

func mythBuildIndex() {
	if mythDeityIndex != nil {
		return
	}
	mythDeityIndex = make(map[string]*mythDeity, len(mythDeities)*2)
	for i := range mythDeities {
		d := &mythDeities[i]
		mythDeityIndex[strings.ToLower(d.Name)] = d
		for _, a := range d.Aliases {
			mythDeityIndex[strings.ToLower(a)] = d
		}
	}
}

func mythListPantheon(p string) string {
	var names []string
	for i := range mythDeities {
		if mythDeities[i].Pantheon == p {
			names = append(names, mythDeities[i].Name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func mythFormatDeity(d *mythDeity) string {
	out := "<b>" + html.EscapeString(d.Name) + "</b>\n"
	out += "<i>" + html.EscapeString(d.Pantheon) + " mythology</i>\n\n"
	out += "<b>Domain:</b> " + html.EscapeString(d.Domain) + "\n"
	out += "<b>Parents:</b> " + html.EscapeString(d.Parents) + "\n"
	out += "<b>Symbols:</b> " + html.EscapeString(d.Symbols) + "\n\n"
	out += "<b>About:</b> " + html.EscapeString(d.Attributes)
	if len(d.Aliases) > 0 {
		out += "\n\n<b>Also known as:</b> " + html.EscapeString(strings.Join(d.Aliases, ", "))
	}
	return out
}

func mythUsage() string {
	out := "<b>Mythology</b>\n\n"
	out += "Usage: <code>/myth &lt;god name&gt;</code>\n\n"
	out += "Examples: <code>/myth zeus</code>, <code>/myth thor</code>, <code>/myth ra</code>\n\n"
	out += "<b>Greek:</b> " + html.EscapeString(mythListPantheon("Greek")) + "\n\n"
	out += "<b>Norse:</b> " + html.EscapeString(mythListPantheon("Norse")) + "\n\n"
	out += "<b>Egyptian:</b> " + html.EscapeString(mythListPantheon("Egyptian"))
	return out
}

func mythSuggest(q string) []string {
	q = strings.ToLower(q)
	var hits []string
	seen := map[string]bool{}
	for i := range mythDeities {
		d := &mythDeities[i]
		if strings.Contains(strings.ToLower(d.Name), q) && !seen[d.Name] {
			hits = append(hits, d.Name)
			seen[d.Name] = true
		}
	}
	for i := range mythDeities {
		d := &mythDeities[i]
		for _, a := range d.Aliases {
			if strings.Contains(strings.ToLower(a), q) && !seen[d.Name] {
				hits = append(hits, d.Name)
				seen[d.Name] = true
			}
		}
	}
	if len(hits) > 8 {
		hits = hits[:8]
	}
	return hits
}

func MythologyHandler(m *tg.NewMessage) error {
	mythBuildIndex()
	args := strings.TrimSpace(m.Args())
	if args == "" {
		m.Reply(mythUsage())
		return nil
	}
	key := strings.ToLower(args)
	if d, ok := mythDeityIndex[key]; ok {
		m.Reply(mythFormatDeity(d))
		return nil
	}
	suggestions := mythSuggest(key)
	if len(suggestions) > 0 {
		out := "<b>No exact match for</b> <code>" + html.EscapeString(args) + "</code>\n\n"
		out += "<b>Did you mean:</b> " + html.EscapeString(strings.Join(suggestions, ", "))
		m.Reply(out)
		return nil
	}
	m.Reply("<b>Unknown deity:</b> <code>" + html.EscapeString(args) + "</code>\n\nUse <code>/myth</code> alone to see the full list.")
	return nil
}

func init() { QueueHandlerRegistration(registerMythologyHandlers) }
func registerMythologyHandlers() {
	c := Client
	c.On("cmd:myth", MythologyHandler)
}
