// Package memlogger provides a simple in-memory logger, intended for testing.
package memlogger

import (
	"container/ring"
	"fmt"
	"strconv"
	"time"

	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/logger"
	"github.com/pkg/errors"
)

type log struct {
	time    time.Time
	level   logger.LogLevel
	message string
}

var now = time.Now

func (l log) String() string {
	return fmt.Sprintf("[%s] [%s] [--] %s\n", l.time.Format(logger.TimeFormat), l.level, l.message)
}

// Logger is an in-memory logger instance. It fulfills both the logger.Logger
// and driver.Logger interfaces
type Logger struct {
	ring *ring.Ring
}

var _ logger.LogWriter = &Logger{}
var _ driver.Logger = &Logger{}

// Init initializes the memory logger. It considers the following configuration
// parameters:
//
// - capacity: The number of log entries to keep in memory. Defaults to 100.
//  - level: The minimum log level to log to the file. (default: info)
func (l *Logger) Init(conf map[string]string) error {
	l.ring = nil
	cap, err := getCapacity(conf)
	if err != nil {
		return err
	}
	l.ring = ring.New(cap)
	return nil
}

func getCapacity(conf map[string]string) (int, error) {
	cap, ok := conf["capacity"]
	if !ok {
		return 100, nil
	}
	c, err := strconv.Atoi(cap)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid capacity '%s'", cap)
	}
	return c, nil
}

// WriteLog logs the message at the designated level.
func (l *Logger) WriteLog(level logger.LogLevel, message string) error {
	l.ring.Value = log{
		time:    now(),
		level:   level,
		message: message,
	}
	l.ring = l.ring.Next()
	return nil
}

// Log returns the requested log.
func (l *Logger) Log(buf []byte, offset int) (int, error) {
	cur := l.ring.Prev()
	var i int
	max := len(buf)
	for i = max; i > 0; {
		if cur.Value == nil {
			// We reached the end of the log
			break
		}
		msg := cur.Value.(log).String()
		if i-len(msg) <= 0 {
			copy(buf[:i], msg[len(msg)-i:])
			i = 0
			break
		}
		copy(buf[i-len(msg):i], msg)
		i = i - len(msg)
		if cur == l.ring {
			// We did a full circle
			break
		}
		cur = cur.Prev()
	}
	if i == 0 {
		return len(buf), nil
	}
	// This means there were fewer logs than requested, so we need to
	// shift the buffer to the beginning.
	len := max - i
	copy(buf, buf[i:])
	return len, nil
}
