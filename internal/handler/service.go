package handler

import (
	"errors"
	"strings"
)

var ErrEmptyURL = errors.New("empty url")

type URLService struct {
	store   *URLStore
	baseURL string
}

func NewURLService(store *URLStore, baseURL string) *URLService {
	return &URLService{
		store:   store,
		baseURL: baseURL,
	}
}

func (s *URLService) Shorten(url string) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", ErrEmptyURL
	}

	id, err := s.store.Save(url)
	if err != nil {
		return "", err
	}

	return s.baseURL + "/" + id, nil
}

func (s *URLService) Resolve(id string) (string, bool) {
	return s.store.Get(id)
}
