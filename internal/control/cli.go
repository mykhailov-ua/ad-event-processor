package control

import (
	"net/http"
	"os"
)

func ProbeHealth(args []string) bool {
	if len(args) > 2 && args[1] == "--health-probe" {
		resp, err := http.Get(args[2])
		if err != nil || resp.StatusCode != 200 {
			os.Exit(1)
		}
		os.Exit(0)
	}
	return false
}
