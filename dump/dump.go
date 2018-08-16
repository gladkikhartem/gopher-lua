package dump

import "encoding/json"

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

type Ptr string

type Value struct {
	Type   int     `json:",omitempty"` // LValueType
	Ptr    Ptr     `json:",omitempty"` // for state,table,and etc...
	String string  `json:",omitempty"`
	Bool   bool    `json:",omitempty"`
	Number float64 `json:",omitempty"`
}

type CallFrame struct {
	Idx        int `json:",omitempty"`
	Fn         Ptr `json:",omitempty"` //*LFunction
	Parent     Ptr `json:",omitempty"` //*callFrame
	Pc         int `json:",omitempty"`
	Base       int `json:",omitempty"`
	LocalBase  int `json:",omitempty"`
	ReturnBase int `json:",omitempty"`
	NArgs      int `json:",omitempty"`
	NRet       int `json:",omitempty"`
	TailCall   int `json:",omitempty"`
}

type CallFrameStack struct {
	Array []Ptr `json:",omitempty"`
	Len   int   `json:",omitempty"`
	Sp    int   `json:",omitempty"`
}
type Registry struct {
	Array []Value `json:",omitempty"`
	Len   int     `json:",omitempty"`
	Top   int     `json:",omitempty"`
	//alloc *allocator
}

type Upvalue struct {
	Next   Ptr   `json:",omitempty"` //*Upvalue
	Reg    Ptr   `json:",omitempty"` //*registry
	Index  int   `json:",omitempty"`
	Value  Value `json:",omitempty"`
	Closed bool  `json:",omitempty"`
}

type Global struct {
	MainThread    Ptr `json:",omitempty"`
	CurrentThread Ptr `json:",omitempty"`
	Registry      Ptr `json:",omitempty"`
	Global        Ptr `json:",omitempty"`

	BuiltinMts map[string]Value `json:",omitempty"`
	//tempFiles  []*os.File  // TODO: save files???
	Gccount int32 `json:",omitempty"`
}

type Options struct {
	CallStackSize       int  `json:",omitempty"`
	RegistrySize        int  `json:",omitempty"`
	SkipOpenLibs        bool `json:",omitempty"`
	IncludeGoStackTrace bool `json:",omitempty"`
}

type DbgCall struct {
	Name string `json:",omitempty"`
	Pc   int    `json:",omitempty"`
}

type State struct {
	G      Ptr `json:",omitempty"` //*Global
	Parent Ptr `json:",omitempty"` //*LState
	Env    Ptr `json:",omitempty"` //*LTable
	//Panic   //func(*LState)
	Dead    bool    `json:",omitempty"`
	Options Options `json:",omitempty"`

	Stop         int32 `json:",omitempty"`
	Reg          Ptr   `json:",omitempty"`
	Stack        Ptr   `json:",omitempty"` //*callFrameStack
	CurrentFrame Ptr   `json:",omitempty"` //*callFrame
	Wrapped      bool  `json:",omitempty"`
	UVCache      Ptr   `json:",omitempty"` //*Upvalue
	HasErrorFunc bool  `json:",omitempty"`
	//MainLoop     //func(*LState, *callFrame)
	//Alloc        //*allocator
	//Ctx          context.Context
}

type UserData struct {
	Type string
	Data interface{}
}

type Data struct {
	G               *Global                 `json:",omitempty"` //for consistency
	States          map[Ptr]*State          `json:",omitempty"`
	Tables          map[Ptr]*Table          `json:",omitempty"`
	UserData        map[Ptr]*UserData       `json:",omitempty"`
	CallFrames      map[Ptr]*CallFrame      `json:",omitempty"`
	CallFrameStacks map[Ptr]*CallFrameStack `json:",omitempty"`
	Registries      map[Ptr]*Registry       `json:",omitempty"`
	Functions       map[Ptr]*Function       `json:",omitempty"`
	FunctionProtos  map[Ptr]*FunctionProto  `json:",omitempty"`
	DbgLocalInfos   map[Ptr]*DbgLocalInfo   `json:",omitempty"`
	Upvalues        map[Ptr]*Upvalue        `json:",omitempty"`
}

type DbgLocalInfo struct {
	Name    string `json:",omitempty"`
	StartPc int    `json:",omitempty"`
	EndPc   int    `json:",omitempty"`
}

type FunctionProto struct {
	SourceName         string   `json:",omitempty"`
	LineDefined        int      `json:",omitempty"`
	LastLineDefined    int      `json:",omitempty"`
	NumUpvalues        uint8    `json:",omitempty"`
	NumParameters      uint8    `json:",omitempty"`
	IsVarArg           uint8    `json:",omitempty"`
	NumUsedRegisters   uint8    `json:",omitempty"`
	Code               []uint32 `json:",omitempty"`
	Constants          []Value  `json:",omitempty"` //[]LValue
	FunctionPrototypes []Ptr    `json:",omitempty"` //[]*FunctionProto

	DbgSourcePositions []int     `json:",omitempty"`
	DbgLocals          []Ptr     `json:",omitempty"` // TODO: []*DbgLocalInfo
	DbgCalls           []DbgCall `json:",omitempty"`
	DbgUpvalues        []string  `json:",omitempty"`

	StringConstants []string `json:",omitempty"`
}
type Function struct {
	IsG       bool  `json:",omitempty"`
	Env       Ptr   `json:",omitempty"` //*LTable
	Proto     Ptr   `json:",omitempty"` //*FunctionProto
	GFunction Ptr   `json:",omitempty"` //LGFunction
	Upvalues  []Ptr `json:",omitempty"` //[]*Upvalue
}

type GFunction struct {
	Name  string          `json:",omitempty"`
	Bound json.RawMessage `json:",omitempty"`
}

type VV struct {
	Key   Value `json:",omitempty"`
	Value Value `json:",omitempty"`
}
type VI struct {
	Key   Value `json:",omitempty"`
	Value int   `json:",omitempty"`
}

type Table struct {
	Metatable Value            `json:",omitempty"`
	Array     []Value          `json:",omitempty"`
	Dict      []VV             `json:",omitempty"`
	Strdict   map[string]Value `json:",omitempty"`
	Keys      []Value          `json:",omitempty"`
	K2i       []VI             `json:",omitempty"`
	dumped    bool
}
