package execution

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strings"
	"time"

	"ok/internal/config"
	"ok/internal/logger"
)

// EmailTool provides email read and send capabilities via IMAP/SMTP.
type EmailTool struct {
	cfg config.EmailConfig
}

// NewEmailTool creates a new EmailTool with the given config.
func NewEmailTool(cfg config.EmailConfig) *EmailTool {
	return &EmailTool{cfg: cfg}
}

func (t *EmailTool) Name() string        { return "email" }
func (t *EmailTool) Description() string {
	return "Read and send emails via IMAP/SMTP. Actions: read (list recent inbox), send (send email), search (search inbox)."
}

func (t *EmailTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"read", "send", "search"},
				"description": "Action to perform: 'read' lists recent emails, 'send' sends an email, 'search' searches inbox",
			},
			// read/search params
			"folder": map[string]any{
				"type":        "string",
				"description": "Mailbox folder to use (default: INBOX)",
			},
			"count": map[string]any{
				"type":        "integer",
				"description": "Number of emails to fetch for 'read' (default: 10)",
				"minimum":     1.0,
				"maximum":     50.0,
			},
			"query": map[string]any{
				"type":        "string",
				"description": "IMAP search query for 'search' action (e.g. 'FROM boss@example.com', 'SUBJECT invoice', 'UNSEEN')",
			},
			// send params
			"to": map[string]any{
				"type":        "string",
				"description": "Recipient email address(es), comma-separated",
			},
			"subject": map[string]any{
				"type":        "string",
				"description": "Email subject",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Email body (plain text)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *EmailTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)

	switch action {
	case "read":
		folder := "INBOX"
		if f, ok := args["folder"].(string); ok && f != "" {
			folder = f
		}
		count := t.cfg.MaxFetch
		if count <= 0 {
			count = 10
		}
		if c, ok := args["count"].(float64); ok && int(c) > 0 {
			count = int(c)
		}
		return t.readEmails(ctx, folder, "ALL", count)

	case "search":
		folder := "INBOX"
		if f, ok := args["folder"].(string); ok && f != "" {
			folder = f
		}
		query, _ := args["query"].(string)
		if query == "" {
			query = "UNSEEN"
		}
		count := t.cfg.MaxFetch
		if count <= 0 {
			count = 10
		}
		if c, ok := args["count"].(float64); ok && int(c) > 0 {
			count = int(c)
		}
		return t.readEmails(ctx, folder, query, count)

	case "send":
		to, _ := args["to"].(string)
		subject, _ := args["subject"].(string)
		body, _ := args["body"].(string)
		if to == "" {
			return ErrorResult("'to' is required for send action")
		}
		if subject == "" {
			return ErrorResult("'subject' is required for send action")
		}
		return t.sendEmail(ctx, to, subject, body)

	default:
		return ErrorResult(fmt.Sprintf("unknown action %q, must be read, search, or send", action))
	}
}

// --- IMAP client ---

type imapClient struct {
	conn net.Conn
	br   *bufio.Reader
	seq  int
}

func dialIMAP(host string, port int, useTLS bool) (*imapClient, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	var conn net.Conn
	var err error

	if useTLS {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	} else {
		conn, err = net.DialTimeout("tcp", addr, 15*time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("IMAP connect to %s: %w", addr, err)
	}

	c := &imapClient{conn: conn, br: bufio.NewReader(conn)}
	// Read server greeting
	if _, err := c.br.ReadString('\n'); err != nil {
		conn.Close()
		return nil, fmt.Errorf("IMAP greeting: %w", err)
	}
	return c, nil
}

func (c *imapClient) close() { c.conn.Close() }

// cmd sends a tagged IMAP command and reads all response lines until the tagged OK/NO/BAD.
func (c *imapClient) cmd(format string, args ...any) ([]string, error) {
	c.seq++
	tag := fmt.Sprintf("A%03d", c.seq)
	line := fmt.Sprintf("%s %s\r\n", tag, fmt.Sprintf(format, args...))
	if _, err := io.WriteString(c.conn, line); err != nil {
		return nil, fmt.Errorf("IMAP write: %w", err)
	}

	var lines []string
	for {
		resp, err := c.br.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("IMAP read: %w", err)
		}
		resp = strings.TrimRight(resp, "\r\n")
		if strings.HasPrefix(resp, tag+" ") {
			status := strings.SplitN(resp[len(tag)+1:], " ", 2)[0]
			if status == "OK" {
				return lines, nil
			}
			return lines, fmt.Errorf("IMAP %s: %s", status, resp)
		}
		lines = append(lines, resp)
	}
}

func (c *imapClient) login(user, pass string) error {
	_, err := c.cmd("LOGIN %q %q", user, pass)
	return err
}

func (c *imapClient) selectFolder(folder string) error {
	_, err := c.cmd("SELECT %q", folder)
	return err
}

func (c *imapClient) logout() {
	_, _ = c.cmd("LOGOUT")
	c.conn.Close()
}

// search runs an IMAP SEARCH command and returns message sequence numbers.
func (c *imapClient) search(criteria string) ([]string, error) {
	lines, err := c.cmd("SEARCH %s", criteria)
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		if strings.HasPrefix(l, "* SEARCH") {
			parts := strings.Fields(l)
			if len(parts) > 2 {
				return parts[2:], nil
			}
			return nil, nil
		}
	}
	return nil, nil
}

type emailMessage struct {
	UID     string
	From    string
	To      string
	Subject string
	Date    string
	Snippet string
}

// fetchHeaders fetches FROM, SUBJECT, DATE headers for given sequence numbers.
func (c *imapClient) fetchHeaders(seqNums []string) ([]emailMessage, error) {
	if len(seqNums) == 0 {
		return nil, nil
	}
	set := strings.Join(seqNums, ",")
	lines, err := c.cmd("FETCH %s (RFC822.SIZE BODY.PEEK[HEADER.FIELDS (FROM TO SUBJECT DATE)])", set)
	if err != nil {
		return nil, err
	}

	var msgs []emailMessage
	var cur *emailMessage
	for _, line := range lines {
		if strings.HasPrefix(line, "* ") && strings.Contains(line, "FETCH") {
			parts := strings.Fields(line)
			uid := parts[1]
			msgs = append(msgs, emailMessage{UID: uid})
			cur = &msgs[len(msgs)-1]
			continue
		}
		if cur == nil {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "from:") {
			cur.From = strings.TrimSpace(line[5:])
		} else if strings.HasPrefix(lower, "to:") {
			cur.To = strings.TrimSpace(line[3:])
		} else if strings.HasPrefix(lower, "subject:") {
			cur.Subject = strings.TrimSpace(line[8:])
		} else if strings.HasPrefix(lower, "date:") {
			cur.Date = strings.TrimSpace(line[5:])
		}
	}
	return msgs, nil
}

func (t *EmailTool) readEmails(ctx context.Context, folder, criteria string, count int) *ToolResult {
	port := t.cfg.IMAPPort
	if port == 0 {
		if t.cfg.IMAPTLS {
			port = 993
		} else {
			port = 143
		}
	}

	c, err := dialIMAP(t.cfg.IMAPHost, port, t.cfg.IMAPTLS)
	if err != nil {
		return ErrorResult(fmt.Sprintf("IMAP connect failed: %v", err))
	}
	defer c.logout()

	if err := c.login(t.cfg.Username, t.cfg.Password); err != nil {
		return ErrorResult(fmt.Sprintf("IMAP login failed: %v", err))
	}

	if err := c.selectFolder(folder); err != nil {
		return ErrorResult(fmt.Sprintf("IMAP SELECT %s failed: %v", folder, err))
	}

	seqNums, err := c.search(criteria)
	if err != nil {
		return ErrorResult(fmt.Sprintf("IMAP search failed: %v", err))
	}

	if len(seqNums) == 0 {
		result := fmt.Sprintf("No messages found in %s matching: %s", folder, criteria)
		return NewToolResult(result)
	}

	// Take the last 'count' messages (most recent)
	if len(seqNums) > count {
		seqNums = seqNums[len(seqNums)-count:]
	}

	msgs, err := c.fetchHeaders(seqNums)
	if err != nil {
		logger.WarnCF("email", "Failed to fetch email headers", map[string]any{"error": err.Error()})
		return ErrorResult(fmt.Sprintf("IMAP fetch failed: %v", err))
	}

	out, _ := json.MarshalIndent(msgs, "", "  ")
	summary := fmt.Sprintf("Found %d messages in %s:\n%s", len(msgs), folder, string(out))
	return NewToolResult(summary)
}

// --- SMTP ---

func (t *EmailTool) sendEmail(ctx context.Context, to, subject, body string) *ToolResult {
	host := t.cfg.SMTPHost
	port := t.cfg.SMTPPort
	if port == 0 {
		port = 587
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	from := t.cfg.Username
	fromName := t.cfg.FromName
	if fromName == "" {
		fromName = from
	}

	toAddrs := strings.Split(to, ",")
	for i := range toAddrs {
		toAddrs[i] = strings.TrimSpace(toAddrs[i])
	}

	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		fromName, from, to, subject, time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"), body)

	auth := smtp.PlainAuth("", t.cfg.Username, t.cfg.Password, host)

	if err := smtp.SendMail(addr, auth, from, toAddrs, []byte(msg)); err != nil {
		return ErrorResult(fmt.Sprintf("SMTP send failed: %v", err))
	}

	result := fmt.Sprintf("Email sent successfully to %s with subject: %s", to, subject)
	logger.InfoCF("email", "Email sent", map[string]any{"to": to, "subject": subject})
	return NewToolResult(result)
}
