package middleware

import (
	"net"
	"net/http"
	"strings"
)

// WithTrustedSubnet — middleware, которое пропускает запрос дальше только если IP-адрес клиента
// из заголовка X-Real-IP входит в заданную подсеть. Если subnet == nil, middleware не должно
// подключаться: вместо этого эндпоинт лучше не регистрировать вовсе. Защитный nil-check
// возвращает 403, чтобы случайное подключение пустого middleware не открывало доступ.
func WithTrustedSubnet(subnet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subnet == nil {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			clientIP := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
			if clientIP == nil || !subnet.Contains(clientIP) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
