package ports

import (
	"os/exec"
	"testing"
	"time"
)

func TestKillChecksStartTime(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	exited := make(chan struct{})

	go func() {
		cmd.Wait()
		close(exited)
	}()

	t.Cleanup(func() { cmd.Process.Kill() })

	started, err := readStartTime(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}

	// A different start time stands in for a PID reused since the scan.
	stale := Process{PID: cmd.Process.Pid, Name: "sleep", started: started + 1}

	if err := Kill(stale, false); err != nil {
		t.Fatal(err)
	}

	select {
	case <-exited:
		t.Fatal("Kill signalled a process with a different start time")
	case <-time.After(200 * time.Millisecond):
	}

	current := Process{PID: cmd.Process.Pid, Name: "sleep", started: started}

	if err := Kill(current, false); err != nil {
		t.Fatal(err)
	}

	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("Kill did not stop the process")
	}

	// Killing a process that has already exited is not an error.
	if err := Kill(current, false); err != nil {
		t.Errorf("Kill after exit: %v", err)
	}
}
