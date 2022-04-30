local _M = {}
local base = os.getenv("ZMX_LUA")
package.path = package.path .. ";" .. base .. "/?.lua"
local F = require "actions.vendor/F"

--- 
-- params module_name string
-- params name string
-- ret string 
local function gen_zsh_export_function(module_name, name)
    -- https://stackoverflow.com/questions/56949901/strange-behavior-caused-by-debug-getinfo1-n-name
    -- bypass tailcall optimization
    name = name:gsub("_", "-")
    -- print(name)
    local ret = F [[
		function {name}() #! 
			lua <<-EOF
				require ("zmx").on_call("{module_name}","{name}","$@")
			EOF
		!#
	]]
    local ret = ret:gsub("#!", '{'):gsub("!#", "}")
    -- print(ret)
    return ret
end
local function write_file(file, data)
    local f = io.open(file, "w")
    f:write(data)

end

function _M.init()
    local module_name = "actions.test_sh"
    local m = require(module_name)
    local out = ""
    for k, f in pairs(m) do
        local zsh_fn = gen_zsh_export_function(module_name, k, f)
        out = out .. zsh_fn
    end
    write_file(os.getenv("ZMX_LUA") .. "/zmx.lua.gen.sh", out)
end

function _M.on_call(module, name, args)
    local name = name:gsub("-", "_")
    local m = require(module)
    m[name](args)
end

return _M
