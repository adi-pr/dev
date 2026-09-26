package ports

import (
	"net/netip"
	"os/user"
	"slices"
	"strconv"

	"github.com/adi-pr/dev/internal/project"
)

// Listener is a TCP port in the LISTEN state and the processes holding it.
type Listener struct {
	Port      int          `json:"port"`
	Addresses []netip.Addr `json:"addresses"`
	UID       int          `json:"uid"`
	User      string       `json:"user"`
	// Project is the project a holding process runs in, if any.
	Project string `json:"project,omitempty"`
	// Processes is empty when the sockets belong to another user, whose
	// file descriptors can't be inspected.
	Processes []Process `json:"processes"`

	// inodes identifies the listening sockets, so WaitClosed can tell when
	// they are gone.
	inodes []uint64
}

type Process struct {
	PID     int      `json:"pid"`
	Name    string   `json:"name"`
	Command []string `json:"command"`
	Dir     string   `json:"dir,omitempty"`

	// started is the start time from /proc/<pid>/stat. Kill compares it to
	// detect a PID that was reused since the scan.
	started uint64
}

// Scan lists every listening TCP port, sorted by port. Processes are labelled
// with the project their working directory is in.
func Scan(projects []project.Project) ([]Listener, error) {
	sockets, err := listeningSockets()
	if err != nil {
		return nil, err
	}

	wanted := make(map[uint64]bool, len(sockets))

	for _, s := range sockets {
		wanted[s.inode] = true
	}

	owners, err := socketOwners(wanted)
	if err != nil {
		return nil, err
	}

	byPort := make(map[int]*Listener)
	processes := make(map[int]*Process)
	users := make(map[int]string)

	for _, s := range sockets {
		l, ok := byPort[s.port]
		if !ok {
			// A port is assumed to have one owner; sockets from different
			// users on the same port would need distinct bind addresses.
			l = &Listener{
				Port:      s.port,
				UID:       s.uid,
				User:      lookupUser(users, s.uid),
				Processes: []Process{},
			}
			byPort[s.port] = l
		}

		l.inodes = append(l.inodes, s.inode)

		if !slices.Contains(l.Addresses, s.addr) {
			l.Addresses = append(l.Addresses, s.addr)
		}

		for _, pid := range owners[s.inode] {
			if slices.ContainsFunc(l.Processes, func(p Process) bool {
				return p.PID == pid
			}) {
				continue
			}

			p, ok := processes[pid]
			if !ok {
				read, err := readProcess(pid)
				if err != nil {
					// Exited since the fd scan.
					continue
				}

				p = &read
				processes[pid] = p
			}

			l.Processes = append(l.Processes, *p)
		}
	}

	listeners := make([]Listener, 0, len(byPort))

	for _, l := range byPort {
		slices.SortFunc(l.Addresses, netip.Addr.Compare)

		slices.SortFunc(l.Processes, func(a, b Process) int {
			return a.PID - b.PID
		})

		for _, p := range l.Processes {
			if p.Dir == "" {
				continue
			}

			if match, ok := project.Containing(projects, p.Dir); ok {
				l.Project = match.Name
				break
			}
		}

		listeners = append(listeners, *l)
	}

	slices.SortFunc(listeners, func(a, b Listener) int {
		return a.Port - b.Port
	})

	return listeners, nil
}

// Find returns the listener on port.
func Find(listeners []Listener, port int) (Listener, bool) {
	for _, l := range listeners {
		if l.Port == port {
			return l, true
		}
	}

	return Listener{}, false
}

func lookupUser(cache map[int]string, uid int) string {
	if name, ok := cache[uid]; ok {
		return name
	}

	name := strconv.Itoa(uid)

	if u, err := user.LookupId(name); err == nil {
		name = u.Username
	}

	cache[uid] = name

	return name
}
