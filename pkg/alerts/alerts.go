package alerts

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// default heartbeat interval 20 min
const interval = time.Minute * 20
const HeartbeatEvent = "heartbeat event"

type Alert struct {
	Time        time.Time
	Message     string
	Reason      string
	Path        string
	ProcessName string
}

func New(msg, reason, path, procName string) Alert {
	return Alert{
		Time:        time.Now(),
		Message:     msg,
		Reason:      reason,
		Path:        path,
		ProcessName: procName,
	}
}

type Sender interface {
	Send(alert Alert) error
}

var registry = []Sender{}

func Register(s Sender) {
	registry = append(registry, s)
}

func Send(alert Alert) error {
	var errs Errors
	for _, s := range registry {
		if err := s.Send(alert); err != nil {
			errs.collect(err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func Heartbeat(ctx context.Context, logger *logrus.Logger, alert Alert) {
	go func() {
		t := time.NewTicker(interval)
		for {
			select {
			case <-t.C:
				alert.Time = time.Now()
				if err := Send(alert); err != nil {
					logger.WithField("heartbeat", "send").Error(err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
