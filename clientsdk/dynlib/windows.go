//go:build windows

package dynlib

import (
	"fmt"
	"syscall"
	"unsafe"
)

type windowsLibrary struct {
	baseLibrary
}

func init() {
	newLibrary = NewWindowsLibrary
}

func NewWindowsLibrary(path string) (DynamicLibrary, error) {
	handle, err := syscall.LoadLibrary(path)
	if err != nil {
		return nil, err
	}
	return &windowsLibrary{baseLibrary{handle: uintptr(handle)}}, nil
}

func (l *windowsLibrary) Close() error {
	return syscall.FreeLibrary(syscall.Handle(l.handle))
}

func (l *windowsLibrary) Call(funcName string, args ...interface{}) (uintptr, uintptr, error) {
	proc, err := syscall.GetProcAddress(syscall.Handle(l.handle), funcName)
	if err != nil {
		return 0, 0, err
	}
	// 将参数转换为 uintptr
	uintptrArgs := make([]uintptr, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case int:
			uintptrArgs[i] = uintptr(v)
		case int32:
			uintptrArgs[i] = uintptr(v)
		case uint32:
			uintptrArgs[i] = uintptr(v)
		case int64:
			uintptrArgs[i] = uintptr(v)
		case uint64:
			uintptrArgs[i] = uintptr(v)
		case *uint8:
			uintptrArgs[i] = uintptr(unsafe.Pointer(v))
		case *int32:
			uintptrArgs[i] = uintptr(unsafe.Pointer(v))
		case *uint32:
			uintptrArgs[i] = uintptr(unsafe.Pointer(v))
		case *int64:
			uintptrArgs[i] = uintptr(unsafe.Pointer(v))
		case *uint64:
			uintptrArgs[i] = uintptr(unsafe.Pointer(v))
		case byte: // char
			uintptrArgs[i] = uintptr(v)
		case []uint8:
			uintptrArgs[i] = uintptr(unsafe.Pointer(&v[0]))
		case string:
			strPtr, err := syscall.BytePtrFromString(v)
			if err != nil {
				return 0, 0, err
			}
			uintptrArgs[i] = uintptr(unsafe.Pointer(strPtr))
		default:
			return 0, 0, fmt.Errorf("unsupported argument type: %T", arg)
		}
	}

	// 调用函数
	ret, _, errno := syscall.SyscallN(proc, uintptrArgs...)
	return ret, uintptr(errno), err
}
