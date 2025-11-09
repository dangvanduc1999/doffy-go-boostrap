package core

import (
	"encoding/json"
	"fmt"
	"time"
)

type LoggerItem struct {
	Event    string
	Messages string
	Error    error       `json:"error,omitempty"`
	Data     interface{} `json:"data"`
}

type Logger interface {
	Infor(*LoggerItem)
}

type logger struct{}

func InitLogger() Logger {
	return &logger{}
}

func (l *logger) Infor(payload *LoggerItem) {
	b, _ := json.MarshalIndent(payload.Data, "", " ")
	fmt.Printf("[Doff-Event]::%s::[Message]::::%s:::[Data]----->`\n%s\n", payload.Event, payload.Messages, string(b))
}

func DefaultLogger() Logger {
	logger := InitLogger()

	payload := &LoggerItem{
		Event:    "initLoggerSuccefully",
		Messages: "init logger successfully",
		Data: struct {
			CreatedAt time.Time `json:"create_at"`
		}{
			CreatedAt: time.Now().UTC(),
		},
	}
	logger.Infor(payload)

	return logger
}
