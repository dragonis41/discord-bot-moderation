package logger

import (
	"fmt"
	"log"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type LogInterface interface {
	LogSuccess(m LogModel)
	LogInfo(m LogModel)
	LogWarning(m LogModel)
	LogError(m LogModel)
	LogFatal(message string)
}

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

// LogSuccess logs a success message to the console and database if provided.
//
//	m.Database can be nil, in which case the message is only logged to the console.
func (l *Logger) LogSuccess(m LogModel) {
	log.Printf("%s[%s] %s%s\n", model.Green.String(), model.TypeSuccess, m.Message, model.Reset.String())

	if m.Database != nil {
		err := m.Database.AddSystemLogEntry(m.GuildID, model.TypeSuccess, m.Function, m.Message)
		if err != nil {
			l.LogError(LogModel{Message: fmt.Sprintf("failed to log success message to database: %s", err.Error())})
		}
	}
}

// LogInfo logs an info message to the console and database if provided.
//
//	m.Database can be nil, in which case the message is only logged to the console.
func (l *Logger) LogInfo(m LogModel) {
	log.Printf("%s[%s] %s%s\n", model.Blue.String(), model.TypeInfo, m.Message, model.Reset.String())

	if m.Database != nil {
		err := m.Database.AddSystemLogEntry(m.GuildID, model.TypeInfo, m.Function, m.Message)
		if err != nil {
			l.LogError(LogModel{Message: fmt.Sprintf("failed to log info message to database: %s", err.Error())})
		}
	}
}

// LogWarning logs a warning message to the console and database if provided.
//
//	m.Database can be nil, in which case the message is only logged to the console.
func (l *Logger) LogWarning(m LogModel) {
	log.Printf("%s[%s] %s%s\n", model.Orange.String(), model.TypeWarning, m.Message, model.Reset.String())

	if m.Database != nil {
		err := m.Database.AddSystemLogEntry(m.GuildID, model.TypeWarning, m.Function, m.Message)
		if err != nil {
			l.LogError(LogModel{Message: fmt.Sprintf("failed to log warning message to database: %s", err.Error())})
		}
	}
}

// LogError logs an error message to the console and database if provided.
//
//	m.Database can be nil, in which case the message is only logged to the console.
func (l *Logger) LogError(m LogModel) {
	log.Printf("%s[%s] %s%s\n", model.Red.String(), model.TypeError, m.Message, model.Reset.String())

	if m.Database != nil {
		err := m.Database.AddSystemLogEntry(m.GuildID, model.TypeError, m.Function, m.Message)
		if err != nil {
			l.LogError(LogModel{Message: fmt.Sprintf("failed to log error message to database: %s", err.Error())})
		}
	}
}

// LogFatal logs a fatal error message to the console and exits the application.
func (l *Logger) LogFatal(message string) {
	log.Fatalf("%s[%s] %s%s\n", model.Red.String(), model.TypeError, message, model.Reset.String())
}
