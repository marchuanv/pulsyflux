package containers

import (
	"fmt"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"reflect"
	"strings"
	"sync"
)

type metadata struct {
	name         string
	Id           string
	typeValue    reflect.Value // now always a pointer
	field        reflect.Value
	dependencies *sliceext.List[*metadata]
	once         *sync.Once
	useField     bool
	setter       reflect.Value
}

var (
	types   = sliceext.NewDictionary[string, *metadata]()
	typesMu sync.RWMutex
)

// convert "wheel04" -> "Wheel04"
func pascalCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// find setter "Set" + PascalCase(argName) on inst, validate it accepts the dependency's type
func findSetterFor(depName string, inst any, depValue any) (reflect.Value, error) {
	methodName := "Set" + pascalCase(depName)
	mv := reflect.ValueOf(inst).MethodByName(methodName)
	if !mv.IsValid() {
		return reflect.Value{}, fmt.Errorf("DI Error: method %s not found on %T", methodName, inst)
	}
	mt := mv.Type()
	if mt.NumIn() != 1 {
		return reflect.Value{}, fmt.Errorf("DI Error: method %s on %T must accept exactly 1 argument", methodName, inst)
	}

	paramType := mt.In(0)
	valType := reflect.TypeOf(depValue)

	// Check assignability: a value of valType must be assignable to paramType
	if !valType.AssignableTo(paramType) {
		// also allow convertible case when the value is assignable via interface (rare edge)
		if !(valType.ConvertibleTo(paramType)) {
			return reflect.Value{}, fmt.Errorf("DI Error: method %s expects %v but dependency has %v", methodName, paramType, valType)
		}
	}

	return mv, nil
}

// Register a type
func addType[T comparable](typeId contracts.TypeId[T], value *T) {
	typesMu.Lock()
	defer typesMu.Unlock()

	if types.Has(string(typeId)) {
		panic(fmt.Sprintf(
			"Type %s (%T) is already registered. Overwriting is not allowed",
			string(typeId),
			value,
		))
	}

	var metaValue reflect.Value

	if value == nil {
		// create pointer to zero value
		metaValue = reflect.New(reflect.TypeOf(*new(T))) // *T
	} else {
		v := reflect.ValueOf(value)
		if v.Kind() == reflect.Ptr {
			metaValue = v
		} else {
			panic(fmt.Sprintf("Type %s must be a pointer", typeId))
		}
	}

	meta := &metadata{
		Id:           string(typeId),
		name:         metaValue.Elem().Type().Name(),
		typeValue:    metaValue, // pointer stored
		dependencies: sliceext.NewList[*metadata](),
		once:         &sync.Once{},
	}

	types.Add(string(typeId), meta)
}

// Register a field dependency
func addArgType[T comparable, ArgT comparable](
	typeId contracts.TypeId[T],
	argTypeId contracts.TypeId[ArgT],
	argName string,
	argValue *ArgT,
) {
	// Ensure arg type exists
	typesMu.RLock()
	exists := types.Has(string(argTypeId))
	typesMu.RUnlock()

	if !exists {
		addType(argTypeId, argValue)
	}

	typesMu.RLock()
	meta := types.Get(string(typeId))
	depMeta := types.Get(string(argTypeId))
	typesMu.RUnlock()

	if meta == nil || depMeta == nil {
		panic("Type or dependency not registered")
	}

	// Find the field in the parent struct
	var field reflect.Value
	found := false
	for i := 0; i < meta.typeValue.Elem().NumField(); i++ {
		f := meta.typeValue.Elem().Field(i)
		if meta.typeValue.Elem().Type().Field(i).Name == argName {
			field = f
			found = true
			break
		}
	}
	if !found {
		panic(fmt.Sprintf("Field %s not found on %s", argName, meta.typeValue.Elem().Type()))
	}

	// Ensure field is a pointer for injection
	if field.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("Field %s on type %s must be a pointer", argName, meta.Id))
	}

	depMeta.useField = field.CanSet()
	if !depMeta.useField {
		parentInst := meta.typeValue.Interface()   // concrete pointer (e.g., *car)
		childInst := depMeta.typeValue.Interface() // concrete pointer (e.g., *wheel)
		s, err := findSetterFor(argName, parentInst, childInst)
		if err != nil {
			panic(err)
		}
		depMeta.setter = s
	}
	depMeta.name = argName
	depMeta.field = field
	meta.dependencies.Add(depMeta)

}

// Recursively inject dependencies with cycle detection
func resolveDependencies(meta *metadata, visited map[*metadata]bool, stack map[*metadata]bool) {
	if visited[meta] {
		return
	}
	if stack[meta] {
		panic(fmt.Sprintf("Cyclic dependency detected at type: %s", meta.Id))
	}
	stack[meta] = true

	for _, dep := range meta.dependencies.All() {
		// Resolve dependencies of the child first
		resolveDependencies(dep, visited, stack)

		if !dep.useField {
			dep.setter.Call([]reflect.Value{dep.typeValue})
		} else {
			// inject pointer
			dep.field.Set(dep.typeValue)
		}

	}

	stack[meta] = false
	visited[meta] = true
}

func Get[T interface{}, T2 any](typeId contracts.TypeId[T2]) T {
	typesMu.RLock()
	meta := types.Get(string(typeId))
	typesMu.RUnlock()

	if meta == nil {
		panic(fmt.Sprintf("Type %s is not registered", typeId))
	}

	// Ensure dependencies are resolved
	meta.once.Do(func() {
		resolveDependencies(meta, make(map[*metadata]bool), make(map[*metadata]bool))
	})

	// Return as interface
	val := meta.typeValue.Interface() // concrete pointer

	tInterface, ok := val.(T)
	if !ok {
		panic(fmt.Sprintf("Type %s does not implement requested interface", typeId))
	}

	return tInterface
}
