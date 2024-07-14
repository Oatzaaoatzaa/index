package store_test

import (
	"bytes"
	"fmt"
	"github.com/memocash/index/db/store"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"path/filepath"
	"testing"
)

const TestTopic = "test"
const TestShard = 0

const PrefixTest = "test"
const PrefixOther = "other"

var (
	testMessageTest0 = &store.Message{Uid: []byte("test-0"), Message: []byte("message-0")}
	testMessageTest1 = &store.Message{Uid: []byte("test-1"), Message: []byte("message-1")}
	testMessageTest2 = &store.Message{Uid: []byte("test-2"), Message: []byte("message-2")}
	testMessageTest3 = &store.Message{Uid: []byte("test-3"), Message: []byte("message-3")}
	testMessageTest4 = &store.Message{Uid: []byte("test-4"), Message: []byte("message-4")}
	testMessageTest5 = &store.Message{Uid: []byte("test-5"), Message: []byte("message-5")}
	testMessageTest6 = &store.Message{Uid: []byte("test-6"), Message: []byte("message-6")}
	testMessageTest7 = &store.Message{Uid: []byte("test-7"), Message: []byte("message-7")}
	testMessageTest8 = &store.Message{Uid: []byte("test-8"), Message: []byte("message-8")}
	testMessageTest9 = &store.Message{Uid: []byte("test-9"), Message: []byte("message-9")}

	testMessageOther0 = &store.Message{Uid: []byte("other-0"), Message: []byte("message-0")}
	testMessageOther1 = &store.Message{Uid: []byte("other-1"), Message: []byte("message-1")}
	testMessageOther2 = &store.Message{Uid: []byte("other-2"), Message: []byte("message-2")}
	testMessageOther3 = &store.Message{Uid: []byte("other-3"), Message: []byte("message-3")}
	testMessageOther4 = &store.Message{Uid: []byte("other-4"), Message: []byte("message-4")}
	testMessageOther5 = &store.Message{Uid: []byte("other-5"), Message: []byte("message-5")}
	testMessageOther6 = &store.Message{Uid: []byte("other-6"), Message: []byte("message-6")}
	testMessageOther7 = &store.Message{Uid: []byte("other-7"), Message: []byte("message-7")}
	testMessageOther8 = &store.Message{Uid: []byte("other-8"), Message: []byte("message-8")}
	testMessageOther9 = &store.Message{Uid: []byte("other-9"), Message: []byte("message-9")}
)

type GetMessagesTest struct {
	Prefixes [][]byte
	Start    []byte
	Max      int
	Newest   bool
	Expected []*store.Message
}

var tests = []GetMessagesTest{{
	Prefixes: [][]byte{[]byte(PrefixTest), []byte(PrefixOther)},
	Max:      5,
	Expected: []*store.Message{
		testMessageTest0, testMessageTest1, testMessageTest2, testMessageTest3, testMessageTest4,
		testMessageOther0, testMessageOther1, testMessageOther2, testMessageOther3, testMessageOther4,
	},
}, {
	Prefixes: [][]byte{[]byte(PrefixTest), []byte(PrefixOther)},
	Start:    []byte(fmt.Sprintf("%s-%d", PrefixOther, 1)),
	Max:      5,
	Expected: []*store.Message{
		testMessageTest0, testMessageTest1, testMessageTest2, testMessageTest3, testMessageTest4,
		testMessageOther1, testMessageOther2, testMessageOther3, testMessageOther4, testMessageOther5,
	},
}, {
	Prefixes: [][]byte{[]byte(PrefixTest), []byte(PrefixOther)},
	Start:    []byte(fmt.Sprintf("%s-%d", PrefixTest, 1)),
	Max:      5,
	Expected: []*store.Message{
		testMessageTest1, testMessageTest2, testMessageTest3, testMessageTest4, testMessageTest5,
	},
}}

func initTestDb() error {
	testDbPath := filepath.Join(os.TempDir(), fmt.Sprintf("goleveldbtest-%d", os.Getuid()))
	if err := os.RemoveAll(testDbPath); err != nil {
		return fmt.Errorf("error removing old db; %w", err)
	}

	db, err := leveldb.OpenFile(testDbPath, nil)
	if err != nil {
		return fmt.Errorf("error opening level db; %w", err)
	}

	store.SetConn(store.GetConnId(TestTopic, TestShard), db)

	if err := store.SaveMessages(TestTopic, TestShard, []*store.Message{
		testMessageTest0, testMessageTest1, testMessageTest2, testMessageTest3, testMessageTest4,
		testMessageTest5, testMessageTest6, testMessageTest7, testMessageTest8, testMessageTest9,
		testMessageOther0, testMessageOther1, testMessageOther2, testMessageOther3, testMessageOther4,
		testMessageOther5, testMessageOther6, testMessageOther7, testMessageOther8, testMessageOther9,
	}); err != nil {
		return fmt.Errorf("error saving prefix messages; %w", err)
	}

	return nil
}

func TestGetMessage(t *testing.T) {
	if err := initTestDb(); err != nil {
		t.Errorf("error initializing test db; %v", err)
	}

	message, err := store.GetMessage(TestTopic, TestShard, testMessageTest1.Uid)
	if err != nil {
		t.Errorf("error getting message; %v", err)
		return
	}

	if message == nil {
		t.Errorf("message not found")
		return
	}

	if !bytes.Equal(message.Message, testMessageTest1.Message) {
		t.Errorf("message not correct")
		return
	}
}

func TestGetByPrefixes(t *testing.T) {
	if err := initTestDb(); err != nil {
		t.Errorf("error initializing test db; %v", err)
	}

	for _, test := range tests {
		messages, err := store.GetMessages(TestTopic, TestShard, test.Prefixes, test.Start, test.Max, test.Newest)
		if err != nil {
			t.Errorf("error getting message; %v", err)
			return
		}

		if len(messages) != len(test.Expected) {
			t.Errorf("unexpected number of messages: %d, expected %d\n", len(messages), len(test.Expected))
			return
		}

		for i := range messages {
			message := messages[i]
			expected := test.Expected[i]

			if !bytes.Equal(message.Uid, expected.Uid) {
				t.Errorf("unexpected message uid: %s, expected %s\n", message.Uid, expected.Uid)
				return
			}

			if !bytes.Equal(message.Message, expected.Message) {
				t.Errorf("unexpected message: %s, expected %s\n", message.Message, expected.Message)
				return
			}
		}
	}
}
