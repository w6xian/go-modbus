package modbus

import (
	"testing"
	"time"
)

func TestClientLockWritePriority(t *testing.T) {
	l := newClientLock()
	if err := l.RLock(); err != nil {
		t.Fatalf("RLock failed: %v", err)
	}

	writerAcquired := make(chan struct{})
	writerRelease := make(chan struct{})
	writerDone := make(chan struct{})
	go func() {
		if err := l.Lock(); err != nil {
			t.Errorf("writer Lock failed: %v", err)
			close(writerDone)
			return
		}
		close(writerAcquired)
		<-writerRelease
		if err := l.Unlock(); err != nil {
			t.Errorf("writer Unlock failed: %v", err)
		}
		close(writerDone)
	}()

	time.Sleep(20 * time.Millisecond)

	readerAcquired := make(chan struct{})
	readerRelease := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		if err := l.RLock(); err != nil {
			t.Errorf("reader RLock failed: %v", err)
			close(readerDone)
			return
		}
		close(readerAcquired)
		<-readerRelease
		if err := l.RUnlock(); err != nil {
			t.Errorf("reader RUnlock failed: %v", err)
		}
		close(readerDone)
	}()

	select {
	case <-writerAcquired:
		t.Fatal("writer should wait while reader holds the bus")
	case <-readerAcquired:
		t.Fatal("new reader should wait while writer is pending")
	case <-time.After(20 * time.Millisecond):
	}

	if err := l.RUnlock(); err != nil {
		t.Fatalf("initial RUnlock failed: %v", err)
	}

	select {
	case <-writerAcquired:
	case <-time.After(time.Second):
		t.Fatal("writer did not acquire the bus after reader released it")
	}

	select {
	case <-readerAcquired:
		t.Fatal("reader should not bypass pending writer")
	case <-time.After(20 * time.Millisecond):
	}

	close(writerRelease)
	<-writerDone

	select {
	case <-readerAcquired:
	case <-time.After(time.Second):
		t.Fatal("reader did not acquire the bus after writer finished")
	}

	close(readerRelease)
	<-readerDone
}

func TestClientLockReentrantWriteAllowsNestedRead(t *testing.T) {
	l := newClientLock()

	if err := l.Lock(); err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	if err := l.RLock(); err != nil {
		t.Fatalf("nested RLock failed: %v", err)
	}
	if err := l.RUnlock(); err != nil {
		t.Fatalf("nested RUnlock failed: %v", err)
	}
	if err := l.Unlock(); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
}
