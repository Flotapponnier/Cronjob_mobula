package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
)

func logInfo(format string, args ...interface{}) {
	timestamp := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf(format, args...)

	coloredContent := colorizeLogContent(content)

	message := fmt.Sprintf("%s: %s", timestamp, coloredContent)
	fmt.Println(message)
}

func logError(format string, args ...interface{}) {
	timestamp := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf(format, args...)

	message := fmt.Sprintf("%s: %sERROR: %s%s", timestamp, ColorRed, content, ColorReset)
	fmt.Println(message)
}

func colorizeLogContent(content string) string {
	if strings.Contains(content, "Successfully uploaded to cloud") {
		parts := strings.SplitN(content, ": ", 2)
		if len(parts) == 2 {
			return fmt.Sprintf("%s%s%s: %s%s%s", ColorGreen, parts[0], ColorReset, ColorBlue, parts[1], ColorReset)
		}
		return fmt.Sprintf("%s%s%s", ColorGreen, content, ColorReset)
	}

	if strings.Contains(content, "Created snapshot directory structure") ||
		strings.Contains(content, "Snapshot will be saved as") ||
		strings.Contains(content, "Encrypted snapshot") && strings.Contains(content, "has been saved") {
		parts := strings.SplitN(content, ": ", 2)
		if len(parts) == 2 {
			return fmt.Sprintf("%s: %s%s%s", parts[0], ColorBlue, parts[1], ColorReset)
		}
	}

	return content
}

func logSectionStart(title string) {
	fmt.Println("---------------------------------------")
	logInfo("%s", title)
	fmt.Println("---------------------------------------")
}

func logSectionEnd() {
	fmt.Println("---------------------------------------")
}

