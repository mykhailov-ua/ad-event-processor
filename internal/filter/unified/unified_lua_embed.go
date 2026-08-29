package unified

import (
	_ "embed"
)

//go:embed unified-filter.lua
var unifiedFilterLua string

var unifiedFilterLuaAny any

func init() {
	unifiedFilterLuaAny = unifiedFilterLua
	budgetFastLuaAny = budgetFastLua
}

func unifiedFilterLuaForScript() string {
	return unifiedFilterLua
}
