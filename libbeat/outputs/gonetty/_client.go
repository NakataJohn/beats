package gonetty

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"

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

	// // setup client pipeline initializer.
	// clientInitializer = func(channel netty.Channel) {
	// 	channel.Pipeline().
	// 		AddLast(frame.DelimiterCodec(102400, "$$", true)).
	// 		AddLast(format.TextCodec()).
	// 		AddLast(nettylogHandler{})
	// }
)

type tcpOut struct {
	connection net.Conn
	bw         *bufio.Writer
	buf        net.Buffers

	address      *net.TCPAddr
	writevEnable bool
	sslEnable    bool
	sslConfig    *tls.Config

	lineDelimiter []byte
	codec         codec.Codec

	observer outputs.Observer
	index    string
}

func newTcpOut(index string, c *gonettyConfig, observer outputs.Observer, codec codec.Codec) (*tcpOut, error) {
	t := &tcpOut{
		writevEnable:  c.WritevEnable,
		sslEnable:     c.SSLEnable,
		lineDelimiter: []byte(c.lineDelimiter),
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

	err = t.newTcpConn()
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	if t.writevEnable {
		t.buf = make([][]byte, c.BufferSize)
	} else {
		t.bw = bufio.NewWriterSize(t.connection, c.BufferSize)
	}

	logp.Info("new gonetty output, address=%v", t.address)
	return t, nil
}

func (t *tcpOut) newTcpConn() (err error) {
	if t.sslEnable {
		t.connection, err = tls.Dial(networkTCP, t.address.String(), t.sslConfig)
	} else {
		t.connection, err = net.DialTCP(networkTCP, nil, t.address)
	}
	return err
}

func (t *tcpOut) closeTcpConn() {
	_ = t.connection.Close()
	t.connection = nil
}

func (t *tcpOut) Close() error {
	logger.Infof("gonetty output connection %v close.", t.address)
	return t.connection.Close()
}

func (t *tcpOut) Publish(
	ctx context.Context,
	batch publisher.Batch,
) error {
	if t.connection == nil {
		err := t.newTcpConn()
		if err != nil {
			return err
		}
	}
	if t.writevEnable {
		return t.publishWritev(batch)
	}
	return t.publish(batch)
}

func (t *tcpOut) publish(batch publisher.Batch) error {
	events := batch.Events()
	t.observer.NewBatch(len(events))

	bulkSize := 0
	dropped := 0
	for i := range events {
		serializedEvent, err := t.codec.Encode(t.index, &events[i].Content)
		if err != nil {
			dropped++
			continue
		}
		fmt.Printf("事件：%v", serializedEvent)
		_, err = t.bw.Write(serializedEvent)
		if err != nil {
			t.observer.WriteError(err)
			dropped++
			continue
		}
		_, err = t.bw.Write([]byte("$$"))
		if err != nil {
			t.observer.WriteError(err)
			dropped++
			continue
		}
		_, err = t.bw.Write(t.lineDelimiter)
		if err != nil {
			t.observer.WriteError(err)
			dropped++
			continue
		}
		bulkSize += len(serializedEvent) + 1
	}
	err := t.bw.Flush()
	if err != nil {
		t.observer.WriteError(err)
		dropped = len(events)
		t.closeTcpConn()
	}

	t.observer.WriteBytes(bulkSize)
	t.observer.Dropped(dropped)
	t.observer.Acked(len(events) - dropped)
	batch.ACK()
	return nil
}

func (t *tcpOut) publishWritev(batch publisher.Batch) error {
	events := batch.Events()
	t.observer.NewBatch(len(events))

	dropped := 0
	t.buf = t.buf[:0]
	for i := range events {
		serializedEvent, err := t.codec.Encode(t.index, &events[i].Content)
		if err != nil {
			dropped++
			continue
		}
		t.buf = append(append(t.buf, append(serializedEvent, t.lineDelimiter...)), []byte("$$"))
	}

	n, err := t.buf.WriteTo(t.connection)
	if err != nil {
		t.observer.WriteError(err)
		dropped = len(events)
		t.closeTcpConn()
	}

	t.observer.WriteBytes(int(n))
	t.observer.Dropped(dropped)
	t.observer.Acked(len(events) - dropped)
	batch.ACK()
	return nil
}

func (t *tcpOut) String() string {
	return "tcp(" + t.address.String() + ")"
}
