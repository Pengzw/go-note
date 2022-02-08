
## 什么是粘包
代码示例

server.go
``` 
package main

import (
	"bufio"
	"fmt"
	"net"
)

func main()  {
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("listened failed, err:", err)
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("accept failed, err:", err)
		}
		// 起一个协程处理连接
		go process(conn)
	}
}

func process(conn net.Conn)  {
	defer conn.Close()
	for {
		reader := bufio.NewReader(conn)
		var buf [128]byte
		n, err := reader.Read(buf[:])
		if err != nil {
			fmt.Println("read from conn err:", err)
			break
		}
		recvStr := string(buf[:n])
		fmt.Println("从client收到消息", recvStr)
	}
}
```
client.go

```
package main

import (
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		fmt.Println("connect to server err:", err)
		return
	}
	defer conn.Close()

	//一次性发送多个消息，会出现tcp粘包现象，这样会导致多条数据粘在一起
	//主要原因就是tcp数据传递模式是流模式，在保持长连接的时候可以进行多次的收和发。
	for i := 0; i < 20; i++ {
		msg := `hello, server`
		conn.Write([]byte(msg))
	}
}
```
当client连续向server端连续发送20个数据包的时候，我们看server端打印的内容。

```
从client收到消息 hello, server
从client收到消息 hello, serverhello, serverhello, serverhello, server
从client收到消息 hello, serverhello, server
从client收到消息 hello, serverhello, serverhello, serverhello, serverhello, serverhello, server
从client收到消息 hello, serverhello, serverhello, serverhello, serverhello, serverhello, serverhello, server
read from conn err: EOF
```
按照我们预想的，server端应该受到20条消息，每一条消息只包含hello，server才对。然而server却收到了不到20条消息，而且消息的长短不一。这就是粘包现象。

## 为什么会出现粘包
主要原因是tcp数据传递的模式是流模式，在保持长连接的时候可以进行多次收和发。

粘包可能发生在发送端也可能发生在接收端：
- 由Nagle算法造成的发送端粘包：Nagle算法是一种改善网络传输效率的算法。简单的来说就是当我们提交一段数据给TCP发送的时候，TCP并不会立即的发送这段数据，而是等一小段时间看看在等待的时间内是否还有其他的数据要发送，若有则一次性把两段数据发送出去。
- 接收端接收不及时造成的粘包：接收端TCP会把收到的数据写入一个缓冲区，然后通知应用层取数据。当应用层由于某些原因不能及时的把数据取走，就会造成TCP缓冲区堆积，存放了几段数据包，造成粘包现象。

## 如何解决粘包
出现粘包的关键在于接收方不能够确定将要接收的数据包的大小，因此我们需要手动对数据进行封包和拆包操作。

封包：封包就是给一段数据加上包头，这样一来数据包就分为包头和包体两个部分内容了（过滤非法包时封包还会加入包尾）。包头部分的长度是固定的，并且它存储了包体的长度。根据包头的长度固定以及包头中所包含的包体的长度就能够正确的实现拆分出一个完整的数据包。

### 怎么去封包、解包呢？

我们可以自己定义一种协议规定，比如数据包的前几个字节为包头，里面存储的是发送的数据的长度。

封包示例

tcp_stick_proto.go
```
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
)

// 封包
func ProtoEncode(message string) ([]byte, error) {
	length := int32(len(message)) // 32位 占4字节
	packet := new(bytes.Buffer)
	// 包头：message的长度
	err := binary.Write(packet, binary.LittleEndian, length)
	if err != nil {
		return nil, err
	}

	// 包体：message
	err = binary.Write(packet, binary.LittleEndian, []byte(message))
	if err != nil {
		return nil, err
	}
	return packet.Bytes(), nil
}

// 解包
func ProtoDecode(reader *bufio.Reader) (string, error) {
	lengthByte, err := reader.Peek(4) // 读取前4个字节
	if err != nil {
		return "", err
	}
	lengthBuff := bytes.NewBuffer(lengthByte) // 用这个字节切片 创建一个用于读取数据的buffer；

	var length int32
	err = binary.Read(lengthBuff, binary.LittleEndian, &length)
	if err != nil {
		return "", err
	}

	// 当前reader可以读取的字节数 小于 头部规定的数据长度+4，说明数据丢失，返回error
	if int32(reader.Buffered()) < length+4 {
		return "", err
	}

	packet := make([]byte, int32(4+length))
	_, err = reader.Read(packet)
	if err != nil {
		return "", err
	}
	return string(packet[4:]), nil
}
```

新修改如下：

server.go
```
func process(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		recvStr, err := ProtoDecode(reader)
		if err != nil {
			fmt.Println("read from conn err:", err)
			break
		}
		fmt.Println("从client收到消息", recvStr)
	}
}
```

client.go
```
for i := 0; i < 20; i++ {
		msg := `hello, server`
		encode, err := ProtoEncode(msg)
		if err != nil {
			panic(err)
		}
		conn.Write(encode)
	}
```

## 为什么UDP不存在粘包问题
UDP不存在粘包问题，是由于UDP发送的时候，没有经过Negal算法优化，不会将多个小包合并一次发送出去。另外，在UDP协议的接收端，采用了链式结构来记录每一个到达的UDP包，这样接收端应用程序一次recv只能从socket接收缓冲区中读出一个数据包。也就是说，发送端send了几次，接收端必须recv几次（无论recv时指定了多大的缓冲区）

## 总结几种 socket 粘包的解包方式：

1. fix length
    消息定长
    无论一次接收到多少数据报，该方式都会按照设置的固定长度进行解码; 
    如果是半包消, 那么会缓存当前包并等待下个包到达后进行拼包, 直到获取到指定长度的一个完整包;
    应用场景:
        运用于数据长度固定的粘包拆包场景, 比如指令发送
    
2. delimiter based
    设置消息边界
    服务端从网络流中按消息边界分离出消息内容。在包尾增加回车换行符或其他特殊字符进行分割，例如FTP协议。

3. length field based frame decoder
    基于数据包长度的解码
    将应用层数据包的长度，作为接收端应用层数据包的拆分依据。 按照应用层数据包的大小，拆包。 
    这个拆包器，有一个要求，就是应用层协议中包含数据包的长度。
    处理粘拆包的主要思想是在生成的数据包中添加一个长度字段，用于记录当前数据包的长度。 
    LengthFieldBasedFrameDecoder会按照参数指定的包长度偏移量数据对接收到的数据进行解码，从而得到目标消息体数据;
    
