module pulsyflux

go 1.23.1

require pulse v1.0.0

require msgbus v1.0.0

require util v1.0.0

require github.com/google/uuid v1.6.0 // indirect

require golang.org/x/sync v0.8.0 // indirect

require github.com/google/wire/cmd/wire@latest v0.7.0

require sliceext v1.0.0

require contracts v1.0.0

replace pulse v1.0.0 => /pulse

replace msgbus v1.0.0 => /msgbus

replace util v1.0.0 => /util

replace sliceext v1.0.0 => /sliceext

replace contracts v1.0.0 => /contracts

replace containers v1.0.0 => /containers

replace httpcontainer v1.0.0 => /httpcontainer

replace msgbuscontainer v1.0.0 => /msgbuscontainer
