package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kings0x/crossPost/internal/cache"
	"github.com/kings0x/crossPost/internal/db"
	rds "github.com/redis/go-redis/v9"
)

type oauthStatePayload struct {
	UserID   string `json:"user_id"`
	Platform string `json:"platform"`
	Intent   string `json:"intent"`
}

func main() {
	var dsn string
	var redisURL string
	flag.StringVar(&dsn, "db", os.Getenv("DATABASE_URL"), "database DSN")
	flag.StringVar(&redisURL, "redis", os.Getenv("REDIS_URL"), "redis url")
	flag.Parse()

	if dsn == "" || redisURL == "" {
		log.Fatal("both -db and -redis (or env DATABASE_URL and REDIS_URL) must be set")
	}

	ctx := context.Background()

	pg, err := db.NewDatabase(ctx, dsn)
	if err != nil {
		log.Fatalf("db.NewDatabase: %v", err)
	}
	defer pg.Close()

	rc, err := cache.NewRedis(ctx, redisURL)
	if err != nil {
		log.Fatalf("cache.NewRedis: %v", err)
	}
	defer rc.Close()

	// migrate oauth_states
	migrateOauthStates(ctx, pg, rc)
	migrateUserTokens(ctx, pg, rc)
	migrateUserSessions(ctx, pg, rc)

	fmt.Println("migration to Redis complete")
}

func migrateOauthStates(ctx context.Context, pg *db.Database, rc *rds.Client) {
	rows, err := pg.Pool.Query(ctx, "SELECT states, user_id, platform, intent, expires_at FROM oauth_states")
	if err != nil {
		log.Printf("migrateOauthStates: %v", err)
		return
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var id string
		var userID string
		var platform string
		var intent string
		var expires sql.NullTime

		if err := rows.Scan(&id, &userID, &platform, &intent, &expires); err != nil {
			log.Printf("migrateOauthStates scan: %v", err)
			continue
		}

		payload := oauthStatePayload{UserID: userID, Platform: platform, Intent: intent}
		b, _ := json.Marshal(payload)

		key := "oauth_state:" + id
		var ttl time.Duration = 5 * time.Minute
		if expires.Valid {
			if t := time.Until(expires.Time); t > 0 {
				ttl = t
			}
		}

		if err := rc.Set(ctx, key, b, ttl).Err(); err != nil {
			log.Printf("migrateOauthStates set: %v", err)
			continue
		}
		count++
	}

	log.Printf("migrateOauthStates: migrated %d rows", count)
}

func migrateUserTokens(ctx context.Context, pg *db.Database, rc *rds.Client) {
	rows, err := pg.Pool.Query(ctx, "SELECT token_hash, user_id, kind, expires_at FROM user_tokens")
	if err != nil {
		log.Printf("migrateUserTokens: %v", err)
		return
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var tokenHash string
		var userID string
		var kind string
		var expires time.Time

		if err := rows.Scan(&tokenHash, &userID, &kind, &expires); err != nil {
			log.Printf("migrateUserTokens scan: %v", err)
			continue
		}

		payload := map[string]string{"user_id": userID, "kind": kind}
		b, _ := json.Marshal(payload)
		key := "token:" + tokenHash
		ttl := time.Until(expires)
		if ttl <= 0 {
			ttl = 5 * time.Minute
		}

		if err := rc.Set(ctx, key, b, ttl).Err(); err != nil {
			log.Printf("migrateUserTokens set: %v", err)
			continue
		}
		count++
	}

	log.Printf("migrateUserTokens: migrated %d rows", count)
}

func migrateUserSessions(ctx context.Context, pg *db.Database, rc *rds.Client) {
	rows, err := pg.Pool.Query(ctx, "SELECT token_hash, user_id, user_agent, ip_address, expires_at FROM user_sessions")
	if err != nil {
		log.Printf("migrateUserSessions: %v", err)
		return
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var tokenHash string
		var userID string
		var userAgent sql.NullString
		var ip sql.NullString
		var expires time.Time

		if err := rows.Scan(&tokenHash, &userID, &userAgent, &ip, &expires); err != nil {
			log.Printf("migrateUserSessions scan: %v", err)
			continue
		}

		payload := map[string]interface{}{"user_id": userID, "user_agent": userAgent.String, "ip_address": ip.String}
		b, _ := json.Marshal(payload)
		key := "session:" + tokenHash
		ttl := time.Until(expires)
		if ttl <= 0 {
			ttl = 30 * 24 * time.Hour
		}

		if err := rc.Set(ctx, key, b, ttl).Err(); err != nil {
			log.Printf("migrateUserSessions set: %v", err)
			continue
		}
		count++
	}

	log.Printf("migrateUserSessions: migrated %d rows", count)
}
