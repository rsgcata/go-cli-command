package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockLockableCommand is a mock command that implements the Command interface
// and will be wrapped by FsLockableCommand for testing
type MockLockableCommand struct {
	CommandWithoutFlags
	id          string
	description string
	execFunc    func() error
	executed    bool
}

func (m *MockLockableCommand) Id() string {
	return m.id
}

func (m *MockLockableCommand) Description() string {
	return m.description
}

func (m *MockLockableCommand) Exec(writer io.Writer) error {
	m.executed = true
	if m.execFunc != nil {
		return m.execFunc()
	}
	return nil
}

func TestLockableCommandHelper_Lock(t *testing.T) {
	// Create a temporary directory for the lock file
	tempDir, err := os.MkdirTemp("", "lockable-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock command
	mockCmd := &MockLockableCommand{
		id:          "test-command",
		description: "Test command for locking",
	}

	// Create a lockable command helper with a custom lock file path
	lockFilePath := filepath.Join(tempDir, "test-command.lock")
	helper := NewLockableCommandHelperWithPath(mockCmd, lockFilePath)

	// Test acquiring the lock
	err = helper.Lock()
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Verify the lock file exists
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Fatalf("Lock file was not created")
	}

	// Test that trying to acquire the lock again fails
	helper2 := NewLockableCommandHelperWithPath(mockCmd, lockFilePath)
	err = helper2.Lock()
	if err == nil {
		t.Fatalf("Expected lock acquisition to fail, but it succeeded")
	}

	// Release the lock
	helper.Unlock()

	// Verify the lock file is removed
	if _, err := os.Stat(lockFilePath); !os.IsNotExist(err) {
		t.Fatalf("Lock file was not removed")
	}

	// Test that we can acquire the lock again after releasing it
	err = helper2.Lock()
	if err != nil {
		t.Fatalf("Failed to acquire lock after it was released: %v", err)
	}
	helper2.Unlock()
}

func TestLockableCommandHelper_Exec(t *testing.T) {
	// Create a temporary directory for the lock file
	tempDir, err := os.MkdirTemp("", "lockable-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock command
	mockCmd := &MockLockableCommand{
		id:          "test-command",
		description: "Test command for locking",
	}

	// Create a lockable command helper with a custom lock file path
	lockFilePath := filepath.Join(tempDir, "test-command.lock")
	helper := NewLockableCommandHelperWithPath(mockCmd, lockFilePath)

	// Test executing the command
	var buf bytes.Buffer
	err = helper.Exec(&buf)
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Verify the command was executed
	if !mockCmd.executed {
		t.Fatalf("Command was not executed")
	}

	// Verify the lock file is removed after execution
	if _, err := os.Stat(lockFilePath); !os.IsNotExist(err) {
		t.Fatalf("Lock file was not removed after execution")
	}
}

func TestLockableCommandHelper_ConcurrentExecution(t *testing.T) {
	// Create a temporary directory for the lock file
	tempDir, err := os.MkdirTemp("", "lockable-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock command that takes some time to execute
	mockCmd := &MockLockableCommand{
		id:          "slow-command",
		description: "Slow command for testing concurrent execution",
		execFunc: func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	// Create a lockable command helper with a custom lock file path
	lockFilePath := filepath.Join(tempDir, "slow-command.lock")
	helper := NewLockableCommandHelperWithPath(mockCmd, lockFilePath)

	// Start executing the command in a goroutine
	var err1 error
	var buf1 bytes.Buffer
	done := make(chan bool)
	go func() {
		err1 = helper.Exec(&buf1)
		done <- true
	}()

	// Try to execute the command again immediately
	var err2 error
	var buf2 bytes.Buffer
	helper2 := NewLockableCommandHelperWithPath(mockCmd, lockFilePath)
	err2 = helper2.Exec(&buf2)

	// Wait for the first execution to complete
	<-done

	// The first execution should succeed
	if err1 != nil {
		t.Fatalf("First execution failed: %v", err1)
	}

	// The second execution should fail with a lock error
	if err2 == nil {
		t.Fatalf("Expected second execution to fail, but it succeeded")
	}
}
