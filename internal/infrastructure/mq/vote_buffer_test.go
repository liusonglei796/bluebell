package mq

import (
	"context"
	"sync"
	"testing"
	"time"

	"bluebell/internal/domain/entity"
)

func TestVoteBuffer_AddAndFlush(t *testing.T) {
	var mu sync.Mutex
	receivedVotes := make(map[string]int8)
	flushCount := 0

	flushFunc := func(ctx context.Context, votes map[string]int8) error {
		mu.Lock()
		defer mu.Unlock()
		for k, v := range votes {
			receivedVotes[k] = v
		}
		flushCount++
		return nil
	}

	interval := 50 * time.Millisecond
	vb := NewVoteBuffer(interval, flushFunc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go vb.Start(ctx)

	// Add some votes
	vb.AddVote("post1", "user1", entity.VoteUp)
	vb.AddVote("post1", "user2", entity.VoteDown)
	vb.AddVote("post2", "user1", entity.VoteUp)

	// Wait for flush
	time.Sleep(interval * 2)

	mu.Lock()
	if len(receivedVotes) != 3 {
		t.Errorf("expected 3 votes, got %d", len(receivedVotes))
	}
	if receivedVotes["post1:user1"] != entity.VoteUp {
		t.Errorf("expected upvote for post1:user1")
	}
	if flushCount == 0 {
		t.Errorf("expected at least one flush")
	}
	mu.Unlock()

	// Add more votes and check next flush
	vb.AddVote("post3", "user3", entity.VoteUp)
	time.Sleep(interval * 2)

	mu.Lock()
	if receivedVotes["post3:user3"] != entity.VoteUp {
		t.Errorf("expected upvote for post3:user3 after second flush")
	}
	mu.Unlock()
}

func TestVoteBuffer_Stop(t *testing.T) {
	receivedVotes := make(map[string]int8)
	flushFunc := func(ctx context.Context, votes map[string]int8) error {
		for k, v := range votes {
			receivedVotes[k] = v
		}
		return nil
	}

	interval := 1 * time.Hour // Long interval to prevent periodic flush
	vb := NewVoteBuffer(interval, flushFunc)

	ctx, cancel := context.WithCancel(context.Background())
	
	// Use a separate goroutine to run Start
	done := make(chan struct{})
	go func() {
		vb.Start(ctx)
		close(done)
	}()

	vb.AddVote("post_final", "user_final", entity.VoteUp)
	
	// Stop the buffer
	cancel()
	<-done

	if receivedVotes["post_final:user_final"] != entity.VoteUp {
		t.Errorf("expected final flush on stop")
	}
}
