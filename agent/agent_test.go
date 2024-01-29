package main

import (
	"fmt"
	"os"
	"testing"
)

func TestWriteLog(t *testing.T) {
	f, err := os.OpenFile("test.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	for i := 0; i < 100000; i++ {
		_, err = f.WriteString(fmt.Sprintf("test log %d\n", i))
		if err != nil {
			t.Fatal(err)
		}
	}
}
