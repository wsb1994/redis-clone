package redisutils

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_insertIntoStuff(t *testing.T) {
	DataStore.DataStore.Store("hello", "world")
	result, ok := DataStore.DataStore.Load("hello")
	assert.Equal(t, result, "world")
	assert.Equal(t, ok, true)
}

func Test_SortingStuff(t *testing.T) {
	entry1 := &EvictionEntry{
		Key:       "one",
		Timestamp: time.Now(),
	}

	time.Sleep(1)
	entry2 := &EvictionEntry{
		Key:       "two",
		Timestamp: time.Now(),
	}
	time.Sleep(3)
	entry3 := &EvictionEntry{
		Key:       "three",
		Timestamp: time.Now(),
	}

	EvictionSchedule.Mtx.Lock()
	EvictionSchedule.EvictionSchedule = append(EvictionSchedule.EvictionSchedule, *entry2)
	EvictionSchedule.Mtx.Unlock()

	EvictionSchedule.Mtx.Lock()
	EvictionSchedule.EvictionSchedule = append(EvictionSchedule.EvictionSchedule, *entry1)
	EvictionSchedule.Mtx.Unlock()

	EvictionSchedule.Mtx.Lock()
	EvictionSchedule.EvictionSchedule = append(EvictionSchedule.EvictionSchedule, *entry3)
	EvictionSchedule.Mtx.Unlock()

	sort.Sort(EvictionSchedule)
	assert.Equal(t, EvictionSchedule.EvictionSchedule[0].Key, entry1.Key)
	assert.Equal(t, EvictionSchedule.EvictionSchedule[1].Key, entry2.Key)
	assert.Equal(t, EvictionSchedule.EvictionSchedule[2].Key, entry3.Key)
}
