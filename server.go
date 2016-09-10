package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"time"
	"io"
	"strings"
	//"bufio"

	"github.com/gorilla/websocket"

	"github.com/zonesan/undefined/chat"
)

const (
	MaxBufferOutputBytes = 1024
)

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
		httpTemplate, err = template.ParseFiles(getTemplateFilePath("home.html"))
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

type ChatConn struct { // implement chat.ReadWriteCloser
	Conn *websocket.Conn

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
	//  from++
	//  to++
	//}
	copy(all_b[old_n:], newb)

	return all_b
}

//====================================================================
// ChatConn implements net.Conn
//====================================================================

func (cc *ChatConn) Read(b []byte) (int, error) {
	// read from buffer
	//bs := []byte{1,   0, 1, 12, 2, 22,   2,   0, 2, 12, 1, 22}
	//nn := copy(b, bs)
	//return nn, nil

	index := 0
	n, err := cc.InputBuffer.Read(b)
	index += n
	if err != nil && err != io.EOF {
		return index, err
	}

	if index > 0 {
		return index, nil
	}

	for {
		// try to read more message data and cache it
		messageType, p, err := cc.Conn.ReadMessage()
		if err != nil && err != io.EOF {
			return index, err
		}
		if messageType != websocket.BinaryMessage { // only accept BinaryMessage messages
			continue
		}

		// n2 must be len(p) if err2 is nil
		n2, err2 := cc.InputBuffer.Write(p) // cache it
		if err2 != nil {
			return index, err2
		}
		if n2 > 0 {
			break
		}
	}

	// read from buffer again

	n, err = cc.InputBuffer.Read(b[index:])
	index += n
	if err != nil && err != io.EOF {
		return index, err
	}

	return index, nil
}

/*
func (cc *ChatConn) Read(b []byte) (int, error) {
	var from = 0
	from = cc.ReadFromBuffer(b, from)

if from != 0 {
	fmt.Println("from = ", from)
}
	if from == len(b) {
		return from, nil
	}

if from != 0 {
	fmt.Println("from = ", from)
}

	var messageType, p, err = cc.Conn.ReadMessage()
	if err != nil || messageType != websocket.TextMessage { // only TextMessage is suppported now
		return from, err
	}

if len(p) != 0 {
	fmt.Println("p = ", from)
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
*/

func (cc *ChatConn) Write(b []byte) (int, error) {
	return len(b), cc.Conn.WriteMessage(websocket.BinaryMessage, b) // todo: maybe not ok
}

/*
func (cc *ChatConn) Write2(newb []byte) (int, error) {
	b := cc.MergeOutputBuffer(newb)

	var n = len(b)
	var from = 0
	var to = 0
	var err error
	for to < n {
		if b[to] == '\n' {
			if to-from > 0 {
				err = cc.Conn.WriteMessage(websocket.TextMessage, b[from:to+1])
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
*/

/*
func (cc *ChatConn) SetDeadline(t time.Time) error {
	var err = cc.SetReadDeadline(t)
	if err == nil {
		err = cc.SetWriteDeadline(t)
	}

	return err
}
*/

func (cc *ChatConn) Close() error {
	return cc.Conn.Close()
}

func (cc *ChatConn) LocalAddr() net.Addr {
	return cc.Conn.LocalAddr()
}

func (cc *ChatConn) RemoteAddr() net.Addr {
	return cc.Conn.RemoteAddr()
}

func (cc *ChatConn) SetDeadline(t time.Time) error {
	var err = cc.Conn.SetReadDeadline(t)
	if err == nil {
		err = cc.Conn.SetWriteDeadline(t)
	}

	return err
}

func (cc *ChatConn) SetReadDeadline(t time.Time) error {
	return cc.Conn.SetReadDeadline(t)
}

func (cc *ChatConn) SetWriteDeadline(t time.Time) error {
	return cc.Conn.SetWriteDeadline(t)
}

//====================================================================
// ...
//====================================================================

const PrefixWS = "/ws/"

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

	if (! strings.HasPrefix(r.URL.Path, PrefixWS)) || len(r.URL.Path) <= len(PrefixWS) {
		http.Error(w, "bad uri", 400)
		return
	}
	roomId := r.URL.Path[len(PrefixWS):]

	var wsConn, err = wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Method not allowed", 405)
		return
	}

	chatServer.OnNewConnection(&ChatConn{Conn: wsConn}, roomId)
}

func createWebsocketServer(port int) {

	log.Printf("Websocket listening at :%d ...\n", port)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc(PrefixWS, websocketHandler) // /ws/:rommid
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
			chatServer.OnNewConnection(conn, "")
		}
	}
}

var chatServer *chat.Server

func main() {
	chatServer = chat.CreateChatServer()

	go createSocketServer(6789)

	createWebsocketServer(5678)
}