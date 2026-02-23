package terminal

import (
	"golang.org/x/sys/unix"
)

type termState struct {
	termios unix.Termios
}

func makeRaw(fd uintptr) (*termState, error) {
	termios, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	if err != nil {
		return nil, err
	}

	old := &termState{termios: *termios}

	raw := *termios
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(int(fd), unix.TCSETS, &raw); err != nil {
		return nil, err
	}

	return old, nil
}

func restore(fd uintptr, state *termState) error {
	return unix.IoctlSetTermios(int(fd), unix.TCSETS, &state.termios)
}
