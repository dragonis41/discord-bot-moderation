package utils

import "time"

var uptime *Uptime

type Uptime struct {
	StartTime time.Time
}

type UptimeInterface interface {
	GetUptime() time.Duration
}

func NewUptime() *Uptime {
	uptime = &Uptime{
		StartTime: time.Now(),
	}
	return uptime
}

func (u *Uptime) GetUptime() time.Duration {
	return time.Since(u.StartTime)
}

func GetUptime() string {
	if uptime == nil {
		return "Unknown"
	}
	duration := uptime.GetUptime()
	return duration.String()
}
