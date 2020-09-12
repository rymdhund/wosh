package eval

import "fmt"

type Env struct {
	vars           map[string]Object
	outputCaptures []string
}

func NewEnv() *Env {
	return &Env{
		map[string]Object{},
		[]string{},
	}
}

func (env *Env) put(key string, obj Object) {
	env.vars[key] = obj
}

func (env *Env) get(key string) (Object, bool) {
	o, ok := env.vars[key]
	return o, ok
}

func (env *Env) SetCaptureOutput() {
	env.outputCaptures = append(env.outputCaptures, "")
}

func (env *Env) PopCaptureOutput() Object {
	out := env.outputCaptures[len(env.outputCaptures)-1]
	env.outputCaptures = env.outputCaptures[:len(env.outputCaptures)-1]
	return StrVal(out)
}

func (env *Env) OutPutStr(s string) {
	if len(env.outputCaptures) > 0 {
		env.outputCaptures[len(env.outputCaptures)-1] += s
	} else {
		fmt.Print(s)
	}
}

func (env *Env) OutAdd(o Object) {
	s := o.GetString()
	env.OutPutStr(s)
}
