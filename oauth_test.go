package facebook_test

import (
	"fmt"
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
		Callback:      "https://autom8ter.com/oauth/facebook",
		Scopes:        []string{"email"},
		CacheDuration: 730 * time.Hour, //1 month
		DashboardPath: "/home",
	}
	auth, err = facebook.NewAuth(config)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestAuth_LoginURL(t *testing.T) {
	url := auth.LoginURL("")
	if url == "" {
		t.Fatal("empty login url")
	}
	fmt.Println(url)
}
