package gosip

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const Version = "0.1.0"

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiRed    = "\x1b[38;5;9m"
	ansiPurple = "\x1b[38;5;141m"
	ansiCyan   = "\x1b[38;5;80m"
	ansiGreen  = "\x1b[38;5;114m"
)

var logo = [...]string{
	"  ██████      ██████      ████████  ██████████  ████████  ",
	"██      ██  ██      ██  ██              ██      ██      ██",
	"██          ██      ██  ██              ██      ██      ██",
	"██  ██████  ██      ██    ██████        ██      ████████  ",
	"██      ██  ██      ██          ██      ██      ██        ",
	"██      ██  ██      ██          ██      ██      ██        ",
	"  ██████      ██████    ████████    ██████████  ██        ",
}

func (r *Router) printBanner(addr string) {
	color := isTerminal() && os.Getenv("NO_COLOR") == ""

	host, port := splitAddr(addr)
	localURL := fmt.Sprintf("http://%s:%s/", prettyHost(host), port)

	purple := func(s string) string { return paint(s, ansiPurple, color) }
	red := func(s string) string { return paint(s, ansiRed, color) }
	boldCyan := func(s string) string { return paint(s, ansiBold+ansiCyan, color) }
	green := func(s string) string { return paint(s, ansiGreen, color) }
	arrow := purple("➜")

	var b strings.Builder
	b.WriteString("\n")
	for _, line := range logo {
		b.WriteString("  " + purple(line) + "\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + red(" v"+Version) + "\n\n")
	fmt.Fprintf(&b, "  %s  %s %s\n", arrow, "Local:  ", boldCyan(localURL))
	fmt.Fprintf(&b, "  %s  %s %s\n", arrow, "Network:", "bound on "+host+":"+port)
	fmt.Fprintf(&b, "  %s  %s %s\n", arrow, "Routes: ", green(strconv.Itoa(r.root.routeCount)))
	fmt.Fprintf(&b, "  %s  %s %s\n", arrow, "PID:    ", red(strconv.Itoa(os.Getpid())))
	b.WriteString("\n")

	fmt.Print(b.String())
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func splitAddr(addr string) (host, port string) {
	if addr == "" {
		return "0.0.0.0", "8000"
	}
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return "0.0.0.0", addr
	}
	host = addr[:i]
	port = addr[i+1:]
	if host == "" {
		host = "0.0.0.0"
	}
	return
}

func prettyHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	}
	return host
}

func paint(s, colorCode string, enabled bool) string {
	if !enabled {
		return s
	}
	return colorCode + s + ansiReset
}
