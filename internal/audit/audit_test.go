package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockObserver struct {
	events []AuditEvent
}

func (m *mockObserver) Notify(e AuditEvent) {
	m.events = append(m.events, e)
}

func TestAuditService_Notify(t *testing.T) {
	svc := NewAuditService()
	obs := &mockObserver{}
	svc.Register(obs)

	svc.Notify(AuditEvent{Action: "shorten", URL: "https://example.com"})
	require.Len(t, obs.events, 1)
	assert.Equal(t, "shorten", obs.events[0].Action)
	assert.Equal(t, "https://example.com", obs.events[0].URL)
	assert.NotZero(t, obs.events[0].TS)
}

func TestAuditService_MultipleObservers(t *testing.T) {
	svc := NewAuditService()
	obs1 := &mockObserver{}
	obs2 := &mockObserver{}
	svc.Register(obs1)
	svc.Register(obs2)

	svc.Notify(AuditEvent{Action: "resolve", URL: "https://example.com"})
	assert.Len(t, obs1.events, 1)
	assert.Len(t, obs2.events, 1)
}

func TestAuditService_NoObservers(t *testing.T) {
	svc := NewAuditService()
	// не должно паниковать
	svc.Notify(AuditEvent{Action: "test", URL: "https://example.com"})
}

func TestFileObserver_Notify(t *testing.T) {
	f, err := os.CreateTemp("", "audit_test_*.jsonl")
	require.NoError(t, err)
	f.Close()
	defer os.Remove(f.Name())

	obs := NewFileObserver(f.Name())
	obs.Notify(AuditEvent{Action: "shorten", URL: "https://example.com", UserID: "user1", TS: 123456})

	data, err := os.ReadFile(f.Name())
	require.NoError(t, err)

	var event AuditEvent
	err = json.Unmarshal(data[:len(data)-1], &event) // отбрасываем завершающий перевод строки
	require.NoError(t, err)
	assert.Equal(t, "shorten", event.Action)
	assert.Equal(t, "https://example.com", event.URL)
}

func TestFileObserver_InvalidPath(t *testing.T) {
	obs := NewFileObserver("/nonexistent/path/audit.jsonl")
	// не должно паниковать
	obs.Notify(AuditEvent{Action: "test", URL: "https://example.com"})
}

func TestHTTPObserver_Notify(t *testing.T) {
	var received AuditEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := NewHTTPObserver(srv.URL)
	obs.Notify(AuditEvent{Action: "shorten", URL: "https://example.com", TS: 999})

	assert.Equal(t, "shorten", received.Action)
	assert.Equal(t, "https://example.com", received.URL)
}

func TestHTTPObserver_InvalidURL(t *testing.T) {
	obs := NewHTTPObserver("http://127.0.0.1:0")
	// не должно паниковать при отказе соединения
	obs.Notify(AuditEvent{Action: "test", URL: "https://example.com"})
}
