package ports

import (
	"encoding/binary"
	"net/netip"
	"testing"
)

// The fixtures were captured on a little-endian machine; see parseAddr.
func skipBigEndian(t *testing.T) {
	if binary.NativeEndian.Uint16([]byte{1, 0}) != 1 {
		t.Skip("fixtures are little-endian")
	}
}

func TestParseSocketTable(t *testing.T) {
	skipBigEndian(t)

	table := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0BB8 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 111 1 0000000000000000 100 0 0 10 0
   1: 0100007F:0BB8 0100007F:D431 01 00000000:00000000 00:00000000 00000000  1000        0 222 1 0000000000000000 20 4 30 10 -1
   2: 00000000:1538 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 333 1 0000000000000000 100 0 0 10 0
`

	sockets, err := parseSocketTable([]byte(table))
	if err != nil {
		t.Fatal(err)
	}

	want := []socket{
		{addr: netip.MustParseAddr("127.0.0.1"), port: 3000, uid: 1000, inode: 111},
		{addr: netip.MustParseAddr("0.0.0.0"), port: 5432, uid: 0, inode: 333},
	}

	if len(sockets) != len(want) {
		t.Fatalf("got %d sockets, want %d: %+v", len(sockets), len(want), sockets)
	}

	for i := range want {
		if sockets[i] != want[i] {
			t.Errorf("socket %d = %+v, want %+v", i, sockets[i], want[i])
		}
	}
}

func TestParseAddr(t *testing.T) {
	skipBigEndian(t)

	tests := []struct {
		hex  string
		want string
	}{
		{"0100007F", "127.0.0.1"},
		{"0964A8C0", "192.168.100.9"},
		{"00000000000000000000000000000000", "::"},
		{"00000000000000000000000001000000", "::1"},
		{"0000000000000000FFFF00000100007F", "127.0.0.1"},
		{"000080FE00000000FF12F4AB85A1F8BE", "fe80::abf4:12ff:bef8:a185"},
	}

	for _, test := range tests {
		got, err := parseAddr(test.hex)
		if err != nil {
			t.Errorf("parseAddr(%q): %v", test.hex, err)
			continue
		}

		if got.String() != test.want {
			t.Errorf("parseAddr(%q) = %s, want %s", test.hex, got, test.want)
		}
	}

	if _, err := parseAddr("0100"); err == nil {
		t.Error("parseAddr accepted a 2-byte address")
	}
}

func TestParseStartTime(t *testing.T) {
	// The command name may contain spaces and parentheses.
	stat := "4242 (next-server (v1) S 1 4242 4242 0 -1 4194560 100 0 0 0 5 1 0 0 20 0 11 0 987654 1234 56 18446744073709551615"

	got, err := parseStartTime(stat)
	if err != nil {
		t.Fatal(err)
	}

	if got != 987654 {
		t.Errorf("start time = %d, want 987654", got)
	}

	if _, err := parseStartTime("4242 (node) S 1"); err == nil {
		t.Error("parseStartTime accepted a truncated stat")
	}
}

func TestSocketInode(t *testing.T) {
	if inode, ok := socketInode("socket:[123456]"); !ok || inode != 123456 {
		t.Errorf("socketInode = %d, %v", inode, ok)
	}

	for _, target := range []string{"pipe:[123]", "/dev/null", "anon_inode:[eventfd]"} {
		if _, ok := socketInode(target); ok {
			t.Errorf("socketInode(%q) matched", target)
		}
	}
}
