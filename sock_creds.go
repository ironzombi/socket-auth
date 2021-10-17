package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"socket/auth" // change for $GOPATH
)

// go run creds.go <group name allowed to connect to the socket>
func init() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(),
			"Usage:\n\t%s <group names>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

// add the groups to our map 'groups'
func parseGroupNames(args []string) map[string]struct{} {
	groups := make(map[string]struct{})

	for _, arg := range args {
		grp, err := user.LookupGroup(arg)
		if err != nil {
			log.Println(err)
			continue
		}

		groups[grp.Gid] = struct{}{}
	}

	return groups
}

// send and recieve a ping pong as demo
func handleConn(conn *net.UnixConn) error {
	msg := []byte("PING")
	b, err := conn.Write(msg)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("wrote %d", b)

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%s", buf)

	return nil
}

func main() {
	flag.Parse()

	groups := parseGroupNames(flag.Args())
	socket := filepath.Join(os.TempDir(), "cred.sock")
	addr, err := net.ResolveUnixAddr("unix", socket)
	if err != nil {
		log.Fatal(err)
	}

	s, err := net.ListenUnix("unix", addr)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		_ = s.Close()
	}()

	fmt.Printf("Listening on %s ..\n", socket)

	for {
		conn, err := s.AcceptUnix()
		if err != nil {
			break
		}
		if auth.Allowed(conn, groups) {
			_, err = conn.Write([]byte("AUTHENTICATED\n"))
			if err == nil {
				err := handleConn(conn)
				if err != nil {
					fmt.Println(err)
				}
				continue
			}
		} else {
			_, err = conn.Write([]byte("ACCESS DENIED\n"))
		}
		if err != nil {
			log.Println(err)
		}
		_ = conn.Close()
	}
}
