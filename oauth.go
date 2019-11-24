//go:generate godocdown -o README.md

package facebook

import (
	"encoding/json"
	"errors"
	"github.com/go-redis/redis"
	"github.com/gorilla/sessions"
	fb "github.com/huandu/facebook"
	"golang.org/x/oauth2"
	oauth2fb "golang.org/x/oauth2/facebook"
	"log"
	"net/http"
	"time"
)

//IdCookie is then name of the cookie that contains a facebook users id(used to query redis for jwt token)
const CookieName = "fb_auth"

//FBVersion is the default version of the Facebook API
var FBVersion = "v2.4"

//OAuth2 handles Oauth2(Authorization code grant) requests and caches the users token in redis for reuse
type Auth struct {
	cache           *redis.Client
	cacheExpiration time.Duration
	dashboardPath   string
	app             *oauth2.Config
	cookieStore     *sessions.CookieStore
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
	SessionSecret string
}

//NewService initializes a new service instance
func NewAuth(c *Config) (*Auth, error) {
	conf := &oauth2.Config{
		ClientID:     c.AppID,
		ClientSecret: c.AppSecret,
		RedirectURL:  c.Callback,
		Scopes:       c.Scopes,
		Endpoint:     oauth2fb.Endpoint,
	}
	a := &Auth{
		cache:           c.Cache,
		cacheExpiration: c.CacheDuration,
		dashboardPath:   c.DashboardPath,
		app:             conf,
		cookieStore:     sessions.NewCookieStore([]byte(c.SessionSecret)),
	}
	if err := a.validate(); err != nil {
		return nil, err
	}
	return a, nil
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
		cookie, err := s.cookieStore.Get(r, CookieName)
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

		cookie.Values["id"] = id.ID

		http.Redirect(w, r, s.dashboardPath, http.StatusTemporaryRedirect)
	}
}

//GetSession gets the users session from the incoming http request
func (s *Auth) GetSession(r *http.Request) (*fb.Session, error) {
	cookie, err := s.cookieStore.Get(r, CookieName)
	if err != nil || cookie == nil {
		return nil, err
	}
	id, ok := cookie.Values["id"].(string)
	if !ok {
		return nil, errors.New("no user id in cookie")
	}
	tokenJSON, err := s.cache.Get(id).Bytes()
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

func (s *Auth) Do(r *http.Request, fn func(session *fb.Session) (fb.Result, error)) (fb.Result, error) {
	sess, err := s.GetSession(r)
	if err != nil {
		return nil, err
	}
	return fn(sess)
}

//LoginURL returns a url  that begins the oauth2 flow at facebooks login portal
func (s *Auth) LoginURL(state string) string {
	return s.app.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (s *Auth) validate() error {
	if s == nil {
		return errors.New("empty auth struct")
	}
	if s.app == nil {
		return errors.New("empty oauth2 config")
	}
	if s.cache == nil {
		return errors.New("empty cache")
	}
	if s.dashboardPath == "" {
		return errors.New("empty dashboard path")
	}
	if len(s.app.Scopes) == 0 {
		return errors.New("empty oauth2 scopes")
	}
	if s.app.ClientID == "" {
		return errors.New("empty oauth2 clientId")
	}
	if s.app.ClientSecret == "" {
		return errors.New("empty oauth2 client secret")
	}
	if s.app.RedirectURL == "" {
		return errors.New("empty oauth2 redirect")
	}
	if s.cookieStore == nil {
		return errors.New("empty cookie store")
	}
	return s.cache.Ping().Err()
}
