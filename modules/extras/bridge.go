package extras

import (
	modules "main/modules"
)

// Populate the bridge variables that core (modules) exposes for optional
// extras-supplied behavior. Without this wiring, the core defaults from
// modules/extras_bridge.go (no-ops) remain in effect.
//
// Runs before any other extras init() that depends on these (none do — the
// flow is core → bridge → extras handler funcs), so Go's per-package init
// ordering (single init per file, files in lexical order) is fine here.
func init() {
	modules.RecordAction = RecordAction
	modules.PostToSpaceBin = postToSpaceBin
	modules.MdToTelegramHTML = mdToTelegramHTML
}
