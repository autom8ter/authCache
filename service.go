package authCache

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/gorilla/sessions"
	"net/http"
)

type Service struct {
	mux *http.ServeMux
	configs map[string]*Config
	store *sessions.CookieStore
	redisClient *redis.Client
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w,r)
}

func NewService(store *sessions.CookieStore, redisClient *redis.Client, configs map[string]*Config) *Service {
	mux := http.NewServeMux()
	for path, config := range configs {
		mux.HandleFunc(path, config.Callback(store, redisClient))
	}
	return &Service{
		mux:     mux,
		configs: configs,
	}
}

func (s *Service) GetClient(r *http.Request, configName string) (*http.Client, error){
	_, ok := s.configs[configName]
	if !ok {
		return nil,  errors.New(fmt.Sprintf("config does not exist: %s", configName))
	}
	return s.configs[configName].GetClient(r, s.store, s.redisClient)
}
