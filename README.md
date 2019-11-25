# facebook
--
    import "github.com/autom8ter/facebook"


## Usage

```go
const CookieName = "fb_auth"
```
IdCookie is then name of the cookie that contains a facebook users id(used to
query redis for jwt token)

```go
var FBVersion = "v2.4"
```
FBVersion is the default version of the Facebook API

#### type Auth

```go
type Auth struct {
}
```

OAuth2 handles Oauth2(Authorization code grant) requests and caches the users
token in redis for reuse

#### func  NewAuth

```go
func NewAuth(c *Config) (*Auth, error)
```
NewService initializes a new service instance

#### func (*Auth) Callback

```go
func (s *Auth) Callback() http.HandlerFunc
```
Callback returns an http.HandlerFunc that may be used as a facebook Oauth2
callback handler(Authorization code grant)

#### func (*Auth) Do

```go
func (s *Auth) Do(r *http.Request, fn AuthFunc, target interface{}) error
```
Do gets the currently logged in users session, then runs a function against it

#### func (*Auth) GetSession

```go
func (s *Auth) GetSession(r *http.Request) (*Session, error)
```
GetSession gets the users session from the incoming http request

#### func (*Auth) LoginURL

```go
func (s *Auth) LoginURL(state string) string
```
LoginURL returns a url that begins the oauth2 flow at facebooks login portal

#### type AuthFunc

```go
type AuthFunc func(*Session) (fb.Result, error)
```

AuthFunc is a first-class function used to run logic against an incoming http
request

#### func (AuthFunc) Do

```go
func (a AuthFunc) Do(s *Session, target interface{}) error
```

#### type Config

```go
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
```

Config contains the required configuration for a Service

#### type Session

```go
type Session struct {
	Cache           *redis.Client
	WebSession      *sessions.Session
	FacebookSession *fb.Session
}
```

Session holds a redis client, a secure web session, & a facebook session for
make api requests
