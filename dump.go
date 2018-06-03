package lua

import (
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strconv"

	"github.com/gladkikhartem/gopher-lua/dump"
)

/* TODO LIST

Function Proto constans disappear after 2nd dump
Restore STDLIB GFunctions
Registry value dissapeared !?
{
"Type": 2,
"Number": 123
}
_G Table losing  globals information about STDLIB and other...
table with _G key is lost


TODO: save original pointer adresses to we could use them
to track values in diff across the time


*/
func (s *LState) Dump() dump.Data {
	d := dumper{
		d: dump.Data{
			G:               make(map[dump.Ptr]*dump.Global), //for consistency
			States:          make(map[dump.Ptr]*dump.State),
			Tables:          make(map[dump.Ptr]*dump.Table),
			CallFrames:      make(map[dump.Ptr]*dump.CallFrame),
			CallFrameStacks: make(map[dump.Ptr]*dump.CallFrameStack),
			Registries:      make(map[dump.Ptr]*dump.Registry),
			Functions:       make(map[dump.Ptr]*dump.Function),
			GFunctions:      make(map[dump.Ptr]*dump.GFunction),
			FunctionProtos:  make(map[dump.Ptr]*dump.FunctionProto),
			DbgLocalInfos:   make(map[dump.Ptr]*dump.DbgLocalInfo),
			Upvalues:        make(map[dump.Ptr]*dump.Upvalue),
		},
	}
	d.dumpState(s)
	return d.d
}

type dumper struct {
	d dump.Data
}

func (d *dumper) dumpLValue(lv LValue) dump.Value {
	switch v := lv.(type) {
	case LBool:
		return dump.Value{Type: int(LTBool), Bool: bool(v)}
	case LNumber:
		return dump.Value{Type: int(LTNumber), Number: float64(v)}
	case LString:
		return dump.Value{Type: int(LTString), String: string(v)}
	case *LNilType:
		return dump.Value{Type: int(LTNil)}
	case *LFunction:
		return dump.Value{Type: int(LTFunction), Ptr: d.dumpFunction(v)}
	case *LState:
		return dump.Value{Type: int(LTThread), Ptr: d.dumpState(v)}
	case *LTable:
		return dump.Value{Type: int(LTTable), Ptr: d.dumpTable(v)}
	}
	return dump.Value{Type: int(LTNil)}
}

func (d *dumpLoader) loadLValue(vv dump.Value) LValue {
	switch LValueType(vv.Type) {
	case LTBool:
		v := LBool(vv.Bool)
		return &v
	case LTNumber:
		v := LNumber(vv.Number)
		return &v
	case LTString:
		v := LString(vv.String)
		return &v
	case LTNil:
		v := LNilType{}
		return &v
	case LTFunction:
		return d.loadFunction(vv.Ptr)
	case LTThread:
		return d.loadState(vv.Ptr)
	case LTTable:
		return d.loadTable(vv.Ptr)
	}
	d.Errors = append(d.Errors, fmt.Errorf("unsupported type: %v", vv.Type))
	v := LNilType{}
	return &v
}

func (d *dumper) dumpState(s *LState) (ptr dump.Ptr) {
	if s == nil {
		return
	}
	ptr = getPtr(s)
	_, ok := d.d.States[ptr]
	if ok {
		return
	}
	ds := dump.State{}
	d.d.States[ptr] = &ds // avoid infinite recursion
	ds.G = d.dumpGlobal(s.G)
	ds.Parent = d.dumpState(s.Parent)
	ds.Env = d.dumpTable(s.Env)
	ds.Options = dump.Options{
		CallStackSize:       s.Options.CallStackSize,
		RegistrySize:        s.Options.RegistrySize,
		SkipOpenLibs:        s.Options.SkipOpenLibs,
		IncludeGoStackTrace: s.Options.IncludeGoStackTrace,
	}
	ds.Stop = s.stop
	ds.Reg = d.dumpRegistry(s.reg)
	ds.Stack = d.dumpCallFrameStack(s.stack)
	ds.CurrentFrame = d.dumpCallFrame(s.currentFrame)
	ds.Wrapped = s.wrapped
	ds.UVCache = d.dumpUpvalue(s.uvcache)
	ds.HasErrorFunc = s.hasErrorFunc
	ds.Dead = s.Dead
	return
}

func (d *dumpLoader) loadState(ptr dump.Ptr) *LState {
	if ptr == "" {
		return nil
	}
	s := d.States[ptr]
	ds := d.Data.States[ptr]
	id := fmt.Sprint("lstate-", ptr)
	if d.Loaded[id] {
		return s
	}
	d.Loaded[id] = true

	s.G = d.loadGlobal(ds.G)
	s.Parent = d.loadState(ds.Parent)
	s.Env = d.loadTable(ds.Env)
	s.Options = Options{
		CallStackSize:       ds.Options.CallStackSize,
		RegistrySize:        ds.Options.RegistrySize,
		SkipOpenLibs:        ds.Options.SkipOpenLibs,
		IncludeGoStackTrace: ds.Options.IncludeGoStackTrace,
	}
	s.stop = ds.Stop
	s.reg = d.loadRegistry(ds.Reg)
	s.stack = d.loadCallFrameStack(ds.Stack)
	s.currentFrame = d.loadCallFrame(ds.CurrentFrame)
	s.wrapped = ds.Wrapped
	s.uvcache = d.loadUpvalue(ds.UVCache)
	s.hasErrorFunc = ds.HasErrorFunc
	s.Dead = ds.Dead
	s.mainLoop = mainLoop
	s.alloc = newAllocator(32)
	return s
}

func (d *dumper) dumpGlobal(g *Global) (ptr dump.Ptr) {
	if g == nil {
		return
	}
	ptr = getPtr(g)
	_, ok := d.d.G[ptr]
	if ok {
		return
	}
	dg := dump.Global{}
	d.d.G[ptr] = &dg // avoid infinite recursion
	dg.MainThread = d.dumpState(g.MainThread)
	dg.CurrentThread = d.dumpState(g.CurrentThread)
	dg.Registry = d.dumpTable(g.Registry)
	dg.Global = d.dumpTable(g.Global)
	dg.Gccount = g.gccount
	dg.BuiltinMts = map[string]dump.Value{}
	for k, v := range g.builtinMts {
		dg.BuiltinMts[fmt.Sprint(k)] = d.dumpLValue(v)
	}
	return
}

func (d *dumpLoader) loadGlobal(ptr dump.Ptr) *Global {
	if ptr == "" {
		return nil
	}
	g := d.G[ptr]
	dg := d.Data.G[ptr]
	id := fmt.Sprint("global-", ptr)
	if d.Loaded[id] {
		return g
	}
	d.Loaded[id] = true

	g.MainThread = d.loadState(dg.MainThread)
	g.CurrentThread = d.loadState(dg.CurrentThread)
	g.Registry = d.loadTable(dg.Registry)
	g.Global = d.loadTable(dg.Global)
	g.gccount = dg.Gccount
	g.builtinMts = map[int]LValue{}
	for k, v := range dg.BuiltinMts {
		intk, _ := strconv.Atoi(k)
		g.builtinMts[intk] = d.loadLValue(v)
	}

	return g
}

func (d *dumper) dumpCallFrame(cf *callFrame) (ptr dump.Ptr) {
	if cf == nil {
		return
	}
	ptr = getPtr(cf)
	_, ok := d.d.CallFrames[ptr]
	if ok {
		return
	}
	dcf := dump.CallFrame{}
	d.d.CallFrames[ptr] = &dcf // avoid infinite recursion

	dcf.Idx = cf.Idx
	dcf.Fn = d.dumpFunction(cf.Fn)
	dcf.Parent = d.dumpCallFrame(cf.Parent)
	dcf.Pc = cf.Pc
	dcf.Base = cf.Base
	dcf.LocalBase = cf.LocalBase
	dcf.ReturnBase = cf.ReturnBase
	dcf.NArgs = cf.NArgs
	dcf.NRet = cf.NRet
	dcf.TailCall = cf.TailCall
	return
}

func (d *dumpLoader) loadCallFrame(ptr dump.Ptr) *callFrame {
	if ptr == "" {
		return nil
	}
	cf := d.CallFrames[ptr]
	dcf := d.Data.CallFrames[ptr]
	id := fmt.Sprint("callframe-", ptr)
	if d.Loaded[id] {
		return cf
	}
	d.Loaded[id] = true

	cf.Idx = dcf.Idx
	cf.Fn = d.loadFunction(dcf.Fn)
	cf.Parent = d.loadCallFrame(dcf.Parent) // TODO: fix pointer to callFrames[] array ???
	cf.Pc = dcf.Pc
	cf.Base = dcf.Base
	cf.LocalBase = dcf.LocalBase
	cf.ReturnBase = dcf.ReturnBase
	cf.NArgs = dcf.NArgs
	cf.NRet = dcf.NRet
	cf.TailCall = dcf.TailCall

	return cf
}

func (d *dumper) dumpRegistry(r *registry) (ptr dump.Ptr) {
	if r == nil {
		return
	}
	ptr = getPtr(r)
	_, ok := d.d.Registries[ptr]
	if ok {
		return
	}
	dr := dump.Registry{}
	d.d.Registries[ptr] = &dr // avoid infinite recursion

	dr.Top = r.top
	dr.Array = make([]dump.Value, len(r.array))
	for i, v := range r.array {
		dr.Array[i] = d.dumpLValue(v)
	}
	return
}

func (d *dumpLoader) loadRegistry(ptr dump.Ptr) *registry {
	if ptr == "" {
		return nil
	}
	r := d.Registries[ptr]
	dr := d.Data.Registries[ptr]
	id := fmt.Sprint("registry-", ptr)
	if d.Loaded[id] {
		return r
	}
	d.Loaded[id] = true

	r.top = dr.Top
	r.array = make([]LValue, len(dr.Array))
	for i, v := range dr.Array {
		r.array[i] = d.loadLValue(v)
	}

	return r
}

func (d *dumper) dumpCallFrameStack(cfs *callFrameStack) (ptr dump.Ptr) {
	if cfs == nil {
		return
	}
	ptr = getPtr(cfs)
	_, ok := d.d.CallFrameStacks[ptr]
	if ok {
		return
	}
	dcfs := dump.CallFrameStack{}
	d.d.CallFrameStacks[ptr] = &dcfs // avoid infinite recursion

	dcfs.Sp = cfs.sp
	dcfs.Array = make([]dump.Ptr, len(cfs.array))
	for i, v := range cfs.array {
		dcfs.Array[i] = d.dumpCallFrame(&v)
	}
	return
}

func (d *dumpLoader) loadCallFrameStack(ptr dump.Ptr) *callFrameStack {
	if ptr == "" {
		return nil
	}
	cfs := d.CallFrameStacks[ptr]
	dcfs := d.Data.CallFrameStacks[ptr]
	id := fmt.Sprint("callframestack-", ptr)
	if d.Loaded[id] {
		return cfs
	}
	d.Loaded[id] = true

	cfs.sp = dcfs.Sp
	cfs.array = make([]callFrame, len(dcfs.Array))
	for i, v := range dcfs.Array {
		cfs.array[i] = *d.loadCallFrame(v) // TODO: modify pointer of call Frame parent
	}

	return cfs
}

func (d *dumper) dumpUpvalue(uv *Upvalue) (ptr dump.Ptr) {
	if uv == nil {
		return
	}
	ptr = getPtr(uv)
	_, ok := d.d.Upvalues[ptr]
	if ok {
		return
	}
	duv := dump.Upvalue{}
	d.d.Upvalues[ptr] = &duv // avoid infinite recursion

	duv.Next = d.dumpUpvalue(uv.next)
	duv.Reg = d.dumpRegistry(uv.reg)
	duv.Index = uv.index
	duv.Value = d.dumpLValue(uv.value)
	duv.Closed = uv.closed
	return
}

func (d *dumpLoader) loadUpvalue(ptr dump.Ptr) *Upvalue {
	if ptr == "" {
		return nil
	}
	uv := d.Upvalues[ptr]
	duv := d.Data.Upvalues[ptr]
	id := fmt.Sprint("upvalue-", ptr)
	if d.Loaded[id] {
		return uv
	}
	d.Loaded[id] = true

	uv.next = d.loadUpvalue(duv.Next)
	uv.reg = d.loadRegistry(duv.Reg)
	uv.index = duv.Index
	uv.value = d.loadLValue(duv.Value)
	uv.closed = duv.Closed

	return uv
}

func (d *dumper) dumpFunction(f *LFunction) (ptr dump.Ptr) {
	if f == nil {
		return
	}
	ptr = getPtr(f)
	_, ok := d.d.Upvalues[ptr]
	if ok {
		return
	}
	df := dump.Function{}
	d.d.Functions[ptr] = &df // avoid infinite recursion

	df.IsG = f.IsG
	df.Env = d.dumpTable(f.Env)
	df.Proto = d.dumpFunctionProto(f.Proto)
	df.GFunction = d.dumpGFunction(f.GFunction)
	df.Upvalues = make([]dump.Ptr, len(f.Upvalues))
	for i, v := range f.Upvalues {
		df.Upvalues[i] = d.dumpUpvalue(v)
	}
	return
}

func (d *dumpLoader) loadFunction(ptr dump.Ptr) *LFunction {
	if ptr == "" {
		return nil
	}
	f := d.Functions[ptr]
	df := d.Data.Functions[ptr]
	id := fmt.Sprint("function-", ptr)
	if d.Loaded[id] {
		return f
	}
	d.Loaded[id] = true

	f.IsG = df.IsG
	f.Env = d.loadTable(df.Env)
	f.Proto = d.loadFunctionProto(df.Proto)
	f.GFunction = d.loadGFunction(df.GFunction) // TODO: CAREFUL
	f.Upvalues = make([]*Upvalue, len(df.Upvalues))
	for i, v := range df.Upvalues {
		f.Upvalues[i] = d.loadUpvalue(v)
	}
	return f
}

func (d *dumper) dumpFunctionProto(fp *FunctionProto) (ptr dump.Ptr) {
	if fp == nil {
		return
	}
	ptr = getPtr(fp)
	_, ok := d.d.FunctionProtos[ptr]
	if ok {
		return
	}
	dfp := dump.FunctionProto{}
	d.d.FunctionProtos[ptr] = &dfp // avoid infinite recursion

	dfp.SourceName = fp.SourceName
	dfp.LineDefined = fp.LineDefined
	dfp.LastLineDefined = fp.LastLineDefined
	dfp.NumUpvalues = fp.NumUpvalues
	dfp.NumParameters = fp.NumParameters
	dfp.IsVarArg = fp.IsVarArg
	dfp.NumUsedRegisters = fp.NumUsedRegisters
	dfp.Code = fp.Code                             // []uint32
	dfp.DbgSourcePositions = fp.DbgSourcePositions //[]int
	dfp.DbgUpvalues = fp.DbgUpvalues               //[]string
	dfp.StringConstants = fp.stringConstants       //[]string

	dfp.Constants = make([]dump.Value, len(fp.Constants))
	for i, v := range fp.Constants {
		dfp.Constants[i] = d.dumpLValue(v)
	}
	dfp.FunctionPrototypes = make([]dump.Ptr, len(fp.FunctionPrototypes))
	for i, v := range fp.FunctionPrototypes {
		dfp.FunctionPrototypes[i] = d.dumpFunctionProto(v)
	}
	dfp.DbgCalls = make([]dump.DbgCall, len(fp.DbgCalls))
	for i, v := range fp.DbgCalls {
		dfp.DbgCalls[i] = dump.DbgCall{
			Pc:   v.Pc,
			Name: v.Name}
	}
	dfp.DbgLocals = make([]dump.Ptr, len(fp.DbgLocals))
	for i, v := range fp.DbgLocals {
		dfp.DbgLocals[i] = d.dumpDbgLocalInfo(v)
	}
	return
}

func (d *dumpLoader) loadFunctionProto(ptr dump.Ptr) *FunctionProto {
	if ptr == "" {
		return nil
	}
	fp := d.FunctionProtos[ptr]
	dfp := d.Data.FunctionProtos[ptr]
	id := fmt.Sprint("functionproto-", ptr)
	if d.Loaded[id] {
		return fp
	}
	d.Loaded[id] = true

	fp.SourceName = dfp.SourceName
	fp.LineDefined = dfp.LineDefined
	fp.LastLineDefined = dfp.LastLineDefined
	fp.NumUpvalues = dfp.NumUpvalues
	fp.NumParameters = dfp.NumParameters
	fp.IsVarArg = dfp.IsVarArg
	fp.NumUsedRegisters = dfp.NumUsedRegisters
	fp.Code = dfp.Code                             // []uint32
	fp.DbgSourcePositions = dfp.DbgSourcePositions //[]int
	fp.DbgUpvalues = dfp.DbgUpvalues               //[]string
	fp.stringConstants = dfp.StringConstants       //[]string

	fp.Constants = make([]LValue, len(dfp.Constants))
	for i, v := range dfp.Constants {
		fp.Constants[i] = d.loadLValue(v)
	}
	fp.FunctionPrototypes = make([]*FunctionProto, len(dfp.FunctionPrototypes))
	for i, v := range dfp.FunctionPrototypes {
		fp.FunctionPrototypes[i] = d.loadFunctionProto(v)
	}
	fp.DbgCalls = make([]DbgCall, len(dfp.DbgCalls))
	for i, v := range dfp.DbgCalls {
		fp.DbgCalls[i] = DbgCall{
			Pc:   v.Pc,
			Name: v.Name}
	}
	fp.DbgLocals = make([]*DbgLocalInfo, len(dfp.DbgLocals))
	for i, v := range dfp.DbgLocals {
		fp.DbgLocals[i] = d.loadDbgLocalInfo(v)
	}

	return fp
}

func (d *dumper) dumpDbgLocalInfo(li *DbgLocalInfo) (ptr dump.Ptr) {
	if li == nil {
		return
	}
	ptr = getPtr(li)
	_, ok := d.d.DbgLocalInfos[ptr]
	if ok {
		return
	}
	dli := dump.DbgLocalInfo{}
	d.d.DbgLocalInfos[ptr] = &dli // avoid infinite recursion

	dli.EndPc = li.EndPc
	dli.Name = li.Name
	dli.StartPc = li.StartPc
	return
}

func (d *dumpLoader) loadDbgLocalInfo(ptr dump.Ptr) *DbgLocalInfo {
	if ptr == "" {
		return nil
	}
	li := d.DbgLocalInfos[ptr]
	dli := d.Data.DbgLocalInfos[ptr]
	id := fmt.Sprint("dbglocal-", ptr)
	if d.Loaded[id] {
		return li
	}
	d.Loaded[id] = true

	li.EndPc = dli.EndPc
	li.Name = dli.Name
	li.StartPc = dli.StartPc

	return li
}

func (d *dumper) dumpTable(t *LTable) (ptr dump.Ptr) {
	if t == nil {
		return
	}
	ptr = getPtr(t)
	_, ok := d.d.Tables[ptr]
	if ok {
		return
	}
	dt := dump.Table{}
	d.d.Tables[ptr] = &dt // avoid infinite recursion

	dt.Metatable = d.dumpLValue(t.Metatable)
	dt.Array = make([]dump.Value, len(t.array))
	for i, v := range t.array {
		dt.Array[i] = d.dumpLValue(v)
	}
	dt.Dict = []dump.VV{}
	for k, v := range t.dict {
		dt.Dict = append(dt.Dict, dump.VV{
			Key:   d.dumpLValue(k),
			Value: d.dumpLValue(v)})
	}
	dt.Strdict = map[string]dump.Value{}
	for k, v := range t.strdict {
		dt.Strdict[k] = d.dumpLValue(v)
	}
	dt.Keys = make([]dump.Value, len(t.keys))
	for i, v := range t.keys {
		dt.Keys[i] = d.dumpLValue(v)
	}
	dt.K2i = []dump.VI{}
	for i, v := range t.k2i {
		dt.K2i = append(dt.K2i, dump.VI{Key: d.dumpLValue(i), Value: v})
	}
	return
}

func (d *dumpLoader) loadTable(ptr dump.Ptr) *LTable {
	if ptr == "" {
		return nil
	}
	t, ok := d.Tables[ptr]
	if !ok {
		log.Printf("table not found! %v %v %#v", ok, ptr, d.Tables)
	}
	dt := d.Data.Tables[ptr]
	id := fmt.Sprint("table-", ptr)
	if d.Loaded[id] {
		return t
	}
	d.Loaded[id] = true

	t.Metatable = d.loadLValue(dt.Metatable)
	t.array = make([]LValue, len(dt.Array))
	for i, v := range dt.Array {
		t.array[i] = d.loadLValue(v)
	}
	t.dict = map[LValue]LValue{}
	for _, pair := range dt.Dict {
		k := d.loadLValue(pair.Key)
		v := d.loadLValue(pair.Value)
		t.dict[k] = v
	}

	//k2i     map[LValue]int
	t.strdict = map[string]LValue{}
	for k, v := range dt.Strdict {
		t.strdict[k] = d.loadLValue(v)
	}
	t.keys = make([]LValue, len(dt.Keys))
	for i, v := range dt.Keys {
		t.keys[i] = d.loadLValue(v)
	}
	t.k2i = map[LValue]int{}
	for _, pair := range dt.K2i {
		k := d.loadLValue(pair.Key)
		t.k2i[k] = pair.Value
	}
	return t
}

func (d *dumper) dumpGFunction(gf LGFunction) (ptr dump.Ptr) {
	if gf == nil {
		return
	}
	ptr = getPtr(gf)
	_, ok := d.d.GFunctions[ptr]
	if ok {
		return
	}
	dgf := dump.GFunction{}
	d.d.GFunctions[ptr] = &dgf // avoid infinite recursion

	f := runtime.FuncForPC(reflect.ValueOf(gf).Pointer())
	dgf.Name = f.Name()
	dgf.File, dgf.Line = f.FileLine(reflect.ValueOf(gf).Pointer())
	return
}

func (d *dumpLoader) loadGFunction(ptr dump.Ptr) LGFunction {
	if ptr == "" {
		return nil
	}
	gf := d.GFunctions[ptr]
	dgf := d.Data.GFunctions[ptr]
	id := fmt.Sprint("gfunction-", ptr)
	if d.Loaded[id] {
		return gf
	}
	d.Loaded[id] = true

	if dgf.Name == "github.com/gladkikhartem/gopher-lua.basePrint" {
		return basePrint
	}
	return func(*LState) int {
		log.Printf("TODO: LOAD EXTERNAL FUNCTION %v \n %v:%v ", dgf.Name, dgf.File, dgf.Line)
		return 0
	}
	//f := runtime.FuncForPC(reflect.ValueOf(gf).Pointer())
	//dgf.Name = f.Name()
	//dgf.File, dgf.Line = f.FileLine(reflect.ValueOf(gf).Pointer())

	return gf
}

func getPtr(ptr interface{}) dump.Ptr {
	v := reflect.ValueOf(ptr)
	if v.IsNil() {
		return "nil"
	}
	return dump.Ptr(fmt.Sprint(v.Type(), "-", v.Pointer()))
}

type dumpLoader struct {
	Errors          []error
	Data            dump.Data
	Loaded          map[string]bool      // ids of objects loaded
	G               map[dump.Ptr]*Global //for consistency
	States          map[dump.Ptr]*LState
	Tables          map[dump.Ptr]*LTable
	CallFrames      map[dump.Ptr]*callFrame
	CallFrameStacks map[dump.Ptr]*callFrameStack
	Registries      map[dump.Ptr]*registry
	Functions       map[dump.Ptr]*LFunction
	GFunctions      map[dump.Ptr]LGFunction
	FunctionProtos  map[dump.Ptr]*FunctionProto
	DbgLocalInfos   map[dump.Ptr]*DbgLocalInfo
	Upvalues        map[dump.Ptr]*Upvalue
}

func (d *dumpLoader) init() {
	d.Loaded = make(map[string]bool)
	d.G = make(map[dump.Ptr]*Global) //for consistency
	d.States = make(map[dump.Ptr]*LState)
	d.Tables = make(map[dump.Ptr]*LTable)
	d.CallFrames = make(map[dump.Ptr]*callFrame)
	d.CallFrameStacks = make(map[dump.Ptr]*callFrameStack)
	d.Registries = make(map[dump.Ptr]*registry)
	d.Functions = make(map[dump.Ptr]*LFunction)
	d.GFunctions = make(map[dump.Ptr]LGFunction)
	d.FunctionProtos = make(map[dump.Ptr]*FunctionProto)
	d.DbgLocalInfos = make(map[dump.Ptr]*DbgLocalInfo)
	d.Upvalues = make(map[dump.Ptr]*Upvalue)
	for k := range d.Data.G {
		d.G[k] = &Global{}
	}
	for k := range d.Data.States {
		d.States[k] = &LState{}
	}
	for k := range d.Data.Tables {
		d.Tables[k] = &LTable{}
	}
	for k := range d.Data.CallFrames {
		d.CallFrames[k] = &callFrame{}
	}
	for k := range d.Data.CallFrameStacks {
		d.CallFrameStacks[k] = &callFrameStack{}
	}
	for k := range d.Data.Registries {
		d.Registries[k] = &registry{}
	}
	for k := range d.Data.Functions {
		d.Functions[k] = &LFunction{}
	}
	for k := range d.Data.FunctionProtos {
		d.FunctionProtos[k] = &FunctionProto{}
	}
	for k := range d.Data.DbgLocalInfos {
		d.DbgLocalInfos[k] = &DbgLocalInfo{}
	}
	for k := range d.Data.Upvalues {
		d.Upvalues[k] = &Upvalue{}
	}
}

func (d *dumpLoader) load() error {
	for _, g := range d.Data.G {
		d.loadState(g.MainThread)
		return nil
	}
	return fmt.Errorf("main thread not found")
}

func LoadDump(d dump.Data) (*LState, error) {
	ld := dumpLoader{Data: d}
	ld.init()
	err := ld.load()
	if err != nil {
		return nil, err
	}
	for _, v := range ld.G {
		return v.CurrentThread, nil
	}
	return nil, fmt.Errorf("Global not found")
}
