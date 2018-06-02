package dump

type Type int

const (
	TState Type = iota
	TTable
	TGlobal
	TRegistry
	TCallFrame
	TCallFrameStack
	TUpvalue
	TFunction
	TGFunction
	TFunctionProto
	TDbgLocalInfo
)

type Ptr struct {
	Type Type
	Ptr  string
}

type Value struct {
	Type   int    // LValueType
	Ptr    string // for state,table,and etc...
	String string
	Bool   bool
	Number float64
}

type CallFrame struct {
	Idx        int
	Fn         Ptr //*LFunction
	Parent     Ptr //*callFrame
	Pc         int
	Base       int
	LocalBase  int
	ReturnBase int
	NArgs      int
	NRet       int
	TailCall   int
}

type CallFrameStack struct {
	Array []Ptr
	Sp    int
}
type Registry struct {
	Array []Value
	Top   int
	//alloc *allocator
}

type Upvalue struct {
	Next   Ptr //*Upvalue
	Reg    Ptr //*registry
	Index  int
	Value  Value
	Closed bool
}

type Global struct {
	MainThread    Ptr
	CurrentThread Ptr
	Registry      Ptr
	Global        Ptr

	BuiltinMts map[string]Value
	//tempFiles  []*os.File  // TODO: save files???
	Gccount int32
}

type Options struct {
	CallStackSize       int
	RegistrySize        int
	SkipOpenLibs        bool
	IncludeGoStackTrace bool
}

type DbgCall struct {
	Name string
	Pc   int
}

type State struct {
	G      Ptr //*Global
	Parent Ptr //*LState
	Env    Ptr //*LTable
	//Panic   //func(*LState)
	Dead    bool
	Options Options

	Stop         int32
	Reg          Ptr
	Stack        Ptr //*callFrameStack
	CurrentFrame Ptr //*callFrame
	Wrapped      bool
	UVCache      Ptr //*Upvalue
	HasErrorFunc bool
	//MainLoop     //func(*LState, *callFrame)
	//Alloc        //*allocator
	//Ctx          context.Context
}

type Data struct {
	G               map[string]*Global //for consistency
	States          map[string]*State
	Tables          map[string]*Table
	CallFrames      map[string]*CallFrame
	CallFrameStacks map[string]*CallFrameStack
	Registries      map[string]*Registry
	Functions       map[string]*Function
	GFunctions      map[string]*GFunction
	FunctionProtos  map[string]*FunctionProto
	DbgLocalInfos   map[string]*DbgLocalInfo
	Upvalues        map[string]*Upvalue
}

type DbgLocalInfo struct {
	Name    string
	StartPc int
	EndPc   int
}

type FunctionProto struct {
	SourceName         string
	LineDefined        int
	LastLineDefined    int
	NumUpvalues        uint8
	NumParameters      uint8
	IsVarArg           uint8
	NumUsedRegisters   uint8
	Code               []uint32
	Constants          []Value //[]LValue
	FunctionPrototypes []Ptr   //[]*FunctionProto

	DbgSourcePositions []int
	DbgLocals          []Ptr // TODO: []*DbgLocalInfo
	DbgCalls           []DbgCall
	DbgUpvalues        []string

	StringConstants []string
}
type Function struct {
	IsG       bool
	Env       Ptr   //*LTable
	Proto     Ptr   //*FunctionProto
	GFunction Ptr   //LGFunction
	Upvalues  []Ptr //[]*Upvalue
}

type GFunction struct {
	Name string
	File string
	Line int
}

type VV struct {
	Key   Value
	Value Value
}
type VI struct {
	Key   Value
	Value int
}

type Table struct {
	Metatable Value
	Array     []Value
	Dict      []VV
	Strdict   map[string]Value
	Keys      []Value
	K2i       []VI
}
