package mocks

import "testing"

func TestStorage_PutGetDelete(t *testing.T) {
	s := NewStorage()
	key, err := s.Put([]byte("hello"), "image/png")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	data, ct, err := s.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(data) != "hello" || ct != "image/png" {
		t.Fatalf("Get() = (%q, %q), want (%q, %q)", data, ct, "hello", "image/png")
	}
	if err := s.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := s.Get(key); err == nil {
		t.Fatal("Get after Delete: want error, got nil")
	}
}
