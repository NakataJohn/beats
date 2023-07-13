package gonetty

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/elastic/beats/v7/ax/client"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/pkg/errors"
)

var (
	logger = logp.NewLogger("output_gonetty_cli")

	//ErrNotConnected indicates failure due to client
	// having no valid connection.
	ErrNotConnected = errors.New("not connected")
	//ErrJSONEncodeFailed indicates encoding failures.
	ErrJSONEncodeFailed = errors.New("json encode failed")

	ErrAnything = errors.New("errors want to close conn")


)

type tcpOut struct {
	// connection   *client.SendMsg
	batchPublish bool
	address      *net.TCPAddr
	sslEnable    bool
	sslConfig    *tls.Config

	lineDelimiter []byte
	codec         codec.Codec

	observer outputs.Observer
	index    string
}

func newTcpOut(index string, c *gonettyConfig, observer outputs.Observer, codec codec.Codec) (*tcpOut, error) {
	t := &tcpOut{
		sslEnable:     c.SSLEnable,
		lineDelimiter: []byte(c.lineDelimiter),
		batchPublish:  c.BatchPublish,
		observer:      observer,
		index:         index,
		codec:         codec,
	}

	addr, err := net.ResolveTCPAddr(networkTCP, net.JoinHostPort(c.Host, c.Port))
	if err != nil {
		logger.Error(err)
		return nil, errors.Wrap(err, "resolve tcp addr failed")
	}
	t.address = addr

	if c.SSLEnable {
		var cert tls.Certificate
		cert, err := tls.LoadX509KeyPair(c.SSLCertPath, c.SSLKeyPath)
		if err != nil {
			logger.Error(err)
			return nil, errors.Wrap(err, "load tls cert failed")
		}
		t.sslConfig = &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
	}

	// err = t.newTcpConn()
	// if err != nil {
	// 	logger.Error(err)
	// }

	logp.Info("new gonetty output, address=%v", t.address)
	return t, nil
}

// func (t *tcpOut) newTcpConn() error {
// 	if t.sslEnable {
// 		logger.Warn("TLS", "gonetty output doesn't support TLS testing")
// 		return errors.Errorf("gonetty output doesn't support TLS testing")
// 	} else {
// 		return nil
// 	}
// }

func (t *tcpOut) Close() error {
	logger.Warnf("gonetty output connection %v close.", t.address)
	// t.connection.CloseConn(ErrAnything)
	return nil
}

func (t *tcpOut) Publish(
	ctx context.Context,
	batch publisher.Batch,
) error {

	events := batch.Events()
	rest, err := t.publish(ctx, events)
	if len(rest) == 0 {
		batch.ACK()
	} else {
		batch.RetryEvents(rest)
	}
	return err
}

func (t *tcpOut) publish(ctx context.Context, data []publisher.Event) ([]publisher.Event, error) {
	begin := time.Now()
	if len(data) == 0 {
		return nil, nil
	}
	// if t.connection == nil {
	// 	return data, t.newTcpConn()
	// }

	var failedEvents []publisher.Event
	sendErr := error(nil)
	if t.batchPublish {
		// Publish events in bulk
		logger.Debugf("Publishing events in batch.")
		//TODO batchpublishevent function.
		// sendErr = t.BatchPublishEvent(data)
		// if sendErr != nil {
		// 	return data, sendErr
		// }
	} else {
		logger.Debugf("Publishing events one by one.")
		for index, event := range data {
			sendErr = t.PublishEvent(event)
			if sendErr != nil {
				// return the rest of the data with the error
				failedEvents = data[index:]
				break
			}
		}
	}
	logger.Debugf("PublishEvents: %d metrics have been published over HTTP in %v.", len(data), time.Since(begin))
	if len(failedEvents) > 0 {
		return failedEvents, sendErr
	}
	return nil, nil
}

// PublishEvent publish a single event to output.
func (t *tcpOut) PublishEvent(data publisher.Event) error {
	// if t.connection == nil {
	// 	return ErrNotConnected
	// }
	event := data
	logger.Debugf("Publish event: %s", event)
	datab, err := codec.Codec.Encode(t.codec, t.index, &data.Content)
	if err != nil {
		logger.Error(err)
	}
	sendErr := client.SendMsg(string(datab))
	if sendErr != nil {
		logger.Warnf("Fail to insert a single event: %s", err)
		if sendErr == ErrJSONEncodeFailed {
			// don't retry unencodable values
			return nil
		}
	}

	// if t.connection == nil {
	// 	return ErrNotConnected
	// }
	return nil
}

func (t *tcpOut) String() string {
	return "gonetty(" + t.address.String() + ")"
}
