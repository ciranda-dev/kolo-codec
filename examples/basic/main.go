// Command basic shows how to encode a record with kolocodec and decode it back.
package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	kolocodec "github.com/ciranda-dev/kolo-codec"
)

func main() {
	key := bytes.Repeat([]byte{0xab}, 32)
	var e kolocodec.Encoder
	e.PutUvarint(300)
	e.PutFixed(key)
	e.PutString("hello kolo")
	e.PutBool(true)
	data := e.Bytes()
	fmt.Println("encoded:", hex.EncodeToString(data))

	d := kolocodec.NewDecoder(data)
	seq, err := d.Uvarint()
	if err != nil {
		log.Fatal(err)
	}
	gotKey, err := d.Fixed(32)
	if err != nil {
		log.Fatal(err)
	}
	body, err := d.String(1024)
	if err != nil {
		log.Fatal(err)
	}
	flag, err := d.Bool()
	if err != nil {
		log.Fatal(err)
	}
	if err := d.Done(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("seq:", seq, "same key:", bytes.Equal(key, gotKey), "body:", body, "flag:", flag)

	_, err = kolocodec.NewDecoder([]byte{0x80, 0x00}).Uvarint()
	fmt.Println("overlong uvarint:", err)
	_, err = kolocodec.NewDecoder([]byte{0xff, 0xff, 0xff, 0x7f}).Bytes(1024)
	fmt.Println("huge length:", err)
	_, err = kolocodec.NewDecoder([]byte{0x02, 0xc3, 0x28}).String(1024)
	fmt.Println("invalid UTF-8:", err)
}
