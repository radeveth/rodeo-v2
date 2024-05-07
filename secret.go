package main

import "app/lib"

var secrets = map[string]map[string]string{
	"development": map[string]string{
		"ENV":             "development",
		"PORT":            "8000",
		"BASE_URL":        "http://localhost:8000",
		"APP_NAME":        "rodeo",
		"COMPANY_NAME":    "Rodeo",
		"EMAIL_FROM":      "noreply@rodeo.so",
		"DISCORD_URL":     "https://discord.gg/6N9braAzms",
		"X_USERNAME":      "Rodeo_Finance",
		"X_CLIENT_ID":     "T1ZfWFZSNTAzRVQ3RmV1emNSZFg6MTpjaQ",
		"X_CLIENT_SECRET": "$e1$eiHAU0cz1LXOxd89y+sqWM6gBadNhxeaz8d5gjpiP98befztUlPi16PFVxrYEA5giUjMVEH1AQqgZmDdhfRC/xnu7fAJhMjttryqqoNy",
		"PRIVATE_KEY":     lib.Env("RODEO_PRIVATE_KEY_KEEPER", "f27af80042ab65cd6fff4ac429e8c4cfdceb0ffbaee28d2e2c3ed2e91fb62cc3"),
		"ONEINCH_API_KEY": lib.Env("RODEO_1INCH_API_KEY", ""),
		"WC_PROJECT_ID":   "e86d5a5950be6b9f9e50f6d27285ef84",
	},
	"production": map[string]string{
		//"PORT":            "8001",
		"BASE_URL":        "https://www.rodeo.so",
		"APP_NAME":        "rodeo",
		"COMPANY_NAME":    "Rodeo",
		"EMAIL_FROM":      "noreply@rodeo.so",
		"DISCORD_URL":     "https://discord.gg/6N9braAzms",
		"X_USERNAME":      "Rodeo_Finance",
		"X_CLIENT_ID":     "T1ZfWFZSNTAzRVQ3RmV1emNSZFg6MTpjaQ",
		"X_CLIENT_SECRET": "$e1$FUzn+pdjGF08TJV8fniZkPNAyNRSguxdNH7l6x8EaXopwDupmK7KVwFQ0XfsM/kNPcj09EPVXFlGukmMXxkjZzi4Y5briTJmLA1M9kD6",
		"WC_PROJECT_ID":   "e86d5a5950be6b9f9e50f6d27285ef84",
		"PRIVATE_KEY":     "$e1$rYXgO3hNdzjVTnk2Xuz/HjbZcISLx/Rlltu8J9DIj2aop8bSqR0vJt3RHvKbKuKlaHPzHAozQUqHbTVLMT7Sm1F87YvN94uzFNhyfWwikXiFJ3qdcCXrXVyqU2M=",
		"RPC_URL_42161":   "$e1$tIADKaGY6UHukxlUPwzAS5IvNJ/OifbiPM1Le7M5xCDvlNMiE7P/Av3tSVDgONOOcsw8Gbnbatmhc4JqKvOp2hNpDqZvOmx7EXqgKz2MtNwB5J6rPaPeB++TpGZzkRUJ/Q==",
		"ONEINCH_API_KEY": "$e1$yNbM9yoHAMhAC9LdltNN2dQezxfa2zfXuCR1NJmUb6fbASErb11dsBlk7MYwqBI2MbOUz8jYu59O4wAv",
		//"DATABASE_URL":    "$e1$++trZKqUe/uPf14GAddUpd3K7OIrhhNkPZGjPDdA5b8kd/gbNBSIvxX/jA7DkUXs3/xaqXA/BgTIa3BOi7tbxW+Zo+dRmUWVBt7mKwqSFSfphY8CLWKO29kcd2l7s3ES2efn6wq2",
	},
}
