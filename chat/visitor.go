package chat

import (
	"bufio"
	"container/list"
	"fmt"
	//"io"
	"log"
	"net"
	"strings"
)

type Visitor struct {
	Server *Server

	Connection     net.Conn
	//LimitInput     *io.LimitedReader
	//Input          *bufio.Reader
	Output         *bufio.Writer
	OutputMessages chan []byte

	Name     string
	NextName string

	LocalIdInRoom byte
	RoomElement *list.Element // in CurrentRoom.Visitors
	CurrentRoom *Room
	NextRoomID  string
	RoomChanged chan int

	ReadClosed  chan int
	WriteClosed chan int
	Closed      chan int
}

func (server *Server) createNewVisitor(c net.Conn, name string, initialRoom string) *Visitor {
	var visitor = &Visitor{
		Server: server,

		LocalIdInRoom : 0,

		Connection:     c,
		OutputMessages: make(chan []byte, MaxVisitorBufferedMessages),

		Name:     name,
		NextName: "",

		RoomElement: nil,
		CurrentRoom: nil,
		NextRoomID:  VoidRoomID,
		RoomChanged: make(chan int),

		ReadClosed:  make(chan int),
		WriteClosed: make(chan int),
		Closed:      make(chan int),
	}

	//visitor.LimitInput = &io.LimitedReader{c, 0}
	//visitor.Input = bufio.NewReader(visitor.LimitInput)
	//visitor.Input = bufio.NewReader(c)
	visitor.Output = bufio.NewWriter(c)

	server.Visitors[strings.ToLower(name)] = visitor

	//visitor.beginChangingRoom(LobbyRoomID)
	visitor.beginChangingRoom(initialRoom)

	log.Printf("New visitor: %s", visitor.Name)

	return visitor
}

func (visitor *Visitor) changeName() {
	var server = visitor.Server

	var newName = visitor.NextName

	rn := []rune(newName)
	if len(rn) < MinVisitorNameLength {
		return
	}

	if len(rn) > MaxVisitorNameLength {
		newName = string(rn[:MaxVisitorNameLength])
	}
	newName = server.NormalizeName(newName)
	if len(newName) < MinVisitorNameLength || len(newName) > MaxVisitorNameLength {
		return
	}

	fmt.Printf("len = %d, min = %d", len(newName), MinVisitorNameLength)

	if server.Visitors[strings.ToLower(newName)] != nil {
		return
	}

	delete(server.Visitors, strings.ToLower(visitor.Name))
	visitor.Name = newName
	server.Visitors[strings.ToLower(visitor.Name)] = visitor

	//visitor.OutputMessages <- server.CreateMessage("Server", fmt.Sprintf("you changed your name to %s", visitor.Name))
}

func (server *Server) destroyVisitor(visitor *Visitor) {
	if visitor.CurrentRoom != nil {
		log.Printf("destroyVisitor: visitor.CurrentRoom != nil")
	}

	delete(server.Visitors, strings.ToLower(visitor.Name))

	visitor.closeConnection()
}

func (visitor *Visitor) closeConnection() error {
	//defer func() {
	//    if err := recover(); err != nil {
	//        log.Println ("CloseConnection paniced", err)
	//    }
	//}()
	// above is to avoid panic on reclose a connection. It may be not essential.

	return visitor.Connection.Close()
}

func (visitor *Visitor) beginChangingRoom(newRoomID string) {
	visitor.RoomChanged = make(chan int) // to block Visitor. Read before room is changed.
	visitor.NextRoomID = newRoomID
	visitor.Server.ChangeRoomRequests <- visitor
}

func (visitor *Visitor) endChangingRoom() {
	visitor.NextRoomID = VoidRoomID
	close(visitor.RoomChanged)
}

func (visitor *Visitor) newVisitorMessage(data[] byte) []byte {
	message := make([]byte, len(data) + 1)
	copy(message[0:], data)
	message[len(data)] = visitor.LocalIdInRoom
	return message
}

func (visitor *Visitor) run() {
	go visitor.read()
	go visitor.write()

	<-visitor.WriteClosed
	<-visitor.ReadClosed
	close(visitor.Closed)

	// let server close visitor
	visitor.beginChangingRoom(VoidRoomID)
}

func (visitor *Visitor) read() {
	var server = visitor.Server
	_ = server

	//visitor.LimitInput.N = MaxMessageLength
	//var inReadingLongMessage bool = false

	var DATA [MaxMessageLength]byte
	data := DATA[:]

	var index int
	var command int
	var done bool

	reset := func() {
		index = 0
		command = -1
		done = false
	}
	reset()

	for {
		select {
		//case <- visitor.ReadClosed:
		//   goto EXIT
		case <-visitor.WriteClosed:
			goto EXIT
		case <-visitor.Closed:
			goto EXIT
		default:
		}

		<-visitor.RoomChanged // wait server change room for vistor, when server has done it, this channel will be closed.

//fmt.Println("000 command=", command)
		switch command {
		default: // command is not confirmed yet
			dataLength := 1
			if index >= dataLength {
				goto EXIT
			}
			n, err := visitor.Connection.Read(data[index:dataLength])

fmt.Println("111 aaaa, ", n, ", err = ", err)
			if err != nil { // todo: check io.EOF
				goto EXIT
			}
			index += n
			if index == dataLength {
				command = int(data[0])
			}

fmt.Println("111 bbb, ", command)
		case Command_MousePosition:
			dataLength := 6
			if index >= dataLength {
				goto EXIT
			}
			n, err := visitor.Connection.Read(data[index:dataLength])

fmt.Println("222 aaaa, ", n)
			if err != nil { // todo: check io.EOF
				goto EXIT
			}
			index += n
			if index == dataLength {
				done = true
			}
		}

		if done {

fmt.Println("5555 command = ", command, ": ", data[:index])
			visitor.OutputMessages <- visitor.newVisitorMessage(data[:index])
			reset()
		}

		/*
		var line, err = visitor.Input.ReadString('\n') // todo: use io.LimitedReader insstead
		if err != nil {
			if err != io.EOF || line == "" {
				goto EXIT
			}
		}

		visitor.LimitInput.N = int64(len(line)) + visitor.LimitInput.N

		rn := []rune(line)
		if len(rn) > MaxMessageLength || len(rn) == MaxMessageLength && line[len(line)-1] != '\n' {
			if !inReadingLongMessage {
				visitor.OutputMessages <- server.CreateMessage("Server", "your message is too long!")
			}
			inReadingLongMessage = line[len(line)-1] != '\n'
			continue
		} else if inReadingLongMessage {
			inReadingLongMessage = false
			continue
		}

		if strings.HasPrefix(line, "/") {
			if strings.HasPrefix(line, "/exit") {
				goto EXIT
			} else if strings.HasPrefix(line, "/room") {
				line = strings.TrimPrefix(line, "/room")
				line = strings.TrimSpace(line)
				if len(line) == 0 { // show current room name
					if visitor.CurrentRoom == nil {
						visitor.OutputMessages <- server.CreateMessage("Server", "you are in lobby now")
					} else {
						visitor.OutputMessages <- server.CreateMessage("Server", fmt.Sprintf("your are in %s now}", visitor.CurrentRoom.Name))
					}
				} else { // change room,
					line = server.NormalizeName(line)

					visitor.beginChangingRoom(line)
				}

				continue
			} else if strings.HasPrefix(line, "/name") {
				line = strings.TrimPrefix(line, "/name")
				line = strings.TrimSpace(line)

				if len(line) == 0 {
					visitor.OutputMessages <- server.CreateMessage("Server", fmt.Sprintf("your name is %s}", visitor.Name))
				} else if len(line) >= MinVisitorNameLength && len(line) <= MaxVisitorNameLength {
					visitor.NextName = line
					server.ChangeNameRequests <- visitor
				}

				continue
			}
		}

		if visitor.CurrentRoom != nil {
			if visitor.CurrentRoom == server.Lobby {
				visitor.OutputMessages <- server.CreateMessage("Server", "you are current in lobby, please input /room room_name to enter a room")
			} else {
				visitor.CurrentRoom.Messages <- server.CreateMessage(visitor.Name, line)
			}
		}
		*/
	}

EXIT:

	close(visitor.ReadClosed)
}

func (visitor *Visitor) write() {
	for {
		select {
		case <-visitor.ReadClosed:
			goto EXIT
		//case <- visitor.WriteClosed:
		//   goto EXIT
		case <-visitor.Closed:
			goto EXIT
		case message := <-visitor.OutputMessages:
			num, err := visitor.Output.Write(message)
			_ = num
			if err != nil {
				goto EXIT
			}

			for {
				select {
				case message = <-visitor.OutputMessages:
					//num, err = visitor.Output.WriteString(message)
					num, err = visitor.Output.Write(message)
					_ = num
					if err != nil {
						goto EXIT
					}
				default:
					goto FLUSH
				}
			}

		FLUSH:

			if visitor.Output.Flush() != nil {
				goto EXIT
			}
		}
	}

EXIT:

	visitor.closeConnection() // to avoud read blocking

	close(visitor.WriteClosed)
}
