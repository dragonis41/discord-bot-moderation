package model

import "github.com/bwmarrin/discordgo"

type LogType string

const (
	SuccessType LogType = "SUCCESS"
	InfoType    LogType = "INFO"
	WarningType LogType = "WARNING"
	ErrorType   LogType = "ERROR"
)

func (l LogType) String() string {
	return string(l)
}

type Color struct {
	ansi string
	hex  int
}

func (c Color) String() string {
	return c.ansi
}

func (c Color) Int() int {
	return c.hex
}

var (
	Red    = Color{ansi: "\033[31m", hex: 0xff0000}
	Green  = Color{ansi: "\033[32m", hex: 0x00dd00}
	Orange = Color{ansi: "\033[33m", hex: 0xff8800}
	Blue   = Color{ansi: "\033[34m", hex: 0x0099ff}
	Reset  = Color{ansi: "\033[0m", hex: 0xffffff}
)

var (
	DefaultFooter       = &discordgo.MessageEmbedFooter{Text: "💡 Hint: Utilisez /help pour lister les commandes disponibles."}
	SelectionMenuFooter = &discordgo.MessageEmbedFooter{Text: "Les sélections sont sauvegardées à chaque modification"}
)
