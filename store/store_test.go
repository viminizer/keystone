package store

import (
	"strconv"
	"sync"
	"testing"
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
