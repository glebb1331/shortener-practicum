package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

type AuditEvent struct {
	Ts     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

type Observer interface {
	Notify(AuditEvent)
}

type AuditService struct {
	observers []Observer
}

type FileObserver struct {
	filePath string
}

type HTTPObserver struct {
	urlString string
}

func NewAuditService() *AuditService {
	return &AuditService{
		observers: make([]Observer, 0),
	}
}

func NewFileObserver(filePath string) *FileObserver {
	return &FileObserver{filePath: filePath}
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{urlString: url}
}

func (as *AuditService) Register(o Observer) {
	as.observers = append(as.observers, o)
}

func (as *AuditService) Notify(event AuditEvent) {
	event.Ts = time.Now().Unix()
	for _, o := range as.observers {
		o.Notify(event)
	}
}

func (fo *FileObserver) Notify(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	f, err := os.OpenFile(fo.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}

func (ho *HTTPObserver) Notify(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	resp, err := http.Post(ho.urlString, "application/json", bytes.NewReader(data))
	if err != nil {
		return
	}
	resp.Body.Close()
}
