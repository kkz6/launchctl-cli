package terminal

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/tui"
)

const (
	clrEOL = "\033[K"

	headerLines = 4
	footerLines = 2
)

var (
	frameBreadcrumbStyle       = lipgloss.NewStyle().Foreground(tui.Slate)
	frameBreadcrumbActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(tui.Indigo)
	frameMutedStyle            = lipgloss.NewStyle().Foreground(tui.Muted)
	frameConnectedStyle        = lipgloss.NewStyle().Bold(true).Foreground(tui.Green)
	frameDividerStyle          = lipgloss.NewStyle().Foreground(tui.DarkSlate)
	frameFooterStyle           = lipgloss.NewStyle().Bold(true).Foreground(tui.White)
)

type frame struct {
	breadcrumb []string
	status     string
	statusTag  string
	footer     string
	headerRows int
	footerRows int
}

func newFrame(serverName, serverIP, username string) *frame {
	if username == "" {
		username = "launcher"
	}

	return &frame{
		breadcrumb: []string{"lctl", "Servers", serverName, "Terminal"},
		status:     fmt.Sprintf("%s@%s", username, serverIP),
		statusTag:  "Connected",
		footer:     "Ctrl+D disconnect",
		headerRows: headerLines,
		footerRows: footerLines,
	}
}

func (f *frame) renderBreadcrumb() string {
	var b strings.Builder

	for i, part := range f.breadcrumb {
		if i > 0 {
			b.WriteString(frameMutedStyle.Render(" > "))
		}

		if i == len(f.breadcrumb)-1 {
			b.WriteString(frameBreadcrumbActiveStyle.Render(part))
		} else {
			b.WriteString(frameBreadcrumbStyle.Render(part))
		}
	}

	return b.String()
}

func (f *frame) renderStatus() string {
	return fmt.Sprintf("%s  %s  %s",
		frameBreadcrumbStyle.Render(f.status),
		frameMutedStyle.Render("•"),
		frameConnectedStyle.Render(f.statusTag),
	)
}

func (f *frame) divider(cols int, double bool) string {
	ch := "─"
	if double {
		ch = "═"
	}

	return frameDividerStyle.Render(strings.Repeat(ch, cols))
}

func (f *frame) scrollRegionSeq(rows int) string {
	top := f.headerRows + 1
	bottom := rows - f.footerRows
	if bottom <= top {
		bottom = top + 1
	}

	return fmt.Sprintf("\033[%d;%dr", top, bottom)
}

// regionClearSeq returns a sequence that moves to region top and erases everything below.
// The footer gets wiped but ensureFooter() redraws it immediately after.
func (f *frame) regionClearSeq() string {
	return fmt.Sprintf("\033[%d;1H\033[J", f.headerRows+1)
}

func (f *frame) draw(cols, rows int) {
	out := os.Stdout

	fmt.Fprint(out, "\033[H\033[2J\033[3J")

	moveTo(out, 1, 1)
	fmt.Fprintf(out, "%s%s\n", f.renderBreadcrumb(), clrEOL)
	fmt.Fprintf(out, "%s%s\n", f.divider(cols, false), clrEOL)
	fmt.Fprintf(out, "%s%s\n", f.renderStatus(), clrEOL)
	fmt.Fprintf(out, "%s%s\n", f.divider(cols, true), clrEOL)

	f.writeFooterDirect(out, cols, rows)
}

func (f *frame) writeFooterDirect(out *os.File, cols, rows int) {
	moveTo(out, rows-1, 1)
	fmt.Fprintf(out, "%s%s", f.divider(cols, false), clrEOL)

	moveTo(out, rows, 1)
	fmt.Fprintf(out, "%s%s", frameFooterStyle.Render(f.footer), clrEOL)
}

func (f *frame) ensureFooter() {
	cols, rows, _ := getTerminalSize()

	var buf bytes.Buffer

	buf.WriteString("\0337")
	buf.WriteString("\033[r")

	buf.WriteString(fmt.Sprintf("\033[%d;1H%s%s", rows-1, f.divider(cols, false), clrEOL))
	buf.WriteString(fmt.Sprintf("\033[%d;1H%s%s", rows, frameFooterStyle.Render(f.footer), clrEOL))

	buf.WriteString(f.scrollRegionSeq(rows))
	buf.WriteString("\0338")

	os.Stdout.Write(buf.Bytes())
}

func (f *frame) redraw(cols, rows int) {
	out := os.Stdout

	fmt.Fprint(out, "\0337")
	fmt.Fprint(out, "\033[r")

	moveTo(out, 1, 1)
	fmt.Fprintf(out, "%s%s\n", f.renderBreadcrumb(), clrEOL)
	fmt.Fprintf(out, "%s%s\n", f.divider(cols, false), clrEOL)
	fmt.Fprintf(out, "%s%s\n", f.renderStatus(), clrEOL)
	fmt.Fprintf(out, "%s%s\n", f.divider(cols, true), clrEOL)

	f.writeFooterDirect(out, cols, rows)

	fmt.Fprint(out, "\0338")
}

func setupScrollRegion(headerRows, footerRows, totalRows int) {
	top := headerRows + 1
	bottom := totalRows - footerRows

	if bottom <= top {
		bottom = top + 1
	}

	out := os.Stdout
	fmt.Fprintf(out, "\033[%d;%dr", top, bottom)
	moveTo(out, top, 1)
}

func resetScrollRegion() {
	out := os.Stdout
	fmt.Fprint(out, "\033[r")
	fmt.Fprint(out, "\033[2J")
	fmt.Fprint(out, "\033[H")
}

func moveTo(out *os.File, row, col int) {
	fmt.Fprintf(out, "\033[%d;%dH", row, col)
}

func enableMouseTracking() {
	fmt.Fprint(os.Stdout, "\033[?1000h\033[?1006h")
}

func disableMouseTracking() {
	fmt.Fprint(os.Stdout, "\033[?1000l\033[?1006l")
}
