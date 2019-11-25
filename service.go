package authCache

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/gorilla/sessions"
	"net/http"
)

//Service stores configs, serves them as oauth2 callback handlers, and provides the cookiestore and redis client needed by each config
type Service struct {
	mux         *http.ServeMux
	configs     map[string]*Config
	store       *sessions.CookieStore
	redisClient *redis.Client
}

//ServeHTTP satisfies the http.Handler interface so it can be added to a go server
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

//NewService returns an initialized configuration
func NewService(store *sessions.CookieStore, redisClient *redis.Client, configs map[string]*Config) (*Service, error) {
	mux := http.NewServeMux()
	for path, config := range configs {
		if err := config.Validate(); err != nil {
			return nil, err
		}
		mux.HandleFunc(path, config.Callback(store, redisClient))
	}
	return &Service{
		mux:     mux,
		configs: configs,
	}, nil
}

//GetClientByConfig gets an authenticated http.Client for the logged in user from a cached jwt token(if it exists) from the named config
func (s *Service) GetClientByConfig(r *http.Request, configName string) (*http.Client, error) {
	_, ok := s.configs[configName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("config does not exist: %s", configName))
	}
	return s.configs[configName].GetClient(r, s.store, s.redisClient)
}
