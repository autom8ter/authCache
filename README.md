# authCache
--
    import "github.com/autom8ter/authCache"


## Usage

```go
const SessionUIDParam = "sessionId"
```

#### type Config

```go
type Config struct {
	RedirectTo    string
	SessionName   string
	App           *oauth2.Config
	Cache         *redis.Client
	CacheDuration time.Duration
	Do            Processor
}
```

Config contains the required configuration for a Service

#### func  NewConfig

```go
func NewConfig(redirectTo string, sessionName string, app *oauth2.Config, cache *redis.Client, cacheDuration time.Duration, do Processor) *Config
```

#### func (*Config) Callback

```go
func (c *Config) Callback(store *sessions.CookieStore) http.HandlerFunc
```
Callback returns an http.HandlerFunc that may be used as a facebook Oauth2
callback handler(Authorization code grant)

#### func (*Config) GetClient

```go
func (s *Config) GetClient(r *http.Request, store *sessions.CookieStore) (*http.Client, error)
```
GetSession gets the users session from the incoming http request

#### func (*Config) LoginURL

```go
func (s *Config) LoginURL(state string) string
```
LoginURL returns a url that begins the oauth2 flow at facebooks login portal

#### func (*Config) Validate

```go
func (s *Config) Validate() error
```

#### type Processor

```go
type Processor func(config *Config, client *http.Client) error
```
