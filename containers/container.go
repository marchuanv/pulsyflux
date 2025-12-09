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
	typeValue    reflect.Value // always a pointer
	dependencies *sliceext.List[*depEdge]
	once         *sync.Once
}

type depEdge struct {
	dep         *metadata
	field       reflect.Value
	setter      reflect.Value
	setterValue reflect.Value
	useField    bool
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

// find setter "Set<DepName>" on inst, prepare reflect.Value for injection
func findSetterFor(depName string, inst any, depValue any) (reflect.Value, reflect.Value, error) {
	methodName := "Set" + pascalCase(depName)
	mv := reflect.ValueOf(inst).MethodByName(methodName)
	if !mv.IsValid() {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf(
			"DI Error: method %s not found on %T", methodName, inst,
		)
	}

	mt := mv.Type()
	if mt.NumIn() != 1 {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf(
			"DI Error: method %s on %T must accept exactly 1 argument", methodName, inst,
		)
	}

	paramType := mt.In(0)
	val := reflect.ValueOf(depValue)
	valType := val.Type()

	if valType.AssignableTo(paramType) {
		return mv, val, nil
	} else if valType.ConvertibleTo(paramType) {
		return mv, val.Convert(paramType), nil
	} else {
		return reflect.Value{}, reflect.Value{}, fmt.Errorf(
			"DI Error: method %s expects %v but dependency has %v",
			methodName, paramType, valType,
		)
	}
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
		metaValue = reflect.New(reflect.TypeOf(*new(T))) // always pointer
	} else {
		v := reflect.ValueOf(value)
		if v.Kind() != reflect.Ptr {
			panic(fmt.Sprintf("Type %s must be a pointer", typeId))
		}
		metaValue = v
	}

	meta := &metadata{
		Id:           string(typeId),
		name:         metaValue.Elem().Type().Name(),
		typeValue:    metaValue,
		dependencies: sliceext.NewList[*depEdge](),
		once:         &sync.Once{},
	}

	types.Add(string(typeId), meta)
}

// Register a dependency field or setter
func addArgType[T comparable, ArgT comparable](
	typeId contracts.TypeId[T],
	argTypeId contracts.TypeId[ArgT],
	argName string,
	argValue *ArgT,
) {
	// Ensure dependency type exists
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

	// Find field in parent struct
	elem := meta.typeValue.Elem()
	var field reflect.Value
	var structField reflect.StructField
	found := false
	for i := 0; i < elem.NumField(); i++ {
		f := elem.Field(i)
		sf := elem.Type().Field(i)
		if sf.Name == argName {
			field = f
			structField = sf
			found = true
			break
		}
	}
	if !found {
		panic(fmt.Sprintf("Field %s not found on %s", argName, elem.Type()))
	}

	// Enforce pointer fields for all injected dependencies (except int/string)
	if structField.Type.Kind() != reflect.Ptr &&
		structField.Type.Kind() != reflect.Int &&
		structField.Type.Kind() != reflect.String {
		panic(fmt.Sprintf("Field %s on type %s must be a pointer", argName, meta.Id))
	}

	useField := field.CanSet()
	var setter reflect.Value
	var setterVal reflect.Value

	if !useField {
		parentInst := meta.typeValue.Interface()
		childInst := depMeta.typeValue.Interface()
		s, val, err := findSetterFor(argName, parentInst, childInst)
		if err != nil {
			panic(err)
		}
		setter = s
		setterVal = val
	}

	edge := &depEdge{
		dep:         depMeta,
		field:       field,
		setter:      setter,
		setterValue: setterVal,
		useField:    useField,
	}

	meta.dependencies.Add(edge)
}

// Recursively inject dependencies
func resolveDependencies(meta *metadata, visited map[*metadata]bool, stack map[*metadata]bool) {
	if visited[meta] {
		return
	}
	if stack[meta] {
		panic(fmt.Sprintf("Cyclic dependency detected at type: %s", meta.Id))
	}
	stack[meta] = true

	for _, edge := range meta.dependencies.All() {
		dep := edge.dep
		resolveDependencies(dep, visited, stack)

		if edge.useField {
			edge.field.Set(dep.typeValue)
		} else {
			edge.setter.Call([]reflect.Value{edge.setterValue})
		}
	}

	stack[meta] = false
	visited[meta] = true
}

// Get returns the singleton instance
func Get[T interface{}, T2 any](typeId contracts.TypeId[T2]) T {
	typesMu.RLock()
	meta := types.Get(string(typeId))
	typesMu.RUnlock()

	if meta == nil {
		panic(fmt.Sprintf("Type %s is not registered", typeId))
	}

	meta.once.Do(func() {
		resolveDependencies(meta, make(map[*metadata]bool), make(map[*metadata]bool))
	})

	val := meta.typeValue.Interface()
	tInterface, ok := val.(T)
	if !ok {
		panic(fmt.Sprintf("Type %s does not implement requested interface", typeId))
	}

	return tInterface
}
