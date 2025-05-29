package queue

import "log"

type Logger interface {
	Println(v ...any)
}

type DefaultLogger struct{}

func (l *DefaultLogger) Println(v ...any) {
	log.Println(v...)
}
