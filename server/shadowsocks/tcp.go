package shadowsocks

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pkg/bufferredConn"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pkg/log"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pool"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	// BasicLen is the basic auth length of [salt][encrypted payload length][length tag][encrypted payload][payload tag]
	BasicLen = 32 + 2 + 16
)

func (s *Server) handleTCP(conn net.Conn) error {
	bConn := bufferredConn.NewBufferedConn(conn)
	defer bConn.Close()
	key, _ := s.authTCP(bConn)
	if key == nil {
		// Auth fail. Drain the conn
		log.Warn("Auth fail. Drain the conn from: %v", conn.RemoteAddr().String())
		_, err := io.Copy(io.Discard, conn)
		return err
	}
	crw := NewSSConn(bConn, CiphersConf[key.method], key.masterKey)

	// Read target
	var firstTwoBytes = pool.Get(2)
	_, err := io.ReadFull(crw, firstTwoBytes)
	n, err := BytesSizeForSocksAddr(firstTwoBytes)
	if err != nil {
		return err
	}
	var bytesSocksAddr = pool.Get(n)
	copy(bytesSocksAddr, firstTwoBytes)
	_, err = io.ReadFull(crw, bytesSocksAddr[2:])
	if err != nil {
		return err
	}
	targetSocksAddr, err := NewSocksAddr(bytesSocksAddr)
	if err != nil {
		return err
	}
	target := net.JoinHostPort(targetSocksAddr.Hostname, strconv.Itoa(int(targetSocksAddr.Port)))

	// Dial and relay
	rConn, err := net.Dial("tcp", target)
	if err != nil {
		return err
	}
	if err = relayTCP(crw, rConn); err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil // ignore i/o timeout
		}
		return fmt.Errorf("[tcp] handleConn relay error: %w", err)
	}
	return nil
}

func relayTCP(lConn, rConn net.Conn) (err error) {
	defer rConn.Close()
	eCh := make(chan error, 1)
	go func() {
		_, e := io.Copy(rConn, lConn)
		rConn.SetDeadline(time.Now())
		lConn.SetDeadline(time.Now())
		eCh <- e
	}()
	_, e := io.Copy(lConn, rConn)
	rConn.SetDeadline(time.Now())
	lConn.SetDeadline(time.Now())
	if e != nil {
		return e
	}
	return <-eCh
}

func (s *Server) authTCP(conn bufferredConn.BufferedConn) (key *Key, err error) {
	var buf = pool.Get(BasicLen)
	defer pool.Put(buf)
	data, err := conn.Peek(BasicLen)
	if err != nil {
		return nil, io.ErrUnexpectedEOF
	}
	ctx := s.GetUserContextOrInsert(conn.RemoteAddr().(*net.TCPAddr).IP.String())
	key, _ = ctx.Auth(func(key Key) ([]byte, bool) {
		return s.probeTCP(buf, data, key)
	})
	return key, nil
}

func (s *Server) probeTCP(buf []byte, data []byte, key Key) ([]byte, bool) {
	//[salt][encrypted payload length][length tag][encrypted payload][payload tag]
	conf := CiphersConf[key.method]

	salt := data[:conf.SaltLen]
	cipherText := data[conf.SaltLen : conf.SaltLen+2+conf.TagLen]

	return conf.Verify(buf, key.masterKey, salt, cipherText, nil)
}
