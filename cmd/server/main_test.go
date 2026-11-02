package main

import (
	"os"
	"testing"
)

func TestRunServerUAT(t *testing.T) {
	if os.Getenv("RUN_LIVE_UAT_SERVER") != "1" {
		t.Skip("Skipping live server daemon unless RUN_LIVE_UAT_SERVER=1")
	}
	if _, err := os.Stat("./web/index.html"); os.IsNotExist(err) {
		_ = os.Chdir("../..")
	}
	main()
}
