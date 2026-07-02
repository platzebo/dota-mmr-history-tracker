package store

import (
	"path/filepath"
	"testing"
)

func TestAutoBackfillCursorPersistsAcrossStoreReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.AutoBackfillCursor(); err != nil || ok {
		t.Fatalf("expected no cursor initially, ok=%t err=%v", ok, err)
	}
	if complete, err := s.AutoBackfillComplete(); err != nil || complete {
		t.Fatalf("expected incomplete history initially, complete=%t err=%v", complete, err)
	}
	if err := s.SetAutoBackfillCursor(12345); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s, err = Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, ok, err := s.AutoBackfillCursor()
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != 12345 {
		t.Fatalf("cursor=(%d,%t), want (12345,true)", got, ok)
	}
	if err := s.ClearAutoBackfillCursor(); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.AutoBackfillCursor(); err != nil || ok {
		t.Fatalf("expected cleared cursor, ok=%t err=%v", ok, err)
	}
	if err := s.SetAutoBackfillComplete(true); err != nil {
		t.Fatal(err)
	}
	if complete, err := s.AutoBackfillComplete(); err != nil || !complete {
		t.Fatalf("expected complete history, complete=%t err=%v", complete, err)
	}
	if err := s.SetAutoBackfillComplete(false); err != nil {
		t.Fatal(err)
	}
	if complete, err := s.AutoBackfillComplete(); err != nil || complete {
		t.Fatalf("expected incomplete history after reset, complete=%t err=%v", complete, err)
	}
}
