package screepssocket

import (
	"regexp"
	"strings"

	"github.com/rs/zerolog"
)

func LogConsolePayload(logger zerolog.Logger, msg any, intercept HandleConsoleLog) {
	payload, ok := msg.(map[string]any)
	if !ok {
		logger.Error().Any("msg", msg).Msg("handle console payload failed")
		return
	}

	shard := "none"
	if payload["shard"] != nil {
		shard = payload["shard"].(string)
	}
	logger = logger.With().Str("shard", shard).Logger()

	// Log each message as output.
	if payload["messages"] != nil {
		messages, ok := payload["messages"].(map[string]any)
		if ok {
			if logs, ok := messages["log"]; ok {
				lines, ok := logs.([]any)
				if ok {
					for _, line := range lines {
						lineStr, _ := line.(string)
						if intercept != nil {
							if intercept(logger, ConsoleLogMeta{Shard: shard}, lineStr) {
								continue
							}
						}

						lineStr = strings.TrimSpace(RemoveHTMLTags(lineStr))
						lvl := zerolog.InfoLevel
						if len(lineStr) > 3 {
							level := lineStr[:3]
							switch level {
							case "FTL":
								lvl = zerolog.FatalLevel
							case "ERR":
								lvl = zerolog.ErrorLevel
							case "WRN":
								lvl = zerolog.WarnLevel
							case "INF":
								lvl = zerolog.InfoLevel
							case "DBG":
								lvl = zerolog.DebugLevel
							}
						}

						// Log formats use HTML for colors. Let's make this better.
						logger.WithLevel(lvl).Msg(lineStr)
					}
				} else {
					logger.Error().Any("log", logs).Msg("Failed to parse log messages")
				}
			}
		}
	}
}

var fontRegex = regexp.MustCompile(`<font color='(?P<color>[^']+)'>(?P<text>[^<]+)<\/font>`)

func RemoveHTMLTags(s string) string {
	return fontRegex.ReplaceAllString(s, "$2")
}
