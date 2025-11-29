package handler

import "strings"

func parseCallbackData(data string) map[string]string {
	result := make(map[string]string)

	parts := strings.Split(data, "?")
	if len(parts) < 2 {
		return result
	}

	params := strings.Split(parts[1], "&")
	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}

	return result
}
