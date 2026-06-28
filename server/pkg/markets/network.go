package markets

import "strings"

var globalNetworkAliases = map[string]string{
	"MTN":             "MTN",
	"MTNMOMO":         "MTN",
	"MTNMONEY":        "MTN",
	"MOOV":            "MOOV",
	"ORANGE":          "ORANGE",
	"FREE":            "FREE",
	"TELECEL":         "TELECEL",
	"VODAFONE":        "TELECEL",
	"AIRTELTIGO":      "AIRTELTIGO",
	"AIRTELTIGOMONEY": "AIRTELTIGO",
	"AT":              "AIRTELTIGO",
	"ATMONEY":         "AIRTELTIGO",
}

var countryNetworkAliases = map[string]map[string]string{
	"GHA": {
		"MTN":             "MTN",
		"MTNMOMO":         "MTN",
		"TELECEL":         "TELECEL",
		"VODAFONE":        "TELECEL",
		"AIRTELTIGO":      "AIRTELTIGO",
		"AIRTELTIGOMONEY": "AIRTELTIGO",
		"AT":              "AIRTELTIGO",
		"ATMONEY":         "AIRTELTIGO",
	},
	"BFA": {
		"MOOV":   "MOOV",
		"ORANGE": "ORANGE",
	},
	"CMR": {
		"MTN":    "MTN",
		"ORANGE": "ORANGE",
	},
	"CIV": {
		"MTN":    "MTN",
		"ORANGE": "ORANGE",
		"MOOV":   "MOOV",
	},
	"SEN": {
		"FREE":   "FREE",
		"ORANGE": "ORANGE",
	},
	"SLE": {
		"ORANGE": "ORANGE",
	},
	"BEN": {
		"MTN":  "MTN",
		"MOOV": "MOOV",
	},
}

func NormalizeNetwork(country, value string) string {
	country = NormalizeCountry(country)

	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")

	if aliases, ok := countryNetworkAliases[country]; ok {
		if network, ok := aliases[value]; ok {
			return network
		}
	}

	if network, ok := globalNetworkAliases[value]; ok {
		return network
	}

	return value
}
