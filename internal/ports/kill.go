package ports

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"golang.org/x/sys/unix"
)

const pollInterval = 100 * time.Millisecond

// Kill sends SIGTERM to p, or SIGKILL when force is set. A process that has
// exited since the scan is not an error, and a PID reused by another process
// in the meantime is never signalled.
func Kill(p Process, force bool) error {
	sig := unix.SIGTERM
	if force {
		sig = unix.SIGKILL
	}

	pidfd, err := unix.PidfdOpen(p.PID, 0)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("kill %s (%d): %w", p.Name, p.PID, err)
	}

	defer unix.Close(pidfd)

	// The pidfd pins whichever process had the PID when it was opened. If
	// that process still has the scanned start time, it is the one that was
	// scanned, and the signal can only reach it.
	started, err := readStartTime(p.PID)
	if err != nil || started != p.started {
		return nil
	}

	err = unix.PidfdSendSignal(pidfd, sig, nil, 0)
	if err != nil && !errors.Is(err, unix.ESRCH) {
		return fmt.Errorf("kill %s (%d): %w", p.Name, p.PID, err)
	}

	return nil
}

// WaitClosed polls until all of l's listening sockets are gone or timeout
// passes, and reports whether they closed.
func WaitClosed(l Listener, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)

	for {
		sockets, err := listeningSockets()
		if err != nil {
			return false, err
		}

		open := slices.ContainsFunc(sockets, func(s socket) bool {
			return s.port == l.Port && slices.Contains(l.inodes, s.inode)
		})

		if !open {
			return true, nil
		}

		if time.Now().After(deadline) {
			return false, nil
		}

		time.Sleep(pollInterval)
	}
}
