package terminal

import (
	"fmt"
	"time"
)

// ANSI color constants
const (
	ColorReset   = "\x1b[0m"
	ColorBold    = "\x1b[1m"
	ColorDim     = "\x1b[2m"

	// Standard colors
	ColorRed     = "\x1b[31m"
	ColorGreen   = "\x1b[32m"
	ColorYellow  = "\x1b[33m"
	ColorBlue    = "\x1b[34m"
	ColorMagenta = "\x1b[35m"
	ColorCyan    = "\x1b[36m"
	ColorWhite   = "\x1b[37m"
	ColorGray    = "\x1b[90m"

	// Bright colors
	ColorBrightRed     = "\x1b[1;31m"
	ColorBrightGreen   = "\x1b[1;32m"
	ColorBrightYellow  = "\x1b[1;33m"
	ColorBrightBlue    = "\x1b[1;34m"
	ColorBrightMagenta = "\x1b[1;35m"
	ColorBrightCyan    = "\x1b[1;36m"
	ColorBrightWhite   = "\x1b[1;37m"
)

// GetWelcomeBanner returns the BusinessOS ASCII art banner with colors
func GetWelcomeBanner() string {
	// Using the user's original ASCII art with gradient colors
	banner := "\r\n" +
		ColorBrightCyan + "╔════════════════════════════════════════════════════════════════════════╗\r\n" +
		ColorBrightCyan + "║                                                                        ║\r\n" +
		ColorBrightGreen + "║    ██████╗ ███████╗     █████╗  ██████╗ ███████╗███╗   ██╗████████╗    ║\r\n" +
		ColorBrightGreen + "║   ██╔═══██╗██╔════╝    ██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝    ║\r\n" +
		ColorBrightBlue + "║   ██║   ██║███████╗    ███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║       ║\r\n" +
		ColorBrightBlue + "║   ██║   ██║╚════██║    ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║       ║\r\n" +
		ColorBrightMagenta + "║   ╚██████╔╝███████║    ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║       ║\r\n" +
		ColorBrightMagenta + "║    ╚═════╝ ╚══════╝    ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝       ║\r\n" +
		ColorBrightCyan + "║                                                                        ║\r\n" +
		ColorBrightYellow + "║                 Business OS AI Agent Terminal v1.0                     ║\r\n" +
		ColorBrightCyan + "║                                                                        ║\r\n" +
		ColorBrightCyan + "╚════════════════════════════════════════════════════════════════════════╝\r\n" +
		ColorReset + "\r\n" +
		ColorGray + "  Type commands to interact with the system. Press Ctrl+D to exit.\r\n" +
		ColorReset + "\r\n"

	return banner
}

// SendWelcomeBanner sends the welcome banner to the PTY session
func SendWelcomeBanner(session *Session) error {
	if session.PTY == nil {
		return fmt.Errorf("PTY not initialized")
	}

	// Small delay to ensure PTY is fully initialized
	time.Sleep(50 * time.Millisecond)

	banner := GetWelcomeBanner()
	_, err := session.PTY.Write([]byte(banner))

	return err
}
