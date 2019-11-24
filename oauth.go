package facebook

import (
	"encoding/json"
	"github.com/go-redis/redis"
	fb "github.com/huandu/facebook"
	"golang.org/x/oauth2"
	oauth2fb "golang.org/x/oauth2/facebook"
	"log"
	"net/http"
	"time"
)

//IdCookie is then name of the cookie that contains a facebook users id(used to query redis for jwt token)
const IdCookie = "fb_id"

//FBVersion is the default version of the Facebook API
var FBVersion = "v2.4"

//OAuth2 handles Oauth2(Authorization code grant) requests and caches the users token in redis for reuse
type Auth struct {
	cache           *redis.Client
	cacheExpiration time.Duration
	dashboardPath   string
	app             *oauth2.Config
}

//Config contains the required configuration for a Service
type Config struct {
	Cache         *redis.Client
	AppID         string
	AppSecret     string
	Callback      string
	Scopes        []string
	CacheDuration time.Duration
	DashboardPath string
}

//NewService initializes a new service instance
func NewAuth(c *Config) *Auth {
	conf := &oauth2.Config{
		ClientID:     c.AppID,
		ClientSecret: c.AppSecret,
		RedirectURL:  c.Callback,
		Scopes:       c.Scopes,
		Endpoint:     oauth2fb.Endpoint,
	}
	return &Auth{
		cache:           c.Cache,
		cacheExpiration: c.CacheDuration,
		dashboardPath:   c.DashboardPath,
		app:             conf,
	}
}

//Callback returns an http.HandlerFunc that may be used as a facebook Oauth2 callback handler(Authorization code grant)
func (s *Auth) Callback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			msg := "[Auth] authorization code empty"
			log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		token, err := s.app.Exchange(oauth2.NoContext, code)
		if err != nil {
			msg := "[Auth] failed to exchange authorization code for token"
			log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		client := s.app.Client(oauth2.NoContext, token)
		// Use OAuth2 client with session.
		session := &fb.Session{
			Version:    FBVersion,
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
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		jsonBytes, err := json.Marshal(token)
		if err != nil {
			msg := "[Auth] failed to marshal jwt"
			log.Print(msg)
			http.Error(w, msg, http.StatusBadRequest)
		}
		s.cache.Set(id.ID, jsonBytes, s.cacheExpiration)
		r.AddCookie(&http.Cookie{
			Name:    IdCookie,
			Value:   id.ID,
			Expires: time.Now().Add(s.cacheExpiration),
		})
		http.Redirect(w, r, s.dashboardPath, http.StatusTemporaryRedirect)
	}
}

//GetSession gets the users session from the incoming http request
func (s *Auth) GetSession(r *http.Request) (*fb.Session, error) {
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

//LoginURL returns a url  that begins the oauth2 flow at facebooks login portal
func (s *Auth) LoginURL(state string) string {
	return s.app.AuthCodeURL(state, oauth2.AccessTypeOnline)
}
