package main

import (
    "fmt"
    "github.com/alexj212/clogviewr"
    "github.com/alexj212/gox"
    "github.com/alexj212/gox/utilx"
    "github.com/droundy/goopt"
    "github.com/gdamore/tcell/v2"
    "github.com/potakhov/loge"
    "log"
    "os"
    "path/filepath"
    "strings"
)

var (
    logDir = goopt.String([]string{"--log"}, os.TempDir(), "log dir")
)
var (
    // OsSignal for sending
    OsSignal chan os.Signal

    // OnShutdownFunc function to be invoked on Service shutdown.
    OnShutdownFunc func(os.Signal)
)

func init() {

    _, appName := filepath.Split(os.Args[0])

    OsSignal = make(chan os.Signal, 1)
    OnShutdownFunc = defaultShutdown

    goopt.Description = func() string {
        return "loge test example"
    }
    goopt.Author = "Alex Jeannopoulos"
    goopt.ExtraUsage = ``
    goopt.Summary = fmt.Sprintf(`

Usage: %s [options]


`, appName)

    goopt.Version = fmt.Sprintf(
        `build information

`)

    //Parse options
    goopt.Parse(nil)
}

func defaultShutdown(sig os.Signal) {
    loge.Printf("caught sig: %v\n\n", sig)
    os.Exit(0)
}

type logeHandler struct {
    ui *clogviewr.UI
}

func (h *logeHandler) WriteOutTransaction(tr *loge.Transaction) {

    /*
       type BufferElement struct {
       	Timestamp   time.Time                  `json:"time"`
       	Timestring  [dateTimeStringLength]byte `json:"-"`
       	Message     string                     `json:"msg"`
       	Level       uint32                     `json:"-"`
       	Levelstring string                     `json:"level,omitempty"`
       	Data        map[string]interface{}     `json:"data,omitempty"`

    */

    for i, mesg := range tr.Items {
        line := mesg.Message

        event := &clogviewr.LogEvent{
            EventID:   fmt.Sprintf("%v:%d", tr.ID, i),
            Level:     clogviewr.LogLevel(mesg.Level),
            Message:   line,
            Source:    "loge",
            Timestamp: mesg.Timestamp,
            Data:      mesg.Data,
        }

        h.ui.AppendEvent(event)
    }

}
func (h *logeHandler) FlushTransactions() {

}

func main() {
    logPath := fmt.Sprintf("%v%v%v%v", *logDir, os.PathSeparator, os.Getpid(), os.PathSeparator)
    fmt.Printf("Logging to:  %v\n", logPath)

    name := gox.GetAppName()

    fmt.Printf("name: %v\n", name)

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

    ui.SetDetailsGenerator(
        func(evt *clogviewr.LogEvent) string {
            text := ""
            text = text + fmt.Sprintf("evt.Message  : %s\n", evt.Message)
            text = text + fmt.Sprintf("evt.Timestamp: %s\n", evt.Timestamp.String())
            text = text + fmt.Sprintf("evt.EventID  : %s\n", evt.EventID)
            text = text + fmt.Sprintf("evt.Level    : %d\n", evt.Level)
            text = text + fmt.Sprintf("evt.Source   : %s\n", evt.Source)
            text = text + "\n\n"

            if evt.Data != nil {
                data := evt.Data.(map[string]interface{})
                for k, v := range data {
                    text = text + fmt.Sprintf("%v   %v\n", k, v)
                }

            }
            return text
        })

    c := &logeHandler{ui: ui}

    logeShutdown := loge.Init(
        loge.Path("."),
        loge.EnableOutputConsole(true),
        loge.EnableOutputFile(false),
        loge.ConsoleOutput(os.Stdout),
        loge.EnableDebug(),
        loge.EnableError(),
        loge.EnableInfo(),
        loge.EnableWarning(),

        loge.Transports(func(list loge.TransactionList) []loge.Transport {
            transport := loge.WrapTransport(list, c)
            return []loge.Transport{transport}
        }),
    )

    defer logeShutdown()

    for i := 0; i < 1000; i++ {
        loge.Printf("%d hello world Printf", i)
        loge.Debug("%d hello world Debug", i)
        loge.Info("%d hello world Info", i)
        loge.Warn("%d hello world Warn", i)
        loge.With("uid", i).Error("%d hello world Error", i)
    }

    //
    //lines := make([]string, 0)
    //for i := 0; i < 1000; i++ {
    //    lines = append(lines, lorem.Sentence(20, 32))
    //}
    //
    //lastTime := time.Date(2000, 1, 1, 1, 1, 1, 1, time.Local)
    //for i, line := range lines {
    //    if len(line) > 0 && rand.Float64() < 0.4 || len(line) > 100 {
    //
    //        event := &clogviewr.LogEvent{
    //            EventID:   strconv.Itoa(i),
    //            Level:     clogviewr.LogLevelInfo,
    //            Message:   line,
    //            Source:    "S " + strconv.Itoa(i),
    //            Timestamp: time.Now(),
    //        }
    //
    //        ui.AppendEvent(event)
    //    }
    //}
    //
    //ui.SetAnchor(lastTime)
    ui.Run()
}
