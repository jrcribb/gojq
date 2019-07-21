package gojq

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

const (
	argcount0 = 1 << iota
	argcount1
	argcount2
	argcount3
)

type function struct {
	argcount int
	callback func(interface{}, []interface{}) interface{}
}

func (fn function) accept(argcount int) bool {
	switch argcount {
	case 0:
		return fn.argcount&argcount0 > 0
	case 1:
		return fn.argcount&argcount1 > 0
	case 2:
		return fn.argcount&argcount2 > 0
	case 3:
		return fn.argcount&argcount3 > 0
	default:
		return false
	}
}

var internalFuncs map[string]function

func init() {
	internalFuncs = map[string]function{
		"empty":          argFunc0(nil),
		"path":           argFunc1(nil),
		"length":         argFunc0(funcLength),
		"utf8bytelength": argFunc0(funcUtf8ByteLength),
		"keys":           argFunc0(funcKeys),
		"has":            argFunc1(funcHas),
		"tonumber":       argFunc0(funcToNumber),
		"tostring":       argFunc0(funcToString),
		"type":           argFunc0(funcType),
		"explode":        argFunc0(funcExplode),
		"implode":        argFunc0(funcImplode),
		"tojson":         argFunc0(funcToJSON),
		"fromjson":       argFunc0(funcFromJSON),
		"_index":         argFunc2(funcIndex),
		"_slice":         argFunc3(funcSlice),
		"_break":         argFunc0(funcBreak),
		"_plus":          argFunc0(funcOpPlus),
		"_negate":        argFunc0(funcOpNegate),
		"_add":           argFunc2(funcOpAdd),
		"_subtract":      argFunc2(funcOpSub),
		"_multiply":      argFunc2(funcOpMul),
		"_divide":        argFunc2(funcOpDiv),
		"_modulo":        argFunc2(funcOpMod),
		"_alternative":   argFunc2(funcOpAlt),
		"_equal":         argFunc2(funcOpEq),
		"_notequal":      argFunc2(funcOpNe),
		"_greater":       argFunc2(funcOpGt),
		"_less":          argFunc2(funcOpLt),
		"_greatereq":     argFunc2(funcOpGe),
		"_lesseq":        argFunc2(funcOpLe),
		"sin":            mathFunc("sin", math.Sin),
		"cos":            mathFunc("cos", math.Cos),
		"tan":            mathFunc("tan", math.Tan),
		"asin":           mathFunc("asin", math.Asin),
		"acos":           mathFunc("acos", math.Acos),
		"atan":           mathFunc("atan", math.Atan),
		"sinh":           mathFunc("sinh", math.Sinh),
		"cosh":           mathFunc("cosh", math.Cosh),
		"tanh":           mathFunc("tanh", math.Tanh),
		"asinh":          mathFunc("asinh", math.Asinh),
		"acosh":          mathFunc("acosh", math.Acosh),
		"atanh":          mathFunc("atanh", math.Atanh),
		"floor":          mathFunc("floor", math.Floor),
		"round":          mathFunc("round", math.Round),
		"rint":           mathFunc("rint", math.Round),
		"ceil":           mathFunc("ceil", math.Ceil),
		"trunc":          mathFunc("trunc", math.Trunc),
		"fabs":           mathFunc("fabs", math.Abs),
		"sqrt":           mathFunc("sqrt", math.Sqrt),
		"cbrt":           mathFunc("cbrt", math.Cbrt),
		"exp":            mathFunc("exp", math.Exp),
		"exp10":          mathFunc("exp10", func(v float64) float64 { return math.Pow(10, v) }),
		"exp2":           mathFunc("exp2", math.Exp2),
		"expm1":          mathFunc("expm1", math.Expm1),
		"frexp":          argFunc0(funcFrexp),
		"modf":           argFunc0(funcModf),
		"log":            mathFunc("log", math.Log),
		"log10":          mathFunc("log10", math.Log10),
		"log1p":          mathFunc("log1p", math.Log1p),
		"log2":           mathFunc("log2", math.Log2),
		"logb":           mathFunc("logb", math.Logb),
		"gamma":          mathFunc("gamma", math.Gamma),
		"tgamma":         mathFunc("tgamma", math.Gamma),
		"lgamma":         mathFunc("lgamma", func(v float64) float64 { v, _ = math.Lgamma(v); return v }),
		"erf":            mathFunc("erf", math.Erf),
		"erfc":           mathFunc("erfc", math.Erfc),
		"j0":             mathFunc("j0", math.J0),
		"j1":             mathFunc("j1", math.J1),
		"y0":             mathFunc("y0", math.Y0),
		"y1":             mathFunc("y1", math.Y1),
		"atan2":          mathFunc2("atan2", math.Atan2),
		"copysign":       mathFunc2("copysign", math.Copysign),
		"drem": mathFunc2("drem", func(l, r float64) float64 {
			x := math.Remainder(l, r)
			if x == 0.0 {
				return math.Copysign(x, l)
			}
			return x
		}),
		"fdim":        mathFunc2("fdim", math.Dim),
		"fmax":        mathFunc2("fmax", math.Max),
		"fmin":        mathFunc2("fmin", math.Min),
		"fmod":        mathFunc2("fmod", math.Mod),
		"hypot":       mathFunc2("hypot", math.Hypot),
		"jn":          mathFunc2("jn", func(l, r float64) float64 { return math.Jn(int(l), r) }),
		"ldexp":       mathFunc2("ldexp", func(l, r float64) float64 { return math.Ldexp(l, int(r)) }),
		"nextafter":   mathFunc2("nextafter", math.Nextafter),
		"nexttoward":  mathFunc2("nexttoward", math.Nextafter),
		"remainder":   mathFunc2("remainder", math.Remainder),
		"scalb":       mathFunc2("scalb", func(l, r float64) float64 { return l * math.Pow(2, r) }),
		"scalbln":     mathFunc2("scalbln", func(l, r float64) float64 { return l * math.Pow(2, r) }),
		"yn":          mathFunc2("yn", func(l, r float64) float64 { return math.Yn(int(l), r) }),
		"pow":         mathFunc2("pow", math.Pow),
		"fma":         mathFunc3("fma", func(x, y, z float64) float64 { return x*y + z }),
		"setpath":     argFunc2(funcSetpath),
		"delpaths":    argFunc1(funcDelpaths),
		"getpath":     argFunc1(funcGetpath),
		"error":       function{argcount0 | argcount1, funcError},
		"builtins":    argFunc0(funcBuiltins),
		"env":         argFunc0(funcEnv),
		"_type_error": argFunc1(internalfuncTypeError),
	}
}

func argFunc0(fn func(interface{}) interface{}) function {
	return function{argcount0, func(v interface{}, _ []interface{}) interface{} {
		return fn(v)
	},
	}
}

func argFunc1(fn func(interface{}, interface{}) interface{}) function {
	return function{argcount1, func(v interface{}, args []interface{}) interface{} {
		return fn(v, args[0])
	},
	}
}

func argFunc2(fn func(interface{}, interface{}, interface{}) interface{}) function {
	return function{argcount2, func(v interface{}, args []interface{}) interface{} {
		return fn(v, args[0], args[1])
	},
	}
}

func argFunc3(fn func(interface{}, interface{}, interface{}, interface{}) interface{}) function {
	return function{argcount3, func(v interface{}, args []interface{}) interface{} {
		return fn(v, args[0], args[1], args[2])
	},
	}
}

func mathFunc(name string, f func(x float64) float64) function {
	return argFunc0(func(v interface{}) interface{} {
		x, err := toFloat64(name, v)
		if err != nil {
			return err
		}
		return f(x)
	})
}

func mathFunc2(name string, g func(x, y float64) float64) function {
	return argFunc2(func(_, x, y interface{}) interface{} {
		l, err := toFloat64(name, x)
		if err != nil {
			return err
		}
		r, err := toFloat64(name, y)
		if err != nil {
			return err
		}
		return g(l, r)
	})
}

func mathFunc3(name string, g func(x, y, z float64) float64) function {
	return argFunc3(func(_, a, b, c interface{}) interface{} {
		x, err := toFloat64(name, a)
		if err != nil {
			return err
		}
		y, err := toFloat64(name, b)
		if err != nil {
			return err
		}
		z, err := toFloat64(name, c)
		if err != nil {
			return err
		}
		return g(x, y, z)
	})
}

func toFloat64(name string, v interface{}) (float64, error) {
	switch v := v.(type) {
	case int:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, &funcTypeError{name, v}
	}
}

func funcLength(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return len(v)
	case map[string]interface{}:
		return len(v)
	case string:
		return len([]rune(v))
	case int:
		if v >= 0 {
			return v
		}
		return -v
	case float64:
		return math.Abs(v)
	case nil:
		return 0
	default:
		return &funcTypeError{"length", v}
	}
}

func funcUtf8ByteLength(v interface{}) interface{} {
	switch v := v.(type) {
	case string:
		return len([]byte(v))
	default:
		return &funcTypeError{"utf8bytelength", v}
	}
}

func funcKeys(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		w := make([]interface{}, len(v))
		for i := range v {
			w[i] = i
		}
		return w
	case map[string]interface{}:
		w := make([]string, len(v))
		var i int
		for k := range v {
			w[i] = k
			i++
		}
		sort.Strings(w)
		u := make([]interface{}, len(v))
		for i, x := range w {
			u[i] = x
		}
		return u
	default:
		return &funcTypeError{"keys", v}
	}
}

func funcHas(v, x interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		switch x := x.(type) {
		case int:
			return 0 <= x && x < len(v)
		case float64:
			return 0 <= int(x) && int(x) < len(v)
		default:
			return &hasKeyTypeError{v, x}
		}
	case map[string]interface{}:
		switch x := x.(type) {
		case string:
			_, ok := v[x]
			return ok
		default:
			return &hasKeyTypeError{v, x}
		}
	default:
		return &hasKeyTypeError{v, x}
	}
}

func funcToNumber(v interface{}) interface{} {
	switch v := v.(type) {
	case int, uint, float64:
		return v
	case string:
		var x float64
		if err := json.Unmarshal([]byte(v), &x); err != nil {
			return fmt.Errorf("%s: %q", err, v)
		}
		return x
	default:
		return &funcTypeError{"tonumber", v}
	}
}

func funcToString(v interface{}) interface{} {
	if s, ok := v.(string); ok {
		return s
	}
	return funcToJSON(v)
}

func funcType(v interface{}) interface{} {
	return typeof(v)
}

func funcExplode(v interface{}) interface{} {
	switch v := v.(type) {
	case string:
		return explode(v)
	default:
		return &funcTypeError{"explode", v}
	}
}

func explode(s string) []interface{} {
	rs := []int32(s)
	xs := make([]interface{}, len(rs))
	for i, r := range rs {
		xs[i] = int(r)
	}
	return xs
}

func funcImplode(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return implode(v)
	default:
		return &funcTypeError{"implode", v}
	}
}

func implode(v []interface{}) interface{} {
	var rs []rune
	for _, r := range v {
		switch r := r.(type) {
		case int:
			rs = append(rs, rune(r))
		case float64:
			rs = append(rs, rune(r))
		default:
			return &funcTypeError{"implode", v}
		}
	}
	return string(rs)
}

func funcToJSON(v interface{}) interface{} {
	xs, err := json.Marshal(v)
	if err != nil {
		xs, err = json.Marshal(normalizeValues(v))
		if err != nil {
			return err
		}
	}
	return string(xs)
}

func funcFromJSON(v interface{}) interface{} {
	switch v := v.(type) {
	case string:
		var w interface{}
		err := json.Unmarshal([]byte(v), &w)
		if err != nil {
			return err
		}
		return w
	default:
		return &funcTypeError{"fromjson", v}
	}
}

func funcIndex(_, v, x interface{}) interface{} {
	switch x := x.(type) {
	case string:
		switch v := v.(type) {
		case nil:
			return nil
		case map[string]interface{}:
			return v[x]
		default:
			return &expectedObjectError{v}
		}
	case int, float64:
		idx, _ := toInt(x)
		switch v := v.(type) {
		case nil:
			return nil
		case []interface{}:
			return funcIndexSlice(nil, nil, &idx, v)
		case string:
			switch v := funcIndexSlice(nil, nil, &idx, explode(v)).(type) {
			case []interface{}:
				return implode(v)
			case int:
				return implode([]interface{}{v})
			case nil:
				return ""
			default:
				panic(v)
			}
		default:
			return &expectedArrayError{v}
		}
	case []interface{}:
		switch v := v.(type) {
		case nil:
			return nil
		case []interface{}:
			var xs []interface{}
			if len(x) == 0 {
				return xs
			}
			for i := 0; i < len(v) && i < len(v)-len(x)+1; i++ {
				var neq bool
				for j, y := range x {
					if neq = compare(v[i+j], y) != 0; neq {
						break
					}
				}
				if !neq {
					xs = append(xs, i)
				}
			}
			return xs
		default:
			return &expectedArrayError{v}
		}
	default:
		return &objectKeyNotStringError{x}
	}
}

func funcSlice(_, v, end, start interface{}) (r interface{}) {
	if w, ok := v.(string); ok {
		v = explode(w)
		defer func() {
			switch s := r.(type) {
			case []interface{}:
				r = implode(s)
			case int:
				r = implode([]interface{}{s})
			case nil:
				r = ""
			default:
				panic(r)
			}
		}()
	}
	switch v := v.(type) {
	case nil:
		return nil
	case []interface{}:
		if start != nil {
			if start, ok := toInt(start); ok {
				if end != nil {
					if end, ok := toInt(end); ok {
						return funcIndexSlice(&start, &end, nil, v)
					}
					return &arrayIndexNotNumberError{end}
				}
				return funcIndexSlice(&start, nil, nil, v)
			}
			return &arrayIndexNotNumberError{start}
		}
		if end != nil {
			if end, ok := toInt(end); ok {
				return funcIndexSlice(nil, &end, nil, v)
			}
			return &arrayIndexNotNumberError{end}
		}
		return v
	default:
		return &expectedArrayError{v}
	}
}

func funcIndexSlice(start, end, index *int, a []interface{}) interface{} {
	l := len(a)
	toIndex := func(i int) int {
		switch {
		case i < -l:
			return -2
		case i < 0:
			return l + i
		case i < l:
			return i
		default:
			return -1
		}
	}
	if index != nil {
		i := toIndex(*index)
		if i < 0 {
			return nil
		}
		return a[i]
	}
	if end != nil {
		i := toIndex(*end)
		if i == -1 {
			i = len(a)
		} else if i == -2 {
			i = 0
		}
		a = a[:i]
	}
	if start != nil {
		i := toIndex(*start)
		if i == -1 || len(a) < i {
			i = len(a)
		} else if i == -2 {
			i = 0
		}
		a = a[i:]
	}
	return a
}

func funcBreak(x interface{}) interface{} {
	return &breakError{x.(string)}
}

func funcFrexp(v interface{}) interface{} {
	x, err := toFloat64("frexp", v)
	if err != nil {
		return err
	}
	f, e := math.Frexp(x)
	return []interface{}{f, e}
}

func funcModf(v interface{}) interface{} {
	x, err := toFloat64("modf", v)
	if err != nil {
		return err
	}
	i, f := math.Modf(x)
	return []interface{}{f, i}
}

func funcSetpath(v, p, w interface{}) interface{} {
	return updatePaths("setpath", clone(v), p, func(interface{}) interface{} {
		return w
	})
}

func funcDelpaths(v, p interface{}) interface{} {
	paths, ok := p.([]interface{})
	if !ok {
		return &funcTypeError{"delpaths", p}
	}
	for _, path := range paths {
		v = updatePaths("delpaths", clone(v), path, func(interface{}) interface{} {
			return struct{}{}
		})
		if _, ok := v.(error); ok {
			return v
		}
	}
	return deleteEmpty(v)
}

func updatePaths(name string, v, p interface{}, f func(interface{}) interface{}) interface{} {
	keys, ok := p.([]interface{})
	if !ok {
		return &funcTypeError{name, p}
	}
	if len(keys) == 0 {
		return f(v)
	}
	u := v
	g := func(w interface{}) interface{} { v = w; return w }
loop:
	for i, x := range keys {
		switch x := x.(type) {
		case string:
			if u == nil {
				if name == "delpaths" {
					break loop
				}
				u = g(make(map[string]interface{}))
			}
			switch uu := u.(type) {
			case map[string]interface{}:
				if _, ok := uu[x]; !ok && name == "delpaths" {
					break loop
				}
				if i < len(keys)-1 {
					u = uu[x]
					g = func(w interface{}) interface{} { uu[x] = w; return w }
				} else {
					uu[x] = f(uu[x])
				}
			default:
				return &expectedObjectError{u}
			}
		case int, float64:
			if u == nil {
				u = g([]interface{}{})
			}
			y, _ := toInt(x)
			switch uu := u.(type) {
			case []interface{}:
				l := len(uu)
				if y >= len(uu) && name == "setpath" {
					l = y + 1
				} else if y < -len(uu) {
					if name == "delpaths" {
						break loop
					}
					return &funcTypeError{name, y}
				} else if y < 0 {
					y = len(uu) + y
				}
				ys := make([]interface{}, l)
				copy(ys, uu)
				uu = ys
				g(uu)
				if y >= len(uu) {
					break loop
				}
				if i < len(keys)-1 {
					u = uu[y]
					g = func(w interface{}) interface{} { uu[y] = w; return w }
				} else {
					uu[y] = f(uu[y])
				}
			default:
				return &expectedArrayError{u}
			}
		default:
			switch u.(type) {
			case []interface{}:
				return &arrayIndexNotNumberError{x}
			default:
				return &objectKeyNotStringError{x}
			}
		}
	}
	return v
}

func funcGetpath(v, p interface{}) interface{} {
	keys, ok := p.([]interface{})
	if !ok {
		return &funcTypeError{"getpath", p}
	}
	u := v
	for _, x := range keys {
		switch v.(type) {
		case map[string]interface{}:
		case []interface{}:
		case nil:
		default:
			return &getpathError{u, p}
		}
		v = funcIndex(nil, v, x)
		if _, ok := v.(error); ok {
			return &getpathError{u, p}
		}
	}
	return v
}

func funcError(v interface{}, args []interface{}) interface{} {
	if len(args) == 0 {
		switch v := v.(type) {
		case string:
			return errors.New(v)
		default:
			return &funcTypeError{"error", v}
		}
	} else if len(args) == 1 {
		switch v := args[0].(type) {
		case string:
			return errors.New(v)
		default:
			return &funcTypeError{"error", v}
		}
	} else {
		return nil
	}
}

func funcBuiltins(interface{}) interface{} {
	var xs []string
	for name, fn := range internalFuncs {
		if name[0] != '_' {
			for i, cnt := 0, fn.argcount; cnt > 0; i, cnt = i+1, cnt>>1 {
				if cnt&1 > 0 {
					xs = append(xs, name+"/"+fmt.Sprint(i))
				}
			}
		}
	}
	for _, q := range builtinFuncs {
		for _, fd := range q.FuncDefs {
			if fd.Name[0] != '_' {
				xs = append(xs, fd.Name+"/"+fmt.Sprint(len(fd.Args)))
			}
		}
	}
	sort.Strings(xs)
	ys := make([]interface{}, len(xs))
	for i, x := range xs {
		ys[i] = x
	}
	return ys
}

func funcEnv(interface{}) interface{} {
	env := make(map[string]interface{})
	for _, kv := range os.Environ() {
		xs := strings.SplitN(kv, "=", 2)
		env[xs[0]] = xs[1]
	}
	return env
}

func internalfuncTypeError(v, x interface{}) interface{} {
	return &funcTypeError{x.(string), v}
}

func toInt(x interface{}) (int, bool) {
	switch x := x.(type) {
	case int:
		return x, true
	case float64:
		return int(x), true
	default:
		return 0, false
	}
}
