package app

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
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
	PING    = "ping"
	ECHO    = "echo"
	GET     = "get"
	DEL     = "del"
	INCR    = "incr"
	DECR    = "decr"
	LPUSH   = "lpush"
	RPUSH   = "rpush"
	LRANGE  = "lrange"
	SET     = "set"
	EXISTS  = "exists"
	XX      = "xx"
	NX      = "nx"
	EX      = "ex"
	PX      = "px"
	KEEPTTL = "keepttl"
)

const PONG_RESP = "+PONG\r\n"
const OK_RESP = "+OK\r\n"
const BULK_STRING_RESP = "$%d\r\n%s\r\n"
const INTEGER_RESP = ":%d\r\n"
const NULL_RESP = "_\r\n"
const ARRAY_RESP = "*%d\r\n%s"
const EMPTY_ARRAY_RESP = "*0\r\n"

const SYNTAX_ERROR = "-syntax error\r\n"
const WRONG_ARG_ERROR = "-ERR wrong number of arguments for command\r\n"
const INTEGER_ERROR = "-value is not an integer or out of range\r\n"
const INVALID_KEY_ERROR = "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
const UNKOWN_ERROR = "-ERR: unknown command %s\r\n"

var store map[string]interface{} = nil
var expiration map[string]*time.Timer = nil
var storeMutex = &sync.RWMutex{}

func NewGodis(addr string) {
	server := NewServer(addr)
	store = make(map[string]interface{})
	expiration = make(map[string]*time.Timer)
	server.Run()
}

func ProcessRequest(req string) string {
	cmds := strings.Split(req, ("\r\n"))
	response := fmt.Sprintf(UNKOWN_ERROR, cmds[2])
	switch strings.ToLower(cmds[2]) {
	case PING:
		response = PONG_RESP
	case ECHO:
		response = cmdEcho(cmds[3:])
	case SET:
		response = cmdSet(cmds[3:])
	case GET:
		response = cmdGet(cmds[3:])
	case EXISTS:
		response = cmdExists(cmds[3:])
	case DEL:
		response = cmdDel(cmds[3:])
	case INCR:
		response = cmdIncr(cmds[3:])
	case DECR:
		response = cmdDecr(cmds[3:])
	case LPUSH:
		response = cmdLPush(cmds[3:])
	case RPUSH:
		response = cmdRPush(cmds[3:])
	case LRANGE:
		response = cmdLRange(cmds[3:])
	}
	return response
}

func cmdSet(cmds []string) string {
	if len(cmds) < 4 {
		return WRONG_ARG_ERROR
	}
	key := cmds[1]
	value := cmds[3]
	if slices.Contains(cmds, XX) && slices.Contains(cmds, NX) {
		return SYNTAX_ERROR
	}
	response := OK_RESP
	if len(cmds) == 4 {
		return response
	}
	flags := cmds[3:]
	if !slices.Contains(flags, KEEPTTL) && expiration[key] != nil {
		expiration[key].Stop()
	}
	for i, flag := range flags {
		storeMutex.RLock()
		value := store[key]
		storeMutex.RUnlock()
		switch strings.ToLower(flag) {
		case GET:
			if value == nil {
				return NULL_RESP
			}
			v, _ := value.(string)
			response = fmt.Sprintf(BULK_STRING_RESP, len(v), v)
		case XX:
			if value == nil {
				return NULL_RESP
			}
		case NX:
			if value != nil {
				return NULL_RESP
			}
		case EX:
			if i+2 >= len(flags) {
				return SYNTAX_ERROR
			}
			expirationTime, err := strconv.ParseInt(flags[i+2], 10, 0)
			if err != nil {
				return INTEGER_ERROR
			}
			if expiration[key] != nil {
				expiration[key].Stop()
			}
			timer := time.NewTimer(time.Duration(expirationTime) * time.Second)
			expiration[key] = timer
			go func() {
				<-timer.C
				storeMutex.Lock()
				store[key] = nil
				storeMutex.Unlock()
			}()
		case PX:
			if i+2 >= len(flags) {
				return SYNTAX_ERROR
			}
			expirationTime, err := strconv.ParseInt(flags[i+2], 10, 0)
			if err != nil {
				return INTEGER_ERROR
			}
			if expiration[key] != nil {
				expiration[key].Stop()
			}
			timer := time.NewTimer(time.Duration(expirationTime) * time.Millisecond)
			expiration[key] = timer
			go func() {
				<-timer.C
				storeMutex.Lock()
				store[key] = nil
				storeMutex.Unlock()
			}()
			// case EXAT:
			// case PXAT:
		}
	}
	storeMutex.Lock()
	store[key] = &value
	storeMutex.Unlock()
	return response
}

func cmdExists(cmds []string) string {
	if len(cmds) < 2 {
		return WRONG_ARG_ERROR
	}
	result := 0
	for i := 1; i < len(cmds); i += 2 {
		storeMutex.RLock()
		exists := store[cmds[i]] != nil
		storeMutex.RUnlock()
		if exists {
			result++
		}
	}
	return fmt.Sprintf(INTEGER_RESP, result)
}

func cmdDel(cmds []string) string {
	if len(cmds) < 2 {
		return WRONG_ARG_ERROR
	}
	result := 0
	for i := 1; i < len(cmds); i += 2 {
		storeMutex.RLock()
		exists := store[cmds[i]] != nil
		storeMutex.RUnlock()
		if exists {
			storeMutex.Lock()
			store[cmds[i]] = nil
			storeMutex.Unlock()
		}
	}
	return fmt.Sprintf(INTEGER_RESP, result)
}

func cmdGet(cmds []string) string {
	if len(cmds) < 2 {
		return WRONG_ARG_ERROR
	}
	key := cmds[1]
	storeMutex.RLock()
	value := store[key]
	storeMutex.RUnlock()
	if value != nil {
		v, _ := value.(string)
		return fmt.Sprintf(BULK_STRING_RESP, len(v), v)
	} else {
		return NULL_RESP
	}
}

func cmdIncr(cmds []string) string {
	if len(cmds) > 3 || len(cmds) < 2 {
		return WRONG_ARG_ERROR
	}
	key := cmds[1]
	storeMutex.RLock()
	value := store[key]
	storeMutex.RUnlock()
	if value == nil {
		value = "0"
	}
	v, _ := value.(string)
	x, err := strconv.ParseInt(v, 10, 0)
	if err != nil {
		return INTEGER_ERROR
	}
	x++
	storeMutex.Lock()
	store[key] = fmt.Sprint(x)
	storeMutex.Unlock()
	return fmt.Sprintf(INTEGER_RESP, x)
}

func cmdDecr(cmds []string) string {
	if len(cmds) > 3 || len(cmds) < 2 {
		return WRONG_ARG_ERROR
	}
	key := cmds[1]
	storeMutex.RLock()
	value := store[key]
	storeMutex.RUnlock()
	if value == nil {
		value = "0"
	}
	v, _ := value.(string)
	x, err := strconv.ParseInt(v, 10, 0)
	if err != nil {
		return INTEGER_ERROR
	}
	x--
	storeMutex.Lock()
	store[key] = fmt.Sprint(x)
	storeMutex.Unlock()
	return fmt.Sprintf(INTEGER_RESP, x)
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
	if amount > 0 {
		return fmt.Sprintf(BULK_STRING_RESP, amount, strings.Join(result, " "))
	} else {
		return "-ERR wrong number of arguments for command\r\n"
	}
}

func cmdLPush(cmds []string) string {
	if len(cmds) < 2 && len(cmds) > 4 {
		return WRONG_ARG_ERROR
	}
	key := cmds[1]
	storeMutex.RLock()
	xs := store[key]
	storeMutex.RUnlock()
	var vs []string
	var ok bool
	if xs == nil {
		vs = make([]string, 0)
	} else {
		vs, ok = xs.([]string)
		if !ok {
			return INVALID_KEY_ERROR
		}
	}
	values := cmds[2:]
	for i := 1; i < len(values); i += 2 {
		vs = append([]string{values[i]}, vs...)
	}
	storeMutex.Lock()
	store[key] = vs
	storeMutex.Unlock()
	return fmt.Sprintf(INTEGER_RESP, len(vs))
}

func cmdRPush(cmds []string) string {
	if len(cmds) < 2 && len(cmds) > 4 {
		return WRONG_ARG_ERROR
	}
	key := cmds[1]
	storeMutex.RLock()
	xs := store[key]
	storeMutex.RUnlock()
	var vs []string
	var ok bool
	if xs == nil {
		vs = make([]string, 0)
	} else {
		vs, ok = xs.([]string)
		if !ok {
			return INVALID_KEY_ERROR
		}
	}
	values := cmds[2:]
	for i := 1; i < len(values); i += 2 {
		vs = append(vs, values[i])
	}
	storeMutex.Lock()
	store[key] = vs
	storeMutex.Unlock()
	return fmt.Sprintf(INTEGER_RESP, len(vs))
}

func cmdLRange(cmds []string) string {
	if len(cmds)/2 != 3 {
		return WRONG_ARG_ERROR
	}
	key := cmds[1]
	storeMutex.RLock()
	xs := store[key]
	storeMutex.RUnlock()
	var vs []string
	var ok bool
	if xs == nil {
		vs = make([]string, 0)
	} else {
		vs, ok = xs.([]string)
		if !ok {
			return INVALID_KEY_ERROR
		}
	}
	start, err := strconv.ParseInt(cmds[3], 10, 0)
	if err != nil {
		return INTEGER_ERROR
	}
	stop, err := strconv.ParseInt(cmds[5], 10, 0)
	if err != nil {
		return INTEGER_ERROR
	}
	if start < 0 {
		start = 0
	}
	if int(start) > len(vs) {
		return EMPTY_ARRAY_RESP
	}
	if stop < 0 {
		stop = int64(len(vs)) + stop
	}
	result := ""
	for i := start; i <= stop; i++ {
		result += fmt.Sprintf(BULK_STRING_RESP, len(vs[i]), vs[i])
	}
	length := stop - start + 1
	if length < -1 {
		length = -1
	}
	return fmt.Sprintf(ARRAY_RESP, length, result)
}
