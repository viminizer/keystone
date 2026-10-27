package store

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/viminizer/keystone/internal/metrics"
)

func TestDeleteExpiredKeys(t *testing.T) {
	db := New(metrics.NewExpiryStats())
	for i := range 100 {
		db.SetWithExpiry("temp"+strconv.Itoa(i), "value", time.Now().Add(time.Millisecond))
	}
	for i := range 50 {
		db.Set("keep"+strconv.Itoa(i), "value")
	}
	time.Sleep(5 * time.Millisecond)

	// Nobody reads the temp keys, so only active expiry can remove them.
	db.DeleteExpiredKeys()

	if len(db.data) != 50 {
		t.Errorf("%d keys left, want the 50 keys without expiry", len(db.data))
	}
	if len(db.expiring) != 0 {
		t.Errorf("%d keys left in expiring, want 0", len(db.expiring))
	}
}

func TestDeleteExpiredSampleKeepsLiveKeys(t *testing.T) {
	db := New(metrics.NewExpiryStats())
	db.SetWithExpiry("name", "kevin", time.Now().Add(time.Hour))

	checked, deleted := db.deleteExpiredSample()
	if checked != 1 || deleted != 0 {
		t.Errorf("checked %d, deleted %d, want 1 and 0", checked, deleted)
	}
	if _, found := db.Get("name"); !found {
		t.Error("a key that has not expired was deleted")
	}
}

func TestExpiringStaysInSync(t *testing.T) {
	db := New(metrics.NewExpiryStats())

	db.SetWithExpiry("a", "1", time.Now().Add(time.Hour))
	db.Set("a", "2")
	if db.expiring["a"] {
		t.Error("Set did not remove the key from expiring")
	}

	db.Set("b", "1")
	db.ExpireAt("b", time.Now().Add(time.Hour))
	if !db.expiring["b"] {
		t.Error("Expire did not add the key to expiring")
	}
	db.Delete("b")
	if db.expiring["b"] {
		t.Error("Delete did not remove the key from expiring")
	}

	db.SetWithExpiry("c", "1", time.Now().Add(time.Millisecond))
	time.Sleep(5 * time.Millisecond)
	db.Get("c")
	if db.expiring["c"] {
		t.Error("lazy expiry did not remove the key from expiring")
	}
}

func TestExpiryStats(t *testing.T) {
	expiryStats := metrics.NewExpiryStats()
	db := New(expiryStats)
	db.SetWithExpiry("read", "value", time.Now().Add(time.Millisecond))
	db.SetWithExpiry("unread1", "value", time.Now().Add(time.Millisecond))
	db.SetWithExpiry("unread2", "value", time.Now().Add(time.Millisecond))
	db.SetWithExpiry("later", "value", time.Now().Add(time.Hour))
	time.Sleep(5 * time.Millisecond)

	db.Get("read")
	db.DeleteExpiredKeys()

	var output strings.Builder
	err := expiryStats.WritePrometheus(&output)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		`keystone_expired_keys_total{by="active"} 2`,
		`keystone_expired_keys_total{by="lookup"} 1`,
		"keystone_keys_with_expiry 1",
	} {
		if !strings.Contains(output.String(), line+"\n") {
			t.Errorf("missing line %s in:\n%s", line, output.String())
		}
	}
}
