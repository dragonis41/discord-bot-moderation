package model

import "testing"

func TestColorInt(t *testing.T) {
	tests := []struct {
		name  string
		color Color
		want  int
	}{
		{"red", Red, 0xff0000},
		{"green", Green, 0x00dd00},
		{"orange", Orange, 0xff8800},
		{"blue", Blue, 0x0099ff},
		{"violet", Violet, 0x9900ff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.color.Int(); got != tt.want {
				t.Errorf("%s.Int() = %#x, want %#x", tt.name, got, tt.want)
			}
		})
	}
}

func TestStringers(t *testing.T) {
	if TypeError.String() != "ERROR" {
		t.Errorf("TypeError.String() = %q, want ERROR", TypeError.String())
	}
	if ActionBan.String() != "BAN" {
		t.Errorf("ActionBan.String() = %q, want BAN", ActionBan.String())
	}
	if Red.String() == "" {
		t.Error("Red.String() should return the ansi escape, got empty")
	}
	if ChannelType(ChannelTypeLog).String() != "LOG" {
		t.Errorf("ChannelType(LOG).String() = %q, want LOG", ChannelType(ChannelTypeLog).String())
	}
	if TypeWarning.String() != "WARNING" || TypeInfo.String() != "INFO" || TypeSuccess.String() != "SUCCESS" {
		t.Error("SystemLogType stringers are wrong")
	}
	if ActionKick.String() != "KICK" || ActionWarn.String() != "WARN" || ActionReport.String() != "REPORT" || ActionDeleteMessage.String() != "DELETE_MESSAGE" {
		t.Error("ModerationLogAction stringers are wrong")
	}
}
