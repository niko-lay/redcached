package rcdaemon

import (
	"bufio"
	"../protocol"
	"io"
	"log"
	"net"
	"strings"
)

type HandlerFn func(req *protocol.McRequest, res *protocol.McResponse) error

type Client struct {
	Addr    string               // conn.RemoteAddr().String()
	Conn    net.Conn             // i/o connection
	methods map[string]HandlerFn // refer to Server.methods
}

func NewClient(conn net.Conn, srv *Server) (c *Client, err error) {
	// TODO set start time

	// TODO set
	//conn.SetKeepAlive(true)
	//conn.SetKeepAlivePeriod(3 * time.Minute)

	return &Client{
		Addr:    conn.RemoteAddr().String(),
		Conn:    conn,
		methods: srv.methods,
	}, nil
}

func (client *Client) Serve() (err error) {
	conn := client.Conn
	defer func() {
		if err != nil {
			// fmt.Fprintf(client.Conn, "-%s\n", err)
		}
		conn.Close()
	}()

	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)

	for {
		req, err := protocol.ReadRequest(br)
		if perr, ok := err.(protocol.ProtocolError); ok {
			log.Printf("%v ReadRequest protocol err: %v", conn, err)
			bw.WriteString("CLIENT_ERROR " + perr.Error() + "\r\n")
			bw.Flush()
			continue
		} else if err == io.EOF {
			log.Printf("client closed connection (got EOF)")
			return nil
		} else if err != nil {
			log.Printf("%v ReadRequest err: %v", conn, err)
			return err
		}
		log.Printf("%v Req: %+v\n", conn, req)

		cmd := strings.ToLower(req.Command)
		if cmd == "quit" {
			log.Printf("client sent quit, connection closed")
			return nil
		}

		res := &protocol.McResponse{}
		fn, exists := client.methods[cmd]
		if exists {
			err := fn(req, res)
			if err != nil {
				log.Printf("ERROR: %v, Conn: %v, Req: %+v\n", err, conn, req)
				res.Response = "SERVER_ERROR " + err.Error()
			}
			if !req.Noreply {
				//log.Printf("%v Res: %+v\n", conn, res)
				bw.WriteString(res.Protocol())
				bw.Flush()
			}
		} else {
			res.Response = "ERROR not implemented cmd '" + cmd + "' in handler"
			bw.WriteString(res.Protocol())
			bw.Flush()
		}
	}
	return nil
}
