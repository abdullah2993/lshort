package lshort

import (
	"testing"

	"github.com/dgraph-io/badger"
)

func TestShrinkAndExpandBolt(t *testing.T) {
	tts := []string{"test", "test1", "test2", "test3", "test4", "test5", "test6", "test7"}

	ls, err := NewLinkShortnerBolt("db.tmp", nil)
	if err != nil {
		t.Errorf("unable to create a shrinker: %v", err)
	}

	for _, tt := range tts {
		key, err := ls.Shrink(tt)
		if err != nil {
			t.Errorf("unable to shrink url: %v", err)
		}
		url, err := ls.Expand(key)
		if err != nil {
			t.Errorf("unable to expand url: %v", err)
		}
		if url != tt {
			t.Fail()
		}
	}
}
func TestShrinkAndExpandBadger(t *testing.T) {
	tts := []string{"test", "test1", "test2", "test3", "test4", "test5", "test6", "test7"}

	ls, err := NewLinkShortnerBadger("db_1.tmp", badger.DefaultOptions)
	if err != nil {
		t.Errorf("unable to create a shrinker: %v", err)
	}

	for _, tt := range tts {
		key, err := ls.Shrink(tt)
		if err != nil {
			t.Errorf("unable to shrink url: %v", err)
		}
		url, err := ls.Expand(key)
		if err != nil {
			t.Errorf("unable to expand url: %v", err)
		}
		if url != tt {
			t.Fail()
		}
	}
}
