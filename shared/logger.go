package shared

import (
	"fmt"
	"log"
	"os"
)

func HandleError(err error, format string, args ...interface{}) {
	if err != nil {
		log.Output(2, fmt.Sprintf("%s | error: %v", fmt.Sprintf(format, args...), err))
		os.Exit(1)
	}
	log.Output(2, fmt.Sprintf("%s | success", fmt.Sprintf(format, args...)))
}

func PrintVerbose(format string, args ...interface{}) {
	log.Output(2, fmt.Sprintf(format, args...))
}
