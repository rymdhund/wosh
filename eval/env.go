package eval

// This will probably be an interface
type Object struct {
	Type  string
	Value int
}

func (o Object) add(o2 Object) Object {
	if o.Type != "int" {
		panic("trying to add non-integer")
	}
	if o2.Type != "int" {
		panic("trying to add non-integer")
	}
	return Object{"int", o.Value + o2.Value}
}

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
