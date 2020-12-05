package obj

type Class struct {
	Name    string
	Methods FunctionList
}

type FunctionList map[string]*FunctionObject

var UnitClass = Class{"Unit", FunctionList{}}
var BoolClass = Class{"Bool", FunctionList{}}
var IntClass = Class{"Int", FunctionList{}}
var StringClass = Class{"Str", FunctionList{}}
var ListClass = Class{"List", FunctionList{}}
var FunctionClass = Class{"Function", FunctionList{}}
var ExceptionClass = Class{"Exception", FunctionList{}}
