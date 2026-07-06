package httpapi

import (
	"net/http"
	"strings"
	"time"
)

func validateCSRFCookieRequest(r *http.Request) (string, bool) {
	headerValue := strings.TrimSpace(r.Header.Get(csrfHeaderName))
	cookieValue, ok := cookieValue(r, csrfTokenCookieName)
	if !ok || headerValue == "" || headerValue != cookieValue {
		return "", false
	}
	return headerValue, true
}

func cookieValue(r *http.Request, name string) (string, bool) {
	cookie, err := r.Cookie(name)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return "", false
	}
	return cookie.Value, true
}

func writeRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	if strings.TrimSpace(token) == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     authCookiePath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func writeCSRFCookie(w http.ResponseWriter, token string) {
	if strings.TrimSpace(token) == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfTokenCookieName,
		Value:    token,
		Path:     authCookiePath,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     authCookiePath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfTokenCookieName,
		Value:    "",
		Path:     authCookiePath,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
