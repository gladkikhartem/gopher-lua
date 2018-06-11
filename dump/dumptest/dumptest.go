package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"

	lua "github.com/gladkikhartem/gopher-lua"
	"github.com/sergi/go-diff/diffmatchpatch"
	ordjson "github.com/virtuald/go-ordered-json"
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

	if err := L.DoString(`pVar = 123
  pVar = 123
  pVar = 123
  pVar = 123
  pVar2 = 124
  pVar3 = 125
  pVar = 155`); err != nil {
		panic(err)
	}
	d := L.Dump()
	data, _ := ordjson.MarshalIndent(d, " ", " ")

	l2, err := lua.LoadDump(d, nil)
	if err != nil {
		panic(err)
	}

	if err := l2.DoString(`
  pVar = 123
  pVar = 123
  pVar2 = 124
  pVar3 = 125
`); err != nil {
		panic(err)
	}
	d2 := l2.Dump()
	data2, _ := ordjson.MarshalIndent(d2, " ", " ")

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(data), string(data2), false)
	fmt.Println(dmp.DiffPrettyText(diffs))
	fmt.Println(len(makeGzip(data2)))

}
