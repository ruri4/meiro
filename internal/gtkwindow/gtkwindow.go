//go:build linux && (amd64 || arm64)

// Package gtkwindow reports the shape the desktop gives a window, so that a
// page can end where the decorations the desktop draws around it do.
package gtkwindow

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Radius reports the corner radius the theme gives a window's decoration, in
// DIPs, or 0 when it rounds none or the theme cannot be read. It is read
// once, the first time a page asks: a theme switched while Meiro runs keeps
// the radius it started with.
func Radius() float32 { return onceRadius() }

var onceRadius = sync.OnceValue(readRadius)

// gvalue is a GValue: its type, then its value, which takes two words.
type gvalue struct {
	typ  uintptr
	data [2]uintptr
}

var (
	gdkScreenGetDefault            func() unsafe.Pointer
	gtkStyleContextNew             func() unsafe.Pointer
	gtkStyleContextSetPath         func(context, path unsafe.Pointer)
	gtkStyleContextSetScreen       func(context, screen unsafe.Pointer)
	gtkStyleContextSetState        func(context unsafe.Pointer, state uint32)
	gtkStyleContextGetProperty     func(context unsafe.Pointer, property *byte, state uint32, value unsafe.Pointer)
	gtkWidgetPathNew               func() unsafe.Pointer
	gtkWidgetPathAppendType        func(path unsafe.Pointer, typ uintptr) int32
	gtkWidgetPathIterAddClass      func(path unsafe.Pointer, pos int32, class *byte)
	gtkWidgetPathIterSetObjectName func(path unsafe.Pointer, pos int32, name *byte)
	gtkWidgetPathIterSetState      func(path unsafe.Pointer, pos int32, state uint32)
	gtkWidgetPathFree              func(path unsafe.Pointer)
	gObjectUnref                   func(object unsafe.Pointer)
	gValueUnset                    func(value unsafe.Pointer)
	gValueGetInt                   func(value unsafe.Pointer) int32
)

func readRadius() float32 {
	gtk, err := purego.Dlopen("libgtk-3.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0
	}
	gdk, err := purego.Dlopen("libgdk-3.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0
	}
	gobject, err := purego.Dlopen("libgobject-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0
	}
	binds := []struct {
		lib  uintptr
		name string
		fn   any
	}{
		{gdk, "gdk_screen_get_default", &gdkScreenGetDefault},
		{gtk, "gtk_style_context_new", &gtkStyleContextNew},
		{gtk, "gtk_style_context_set_path", &gtkStyleContextSetPath},
		{gtk, "gtk_style_context_set_screen", &gtkStyleContextSetScreen},
		{gtk, "gtk_style_context_set_state", &gtkStyleContextSetState},
		{gtk, "gtk_style_context_get_property", &gtkStyleContextGetProperty},
		{gtk, "gtk_widget_path_new", &gtkWidgetPathNew},
		{gtk, "gtk_widget_path_append_type", &gtkWidgetPathAppendType},
		{gtk, "gtk_widget_path_iter_add_class", &gtkWidgetPathIterAddClass},
		{gtk, "gtk_widget_path_iter_set_object_name", &gtkWidgetPathIterSetObjectName},
		{gtk, "gtk_widget_path_iter_set_state", &gtkWidgetPathIterSetState},
		{gtk, "gtk_widget_path_free", &gtkWidgetPathFree},
		{gobject, "g_object_unref", &gObjectUnref},
		{gobject, "g_value_unset", &gValueUnset},
		{gobject, "g_value_get_int", &gValueGetInt},
	}
	for _, bind := range binds {
		symbol, err := purego.Dlsym(bind.lib, bind.name)
		if err != nil {
			return 0
		}
		purego.RegisterFunc(bind.fn, symbol)
	}
	// A display is needed for the theme to resolve at all.
	screen := gdkScreenGetDefault()
	if screen == nil {
		return 0
	}
	// The decoration node the theme rounds: the window's own path, with the
	// classes a client-side decorated window carries, then a node of its own.
	// The node takes no classes: a themed selector for the bare node stops
	// matching once it has any.
	path := gtkWidgetPathNew()
	if path == nil {
		return 0
	}
	defer gtkWidgetPathFree(path)
	gtkWidgetPathAppendType(path, 0)
	gtkWidgetPathIterAddClass(path, -1, cstr("background"))
	gtkWidgetPathIterAddClass(path, -1, cstr("csd"))
	gtkWidgetPathIterSetObjectName(path, gtkWidgetPathAppendType(path, 0), cstr("decoration"))
	gtkWidgetPathIterSetState(path, -1, 0)

	context := gtkStyleContextNew()
	if context == nil {
		return 0
	}
	defer gObjectUnref(context)
	gtkStyleContextSetScreen(context, screen)
	gtkStyleContextSetPath(context, path)
	gtkStyleContextSetState(context, 0)

	// GTK initializes the value to the property's own type, and warns if it
	// arrives initialized already, so it is handed over zeroed.
	var value gvalue
	defer gValueUnset(unsafe.Pointer(&value))
	gtkStyleContextGetProperty(context, cstr("border-radius"), 0, unsafe.Pointer(&value))
	radius := gValueGetInt(unsafe.Pointer(&value))
	if radius <= 0 {
		return 0
	}
	return float32(radius)
}

// cstr returns s as a C string. The caller passes it straight to GTK, which
// copies what it keeps.
func cstr(s string) *byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return &b[0]
}
