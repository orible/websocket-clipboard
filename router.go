package main

import (
	"container/ring"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

const (
	pongWait                = 60 * time.Second
	SOCKET_WRITE_WAIT       = 10 * time.Second
	SOCKET_PING_PERIOD      = (pongWait * 9) / 10
	SOCKET_MAX_MESSAGE_SIZE = 2048

	PROTOCOL_BAD          = 0
	PROTOCOL_OK           = 1
	CLIENT_CONNECT        = 1
	CLIENT_DISCONNECT     = 2
	CLIENT_LIST           = 3
	CLIENT_PAIR_ROLL      = 4
	CLIENT_PAIR_CONNECT   = 5
	CLIENT_PUSH_CLIPBOARD = 6
	FLAG_MULTIPLEX_BUFFER = 0x1

	SERVER_BUFFER_PART = 12

	SERVER_CLIPBOARD_PUSH = 10
	SERVER_CONNECT_OK     = 11
	SERVER_CLOSING        = 12
	SERVER_PAIR_CLOSE     = 20

	SERVER_RESPONSE_OK      = 13
	SERVER_RESPONSE_BAD     = 14
	RESPONSE_BAD_TRANSPORT  = -1
	RESPONSE_ARG_BAD_FORMAT = -2
	RESPONSE_ARG_INVALID    = -3
)

type SNetworkPacketOut struct {
	Header int
}

type SThreadClientMessage struct {
	Sender *Client
	Packet SNetworkPacketJson
	debug  bool
}
type SPairRoom struct {
	eventHistory     []string
	clipboardHistory ring.Ring
	id               int
	//clients          map[*Client]int
	creator *Client
	child   *Client
}

type SPairTable struct {
	ptr          *Client
	uuid         int
	starttime    int64
	handCallback int
}

type Client struct {
	conn          *websocket.Conn
	router        *SocketRouter
	rollPairUUID  int
	bucketTime    time.Time
	send          chan *SThreadMessage
	Auth          int
	cookieUUID    int
	dead          bool
	responseTable []int
	pairTable     map[int]*SPairTable
	//req          *http.Request //request object so we can read the cookie store, replace this when using the filesystem store
	timeLost int64
}

func deleteClient(s *Client) {
	for i, v := range s.pairTable {
		/* close pair requests */
		if !s.dead {
			SendPacket(s, CreatePacketResponseEx(SERVER_RESPONSE_BAD, v.handCallback, nil))
		}
		delete(s.pairTable, i)
	}
}

func IsOk(c *Client) bool {
	return c != nil && c.dead != true
}
func (this *Client) getResponse(uuid int) int {
	//for i, v := range this.responseTable {
	//	if v == uuid {
	//		return v
	//	}
	//}
	return -1
}

type SocketRouter struct {
	UUIDCounter int

	clients map[*Client]bool // Registered clients.

	lostClients []*Client
	// Inbound messages from the clients.
	broadcast chan *SThreadMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	pairTable map[int]SPairRoom
}

func NewRouter() *SocketRouter {
	h := &SocketRouter{
		broadcast:  make(chan *SThreadMessage, 0xFFFF),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),

		pairTable: make(map[int]SPairRoom),
		//lostClients: make(map[*Client]bool),
		UUIDCounter: 1,
	}
	return h
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func (h *SocketRouter) run() int {
	ticker := time.NewTicker(1000 * time.Millisecond)
	runRouter := true
	for runRouter {
		if !runRouter {
			break
		}
		select {
		case client := <-h.register:
			h.clients[client] = true
			fmt.Printf("[Router] -> client connected\n")
			break
		case client := <-h.unregister:
			fmt.Printf("[Router] -> client disconnected\n")
			delete(h.clients, client)
			close(client.send)

			client.timeLost = GetTimeUnixMilliseconds()
			//store in lost clients, with timeout

			fmt.Printf("[Router] -> client sleeping\n")
			h.lostClients = append(h.lostClients, client)

			//delete client, eg, when socket lost
			break
		case message := <-h.broadcast:
			switch message.Type {
			case 1001:
				r := message.Ptr.(*SReadResponse)
				switch r.Type {
				case 0:
					/* Experimental idea, if the client exists in the old table, return it to the caller */
					v := r.Ptr.(int)
					var c *Client
					var i int
					for ix, x := range h.lostClients {
						if x.cookieUUID == v && (x.timeLost+(1000*60*10) > GetTimeUnixMilliseconds()) {
							c = x
							i = ix
							break
						}
					}
					if c != nil {
						r.ChanResponse <- &SThreadMessage{
							Type: 1,
							Ptr:  c,
						}
						h.lostClients = append(h.lostClients[:i], h.lostClients[i+1:]...)
					} else {
						r.ChanResponse <- &SThreadMessage{
							Type: 1,
							Ptr:  false,
						}
					}
					break
				case 1:
					break
				}
				break
			case 0:
				r := message.Ptr.(SThreadClientMessage)
				if r.Sender == nil || (r.Packet.Type > CLIENT_CONNECT && r.Sender.Auth == 0) {
					continue /* drop */
				}
				switch r.Packet.Type {
				//case CLIENT_OPEN_MULTIPLEX_BUFFER:
				/* TODO: Open multiplexed buffer stream, referenced by callback ID */
				/* Remarks: Actually, no this is a bad idea */
				/* Do a typical packet protocol */
				//	break
				case CLIENT_CONNECT:
					fmt.Printf("CLIENT_CONNECT -> \n")
					r.Sender.Auth = 1
					SendPacket(r.Sender,
						CreatePacket(SERVER_CONNECT_OK,
							&struct {
								Uptime  int
								Users   int
								Version string
							}{
								Uptime:  0,
								Users:   0,
								Version: "0.0.0.0",
							}))
					break
				case CLIENT_DISCONNECT:
					fmt.Printf("CLIENT_DISCONNECT -> \n")
					r.Sender.Auth = -1
					break
				case CLIENT_LIST:
					fmt.Printf("CLIENT_LIST -> \n")
					break
				case CLIENT_PAIR_CONNECT:
					fmt.Printf("CLIENT_PAIR_CONNECT -> \n")
					var data map[string]interface{}
					var f float64
					var ok bool
					if data, ok = r.Packet.Transport.(map[string]interface{}); !ok {
						break
					}
					if f, ok = data["Key"].(float64); !ok {
						break
					}
					key := int(f)

					var pair *SPairTable
					var creator *Client
					ok = false
					for i, v := range h.clients {
						if i != r.Sender && v {
							pair, ok = i.pairTable[key]
							if ok {
								creator = i
								break
							}
						}
					}
					if !ok {
						fmt.Printf("[ROUTER] pair key does not exist\n")
						SendPacketResponseFault(r.Sender, &r.Packet, RESPONSE_ARG_INVALID)
						break
					}
					delete(creator.pairTable, key)
					//pair.uuid = r.Sender.cookieUUID
					//pair.ptr = r.Sender
					//r.Sender.pairTable[key] = &SPairTable{
					//	uuid: creator.cookieUUID,
					//	ptr:  creator,
					//}

					if creator == r.Sender {
						panic(0)
						//lol
					}
					room := SPairRoom{}
					room.creator = creator
					room.child = r.Sender
					h.pairTable[key] = room

					SendPacket(r.Sender,
						CreatePacketResponse(&r.Packet,
							struct {
								Response   int
								ClientId   int
								ClientType string
								Key        int
							}{
								Response:   1,
								ClientId:   creator.cookieUUID,
								ClientType: "windows",
								Key:        key,
							},
						))
					SendPacket(creator,
						CreatePacketResponseEx(SERVER_RESPONSE_OK, pair.handCallback,
							struct {
								Response   int
								ClientId   int
								ClientType string
								Key        int
							}{
								Response:   1,
								ClientId:   r.Sender.cookieUUID,
								ClientType: "windows",
								Key:        key,
							},
						))
					fmt.Printf("Router -> client connected\n")
					break
				case CLIENT_PAIR_ROLL:
					fmt.Printf("CLIENT_PAIR_ROLL -> \n")
					val := random(100000, 999999)
					//roll new pair
					r.Sender.pairTable[val] = &SPairTable{
						uuid:         -1,
						starttime:    GetTimeUnixMilliseconds(),
						handCallback: r.Packet.Callback,
					}
					SendPacket(r.Sender, CreatePacketResponse(&r.Packet, struct {
						Response int
						Key      int
						Timeout  int
					}{
						Response: 2,
						Key:      val,
						Timeout:  1000 * 60,
					}))
					break
				case CLIENT_PUSH_CLIPBOARD:
					var data map[string]interface{}
					var item SNetworkClipboardItem
					var ok bool
					var f float64
					if data, ok = r.Packet.Transport.(map[string]interface{}); !ok {
						SendPacketResponseFault(r.Sender, &r.Packet, RESPONSE_BAD_TRANSPORT)
						break
					}
					if f, ok = data["Type"].(float64); !ok {
						SendPacketResponseFault(r.Sender, &r.Packet, RESPONSE_BAD_TRANSPORT)
						break
					}
					item.Type = int(f)
					if f, ok = data["Spec"].(float64); !ok {
						SendPacketResponseFault(r.Sender, &r.Packet, RESPONSE_BAD_TRANSPORT)
						break
					}
					item.Spec = int(f)
					if f, ok = data["Key"].(float64); !ok {
						SendPacketResponseFault(r.Sender, &r.Packet, RESPONSE_BAD_TRANSPORT)
						break
					}
					item.Key = int(f)

					if item.Buffer, ok = data["Buffer"].(string); !ok {
						SendPacketResponseFault(r.Sender, &r.Packet, RESPONSE_BAD_TRANSPORT)
						break
					}
					fmt.Printf("pair key -> %d\n", item.Key)
					switch item.Type {
					case 1:
						fmt.Printf("[EVENT] -> PUSH -> CTRL+C\n")
						break
					case 2:
						fmt.Printf("[EVENT] -> PUSH -> CTRL+V\n")
						break
					case 3:
						fmt.Printf("[EVENT] -> PUSH -> CTRL+Q+C\n")
						if item.Spec == 2 {
							/* gzip*/
						}
						break
					}
					if pair, ok := h.pairTable[item.Key]; ok {
						var to *Client
						if r.Sender == pair.child {
							to = pair.creator
						} else {
							to = pair.child
						}
						if !IsOk(to) {
							break
						}

						SendPacket(to, CreatePacket(SERVER_CLIPBOARD_PUSH, item))
						/*SNetworkClipboardItem{
							Buffer: item.Buffer,
							Type:   item.Type,
						}))*/
						SendPacket(r.Sender,
							CreatePacketResponse(&r.Packet, struct {
								Response int
							}{
								Response: 1,
							}))
					} else {
						SendPacketResponseFault(r.Sender, &r.Packet, RESPONSE_ARG_BAD_FORMAT)
					}
					break
				}
				break
			case 1:
				/* stop all clients, and signal close*/
				for i, _ := range h.clients {
					delete(h.clients, i)
					close(i.send)
				}
				break
			}
			break
		case <-ticker.C:
			for i, v := range h.lostClients {
				if ((v.timeLost + 1000) * 60) < GetTimeUnixMilliseconds() {
					fmt.Printf("[router] deleting expired client\n")
					/* expired */
					deleteClient(v)
					v = nil

					//delete it
					h.lostClients = append(h.lostClients[:i], h.lostClients[i+1:]...)
				}
			}
			break
		}

	}
	return 0
}

func (c *Client) readThread() {
	fmt.Printf("[client] socket read thread starting\n")
	defer func() {
		c.router.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(SOCKET_MAX_MESSAGE_SIZE)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[client] error: %v", err)
			}
			break
		}
		fmt.Printf("[client] read message\n")
		var msg SThreadClientMessage
		msg.Sender = c
		decode := json.Unmarshal(message, &msg.Packet)

		if decode != nil {
			log.Printf("[client] failed to decode message: %v", err)
			return
		}
		c.router.broadcast <- &SThreadMessage{
			Type: 0,
			Ptr:  msg,
		}
	}
}

func CreatePacketResponseEx(code int, callback int, ptr interface{}) *SNetworkPacketJson {
	return &SNetworkPacketJson{
		Type:      code,
		Time:      GetTimeUnixMilliseconds(),
		Transport: ptr,
		Callback:  callback,
	}
}
func CreatePacketResponse(in *SNetworkPacketJson, ptr interface{}) *SNetworkPacketJson {
	return &SNetworkPacketJson{
		Type:      SERVER_RESPONSE_OK,
		Time:      GetTimeUnixMilliseconds(),
		Transport: ptr,
		Callback:  in.Callback,
	}
}
func CreatePacket(Type int, ptr interface{}) *SNetworkPacketJson {
	return &SNetworkPacketJson{
		Type:      Type,
		Time:      GetTimeUnixMilliseconds(),
		Transport: ptr,
		Callback:  -1,
	}
}
func SendPacketResponseFault(to *Client, msg *SNetworkPacketJson, code int) bool {
	fmt.Println("[net] SendPacketResponseFault")
	select {
	case to.send <- &SThreadMessage{
		Type: 0,
		Ptr: CreatePacketResponseEx(SERVER_RESPONSE_BAD, msg.Callback,
			struct {
				Response int
			}{
				Response: code,
			}),
	}:
	default:
		fmt.Printf("[net] SendPacketResponseFault -> failed\n")
		return false
	}
	return true
}
func SendPacket(to *Client, msg *SNetworkPacketJson) bool {
	select {
	case to.send <- &SThreadMessage{
		Type: 0,
		Ptr:  msg,
	}:
	default:
		fmt.Printf("SendPacket -> failed\n")
		return false
	}
	return true
}
func SendPacketEx(from *Client, to *Client, msg *SNetworkPacketJson) bool {
	callback := msg.Callback
	if from != to {
		msg.Callback = 0
	}
	select {
	case to.send <- &SThreadMessage{
		Type: 0,
		Ptr:  msg,
	}:
	default:
		fmt.Printf("SendPacket -> failed\n")
		msg.Callback = callback
		return false
	}
	msg.Callback = callback
	return true
}

func (c *Client) writeThread() {
	fmt.Printf("[client] socket write thread starting\n")
	ticker := time.NewTicker(SOCKET_PING_PERIOD)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			fmt.Printf("[client] write message\n")
			c.conn.SetWriteDeadline(time.Now().Add(SOCKET_WRITE_WAIT))
			if !ok {
				fmt.Printf("[client] failed set write deadline\n")
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				fmt.Printf("[client] failed to get next writer\n")
				return
			}
			var packet SNetworkGroup
			packet.Data = append(packet.Data, message.Ptr)
			n := len(c.send)
			for i := 1; i < n; i++ {
				message, ok := <-c.send
				if !ok {
					continue
				}
				packet.Data = append(packet.Data, message.Ptr)
			}
			data, err := json.Marshal(packet)
			if err != nil {
				fmt.Printf("[client] write -> failed to marshal output\n")
				return
			}
			w.Write(data)
			if err := w.Close(); err != nil {
				fmt.Printf("[client] failed to close writer\n")
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(SOCKET_WRITE_WAIT))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var upgrader = websocket.Upgrader{}

func DoRouterWaitResponse(h *SocketRouter, notifier http.CloseNotifier, Header int, Ptr interface{}) interface{} {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *SThreadMessage)
	h.broadcast <- &SThreadMessage{
		Type: 1001,
		Ptr: &SReadResponse{
			Type:         Header,
			ChanResponse: ch,
			Context:      ctx,
			Ptr:          Ptr,
		},
	}
	var ret interface{}
	select {
	case result := <-ch:
		ret = result.Ptr
		cancel() //notify remote thread?
		return ret
	case <-time.After(time.Second * 10):
		fmt.Println("Timed out waiting.")
	case <-notifier.CloseNotify():
		fmt.Println("Client has disconnected.")
	}
	cancel() //notify remote thread
	<-ch     //close
	return nil
}

func startClient(core *SocketRouter, w http.ResponseWriter, r *http.Request) int {
	//cookie1 := &http.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
	//http.SetCookie(w, cookie1)

	fmt.Printf("[upgrade] -> start client\n")
	notifier, ok := w.(http.CloseNotifier)
	if !ok {
		panic("[upgrade] Expected http.ResponseWriter to be an http.CloseNotifier")
	}

	session, ok := r.Context().Value("sessionstore").(*sessions.Session)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return -2
	}
	sessionwrapper := Ct_session(session)
	if sessionwrapper.GetUUID() < 0 {

	}

	oldUUID := sessionwrapper.GetUUID()
	clientExists := false
	var clientPtr *Client
	if oldUUID > 0 {
		ret := DoRouterWaitResponse(core, notifier, 0, oldUUID)
		if ret == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return -1
		} else if ret == false {
			clientExists = false
		} else {
			clientExists = true
			clientPtr = ret.(*Client)
		}
	}
	if clientExists {
		session.Values[SESSION_UUID] = clientPtr.cookieUUID
		if err := session.Save(r, w); err != nil {
			fmt.Printf("[upgrade] session write failed to save: %s\n", err)
			log.Fatal(err)
		}
	} else {
		core.UUIDCounter++
		session.Values[SESSION_UUID] = core.UUIDCounter
		if err := session.Save(r, w); err != nil {
			fmt.Printf("[upgrade] session write failed to save: %s\n", err)
			log.Fatal(err)
		}
	}
	fmt.Printf("[upgrade] ID: %s\n", session.ID)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return -1
	}
	var client *Client
	/* This is all in-testing, expect it to break*/
	if clientExists {
		fmt.Printf("[upgrade] cloned old session\n")
		//client = clientPtrs

		//client.pairTable = make(map[int]*SPairTable)
		client = clientPtr
		client.pairTable = clientPtr.pairTable
		client.router = core
		client.conn = conn
		client.send = make(chan *SThreadMessage, 256)
		//client.cookieUUID = core.UUIDCounter
		client.dead = false
	} else {
		fmt.Printf("[upgrade] using new session\n")
		/* overwrite client ptr to keep pointers */
		client = &Client{
			router:     core,
			conn:       conn,
			send:       make(chan *SThreadMessage, 256),
			cookieUUID: core.UUIDCounter,
			pairTable:  make(map[int]*SPairTable),
		}
	}
	client.router.register <- client //tell core pump about the new client request

	go client.writeThread()
	go client.readThread()
	fmt.Printf("[upgrade] client started\n")

	return 0
}
