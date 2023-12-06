package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"strings"
)

type Header struct {
	b [12]byte
}

func NewHeader() Header {
	return Header{[12]byte{}}
}

func (h Header) GetID() uint16 {
	return binary.BigEndian.Uint16(h.b[:2])
}

func (h *Header) SetID(value uint16) {
	bytes := binary.BigEndian.AppendUint16([]byte{}, value)
	h.b[0] = bytes[0]
	h.b[1] = bytes[1]
}

func (h *Header) SetFlags(vals uint16) {
	bytes := binary.BigEndian.AppendUint16([]byte{}, vals)
	h.b[2] = bytes[0]
	h.b[3] = bytes[1]
}

func (h *Header) SetCounts(qdCount uint16, anCount uint16, nsCount uint16, arCount uint16) {
	bytes := binary.BigEndian.AppendUint16([]byte{}, qdCount)
	bytes = binary.BigEndian.AppendUint16(bytes, anCount)
	bytes = binary.BigEndian.AppendUint16(bytes, nsCount)
	bytes = binary.BigEndian.AppendUint16(bytes, arCount)
	h.b[4] = bytes[0]
	h.b[5] = bytes[1]
	h.b[6] = bytes[2]
	h.b[7] = bytes[3]
	h.b[8] = bytes[4]
	h.b[9] = bytes[5]
	h.b[10] = bytes[6]
	h.b[11] = bytes[7]

}

func (h Header) IsResponse() bool {
	return 0b10000000 & h.b[3] != 0
}

func (h Header) OpCode() (byte, string) {
	code := (h.b[3] >> 3) | 0b00001111
	dict := map[byte]string{
		0b00000000: "QUERY",
		0b00000001: "IQUERY",
		0b00000010: "STATUS",
	}
	if kind, ok := dict[code]; ok {
		return code, kind
	} else {
		return code, ""
	}
}

func NewQuestionSection(qname string, qtype uint16, qclass uint16) []byte {
	result := encodeName(qname)
	result = binary.BigEndian.AppendUint16(result, qtype)
	result = binary.BigEndian.AppendUint16(result, qclass)
	return result
}

func encodeName(qname string) []byte {
	result := make([]byte, 0)
	fields := strings.Split(qname, ".")
	for _, f := range fields {
		l := byte(0)
		for range f {
			l++
		}
		result = append(result, l)
		result = append(result, []byte(f)...)
	}
	result = append(result, 0)
	return result
}

type field struct {
	size byte
	value []byte
}

func BuildQuery(name string) ([]byte, error) {

	vBytes := make([]byte, 2)
	_, err := rand.Read(vBytes)
	if err != nil {
		return nil, err
	}

	h := NewHeader()
	h.SetID(binary.BigEndian.Uint16(vBytes))
	h.SetFlags(0b0000000100000000)
	h.SetCounts(1,0,0,0)

	q := NewQuestionSection(name,1,1)
	msg := append(h.b[:], q...)

	return msg, nil
}

func haveSameId(id1, id2 [2]byte) bool {
	return id1[0] == id2[0] && id1[1] == id2[1]
}

func main() {
	ipAddr := flag.String("a", "", "The address to send the DNS query to")
	toFind := flag.String("u", "", "The lookup address")
	flag.Parse()

	if *ipAddr == "" || *toFind == "" {
		panic("No address provided")
	}
	*ipAddr = *ipAddr + ":53"
	conn, err := net.Dial("udp", *ipAddr)
	handleErr(err)
	defer conn.Close()

	query, err := BuildQuery(*toFind)
	handleErr(err)

	fmt.Println("ID: ", hex.EncodeToString(query[:2]))
	fmt.Println("Query: ", hex.EncodeToString(query))
	_, err = conn.Write(query)
	handleErr(err)

	reply := make([]byte, 1024)
	_, err = conn.Read(reply)
	handleErr(err)
	fmt.Println("Reply: ", hex.EncodeToString(reply))
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
