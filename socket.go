package response

import (
	"github.com/gorilla/websocket"
	"github.com/suboat/go-response/log"

	//"net/http"
	"encoding/json"
	"sync"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	WsWriteWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	WsPongWait = 20 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	WsPingPeriod = (WsPongWait * 9) / 10

	// Maximum message size allowed from peer.
	WsMaxMessageSize = 1024 * 2
)

// message push type
const (
	MessageWsTypePing   = iota // 0: nothing
	MessageWsTypeLogin         // 1: login
	MessageWsTypeLogout        // 2: logout
	MessageWsTypeStatus        // 3: update detail
	MessageWsTypeMsg           // 4: have new message
)

var (
	// upgrader
	WsUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	// hub set
	//HubWsSet = HubWs{
	//	Broadcast:  make(chan []byte),
	//	Register:   make(chan *ConnWs),
	//	Unregister: make(chan *ConnWs),
	//	ConnWss:    make(map[orm.Uid][]*ConnWs),
	//	//ConnWss: make(map[*ConnWs]bool),
	//}
)

// hub maintains the set of active ConnWss and broadcasts messages to the
// ConnWss.
type HubWs struct {
	lock *sync.RWMutex

	// Registered ConnWss.
	ConnWss map[string]map[*ConnWs]bool
	//ConnWss map[orm.Uid][]*ConnWs
	//ConnWss map[*ConnWs]bool

	// Inbound messages from the ConnWss.
	Broadcast chan []byte

	// Register requests from the ConnWss.
	Register chan *ConnWs

	// Unregister requests from ConnWss.
	Unregister chan *ConnWs
}

// ConnWs is an middleman between the websocket ConnWs and the hub.
type ConnWs struct {
	// The websocket ConnWs.
	Ws *websocket.Conn

	// uid
	Uid string

	// Buffered channel of outbound messages.
	Send     chan []byte
	SendText chan string

	// Handler
	Handler LogicHandler

	// hub
	Hub *HubWs
}

// MessageWs is a general type of push message by websocket
type MessageWs struct {
	Category int    `json:"category"`
	Content  string `json:"content"`
}

//
type MessageWsPack struct {
	TargetUid   *string      `json:"-"` // 推送到用户
	TargetOther *string      `json:"-"` // 其它规则的推送
	MessageLis  []*MessageWs `json:"messageLis"`
}

// change conn uid
func (c *ConnWs) UidUpdate(uid string) (err error) {
	if c.Hub == nil {
		err = ErrSocketConnHubEmpty
		return
	}
	if c.Uid != uid {
		if err = c.Hub.EnsureUser(uid); err != nil {
			return
		}
		c.Hub.ConnWss[uid][c] = true
		log.Debug("old:", len(c.Hub.ConnWss[c.Uid]), " new:", len(c.Hub.ConnWss[uid]))
		delete(c.Hub.ConnWss[c.Uid], c)
		c.Uid = uid
	}
	return
}

// make user map in hub
func (h *HubWs) EnsureUser(uid string) (err error) {
	// TODO: lock whith register
	h.lock.Lock()
	defer h.lock.Unlock()
	if _, ok := h.ConnWss[uid]; ok == false {
		h.ConnWss[uid] = make(map[*ConnWs]bool)
	}
	return
}

// broadcasts to users
func (h *HubWs) BroadcastTo(uid *string, data []byte) (err error) {
	if uid == nil {
		// broadcasts to all
		h.Broadcast <- data
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()
	for conn, _ := range h.ConnWss[*uid] {
		select {
		case conn.Send <- data:
			break
		default:
			go func() {
				conn.Send <- data
			}()
		}
	}
	return
}

// broadcasts json
func (h *HubWs) BroadcastJson(uid *string, inf interface{}) (err error) {
	var data = []byte{}
	if data, err = json.Marshal(inf); err != nil {
		return
	}
	return h.BroadcastTo(uid, data)
}

// Hub run
func (h *HubWs) Run() {
	// TODO: lock
	for {
		select {
		case c := <-h.Register:
			if err := h.EnsureUser(c.Uid); err != nil {
				log.Error("hub EnsuerUser: ", err.Error())
				return
			}
			if _, ok := h.ConnWss[c.Uid][c]; ok == true {
				log.Error("unknown: ConnWs dunplicate", c)
			}
			h.ConnWss[c.Uid][c] = true
			log.Debug("newone: ", c.Uid)
		case c := <-h.Unregister:
			c.Hub.lock.Lock()
			delete(h.ConnWss[c.Uid], c)
			close(c.Send)
			close(c.SendText)
			c.Hub.lock.Unlock()
		case m := <-h.Broadcast:
			// 广播给所有连接
			h.lock.Lock()
			for _, uConns := range h.ConnWss {
				for c, _ := range uConns {
					select {
					case c.Send <- m:
					default:
						// 连接的send有数据, 跳过，不阻塞 TODO: 稍后再发
						//close(c.Send)
						//delete(h.ConnWss, uid)
					}
				}
			}
			h.lock.Unlock()

			//for _, cLis := range h.ConnWss {
			//	for _, c := range cLis {
			//		select {
			//		case c.Send <- m:
			//		default:
			//			close(c.Send)
			//			//delete(h.ConnWss, uid)
			//		}
			//	}
			//}
		}
	}
}

// readPump pumps messages from the websocket ConnWs to the hub.
func (c *ConnWs) ReadPump() {
	defer func() {
		//HubWsSet.Unregister <- c
		c.Hub.Unregister <- c
		c.Ws.Close()
	}()

	c.Ws.SetReadLimit(WsMaxMessageSize)
	c.Ws.SetReadDeadline(time.Now().Add(WsPongWait))
	c.Ws.SetPongHandler(func(string) error { c.Ws.SetReadDeadline(time.Now().Add(WsPongWait)); return nil })

	for {
		msgType, message, err := c.Ws.ReadMessage()
		if err != nil {
			println("read ws error", err.Error())
			break
		}
		//println("recive text:", c.Uid, string(message))
		que, err2 := SerializeHttpWs(c, msgType, message)
		// serial error
		if err2 != nil {
			res := new(Response)
			res.Error = err2
			CreateResponseWs(c, res)
			continue
		}
		if c.Handler == nil {
			continue
		}
		// handler
		res := c.Handler(que)
		// change uid
		if len(res.Uid) > 0 {
			err3 := c.UidUpdate(res.Uid)
			if res.Error == nil {
				res.Error = err3
			}
		}
		CreateResponseWs(c, res)

		// message push
		if res.MessageWsPack != nil {
			if err = c.BroadcastJson(res.MessageWsPack.TargetUid, res.MessageWsPack); err != nil {
				log.Error("c.BroadcastJson error: ", err)
			}
		}

		//c.SendText <- string(message)
		//hubSet.broadcast <- message
		//c.Send <- message
	}
}

// write writes a message with the given message type and payload.
func (c *ConnWs) Write(mt int, payload []byte) error {
	c.Ws.SetWriteDeadline(time.Now().Add(WsWriteWait))
	return c.Ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket ConnWs.
func (c *ConnWs) WritePump() {
	ticker := time.NewTicker(WsPingPeriod)
	defer func() {
		ticker.Stop()
		c.Ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			println("send bytea to", ok, c.Uid)
			if !ok {
				c.Write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Write(websocket.TextMessage, message); err != nil {
				return
			}
		case message, ok := <-c.SendText:
			//println("send text to", ok, c.Uid.String(), message)
			// text msg
			if !ok {
				c.Write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Write(websocket.TextMessage, []byte(message)); err != nil {
				return
			}
		case <-ticker.C:
			//println("ticker time, ping again", c.Uid.String())
			if err := c.Write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// broadcasts to users
func (c *ConnWs) Broadcast(uid *string, data []byte) (err error) {
	return c.Hub.BroadcastTo(uid, data)
	//if uid == nil {
	//	// broadcasts to all
	//	c.Hub.Broadcast <- data
	//	return
	//}

	//c.Hub.lock.Lock()
	//defer c.Hub.lock.Unlock()
	//for conn, _ := range c.Hub.ConnWss[*uid] {
	//	select {
	//	case conn.Send <- data:
	//		break
	//	default:
	//		go func() {
	//			conn.Send <- data
	//		}()
	//	}
	//}
	return
}

// broadcasts json
func (c *ConnWs) BroadcastJson(uid *string, inf interface{}) (err error) {
	return c.Hub.BroadcastJson(uid, inf)
	//var data = []byte{}
	//if data, err = json.Marshal(inf); err != nil {
	//	return
	//}
	//return c.Broadcast(uid, data)
}

func NewHubWs(h *HubWs) (n *HubWs) {
	// placehold
	if h != nil {
		n = h
		return
	}

	n = &HubWs{
		lock:       new(sync.RWMutex),
		Broadcast:  make(chan []byte),
		Register:   make(chan *ConnWs),
		Unregister: make(chan *ConnWs),
		ConnWss:    make(map[string]map[*ConnWs]bool),
		//ConnWss: make(map[*ConnWs]bool),
	}
	return
}

func NewMessageWsPack(d *MessageWsPack) (n *MessageWsPack) {
	// placehold
	if d != nil {
		n = d
		return
	}

	n = new(MessageWsPack)
	n.MessageLis = []*MessageWs{}
	return
}
