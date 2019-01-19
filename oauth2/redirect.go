package oauth2

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
)

type redirectError struct {
	u    *url.URL
	code int
}

func SendRedirect(w http.ResponseWriter, url string, code int) {
	w.Header().Set("Location", url)
	w.WriteHeader(code) // http.StatusFound
	fmt.Fprintf(w, "You are being redirected to <a href=\"%s\">%s</a>. If you see this message, please manually follow the link.", html.EscapeString(url), html.EscapeString(url))
	// try various stuff to cause the redirect to happen in case header failed to happen
	if js, err := json.Marshal(url); err == nil {
		fmt.Fprintf(w, "<script language=\"javascript\">window.location = %s;</script>", js)
	}
	fmt.Fprintf(w, "<meta http-equiv=\"Refresh\" content=\"0; url=%s\"/>", html.EscapeString(url))
	fmt.Fprintf(w, "<meta http-equiv=\"Location\" content=\"%s\"/>", html.EscapeString(url))
}

// code can be one of http.StatusMovedPermanently or http.StatusFound or
// any 3xx http status code
func RedirectErrorCode(u *url.URL, code int) error {
	// generate a redirect error
	n := &redirectError{u: new(url.URL), code: code}
	// copy url
	*n.u = *u

	return n
}

func RedirectError(u *url.URL) error {
	// generate a redirect error
	n := &redirectError{u: new(url.URL), code: http.StatusFound}
	// copy url
	*n.u = *u

	return n
}

func (e *redirectError) Error() string {
	return fmt.Sprintf("Redirect required to %s", e.u)
}

func (e *redirectError) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	SendRedirect(w, e.u.String(), e.code)
}

func (e *redirectError) URL() *url.URL {
	return e.u
}
