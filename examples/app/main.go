package main

import (
    "fmt"
    "github.com/alexj212/clogviewr"
    "github.com/alexj212/gox/lorem"
    "github.com/alexj212/gox/utilx"
    "github.com/gdamore/tcell/v2"
    "log"
    "math/rand"
    "os"
    "strconv"
    "strings"
    "time"
)

func main() {
    err := utilx.CheckEnvironmentForRendering()
    if err != nil {
        log.Fatalln(err)
    }

    ui := clogviewr.CreateAppUI()

    ui.SetHighlightPattern(`(?P<lavender>[\d.]+)\s+(-)\s+(?P<lightgreen>.*)\s+\[(?P<yellow>[^]]+)]\s+"(?P<cadetblue>.*)"\s+(?P<skyblue_maroon>\d{3})\s+(?P<blanchedalmond>\d+)`)

    ui.SetErrorBgColor(tcell.Color52)
    ui.SetWarningBgColor(tcell.Color100)

    ui.SetHighlighting(true)
    ui.SetLevelHighlighting(true)

    ui.SetHighlightCurrentEvent(true)
    ui.SetShowTimestamp(true)

    ui.SetStatusViewText("Welcome to CLogViewr!- Pres ESC to enter command entry, Q to exit")
    ui.SetStatusViewTextColor(tcell.ColorRed)

    ui.SetInputFieldLabel(" [#ffff00]CMD:  [#0000ff]")

    ui.SetExecuteCmdFunc(func(s string) {
        if s == "" {
            return
        }

        if s == "exit" || s == "quit" {
            ui.Stop()
            os.Exit(1)
            return
        }

        if strings.HasPrefix(s, "/") {
            ui.HandleSearch(s[1:])
            return
        }
        if strings.HasPrefix(s, ":") {
            ui.HandleGotoLine(s[1:])
            return
        }

        ui.SetStatusViewText(fmt.Sprintf("unknown command: %s", s))
    })

    lines := make([]string, 0)
    for i := 0; i < 1000; i++ {
        lines = append(lines, lorem.Sentence(20, 32))
    }

    lastTime := time.Date(2000, 1, 1, 1, 1, 1, 1, time.Local)
    for i, line := range lines {
        if len(line) > 0 && rand.Float64() < 0.4 || len(line) > 100 {

            event := &clogviewr.LogEvent{
                EventID:   strconv.Itoa(i),
                Level:     clogviewr.LogLevelInfo,
                Message:   line,
                Source:    "S " + strconv.Itoa(i),
                Timestamp: time.Now(),
            }

            ui.AppendEvent(event)
        }
    }

    ui.SetAnchor(lastTime)
    ui.Run()
}
