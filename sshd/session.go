package sshd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/armon/go-radix"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type session struct {
	l        *logrus.Entry
	c        *ssh.ServerConn
	term     *terminal.Terminal
	commands *radix.Tree
	exitChan chan bool
}

func NewSession(commands *radix.Tree, conn *ssh.ServerConn, chans <-chan ssh.NewChannel, l *logrus.Entry) *session {
	s := &session{
		commands: radix.NewFromMap(commands.ToMap()),
		l:        l,
		c:        conn,
		exitChan: make(chan bool),
	}

	s.commands.Insert("logout", &Command{
		Name:             "logout",
		ShortDescription: "Ends the current session",
		Callback: func(a interface{}, args []string, w StringWriter) error {
			s.Close()
			return nil
		},
	})

	go s.handleChannels(chans)
	return s
}

func (s *session) handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			s.l.WithField("sshChannelType", newChannel.ChannelType()).Error("unknown channel type")
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			s.l.WithError(err).Warn("could not accept channel")
			continue
		}

		go s.handleRequests(requests, channel)
	}
}

func (s *session) handleRequests(in <-chan *ssh.Request, channel ssh.Channel) {
	for req := range in {
		var err error
		//TODO: maybe support window sizing?
		switch req.Type {
		case "shell":
			if s.term == nil {
				s.term = s.createTerm(channel)
				err = req.Reply(true, nil)
			} else {
				err = req.Reply(false, nil)
			}

		case "pty-req":
			err = req.Reply(true, nil)

		case "window-change":
			err = req.Reply(true, nil)

		case "exec":
			var payload = struct{ Value string }{}
			cErr := ssh.Unmarshal(req.Payload, &payload)
			if cErr != nil {
				req.Reply(false, nil)
				return
			}

			req.Reply(true, nil)
			s.dispatchCommand(payload.Value, &stringWriter{channel})

			//TODO: Fix error handling and report the proper status back
			status := struct{ Status uint32 }{uint32(0)}
			//TODO: I think this is how we shut down a shell as well?
			channel.SendRequest("exit-status", false, ssh.Marshal(status))
			channel.Close()
			return

		default:
			s.l.WithField("sshRequest", req.Type).Debug("Rejected unknown request")
			err = req.Reply(false, nil)
		}

		if err != nil {
			s.l.WithError(err).Info("Error handling ssh session requests")
			s.Close()
			return
		}
	}
}

func (s *session) createTerm(channel ssh.Channel) *terminal.Terminal {
	//TODO: PS1 with nebula cert name
	term := terminal.NewTerminal(channel, s.c.User()+"@nebula > ")
	term.AutoCompleteCallback = func(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
		// key 9 is tab
		if key == 9 {
			cmds := matchCommand(s.commands, line)
			if len(cmds) == 1 {
				return cmds[0] + " ", len(cmds[0]) + 1, true
			}

			sort.Strings(cmds)
			term.Write([]byte(strings.Join(cmds, "\n") + "\n\n"))
		}

		return "", 0, false
	}

	go s.handleInput(channel)
	return term
}

func (s *session) handleInput(channel ssh.Channel) {
	defer s.Close()
	w := &stringWriter{w: s.term}
	for {
		line, err := s.term.ReadLine()
		if err != nil {
			//TODO: log
			break
		}

		s.dispatchCommand(line, w)
	}
}

func (s *session) dispatchCommand(line string, w StringWriter) {
	args, err := shlex.Split(line, true)
	if err != nil {
		//todo: LOG IT
		return
	}

	if len(args) == 0 {
		dumpCommands(s.commands, w)
		return
	}

	c, err := lookupCommand(s.commands, args[0])
	if err != nil {
		//TODO: handle the error
		return
	}

	if c == nil {
		err := w.WriteLine(fmt.Sprintf("did not understand: %s", line))
		//TODO: log error
		_ = err

		dumpCommands(s.commands, w)
		return
	}

	if checkHelpArgs(args) {
		s.dispatchCommand(fmt.Sprintf("%s %s", "help", c.Name), w)
		return
	}

	err = execCommand(c, args[1:], w)
	if err != nil {
		//TODO: log the error
	}
	return
}

func (s *session) Close() {
	s.c.Close()
	s.exitChan <- true
}
