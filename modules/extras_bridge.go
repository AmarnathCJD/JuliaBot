package modules

// This file is the seam between core (modules) and the optional
// `modules/extras` package. Core can't import extras (extras imports core,
// so that would cycle). Instead, extras registers its implementations into
// these function variables at init() time, and core calls through them.
//
// Each variable defaults to a no-op or a sensible fallback, so core stays
// functional even when extras isn't blank-imported (e.g. during tests, or
// if someone deliberately strips the extras directory).

// RecordAction is populated by extras/warns.go. Used by admin.go to log
// ban/mute/etc. actions to the warns history.
var RecordAction = func(chatID, userID, adminID int64, actionType string, data map[string]interface{}) {}

// PostToSpaceBin is populated by extras/misc.go. Used by dev.go to upload
// large outputs (mediainfo, traces) to a paste service. Returns (url, key, error).
var PostToSpaceBin = func(content string) (string, string, error) { return "", "", nil }

// MdToTelegramHTML is populated by extras/mdhtml.go. Used by zai.go to
// convert markdown answers into Telegram-safe HTML.
var MdToTelegramHTML = func(s string) string { return s }
