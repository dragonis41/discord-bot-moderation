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

func (l *Logger) LogSuccess(m LogModel) {
	log.Printf("%s[%s] %s%s\n", model.Green.String(), model.SuccessType, m.Message, model.Reset.String())

	if m.Database != nil {
		err := m.Database.AddSystemLogEntry(m.GuildID, model.SuccessType, m.Function, m.Message)
		if err != nil {
			l.LogError(LogModel{Message: fmt.Sprintf("failed to log success message to database: %s", err.Error())})
		}
	}
}

func (l *Logger) LogInfo(m LogModel) {
	log.Printf("%s[%s] %s%s\n", model.Blue.String(), model.InfoType, m.Message, model.Reset.String())

	if m.Database != nil {
		err := m.Database.AddSystemLogEntry(m.GuildID, model.InfoType, m.Function, m.Message)
		if err != nil {
			l.LogError(LogModel{Message: fmt.Sprintf("failed to log info message to database: %s", err.Error())})
		}
	}
}

func (l *Logger) LogWarning(m LogModel) {
	log.Printf("%s[%s] %s%s\n", model.Orange.String(), model.WarningType, m.Message, model.Reset.String())

	if m.Database != nil {
		err := m.Database.AddSystemLogEntry(m.GuildID, model.WarningType, m.Function, m.Message)
		if err != nil {
			l.LogError(LogModel{Message: fmt.Sprintf("failed to log warning message to database: %s", err.Error())})
		}
	}
}

func (l *Logger) LogError(m LogModel) {
	log.Printf("%s[%s] %s%s\n", model.Red.String(), model.ErrorType, m.Message, model.Reset.String())

	if m.Database != nil {
		err := m.Database.AddSystemLogEntry(m.GuildID, model.ErrorType, m.Function, m.Message)
		if err != nil {
			l.LogError(LogModel{Message: fmt.Sprintf("failed to log error message to database: %s", err.Error())})
		}
	}
}

func (l *Logger) LogFatal(message string) {
	log.Fatalf("%s[%s] %s%s\n", model.Red.String(), model.ErrorType, message, model.Reset.String())
}
