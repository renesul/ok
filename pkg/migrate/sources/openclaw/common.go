package openclaw

var migrateableFiles = []string{
	"AGENTS.md",
	"SOUL.md",
	"USER.md",
	"TOOLS.md",
	"HEARTBEAT.md",
}

var migrateableDirs = []string{
	"memory",
	"skills",
}

var supportedChannels = map[string]bool{
	"whatsapp":  true,
	"telegram":  true,
	"discord":   true,
	"maixcam":   true,
	"qq":        true,
	"slack":     true,
	"line":      true,
	"onebot":    true,
	"wecom":     true,
	"wecom_app": true,
}
