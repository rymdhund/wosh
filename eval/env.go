package eval

import (
	"fmt"
	"os"
)

type Env struct {
	outer       *Env
	vars        map[string]Object
	outCaptures []string
	errCaptures []string
}

func NewEnv() *Env {
	return &Env{
		nil,
		map[string]Object{},
		[]string{},
		[]string{},
	}
}

func NewInnerEnv(env *Env) *Env {
	return &Env{
		env,
		map[string]Object{},
		[]string{},
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
	env.outCaptures = append(env.outCaptures, "")
}

func (env *Env) PopCaptureOutput() Object {
	out := env.outCaptures[len(env.outCaptures)-1]
	env.outCaptures = env.outCaptures[:len(env.outCaptures)-1]
	return StrVal(out)
}

func (env *Env) OutPutStr(s string) {
	if len(env.outCaptures) > 0 {
		env.outCaptures[len(env.outCaptures)-1] += s
	} else if env.outer != nil {
		env.outer.OutPutStr(s)
	} else {
		fmt.Print(s)
	}
}

func (env *Env) SetCaptureErr() {
	env.errCaptures = append(env.errCaptures, "")
}

func (env *Env) PopCaptureErr() Object {
	out := env.errCaptures[len(env.errCaptures)-1]
	env.errCaptures = env.errCaptures[:len(env.errCaptures)-1]
	return StrVal(out)
}

func (env *Env) ErrPutStr(s string) {
	if len(env.errCaptures) > 0 {
		env.errCaptures[len(env.errCaptures)-1] += s
	} else if env.outer != nil {
		env.outer.ErrPutStr(s)
	} else {
		fmt.Fprint(os.Stderr, s)
	}
}
