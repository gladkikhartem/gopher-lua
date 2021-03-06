package lua

import (
	"fmt"
	"log"
	"reflect"
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
func (s *LState) Dump(du DumpUserData, df DumpGFunction) dump.Data {
	d := dumper{
		d: dump.Data{
			States:          make(map[dump.Ptr]*dump.State),
			Tables:          make(map[dump.Ptr]*dump.Table),
			CallFrames:      make(map[dump.Ptr]*dump.CallFrame),
			CallFrameStacks: make(map[dump.Ptr]*dump.CallFrameStack),
			Registries:      make(map[dump.Ptr]*dump.Registry),
			Functions:       make(map[dump.Ptr]*dump.Function),
			FunctionProtos:  make(map[dump.Ptr]*dump.FunctionProto),
			DbgLocalInfos:   make(map[dump.Ptr]*dump.DbgLocalInfo),
			Upvalues:        make(map[dump.Ptr]*dump.Upvalue),
			UserData:        make(map[dump.Ptr]*dump.UserData),
		},
		prefixCount:   make(map[string]int),
		ptrMap:        make(map[string]string),
		dumpData:      du,
		dumpGFunction: df,
		dumped:        make(map[dump.Ptr]bool),
	}
	d.dumpState(s, "dumpState", false)
	return d.d
}

type dumper struct {
	d             dump.Data
	dumpData      DumpUserData
	dumpGFunction DumpGFunction
	ptrMap        map[string]string
	prefixCount   map[string]int
	dumped        map[dump.Ptr]bool
}

func (d *dumper) dumpLValue(lv LValue, name string, initonly bool) dump.Value {
	if lv == nil {
		return dump.Value{Type: int(LTNil)}
	}
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
		return dump.Value{Type: int(LTFunction), Ptr: d.dumpFunction(v, name, initonly)}
	case *LState:
		return dump.Value{Type: int(LTThread), Ptr: d.dumpState(v, name, initonly)}
	case *LTable:
		return dump.Value{Type: int(LTTable), Ptr: d.dumpTable(v, name, initonly)}
	case *LUserData:
		return dump.Value{Type: int(LTUserData), Ptr: d.dumpUserData(v, name, initonly)}
	default:
		return dump.Value{Type: int(LTNil)}
	}
}

func (d *dumpLoader) loadLValue(vv dump.Value) (LValue, error) {
	switch LValueType(vv.Type) {
	case LTBool:
		v := LBool(vv.Bool)
		return v, nil
	case LTNumber:
		v := LNumber(vv.Number)
		return v, nil
	case LTString:
		v := LString(vv.String)
		return v, nil
	case LTNil:
		v := LNilType{}
		return &v, nil
	case LTFunction:
		return d.loadFunction(vv.Ptr)
	case LTThread:
		return d.loadState(vv.Ptr)
	case LTTable:
		return d.loadTable(vv.Ptr)
	case LTUserData:
		return d.loadUserData(vv.Ptr)
	default:
		return nil, fmt.Errorf("unsupported type: %#v", vv.Type)
	}
}

func (d *dumper) dumpState(s *LState, name string, initonly bool) (ptr dump.Ptr) {
	if s == nil {
		return
	}
	ptr = d.getPtr(s, name)
	_, ok := d.d.States[ptr]
	if ok && d.dumped[ptr] {
		return
	}
	ds := dump.State{}
	d.d.States[ptr] = &ds // avoid infinite recursion
	if initonly {
		return
	}
	d.dumped[ptr] = true
	d.dumpGlobal(s.G)
	ds.Parent = d.dumpState(s.Parent, ".parent", false)
	ds.Env = d.dumpTable(s.Env, ".env", false)
	ds.Options = dump.Options{
		CallStackSize:       s.Options.CallStackSize,
		RegistrySize:        s.Options.RegistrySize,
		SkipOpenLibs:        s.Options.SkipOpenLibs,
		IncludeGoStackTrace: s.Options.IncludeGoStackTrace,
	}
	ds.Stop = s.stop
	ds.UVCache = d.dumpUpvalue(s.uvcache, ".uvCache")
	ds.Reg = d.dumpRegistry(s.reg, ".reg")
	ds.Stack = d.dumpCallFrameStack(s.stack, ".stack")
	ds.CurrentFrame = d.dumpCallFrame(s.currentFrame, ".curFrame")
	ds.Wrapped = s.wrapped
	ds.HasErrorFunc = s.hasErrorFunc
	ds.Dead = s.Dead
	return
}

func (d *dumpLoader) loadState(ptr dump.Ptr) (*LState, error) {
	if ptr == "" {
		return nil, nil
	}
	s := d.States[ptr]
	ds := d.Data.States[ptr]
	id := fmt.Sprint("lstate-", ptr)
	if d.Loaded[id] {
		return s, nil
	}
	d.Loaded[id] = true
	var err error
	s.G, err = d.loadGlobal(ds.G)
	if err != nil {
		return nil, err
	}
	s.Parent, err = d.loadState(ds.Parent)
	if err != nil {
		return s, err
	}
	s.Env, err = d.loadTable(ds.Env)
	if err != nil {
		return nil, err
	}
	s.Options = Options{
		CallStackSize:       ds.Options.CallStackSize,
		RegistrySize:        ds.Options.RegistrySize,
		SkipOpenLibs:        ds.Options.SkipOpenLibs,
		IncludeGoStackTrace: ds.Options.IncludeGoStackTrace,
	}
	s.stop = ds.Stop
	s.reg, err = d.loadRegistry(ds.Reg)
	if err != nil {
		return nil, err
	}
	s.currentFrame, err = d.loadCallFrame(ds.CurrentFrame)
	if err != nil {
		return nil, err
	}
	s.stack, err = d.loadCallFrameStack(ds.Stack)
	if err != nil {
		return nil, err
	}
	changedRef, ok := d.cfParents[s.currentFrame] // check if s.currentFrame points to callFrame Stack
	if ok {
		s.currentFrame = changedRef
	} else if s.currentFrame != nil {
		changedRef, ok = d.cfParents[s.currentFrame.Parent] // check if s.currentFrame.Parent points to callFrameStack
		if ok {
			s.currentFrame.Parent = changedRef
		}
	}
	for _, v := range s.stack.array {
		changedRef, ok := d.cfParents[v.Parent] // check if stack.callFrame.Parent points to the same callFrameStack
		if ok {
			v.Parent = changedRef
		}
	}

	s.wrapped = ds.Wrapped
	s.uvcache, err = d.loadUpvalue(ds.UVCache)
	if err != nil {
		return nil, err
	}
	s.hasErrorFunc = ds.HasErrorFunc
	s.Dead = ds.Dead
	s.alloc = d.alloc
	if s.Options.IncludeGoStackTrace {
		s.Panic = panicWithTraceback
	} else {
		s.Panic = panicWithoutTraceback
	}
	s.mainLoop = mainLoop
	return s, nil
}

func (d *dumper) dumpGlobal(g *Global) {
	if d.d.G != nil {
		return
	}
	d.d.G = &dump.Global{}
	d.d.G.Global = d.dumpTable(g.Global, "global", false)
	d.d.G.MainThread = d.dumpState(g.MainThread, "mainThread", false)
	d.d.G.CurrentThread = d.dumpState(g.CurrentThread, "curThread", false)
	d.d.G.Registry = d.dumpTable(g.Registry, "reg", false)
	d.d.G.Gccount = g.gccount
	d.d.G.BuiltinMts = map[string]dump.Value{}
	for k, v := range g.builtinMts { // init pointers first to make them beautiful and consistent
		d.d.G.BuiltinMts[fmt.Sprint(k)] = d.dumpLValue(v, fmt.Sprint("builtin-", k), true)
	}
	for k, v := range g.builtinMts {
		d.d.G.BuiltinMts[fmt.Sprint(k)] = d.dumpLValue(v, fmt.Sprint("builtin-", k), false)
	}
	return
}

func (d *dumpLoader) loadGlobal(ptr dump.Ptr) (*Global, error) {
	if d.G.MainThread != nil {
		return d.G, nil
	}
	var err error
	d.G.MainThread, err = d.loadState(d.Data.G.MainThread)
	if err != nil {
		return nil, err
	}
	d.G.CurrentThread, err = d.loadState(d.Data.G.CurrentThread)
	if err != nil {
		return nil, err
	}
	d.G.Registry, err = d.loadTable(d.Data.G.Registry)
	if err != nil {
		return nil, err
	}
	d.G.Global, err = d.loadTable(d.Data.G.Global)
	if err != nil {
		return nil, err
	}
	d.G.gccount = d.Data.G.Gccount
	d.G.builtinMts = map[int]LValue{}
	for k, v := range d.Data.G.BuiltinMts {
		intk, _ := strconv.Atoi(k)
		d.G.builtinMts[intk], err = d.loadLValue(v)
		if err != nil {
			return nil, err
		}
	}
	return d.G, nil
}

func (d *dumper) dumpCallFrame(cf *callFrame, name string) (ptr dump.Ptr) {
	if cf == nil {
		return
	}
	ptr = d.getPtr(cf, name)
	_, ok := d.d.CallFrames[ptr]
	if ok {
		return
	}
	dcf := dump.CallFrame{}
	d.d.CallFrames[ptr] = &dcf // avoid infinite recursion

	dcf.Idx = cf.Idx
	dcf.Fn = d.dumpFunction(cf.Fn, "cf-func", false)
	dcf.Parent = d.dumpCallFrame(cf.Parent, "cf-parent")
	dcf.Pc = cf.Pc
	dcf.Base = cf.Base
	dcf.LocalBase = cf.LocalBase
	dcf.ReturnBase = cf.ReturnBase
	dcf.NArgs = cf.NArgs
	dcf.NRet = cf.NRet
	dcf.TailCall = cf.TailCall
	return
}

func (d *dumpLoader) loadCallFrame(ptr dump.Ptr) (*callFrame, error) {
	if ptr == "" {
		return nil, nil
	}
	cf := d.CallFrames[ptr]
	dcf := d.Data.CallFrames[ptr]
	id := fmt.Sprint("callframe-", ptr)
	if d.Loaded[id] {
		return cf, nil
	}
	d.Loaded[id] = true

	var err error
	cf.Idx = dcf.Idx
	cf.Fn, err = d.loadFunction(dcf.Fn)
	if err != nil {
		return nil, err
	}
	cf.Parent, err = d.loadCallFrame(dcf.Parent) // TODO: fix pointer to callFrames[] array ???
	if err != nil {
		return nil, err
	}
	cf.Pc = dcf.Pc
	cf.Base = dcf.Base
	cf.LocalBase = dcf.LocalBase
	cf.ReturnBase = dcf.ReturnBase
	cf.NArgs = dcf.NArgs
	cf.NRet = dcf.NRet
	cf.TailCall = dcf.TailCall

	return cf, nil
}

func (d *dumper) dumpRegistry(r *registry, name string) (ptr dump.Ptr) {
	if r == nil {
		return
	}
	ptr = d.getPtr(r, name)
	_, ok := d.d.Registries[ptr]
	if ok {
		return
	}
	dr := dump.Registry{}
	d.d.Registries[ptr] = &dr // avoid infinite recursion

	dr.Top = r.top
	dr.Array = make([]dump.Value, len(r.array))
	for i, v := range r.array { // init pointers first to make them beautiful and consistent
		dr.Array[i] = d.dumpLValue(v, fmt.Sprintf("%v.[%v]", name, i), true)
	}
	for i, v := range r.array {
		dr.Array[i] = d.dumpLValue(v, fmt.Sprintf("%v.[%v]", name, i), false)
	}
	// skip empty values to save dump space
	for i := len(dr.Array) - 1; i >= 0; i-- {
		if r.array[i] == nil {
			continue
		}
		dr.Array = dr.Array[:i+1]
		break
	}
	dr.Len = len(r.array)
	return
}

func (d *dumpLoader) loadRegistry(ptr dump.Ptr) (*registry, error) {
	if ptr == "" {
		return nil, nil
	}
	r := d.Registries[ptr]
	dr := d.Data.Registries[ptr]
	id := fmt.Sprint("registry-", ptr)
	if d.Loaded[id] {
		return r, nil
	}
	d.Loaded[id] = true

	var err error
	r.alloc = d.alloc
	r.top = dr.Top
	r.array = make([]LValue, dr.Len)
	for i, v := range dr.Array {
		r.array[i], err = d.loadLValue(v)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (d *dumper) dumpCallFrameStack(cfs *callFrameStack, name string) (ptr dump.Ptr) {
	if cfs == nil {
		return
	}
	ptr = d.getPtr(cfs, name)
	_, ok := d.d.CallFrameStacks[ptr]
	if ok {
		return
	}
	dcfs := dump.CallFrameStack{}
	d.d.CallFrameStacks[ptr] = &dcfs // avoid infinite recursion

	dcfs.Sp = cfs.sp
	dcfs.Array = make([]dump.Ptr, len(cfs.array))
	for i, v := range cfs.array {
		dcfs.Array[i] = d.dumpCallFrame(&v, fmt.Sprintf("%v.arr.[%v]", ptr, i))
	}
	// skip empty values to save dump space
	for i := len(dcfs.Array) - 1; i >= 0; i-- {
		if cfs.array[i].Fn == nil {
			continue
		}
		dcfs.Array = dcfs.Array[:i+1]
		break
	}
	dcfs.Len = len(cfs.array)
	return
}

func (d *dumpLoader) loadCallFrameStack(ptr dump.Ptr) (*callFrameStack, error) {
	if ptr == "" {
		return nil, nil
	}
	cfs := d.CallFrameStacks[ptr]
	dcfs := d.Data.CallFrameStacks[ptr]
	id := fmt.Sprint("callframestack-", ptr)
	if d.Loaded[id] {
		return cfs, nil
	}
	d.Loaded[id] = true

	cfs.sp = dcfs.Sp
	cfs.array = make([]callFrame, dcfs.Len)
	for i, v := range dcfs.Array {
		pv, err := d.loadCallFrame(v)
		if err != nil {
			return nil, err
		}
		cfs.array[i] = *pv
		d.cfParents[pv] = &cfs.array[i]
	}

	return cfs, nil
}

func (d *dumper) dumpUpvalue(uv *Upvalue, name string) (ptr dump.Ptr) {
	if uv == nil {
		return
	}
	ptr = d.getPtr(uv, name)
	_, ok := d.d.Upvalues[ptr]
	if ok {
		return
	}
	duv := dump.Upvalue{}
	d.d.Upvalues[ptr] = &duv // avoid infinite recursion

	duv.Next = d.dumpUpvalue(uv.next, string(ptr)+".next")
	duv.Reg = d.dumpRegistry(uv.reg, string(ptr)+".reg")
	duv.Index = uv.index
	duv.Value = d.dumpLValue(uv.value, string(ptr)+".value", false)
	duv.Closed = uv.closed
	return
}

func (d *dumpLoader) loadUpvalue(ptr dump.Ptr) (*Upvalue, error) {
	if ptr == "" {
		return nil, nil
	}
	uv := d.Upvalues[ptr]
	duv := d.Data.Upvalues[ptr]
	id := fmt.Sprint("upvalue-", ptr)
	if d.Loaded[id] {
		return uv, nil
	}
	d.Loaded[id] = true

	var err error
	uv.next, err = d.loadUpvalue(duv.Next)
	if err != nil {
		return nil, err
	}
	uv.reg, err = d.loadRegistry(duv.Reg)
	if err != nil {
		return nil, err
	}
	uv.index = duv.Index
	uv.value, err = d.loadLValue(duv.Value)
	if err != nil {
		return nil, err
	}
	uv.closed = duv.Closed
	return uv, nil
}

func (d *dumper) dumpFunction(f *LFunction, name string, initonly bool) (ptr dump.Ptr) {
	if f == nil {
		return
	}
	ptr = d.getPtr(f, name)
	_, ok := d.d.Upvalues[ptr]
	if ok && d.dumped[ptr] {
		return
	}
	df := dump.Function{}
	d.d.Functions[ptr] = &df // avoid infinite recursion
	if initonly {
		return
	}
	d.dumped[ptr] = true

	df.IsG = f.IsG
	df.Env = d.dumpTable(f.Env, string(ptr)+".env", false)
	if df.IsG {
		df.GFunction = d.dumpGFunction(f.GFunction)
	}
	df.Proto = d.dumpFunctionProto(f.Proto, string(ptr)+".proto")
	df.Upvalues = make([]dump.Ptr, len(f.Upvalues))
	for i, v := range f.Upvalues {
		df.Upvalues[i] = d.dumpUpvalue(v, fmt.Sprintf("%v.upv.[%v]", ptr, i))
	}
	return
}

func (d *dumpLoader) loadFunction(ptr dump.Ptr) (*LFunction, error) {
	if ptr == "" {
		return nil, nil
	}
	f := d.Functions[ptr]
	df := d.Data.Functions[ptr]
	id := fmt.Sprint("function-", ptr)
	if d.Loaded[id] {
		return f, nil
	}
	d.Loaded[id] = true
	var err error
	f.IsG = df.IsG
	if f.IsG {
		f.GFunction, err = d.parseFunction(df.GFunction)
		if err != nil {
			return nil, err
		}
	}
	f.Env, err = d.loadTable(df.Env)
	if err != nil {
		return nil, err
	}
	if df.Proto != "" {
		f.Proto, err = d.loadFunctionProto(df.Proto)
		if err != nil {
			return nil, err
		}
	}
	f.Upvalues = make([]*Upvalue, len(df.Upvalues))
	for i, v := range df.Upvalues {
		f.Upvalues[i], err = d.loadUpvalue(v)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

func (d *dumper) dumpFunctionProto(fp *FunctionProto, name string) (ptr dump.Ptr) {
	if fp == nil {
		return
	}
	ptr = d.getPtr(fp, name)
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
	for i, v := range fp.Constants { // init pointers first to make them beautiful and consistent
		dfp.Constants[i] = d.dumpLValue(v, fmt.Sprintf("%v.const.[%v]", ptr, i), true)
	}
	for i, v := range fp.Constants {
		dfp.Constants[i] = d.dumpLValue(v, fmt.Sprintf("%v.const.[%v]", ptr, i), false)
	}
	dfp.FunctionPrototypes = make([]dump.Ptr, len(fp.FunctionPrototypes))
	for i, v := range fp.FunctionPrototypes {
		dfp.FunctionPrototypes[i] = d.dumpFunctionProto(v, fmt.Sprintf("%v.protos.[%v]", ptr, i))
	}
	dfp.DbgCalls = make([]dump.DbgCall, len(fp.DbgCalls))
	for i, v := range fp.DbgCalls {
		dfp.DbgCalls[i] = dump.DbgCall{
			Pc:   v.Pc,
			Name: v.Name}
	}
	dfp.DbgLocals = make([]dump.Ptr, len(fp.DbgLocals))
	for i, v := range fp.DbgLocals {
		dfp.DbgLocals[i] = d.dumpDbgLocalInfo(v, fmt.Sprintf("%v.dbglocals.[%v]", ptr, i))
	}
	return
}

func (d *dumpLoader) loadFunctionProto(ptr dump.Ptr) (*FunctionProto, error) {
	if ptr == "" {
		return nil, nil
	}
	fp := d.FunctionProtos[ptr]
	dfp := d.Data.FunctionProtos[ptr]
	id := fmt.Sprint("functionproto-", ptr)
	if d.Loaded[id] {
		return fp, nil
	}
	d.Loaded[id] = true
	var err error
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
		fp.Constants[i], err = d.loadLValue(v)
		if err != nil {
			return nil, err
		}
	}
	fp.FunctionPrototypes = make([]*FunctionProto, len(dfp.FunctionPrototypes))
	for i, v := range dfp.FunctionPrototypes {
		fp.FunctionPrototypes[i], err = d.loadFunctionProto(v)
		if err != nil {
			return nil, err
		}
	}
	fp.DbgCalls = make([]DbgCall, len(dfp.DbgCalls))
	for i, v := range dfp.DbgCalls {
		fp.DbgCalls[i] = DbgCall{
			Pc:   v.Pc,
			Name: v.Name}
	}
	fp.DbgLocals = make([]*DbgLocalInfo, len(dfp.DbgLocals))
	for i, v := range dfp.DbgLocals {
		fp.DbgLocals[i], err = d.loadDbgLocalInfo(v)
		if err != nil {
			return nil, err
		}
	}

	return fp, nil
}

func (d *dumper) dumpDbgLocalInfo(li *DbgLocalInfo, name string) (ptr dump.Ptr) {
	if li == nil {
		return
	}
	ptr = d.getPtr(li, name)
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

func (d *dumpLoader) loadDbgLocalInfo(ptr dump.Ptr) (*DbgLocalInfo, error) {
	if ptr == "" {
		return nil, nil
	}
	li := d.DbgLocalInfos[ptr]
	dli := d.Data.DbgLocalInfos[ptr]
	id := fmt.Sprint("dbglocal-", ptr)
	if d.Loaded[id] {
		return li, nil
	}
	d.Loaded[id] = true

	li.EndPc = dli.EndPc
	li.Name = dli.Name
	li.StartPc = dli.StartPc

	return li, nil
}

func (d *dumper) dumpTable(t *LTable, name string, initonly bool) (ptr dump.Ptr) {
	if t == nil {
		return
	}
	ptr = d.getPtr(t, name)
	_, ok := d.d.Tables[ptr]
	if ok && d.dumped[ptr] {
		return
	}
	dt := dump.Table{}
	d.d.Tables[ptr] = &dt // avoid infinite recursion
	if initonly {
		return
	}
	d.dumped[ptr] = true
	dt.Metatable = d.dumpLValue(t.Metatable, string(ptr)+".meta", false)
	dt.Array = make([]dump.Value, len(t.array))
	for i, v := range t.array { // init pointers first to make them beautiful and consistent
		dt.Array[i] = d.dumpLValue(v, fmt.Sprintf("%v.[%v]", ptr, i), true)
	}
	for i, v := range t.array {
		dt.Array[i] = d.dumpLValue(v, fmt.Sprintf("%v.[%v]", ptr, i), false)
	}
	dt.Dict = []dump.VV{}
	for k, v := range t.dict {
		dt.Dict = append(dt.Dict, dump.VV{ // init pointers first to make them beautiful and consistent
			Key:   d.dumpLValue(k, fmt.Sprintf("%v.[%v].key", ptr, len(dt.Dict)), true),
			Value: d.dumpLValue(v, fmt.Sprintf("%v.[%v].value", ptr, len(dt.Dict)), true)})
	}
	for k, v := range t.dict {
		dt.Dict = append(dt.Dict, dump.VV{
			Key:   d.dumpLValue(k, fmt.Sprintf("%v.[%v].key", ptr, len(dt.Dict)), false),
			Value: d.dumpLValue(v, fmt.Sprintf("%v.[%v].value", ptr, len(dt.Dict)), false)})
	}
	dt.Strdict = map[string]dump.Value{}
	for k, v := range t.strdict { // init pointers first to make them beautiful and consistent
		dt.Strdict[k] = d.dumpLValue(v, fmt.Sprintf("%v.%v", ptr, k), true)
	}
	for k, v := range t.strdict {
		dt.Strdict[k] = d.dumpLValue(v, fmt.Sprintf("%v.%v", ptr, k), false)
	}
	return
}

func (d *dumpLoader) loadTable(ptr dump.Ptr) (*LTable, error) {
	if ptr == "" {
		return nil, nil
	}
	t, ok := d.Tables[ptr]
	if !ok {
		return nil, fmt.Errorf("table not found! %v %v %#v", ok, ptr, d.Tables)
	}
	dt := d.Data.Tables[ptr]
	id := fmt.Sprint("table-", ptr)
	if d.Loaded[id] {
		return t, nil
	}
	d.Loaded[id] = true
	var err error
	t.Metatable, err = d.loadLValue(dt.Metatable)
	if err != nil {
		return nil, err
	}
	t.array = make([]LValue, len(dt.Array))
	for i, v := range dt.Array {
		t.array[i], err = d.loadLValue(v)
		if err != nil {
			return nil, err
		}
	}
	t.k2i = map[LValue]int{}
	t.dict = map[LValue]LValue{}
	t.keys = []LValue{}
	for _, pair := range dt.Dict {
		k, err := d.loadLValue(pair.Key)
		if err != nil {
			return nil, err
		}
		v, err := d.loadLValue(pair.Value)
		if err != nil {
			return nil, err
		}
		t.dict[k] = v
		t.k2i[k] = len(t.keys)
		t.keys = append(t.keys, k)
	}
	t.strdict = map[string]LValue{}
	for k, v := range dt.Strdict {
		t.strdict[k], err = d.loadLValue(v)
		if err != nil {
			return nil, err
		}
		lkey := LString(k)
		t.k2i[lkey] = len(t.keys)
		t.keys = append(t.keys, lkey)
	}
	return t, nil
}

func (d *dumper) dumpUserData(t *LUserData, name string, initonly bool) (ptr dump.Ptr) {
	if t == nil {
		return
	}
	ptr = d.getPtr(t, name)
	_, ok := d.d.UserData[ptr]
	if ok && d.dumped[ptr] {
		return
	}
	var ud dump.UserData
	d.d.UserData[ptr] = &ud // avoid infinite recursion
	if initonly {
		return
	}
	d.dumped[ptr] = true
	ud = d.dumpData(t.Value)
	return
}

func (d *dumpLoader) loadUserData(ptr dump.Ptr) (*LUserData, error) {
	if ptr == "" {
		return nil, nil
	}
	t, ok := d.UserData[ptr]
	if !ok {
		log.Printf("userdata not found! %v %v %#v", ok, ptr, d.Tables)
	}
	dt := d.Data.UserData[ptr]
	id := fmt.Sprint("userdata-", ptr)
	if d.Loaded[id] {
		return t, nil
	}
	d.Loaded[id] = true
	var err error
	t, err = d.parseData(d.G.MainThread, *dt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (d dumper) getPtr(ptr interface{}, prefix string) dump.Ptr {
	v := reflect.ValueOf(ptr)
	if v.IsNil() {
		return "nil"
	}
	strPtr := fmt.Sprint(v.Type(), "-", v.Pointer())
	alias := d.ptrMap[strPtr]
	if alias != "" {
		return dump.Ptr(alias)
	}

	count := d.prefixCount[prefix]
	d.prefixCount[prefix]++
	alias = fmt.Sprintf("%v-%v", prefix, count)
	if count == 0 && prefix != "" {
		alias = fmt.Sprintf("%v", prefix)
	}
	d.ptrMap[strPtr] = alias
	return dump.Ptr(alias)
}

type dumpLoader struct {
	Errors          []error
	Data            dump.Data
	Loaded          map[string]bool // ids of objects loaded
	G               *Global         //for consistency
	States          map[dump.Ptr]*LState
	Tables          map[dump.Ptr]*LTable
	UserData        map[dump.Ptr]*LUserData
	CallFrames      map[dump.Ptr]*callFrame
	CallFrameStacks map[dump.Ptr]*callFrameStack
	Registries      map[dump.Ptr]*registry
	Functions       map[dump.Ptr]*LFunction
	FunctionProtos  map[dump.Ptr]*FunctionProto
	DbgLocalInfos   map[dump.Ptr]*DbgLocalInfo
	Upvalues        map[dump.Ptr]*Upvalue
	cfParents       map[*callFrame]*callFrame
	alloc           *allocator
	parseData       ParseUserData
	parseFunction   ParseGFunction
}

func (d *dumpLoader) init() {
	d.alloc = newAllocator(32)
	d.Loaded = make(map[string]bool)
	d.States = make(map[dump.Ptr]*LState)
	d.Tables = make(map[dump.Ptr]*LTable)
	d.CallFrames = make(map[dump.Ptr]*callFrame)
	d.CallFrameStacks = make(map[dump.Ptr]*callFrameStack)
	d.Registries = make(map[dump.Ptr]*registry)
	d.Functions = make(map[dump.Ptr]*LFunction)
	d.FunctionProtos = make(map[dump.Ptr]*FunctionProto)
	d.DbgLocalInfos = make(map[dump.Ptr]*DbgLocalInfo)
	d.Upvalues = make(map[dump.Ptr]*Upvalue)
	d.cfParents = make(map[*callFrame]*callFrame)
	d.UserData = make(map[dump.Ptr]*LUserData)
	d.G = &Global{}
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
	for k := range d.Data.UserData {
		d.UserData[k] = &LUserData{}
	}
}

type ParseUserData func(*LState, dump.UserData) (*LUserData, error)
type DumpUserData func(interface{}) dump.UserData

type ParseGFunction func(dump.Ptr) (LGFunction, error)
type DumpGFunction func(interface{}) dump.Ptr

func LoadDump(d dump.Data, pd ParseUserData, pf ParseGFunction) (*LState, error) {
	ld := dumpLoader{Data: d, parseData: pd, parseFunction: pf}
	ld.init()
	return ld.loadState(ld.Data.G.CurrentThread)
}
