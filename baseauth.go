package gorouter

// BaseAuth checks simple base authorization by login/password

import (
	"bytes"
	"encoding/base64"
	"strings"

	"github.com/valyala/fasthttp"
)

// BaseAuthLogFunction is a function to log the authorization results
type BaseAuthLogFunction func(fastCtx *fasthttp.RequestCtx, user, password string, success bool)

// BaseAuthAccess is a function to check need to check access or not
type BaseAuthAccess func(fastCtx *fasthttp.RequestCtx) bool

func emptyAccess(_ *fasthttp.RequestCtx) bool {
	return true
}

// BaseAuth checks user authorization
type BaseAuth struct {
	use         bool
	message     string
	charset     string
	header      string
	roles       map[string]string
	useLogFunc  bool
	logFunc     BaseAuthLogFunction
	checkAccess BaseAuthAccess
}

func NewBaseAuth() *BaseAuth {
	return &BaseAuth{
		// default values
		use:         false,
		roles:       map[string]string{},
		charset:     "UTF-8",
		message:     "Access to the staging site",
		header:      `Basic realm="Access to the staging site", charset="UTF-8"`,
		checkAccess: emptyAccess,
	}
}

func (h *BaseAuth) SetMessage(message, charset string) *BaseAuth {
	h.message = message
	h.charset = charset
	h.header = `Basic realm="` + h.message + `", charset="` + h.charset + `"`
	return h
}

func (h *BaseAuth) SetCheckAccessFunc(checkAccess BaseAuthAccess) *BaseAuth {
	h.checkAccess = checkAccess
	return h
}

func (h *BaseAuth) SetLogFunc(logFunc BaseAuthLogFunction) *BaseAuth {
	h.useLogFunc = true
	h.logFunc = logFunc
	return h
}

func (h *BaseAuth) Message() string {
	return h.message
}

func (h *BaseAuth) SetUse(u bool) *BaseAuth {
	h.use = u
	return h
}

func (h *BaseAuth) Use() bool {
	return h.use
}

func (h *BaseAuth) SetRoles(roles map[string]string) *BaseAuth {
	h.roles = roles
	return h
}

func (h *BaseAuth) Roles() map[string]string {
	return h.roles
}

func (h *BaseAuth) Check(fastCtx *fasthttp.RequestCtx) (bool, error) {
	if !h.use {
		return true, nil
	}

	user, passwd := h.getUserPassword(fastCtx.Request.Header.Peek(fasthttp.HeaderAuthorization))
	find := user != "" && h.roles[user] == passwd
	if h.useLogFunc {
		h.logFunc(fastCtx, user, passwd, find)
	}

	if !find {
		fastCtx.Response.Header.DisableNormalizing()
		fastCtx.Response.Header.SetStatusCode(fasthttp.StatusUnauthorized)
		fastCtx.Response.Header.Set(fasthttp.HeaderWWWAuthenticate, h.header)
		_, err := fastCtx.Write([]byte{})
		return false, err
	}

	return true, nil
}

func (h *BaseAuth) getUserPassword(auth []byte) (string, string) {
	if len(auth) == 0 {
		return "", ""
	}

	i := bytes.IndexByte(auth, ' ')
	if i == -1 {
		return "", ""
	}

	if !bytes.EqualFold(auth[:i], []byte("basic")) {
		return "", ""
	}

	decoded, err := base64.StdEncoding.DecodeString(string(auth[i+1:]))
	if err != nil {
		return "", ""
	}

	credentials := bytes.Split(decoded, []byte(":"))
	if len(credentials) <= 1 {
		return "", ""
	}

	return strings.ToLower(string(credentials[0])), string(credentials[1])
}
