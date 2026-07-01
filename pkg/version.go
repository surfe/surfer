package pkg

import (
	"fmt"
	"strconv"
	"time"
)

var (
	BuildVersion = "main"
	BuildCommit  = ""
	BuildTime    = ""
)

func PrintVersion() {
	fmt.Printf("Version: %s\n", BuildVersion)
	fmt.Printf("Commit: %s\n", BuildCommit)
	timestamp, err := strconv.ParseInt(BuildTime, 10, 64)
	if err == nil {
		BuildTime = time.Unix(timestamp, 0).String()
	}
	fmt.Printf("Build Time: %s\n", BuildTime)
}
