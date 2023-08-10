package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/gocarina/gocsv"
)

var delay = 0

type Mail struct {
	Sender  string
	To      string
	Subject string
	Body    bytes.Buffer
}

type User struct {
	FirstName string `csv:"first_name"`
	LastName  string `csv:"last_name"`
	Email     string `csv:"email"`
	Subject   string
	From      string
	Url       string
	Date      string
}

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unknown from server")
		}
	}
	return nil, nil
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

// validateLine checks to see if a line has CR or LF as per RFC 5321
func validateLine(line string) error {
	if strings.ContainsAny(line, "\n\r") {
		return errors.New("smtp: A line must not contain CR or LF")
	}
	return nil
}

var testHookStartTLS func(*tls.Config) // nil, except for tests

func SendMail(c *smtp.Client, a smtp.Auth, from string, to []string, msg []byte) error {
	var err error
	if err = validateLine(from); err != nil {
		return err
	}
	for _, recp := range to {
		if err := validateLine(recp); err != nil {
			return err
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return err
}

func main() {

	var host, username, password, toFile, emailTemplateFile, url, sender, subject string
	var port int
	var debug, ssl bool
	flag.StringVar(&host, "host", "", "the smtp host")
	flag.IntVar(&port, "port", 25, "the smtp port")

	flag.StringVar(&username, "username", "", "the smtp username")
	flag.StringVar(&password, "password", "", "the smtp password")
	flag.StringVar(&toFile, "f", "", "csv file containing list of 'first_name,last_name,email' with header row (check example.csv)")
	flag.StringVar(&emailTemplateFile, "t", "", "Email template file")
	flag.StringVar(&sender, "sender", "Test Person <test@example.com>", "the from address")
	flag.StringVar(&subject, "subject", "Test mail", "the email subject line")
	flag.BoolVar(&debug, "debug", false, "enable verbosity")
	flag.StringVar(&url, "url", "", "The shonky URL to direct users to")
	flag.BoolVar(&ssl, "ssl", false, "Force use of SSL (typically on port 465)")
	flag.IntVar(&delay, "delay", 0, "Delay between sending emails in ms (rate limit requests)")

	flag.Parse()
	currTime := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	server := fmt.Sprintf("%s:%d", host, port)
	if debug {
		log.Printf("Attempting to connect to SMTP server: %s:%d", host, port)
	}
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	// connect with a timeout of 10secs
	var c *smtp.Client
	conn, err := net.DialTimeout("tcp", server, time.Second*10)
	if err != nil {
		log.Fatalf("Error (DialTimeout) - %v\n", err)
	}
	if ssl {
		tlscon := tls.Client(conn, tlsconfig)
		err = tlscon.Handshake()
		if err != nil {
			log.Fatalf("Error (tls Client) - %v\n", err)
		}
		c, err = smtp.NewClient(tlscon, host)

	} else {
		c, err = smtp.NewClient(conn, host)
	}

	if err != nil {
		log.Fatalf("Error (smtp.NewClient) - %v\n", err)
	}

	defer c.Close()
	if err = c.Hello("localhost"); err != nil {
		log.Printf("Error (c.Hello) - %v\n", err)
	}
	if !ssl {
		if err = c.StartTLS(tlsconfig); err != nil {
			log.Fatalf("Error (c.StartTLS) - %v\n", err)
		}
	}

	var auth smtp.Auth
	if username == "" {
		if debug {
			log.Printf("No credentials supplied, skipping auth.")
		}
		auth = nil
	} else {
		if debug {
			log.Printf("Attempting to auth to SMTP server using Username: %s and Password: %s******", username, password[:3])
		}
		auth = LoginAuth(username, password)

		if err = c.Auth(auth); err != nil {
			log.Fatalf("Error - %v\n", err)
		}
	}

	if debug {
		log.Printf("Reading email addresses from: %s", toFile)
	}
	in, err := os.Open(toFile)
	if err != nil {
		log.Fatalf("Error - %v\n", err)
	}
	defer in.Close()

	users := []*User{}

	if err := gocsv.UnmarshalFile(in, &users); err != nil {
		log.Fatalf("Error - %v\n", err)
	}
	if debug {
		log.Printf("Reading email template: %s", emailTemplateFile)
	}
	templateData, err := os.ReadFile(emailTemplateFile)
	// log.Println(users)
	// log.Println(templateData)
	if err != nil {
		log.Fatalf("Error - %v\n", err)
	}

	re := regexp.MustCompile(`<(.*@.*)>`)
	var emails, errors int
	if debug {
		log.Printf("Preparing and sending emails")
	}
	for index, user := range users {
		user.Subject = subject
		user.From = sender
		user.FirstName = strings.Title(user.FirstName)
		user.LastName = strings.Title(user.LastName)
		user.Url = url
		user.Date = currTime

		t := template.Must(template.New("template_data").Parse(string(templateData)))
		var body bytes.Buffer

		err := t.Execute(&body, user)
		if err != nil {
			errors++
			log.Fatalf("Error - unable to something the email or whatever: %v\n", err)
		}
		msg := body.Bytes()

		from := sender
		// Mailbox format in use for from line!
		if strings.Contains(sender, "<") {
			from = re.FindStringSubmatch(sender)[1]

		}
		if debug {
			log.Printf("Sending Email (%d / %d):\n=============\n%s\n=============\nFrom: %s, using: un: %s pw: %s host: %s\n", index+1, len(users), string(msg[:300]), from, username, password, server)
		}
		err = SendMail(c, auth, from, []string{user.Email}, msg)
		time.Sleep(time.Duration(delay) * time.Millisecond)

		if err != nil {
			log.Printf("Error - unable to send the thing to <%s> i think or whatever: %v\n", user.Email, err)
			errors++
			time.Sleep(time.Duration(delay) * time.Millisecond)

			continue
		}

		emails++
		log.Printf("Email to <%s> sent successfully at %s\n", user.Email, time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"))

	}
	c.Quit()
	log.Printf("%d - Emails sent successfully\n (%d errors)", emails, errors)
}
