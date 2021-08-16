## ioer：UDP端口复用

****

当使用
```golang
net.DialUDP("udp",&net.UDPAddr{IP: nil,Port: 19986},&net.UDPAddr{IP: net.ParseIP("3.3.3.3"),Port: 19986})
```
后，本地的19986端口就不能再被使用了。但是理论上只要四元组中存在不同，就是可以建立连接的。

ioer实际是通过`net.ListenUDP`来实现的，发送数据时使用`conn.WriteTo`，接收数据时通过`raddr`将数据路由到对应的`ioer.Conn`。




