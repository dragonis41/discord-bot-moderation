package utils

import "log"

var (
	red    = "\033[31m"
	green  = "\033[32m"
	orange = "\033[33m"
	blue   = "\033[34m"
	reset  = "\033[0m"

	SuccessType = "SUCCESS"
	InfoType    = "INFO"
	WarningType = "WARNING"
	ErrorType   = "ERROR"
)

// TODO : Also log in the database

func LogSuccess(message string) {
	log.Printf("%s[%s] %s%s\n", green, SuccessType, message, reset)
}

func LogInfo(message string) {
	log.Printf("%s[%s] %s%s\n", blue, InfoType, message, reset)
}

func LogWarning(message string) {
	log.Printf("%s[%s] %s%s\n", orange, WarningType, message, reset)
}

func LogError(message string) {
	log.Printf("%s[%s] %s%s\n", red, ErrorType, message, reset)
}

func LogFatal(message string) {
	log.Fatalf("%s[%s] %s%s\n", red, ErrorType, message, reset)
}
