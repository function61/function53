package main

import (
	"github.com/function61/gokit/logex"
	"log"
)

type QueryLogger interface {
	LogQuery(name string, client string)
}

type queryLogLogger struct {
	logl *logex.Leveled
}

func NewLogQueryLogger(logger *log.Logger) QueryLogger {
	return &queryLogLogger{logex.Levels(logger)}
}

func (q *queryLogLogger) LogQuery(name string, client string) {
	q.logl.Debug.Printf("name:%s client:%s", name, client)
}

type nilLogger struct{}

func (n *nilLogger) LogQuery(name string, client string) {}

func NewNilQueryLogger() QueryLogger {
	return &nilLogger{}
}
