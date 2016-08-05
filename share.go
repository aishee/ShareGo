package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
  "runtime"
  "syscall"
  "strconv"
)

type Buffer struct {
	content string
}

func (buf *Buffer) set(val string) {
	buf.content = val
}

func (buf *Buffer) get() string {
	return buf.content
}

var (
	pidfile_path = "/var/run/sharego"
	pidfile_name = "sharego.pid"
	unix_sock    = "/tmp/sharego_local"
	logfile      = "/var/log/sharego.log"
	port         = "8002"
	buf          = Buffer
)

//Handle incoming connection, receive content send to host
func handleConnection(conn net.Conn) {
	//receive the message
	var text string
	cli := conn.RemoteAddr()
	log.Println("Receive message from", cli.String())
	err := gob.NewDecoder(conn).Decode(&text)
	if err != nil {
		log.Fatalln(err)
	} else {
		buf.set(text)
	}
	conn.Close()
}

//Listen for incoming connections to share clip content
func sharego_server() {
	log.Println("Receive Listening")
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		//handle error
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			//handle error
		}
		go handleConnection(conn)
	}
}

//connect to peers to send clip content
func sharego_client(hosts []string) {
	//Connect to hosts
	var host string
	for key := range hosts {
		host = hosts[key] + ":" + port
		log.Println("Connecting to host", host)
		c, err := net.Dial("tcp", host)
		if err != nil {
			log.Fatalln(err)
			return
		}
		//send the text
		text := get_clip_text()
		err = gob.NewDecoder(c).Encode(text)
		if err != nil {
			log.Fatalln(err)
		}
		//close connections
		c.Close()
	}
}

//set received text as clipboard content
//todo: move this to external package
func set_clip_text(text string) {
	cmd := exec.Command("xsel", "-b", "-i")
	cmd_stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalln(err)
		return
	}
	_, err := cmd_stdin.Write([]byte(text))
	if err != nil {
		log.Fatalln(err)
		return
	}
	err = cmd.Start()
	if err != nil {
		log.Fatalln(err)
		return
	}
	cmd_stdin.Close()
	if err != nil {
		log.Fatalln(err)
		return
	}
}

//Get the content of clipboard
//todo: move this to external package
func get_clip_text() string {
	//get clipboard data from xsel
	cmd := exec.Command("xsel", "-b", "-o")
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(out)
}

//Handle requests coming to the unix socket for the process
func handleReq(conn *net.UnixConn) {
	var buff [1024]byte
	n, err := conn.Read(buff[:])
	if err != nil {
		panic(err)
		return
	}
	input := string(buff[:n])
	//Check if input is to set or get
	input_str := string.Trim(input)
	input_str = string.Trim(input, " ")
	if strings.Contains(input_str, "get") {
		text := buf.get()
		log.Println("setting clipboard text")
		set_clip_text(text)
	} else if strings.Contains(input_str, "set") {
		hosts := strings.Fields(input)
		log.Println("Sending text to hosts", hosts[1:])
		sharego_client(hosts[1:])
	} else if strings.Contains(input, "stop") {
		sharego_destroy()
	} else {
		fmt.Println("Invalid argument")
		return
	}
}

//Listen for connetions from other instance of clipshare
func sharego_local() {
	//set up unix socket here
	l, err := net.ListenUnix("unix", &net.UnixAddr{unix_sock, "unix"})
	if err != nil {
		panic(err)
		return
	}
	defer os.Remove(unix_sock)
	for {
		conn, err := l.AcceptUnix()
		if err != nil {
			panic(err)
			return
		}
		go handleReq(conn)
	}
}

//Connect other instances of the process to the main process through unix socket
func connect_local_sock(args []string) {
	raddr := net.UnixAddr{"/tmp/sharego_local", "unix"}
	conn, err := net.DialUnix("unix", nil, &raddr)
	if err != nil {
		panic(err)
		return
	}
	var string_args string
	for key := range args {
		string_args = string_args + " " + args[key]
	}
	_, err = conn.Write([]byte(string_args))

  if err != nil {
    panic(err)
    return
  }
  conn.Close()
}

//Destroy main process
func sharego_destroy() {
  pidfile := pidfile_path + "/" + pidfile_name
  file, err := os.Open(pidfile)
  if err != nil {
    panic(err)
    return
  }
  defer file.Close()a
  //set the file size
  stat, err := file.Stat()
  if err != nil {
    log.Fatalln(err)
    return
  }

  //read the file
  bs := make([]byte, stat.Size())
  _, err = file.Read(bs)
  if err != nil {
    log.Fatalln(err)
    return
  }
  pid_str := string(bs)
  pid, err := strconv.ParseInt(pid_str, 10, 64)

  //remove pid file
  err = os.RemoveAll(pidfile_path)
  if err != nil {
    log.Fatalln(err)
    return
  }
  //remove unix socket
  err = os.Remove(unix_sock)
  if err != nil {
    log.Fatalln(err)
    return
  }

  //kill process
  err = syscall.Kill(int(pid), 15)
  if err != nil {
    log.Fatalln(err)
    return
  }
}

func create_pidfile() error {
  //Create directory for sharego and set path
  err := os.Mkdir(pidfile_path, 777)
  if err != nil {
    return err
  }
  pidfile.SetPidfilePath(pidfile_path + "/" + pidfile_name)
  err = pidfile.Write()
  if err != nil {
    return err
  }
  return nil
}

func enable_logging() error {
  f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
  if err != nil {
    return err
  }
  defer f.Close()
  //assign it to the standard logger
  log.SetOutput(f)
  return nil
}

//Start sharego process, remote and local server
func sharego_init() {
  err := enable_logging()
  if err != nil {
    panic(err)
    return
  }
  log.Println("Starting ShareGo...")
  err = create_pidfile()
  if err != nil {
    panic(err)
    return
  }
  //Listen for extenal connections parallely
  runtime.GOMAXPROCS(2)

  //Start listening to receive data from other peers
  go sharego_server()

  //Listen for local messages
  sharego_local()
}

//Check for already running process
func process_running() bool {
  //check if pidfile exists
  name := pidfile_path + "/" + pidfile_name
  _, err := os.Stat(name)
  if os.IsNotExist(err) {
    return false
  } else {
    return true
  }
}

func main() {
  if len(os.Args) == 1 {
    fmt.Printf("Usage: sharego start | get | set host(s) | stop ")
  } else {
    args := os.Args[1:]
    //check if sharego is already running
    if !process_running() {
      if args[0] == "start" {
        time.Sleep(5)
        sharego_init()
      } else {
        fmt.Println("ShareGo is not running")
        fmt.Println("ShareGo start to start sharego")
        return
      }
    } else {
      if args[0] == "start" {
        fmt.Println("ShareGo already running")
        return
      } else {
        //if args passed, send to open socket
        connect_local_sock(args)
      }
    }
  }
}
