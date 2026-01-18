package queue

import (
	"context"
	"errors"
)

// WorkType описывает тип задания для отдельной обработки.
type WorkType string

const (
	WorkCredential WorkType = "credential"
	WorkFile       WorkType = "file"
	WorkCard       WorkType = "card"
	WorkText       WorkType = "text"
)

// Task описывает единицу работы в очереди.
type Task struct {
	ID      string
	Type    WorkType
	Payload []byte
}

// Queue простая неблокирующая очередь на канале.
type Queue struct {
	ch chan Task
}

// New создаёт очередь с заданным буфером.
func New(size int) *Queue {
	if size <= 0 {
		size = 16
	}
	return &Queue{ch: make(chan Task, size)}
}

// Enqueue добавляет задачу.
func (q *Queue) Enqueue(t Task) error {
	select {
	case q.ch <- t:
		return nil
	default:
		return errors.New("queue is full")
	}
}

// Dequeue забирает задачу или возвращает контекстную ошибку.
func (q *Queue) Dequeue(ctx context.Context) (Task, error) {
	select {
	case t := <-q.ch:
		return t, nil
	case <-ctx.Done():
		return Task{}, ctx.Err()
	}
}
