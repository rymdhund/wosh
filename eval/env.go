package eval

type Env struct {
	vars map[string]Object
}

func NewEnv() *Env {
	return &Env{map[string]Object{}}
}

func (env *Env) put(key string, obj Object) {
	env.vars[key] = obj
}

func (env *Env) get(key string) (Object, bool) {
	o, ok := env.vars[key]
	return o, ok
}
