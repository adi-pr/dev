package ports

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// tcpListen is the kernel's TCP_LISTEN state as printed in /proc/net/tcp.
const tcpListen = "0A"

// socket is one listening TCP socket from /proc/net/tcp or /proc/net/tcp6.
type socket struct {
	addr  netip.Addr
	port  int
	uid   int
	inode uint64
}

// listeningSockets reads both socket tables. tcp6 is missing when IPv6 is
// disabled, which is not an error.
func listeningSockets() ([]socket, error) {
	var sockets []socket

	for _, table := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		data, err := os.ReadFile(table)
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			return nil, err
		}

		parsed, err := parseSocketTable(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", table, err)
		}

		sockets = append(sockets, parsed...)
	}

	return sockets, nil
}

// parseSocketTable returns the LISTEN entries of a /proc/net/tcp{,6} table.
//
//	sl  local_address rem_address   st tx_queue:rx_queue tr:tm->when retrnsmt uid timeout inode
//	0: 0100007F:0BB8 00000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 123456 ...
func parseSocketTable(data []byte) ([]socket, error) {
	var sockets []socket

	rows := strings.Split(strings.TrimSpace(string(data)), "\n")

	// The first row is the header.
	for _, row := range rows[1:] {
		fields := strings.Fields(row)
		if len(fields) < 10 {
			return nil, fmt.Errorf("short row %q", row)
		}

		if fields[3] != tcpListen {
			continue
		}

		addrHex, portHex, ok := strings.Cut(fields[1], ":")
		if !ok {
			return nil, fmt.Errorf("bad local address %q", fields[1])
		}

		addr, err := parseAddr(addrHex)
		if err != nil {
			return nil, err
		}

		port, err := strconv.ParseUint(portHex, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("bad port %q", portHex)
		}

		uid, err := strconv.Atoi(fields[7])
		if err != nil {
			return nil, fmt.Errorf("bad uid %q", fields[7])
		}

		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad inode %q", fields[9])
		}

		sockets = append(sockets, socket{
			addr:  addr,
			port:  int(port),
			uid:   uid,
			inode: inode,
		})
	}

	return sockets, nil
}

// parseAddr decodes an address from a socket table. The kernel prints each
// 32-bit word of the network-order address as a native-endian integer, so
// every word is re-encoded in native order to recover the original bytes.
func parseAddr(value string) (netip.Addr, error) {
	raw, err := hex.DecodeString(value)
	if err != nil || (len(raw) != 4 && len(raw) != 16) {
		return netip.Addr{}, fmt.Errorf("bad address %q", value)
	}

	for i := 0; i < len(raw); i += 4 {
		word := binary.BigEndian.Uint32(raw[i:])
		binary.NativeEndian.PutUint32(raw[i:], word)
	}

	addr, _ := netip.AddrFromSlice(raw)

	// Dual-stack sockets report IPv4 peers as ::ffff:a.b.c.d.
	return addr.Unmap(), nil
}

// socketOwners maps each wanted socket inode to the PIDs holding it open.
// Other users' file descriptors can't be read, so their sockets are missing
// from the result.
func socketOwners(wanted map[uint64]bool) (map[uint64][]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	owners := make(map[uint64][]int)

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		dir := filepath.Join("/proc", entry.Name(), "fd")

		// Permission denied, or the process exited mid-scan.
		fds, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		held := make(map[uint64]bool)

		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join(dir, fd.Name()))
			if err != nil {
				continue
			}

			inode, ok := socketInode(target)
			if ok && wanted[inode] && !held[inode] {
				held[inode] = true
				owners[inode] = append(owners[inode], pid)
			}
		}
	}

	return owners, nil
}

// socketInode parses an fd link target of the form socket:[12345].
func socketInode(target string) (uint64, bool) {
	value, ok := strings.CutPrefix(target, "socket:[")
	if !ok {
		return 0, false
	}

	inode, err := strconv.ParseUint(strings.TrimSuffix(value, "]"), 10, 64)

	return inode, err == nil
}

// readProcess reads a process's name, command line, working directory and
// start time. It fails if the process has exited.
func readProcess(pid int) (Process, error) {
	p := Process{PID: pid}

	base := filepath.Join("/proc", strconv.Itoa(pid))

	started, err := readStartTime(pid)
	if err != nil {
		return p, err
	}

	p.started = started

	if comm, err := os.ReadFile(filepath.Join(base, "comm")); err == nil {
		p.Name = strings.TrimSpace(string(comm))
	}

	if cmdline, err := os.ReadFile(filepath.Join(base, "cmdline")); err == nil {
		p.Command = splitCmdline(cmdline)
	}

	if cwd, err := os.Readlink(filepath.Join(base, "cwd")); err == nil {
		p.Dir = cwd
	}

	return p, nil
}

func splitCmdline(data []byte) []string {
	data = bytes.TrimRight(data, "\x00")
	if len(data) == 0 {
		return nil
	}

	return strings.Split(string(data), "\x00")
}

// readStartTime returns when the process started, in clock ticks since boot.
// Together with the PID it identifies a process, since PIDs are reused.
func readStartTime(pid int) (uint64, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, err
	}

	return parseStartTime(string(data))
}

// parseStartTime extracts field 22 of /proc/<pid>/stat. The command name in
// field 2 may itself contain spaces and parentheses, so fields are counted
// from the last closing parenthesis.
func parseStartTime(stat string) (uint64, error) {
	end := strings.LastIndexByte(stat, ')')
	if end < 0 {
		return 0, fmt.Errorf("bad stat %q", stat)
	}

	// Fields after the name start at field 3 (state).
	fields := strings.Fields(stat[end+1:])

	const startTimeIndex = 22 - 3

	if len(fields) <= startTimeIndex {
		return 0, fmt.Errorf("bad stat %q", stat)
	}

	return strconv.ParseUint(fields[startTimeIndex], 10, 64)
}
