# undefined


```



========= 格式定义

[command(1), ...]

========= 命令类型

Command_MousePosition = 0
Command_MouseDown = 1
Command_MouseUp = 2

Command_UserInfo = 254
Command_ServerVersion = 255

========= 发送格式

[Command_MousePosition, x1, x0, y1, y0]
[Command_MouseDown]
[Command_MouseUp]

percentX = (absoluteX / cancasWidth) * 0xFFFF
x1 = (percentX >> 8) & 0xFF
x0 = (percentX >> 0) & 0xFF

percentY = (absoluteY / cancasHeight) * 0xFFFF
y1 = (percentY >> 8) & 0xFF
y0 = (percentY >> 0) & 0xFF

========= 接收解析

[Command_MousePosition, x1, x0, y1, y0, ID3, ID2, ID1, ID0]
[Command_MouseDown, ID3, ID2, ID1, ID0]
[Command_MouseUp, ID3, ID2, ID1, ID0]

[Command_UserInfo, ID3, ID2, ID1, ID0, color2, color1, color0]
[Command_ServerVersion, version1, version0]

absoluteX = cancasWidth * ((x1 << 8) | (x0 << 0)) / 0xFFFF
absoluteY = cancasHeight * ((y1 << 8) | (y0 << 0)) / 0xFFFF

userId = (ID3 << 24) | (ID2 << 16) | (ID1 << 8) | (ID0 << 0)

userColor = (color2 << 16) | (color1 << 8) | (color0 << 0)

serverVersion = (version1 << 8) | (version0 << 0)





```