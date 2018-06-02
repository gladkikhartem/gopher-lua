package main

import (
	"encoding/json"
	"log"

	lua "github.com/gladkikhartem/gopher-lua"
	"github.com/kr/pretty"
)

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

	if err := L.DoString(`print("hello")`); err != nil {
		panic(err)
	}
	d := L.Dump()
	data, _ := json.Marshal(d)
	log.Printf("DUMP: %# v", pretty.Formatter(d))
	log.Printf("LEN: %# v", len(data))
}
