package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"../lib"
	"github.com/gorilla/websocket"
)

type SNetworkClipboardItem struct {
	Type   int
	Spec   int
	Buffer string
}

type SNetworkPacketJson struct {
	Type      int
	Time      int64
	Callback  int
	Transport interface{}
}

type SThreadMessage struct {
	Type int
	Ptr  interface{}
}

const (
	PROTOCOL_BAD = 0
	PROTOCOL_OK  = 1

	CLIENT_CONNECT        = 1
	CLIENT_DISCONNECT     = 2
	CLIENT_LIST           = 3
	CLIENT_PAIR_ROLL      = 4
	CLIENT_PAIR_CONNECT   = 5
	CLIENT_PUSH_CLIPBOARD = 6

	SERVER_CLIPBOARD_PUSH = 10
	SERVER_CONNECT_OK     = 11
	SERVER_CLOSING        = 12

	SERVER_RESPONSE_OK      = 13
	SERVER_RESPONSE_BAD     = 14
	RESPONSE_BAD_TRANSPORT  = -1
	RESPONSE_ARG_BAD_FORMAT = -2
	RESPONSE_ARG_INVALID    = -3
)

var send chan *SThreadMessage
var read chan *SThreadMessage
var callbackUUID int

func GetTimeUnixMilliseconds() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func CreatePacketAction(Type int, ptr interface{}, cb FNetworkCallback) *SNetworkPacketJson {
	callbackUUID++
	PushActionCallback(Type, 1000*10, callbackUUID, cb)
	return &SNetworkPacketJson{
		Type:      Type,
		Time:      GetTimeUnixMilliseconds(),
		Transport: ptr,
		Callback:  callbackUUID,
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

func SendPacket(msg *SNetworkPacketJson) bool {
	select {
	case send <- &SThreadMessage{
		Type: 0,
		Ptr:  msg,
	}:
	default:
		fmt.Printf("SendPacket -> failed\n")
		return false
	}
	return true
}

var actionTable []SNetworkAction

type SNetworkTable struct {
	Data []SNetworkPacketJson
}
type FNetworkCallback func(Type int, data *map[string]interface{})
type SNetworkAction struct {
	Type     int
	Callback int
	Timeout  int64
	SendTime int64
	onEvent  FNetworkCallback
}

func GetByCallback(uuid int) (int, *SNetworkAction) {
	for i, v := range actionTable {
		if v.Callback == uuid {
			return i, &v

		}
	}
	return -1, nil
}
func RemoveAction(i int) {
	actionTable = append(actionTable[:i], actionTable[i+1:]...)
}
func PushActionCallback(header int, timeout int, callback int, cb FNetworkCallback) {
	actionTable = append(actionTable, SNetworkAction{
		Type:     header,
		Callback: callback,
		Timeout:  1000,
		SendTime: GetTimeUnixMilliseconds(),
		onEvent:  cb,
	})
}
func PollActions() {
	//fmt.Printf("Scanning action table...")
	for i, v := range actionTable {
		if v.SendTime+v.Timeout < GetTimeUnixMilliseconds() {
			/* expired */
			fmt.Printf("[PollActions] -> Expired %d\n", i)
			v.onEvent(-1, nil)
			RemoveAction(i)
		}
	}
}

var s eventwindows.SHook
var addr = flag.String("addr", "localhost:8080", "http service address")
var flagPair = flag.String("pair", "", "key to pair to when program starts")
var loopRun bool

func main() {
	flag.Parse()
	log.SetFlags(0)

	send = make(chan *SThreadMessage, 0xFF)
	read = make(chan *SThreadMessage, 0xFF)

	events := make(chan *eventwindows.SEvent, 0xFFFF)
	interrupt := make(chan os.Signal, 1)

	signal.Notify(interrupt, os.Interrupt)
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})
	fmt.Printf("Starting hooks\n")
	s.Start(events)
	fmt.Printf("Starting read thread\n")
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			//log.Printf("recv: %s", message)
			var table SNetworkTable
			decode := json.Unmarshal(message, &table)
			if decode != nil {
				log.Printf("Failed to decode message: %v", err)
				return
			}
			for i, _ := range table.Data {
				read <- &SThreadMessage{
					Type: 0,
					Ptr:  table.Data[i],
				}
			}
		}
	}()

	fmt.Printf("Starting main pump thread\n")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	loopRun = true
	SendPacket(CreatePacket(1, nil))
	for {
		if !loopRun {
			break
		}
		select {
		case msg := <-send:
			switch msg.Type {
			case 0:
				r := msg.Ptr.(*SNetworkPacketJson)
				w, err := c.NextWriter(websocket.TextMessage)
				if err != nil {
					fmt.Printf("%v\n", err)
					return
				}
				data, err := json.Marshal(r)
				if err != nil {
					fmt.Printf("Failed to marshal output\n")
					fmt.Printf("%v\n", err)
					return
				}
				w.Write(data)
				if err := w.Close(); err != nil {
					fmt.Printf("%v\n", err)
					return
				}
				break
			}
			break
		case msg := <-read:
			switch msg.Type {
			case 0:
				r := msg.Ptr.(SNetworkPacketJson)
				var data map[string]interface{}
				fmt.Printf("MESSAGE -> ")
				switch r.Type {
				case SERVER_CLOSING:
					fmt.Printf("SERVER_CLOSING\n")
					break
				case SERVER_CONNECT_OK:
					fmt.Printf("SERVER_CONNECT_OK\n")
					if len(*flagPair) > 0 {
						i, err := strconv.Atoi(*flagPair)
						if err != nil {
							fmt.Printf("Failed to convert pair key\n", err)
							break
						}
						SendPacket(CreatePacketAction(CLIENT_PAIR_CONNECT, struct{ Key int }{Key: i},
							func(Type int, data *map[string]interface{}) {
								fmt.Printf("Callback\n")
								if Type < 0 {
									return
								}
								fmt.Printf("%v\n", data)
							}))
					}
					break
				default:
					if r.Type != SERVER_RESPONSE_OK && r.Type != SERVER_RESPONSE_BAD {
						fmt.Printf("[net] invalid message\n")
						break
					}
					fmt.Printf("SERVER_RESPONSE\n")
					var ok bool
					if data, ok = r.Transport.(map[string]interface{}); !ok {
						fmt.Printf("bad data\n")
						break
					}
					i, c := GetByCallback(r.Callback)
					if c == nil {
						fmt.Printf("callback doesn't exist!\n")
						break
					}
					c.onEvent(0, &data)
					RemoveAction(i)
					break
				}
				break
			}
			break
		case event := <-events:
			SendPacket(CreatePacket(6, &SNetworkClipboardItem{
				Type: event.Type,
				Spec: event.Spec,
			}))
			break
		case <-done:
			return
		case t := <-ticker.C:
			t.Hour()
			PollActions()
			//SendPacket(CreatePacket(2, nil))
		case <-interrupt:
			log.Println("[interrupt] -> interrupt\n")
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("[interrupt] -> write close:", err)
				fmt.Printf("[interrupt] -> set close\n")
				loopRun = false
				break
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			fmt.Printf("[interrupt] -> set close\n")
			loopRun = false
		}
	}
	fmt.Printf("network closed, unloading system hooks\n")
	s.Close()
	fmt.Printf("Shutdown\nGoodbye :)\n")
}