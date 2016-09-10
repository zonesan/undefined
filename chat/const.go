package chat

const (
	MaxRoomCapacity               = 8
	MaxRoomBufferedMessages       = 64
	MinVisitorNameLength          = 1
	MaxVisitorNameLength          = 32
	MaxVisitorBufferedMessages    = 64
	MaxPendingConnections         = 128
	MaxBufferedChangeRoomRequests = 128
	MaxBufferedChangeNameRequests = 64
	MaxMessageLength              = 1024
	LobbyRoomID                   = "Lobby"
	VoidRoomID                    = ""
)


const (
	Command_MousePosition = 0
	Command_MouseDown = 1
	Command_MouseUp = 2

	Command_UserInfo = 254
	Command_ServerVersion = 255
)