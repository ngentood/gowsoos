package banner

import (
	"fmt"
)

const (
	SoftPink  = "\033[38;5;218m"
	LightPink = "\033[38;5;225m"
	Reset     = "\033[0m"
)

func PrintBanner() {
	fmt.Printf(`
%s    ____        __        __  _
%s   / __ \____  / /_____  / /_(_)___  ____
%s  / /_/ / __ \/ __/ __ \/ __/ / __ \/ __ \
%s / ____/ /_/ / /_/ /_/ / /_/ / /_/ / / / /
%s/_/    \____/\__/\____/\__/_/\____/_/ /_/
%s
%s  High-Performance SSH over HTTP WebSocket Proxy
%s
`, SoftPink, LightPink, SoftPink, LightPink, SoftPink, LightPink, SoftPink, Reset)
}