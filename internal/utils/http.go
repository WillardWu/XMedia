package utils

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func GetRequestScheme(r *http.Request) (scheme string) {
	scheme = "http"
	if IsHTTPSRequest(r) {
		scheme = "https"
	}
	return
}

func GetRequestHref(r *http.Request) (href string) {
	scheme := "http://"
	if IsHTTPSRequest(r) {
		scheme = "https://"
	}
	href = strings.Join([]string{scheme, r.Host, r.RequestURI}, "")
	return
}

func GetRequestHostname(r *http.Request) (hostname string) {
	if _url, err := url.Parse(GetRequestHref(r)); err == nil {
		hostname = _url.Hostname()
	}
	return
}

func GetRequestRemotePort(r *http.Request) (port int) {
	if _, _port, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		port, _ = strconv.Atoi(_port)
	}
	return
}

func GetRequestRemoteIP(r *http.Request) (ip string) {
	if h := r.Header; h != nil {
		ip = strings.TrimSpace(strings.Split(h.Get("X-Forwarded-For"), ",")[0])
		if ip == "" {
			ip = strings.TrimSpace(h.Get("X-Real-IP"))
		}
	}
	if ip == "" || ip == "127.0.0.1" || ip == "localhost" {
		if _ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil && _ip != "" {
			ip = _ip
		}
	}
	return
}

func GetRequestRemoteIPDirectly(r *http.Request) (ip string) {
	if _ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil && _ip != "" {
		ip = _ip
	}
	return
}

func IsHTTPSRequest(r *http.Request) (ret bool) {
	if r.TLS != nil {
		ret = true
		return
	}
	if h := r.Header; h != nil && strings.EqualFold(h.Get("X-Forwarded-Proto"), "https") {
		ret = true
		return
	}
	return
}

func IsAjaxRequest(r *http.Request) (ret bool) {
	if r.Header == nil {
		return
	}
	ret = strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest")
	return
}
