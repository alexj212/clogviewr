package main

import (
    "code.rocketnine.space/tslocum/cview"
    "fmt"
    "github.com/alexj212/clogviewr"
    "github.com/araddon/dateparse"
    "github.com/gdamore/tcell/v2"
    "io/ioutil"
    "log"
    "math/rand"
    "regexp"
    "strconv"
    "strings"
    "time"
)

type UI struct {
    app       *cview.Application
    histogram *clogviewr.LogVelocityView
    logView   *clogviewr.LogView
}

// CreateAppUI creates base UI layout
func CreateAppUI() *UI {
    app := cview.NewApplication()
    app.EnableMouse(true)
    app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        if clogviewr.HitShortcut(event, []string{"q"}) {
            app.Stop()
        }
        return event
    })

    logView := clogviewr.NewLogView()
    logView.SetBorder(false)

    histogramView := clogviewr.NewLogVelocityView(1 * time.Second)

    flex := cview.NewFlex()
    flex.SetDirection(cview.FlexRow)
    flex.AddItem(logView, 0, 1, true)
    flex.AddItem(histogramView, 5, 1, false)

    app.SetRoot(flex, true)

    return &UI{
        histogram: histogramView,
        logView:   logView,
        app:       app,
    }
}

// Run starts the application event loop
func (ui *UI) Run() {
    if err := ui.app.Run(); err != nil {
        log.Fatalln(err)
    }
}

// Stop the UI
func (ui *UI) Stop() {
    ui.app.Stop()
}

func main() {
    ui := CreateAppUI()

    ui.logView.SetHighlightPattern(`(?P<lavender>[\d.]+)\s+(-)\s+(?P<lightgreen>.*)\s+\[(?P<yellow>[^]]+)]\s+"(?P<cadetblue>.*)"\s+(?P<skyblue_maroon>\d{3})\s+(?P<blanchedalmond>\d+)`)

    ui.logView.SetErrorBgColor(tcell.Color52)
    ui.logView.SetWarningBgColor(tcell.Color100)

    ui.logView.SetHighlighting(true)
    ui.logView.SetLevelHighlighting(true)

    ui.logView.SetHighlightCurrentEvent(true)
    ui.logView.SetShowTimestamp(true)

    content, err := ioutil.ReadFile("examples/apache.log")

    if err != nil {
        log.Fatal(err)
    }

    dtparse := regexp.MustCompile(`\[(.*)]`)
    status := regexp.MustCompile(`404|503`)
    lines := strings.Split(string(content), "\n")

    start := time.Now().Add(-15 * time.Minute)
    current := start

    lastTime := time.Date(2000, 1, 1, 1, 1, 1, 1, time.Local)
    for i, line := range lines {
        if len(line) > 0 && rand.Float64() < 0.4 || len(line) > 100 {
            st := status.FindString(line)
            date := dtparse.FindStringSubmatch(line)
            dt, err := dateparse.ParseLocal(date[1])
            if err != nil {
                panic(err)
            }
            level := clogviewr.LogLevelInfo
            if st == "404" {
                level = clogviewr.LogLevelWarning
            } else if st == "503" {
                level = clogviewr.LogLevelError
            }
            event := &clogviewr.LogEvent{
                EventID:   strconv.Itoa(i),
                Level:     level,
                Message:   line,
                Source:    "S " + strconv.Itoa(i),
                Timestamp: dt,
            }
            if dt.After(lastTime) {
                lastTime = dt
            }
            //current = current.Add(time.Duration(rand.Float64() / 2 * float64(time.Second)))
            ui.logView.AppendEvent(event)
            ui.histogram.AppendLogEvent(event)
        }
    }

    ui.histogram.SetAnchor(lastTime)

    diff := current.Sub(start)

    //event := logv.NewLogEvent("1", "20:17:51.894 [sqsTaskExecutor-10] ERROR  c.s.d.l.s.s.CopyrightDetectionService - This is the extra long line which originally said just this text that follows: Stored copyright data for pkg:npm/%40mpen/rollup-plugin-clean@0.1.8?checksum=sha1:097f0110bbc8aa5bc1026f2d689f45dcf98fcbc5&sonatype_repository=npmjs.org&type=tgz")
    //event.Level = logv.LogLevelWarning
    //ui.logView.AppendEvent(event)

    ui.Run()

    fmt.Println(diff)
}
