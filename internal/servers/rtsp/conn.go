package rtsp

import (
	"XMedia/internal/conf"
	"XMedia/internal/logger"
	"net"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/google/uuid"
)

type connParent interface {
	logger.Writer
	//findSessionByRSessionUnsafe(rsession *gortsplib.ServerSession) *session
}

type conn struct {
	isTLS       bool
	rtspAddress string
	readTimeout conf.Duration
	// runOnConnect        string
	// runOnConnectRestart bool
	// runOnDisconnect     string
	rconn   *gortsplib.ServerConn
	rserver *gortsplib.Server
	parent  connParent

	uuid    uuid.UUID
	created time.Time
}

// Log implements logger.Writer.
func (c *conn) Log(level logger.Level, format string, args ...interface{}) {
	c.parent.Log(level, "[conn %v] "+format, append([]interface{}{c.rconn.NetConn().RemoteAddr()}, args...)...)
}

func (c *conn) initialize() {
	c.uuid = uuid.New()
	c.created = time.Now()
	c.Log(logger.Info, "opened")
}

func (c *conn) ip() net.IP {
	return c.rconn.NetConn().RemoteAddr().(*net.TCPAddr).IP
}

// onClose is called by rtspServer.
func (c *conn) onClose(err error) {
	c.Log(logger.Info, "closed: %v", err)
	//c.onDisconnectHook()
}

// onRequest is called by rtspServer.
func (c *conn) onRequest(req *base.Request) {
	c.Log(logger.Info, "[c->s] ============\n%v", req)
}

// OnResponse is called by rtspServer.
func (c *conn) OnResponse(res *base.Response) {
	c.Log(logger.Info, "[s->c] ============\n%v", res)
}
