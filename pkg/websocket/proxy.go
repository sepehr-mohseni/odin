package websocket

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Config struct {
	ReadBufferSize    int           `yaml:"readBufferSize"`
	WriteBufferSize   int           `yaml:"writeBufferSize"`
	HandshakeTimeout  time.Duration `yaml:"handshakeTimeout"`
	ReadTimeout       time.Duration `yaml:"readTimeout"`
	WriteTimeout      time.Duration `yaml:"writeTimeout"`
	PingPeriod        time.Duration `yaml:"pingPeriod"`
	MaxMessageSize    int64         `yaml:"maxMessageSize"`
	EnableCompression bool          `yaml:"enableCompression"`
}

type Proxy struct {
	config   Config
	logger   *logrus.Logger
	upgrader websocket.Upgrader
}

type Connection struct {
	clientConn *websocket.Conn
	serverConn *websocket.Conn
	proxy      *Proxy
	target     string
	done       chan struct{}
	once       sync.Once
}

func NewProxy(config Config, logger *logrus.Logger) *Proxy {
	if config.ReadBufferSize == 0 {
		config.ReadBufferSize = 4096
	}
	if config.WriteBufferSize == 0 {
		config.WriteBufferSize = 4096
	}
	if config.HandshakeTimeout == 0 {
		config.HandshakeTimeout = 10 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 60 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}
	if config.PingPeriod == 0 {
		config.PingPeriod = 54 * time.Second
	}
	if config.MaxMessageSize == 0 {
		config.MaxMessageSize = 512 * 1024
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:   config.ReadBufferSize,
		WriteBufferSize:  config.WriteBufferSize,
		HandshakeTimeout: config.HandshakeTimeout,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: config.EnableCompression,
	}

	return &Proxy{
		config:   config,
		logger:   logger,
		upgrader: upgrader,
	}
}

func (p *Proxy) ProxyWebSocket(c echo.Context, targetURL string) error {
	clientConn, err := p.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		p.logger.WithError(err).Error("Failed to upgrade client connection")
		return err
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		clientConn.Close()
		return err
	}

	wsScheme := "ws"
	if target.Scheme == "https" {
		wsScheme = "wss"
	}

	wsURL := fmt.Sprintf("%s://%s%s", wsScheme, target.Host, target.Path)
	if target.RawQuery != "" {
		wsURL += "?" + target.RawQuery
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: p.config.HandshakeTimeout,
	}

	serverConn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		p.logger.WithError(err).WithField("target", wsURL).Error("Failed to connect to target WebSocket")
		clientConn.Close()
		return err
	}

	conn := &Connection{
		clientConn: clientConn,
		serverConn: serverConn,
		proxy:      p,
		target:     wsURL,
		done:       make(chan struct{}),
	}

	go conn.proxyClientToServer()
	go conn.proxyServerToClient()

	<-conn.done

	return nil
}

func (c *Connection) proxyClientToServer() {
	defer c.close()

	c.clientConn.SetReadDeadline(time.Now().Add(c.proxy.config.ReadTimeout))
	c.clientConn.SetPongHandler(func(string) error {
		c.clientConn.SetReadDeadline(time.Now().Add(c.proxy.config.ReadTimeout))
		return nil
	})

	for {
		messageType, data, err := c.clientConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.proxy.logger.WithError(err).Error("Unexpected WebSocket close error from client")
			}
			break
		}

		if int64(len(data)) > c.proxy.config.MaxMessageSize {
			c.proxy.logger.Error("Message size exceeds maximum allowed size")
			break
		}

		c.serverConn.SetWriteDeadline(time.Now().Add(c.proxy.config.WriteTimeout))
		if err := c.serverConn.WriteMessage(messageType, data); err != nil {
			c.proxy.logger.WithError(err).Error("Failed to write message to server")
			break
		}
	}
}

func (c *Connection) proxyServerToClient() {
	defer c.close()

	c.serverConn.SetReadDeadline(time.Now().Add(c.proxy.config.ReadTimeout))
	c.serverConn.SetPongHandler(func(string) error {
		c.serverConn.SetReadDeadline(time.Now().Add(c.proxy.config.ReadTimeout))
		return nil
	})

	for {
		messageType, data, err := c.serverConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.proxy.logger.WithError(err).Error("Unexpected WebSocket close error from server")
			}
			break
		}

		if int64(len(data)) > c.proxy.config.MaxMessageSize {
			c.proxy.logger.Error("Message size exceeds maximum allowed size")
			break
		}

		c.clientConn.SetWriteDeadline(time.Now().Add(c.proxy.config.WriteTimeout))
		if err := c.clientConn.WriteMessage(messageType, data); err != nil {
			c.proxy.logger.WithError(err).Error("Failed to write message to client")
			break
		}
	}
}

func (c *Connection) close() {
	c.once.Do(func() {
		c.clientConn.Close()
		c.serverConn.Close()
		close(c.done)
	})
}

func (p *Proxy) HandleWebSocketUpgrade(c echo.Context) error {
	targetURL := c.Get("target_url").(string)
	return p.ProxyWebSocket(c, targetURL)
}

func WebSocketMiddleware(proxy *Proxy) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if isWebSocketUpgrade(c.Request()) {
				targetURL := c.Get("target_url")
				if targetURL == nil {
					return echo.NewHTTPError(http.StatusBadRequest, "No target URL specified for WebSocket")
				}
				return proxy.ProxyWebSocket(c, targetURL.(string))
			}
			return next(c)
		}
	}
}

func isWebSocketUpgrade(r *http.Request) bool {
	return r.Header.Get("Connection") == "Upgrade" && r.Header.Get("Upgrade") == "websocket"
}
