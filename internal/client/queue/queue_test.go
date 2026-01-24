package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/queue"
)

func TestQueue_EnqueueDequeue(t *testing.T) {
	q := queue.New(10)

	// Enqueue task
	task := queue.Task{
		Type: queue.WorkCredential,
		ID:   "test-id",
	}

	err := q.Enqueue(task)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Dequeue task
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	received, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("failed to dequeue: %v", err)
	}

	if received.ID != task.ID {
		t.Errorf("expected task ID %s, got %s", task.ID, received.ID)
	}
	if received.Type != task.Type {
		t.Errorf("expected task type %v, got %v", task.Type, received.Type)
	}
}

func TestQueue_MultipleEnqueueDequeue(t *testing.T) {
	q := queue.New(10)

	// Enqueue multiple tasks
	tasks := []queue.Task{
		{Type: queue.WorkCredential, ID: "cred-1"},
		{Type: queue.WorkFile, ID: "file-1"},
		{Type: queue.WorkCard, ID: "card-1"},
		{Type: queue.WorkText, ID: "text-1"},
	}

	for _, task := range tasks {
		if err := q.Enqueue(task); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}
	}

	// Dequeue and verify
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i, expected := range tasks {
		received, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("task %d: failed to dequeue: %v", i, err)
		}

		if received.ID != expected.ID {
			t.Errorf("task %d: expected ID %s, got %s", i, expected.ID, received.ID)
		}
		if received.Type != expected.Type {
			t.Errorf("task %d: expected type %v, got %v", i, expected.Type, received.Type)
		}
	}
}

func TestQueue_EmptyQueue(t *testing.T) {
	q := queue.New(10)

	// Try to dequeue from empty queue (should timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := q.Dequeue(ctx)
	if err == nil {
		t.Fatal("expected timeout error from empty queue")
	}
}

func TestQueue_AllTaskTypes(t *testing.T) {
	q := queue.New(10)

	taskTypes := []queue.WorkType{
		queue.WorkCredential,
		queue.WorkFile,
		queue.WorkCard,
		queue.WorkText,
	}

	for _, taskType := range taskTypes {
		task := queue.Task{
			Type: taskType,
			ID:   "test-id",
		}
		if err := q.Enqueue(task); err != nil {
			t.Fatalf("failed to enqueue: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		received, err := q.Dequeue(ctx)
		cancel()

		if err != nil {
			t.Fatalf("failed to dequeue for type %v: %v", taskType, err)
		}

		if received.Type != taskType {
			t.Errorf("expected type %v, got %v", taskType, received.Type)
		}
	}
}
