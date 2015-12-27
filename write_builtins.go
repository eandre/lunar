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

function builtins.mapLength(m)
	local l = 0
	for _ in pairs(m) do
		l = l + 1
	end
	return l
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
`))
}