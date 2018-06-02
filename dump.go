package lua

import (
	"fmt"
	"reflect"
	"runtime"

	"github.com/gladkikhartem/gopher-lua/dump"
)

func (s *LState) Dump() dump.Data {
	d := dumper{
		d: dump.Data{
			G:               make(map[string]*dump.Global), //for consistency
			States:          make(map[string]*dump.State),
			Tables:          make(map[string]*dump.Table),
			CallFrames:      make(map[string]*dump.CallFrame),
			CallFrameStacks: make(map[string]*dump.CallFrameStack),
			Registries:      make(map[string]*dump.Registry),
			Functions:       make(map[string]*dump.Function),
			GFunctions:      make(map[string]*dump.GFunction),
			FunctionProtos:  make(map[string]*dump.FunctionProto),
			DbgLocalInfos:   make(map[string]*dump.DbgLocalInfo),
			Upvalues:        make(map[string]*dump.Upvalue),
		},
	}
	d.dumpState(s)
	return d.d
}

type dumper struct {
	d dump.Data
}

func (d dumper) dumpLValue(lv LValue) dump.Value {
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
		return dump.Value{Type: int(LTFunction), Ptr: ptrToString(v)}
	case *LState:
		return dump.Value{Type: int(LTThread), Ptr: ptrToString(v)}
	case *LTable:
		return dump.Value{Type: int(LTTable), Ptr: ptrToString(v)}
	}
	return dump.Value{Type: int(LTNil)}
}

func (d dumpLoader) loadLValue(vv *dump.Value) (LValue, error) {
	switch LValueType(vv.Type) {
	case LTBool:
		v := LBool(vv.Bool)
		return &v, nil
	case LTNumber:
		v := LNumber(vv.Number)
		return &v, nil
	case LTString:
		v := LString(vv.String)
		return &v, nil
	case LTNil:
		v := LNilType{}
		return &v, nil
	case LTFunction:
		v, err := d.loadFunction(vv.Ptr)
		return v, err
	case LTThread:
		v, err := d.loadState(vv.Ptr)
		return v, err
	case LTTable:
		v, err := d.loadTable(vv.Ptr)
		return v, err
	}
	return nil, fmt.Errorf("unsupported type: %v", vv.Type)
}

func (d dumper) dumpState(s *LState) (p dump.Ptr) {
	if s == nil {
		return
	}
	p = dump.Ptr{Type: dump.TState, Ptr: ptrToString(s)}
	_, ok := d.d.States[p.Ptr]
	if ok {
		return
	}
	ds := dump.State{}
	d.d.States[p.Ptr] = &ds // avoid infinite recursion
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

func (d dumpLoader) loadState(ptr string) (*LState, error) {
	return nil, nil
}

func (d dumper) dumpGlobal(g *Global) (p dump.Ptr) {
	if g == nil {
		return
	}
	p = dump.Ptr{Type: dump.TGlobal, Ptr: ptrToString(g)}
	_, ok := d.d.G[p.Ptr]
	if ok {
		return
	}
	dg := dump.Global{}
	d.d.G[p.Ptr] = &dg // avoid infinite recursion
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

func (d dumper) dumpCallFrame(cf *callFrame) (p dump.Ptr) {
	if cf == nil {
		return
	}
	p = dump.Ptr{Type: dump.TCallFrame, Ptr: ptrToString(cf)}
	_, ok := d.d.CallFrames[p.Ptr]
	if ok {
		return
	}
	dcf := dump.CallFrame{}
	d.d.CallFrames[p.Ptr] = &dcf // avoid infinite recursion

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

func (d dumper) dumpRegistry(r *registry) (p dump.Ptr) {
	if r == nil {
		return
	}
	p = dump.Ptr{Type: dump.TRegistry, Ptr: ptrToString(r)}
	_, ok := d.d.Registries[p.Ptr]
	if ok {
		return
	}
	dr := dump.Registry{}
	d.d.Registries[p.Ptr] = &dr // avoid infinite recursion

	dr.Top = r.top
	dr.Array = make([]dump.Value, len(r.array))
	for i, v := range r.array {
		dr.Array[i] = d.dumpLValue(v)
	}
	return
}

func (d dumper) dumpCallFrameStack(cfs *callFrameStack) (p dump.Ptr) {
	if cfs == nil {
		return
	}
	p = dump.Ptr{Type: dump.TCallFrameStack, Ptr: ptrToString(cfs)}
	_, ok := d.d.CallFrameStacks[p.Ptr]
	if ok {
		return
	}
	dcfs := dump.CallFrameStack{}
	d.d.CallFrameStacks[p.Ptr] = &dcfs // avoid infinite recursion

	dcfs.Sp = cfs.sp
	dcfs.Array = make([]dump.Ptr, len(cfs.array))
	for i, v := range cfs.array {
		dcfs.Array[i] = d.dumpCallFrame(&v)
	}
	return
}

func (d dumper) dumpUpvalue(uv *Upvalue) (p dump.Ptr) {
	if uv == nil {
		return
	}
	p = dump.Ptr{Type: dump.TUpvalue, Ptr: ptrToString(uv)}
	_, ok := d.d.Upvalues[p.Ptr]
	if ok {
		return
	}
	duv := dump.Upvalue{}
	d.d.Upvalues[p.Ptr] = &duv // avoid infinite recursion

	duv.Next = d.dumpUpvalue(uv.next)
	duv.Reg = d.dumpRegistry(uv.reg)
	duv.Index = uv.index
	duv.Value = d.dumpLValue(uv.value)
	duv.Closed = uv.closed
	return
}

func (d dumper) dumpFunction(f *LFunction) (p dump.Ptr) {
	if f == nil {
		return
	}
	p = dump.Ptr{Type: dump.TFunction, Ptr: ptrToString(f)}
	_, ok := d.d.Upvalues[p.Ptr]
	if ok {
		return
	}
	df := dump.Function{}
	d.d.Functions[p.Ptr] = &df // avoid infinite recursion

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

func (d dumper) dumpFunctionProto(fp *FunctionProto) (p dump.Ptr) {
	if fp == nil {
		return
	}
	p = dump.Ptr{Type: dump.TFunctionProto, Ptr: ptrToString(fp)}
	_, ok := d.d.FunctionProtos[p.Ptr]
	if ok {
		return
	}
	dfp := dump.FunctionProto{}
	d.d.FunctionProtos[p.Ptr] = &dfp // avoid infinite recursion

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

func (d dumper) dumpDbgLocalInfo(li *DbgLocalInfo) (p dump.Ptr) {
	if li == nil {
		return
	}
	p = dump.Ptr{Type: dump.TDbgLocalInfo, Ptr: ptrToString(li)}
	_, ok := d.d.DbgLocalInfos[p.Ptr]
	if ok {
		return
	}
	dli := dump.DbgLocalInfo{}
	d.d.DbgLocalInfos[p.Ptr] = &dli // avoid infinite recursion

	dli.EndPc = li.EndPc
	dli.Name = li.Name
	dli.StartPc = li.StartPc
	return
}

func (d dumper) dumpGFunction(gf LGFunction) (p dump.Ptr) {
	if gf == nil {
		return
	}
	p = dump.Ptr{Type: dump.TGFunction, Ptr: ptrToString(gf)}
	_, ok := d.d.GFunctions[p.Ptr]
	if ok {
		return
	}
	dgf := dump.GFunction{}
	d.d.GFunctions[p.Ptr] = &dgf // avoid infinite recursion

	f := runtime.FuncForPC(reflect.ValueOf(gf).Pointer())
	dgf.Name = f.Name()
	dgf.File, dgf.Line = f.FileLine(reflect.ValueOf(gf).Pointer())
	return
}

func (d dumper) dumpTable(t *LTable) (p dump.Ptr) {
	if t == nil {
		return
	}
	p = dump.Ptr{Type: dump.TTable, Ptr: ptrToString(t)}
	_, ok := d.d.Tables[p.Ptr]
	if ok {
		return
	}
	dt := dump.Table{}
	d.d.Tables[p.Ptr] = &dt // avoid infinite recursion

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

func ptrToString(ptr interface{}) string {
	v := reflect.ValueOf(ptr)
	if v.IsNil() {
		return "nil"
	}
	return fmt.Sprint(v.Type()) + "-" + fmt.Sprint(v.Pointer())
}

type dumpLoader struct {
	Data            dump.Data
	Loaded          map[string]struct{} // ids of objects loaded
	G               map[string]*Global  //for consistency
	States          map[string]*LState
	Tables          map[string]*LTable
	CallFrames      map[string]*callFrame
	CallFrameStacks map[string]*callFrameStack
	Registries      map[string]*registry
	Functions       map[string]*LFunction
	GFunctions      map[string]*LGFunction
	FunctionProtos  map[string]*FunctionProto
	DbgLocalInfos   map[string]*DbgLocalInfo
	Upvalues        map[string]*Upvalue
}

func (d dumpLoader) init() {
	d.G = make(map[string]*Global) //for consistency
	d.States = make(map[string]*LState)
	d.Tables = make(map[string]*LTable)
	d.CallFrames = make(map[string]*callFrame)
	d.CallFrameStacks = make(map[string]*callFrameStack)
	d.Registries = make(map[string]*registry)
	d.Functions = make(map[string]*LFunction)
	d.GFunctions = make(map[string]*LGFunction)
	d.FunctionProtos = make(map[string]*FunctionProto)
	d.DbgLocalInfos = make(map[string]*DbgLocalInfo)
	d.Upvalues = make(map[string]*Upvalue)
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

func (d dumpLoader) loadPtr(dump.Ptr) {

}

func (d dumpLoader) load() {

}

func LoadDump(d dump.Data) (*LState, error) {
	ld := dumpLoader{Data: d}
	ld.init()
	ld.load()
	return nil, nil
}

func (d dumpLoader) loadGlobal(ptr string) (*Global, error) {
	return nil, nil
}

func (d dumpLoader) loadCallFrame(ptr string) (*callFrame, error) {
	return nil, nil
}

func (d dumpLoader) loadRegistry(ptr string) (*registry, error) {
	return nil, nil
}

func (d dumpLoader) loadCallFrameStack(ptr string) (*callFrameStack, error) {
	return nil, nil
}

func (d dumpLoader) loadUpvalue(ptr string) (*Upvalue, error) {
	return nil, nil
}

func (d dumpLoader) loadFunction(ptr string) (*LFunction, error) {
	return nil, nil
}

func (d dumpLoader) loadFunctionProto(ptr string) (*FunctionProto, error) {
	return nil, nil
}

func (d dumpLoader) loadDbgLocalInfo(ptr string) (*DbgLocalInfo, error) {
	return nil, nil
}

func (d dumpLoader) loadGFunction(ptr string) (*LGFunction, error) {
	return nil, nil
}

func (d dumpLoader) loadTable(ptr string) (*LTable, error) {
	return nil, nil
}
