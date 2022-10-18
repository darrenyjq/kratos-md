package xamqp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"codeup.aliyun.com/qimao/go-contrib/prototype/broker"
	"codeup.aliyun.com/qimao/go-contrib/prototype/log"
	"codeup.aliyun.com/qimao/go-contrib/prototype/utils/async"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

type HandlerFunc func(amqp.Delivery) error

type Config struct {
	URL          string `json:"url" mapstructure:"url"`
	QueueName    string `json:"queue_name" mapstructure:"queue_name"`
	ExchangeName string `json:"exchange_name" mapstructure:"exchange_name"`
	AutoAck      bool   `json:"auto_ack" mapstructure:"auto_ack"`
}

func NewBroker(cfg Config, opts ...Option) broker.Broker {
	if regexp.MustCompile("^aliqp://.*").MatchString(cfg.URL) {
		cfg.URL = parseURL(cfg.URL)
	}
	bk := &amqpBK{
		done:        make(chan bool),
		notifyReady: make(chan struct{}),
	}
	bk.opts = Options{
		URL:          cfg.URL,
		QueueName:    cfg.QueueName,
		ExchangeName: cfg.ExchangeName,
		AutoAck:      cfg.AutoAck,
	}
	for _, o := range opts {
		o(&bk.opts)
	}
	return bk
}

type amqpBK struct {
	opts            Options
	connection      *amqp.Connection
	channel         *amqp.Channel
	done            chan bool
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	notifyReady     chan struct{}
	isReady         bool
	consumerTag     string
	wg              sync.WaitGroup
}

func (bk *amqpBK) Start(ctx context.Context) error {
	return async.RunWithContext(ctx, bk.start)
}

func (bk *amqpBK) start() error {
	err := bk.handleConnect()
	if err != nil {
		return errors.WithStack(err)
	}
	err = bk.consume()
	if err == errDone {
		return nil
	}
	return errors.WithStack(err)
}

func (bk *amqpBK) Stop() error {
	log.Info("AMQP Broker：", bk.opts.QueueName, " stop")
	if !bk.isReady {
		return errAlreadyClosed
	}
	close(bk.done)
	bk.isReady = false
	err := bk.channel.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	err = bk.connection.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

const (
	// When reconnecting to the server after connection failure
	reconnectDelay = 5 * time.Second

	// When setting up the channel after a channel exception
	reInitDelay = 2 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errDone          = errors.New("done")
)

func (bk *amqpBK) handleConnect() error {
	bk.isReady = false
	conn, err := bk.connect()
	if err != nil {
		return err
	}
	err = bk.init(conn)
	if err != nil {
		return err
	}
	go bk.handleReconnect()
	return nil
}

// handleReconnect will wait for a connection error on
// notifyConnClose, and then continuously attempt to reconnect.
func (bk *amqpBK) handleReconnect() {
	for {
		select {
		case <-bk.done:
			return
		case <-bk.notifyConnClose:
			log.Info("AMQP Connection closed. Reconnecting...")
		}

		bk.isReady = false
		log.Info("AMQP Attempting to connect")

		conn, err := bk.connect()

		if err != nil {
			select {
			case <-bk.done:
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		if done := bk.handleReInit(conn); done {
			break
		}
	}
}

// connect will create a new AMQP connection
func (bk *amqpBK) connect() (*amqp.Connection, error) {
	conn, err := amqp.Dial(bk.opts.URL)

	if err != nil {
		return nil, err
	}

	bk.changeConnection(conn)
	log.Info("AMQP Connected!")
	return conn, nil
}

// handleReconnect will wait for a channel error
// and then continuously attempt to re-initialize both channels
func (bk *amqpBK) handleReInit(conn *amqp.Connection) bool {
	for {
		bk.isReady = false

		err := bk.init(conn)

		if err != nil {
			log.Error("AMQP Failed to initialize channel. Retrying...")

			select {
			case <-bk.done:
				return true
			case <-time.After(reInitDelay):
			}
			continue
		}

		select {
		case <-bk.done:
			return true
		case <-bk.notifyConnClose:
			log.Info("AMQP Connection closed. Reconnecting...")
			return false
		case <-bk.notifyChanClose:
			log.Info("AMQP Channel closed. Re-running init...")
		}
	}
}

// init will initialize channel & declare queue
func (bk *amqpBK) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()

	if err != nil {
		return err
	}
	_, err = ch.QueueDeclare(
		bk.opts.QueueName,
		true,  // Durable
		false, // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return err
	}
	err = ch.ExchangeDeclare(
		bk.opts.ExchangeName,
		amqp.ExchangeDirect,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	err = ch.QueueBind(
		bk.opts.QueueName,
		"",
		bk.opts.ExchangeName,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	bk.changeChannel(ch)
	bk.isReady = true
	close(bk.notifyReady)

	return nil
}

// changeConnection takes a new connection to the queue,
// and updates the close listener to reflect this.
func (bk *amqpBK) changeConnection(connection *amqp.Connection) {
	bk.connection = connection
	bk.notifyConnClose = make(chan *amqp.Error, 1)
	bk.connection.NotifyClose(bk.notifyConnClose)
}

// changeChannel takes a new channel to the queue,
// and updates the channel listeners to reflect this.
func (bk *amqpBK) changeChannel(channel *amqp.Channel) {
	bk.channel = channel
	bk.notifyChanClose = make(chan *amqp.Error, 1)
	bk.channel.NotifyClose(bk.notifyChanClose)
}

func (bk *amqpBK) consume() error {
	if !bk.isReady {
		return errNotConnected
	}
	for {
		<-bk.notifyReady
		log.Info("AMQP Start Consume")
		bk.consumerTag = uniqueConsumerTag()
		ch, err := bk.channel.Consume(
			bk.opts.QueueName,
			bk.consumerTag,  // Consumer
			bk.opts.AutoAck, // Auto-Ack
			false,           // Exclusive
			false,           // No-local
			false,           // No-Wait
			nil,             // Args
		)
		if err != nil {
			log.Errorf("AMQP consume err: %+v\n", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func(ch <-chan amqp.Delivery, ctx context.Context) {
			log.Info("AMQP Start Stream")
		LoopEnd:
			for {
				select {
				case d, ok := <-ch:
					if !ok { // closing breaks out of the loop
						break LoopEnd
					}
					bk.wg.Add(1)
					e := bk.opts.Handler(d)
					bk.wg.Done()
					if e == nil && !bk.opts.AutoAck {
						// manual ack
						_ = d.Ack(false)
					} else if e != nil && !bk.opts.AutoAck {
						// message redelivery
						_ = d.Nack(false, true)
					}
				case <-ctx.Done():
					break LoopEnd
				}
			}
			log.Info("AMQP Close Stream")
		}(ch, ctx)
		select {
		case <-bk.done:
			cancel()
			bk.waitGroupWithTimeout(time.Second * 3)
			return errDone
		case <-bk.notifyConnClose:
			log.Info("AMQP Connection closed. Reconnecting...")
		case <-bk.notifyChanClose:
			log.Info("AMQP Channel closed. Re-running init...")
		}
		cancel()
		log.Info("AMQP Close Consume")
		bk.notifyReady = make(chan struct{})
	}
}

func (bk *amqpBK) waitGroupWithTimeout(gracefulTimeout time.Duration) {
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), gracefulTimeout)
	defer timeoutCancel()
	wait := make(chan struct{}, 1)
	go func() {
		bk.wg.Wait()
		wait <- struct{}{}
	}()
	select {
	case <-wait:
	case <-timeoutCtx.Done():
	}
}

var consumerSeq uint64

const consumerTagLengthMax = 0xFF // see writeShortstr

func uniqueConsumerTag() string {
	return commandNameBasedUniqueConsumerTag(os.Args[0])
}

func commandNameBasedUniqueConsumerTag(commandName string) string {
	tagPrefix := "ctag-"
	tagInfix := commandName
	tagSuffix := "-" + strconv.FormatUint(atomic.AddUint64(&consumerSeq, 1), 10)

	if len(tagPrefix)+len(tagInfix)+len(tagSuffix) > consumerTagLengthMax {
		tagInfix = "streadway-amqp"
	}

	return strings.ReplaceAll(tagPrefix+tagInfix+tagSuffix, "/", "-")
}

func parseURL(urlAddr string) string {
	u, err := url.Parse(urlAddr)
	if err != nil {
		panic(err)
	}
	ak := u.User.Username()
	sk, ok := u.User.Password()
	if !ok {
		panic("amqp lost password")
	}
	hostname := u.Hostname()
	port := u.Port()
	path := u.EscapedPath()
	sp := strings.Split(hostname, ".")
	resourceOwnerId := sp[0]
	username := getUserName(ak, resourceOwnerId)
	password := getPassword(sk)
	urlNew := &url.URL{
		Scheme:     "amqp",
		Opaque:     "",
		User:       url.UserPassword(username, password),
		Host:       hostname + ":" + port,
		Path:       path,
		RawPath:    "",
		ForceQuery: false,
		RawQuery:   "",
		Fragment:   "",
	}
	return urlNew.String()
}

const (
	accessFromUser = 0
	colon          = ":"
)

func getUserName(ak string, resourceOwnerId string) string {
	var buffer bytes.Buffer
	buffer.WriteString(strconv.Itoa(accessFromUser))
	buffer.WriteString(colon)
	buffer.WriteString(resourceOwnerId)
	buffer.WriteString(colon)
	buffer.WriteString(ak)
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func getPassword(sk string) string {
	now := time.Now()
	currentMillis := strconv.FormatInt(now.UnixNano()/1000000, 10)
	var buffer bytes.Buffer
	buffer.WriteString(strings.ToUpper(hmacSha1(currentMillis, sk)))
	buffer.WriteString(colon)
	buffer.WriteString(currentMillis)
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func hmacSha1(keyStr string, message string) string {
	key := []byte(keyStr)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
