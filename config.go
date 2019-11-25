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

//SessionUIDParam is the parameter that links to a unique identifier of a users session
const SessionUIDParam = "sessionId"

//DefaultCacheDuration is the default number of hours to cache JWT tokens
var DefaultCacheDuration = 730 * time.Minute

//Processor runs a function against a config and an authenticated http client (use to do things like send  to pubsub/save to db etc within the Callback function)
type Processor func(config *Config, client *http.Client) error

//Config contains the required configuration for a Service
type Config struct {
	RedirectTo string
	App        *oauth2.Config
	Do         Processor
}

//NewConfig returns an initialized configuration
func NewConfig(redirectTo string, app *oauth2.Config, do Processor) *Config {
	return &Config{RedirectTo: redirectTo, App: app, Do: do}
}

//Callback returns an http.HandlerFunc that may be used as a Oauth2 callback handler(Authorization code grant).
// Use GetClient() to continue to make api requests after the user visits to other handlers.
func (c *Config) Callback(store *sessions.CookieStore, cache *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			msg := "[Auth] authorization code empty"
			log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		cookie, err := store.Get(r, c.App.Endpoint.AuthURL)
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
		cache.Set(SessionUIDParam, jsonBytes, DefaultCacheDuration)
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

//GetClient gets an authenticated http.Client for the logged in user from a cached jwt token(if it exists)
func (s *Config) GetClient(r *http.Request, store *sessions.CookieStore, cache *redis.Client) (*http.Client, error) {
	cookie, err := store.Get(r, s.App.Endpoint.AuthURL)
	if err != nil || cookie == nil {
		return nil, err
	}
	id, ok := cookie.Values[SessionUIDParam].(string)
	if !ok {
		return nil, errors.New("no user id in cookie")
	}
	tokenJSON, err := cache.Get(id).Bytes()
	if err != nil {
		return nil, err
	}
	var token = &oauth2.Token{}
	if err := json.Unmarshal(tokenJSON, token); err != nil {
		return nil, err
	}
	return s.App.Client(oauth2.NoContext, token), nil
}

//LoginURL returns a url  that begins the oauth2 flow at the oauth2 authorize portal
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
	return nil
}
