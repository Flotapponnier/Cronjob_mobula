package main

import (
	"fmt"
	"os"
	"time"
)

const logFile = "/var/log/cron.log"

func logInfo(format string, args ...interface{}) {
	message := fmt.Sprintf("%s: %s", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
	fmt.Println(message)
	
	// Also log to file
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer file.Close()
		fmt.Fprintln(file, message)
	}
}

func logError(format string, args ...interface{}) {
	message := fmt.Sprintf("%s: ERROR: %s", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
	fmt.Println(message)
	
	// Also log to file
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer file.Close()
		fmt.Fprintln(file, message)
	}
}