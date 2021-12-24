An experimental, just-for-fun programming language for playing around with syntax choices and language constructions like algebraic effects.

Features:
- Simple syntax
- Bytecode compiler inspired by python
- Algebraic effects

See `examples/` for example syntax. 

### Algebraic effects 

Algebraic effects are like generalized exceptions that you can resume. They look like this:

```
fn foo() {
  do eff(10)
  1
}

fn bar() {
  a = 0
  try {
    foo() + a
  } handle {
    eff(x) @ k -> {
      a = x
      resume k
    }
  }
}

bar() # will return 11
```
