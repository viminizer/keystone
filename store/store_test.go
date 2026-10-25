package store

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	db := New()
	db.Set("name", "kevin")

	value, found := db.Get("name")
	if !found || value != "kevin" {
		t.Errorf("Get(\"name\") = %q, %v", value, found)
	}
}

func TestSetReplacesOldValue(t *testing.T) {
	db := New()
	db.Set("name", "kevin")
	db.Set("name", "alex")

	value, _ := db.Get("name")
	if value != "alex" {
		t.Errorf("Get(\"name\") = %q, want \"alex\"", value)
	}
}

func TestGetMissingKey(t *testing.T) {
	db := New()

	_, found := db.Get("name")
	if found {
		t.Error("Get on a missing key returned found = true")
	}
}

func TestManyGoroutines(t *testing.T) {
	db := New()
	var group sync.WaitGroup

	// Without the mutex, Go stops the program here with
	// "concurrent map writes".
	for i := range 100 {
		group.Add(1)
		go func() {
			db.Set(strconv.Itoa(i), "value")
			group.Done()
		}()
	}
	group.Wait()

	for i := range 100 {
		_, found := db.Get(strconv.Itoa(i))
		if !found {
			t.Errorf("key %d is missing", i)
		}
	}
}

func TestDelete(t *testing.T) {
	db := New()
	db.Set("name", "kevin")

	if !db.Delete("name") {
		t.Error("Delete on an existing key returned false")
	}
	if _, found := db.Get("name"); found {
		t.Error("key still exists after Delete")
	}
	if db.Delete("name") {
		t.Error("Delete on a missing key returned true")
	}
}

func TestExpiredKeyIsGone(t *testing.T) {
	db := New()
	db.Set("name", "kevin")
	db.Expire("name", 10*time.Millisecond)

	if _, found := db.Get("name"); !found {
		t.Fatal("key is gone before it expired")
	}
	time.Sleep(20 * time.Millisecond)
	if _, found := db.Get("name"); found {
		t.Error("key still exists after it expired")
	}
}

func TestExpireMissingKey(t *testing.T) {
	db := New()
	if db.Expire("name", time.Second) {
		t.Error("Expire on a missing key returned true")
	}
}

func TestSetRemovesExpiry(t *testing.T) {
	db := New()
	db.Set("name", "kevin")
	db.Expire("name", time.Second)
	db.Set("name", "alex")

	expiresAt, _ := db.ExpiresAt("name")
	if !expiresAt.IsZero() {
		t.Errorf("ExpiresAt = %v, want no expiry", expiresAt)
	}
}

func TestSetWithExpiry(t *testing.T) {
	db := New()
	db.SetWithExpiry("name", "kevin", 10*time.Millisecond)

	if value, found := db.Get("name"); !found || value != "kevin" {
		t.Fatalf("Get(\"name\") = %q, %v", value, found)
	}
	time.Sleep(20 * time.Millisecond)
	if _, found := db.Get("name"); found {
		t.Error("key still exists after it expired")
	}
}

func TestExpireAt(t *testing.T) {
	db := New()
	db.Set("name", "kevin")
	at := time.Now().Add(time.Hour)

	if !db.ExpireAt("name", at) {
		t.Fatal("ExpireAt on an existing key returned false")
	}
	expiresAt, _ := db.ExpiresAt("name")
	if !expiresAt.Equal(at) {
		t.Errorf("ExpiresAt = %v, want %v", expiresAt, at)
	}

	db.ExpireAt("name", time.Now().Add(-time.Second))
	if _, found := db.Get("name"); found {
		t.Error("key with an expiry in the past still exists")
	}
}
