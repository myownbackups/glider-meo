package vmess

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/nadoo/glider/pkg/log"
	"github.com/nadoo/glider/proxy"
)

// VMess struct.
type VMess struct {
	dialer proxy.Dialer
	addr   string

	uuid     string
	aead     bool
	alterID  int
	security string

	client *Client
}

type VMessJson struct {
	V    string `json:"v"`
	Ps   string `json:"ps"`
	Add  string `json:"add"`
	Port string `json:"port"`
	ID   string `json:"id"`
	Aid  string `json:"aid"`
	Net  string `json:"net"`
	Type string `json:"type"`
	Host string `json:"host"`
	Path string `json:"path"`
	TLS  string `json:"tls"`
}

func init() {
	proxy.RegisterDialer("vmess", NewVMessDialer)
}

func addPaddingIfNeeded(base64String string) string {
	// 计算需要添加的填充字符数量
	padding := strings.Repeat("=", (4-len(base64String)%4)%4)
	return base64String + padding
}

// NewVMess returns a vmess proxy.
func NewVMess(s string, d proxy.Dialer) (*VMess, error) {

	ss := s[8:]
	paddedBase64String := addPaddingIfNeeded(ss)
	jsonStr, err := base64.StdEncoding.DecodeString(paddedBase64String)
	if err != nil {
		log.F("base64 decode err: %s", err)
		return nil, err
	}
	var data VMessJson

	err = json.Unmarshal(jsonStr, &data)
	if err != nil {
		log.F("json unmarshal err: %s", err)
	}

	if data.Host != "" {
		s = fmt.Sprintf("vmess://%s@%s:%s", data.ID, data.Host, data.Port)
	} else {
		s = fmt.Sprintf("vmess://%s@%s:%s", data.ID, data.Add, data.Port)
	}

	u, err := url.Parse(s)
	if err != nil {
		log.F("parse url err: %s", err)
		return nil, err
	}

	addr := u.Host
	security := u.User.Username()
	uuid, ok := u.User.Password()
	if !ok {
		// no security type specified, vmess://uuid@server
		uuid = security
		security = ""
	}

	query := u.Query()
	aid := query.Get("alterID")
	if aid == "" {
		aid = "0"
	}

	alterID, err := strconv.ParseUint(aid, 10, 32)
	if err != nil {
		log.F("parse alterID err: %s", err)
		return nil, err
	}

	aead := alterID == 0
	client, err := NewClient(uuid, security, int(alterID), aead)
	if err != nil {
		log.F("create vmess client err: %s", err)
		return nil, err
	}

	p := &VMess{
		dialer:   d,
		addr:     addr,
		uuid:     uuid,
		alterID:  int(alterID),
		security: security,
		client:   client,
	}

	return p, nil
}

// NewVMessDialer returns a vmess proxy dialer.
func NewVMessDialer(s string, dialer proxy.Dialer) (proxy.Dialer, error) {
	return NewVMess(s, dialer)
}

// Addr returns forwarder's address.
func (s *VMess) Addr() string {
	if s.addr == "" {
		return s.dialer.Addr()
	}
	return s.addr
}

// Dial connects to the address addr on the network net via the proxy.
func (s *VMess) Dial(network, addr string) (net.Conn, error) {
	rc, err := s.dialer.Dial("tcp", s.addr)
	if err != nil {
		return nil, err
	}

	return s.client.NewConn(rc, addr, CmdTCP)
}

// DialUDP connects to the given address via the proxy.
func (s *VMess) DialUDP(network, addr string) (net.PacketConn, error) {
	tgtAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.F("[vmess] error in ResolveUDPAddr: %v", err)
		return nil, err
	}

	rc, err := s.dialer.Dial("tcp", s.addr)
	if err != nil {
		return nil, err
	}
	rc, err = s.client.NewConn(rc, addr, CmdUDP)
	if err != nil {
		return nil, err
	}

	return NewPktConn(rc, tgtAddr), err
}

func init() {
	proxy.AddUsage("vmess", `
VMess scheme:
  vmess://[security:]uuid@host:port[?alterID=num]
    if alterID=0 or not set, VMessAEAD will be enabled
  
  Available security for vmess:
    zero, none, aes-128-gcm, chacha20-poly1305
`)
}
