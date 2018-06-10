package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"time"

	lua "github.com/gladkikhartem/gopher-lua"
	"github.com/gladkikhartem/gopher-lua/dump"
)

func makeGzip(data []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(data)
	if err != nil {
		log.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func main() {
	L := lua.NewState(lua.Options{
		RegistrySize:        128,
		CallStackSize:       64,
		SkipOpenLibs:        true,
		IncludeGoStackTrace: true,
	})
	defer L.Close()

	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		//  {lua.LoadLibName, lua.OpenPackage}, // Must be first
		//  {lua.BaseLibName, lua.OpenBase},
		//  {lua.TabLibName, lua.OpenTable},
	} {
		if err := L.CallByParam(lua.P{
			Fn:      L.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			panic(err)
		}
	}

	if err := L.DoString(`pVar = 123`); err != nil {
		panic(err)
	}
	d := L.Dump()
	data, _ := json.MarshalIndent(d, " ", " ")
	//log.Printf("LEN: %# v", len(data))
	//log.Printf("GZIP: %# v", len(makeGzip(data)))
	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	encoder.Encode(d)
	fmt.Printf("DUMP: %v\n", string(b.String()))
	fmt.Printf("BLEN: %v\n", b.Len())

	return
	l2, err := lua.LoadDump(d, nil)
	if err != nil {
		panic(err)
	}
	if err := l2.DoString(`print("hello2")
print(pVar)
`); err != nil {
		panic(err)
	}
	var d2 dump.Data
	t := time.Now()
	for i := 0; i < 100; i++ {
		d2 = l2.Dump()
	}
	log.Printf("average dump: %v", float64(time.Since(t).Nanoseconds()/1000000)/100)
	data2, _ := json.MarshalIndent(d2, " ", " ")
	//fmt.Printf("DUMP2: %v\n", string(data2))
	fmt.Printf("LEN: %# v\n", len(data))
	fmt.Printf("GZIP: %# v\n", len(makeGzip(data)))
	fmt.Printf("LEN2: %# v\n", len(data2))
	fmt.Printf("GZIP2: %# v\n", len(makeGzip(data2)))
}
