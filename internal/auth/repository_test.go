package auth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	rds "github.com/redis/go-redis/v9"
)

func TestRepoSessionInsertAndGet(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	client := rds.NewClient(&rds.Options{Addr: mr.Addr()})
	repo := &AuthRepository{db: nil, redis: client}

	ctx := context.Background()
	tokenHash := "testhash"

	if err := repo.repoInsertToken(ctx, "u1", tokenHash, "ua", "127.0.0.1", time.Now().Add(1*time.Hour)); err != nil {
		t.Fatalf("repoInsertToken: %v", err)
	}

	got, err := repo.repoGetSession(ctx, tokenHash)
	if err != nil {
		t.Fatalf("repoGetSession: %v", err)
	}
	var out SessionPayload
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.UserID != "u1" {
		t.Fatalf("unexpected user_id: %v", out.UserID)
	}
}

func TestRepoRotateSession(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	client := rds.NewClient(&rds.Options{Addr: mr.Addr()})
	repo := &AuthRepository{db: nil, redis: client}

	ctx := context.Background()
	oldHash := "oldhash"
	// prepare payload for old session
	payload := SessionPayload{UserID: "u1"}
	b, _ := json.Marshal(payload)
	if err := repo.redis.Set(ctx, "session:"+oldHash, string(b), time.Hour).Err(); err != nil {
		t.Fatalf("set old session: %v", err)
	}

	newHash := "newhash"
	if err := repo.repoRotateSession(ctx, oldHash, newHash, b, time.Hour); err != nil {
		t.Fatalf("repoRotateSession: %v", err)
	}

	if _, err := repo.redis.Get(ctx, "session:"+oldHash).Result(); err == nil {
		t.Fatalf("old session should be gone")
	}
	if _, err := repo.redis.Get(ctx, "session:"+newHash).Result(); err != nil {
		t.Fatalf("new session missing: %v", err)
	}
}
