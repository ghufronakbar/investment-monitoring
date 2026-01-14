package utils

import (
	"net"
	"net/http"
	"strings"

	"github.com/mssola/user_agent"
)

type ClientInfo struct {
	UserAgent string
	ClientIP  string
	Browser   string
	Platform  string // OS (Windows, Mac, Linux, Android)
	OS        string // OS Version
	Device    string // Mobile / Desktop / Bot
}

func ExtractClientInfo(r *http.Request) ClientInfo {
	uaStr := r.UserAgent()
	ua := user_agent.New(uaStr)

	name, version := ua.Browser()
	browser := name + " " + version

	// Extract IP (Handle Proxy / Cloudflare)
	clientIP := getClientIP(r)

	device := "Desktop"
	if ua.Mobile() {
		device = "Mobile"
	} else if ua.Bot() {
		device = "Bot"
	}

	return ClientInfo{
		UserAgent: uaStr,
		ClientIP:  clientIP,
		Browser:   browser,
		Platform:  ua.Platform(), // ex: Windows, Linux, Macintosh
		OS:        ua.OS(),       // ex: Intel Mac OS X 10_15_7
		Device:    device,
	}
}

// Helper untuk mendapatkan Real IP (menangani X-Forwarded-For)
func getClientIP(r *http.Request) string {
	// 1. Cek Header Cloudflare
	cfIP := r.Header.Get("CF-Connecting-IP")
	if cfIP != "" {
		return cfIP
	}

	// 2. Cek X-Forwarded-For (Standard Proxy)
	// Format: client, proxy1, proxy2
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 3. Cek X-Real-IP
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// 4. Fallback ke RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
