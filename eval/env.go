package eval

import (
	"fmt"
	"os"

	"github.com/rymdhund/wosh/obj"
)

type Env struct {
	outer       *Env
	vars        map[string]obj.Object
	outCaptures []string
	errCaptures []string
}

func NewEnv() *Env {
	return &Env{
		nil,
		map[string]obj.Object{},
		[]string{},
		[]string{},
	}
}

func NewInnerEnv(env *Env) *Env {
	return &Env{
		env,
		map[string]obj.Object{},
		[]string{},
		[]string{},
	}
}

func (env *Env) put(key string, o obj.Object) {
	env.vars[key] = o
}

func (env *Env) get(key string) (obj.Object, bool) {
	o, ok := env.vars[key]
	if !ok && env.outer != nil {
		return env.outer.get(key)
	}
	return o, ok
}

func (env *Env) SetCaptureOutput() {
	env.outCaptures = append(env.outCaptures, "")
}

func (env *Env) PopCaptureOutput() obj.Object {
	out := env.outCaptures[len(env.outCaptures)-1]
	env.outCaptures = env.outCaptures[:len(env.outCaptures)-1]
	return obj.StrVal(out)
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

func (env *Env) PopCaptureErr() obj.Object {
	out := env.errCaptures[len(env.errCaptures)-1]
	env.errCaptures = env.errCaptures[:len(env.errCaptures)-1]
	return obj.StrVal(out)
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
