/* TODO
override the AddCommand() and HandleCommand() functions to accept a function pointer with parameters *ChatBot as well as string
*/

package twc

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

const addr string = "irc.chat.twitch.tv:6667"

type Command struct {
	name string
	fp   func(string)
}

type ChatBot struct {
	channel  string
	conn     *net.Conn
	reader   *bufio.Reader
	commands []*Command
	firstcmd bool
	handling bool
}

func NewChatBot(username string, oauth string, channel string) (*ChatBot, error) {
	ch := new(ChatBot)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return ch, err
	}
	reader := bufio.NewReader(conn)

	fmt.Fprintf(conn, "PASS %s\r\n", oauth)
	fmt.Fprintf(conn, "NICK %s\r\n", username)
	fmt.Fprintf(conn, "JOIN #%s\r\n", channel)

	ch.reader = reader
	ch.channel = channel
	ch.conn = &conn
	ch.firstcmd = false
	ch.handling = false

	return ch, nil
}

func (c *ChatBot) OnMessage(fp func(string)) {
	msg, err := c.reader.ReadString('\r')
	if !IsMessage(msg) {
		return
	}
	if err != nil {
		fmt.Println(err)
	}
	fp(msg)
}

func (c *ChatBot) OnCommand(command string, fp func(string)) {
	msg, err := c.reader.ReadString('\r')
	if err != nil {
		fmt.Println(err)
	}
	if strings.Contains(msg, command) {
		fp(msg)
	}
}

func (c *ChatBot) SendMessage(message string) {
	fmt.Fprintf(*c.conn, "PRIVMSG #%s :%s\r\n", c.channel, message)
}

// Used alongside HandleAllCommands()
func (c *ChatBot) AddCommand(name string, fp func(string)) {
	cmd := new(Command)
	cmd.name = name
	cmd.fp = fp
	c.firstcmd = true
	c.commands = append(c.commands, cmd)
}

// User does not need to call HandleAllCommands() as this function starts a go routine for them
// if this function is called after the first one, it effectively becomes AddCommand(), and the user can do that too.
func (c *ChatBot) HandleCommand(name string, fp func(string)) {
	if !c.firstcmd && !c.handling {
		log.Println("Started handle routine...")
		go c.HandleAllCommands()
	}
	c.AddCommand(name, fp)
}

func (c *ChatBot) HandleAllCommands() {
	if c.handling { // dont start two go routines
		return
	}
	c.handling = true
	for { // will block unless called from go routine
		msg, err := c.reader.ReadString('\r')
		if err != nil {
			fmt.Println(err)
		}
		for _, cmd := range c.commands {
			if strings.Contains(msg, cmd.name) {
				cmd.fp(msg)
			}
		}
	}
}

func ParseMessage(raw string) string {
	userIndex := strings.Index(raw, "!")
	if userIndex == -1 {
		return ""
	}
	username := raw[2:userIndex]

	msgSubstr := raw[userIndex:]
	msgIndex := strings.Index(msgSubstr, ":") + 1
	if msgIndex == -1 {
		return ""
	}
	endIndex := strings.Index(msgSubstr, "\r")
	if endIndex == -1 {
		return ""
	}
	message := msgSubstr[msgIndex:endIndex]

	return fmt.Sprintf("%s: %s", username, message)
}

func IsMessage(raw string) bool {
	return strings.Contains(raw, "PRIVMSG")
}

func GetCommandArgs(name string, rawInput string) []string {
	end := len(name) + 1
	cmdIndex := strings.Index(rawInput, name) + end
	if cmdIndex == len(rawInput) {
		return nil
	}
	return strings.Split(rawInput[cmdIndex:len(rawInput)-1], " ")
}
