package main

import "time"

func parseTime(t interface{}) time.Time {
	switch v := t.(type) {
	case map[string]interface{}:
		u, _ := v["unix"].(int64)
		return time.Unix(u, 0)
	case int64:
		return time.Unix(v, 0)
	default:
		return time.Time{} // ???
	}
}
