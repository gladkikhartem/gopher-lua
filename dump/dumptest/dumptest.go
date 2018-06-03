package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"log"

	lua "github.com/gladkikhartem/gopher-lua"
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
		CallStackSize: 16,
		RegistrySize:  128,
		SkipOpenLibs:  true,
	})
	defer L.Close()

	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		//  {lua.LoadLibName, lua.OpenPackage}, // Must be first
		{lua.BaseLibName, lua.OpenBase},
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

	if err := L.DoString(`pVar = 123
print("hello")
print(pVar)`); err != nil {
		panic(err)
	}
	d := L.Dump()
	data, _ := json.MarshalIndent(d, " ", " ")
	//log.Printf("DUMP: %v", string(data))
	log.Printf("LEN: %# v", len(makeGzip(data)))

	l2, err := lua.LoadDump(d)
	if err != nil {
		panic(err)
	}
	d2 := l2.Dump()
	data2, _ := json.MarshalIndent(d2, " ", " ")
	//log.Printf("DUMP: %v", string(data2))
	log.Printf("LEN: %# v", len(makeGzip(data2)))
	if err := l2.DoString(`print("hello2")
  print(pVar)`); err != nil {
		panic(err)
	}
}
