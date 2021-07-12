package shared

import (
	"fmt"
	"log"
)

func HandleError(err error, format string, args ...interface{}) {
	if err != nil {
		log.Fatalf("%s | error: %v", fmt.Sprintf(format, args...), err)
	}
	log.Output(2, fmt.Sprintf("%s | success", fmt.Sprintf(format, args...)))
}

func PrintVerbose(format string, args ...interface{}) {
	log.Output(2, fmt.Sprintf(format, args...))
}
