package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func TestGoogle(t *testing.T) {
	vBytes := make([]byte, 2)
	_, err := rand.Read(vBytes)
	if err != nil {
		t.Fatalf(err.Error())
	}

	h := NewHeader()
	h.SetID(binary.BigEndian.Uint16(vBytes))
	h.SetFlags(0b0000000100000000)
	h.SetCounts(1,0,0,0)

	q := NewQuestionSection("dns.google.com",1,1)
	msg := append(h.b[:], q...)

	actual := hex.EncodeToString(msg)
	id := hex.EncodeToString(vBytes)
	expected := id + "0100000100000000000003646e7306676f6f676c6503636f6d0000010001"
	if actual != expected {
		
		t.Fatalf("\nExpected %s,\nGot      %s", expected, actual)
	}
}
