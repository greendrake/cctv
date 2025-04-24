package dvrip

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/greendrake/cctv/dvr/message"
	"github.com/greendrake/cctv/dvr/packet"
	"github.com/greendrake/cctv/util"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	alnum        = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	portTCP      = "34567"
	DialTimout   = time.Second * 5
	ReadTimeout  = 500 * time.Millisecond
	WriteTimeout = time.Second * 5
)

var magicEnd = []byte{0x0A, 0x00}

type statusCode int

const (
	statusOK                                  statusCode = 100
	statusUnknownError                        statusCode = 101
	statusUnsupportedVersion                  statusCode = 102
	statusRequestNotPermitted                 statusCode = 103
	statusUserAlreadyLoggedIn                 statusCode = 104
	statusUserIsNotLoggedIn                   statusCode = 105
	statusUsernameOrPasswordIsIncorrect       statusCode = 106
	statusUserDoesNotHaveNecessaryPermissions statusCode = 107
	statusPasswordIsIncorrect                 statusCode = 203
	statusWrongUser                           statusCode = 205
	statusStartOfUpgrade                      statusCode = 511
	statusUpgradeWasNotStarted                statusCode = 512
	statusUpgradeDataErrors                   statusCode = 513
	statusUpgradeError                        statusCode = 514
	statusUpgradeSuccessful                   statusCode = 515
)

var statusCodes = map[statusCode]string{
	statusOK:                                  "OK",
	statusUnknownError:                        "Unknown error",
	statusUnsupportedVersion:                  "Unsupported version",
	statusRequestNotPermitted:                 "Request not permitted",
	statusUserAlreadyLoggedIn:                 "User already logged in",
	statusUserIsNotLoggedIn:                   "User is not logged in",
	statusUsernameOrPasswordIsIncorrect:       "Username or password is incorrect",
	statusUserDoesNotHaveNecessaryPermissions: "User does not have necessary permissions",
	statusPasswordIsIncorrect:                 "Password is incorrect",
	statusWrongUser:                           "Username is incorrect",
	statusStartOfUpgrade:                      "Start of upgrade",
	statusUpgradeWasNotStarted:                "Upgrade was not started",
	statusUpgradeDataErrors:                   "Upgrade data errors",
	statusUpgradeError:                        "Upgrade error",
	statusUpgradeSuccessful:                   "Upgrade successful",
}

var keyCodes = map[string]string{
	"M": "Menu",
	"I": "Info",
	"E": "Esc",
	"F": "Func",
	"S": "Shift",
	"L": "Left",
	"U": "Up",
	"R": "Right",
	"D": "Down",
}

type Settings struct {
	Address      string
	User         string
	PasswordHash string
}

type Client struct {
	settings          *Settings
	session           int32
	packetSequence    uint32
	c                 net.Conn
	keepAliveInterval uint8
	lastKeepAlivePing int64
	Ctx               context.Context
}

var TimeoutError = errors.New("Network timeout")

func NewClient(ctx context.Context, address string, args ...string) (*Client, error) {
	user := "admin"
	if len(args) > 0 && len(args[0]) > 0 {
		user = args[0]
	}
	password := ""
	if len(args) > 1 && len(args[1]) > 0 {
		password = args[1]
	}
	address = address + ":" + portTCP
	client := &Client{
		settings: &Settings{
			Address:      address,
			User:         user,
			PasswordHash: sofiaHash(password),
		},
		Ctx:               ctx,
		lastKeepAlivePing: time.Now().Unix(),
	}
	var (
		err    error
		dialer net.Dialer
	)
	dialCtx, cancel := context.WithTimeout(ctx, DialTimout)
	defer cancel()
	client.c, err = dialer.DialContext(dialCtx, "tcp", address)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			// Wait DialTimout because the dialer doesn't (it tries to re-connect a few times per second)
			util.SleepCtx(ctx, DialTimout)
		}
		return nil, err
	}
	err = client.Login()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) send(msgID packet.Code, data []byte) error {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, packet.Header{
		HeadFlag:       255,
		Version:        0,
		SessionId:      c.session,
		SequenceNumber: c.packetSequence,
		Code:           msgID,
		DataLength:     uint32(len(data)) + 2,
	}); err != nil {
		return err
	}
	err := binary.Write(&buf, binary.LittleEndian, append(data, magicEnd...))
	if err != nil {
		return err
	}
	c.c.SetWriteDeadline(time.Now().Add(WriteTimeout))
	_, err = c.c.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Getting a message involves receiving one or more packets
func (c *Client) GetMessage() (*message.Message, error) {
	packet, err := c.getPacket()
	if err != nil {
		return nil, err
	}
	data := packet.Data
	if !packet.IsSingle() {
		if packet.IsMedia() {
			for !packet.IsLast() {
				packet, err = c.getPacket()
				if err != nil {
					return nil, err
				}
				if !packet.IsMedia() {
					break
				}
				data = append(data, packet.Data...)
			}
		} else {
			total := packet.GetTotal()
			// Get the rest of the packets
			var i uint8 = 1
			for ; i < total; i++ {
				packet, err = c.getPacket()
				if err != nil {
					return nil, err
				}
				if packet.GetOrdinal() != i {
					return nil, fmt.Errorf("Wrong packet ordinal number. Expected: %d. Got: %d\n", i, packet.GetOrdinal())
				}
				data = append(data, packet.Data...)
			}
		}
	}
	return &message.Message{
		Code: packet.Header.Code,
		Data: data,
	}, nil
}

func (c *Client) getPacket() (*packet.Packet, error) {
	var header packet.Header
	// Read 20 bytes
	var b = make([]byte, 20)
	c.c.SetReadDeadline(time.Now().Add(ReadTimeout))
	_, err := c.c.Read(b)
	if err != nil {
		return nil, err
	}
	// Map those 20 bytes on a new packet header struct
	err = binary.Read(bytes.NewReader(b), binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	// Do validation
	if header.HeadFlag != 0xFF {
		return nil, fmt.Errorf("Unexpected packet HeadFlag byte: %x", header.HeadFlag)
	}
	c.packetSequence += 1
	// Read the data part
	var data = make([]byte, header.DataLength)
	c.c.SetReadDeadline(time.Now().Add(ReadTimeout))
	err = binary.Read(c.c, binary.LittleEndian, &data)
	if err != nil {
		return nil, TimeoutError
	}
	return &packet.Packet{
		Header: header,
		Data:   data,
	}, nil
}

func sofiaHash(password string) string {
	digest := md5.Sum([]byte(password))
	hash := make([]byte, 0, 8)
	for i := 1; i < len(digest); i += 2 {
		sum := int(digest[i-1]) + int(digest[i])
		hash = append(hash, alnum[sum%len(alnum)])
	}
	return string(hash)
}

func (c *Client) Login() error {
	body, err := json.Marshal(map[string]string{
		"EncryptType": "MD5",
		"LoginType":   "DVRIP-WEB",
		"PassWord":    c.settings.PasswordHash,
		"UserName":    c.settings.User,
	})
	if err != nil {
		return err
	}
	err = c.send(packet.LOGIN_REQ, body)
	if err != nil {
		return err
	}
	responsePacket, err := c.getPacket()
	if err != nil {
		return err
	}
	if responsePacket.Header.Code != packet.LOGIN_RSP {
		return fmt.Errorf("Unexepcted response code to login request: %d", responsePacket.Header.Code)
	}
	resp := responsePacket.Data
	// Avoid "invalid character '\x00' after top-level value" error
	if len(resp) > 2 && bytes.Compare(resp[len(resp)-2:], []byte{10, 0}) == 0 {
		resp = resp[:len(resp)-2]
	}
	m := map[string]interface{}{}
	err = json.Unmarshal(resp, &m)
	if err != nil {
		return err
	}
	status, ok := m["Ret"].(float64)
	if !ok {
		return fmt.Errorf("ret is not an int: %v", m["Ret"])
	}
	sc := statusCode(status)
	if sc == statusPasswordIsIncorrect || sc == statusWrongUser {
		return &util.WrongCredentialsError{statusCodes[sc]}
	}
	if (sc != statusOK) && (sc != statusUpgradeSuccessful) {
		return fmt.Errorf("unexpected status code: %v - %v", status, statusCodes[sc])
	}
	session, err := strconv.ParseUint(m["SessionID"].(string), 0, 32)
	if err != nil {
		return err
	}
	c.session = int32(session)
	c.keepAliveInterval = uint8(m["AliveInterval"].(float64))
	return nil
}

func (c *Client) makeCommand(command packet.Code, data interface{}) ([]byte, error) {
	m := map[string]interface{}{
		"Name":      packet.Commands[command],
		"SessionID": fmt.Sprintf("%08X", c.session),
	}
	if data != nil {
		m[packet.Commands[command]] = data
	}
	return json.Marshal(m)
}

func (c *Client) Command(command packet.Code, data interface{}, getMessage bool) error {
	params, err := c.makeCommand(command, data)
	if err != nil {
		return err
	}
	err = c.send(command, params)
	if !getMessage || err != nil {
		return err
	}
	_, err = c.GetMessage()
	return err
}

func (c *Client) SetTime() error {
	return c.Command(packet.SYSMANAGER_REQ, time.Now().UTC().Format("2006-01-02 15:04:05"), true)
}

func (c *Client) Logout() error {
	return c.Command(packet.LOGOUT_REQ, nil, true)
}

func (c *Client) Disconnect() error {
	return c.c.Close()
}

func (c *Client) MaybePingKeepAlive() error {
	var err error
	now := time.Now().Unix()
	if uint8(now-c.lastKeepAlivePing) > c.keepAliveInterval {
		err = c.Command(packet.KEEPALIVE_REQ, nil, false)
		if err == nil {
			c.lastKeepAlivePing = now
		}
	}
	return err
}

func (c *Client) Monitor(stream string) error {
	err := c.Command(packet.MONITOR_CLAIM, map[string]interface{}{
		"Action": "Claim",
		"Parameter": map[string]interface{}{
			"Channel":    0,
			"CombinMode": "NONE",
			"StreamType": stream,
			"TransMode":  "TCP",
		},
	}, true)
	if err != nil {
		return err
	}
	// TODO: check resp
	data, err := json.Marshal(map[string]interface{}{
		"Name":      "OPMonitor",
		"SessionID": fmt.Sprintf("%08X", c.session),
		"OPMonitor": map[string]interface{}{
			"Action": "Start",
			"Parameter": map[string]interface{}{
				"Channel":    0,
				"CombinMode": "NONE",
				"StreamType": stream,
				"TransMode":  "TCP",
			},
		},
	})
	return c.send(1410, data)
}
