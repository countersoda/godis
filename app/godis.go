package app

import (
	"fmt"
	"strconv"
	"strings"
)

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

const (
	PING = "ping"
	ECHO = "echo"
	GET  = "get"
	SET  = "set"
)

var store map[string]string = nil

func NewGodis(addr string) {
	server := NewServer(addr)
	store = make(map[string]string)
	server.Run()
}

func ProcessRequest(req string) string {
	cmds := strings.Split(req, ("\r\n"))
	fmt.Printf("%s\n", cmds)
	response := fmt.Sprintf("-ERR: unknown command %s\r\n", cmds[2])
	switch strings.ToLower(cmds[2]) {
	case PING:
		response = "+PONG\r\n"
	case ECHO:
		response = cmdEcho(cmds[3:])
	case SET:
		response = cmdSet(cmds[3:])
	case GET:
		response = cmdGet(cmds[3:])
	}
	return response
}

func cmdSet(cmds []string) string {
	key := cmds[1]
	value := cmds[3]
	store[key] = value
	return "+OK\r\n"
}

func cmdGet(cmds []string) string {
	return "+OK\r\n"
}

func cmdEcho(cmds []string) string {
	result := make([]string, 0)
	amount := 0
	for _, cmd := range cmds {
		if strings.Contains(cmd, "$") {
			stringLength, _ := strconv.ParseInt(strings.ReplaceAll(cmd, "$", ""), 10, 0)
			amount += int(stringLength)
		} else if len(cmd) > 0 {
			result = append(result, cmd)
		}
	}
	amount += len(result) - 1
	println(amount)
	if amount > 0 {
		return fmt.Sprintf("$%d\r\n%s\r\n", amount, strings.Join(result, " "))
	} else {
		return "-ERR wrong number of arguments for command\r\n"
	}
}
