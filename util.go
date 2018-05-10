package deepjoy

import "math/rand"

func chooseRandom(addrs []string) string {
	if len(addrs) == 0 {
		return ""
	}

	return addrs[rand.Intn(len(addrs))]
}
