package guacamole

import (
	"github.com/gorilla/websocket"
	"strconv"
)

func Disconnect(ws *websocket.Conn, code int, reason string) {
	// guacd 无法处理中文字符，所以进行了base64编码。
	//encodeReason := base64.StdEncoding.EncodeToString([]byte(reason))
	err := NewInstruction("error", reason, strconv.Itoa(code))
	_ = ws.WriteMessage(websocket.TextMessage, []byte(err.String()))
	disconnect := NewInstruction("disconnect")
	_ = ws.WriteMessage(websocket.TextMessage, []byte(disconnect.String()))
}
