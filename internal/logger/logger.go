package logger

import (
	"log"
	"os"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.LUTC)
}

func Info(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

func Error(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

func Fatal(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}
