package notifier

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/effective-security/x/values"
	"github.com/effective-security/xdb"
	"github.com/effective-security/xlog"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

var logger = xlog.NewPackageLogger("github.com/effective-security/xdb", "notifier")

const (
	DefaultMinReconnectInterval = 10 * time.Millisecond
	DefaultMaxReconnectInterval = time.Minute
)

type Notification struct {
	Channel string
	Payload values.MapAny
	// RawPayload, or the empty string if unspecified.
	RawPayload string
}

// Listener interface connects to the database and allows callers to listen to a
// particular topic by issuing a LISTEN command. WaitForNotification blocks
// until receiving a notification or until the supplied context expires. The
// default implementation is tightly coupled to pgx (following River's
// implementation), but callers may implement their own listeners for any
// backend they'd like.
type Listener interface {
	io.Closer
	Listen(ctx context.Context, topic string, callback func(n *Notification)) error
}

type listener struct {
	listener *pq.Listener
}

func eventCallBack(ev pq.ListenerEventType, err error) {
	typ := ""
	switch ev {
	case pq.ListenerEventConnected:
		typ = "connected"
	case pq.ListenerEventConnectionAttemptFailed:
		typ = "connection_attempt_failed"
	case pq.ListenerEventDisconnected:
		typ = "disconnected"
	case pq.ListenerEventReconnected:
		typ = "reconnected"
	}
	if err != nil {
		logger.KV(xlog.ERROR,
			"event", typ,
			"error", err.Error())
	} else {
		logger.KV(xlog.DEBUG, "event", typ)
	}
}

func NewListener(p xdb.Provider, minReconnectInterval time.Duration, maxReconnectInterval time.Duration) Listener {
	minReconnectInterval = values.NumbersCoalesce(minReconnectInterval, DefaultMinReconnectInterval)
	maxReconnectInterval = values.NumbersCoalesce(maxReconnectInterval, DefaultMaxReconnectInterval)

	lp := pq.NewListener(p.ConnectionString(), minReconnectInterval, maxReconnectInterval, eventCallBack)

	l := &listener{
		listener: lp,
	}

	return l
}

func (l *listener) Close() error {
	if l.listener != nil {
		err := l.listener.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *listener) Listen(ctx context.Context, topic string, callback func(n *Notification)) error {
	err := l.listener.Listen(topic)
	if err != nil {
		return errors.Wrapf(err, "failed to listen to channel: %s", topic)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.KV(xlog.INFO,
					"reason", "context_done",
					"channel", topic)
				err = l.listener.Unlisten(topic)
				if err != nil {
					logger.KV(xlog.ERROR,
						"reason", "unlisten",
						"channel", topic,
						"error", err.Error())
				}
				return
			case n := <-l.listener.Notify:
				if n != nil {
					callback(parsePayload(n))
				}
			case <-time.After(time.Minute):
				go func() {
					err := l.listener.Ping()
					if err != nil {
						logger.KV(xlog.ERROR,
							"reason", "ping",
							"error", err.Error())
					}
				}()
				// Check if there's more work available, just in case it takes
				// a while for the Listener to notice connection loss and
				// reconnect.
				logger.KV(xlog.DEBUG,
					"reason", "no_events",
					"channel", topic)
			}
		}
	}()
	return nil
}

func parsePayload(in *pq.Notification) *Notification {
	n := &Notification{
		Channel:    in.Channel,
		RawPayload: in.Extra,
	}
	s := in.Extra
	if s != "" && s != "{}" && s != "[]" {
		err := json.Unmarshal([]byte(s), &n.Payload)
		if err != nil {
			logger.KV(xlog.DEBUG,
				"reason", "unmarshal",
				"val", s,
				"err", err.Error())
		}
	}
	return n
}
