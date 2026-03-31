package main

import (
	"net/http"
	"time"
)

const (
	CSRFToken = "4IhFspJNDLLUcj4kM5lW6nXr7FYtKeLZiq9QnGaU"

	Cookies = "cf_clearance=2uhbGeLDu1bbnJdHFGgzkr6qbGVCkw740_ng66kmQG0-1768791936-1.2.1.1-rhMpi5_5RtWtoLwP5vY.7fH9X9XRRY0t2PIqtC9nzANt7mwVId8Ai9U3cRt9JJZNxJ8TcHJXn22.b5nSowfsxJ6J_qcjv5bLgnTGnOxDrQRCiu0rNIW3cgGLZw3dOCCF1exwlhPzeR97ztVEKawWXO5Z7v4MwBu2ERBoMuznwBpX3dunPw0KbhLEqr_QoV6VvXAVPs1IDTbwjcJWH.L1dMG2d4h06y9ZKBFq7EnTlFo; _fbp=fb.1.1768792105881.346405700682289015; _ga=GA1.2.1307093810.1768792131; XSRF-TOKEN=eyJpdiI6IkFRSlZwYXQxdldiczUyc292cEtVMVE9PSIsInZhbHVlIjoiM3orY0lOT1RhbTE2RkpUK0hvd1NMRDNFTitzRUFsTFZkVFZUc2EvaUd5OERYelBsbk1SQlNxQmhhZmszTFU3NGttOXlxczRFYi9RYjFNLzV5WnNTVjdXbk83eUdIV0xwcllPejN0K3FSQ1JqRHlVSEFPYkRraS82Qjh6VzZreEEiLCJtYWMiOiJhYmI5ZDNlZWU3Y2Q0ZWQ2MmY2ZGNiYTI3MDMxYzE1OTI1NWE1YzI3MDNlZDBiZDVmNjJlZjVjZWY5MmJmYjY1IiwidGFnIjoiIn0%3D; ivas_sms_session=eyJpdiI6Im5EaGdhR0V5d1p6a1ZFMWY0K05sSkE9PSIsInZhbHVlIjoiZDRXZEVxZnJHalRwWUsyZ2crQ2d4SkVLckMydUd6UzdKcDhYZ04zT2orNVhFKzhNQkl3emZEaHVuUEpxWHI5L0pYY21xclA3VlRJUXZGRmliZVdoTkFIc1VWSkpxZVF4cXZveFVQQ0tHNGZBUzl2aWVnYUQwTUtFdTRJRUhKb3MiLCJtYWMiOiJlNmMzODM2MDFiODQxMTg5MzkxMDczNGU5ODJjMDJlOGQ5OGVhMDY1NDg4NDE2ODNlZTQ0ZmI5ZDQxMWExZDExIiwidGFnIjoiIn0%3D"

	UserAgent = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Mobile Safari/537.36"
)

func getStartDate() string {
	loc, err := time.LoadLocation("Asia/Karachi")
	now := time.Now()
	if err == nil {
		now = now.In(loc)
	}
	return now.AddDate(0, 0, -1).Format("2006-01-02")
}

func getEndDate() string {
	loc, err := time.LoadLocation("Asia/Karachi")
	now := time.Now()
	if err == nil {
		now = now.In(loc)
	}
	return now.AddDate(0, 0, 1).Format("2006-01-02")
}

func setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Cookie", Cookies)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-CSRF-TOKEN", CSRFToken)
	req.Header.Set("Accept", "text/html, */*; q=0.01")
}
