package main

import (
	"net/http"
	"time"
)

const (
	CSRFToken = "4IhFspJNDLLUcj4kM5lW6nXr7FYtKeLZiq9QnGaU"

	Cookies = "cf_clearance=2uhbGeLDu1bbnJdHFGgzkr6qbGVCkw740_ng66kmQG0-1768791936-1.2.1.1-rhMpi5_5RtWtoLwP5vY.7fH9X9XRRY0t2PIqtC9nzANt7mwVId8Ai9U3cRt9JJZNxJ8TcHJXn22.b5nSowfsxJ6J_qcjv5bLgnTGnOxDrQRCiu0rNIW3cgGLZw3dOCCF1exwlhPzeR97ztVEKawWXO5Z7v4MwBu2ERBoMuznwBpX3dunPw0KbhLEqr_QoV6VvXAVPs1IDTbwjcJWH.L1dMG2d4h06y9ZKBFq7EnTlFo; _fbp=fb.1.1768792105881.346405700682289015; _ga=GA1.2.1307093810.1768792131; XSRF-TOKEN=eyJpdiI6ImduQzV2RWhqa2FyMk9wZE1FenVqNUE9PSIsInZhbHVlIjoib2grSkhIUTRmcHh4a3VxaVRkazk1bGY3amJlcjJRNk1obW0vbUNQSElaWjZhMVAxRFJzQWk1TkttZEs5SVRrVDg4b2tHZFVtYVVQUG9OSXlqbGpDc1gzb0hNYkQvL3NsTkpVZUkxSnlHbmFHWlNPQjE5SzBWTTB4aGpTSEpINGsiLCJtYWMiOiJiMzc1Yjg2NGQ1MWUxM2IxYWVlNTZlYmViNzQxNTVmNWJmYmQzYzc3ZWM0MWI1NzRiZjRkZTE3ZDQ5YmY2NzBiIiwidGFnIjoiIn0%3D; ivas_sms_session=eyJpdiI6Ik1qa1hFdFlPZGp1K09lNEppVTFjK3c9PSIsInZhbHVlIjoiQU9iNll1KzJFTFU1QWs0dElyNkZzK2FyaXhDRHp6RHlXdjkvUktmbUVxWHU4UUczUkVGeFZFb09rNEJYWDhWbkVBWURLSGVzNHpvTGhKZ28rcVNYVDV3bVR5elJZbGliWnBvVmQ1SU04TDJFL2dHbmNJWHJvWlFHa25DUUNIN0MiLCJtYWMiOiIwZTM3MGZmYmU4YzdhNTRkNGI2MzZiM2NkYzU3MDhlYTJhNTJhNjI0ZWVmZDMyNzc3YmE1ZmZmMjBjYzhlZTllIiwidGFnIjoiIn0%3D"

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
