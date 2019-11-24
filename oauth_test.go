package facebook_test

import (
	"github.com/autom8ter/facebook"
	"github.com/go-redis/redis"
	"os"
	"testing"
	"time"
)

var auth *facebook.Auth
var err error

func TestNewAuth(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	config := &facebook.Config{
		Cache:         redisClient,
		AppID:         os.Getenv("FACEBOOK_APP_ID"),
		AppSecret:     os.Getenv("FACEBOOK_APP_SECRET"),
		Callback:      "localhost:8080",
		Scopes:        []string{"email"},
		CacheDuration: 730 *time.Hour, //1 month
		DashboardPath: "/home",
	}
	auth, err = facebook.NewAuth(config)
	if err != nil {
		t.Fatal(err.Error())
	}
}
