package controller

import (
	"testing"
)

func TestNewReturnsController(t *testing.T) {
	c := New("foo")
	if c.config.Namespace != "foo" {
		t.Fatalf("Expected foo as namespace, got %s instead", c.config.Namespace)
	}
}

// func TestWatchCanariesReturnsChannels(t *testing.T) {
// 	evCh, erCh := New("foo").watchCanaries()
//
// 	event := &Event{}
//
// 	for ev := range evCh {
// 		if ev != event {
// 			t.Fatal("No event equality")
// 		}
// 	}
//
// 	err := fmt.Errorf("lol")
//
// 	for er := range erCh {
// 		if er != err {
// 			t.Fatal("No error equality")
// 		}
// 	}
// }
