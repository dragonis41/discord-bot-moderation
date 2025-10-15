package utils

import "log"

var (
	red    = "\033[31m"
	green  = "\033[32m"
	orange = "\033[33m"
	blue   = "\033[34m"
	reset  = "\033[0m"
)

// TODO : Also log in the database

func LogSuccess(message string) {
	log.Printf("%s%s%s\n", green, message, reset)
}

func LogInfo(message string) {
	log.Printf("%s[INFO] %s%s\n", blue, message, reset)
}

func LogWarning(message string) {
	log.Printf("%s[WARNING] %s%s\n", orange, message, reset)
}

func LogError(message string) {
	log.Printf("%s[ERROR] %s%s\n", red, message, reset)
}

func LogFatal(message string) {
	log.Fatalf("%s[ERROR] %s%s\n", red, message, reset)
}
