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

func (h Header) IsAuth() bool {
	return (h.b[2] & 0b00000100) != 0
}

func (h Header) RcrsnAvail() bool {
	return (h.b[3] & 0b10000000) != 0
}

func (h Header) GetRCode() (byte, string) {
	code := h.b[3] & 0b00001111
	var msg string
	switch code {
	case 0: msg = "NO_ERR"
	case 1: msg = "FMT_ERR"
	case 2: msg = "SRV_FAIL"
	case 3: msg = "NAME_ERR"
	case 4: msg = "NOT_IMPL"
	case 5: msg = "REFUSED"
	default: msg = "RESERVED"
	}

	return code, msg
}

func (h Header) GetQdCount() uint16 {
	return binary.BigEndian.Uint16(h.b[4:6])
}

func (h Header) GetAnCount() uint16 {
	return binary.BigEndian.Uint16(h.b[6:8])
}

func (h Header) GetNsCount() uint16 {
	return binary.BigEndian.Uint16(h.b[8:10])
}

func (h Header) GetArCount() uint16 {
	return binary.BigEndian.Uint16(h.b[10:])
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
		return code, "OTHER"
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

type ResourceRecord struct {
	b []byte
	name string
	rrType RrType
	class uint16
	ttl uint32
	rdLength uint16
	rData []byte
}

type RrType uint16 

const (
	A RrType = iota+1
	NS
	MD 
	MF
	CNAME
	SOA
	MB
	MG
	MR
	NULL
	WKS
	PTR
	HINFO
	MINFO
	MX
	TXT
)

func (r ResourceRecord) GetName(header []byte) (int, string) {
	prefix := r.b[0] & 0b11000000
	var name string
	var b int
	if prefix == 0b11000000 {
		offset := binary.BigEndian.Uint16([]byte{r.b[0] & 0b00111111, r.b[1]})
		fmt.Println("Pointer!")
		_, name = ParseName(header[offset:])
		b = 16
	} else if prefix == 0 {
		b, name = ParseName(header[2:])	
	} 

	return b, name
}

func ParseResourceRecord(chunk []byte, wholeResponse []byte) (int, ResourceRecord) {
	rr := ResourceRecord{b: chunk}
	b, name := rr.GetName(wholeResponse)
	rrType := binary.BigEndian.Uint16(chunk[b:b+2])
	class := binary.BigEndian.Uint16(chunk[b+2:b+4])
	ttl := binary.BigEndian.Uint32(chunk[b+4:b+8])
	rdLen := binary.BigEndian.Uint16(chunk[b+8:b+10])
	rr.name = name
	rr.rrType = RrType(rrType)
	rr.class = class
	rr.ttl = ttl
	rr.rdLength = rdLen
	return b+10, rr
	// rData :=  //TODO complete
}

func (rr ResourceRecord) String() string {
	return fmt.Sprintf(
		"{name=%s; type=%d; class=%d; ttl=%d; rdLen=%d;}",
		rr.name,
		rr.rrType,
		rr.class,
		rr.ttl,
		rr.rdLength,
	)
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

func ParseName(name []byte) (int, string) {
	fields := make([]string, 0)
	
	size := int(name[0])
	currentIdx := 0
	for size != 0 {
		fields = append(fields, string(name[currentIdx:currentIdx+size+1]))
		currentIdx += size+1
		size = int(name[currentIdx])
	}

	return currentIdx+size, strings.Join(fields, ".")
}

func GetQSection(query []byte) (int, string, string, string) {
	currentIdx, qName := ParseName(query)
	fields := make([]string, 0)
	
	size := int(query[currentIdx+1])
	// currentIdx := 0
	for size != 0 {
		fields = append(fields, string(query[currentIdx:currentIdx+size+1]))
		currentIdx += size+1
		size = int(query[currentIdx])
	}
	
	qType := hex.EncodeToString(query[currentIdx:currentIdx+2])
	qClass := hex.EncodeToString(query[currentIdx+2:currentIdx+4])

	return currentIdx+4, qName, qType, qClass
}

func haveSameId(id1, id2 [2]byte) bool {
	return id1[0] == id2[0] && id1[1] == id2[1]
}

func SendQuery(ipAddr, toFind string) ([]byte, error) {
	ipAddr = ipAddr + ":53"
	conn, err := net.Dial("udp", ipAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	query, err := BuildQuery(toFind)
	if err != nil {
		return nil, err
	}
	fmt.Println("Query: ", hex.EncodeToString(query))
	fmt.Println("Headerless Query: ", hex.EncodeToString(query[12:]))

	_, err = conn.Write(query)
	if err != nil {
		return nil, err
	}

	reply := make([]byte, 512)
	_, err = conn.Read(reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func main() {
	ipAddr := flag.String("a", "", "The address to send the DNS query to")
	toFind := flag.String("u", "", "The lookup address")
	flag.Parse()

	if *ipAddr == "" || *toFind == "" {
		panic("No address provided")
	}
	reply, err := SendQuery(*ipAddr, *toFind)
	handleErr(err)
	replyString := strings.TrimRight(hex.EncodeToString(reply), "0")
	fmt.Println("Headerless Reply: ", strings.TrimPrefix(replyString, hex.EncodeToString(reply[:12])))
	var headerB [12]byte
	copy(headerB[:], reply[:12])
	header := Header{b: headerB}
	fmt.Printf("ID: %s\n", hex.EncodeToString(binary.BigEndian.AppendUint16([]byte{}, header.GetID())))
	fmt.Printf("Is Response: %t\n", header.IsResponse())
	fmt.Printf("Is Auth: %t\n", header.IsAuth())
	_, rcode := header.GetRCode()
	fmt.Printf("RCode: %s\n", rcode)
	fmt.Printf("Recursion Available: %t\n", header.RcrsnAvail())
	fmt.Printf("QDCount: %d\n", header.GetQdCount())
	fmt.Printf("ANCount: %d\n", header.GetAnCount())
	fmt.Printf("NSCount: %d\n", header.GetNsCount())
	fmt.Printf("ARCount: %d\n", header.GetArCount())
	code, opCode := header.OpCode()
	fmt.Printf("Op Code: %s (%b)\n", opCode, code)
	b, qName, qType, qClass := GetQSection(reply[12:])
	fmt.Println("QName:  ", qName)
	fmt.Println("QType:  ", qType)
	fmt.Println("QClass: ", qClass)

	remaining := reply[12+b:]

	// TODO change to 'for b < len()..."
	_, records := ParseResourceRecord(remaining, reply)
	fmt.Println(records)
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
