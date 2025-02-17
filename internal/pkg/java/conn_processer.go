package java

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/haveachin/infrared/internal/pkg/java/protocol/handshaking"
	"github.com/haveachin/infrared/internal/pkg/java/protocol/login"
	"github.com/pires/go-proxyproto"
)

type InfraredConnProcessor struct {
	ConnProcessor
	mu sync.RWMutex
}

func (cp *InfraredConnProcessor) ClientTimeout() time.Duration {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.ConnProcessor.ClientTimeout
}

type ConnProcessor struct {
	ClientTimeout time.Duration
}

func (cp ConnProcessor) ProcessConn(c net.Conn) (net.Conn, error) {
	pc := ProcessedConn{
		Conn:       *c.(*Conn),
		remoteAddr: c.RemoteAddr(),
	}

	if pc.proxyProtocol {
		header, err := proxyproto.Read(pc.r)
		if err != nil {
			return nil, err
		}
		pc.remoteAddr = header.SourceAddr
	}

	pks, err := pc.ReadPackets(2)
	if err != nil {
		return nil, err
	}
	pc.readPks = pks

	hs, err := handshaking.UnmarshalServerBoundHandshake(pks[0])
	if err != nil {
		return nil, err
	}
	pc.handshake = hs

	pc.serverAddr = hs.ParseServerAddress()
	if strings.Contains(pc.serverAddr, ":") {
		pc.serverAddr, _, err = net.SplitHostPort(pc.serverAddr)
		if err != nil {
			return nil, err
		}
	}

	if pc.realIP {
		addr, _, _, err := hs.ParseRealIP()
		if err != nil {
			return nil, err
		}
		pc.remoteAddr = addr
	}

	if hs.IsStatusRequest() {
		return &pc, nil
	}

	ls, err := login.UnmarshalServerBoundLoginStart(pks[1])
	if err != nil {
		return nil, err
	}
	pc.username = string(ls.Name)

	return &pc, nil
}
