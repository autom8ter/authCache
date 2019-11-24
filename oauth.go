package facebook

import (
	"encoding/json"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"github.com/go-redis/redis"
	fb "github.com/huandu/facebook"
	oauth2fb "golang.org/x/oauth2/facebook"
	"time"
)

const IdCookie = "fb_id"
const FBVersion = "v2.4"

type Service struct {
	cache *redis.Client
	cacheExpiration  time.Duration
	dashboardPath string
	app *oauth2.Config
}

type Config struct {
	Cache *redis.Client
	AppID string
	AppSecret string
	Callback string
	Scopes []string
	CacheDuration time.Duration
	DashboardPath string
}

func NewService(c *Config) *Service {
	conf := &oauth2.Config{
		ClientID:     c.AppID,
		ClientSecret: c.AppSecret,
		RedirectURL:  c.Callback,
		Scopes:       c.Scopes,
		Endpoint:     oauth2fb.Endpoint,
	}
	return &Service{
		cache:           c.Cache,
		cacheExpiration: c.CacheDuration,
		dashboardPath:   c.DashboardPath,
		app:             conf,
	}
}

func (s *Service) OAuthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			msg := "[Auth] authorization code empty"
			log.Print(msg)
			http.Error(w,msg,  http.StatusBadRequest)
			return
		}
		token, err := s.app.Exchange(oauth2.NoContext, code)
		if err != nil {
			msg := "[Auth] failed to exchange authorization code for token"
			log.Print(msg)
			http.Error(w, msg,  http.StatusBadRequest)
			return
		}
		client := s.app.Client(oauth2.NoContext, token)
		// Use OAuth2 client with session.
		session := &fb.Session{
			Version:   FBVersion,
			HttpClient: client,
		}
		type ID struct {
			ID string `json:"id"`
		}
		var id = &ID{}
		// Use session.
		res, _ := session.Get("/me", fb.Params{
			"fields": "id",
		})
		if err := res.Decode(id); err != nil {
			msg := "[Auth] failed to decode facebook user id"
			log.Print(msg)
			http.Error(w, msg,  http.StatusBadRequest)
			return
		}
		jsonBytes, err := json.Marshal(token)
		if err != nil {
			msg := "[Auth] failed to marshal jwt"
			log.Print(msg)
			http.Error(w, msg,  http.StatusBadRequest)
		}
		s.cache.Set(id.ID, jsonBytes, s.cacheExpiration)
		r.AddCookie(&http.Cookie{
			Name:       IdCookie,
			Value:      id.ID,
			Expires:    time.Now().Add(s.cacheExpiration),
		})
		http.Redirect(w, r, s.dashboardPath, http.StatusTemporaryRedirect)
	}
}

func (s *Service) GetSession(r *http.Request) (*fb.Session, error) {
	cookie, err := r.Cookie(IdCookie)
	if err != nil {
		return nil, err
	}
	tokenJSON, err := s.cache.Get(cookie.Value).Bytes()
	if err != nil {
		return nil, err
	}
	var token = &oauth2.Token{}
	if err := json.Unmarshal(tokenJSON, token); err != nil {
		return nil, err
	}

	return &fb.Session{
		Version:    FBVersion,
		HttpClient: s.app.Client(oauth2.NoContext, token),
	}, nil
}

func (s *Service) LoginURL(state string) (string) {
	return s.app.AuthCodeURL(state, oauth2.AccessTypeOnline)
}