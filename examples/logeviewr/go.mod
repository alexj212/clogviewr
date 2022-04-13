module logeviewr

go 1.17

replace (
	github.com/alexj212/clogviewr => ../../
	github.com/alexj212/gox => ../../../gox/
)

require (
	github.com/alexj212/clogviewr v0.0.0-00010101000000-000000000000
	github.com/alexj212/gox v0.0.0-20220220190923-4e49efc47a5e
	github.com/droundy/goopt v0.0.0-20220217183150-48d6390ad4d1
	github.com/gdamore/tcell/v2 v2.4.1-0.20210828201608-73703f7ed490
	github.com/potakhov/loge v0.2.0
)

require (
	code.rocketnine.space/tslocum/cbind v0.1.5 // indirect
	code.rocketnine.space/tslocum/cview v1.5.7 // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.14-0.20210830053702-dc8fe66265af // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/potakhov/cache v0.0.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	golang.org/x/sys v0.0.0-20220318055525-2edf467146b5 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
)
