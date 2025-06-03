package clb

import "golang.org/x/time/rate"

var limiter map[string]*rate.Limiter = make(map[string]*rate.Limiter)

func SetRateLimit(limits map[string]int) {
	for apiName, limit := range limits {
		limit := rate.NewLimiter(rate.Limit(limit), 1)
		limiter[apiName] = limit
	}
}
