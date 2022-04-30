#!/bin/bash

function init-zmx-lua() {
	lua <<EOF
	require "zmx".init()
EOF
	echo "soure gened sh"
	source $ZMX_LUA/zmx.lua.gen.sh
}

init-zmx-lua
