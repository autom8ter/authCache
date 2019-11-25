//go:generate godocdown -o README.md

package authCache

import (
	"encoding/json"
	"errors"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"time"
)

const SessionUIDParam = "sessionId"

type Processor func(config *Config, client *http.Client) error

//Config contains the required configuration for a Service
type Config struct {
	RedirectTo    string
	SessionName   string
	App           *oauth2.Config
	Cache         *redis.Client
	CacheDuration time.Duration
	Do            Processor
}

func NewConfig(redirectTo string, sessionName string, app *oauth2.Config, cache *redis.Client, cacheDuration time.Duration, do Processor) *Config {
	return &Config{RedirectTo: redirectTo, SessionName: sessionName, App: app, Cache: cache, CacheDuration: cacheDuration, Do: do}
}

//Callback returns an http.HandlerFunc that may be used as a facebook Oauth2 callback handler(Authorization code grant)
func (c *Config) Callback(store *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			msg := "[Auth] authorization code empty"
			log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		cookie, err := store.Get(r, c.SessionName)
		if err != nil || cookie == nil {
			msg := "[Auth] failed to get session cookie"
			log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
		}
		defer func() {
			if err := cookie.Save(r, w); err != nil {
				log.Print(err.Error())
			}
		}()
		uid := uuid.New()
		cookie.Values[SessionUIDParam] = uid
		token, err := c.App.Exchange(oauth2.NoContext, code)
		if err != nil {
			msg := "[Auth] failed to exchange authorization code for token"
			log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		jsonBytes, err := json.Marshal(token)
		if err != nil {
			msg := "[Auth] failed to marshal jwt"
			log.Print(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		c.Cache.Set(SessionUIDParam, jsonBytes, c.CacheDuration)
		client := c.App.Client(oauth2.NoContext, token)
		if c.Do != nil {
			if err := c.Do(c, client); err != nil {
				msg := "[Auth] failed to process function"
				log.Print(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
		}
		http.Redirect(w, r, c.RedirectTo, http.StatusTemporaryRedirect)
	}
}

//GetSession gets the users session from the incoming http request
func (s *Config) GetClient(r *http.Request, store *sessions.CookieStore) (*http.Client, error) {
	cookie, err := store.Get(r, s.SessionName)
	if err != nil || cookie == nil {
		return nil, err
	}
	id, ok := cookie.Values[SessionUIDParam].(string)
	if !ok {
		return nil, errors.New("no user id in cookie")
	}
	tokenJSON, err := s.Cache.Get(id).Bytes()
	if err != nil {
		return nil, err
	}
	var token = &oauth2.Token{}
	if err := json.Unmarshal(tokenJSON, token); err != nil {
		return nil, err
	}
	return s.App.Client(oauth2.NoContext, token), nil
}

//LoginURL returns a url  that begins the oauth2 flow at facebooks login portal
func (s *Config) LoginURL(state string) string {
	return s.App.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (s *Config) Validate() error {
	if s == nil {
		return errors.New("empty auth struct")
	}
	if s.App == nil {
		return errors.New("empty oauth2 config")
	}
	if s.Cache == nil {
		return errors.New("empty cache")
	}
	if s.RedirectTo == "" {
		return errors.New("empty redirectTo path")
	}
	if len(s.App.Scopes) == 0 {
		return errors.New("empty oauth2 scopes")
	}
	if s.App.ClientID == "" {
		return errors.New("empty oauth2 clientId")
	}
	if s.App.ClientSecret == "" {
		return errors.New("empty oauth2 client secret")
	}
	if s.App.RedirectURL == "" {
		return errors.New("empty oauth2 redirect")
	}
	return s.Cache.Ping().Err()
}
