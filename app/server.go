package app

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"unicode"
)

// Server defines the minimum contract our
// TCP and UDP server implementations must satisfy.
type Server interface {
	Run() error
	Close() error
}

// TCPServer holds the structure of our TCP
// implementation.
type TCPServer struct {
	addr   string
	server net.Listener
}

// NewServer creates a new Server using given protocol
// and addr.
func NewServer(addr string) Server {
	return &TCPServer{
		addr: addr,
	}
}

// Run starts the TCP Server.
func (t *TCPServer) Run() (err error) {
	t.server, err = net.Listen("tcp", t.addr)
	if err != nil {
		return err
	}
	defer t.Close()
	return t.handleConnections()
}

// Close shuts down the TCP Server
func (t *TCPServer) Close() (err error) {
	return t.server.Close()
}

// handleConnections is used to accept connections on
// the TCPServer and handle each of them in separate
// goroutines.
func (t *TCPServer) handleConnections() (err error) {
	for {
		conn, err := t.server.Accept()
		if err != nil || conn == nil {
			return errors.New("could not accept connection")
		}
		go t.handleConnection(conn)
	}
}

const (
	SIMPLE_STRING   = '+'
	SIMPLE_ERROR    = '-'
	INTEGER         = ':'
	BULK_STRING     = '$'
	ARRAY           = '*'
	NULL            = '_'
	BOOLEAN         = '#'
	DOUBLE          = ','
	BIG_NUMBER      = '('
	BULK_ERROR      = '!'
	VERBATIM_STRING = '='
	MAPS            = '%'
	SETS            = '~'
)

// handleConnections deals with the business logic of
// each connection and their requests.
func (t *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	for {
		c, _, err := rw.ReadRune()
		if err != nil {
			return
		}
		rw.WriteString(fmt.Sprintf("TYPE = %c\n", c))
		switch c {
		case BULK_STRING:
			bulkLength := 0
			c, _, _ = rw.ReadRune()
			for unicode.IsNumber(c) {
				if unicode.IsNumber(c) {
					x, _ := strconv.ParseInt(string(c), 10, 0)
					bulkLength = (bulkLength * 10) + int(x)
				}
				c, _, _ = rw.ReadRune()
			}
			rw.WriteString(fmt.Sprintf("Length: %d\n", bulkLength))
			rw.ReadRune()           // \r
			c, _, _ = rw.ReadRune() // \n
			bulkString := ""
			for i := 0; i <= bulkLength && !unicode.IsControl(c); i++ {
				println(string(c))
				bulkString += string(c)
				c, _, _ = rw.ReadRune()
			}
			if bulkString == "ping" {
				rw.WriteString("+PONG\r\n")
			} else {
				rw.WriteString(bulkString)
			}
			rw.ReadRune()
			rw.ReadRune()
		default:
			rw.WriteString("+OK\r\n")
		}
		rw.Flush()
	}
}
