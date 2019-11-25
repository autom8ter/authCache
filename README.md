# authCache
--
    import "github.com/autom8ter/authCache"


## Usage

```go
const SessionUIDParam = "sessionId"
```
SessionUIDParam is the parameter that links to a unique identifier of a users
session

```go
var DefaultCacheDuration = 730 * time.Minute
```
DefaultCacheDuration is the default number of hours to cache JWT tokens

#### type Config

```go
type Config struct {
	RedirectTo string
	App        *oauth2.Config
	Do         Processor
}
```

Config contains the required configuration for a Service

#### func  NewConfig

```go
func NewConfig(redirectTo string, app *oauth2.Config, do Processor) *Config
```
NewConfig returns an initialized configuration

#### func (*Config) Callback

```go
func (c *Config) Callback(store *sessions.CookieStore, cache *redis.Client) http.HandlerFunc
```
Callback returns an http.HandlerFunc that may be used as a Oauth2 callback
handler(Authorization code grant). Use GetClient() to continue to make api
requests after the user visits to other handlers.

#### func (*Config) GetClient

```go
func (s *Config) GetClient(r *http.Request, store *sessions.CookieStore, cache *redis.Client) (*http.Client, error)
```
GetClient gets an authenticated http.Client for the logged in user from a cached
jwt token(if it exists)

#### func (*Config) LoginURL

```go
func (s *Config) LoginURL(state string) string
```
LoginURL returns a url that begins the oauth2 flow at the oauth2 authorize
portal

#### func (*Config) Validate

```go
func (s *Config) Validate() error
```

#### type Processor

```go
type Processor func(config *Config, client *http.Client) error
```

Processor runs a function against a config and an authenticated http client (use
to do things like send to pubsub/save to db etc within the Callback function)
