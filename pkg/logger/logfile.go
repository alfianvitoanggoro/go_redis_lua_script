package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

var logFilePath string

type LogData struct {
	Status       string  `json:"status"`
	Source       string  `json:"source"`
	Payload      any     `json:"payload"`
	ErrorDetails *string `json:"error_details,omitempty"`
	Timestamp    string  `json:"timestamp"`
}

func InitLogFile(path string) {
	logFilePath = path
	dir := filepath.Dir(path)
	perm := os.FileMode(0777)
	if err := os.MkdirAll(dir, perm); err != nil {
		Panic("failed, creating log directory: " + err.Error())
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.OpenFile(path, os.O_CREATE, 0666)
		if err != nil {
			Panic("failed, creating log file: " + err.Error())
		}
		file.Close()
	}
	Infof("✅ Log file initialized at %s", path)
}

// WriteLogToFile menulis log JSON
func WriteLogToFile(status string, source string, payload any, errorDetails *string) {
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		Panic("failed, opening log file: " + err.Error())
	}
	defer file.Close()

	logData := LogData{
		Status:       status,
		Source:       source,
		Payload:      payload,
		ErrorDetails: errorDetails,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	logJSON, err := json.Marshal(logData)
	if err != nil {
		return
	}

	file.Write(append(logJSON, '\n'))
}
