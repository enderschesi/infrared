package infrared

import (
	"net"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/haveachin/infrared/protocol/handshaking"
	"github.com/haveachin/infrared/protocol/login"
	"github.com/pires/go-proxyproto"
)

type ConnProcessor interface {
	ProcessConn()
}

// Connection Processing Node
type CPN struct {
	Log logr.Logger
}

func (cpn *CPN) Start(cpnChan <-chan ProcessingConn, srvChan chan<- ProcessingConn) {
	for {
		c, ok := <-cpnChan
		if !ok {
			break
		}
		cpn.Log.Info("processing",
			"remoteAddress", c.RemoteAddr(),
		)

		if err := cpn.ProcessConn(&c); err != nil {
			cpn.Log.Error(err, "processing",
				"remoteAddress", c.RemoteAddr(),
			)
			c.Close()
			continue
		}
		srvChan <- c
	}
}

func (cpn *CPN) ProcessConn(c *ProcessingConn) error {
	// TODO: Add client timeout config setting
	c.SetReadDeadline(time.Now().Add(time.Second))
	if c.proxyProtocol {
		header, err := proxyproto.Read(c.Reader())
		if err != nil {
			return err
		}
		c.remoteAddr = header.SourceAddr
	}

	pks, err := c.ReadPackets(2)
	if err != nil {
		return err
	}
	c.SetReadDeadline(time.Time{})
	c.readPks = pks

	hs, err := handshaking.UnmarshalServerBoundHandshake(pks[0])
	if err != nil {
		return err
	}
	c.handshake = hs

	c.srvHost = hs.ParseServerAddress()
	if strings.Contains(c.srvHost, ":") {
		c.srvHost, _, err = net.SplitHostPort(hs.ParseServerAddress())
		if err != nil {
			return err
		}
	}

	if c.realIP {
		addr, _, _, err := hs.ParseRealIP()
		if err != nil {
			return err
		}
		c.remoteAddr = addr
	}

	if hs.IsStatusRequest() {
		return nil
	}

	ls, err := login.UnmarshalServerBoundLoginStart(pks[1])
	if err != nil {
		return err
	}
	c.username = string(ls.Name)

	return nil
}
