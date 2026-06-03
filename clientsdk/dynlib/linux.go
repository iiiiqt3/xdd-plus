//go:build linux

package dynlib

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <stdint.h>

typedef uintptr_t (*func0)();
typedef uintptr_t (*func1)(uintptr_t);
typedef uintptr_t (*func2)(uintptr_t, uintptr_t);
typedef uintptr_t (*func3)(uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*func4)(uintptr_t, uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*func5)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t);
typedef uintptr_t (*func6)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t);

uintptr_t callFunc0(void* f) {
    return ((func0)f)();
}

uintptr_t callFunc1(void* f, uintptr_t a1) {
    return ((func1)f)(a1);
}

uintptr_t callFunc2(void* f, uintptr_t a1, uintptr_t a2) {
    return ((func2)f)(a1, a2);
}

uintptr_t callFunc3(void* f, uintptr_t a1, uintptr_t a2, uintptr_t a3) {
    return ((func3)f)(a1, a2, a3);
}

uintptr_t callFunc4(void* f, uintptr_t a1, uintptr_t a2, uintptr_t a3, uintptr_t a4) {
    return ((func4)f)(a1, a2, a3, a4);
}

uintptr_t callFunc5(void* f, uintptr_t a1, uintptr_t a2, uintptr_t a3, uintptr_t a4, uintptr_t a5) {
    return ((func5)f)(a1, a2, a3, a4, a5);
}

uintptr_t callFunc6(void* f, uintptr_t a1, uintptr_t a2, uintptr_t a3, uintptr_t a4, uintptr_t a5, uintptr_t a6) {
    return ((func6)f)(a1, a2, a3, a4, a5, a6);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

type linuxLibrary struct {
	handle unsafe.Pointer
}

func init() {
	newLibrary = NewLinuxLibrary
}

func NewLinuxLibrary(path string) (DynamicLibrary, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.dlopen(cPath, C.RTLD_LAZY)
	if handle == nil {
		return nil, errors.New(C.GoString(C.dlerror()))
	}
	return &linuxLibrary{handle: handle}, nil
}

func (l *linuxLibrary) Close() error {
	if C.dlclose(l.handle) != 0 {
		return errors.New(C.GoString(C.dlerror()))
	}
	return nil
}

func (l *linuxLibrary) Call(funcName string, args ...interface{}) (uintptr, uintptr, error) {
	cFuncName := C.CString(funcName)
	defer C.free(unsafe.Pointer(cFuncName))

	proc := C.dlsym(l.handle, cFuncName)
	if proc == nil {
		return 0, 0, errors.New(C.GoString(C.dlerror()))
	}

	// 将参数转换为 C 类型
	cArgs := make([]C.uintptr_t, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case int:
			cArgs[i] = C.uintptr_t(v)
		case int32:
			cArgs[i] = C.uintptr_t(v)
		case uint32:
			cArgs[i] = C.uintptr_t(v)
		case int64:
			cArgs[i] = C.uintptr_t(v)
		case uint64:
			cArgs[i] = C.uintptr_t(v)
		case *uint8:
			cArgs[i] = C.uintptr_t(uintptr(unsafe.Pointer(v)))
		case *int32:
			cArgs[i] = C.uintptr_t(uintptr(unsafe.Pointer(v)))
		case *uint32:
			cArgs[i] = C.uintptr_t(uintptr(unsafe.Pointer(v)))
		case *int64:
			cArgs[i] = C.uintptr_t(uintptr(unsafe.Pointer(v)))
		case *uint64:
			cArgs[i] = C.uintptr_t(uintptr(unsafe.Pointer(v)))
		case byte: // char
			cArgs[i] = C.uintptr_t(v)
		case []uint8:
			cArgs[i] = C.uintptr_t(uintptr(unsafe.Pointer(&v[0])))
		case string:
			cStr := C.CString(v)
			defer C.free(unsafe.Pointer(cStr))
			cArgs[i] = C.uintptr_t(uintptr(unsafe.Pointer(cStr)))
		default:
			return 0, 0, fmt.Errorf("unsupported argument type: %T", arg)
		}
	}

	// 调用函数
	var ret1, ret2 uintptr
	switch len(cArgs) {
	case 0:
		ret := C.callFunc0(proc)
		ret1 = uintptr(ret)
	case 1:
		ret := C.callFunc1(proc, cArgs[0])
		ret1 = uintptr(ret)
	case 2:
		ret := C.callFunc2(proc, cArgs[0], cArgs[1])
		ret1 = uintptr(ret)
	case 3:
		ret := C.callFunc3(proc, cArgs[0], cArgs[1], cArgs[2])
		ret1 = uintptr(ret)
	case 4:
		ret := C.callFunc4(proc, cArgs[0], cArgs[1], cArgs[2], cArgs[3])
		ret1 = uintptr(ret)
	case 5:
		ret := C.callFunc5(proc, cArgs[0], cArgs[1], cArgs[2], cArgs[3], cArgs[4])
		ret1 = uintptr(ret)
	case 6:
		ret := C.callFunc6(proc, cArgs[0], cArgs[1], cArgs[2], cArgs[3], cArgs[4], cArgs[5])
		ret1 = uintptr(ret)
	default:
		return 0, 0, fmt.Errorf("too many arguments: %d", len(cArgs))
	}

	return ret1, ret2, nil
}
