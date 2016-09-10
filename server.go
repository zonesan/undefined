package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"time"

	//"io"
	//"strings"
	//"bufio"

	"github.com/gorilla/websocket"

	"github.com/zonesan/undefined/chat"
)

//====================================================================
// home page
//====================================================================

func getTemplateFilePath(tn string) string {
	return "template/" + tn
}

func sendPageData(w http.ResponseWriter, pageDataBytes []byte, contextType string) error {

	w.Header().Set("Content-Type", contextType)

	var numWrites int
	var err error

	numBytes := len(pageDataBytes)
	for numBytes > 0 {
		numWrites, err = w.Write(pageDataBytes)
		if err != nil {
			return err
		}
		numBytes -= numWrites
	}

	return nil
}

var httpTemplate *template.Template
var httpContentCache []byte

func httpHandler(w http.ResponseWriter, r *http.Request) {

	//>> for debug
	var httpTemplate *template.Template = nil
	var httpContentCache []byte = nil
	//<<

	var err error

	if httpTemplate == nil {
		httpTemplate, err = template.ParseFiles(getTemplateFilePath("index.html"))
		if err != nil {
			sendPageData(w, []byte("Parse template error."), "text/plain; charset=utf-8")
			return
		}
	}

	if httpContentCache == nil {
		var buf bytes.Buffer
		err = httpTemplate.Execute(&buf, nil)
		if err != nil {
			sendPageData(w, []byte("Render page error."), "text/plain; charset=utf-8")
			return
		}

		httpContentCache = buf.Bytes()
	}

	sendPageData(w, httpContentCache, "text/html; charset=utf-8")
}

//====================================================================
// ChatConn
//====================================================================

const (
	MaxBufferOutputBytes = 1024
)

type ChatConn struct { // implement chat.ReadWriteCloser
	*websocket.Conn

	InputBuffer  bytes.Buffer
	OutputBuffer bytes.Buffer
}

func (cc *ChatConn) ReadFromBuffer(b []byte, from int) int {
	to := len(b) - from
	if to > cc.InputBuffer.Len() {
		to = cc.InputBuffer.Len()
	}

	to += from
	for from < to {
		b[from], _ = cc.InputBuffer.ReadByte()
		from++
	}

	return from
}

func (cc *ChatConn) MergeOutputBuffer(newb []byte) []byte {
	var old_n = cc.OutputBuffer.Len()
	if old_n == 0 {
		return newb
	}

	var new_n = len(newb)
	var all_n = old_n + new_n
	var all_b = make([]byte, all_n)
	cc.OutputBuffer.Read(all_b)
	//var to = old_n
	//var from = 0
	//for from < new_n {
	//	all_b[to] = newb[from]
	//	from++
	//	to++
	//}
	copy(all_b[old_n:], newb)

	return all_b
}

//====================================================================
// ChatConn implements net.Conn
//====================================================================

func (cc *ChatConn) Read(b []byte) (int, error) {
	var from = 0
	from = cc.ReadFromBuffer(b, from)
	if from == len(b) {
		return from, nil
	}

	var messageType, p, err = cc.ReadMessage()
	if err != nil || messageType != websocket.BinaryMessage { // only BinaryMessage is suppported now
		return from, err
	}

	_, err = cc.InputBuffer.Write(p)
	if err != nil {
		return from, err
	}

	err = cc.InputBuffer.WriteByte('\n')
	if err != nil {
		return from, err
	}

	from = cc.ReadFromBuffer(b, from)

	return from, nil
}

func (cc *ChatConn) Write(newb []byte) (int, error) {
	b := cc.MergeOutputBuffer(newb)

	var n = len(b)
	var from = 0
	var to = 0
	var err error
	for to < n {
		if b[to] == '\n' {
			if to-from > 0 {
				err = cc.WriteMessage(websocket.TextMessage, b[from:to+1])
				if err != nil {
					break
				}
			}

			from = to + 1
		}

		to++
	}

	if from < n {
		cc.OutputBuffer.Write(b[from:])
	}

	//if (cc.OutputBuffer.Len() > MaxBufferOutputBytes)
	//  err = error.New ("...")

	return len(newb), nil
}

func (cc *ChatConn) SetDeadline(t time.Time) error {
	var err = cc.SetReadDeadline(t)
	if err == nil {
		err = cc.SetWriteDeadline(t)
	}

	return err
}

//====================================================================
// ...
//====================================================================

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var wsConn, err = wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Method not allowed", 405)
		return
	}

	chatServer.OnNewConnection(&ChatConn{Conn: wsConn})
}

func createWebsocketServer(port int) {

	log.Printf("Websocket listening at :%d ...\n", port)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/ws", websocketHandler)
	http.HandleFunc("/", httpHandler)

	var address = fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(address, nil)

	if err != nil {
		log.Fatal("Websocket server failt to start: ", err)
	}
}

func createSocketServer(port int) {

	var address = fmt.Sprintf(":%d", port)
	var listener, err = net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("General socket listen error: %s\n", err.Error())
	}

	log.Printf("General socket listening at %s: ...\n", listener.Addr())

	for {
		var conn, err = listener.Accept()
		if err != nil {
			log.Printf("General socket accept new connection error: %s\n", err.Error())
		} else {
			chatServer.OnNewConnection(conn)
		}
	}
}

var chatServer *chat.Server

func main() {
	chatServer = chat.CreateChatServer()

	go createSocketServer(6789)

	createWebsocketServer(5678)
}