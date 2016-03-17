package lunar

import (
	"io"
)

func WriteBuiltins(w io.Writer) (n int, err error) {
	return w.Write([]byte(`
local builtins = _G.lunar_go_builtins or {}
_G.lunar_go_builtins = builtins

function builtins.append(dst, ...)
	for i=1, select('#', ...) do
		local val = select(i, ...)
		table.insert(dst, val)
	end
	return dst
end

function builtins.delete(map, key)
	map[key] = nil
end

function builtins.mapLength(m)
	local l = 0
	for _ in pairs(m) do
		l = l + 1
	end
	return l
end

function builtins.makeSlice(f, n)
	local s = {}
	if n == nil then
		n = 0
	end
	for i = 1, n do
		table.insert(s, f())
	end
	return s
end

local inits = {}
function builtins.add_init(f)
	table.insert(inits, f)
end

function builtins.run_inits()
	for _, f in ipairs(inits) do
		f()
	end
end

local closureCache = setmetatable({}, {__mode="k"}) -- weak keys
function builtins.create_closure(obj, funcName)
	-- See if we have a closure cache for this object already
	local objClosures = closureCache[obj]
	if objClosures == nil then
		-- No cache for this object; create one
		objClosures = {}
		closureCache[obj] = objClosures
	end

	-- See if we have a closure created for this obj+funcName already
	local f = objClosures[funcName]
	if f ~= nil then
		return f
	end

	-- No closure created; create a new one
	f = function(...)
		return obj[funcName](obj, ...)
	end

	-- Store the new closure in the cache
	objClosures[funcName] = f
	return f
end
`))
}
