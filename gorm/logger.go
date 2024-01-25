package duty

import (
	"log"
	"os"
	"time"

	gLogger "gorm.io/gorm/logger"
)

const Debug = "debug"
const Trace = "trace"

func NewGormLogger(level string) gLogger.Interface {
	if level == Trace {
		return gLogger.New(
			log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile), // io writer
			gLogger.Config{
				SlowThreshold:             time.Second,  // Slow SQL threshold
				LogLevel:                  gLogger.Info, // Log level
				IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      false,        // Don't include params in the SQL log
				Colorful:                  true,         // Disable color,
			},
		)
	}

	if level == Debug {
		return gLogger.New(
			log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile), // io writer
			gLogger.Config{
				SlowThreshold:             time.Second,  // Slow SQL threshold
				LogLevel:                  gLogger.Info, // Log level
				IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      true,         // Don't include params in the SQL log
				Colorful:                  true,         // Disable color
			},
		)
	}

	return gLogger.New(
		log.New(os.Stdout, "\r\n", log.Ldate|log.Ltime|log.Lshortfile), // io writer
		gLogger.Config{
			SlowThreshold:             time.Second,    // Slow SQL threshold
			LogLevel:                  gLogger.Silent, // Log level
			IgnoreRecordNotFoundError: true,           // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,          // Don't include params in the SQL log
			Colorful:                  true,           // Disable color
		},
	)
}
