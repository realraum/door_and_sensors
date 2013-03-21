package main
import "fmt"
import "net"
import "bufio"
import "strings"
import "os"
import "io"
import "svn.spreadspace.org/realraum.svn/go/termios"
import "flag"
import "regexp"
import "sync"
import "time"

var lock sync.RWMutex

type DoorConnection struct {
  rw * bufio.ReadWriter
  c net.Conn
  wchan chan string
}

var DoorConnectionMap = map[uint]DoorConnection {
}

var cmdHandler = map[string]func([]string,string,*bufio.ReadWriter ) {
  "test":handleCmdTest,
  "open":handleCmdController,
  "close":handleCmdController,
  "toggle":handleCmdController,
}


func readLineSafe(rw *bufio.Reader) (string, error) {
  wasPrefix:=false
  var line string
  for isPrefix:=true;isPrefix; {
    var lineBuf []byte 
    var err error
    lineBuf,isPrefix,err = rw.ReadLine()
    if err != nil {
        return "",err
    }
    if isPrefix {
      wasPrefix=true
    } else {
      line=string(lineBuf)
    }
  }
  if wasPrefix {
      fmt.Println("line too long")
      //fmt.Fprintf(rw,"line too long\n")
      //rw.Flush()
      return "",nil
  }
  reg := regexp.MustCompile("\r")
  safe := reg.ReplaceAllString(line, "")
  return safe,nil
}

func connToReadWriter(c io.Reader,cw io.Writer) (*bufio.ReadWriter) {
    client_r := bufio.NewReaderSize(c,1024)
    client_w := bufio.NewWriterSize(cw,1024)
    return bufio.NewReadWriter(client_r,client_w)
}

func handleConnection(c net.Conn, client * bufio.ReadWriter, connID uint) () {
    fmt.Println("new connection")
    defer func () {
     lock.Lock()
     delete(DoorConnectionMap,connID)
     lock.Unlock()
    }()
    for {
         line,err:=readLineSafe(bufio.NewReader(client))
         if err != nil {
          if err.Error() != "EOF" {
            fmt.Printf("Error: readLineSafe returned %v\n",err.Error())
          } else {
            fmt.Printf("Connection closed by remote host\n");
          }
          c.Close()
          return
         }
         if line == "" {
           continue
         }
         fmt.Printf("Received: %v\n", line)
         tokens:=strings.Fields(line)
         remainStr:=strings.Join(tokens[1:]," ")
         handleCmd(tokens,remainStr,client)
    }
}

func handleCmd(tokens []string, remainStr string,client * bufio.ReadWriter) {
  cmd:=tokens[0]
  func_ptr,present := cmdHandler[cmd]
  if present {
    func_ptr(tokens, remainStr,client)
  } else {
    fmt.Printf("Error: unknown Cmd: %v\n", cmd)
  }
}

func handleCmdTest(tokens []string, remainStr string, client * bufio.ReadWriter) {
  //cmd:=tokens[0]
  fmt.Printf("Test: %v\n", remainStr)
}

func handleCmdController(tokens []string, remainStr string, client * bufio.ReadWriter) {
  cmd:=tokens[0]
  s_r:=strings.NewReader(cmd)
  char := make([]byte,1)
  s_r.Read(char)
  fmt.Println(string(char))
}


func openTTY(name string) *os.File {
  file, err := os.OpenFile(name,os.O_RDWR  ,0600) // For read access.
  if err != nil {
    fmt.Println(err.Error())
  }
  termios.Ttyfd(file.Fd())
  termios.SetRaw()
  return file 
}
func usage() {
    fmt.Fprintf(os.Stderr, "usage: myprog [inputfile]\n")
    flag.PrintDefaults()
    os.Exit(2)
}

func SerialWriter(c chan string, serial * os.File ) {
  for {
    serial.WriteString(<-c)
    serial.Sync()
  }
}

func SerialReader(c chan string , serial * bufio.Reader) {
  for {
    s,err := readLineSafe(serial)
    if (s=="") {
     continue
    }
    if (err!=nil) {
     fmt.Printf("Error in read from serial: %v\n",err.Error())
     os.Exit(1)
    }
    //fmt.Printf("Serial: Read %v\n",s);
    c<-s
  }
}

func openSerial(filename string) (chan string,chan string) {
  serial:=openTTY(filename)
  in:=make(chan string)
  out:=make(chan string)
  go SerialWriter(out,serial)
  go SerialReader(in,bufio.NewReaderSize(serial,128))
  return in,out
}

func SerialHandler(serial_i chan string) {
  for {
    line:=<-serial_i
    fmt.Printf("Serial Read: %s\n",line)
    lock.RLock()
    for _, v := range DoorConnectionMap{
      select {
        case v.wchan<-line:

        case <-time.After(2):

      }
    }
    lock.RUnlock()
  }
}

func chanWriter(c chan string, w io.Writer) {
  for {
    line := <-c
     w.Write([]byte(line))
     w.Write([]byte("\n"))
  }
}
func main() {
//    lock = make(sync.RWMutex)
    flag.Usage = usage
    flag.Parse()

    args := flag.Args()
    if len(args) < 1 {
        fmt.Println("Input file is missing.");
        os.Exit(1);
    }
  serial_i,serial_o:=openSerial(args[0]) 
  go SerialHandler(serial_i)
  serial_o<-"f"

  ln, err := net.Listen("unix", "/tmp/test.sock")
  if err != nil {
    fmt.Printf("Error: %s\n",err.Error())
    return
  }
  fmt.Printf("Listener started\n")

  var connectionID uint =0
  for {
    conn, err := ln.Accept()
    if err != nil {
      // handle error
     continue
    }
   client:=connToReadWriter(conn,conn)
   connectionID++
   writeChan := make(chan string)
   lock.Lock()
   DoorConnectionMap[connectionID]= DoorConnection{ rw:client,c:conn, wchan:writeChan }
   lock.Unlock()
   go handleConnection(conn,client,connectionID)
   go chanWriter(writeChan,conn)
  }
}
