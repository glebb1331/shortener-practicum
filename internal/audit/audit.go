package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

// AuditEvent описывает одно аудит-событие в системе.
type AuditEvent struct {
	TS     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

// Observer — интерфейс получателя аудит-событий.
type Observer interface {
	Notify(AuditEvent)
}

// AuditService рассылает события всем зарегистрированным наблюдателям.
type AuditService struct {
	observers []Observer
}

// FileObserver записывает аудит-события в файл в формате JSONL.
type FileObserver struct {
	filePath string
}

// HTTPObserver отправляет аудит-события на HTTP-эндпоинт.
type HTTPObserver struct {
	urlString string
}

// NewAuditService создаёт новый AuditService без наблюдателей.
func NewAuditService() *AuditService {
	return &AuditService{
		observers: make([]Observer, 0),
	}
}

// NewFileObserver создаёт наблюдателя, пишущего события в filePath.
func NewFileObserver(filePath string) *FileObserver {
	return &FileObserver{filePath: filePath}
}

// NewHTTPObserver создаёт наблюдателя, отправляющего события на url.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{urlString: url}
}

// Register добавляет наблюдателя в список рассылки.
func (as *AuditService) Register(o Observer) {
	as.observers = append(as.observers, o)
}

// Notify рассылает событие всем зарегистрированным наблюдателям.
func (as *AuditService) Notify(event AuditEvent) {
	event.TS = time.Now().Unix()
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
