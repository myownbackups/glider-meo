package main

import (
	// comment out the services you don't need to make the compiled binary smaller.
	_ "github.com/meoww-bot/glider/service/dhcpd"

	// comment out the protocols you don't need to make the compiled binary smaller.
	_ "github.com/meoww-bot/glider/proxy/redir"
	_ "github.com/meoww-bot/glider/proxy/tproxy"
	_ "github.com/meoww-bot/glider/proxy/unix"
	_ "github.com/meoww-bot/glider/proxy/vsock"
)
