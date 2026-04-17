package detector

import "github.com/mileusna/useragent"

func IsBot(userAgentString string) bool {
	ua := useragent.Parse(userAgentString)
	return ua.Bot
}
