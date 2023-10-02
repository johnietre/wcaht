package main

import (
	"bufio"
  "encoding/json"
	"flag"
	"fmt"
	"log"
  "net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	webs "golang.org/x/net/websocket"
	"wschat/wschat-go/common"
)

var (
	addr                  string
	numConns, msgsPerConn uint
	sameStart             bool
	test                  bool
	testTimeout           time.Duration

	startedChan, startChan = make(chan bool, 5), make(chan bool, 1)
	wg                     sync.WaitGroup

	testChan = make(chan *TestResults, 100)
)

func main() {
	log.SetFlags(0)

	flag.StringVar(&addr, "addr", "", "Address to connect to (with proto)")
	flag.UintVar(&numConns, "c", 1, "Number of connections")
	flag.UintVar(&msgsPerConn, "mpc", 1, "Number of messages to be sent from each connection")
	flag.BoolVar(
		&sameStart,
		"same-start",
		false,
		"Start all workers at the same time (after each has connected)",
	)
	timed := flag.Bool("time", false, "Time it")
	flag.BoolVar(
    &test, "test", false,
    "Run in test mode (messages from the server will be checked)",
  )
	flag.DurationVar(
    &testTimeout, "test-timeout", time.Minute,
    "Max duration to connect and read/write for",
  )
	flag.Parse()

	if addr == "" {
		log.Fatal("must provide address")
	}
	if _, err := url.Parse(addr); err != nil {
		log.Fatalf("bad address: %v", err)
	}

	if numConns == 0 {
		return
	}

	start := time.Now()
	for i := uint(0); i < numConns; i++ {
		wg.Add(1)
		if test {
			go runClientTest(i + 1)
		} else {
			go runClient(i + 1)
		}
	}
	if sameStart {
		for i := uint(0); i < numConns; i++ {
			<-startedChan
		}
		close(startChan)
	}
	if test {
		runTest(start)
		return
	}
	wg.Wait()
	if *timed {
		fmt.Printf("%f\n", time.Since(start).Seconds())
	}
	return
}

func runClient(id uint) {
	defer wg.Done()

	logFunc := func(format string, args ...any) {
		log.Printf(fmt.Sprintf("Worker #%d: %s", id, format), args...)
	}
	ws, err := webs.Dial(addr, "", "http://localhost")
	if sameStart {
		startedChan <- true
	}
	if err != nil {
		logFunc("error connecting: %v", err)
		return
	}
	defer ws.Close()

	if sameStart {
		_, _ = <-startChan
	}

	msg := common.Message{Action: common.ActionChat}
	contentsBuf := &strings.Builder{}
	for i := uint(0); i < msgsPerConn; i++ {
		fmt.Fprintf(contentsBuf, "Worker #%d: Message %d", id, i+1)
		msg.Contents = contentsBuf.String()
		if err := webs.JSON.Send(ws, msg); err != nil {
			logFunc("error sending message #%d: %v", i, err)
			return
		}
		contentsBuf.Reset()
	}
}

func runClientTest(id uint) {
	tres := newTestResults(id)
	defer func() {
		testChan <- tres
	}()

	config, err := webs.NewConfig(addr, "http://localhost")
	if err != nil {
		tres.connectErr = err
    startedChan <- true
		return
	}
  config.Dialer = &net.Dialer{Timeout: testTimeout}
	start := time.Now()
	ws, err := webs.DialConfig(config)
	tres.connectDur = time.Since(start)
	if sameStart {
		startedChan <- true
	}
	if err != nil {
		tres.connectErr = err
		return
	}
  tres.connected = true

	if sameStart {
		_, _ = <-startChan
	}

  // Get the UUID, this SHOULD be immediate/already here
  var msg common.Message
	ws.SetReadDeadline(time.Now().Add(testTimeout))
	if err := webs.JSON.Receive(ws, &msg); err != nil {
    tres.recvErr = fmt.Errorf("error receiving UUID: %v", err)
		return
	}
	if msg.Action != common.ActionConnect {
		tres.stopRecvReason = newUnexpectedMsgErr(common.ActionConnect, msg)
		return
	}
  uuid := msg.Contents
	tres.uuid = uuid

	doneChan := make(chan struct{})
	// NOTE: tres can't be accessed until after this is done (closes doneChan)
	go runRecvTest(ws, tres, doneChan)

	//msg := common.Message{Action: common.ActionChat}
  msg.Sender, msg.Action = uuid, common.ActionChat
	contentsBuf := &strings.Builder{}

	start = time.Now()
	ws.SetWriteDeadline(start.Add(testTimeout))

	// NOTE: Can't access tres since it's being used by runRecvTest
	// NOTE: Do this to limit number of heap derefs (vs using tres.msgsRecvd)?
	var msgsSent uint
	for ; msgsSent < msgsPerConn; msgsSent++ {
		fmt.Fprintf(contentsBuf, "Worker #%d: Message %d", id, msgsSent+1)
		msg.Contents = contentsBuf.String()
		if err = webs.JSON.Send(ws, msg); err != nil {
			break
		}
		contentsBuf.Reset()
	}
	sendDur := time.Since(start)

	_, _ = <-doneChan
	ws.Close()

	// NOTE: Useless check for disconnect based on the way the recv test is done
	/*
	  if tres.msgsRecvd == msgsPerConn && !tres.disconnected {
	    for {
	      if err := webs.JSON.Receive(ws, &msg); err != nil {
	        tres.disconnectErr = err
	        break
	      }
	      if msg.Action == common.ActionDisconnect && msg.Contents == tres.uuid {
	        tres.disconnected = true
	        break
	      }
	    }
	  }
	*/

	tres.msgsSent, tres.sendDur, tres.sendErr = msgsSent, sendDur, err
}

func runRecvTest(ws *webs.Conn, tres *TestResults, doneChan chan struct{}) {
	defer close(doneChan)

	start := time.Now()
	ws.SetReadDeadline(start.Add(testTimeout))

	msg := common.Message{}

	// NOTE: Do this to limit number of heap derefs (vs using tres.msgsRecvd)?
	var msgsRecvd uint
  msgStr := ""
MsgLoop:
	for msgsRecvd != msgsPerConn {
    /*
		if err := webs.JSON.Receive(ws, &msg); err != nil {
			tres.recvErr = err
			break
		}
    */
    if err := webs.Message.Receive(ws, &msgStr); err != nil {
      tres.recvErr = err
      break
    }
    if err := json.Unmarshal([]byte(msgStr), &msg); err != nil {
      tres.recvErr = fmt.Errorf("%w (msg: %s)", err, msgStr)
    }

		switch msg.Action {
		case common.ActionChat:
			if msg.Sender == tres.uuid {
				msgsRecvd++
			}
		case common.ActionDisconnect:
			if msg.Contents == tres.uuid {
				tres.stopRecvReason = newUnexpectedMsgErr(common.ActionChat, msg)
				break MsgLoop
			}
		case common.ActionError:
			tres.serverErr = msg.Contents
			break MsgLoop
		}
	}
	tres.msgsRecvd, tres.recvDur = msgsRecvd, time.Since(start)
}

func runTest(start time.Time) {
	var numPassed uint

	var passedConnectDurSum, passedSendDurSum, passedRecvDurSum float64
	var failedConnectDurSum, failedSendDurSum, failedRecvDurSum float64

	var minPassedConnectDur, minPassedSendDur, minPassedRecvDur = time.Hour, time.Hour, time.Hour
	var maxPassedConnectDur, maxPassedSendDur, maxPassedRecvDur time.Duration
	var maxFailedConnectDur, maxFailedSendDur, maxFailedRecvDur time.Duration

	var failedMsgsSentSum, failedMsgsRecvdSum uint
	var minMsgsSent, minMsgsRecvd = msgsPerConn, msgsPerConn
	var maxMsgsSent, maxMsgsRecvd uint

	var connectErrs, recvErrs, sendErrs []error
	var stopRecvReasons []error
	var serverErrs []string

	for i := uint(0); i < numConns; i++ {
    // TODO: Show progress?
		passed, tres := true, <-testChan
		if !tres.connected {
			passed = false

			failedConnectDurSum += tres.connectDur.Seconds()
			if tres.connectDur > maxFailedConnectDur {
				maxFailedConnectDur = tres.connectDur
			}

			connectErrs = append(connectErrs, tres.connectErr)
			continue
		} else {
			passedConnectDurSum += tres.connectDur.Seconds()
			if tres.connectDur < minPassedConnectDur {
				minPassedConnectDur = tres.connectDur
			}
			if tres.connectDur > maxPassedConnectDur {
				maxPassedConnectDur = tres.connectDur
			}
		}

		if tres.msgsSent != msgsPerConn {
			passed = false

			failedMsgsSentSum += tres.msgsSent
			if tres.msgsSent > maxMsgsSent {
				maxMsgsSent = tres.msgsSent
			}
			if tres.msgsSent < minMsgsSent {
				minMsgsSent = tres.msgsSent
			}

			failedSendDurSum += tres.sendDur.Seconds()
			if tres.sendDur > maxFailedSendDur {
				maxFailedSendDur = tres.sendDur
			}

			sendErrs = append(sendErrs, tres.sendErr)
		} else {
			passedSendDurSum += tres.sendDur.Seconds()
			if tres.sendDur < minPassedSendDur {
				minPassedSendDur = tres.sendDur
			}
			if tres.sendDur > maxPassedSendDur {
				maxPassedSendDur = tres.sendDur
			}
		}

		if tres.msgsRecvd != msgsPerConn {
			passed = false

			failedMsgsRecvdSum += tres.msgsRecvd
			if tres.msgsRecvd > maxMsgsRecvd {
				maxMsgsRecvd = tres.msgsRecvd
			}
			if tres.msgsRecvd < minMsgsRecvd {
				minMsgsRecvd = tres.msgsRecvd
			}

			failedRecvDurSum += tres.recvDur.Seconds()
			if tres.recvDur > maxFailedRecvDur {
				maxFailedSendDur = tres.recvDur
			}

			if tres.recvErr != nil {
				recvErrs = append(recvErrs, tres.recvErr)
			} else if tres.stopRecvReason != nil {
				stopRecvReasons = append(stopRecvReasons, tres.stopRecvReason)
			} else {
				serverErrs = append(serverErrs, tres.serverErr)
			}
		} else {
			passedRecvDurSum += tres.recvDur.Seconds()
			if tres.recvDur < minPassedRecvDur {
				minPassedRecvDur = tres.recvDur
			}
			if tres.recvDur > maxPassedRecvDur {
				maxPassedRecvDur = tres.recvDur
			}
		}

		if passed {
			numPassed++
		}
	}

	elapsed := time.Since(start)
	fNumConns, fNumPassed, numFailed := float64(numConns), float64(numPassed), numConns-numPassed
	numConnected := numConns - uint(len(connectErrs))
	fNumConnected := float64(numConnected)
	if numPassed != 0 {
		maxMsgsSent, maxMsgsRecvd = msgsPerConn, msgsPerConn
	}

  // Declarations to satisfy goto
  var numPassedSend, numPassedRecv, numFailedRecv uint

	fmt.Printf("Total test time: %f secs\n", elapsed.Seconds())

	fmt.Printf(
    "%d passed, %d failed, %d total (%.2f%% passed)\n",
    numPassed, numFailed, numConns, fNumPassed/fNumConns*100.0,
  )
	fmt.Println()

	// Connect
	fmt.Printf(
		"Average time to connect (total): %f secs\n",
		(passedConnectDurSum+failedConnectDurSum)/fNumConns,
	)
	if numConnected != 0 {
		fmt.Printf(
			"\tMin, Average, Max time to connect (passed): %f, %f, %f secs\n",
			minPassedConnectDur.Seconds(), passedConnectDurSum/fNumConnected, maxPassedConnectDur.Seconds(),
		)
	}
	if l := len(connectErrs); l != 0 {
		failed := float64(l)
		fmt.Printf(
			"\tAverage, Max time to attempt connect (failed): %f, %f secs\n",
			failedConnectDurSum/failed, maxFailedConnectDur.Seconds(),
		)
		// TODO: loop?
	}
	fmt.Println()
	if numConnected == 0 {
    goto ErrorsMenu
	}

  numPassedSend = numConns - uint(len(sendErrs))

	// Send
	fmt.Printf(
		"Average time to send msgs (total): %f secs\n",
		(passedSendDurSum+failedSendDurSum)/fNumConns,
	)
	if numPassedSend != 0 {
    fNumPassedSend := float64(numPassedSend)
		fmt.Printf(
			"\tMin, Average, Max time to send msgs (passed): %f, %f, %f secs\n",
			minPassedSendDur.Seconds(), passedSendDurSum/fNumPassedSend, maxPassedSendDur.Seconds(),
		)
	}
	if l := len(sendErrs); l != 0 {
		failed := float64(l)
		fmt.Printf(
			"\tAverage, Max time to send msgs (failed): %f, %f secs\n",
			failedSendDurSum/failed, maxFailedSendDur.Seconds(),
		)
		fmt.Printf(
			"\t\tMin, Average, Max number of messages sent (failed): %d, %.3f, %d msgs\n",
			minMsgsSent, float64(failedMsgsSentSum)/failed, maxMsgsSent,
		)
		// TODO: loop?
	}
	fmt.Println()

  numFailedRecv = uint(len(recvErrs) + len(stopRecvReasons) + len(serverErrs))
  numPassedRecv = numConns - numFailedRecv

	// Recv
	fmt.Printf(
		"Average time to receive msgs (total): %f secs\n",
		(passedRecvDurSum+failedRecvDurSum)/fNumConnected,
	)
	if numPassedRecv != 0 {
    fNumPassedRecv := float64(numPassedRecv)
		fmt.Printf(
			"\tMin, Average, Max time to receive msgs (passed): %f, %f, %f secs\n",
			minPassedRecvDur.Seconds(), passedRecvDurSum/fNumPassedRecv, maxPassedRecvDur.Seconds(),
		)
	}
	if numFailedRecv != 0 {
		failed := float64(numFailedRecv)
		fmt.Printf(
			"\tAverage, Max time to receive msgs (failed): %f, %f secs\n",
			failedRecvDurSum/failed, maxFailedRecvDur.Seconds(),
		)
		fmt.Printf(
			"\t\tMin, Average, Max number of messages received (failed): %d, %.3f, %d msgs\n",
			minMsgsRecvd, float64(failedMsgsRecvdSum)/failed, maxMsgsRecvd,
		)
		// TODO: loop (recvErrs, stopRecvReasons, serverErrs)?
	}
	fmt.Println()

	if numFailed == 0 {
    return
  }
ErrorsMenu:
  prompt := "Go through errors? [Y/n] "
  if conf := strings.ToLower(readline(prompt)); conf != "y" && conf != "yes" {
    return
  }
  for {
    choices := make(map[int]func(), 5)
    fmt.Println("Choose error type to view")
    if l := len(connectErrs); l != 0 {
      fmt.Printf("1) Connect Error (%d error(s))\n", l)
      choices[1] = func() {
        fmt.Printf("Connect error: %v\n", connectErrs[0])
        connectErrs = connectErrs[1:]
      }
    }
    if l := len(sendErrs); l != 0 {
      fmt.Printf("2) Send Error (%d error(s))\n", l)
      choices[2] = func() {
        fmt.Printf("Send error: %v\n", sendErrs[0])
        sendErrs = sendErrs[1:]
      }
    }
    if l := len(recvErrs); l != 0 {
      fmt.Printf("3) Receive Error (%d error(s))\n", l)
      choices[3] = func() {
        fmt.Printf("Receive error: %v\n", recvErrs[0])
        recvErrs = recvErrs[1:]
      }
    }
    if l := len(stopRecvReasons); l != 0 {
      fmt.Printf("4) Stop Receive Reason (%d reason(s))\n", l)
      choices[4] = func() {
        fmt.Printf("Stopped receive reasons: %v\n", stopRecvReasons[0])
        stopRecvReasons = stopRecvReasons[1:]
      }
    }
    if l := len(serverErrs); l != 0 {
      fmt.Printf("5) Server Error (%d error(s))\n", l)
      choices[5] = func() {
        fmt.Printf("Server error: %s\n", serverErrs[0])
        serverErrs = serverErrs[1:]
      }
    }
    if len(choices) == 0 {
      return
    }
    fmt.Println("0) Exit")
    //fmt.Print("Choice: ")
    choice := readlineWith("Choice: ", func(s string) (int, bool) {
      i, err := strconv.Atoi(s)
      if err != nil {
        return 0, false
      }
      _, ok := choices[i]
      return i, ok || i == 0
    })
    if choice == 0 {
      return
    }
    choices[choice]()
  }
}

type TestResults struct {
	id   uint
	uuid string

	connectDur time.Duration
	sendDur    time.Duration
	recvDur    time.Duration

	connected bool
	//disconnected bool

	msgsSent uint
	// The number of messages the specific client sent that were echoed by the
	// server (i.e., that were sent out by the server and broadcast to all).
	msgsRecvd uint

	// Error in connecting
	connectErr error
	// The first error encountered running test (sending)
	sendErr error
	// The first error encountered running test (receiving)
	recvErr error
	// Error in waiting for disconnect
	//disconnectErr error

	// Reason for stopping receiving early
	stopRecvReason error
	// Error sent by the server
	serverErr string
}

func newTestResults(id uint) *TestResults {
	return &TestResults{id: id}
}

type UnexpectedMsgError struct {
	expected common.Action
	msg      common.Message
}

func newUnexpectedMsgErr(expected common.Action, msg common.Message) *UnexpectedMsgError {
	return &UnexpectedMsgError{
		expected: expected,
		msg:      msg,
	}
}

func (ume *UnexpectedMsgError) Error() string {
	return fmt.Sprintf(`expected "connect", got %q`, ume.msg.Action)
}

var stdinReader = bufio.NewReader(os.Stdin)

func readline(prompt string) string {
  fmt.Print(prompt)
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		log.Fatalf("error reading stdin: %v", err)
	}
	return strings.TrimSpace(line)
}

func readlineWith[T any](prompt string, f func(string) (T, bool)) T {
	for {
		val, ok := f(readline(prompt))
		if ok {
			return val
		}
	}
}
