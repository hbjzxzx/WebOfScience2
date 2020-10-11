package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type tunnelRecord struct {
	mux    sync.RWMutex
	id2ins map[string]*tunnelImplement
}

type Tunnel interface {
	io.Writer
	io.Reader
	Stop()
}

type tunnelImplement struct {
	availableConn     []net.Conn
	capacityConn      int
	listeningPort     map[string]bool
	listenGoRoutineWg *sync.WaitGroup
	tunnelID          string
	cxt               context.Context
	cancelFunc        context.CancelFunc
}

func (t *tunnelImplement) Stop() {
	t.cancelFunc()
}

func (t *tunnelImplement) run() {

	select {
	case <-t.cxt.Done():
		//close all the resource
		t.releaseConn()

	}
}

func (t *tunnelImplement) tunnelChannelDown(conn net.Conn) {
	var newAva []net.Conn
	for ind, c := range t.availableConn {
		if c == conn {
			newAva = append(t.availableConn[:ind], t.availableConn[ind+1:]...)
			break
		}
	}
	t.availableConn = newAva
	t.listenGoRoutineWg.Add(1)
	go t.listenOnPortInLimitTime()
}

func (t *tunnelImplement) tunnelChannelAdd() {
	go t.listenOnPortInLimitTime()
}

func (t *tunnelImplement) keepSupConnection() {
	for {
		select {
		case <-t.cxt.Done():
			return
		case <-time.Tick(time.Second * 10):
			if len(t.availableConn)+len(t.listeningPort) < t.capacityConn {
				t.listenGoRoutineWg.Add(1)
				go t.listenOnPortInLimitTime()
			}
		}
	}
}

func (t *tunnelImplement) listenOnPortInLimitTime() {
	defer t.listenGoRoutineWg.Done()
	if len(t.availableConn)+len(t.listeningPort) >= t.capacityConn {
		return
	}
	listen, err := net.Listen("tcp4", ":0")
	if err != nil {
		log.Printf("can not open listen port %v", err)
		return
	}
	defer listen.Close()
	portStr := fmt.Sprintf("%d", (listen.Addr().(*net.TCPAddr).Port))
	log.Printf("Tunnel: %s, listening on port: %s ", t.tunnelID, portStr)
	t.listeningPort[portStr] = true

	ch := make(chan net.Conn, 1)
	defer close(ch)
	go func() {
		c, e := listen.Accept()
		if e != nil {
			return
		}
		ch <- c
	}()

	select {
	case conn := <-ch:
		delete(t.listeningPort, portStr)
		t.availableConn = append(t.availableConn, conn)
		log.Printf("Tunnel: %s, listening on port: %s success", t.tunnelID, portStr)
	case <-time.After(10 * time.Second):
		log.Printf("Tunnel: %s, listening on port: %s timeout", t.tunnelID, portStr)
		return
	}
}

func (t *tunnelImplement) releaseConn() {
	for _, c := range t.availableConn {
		c.Close()
	}
}

func newTunnelImplement(tunnelID string, defaultLen int) *tunnelImplement {
	cxt, cxtfunc := context.WithCancel(context.Background())
	t := &tunnelImplement{
		availableConn:     make([]net.Conn, defaultLen, 2*defaultLen),
		capacityConn:      defaultLen,
		listeningPort:     make(map[string]bool, defaultLen),
		listenGoRoutineWg: &sync.WaitGroup{},
		tunnelID:          tunnelID,
		cxt:               cxt,
		cancelFunc:        cxtfunc,
	}
	for i := 0; i < defaultLen; i++ {
		go t.listenOnPortInLimitTime()
	}
	return t
}
