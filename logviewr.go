package clogviewr

import (
    "code.rocketnine.space/tslocum/cbind"
    "code.rocketnine.space/tslocum/cview"
    "fmt"
    "github.com/alexj212/gox"
    "github.com/gdamore/tcell/v2"
    "log"
    "strings"
    "sync"
    "time"
)

type CmdExecFunc func(string)

type AppMode int

type UI struct {
    app             *cview.Application
    histogram       *LogVelocityView
    logView         *LogView
    statusView      *cview.TextView
    logScreenLayout *cview.Flex
    exitModal       *cview.Modal
    detailsModal    *cview.Modal
    helpModal       *cview.Modal
    inputField      *cview.InputField
    focusManager    *cview.FocusManager
    cmdExecFunc     CmdExecFunc
    mode            AppMode
    cmdHistory      []string
    cmdHistoryPos   int

    lastSearch           string
    lastSearchEventIDHit string

    detailsGenerator func(evt *LogEvent) (text string)
    sync.RWMutex
}

const (
    ExitModal AppMode = iota
    InfoDialogModal
    CommandEntry
    LogViewer
)

func (ui *UI) IsLogViewerVisible() bool {
    ui.RLock()
    defer ui.RUnlock()
    return ui.mode == LogViewer
}

func (ui *UI) IsInfoDialogModalVisible() bool {
    ui.RLock()
    defer ui.RUnlock()
    return ui.mode == InfoDialogModal
}

func (ui *UI) IsCommandEntryVisible() bool {
    ui.RLock()
    defer ui.RUnlock()
    return ui.mode == CommandEntry
}

func (ui *UI) IsExitModalVisible() bool {
    ui.RLock()
    defer ui.RUnlock()
    return ui.mode == ExitModal
}

func (ui *UI) ShowExitModal() {
    ui.Lock()
    defer ui.Unlock()
    ui.mode = ExitModal
    ui.exitModal.SetVisible(true)
    ui.app.SetRoot(ui.exitModal, true)
    ui.app.QueueUpdateDraw(func() {})
}

func (ui *UI) ShowLogEventDetails(evt *LogEvent) {
    if ui.detailsGenerator == nil {
        ui.ShowDetailsModal("Log Entry Details", "detailsGenerator not set")
        return
    }

    text := ui.detailsGenerator(evt)
    ui.ShowDetailsModal("Log Entry Details", text)
}

func (ui *UI) SetDetailsGenerator(g func(evt *LogEvent) string) {
    ui.detailsGenerator = g
}

func Details(evt *LogEvent) (text string) {
    text = text + fmt.Sprintf("evt.Message  : %s\n", evt.Message)
    text = text + fmt.Sprintf("evt.Timestamp: %s\n", evt.Timestamp.String())
    text = text + fmt.Sprintf("evt.EventID  : %s\n", evt.EventID)
    text = text + fmt.Sprintf("evt.Level    : %d\n", evt.Level)
    text = text + fmt.Sprintf("evt.Source   : %s\n", evt.Source)
    text = text + "\n\n"
    return
}

func (ui *UI) ShowDetailsModal(title, message string) {
    ui.Lock()
    defer ui.Unlock()

    ui.mode = InfoDialogModal

    ui.detailsModal.SetTitle(title)
    ui.detailsModal.SetText(message)
    ui.app.SetRoot(ui.detailsModal, true)
    ui.app.QueueUpdateDraw(func() {})
}

func (ui *UI) ShowInputField() {
    ui.Lock()
    defer ui.Unlock()

    ui.mode = CommandEntry
    ui.inputField.SetText("")
    ui.inputField.SetVisible(true)
    ui.focusManager.Focus(ui.inputField)
    ui.SetStatusViewText("Enter a command: :top, :1, :bottom, /<expression>, exit etc.")
    ui.app.SetRoot(ui.logScreenLayout, true)
    ui.app.QueueUpdateDraw(func() {})

    ui.focusManager.Focus(ui.inputField)
}

func (ui *UI) ShowLogViewer() {
    ui.Lock()
    defer ui.Unlock()

    ui.mode = LogViewer

    ui.SetStatusViewText("")
    ui.inputField.SetVisible(false)
    ui.focusManager.Focus(ui.logView)
    ui.app.SetRoot(ui.logScreenLayout, true)
    ui.statusView.ScrollToEnd()
    ui.app.QueueUpdateDraw(func() {})
}

func wrap(f func()) func(ev *tcell.EventKey) *tcell.EventKey {
    return func(ev *tcell.EventKey) *tcell.EventKey {
        f()
        return nil
    }
}

// CreateAppUI creates base UI layout
func CreateAppUI() *UI {
    ui := &UI{}

    ui.app = cview.NewApplication()
    ui.app.EnableMouse(true)

    ui.logView = NewLogView()
    ui.logView.SetBorder(false)
    ui.logView.SetLineWrap(false)

    ui.histogram = NewLogVelocityView(1 * time.Second)

    ui.inputField = cview.NewInputField()
    ui.inputField.SetLabel("")
    ui.inputField.SetFieldWidth(0)

    ui.statusView = cview.NewTextView()
    ui.statusView.SetText("")
    ui.statusView.SetTextColor(tcell.ColorBlack)
    ui.statusView.SetBackgroundColor(tcell.ColorWhite)
    ui.statusView.SetVisible(true)
    ui.statusView.SetDynamicColors(true)
    ui.statusView.SetChangedFunc(func() {
        ui.app.Draw()
    })

    ui.logScreenLayout = cview.NewFlex()
    //ui.logScreenLayout.SetPadding(1, 1, 1, 1) // APJ for border
    ui.logScreenLayout.SetDirection(cview.FlexRow)
    ui.logScreenLayout.AddItem(ui.logView, 0, 1, true)
    ui.logScreenLayout.AddItem(cview.NewBox(), 1, 1, false)
    ui.logScreenLayout.AddItem(ui.statusView, 1, 1, false)
    ui.logScreenLayout.AddItem(ui.inputField, 1, 1, false)
    ui.logScreenLayout.SetBorder(true)
    ui.logScreenLayout.SetTitle(gox.GetAppName())

    ui.exitModal = cview.NewModal()
    ui.exitModal.SetTitle("Quit Application")
    ui.exitModal.SetText("Do you want to quit the application?")
    ui.exitModal.AddButtons([]string{"Quit", "Cancel"})
    ui.exitModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
        if buttonLabel == "Quit" {
            ui.app.Stop()
            return
        }

        ui.ShowLogViewer()
        return
    })
    ui.exitModal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        if event.Rune() == 'q' || event.Rune() == 'Q' || event.Key() == tcell.KeyESC {
            ui.app.Stop()
            return nil
        }
        return nil
    })

    ui.detailsModal = cview.NewModal()
    ui.detailsModal.SetBorder(true)

    ui.detailsModal.SetTextAlign(cview.AlignLeft)
    ui.detailsModal.AddButtons([]string{"Ok"})
    ui.detailsModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
        ui.ShowLogViewer()
        return
    })
    ui.detailsModal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        ui.ShowLogViewer()
        return nil
    })

    ui.focusManager = cview.NewFocusManager(ui.app.SetFocus)
    ui.focusManager.SetWrapAround(true)
    ui.focusManager.Add(ui.logView, ui.inputField)

    inputHandler := cbind.NewConfiguration()
    for _, key := range cview.Keys.MovePreviousField {
        err := inputHandler.Set(key, wrap(ui.focusManager.FocusPrevious))
        if err != nil {
            log.Fatal(err)
        }
    }
    for _, key := range cview.Keys.MoveNextField {
        err := inputHandler.Set(key, wrap(ui.focusManager.FocusNext))
        if err != nil {
            log.Fatal(err)
        }
    }

    inputHandler.SetKey(tcell.ModNone, tcell.KeyESC, func(ev *tcell.EventKey) *tcell.EventKey {

        if ui.IsCommandEntryVisible() {
            ui.inputField.SetText("")
            ui.ShowLogViewer()
            return nil
        }

        if ui.IsInfoDialogModalVisible() {
            ui.ShowLogViewer()
            return nil
        }

        if ui.IsLogViewerVisible() {
            ui.ShowInputField()
            return nil
        }

        if ui.IsExitModalVisible() {
            ui.ShowLogViewer()
            return nil
        }

        return nil
    })

    inputHandler.SetKey(tcell.ModNone, tcell.KeyUp, func(ev *tcell.EventKey) *tcell.EventKey {
        if ui.IsCommandEntryVisible() {
            ui.cmdHistoryPos--
            if ui.cmdHistoryPos < 0 {
                ui.cmdHistoryPos = len(ui.cmdHistory) - 1
            }

            if len(ui.cmdHistory) > 0 && ui.cmdHistoryPos < len(ui.cmdHistory) {
                ui.inputField.SetText(ui.cmdHistory[ui.cmdHistoryPos])
            }
            return nil
        }
        return ev
    })

    inputHandler.SetKey(tcell.ModNone, tcell.KeyDown, func(ev *tcell.EventKey) *tcell.EventKey {
        if ui.IsCommandEntryVisible() {
            ui.cmdHistoryPos++
            if ui.cmdHistoryPos > len(ui.cmdHistory)-1 {
                ui.cmdHistoryPos = 0
            }

            if ui.cmdHistoryPos < len(ui.cmdHistory) {
                ui.inputField.SetText(ui.cmdHistory[ui.cmdHistoryPos])
            }
            return nil
        }
        return ev
    })

    inputHandler.Set("q", func(ev *tcell.EventKey) *tcell.EventKey {

        if ui.IsLogViewerVisible() {
            ui.ShowExitModal()
            return nil
        }

        return ev
    })

    inputHandler.SetKey(tcell.ModNone, tcell.KeyEnter, func(ev *tcell.EventKey) *tcell.EventKey {

        if ui.IsCommandEntryVisible() {

            cmd := ui.inputField.GetText()
            ui.handleCommandEntered(cmd)
            return nil
        }

        if ui.IsLogViewerVisible() {
            currEvent := ui.logView.GetCurrentEvent()
            ui.ShowLogEventDetails(currEvent)
            return nil
        }

        if ui.IsInfoDialogModalVisible() {
            ui.ShowLogViewer()
            return nil
        }

        return nil
    })

    ui.app.SetInputCapture(inputHandler.Capture)
    ui.cmdExecFunc = ui.defaultCommandHandler
    ui.ShowLogViewer()
    return ui
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

func (ui *UI) SetShowTimestamp(b bool) {
    ui.logView.SetShowTimestamp(b)
}

func (ui *UI) SetHighlightCurrentEvent(b bool) {
    ui.logView.SetHighlightCurrentEvent(b)
}

func (ui *UI) SetLevelHighlighting(b bool) {
    ui.logView.SetLevelHighlighting(b)
}

func (ui *UI) SetHighlighting(b bool) {
    ui.logView.SetHighlighting(b)
}

func (ui *UI) SetWarningBgColor(c tcell.Color) {
    ui.logView.SetWarningBgColor(c)
}

func (ui *UI) SetErrorBgColor(c tcell.Color) {
    ui.logView.SetErrorBgColor(c)
}

func (ui *UI) SetHighlightPattern(pattern string) {
    ui.logView.SetHighlightPattern(pattern)
}

func (ui *UI) AppendEvent(event *LogEvent) {
    ui.logView.AppendEvent(event)
}

func (ui *UI) SetAnchor(lastTime time.Time) {
    ui.histogram.SetAnchor(lastTime)
}

func (ui *UI) SetExecuteCmdFunc(f CmdExecFunc) {
    ui.cmdExecFunc = f
}

func (ui *UI) handleCommandEntered(cmd string) {
    ui.Lock()
    defer ui.Unlock()

    if cmd != "" {
        ui.cmdHistory = append(ui.cmdHistory, cmd)
    }
    if ui.cmdExecFunc != nil {
        ui.cmdExecFunc(cmd)
    }

}

func (ui *UI) defaultCommandHandler(cmd string) {
    ui.inputField.SetText("")
    ui.ShowDetailsModal("cmd exec", fmt.Sprintf("cmd exec: %s", cmd))
}

func (ui *UI) GetCommandHistory() []string {
    ui.RLock()
    defer ui.RUnlock()

    return ui.cmdHistory
}

func (ui *UI) HandleSearch(s string) {

    ui.SetStatusViewText(fmt.Sprintf("handleSearch: %s", s))
    ui.inputField.SetText("")
    if s == "" {
        s = ui.lastSearch
    }

    if s == "" {
        ui.SetStatusViewText(fmt.Sprintf("Unable to search on empty pattern"))
        return
    }

    ui.lastSearch = s

    hits := ui.logView.FindTotalMatches(func(event *LogEvent) bool {
        if strings.Contains(event.Message, s) {
            return true
        }

        return false
    })

    event := ui.logView.FindMatchingEvent(ui.lastSearchEventIDHit,
        func(event *LogEvent) bool {
            if strings.Contains(event.Message, s) {
                return true
            }

            return false
        })

    if event != nil {
        ui.lastSearchEventIDHit = event.EventID
        ui.SetStatusViewText(fmt.Sprintf("Found EventID: %s total maches: %d", event.EventID, hits))
        ui.logView.ScrollToEventID(event.EventID)
        ui.logView.RefreshHighlights()
        return
    }

    ui.lastSearchEventIDHit = ""
    ui.SetStatusViewText(fmt.Sprintf("Unable to find search pattern: %s", s))
    return

}

func (ui *UI) HandleGotoLine(s string) {

    if s == "top" || s == "1" {
        ui.logView.ScrollToTop()
        ui.inputField.SetText("")
        return
    }
    if s == "bottom" {
        ui.logView.ScrollToBottom()
        ui.inputField.SetText("")
        return
    }

    ui.SetStatusViewText(fmt.Sprintf("unknown navigate value: %s", s))
}

func (ui *UI) SetStatusViewText(message string) {
    ui.statusView.SetText(message)
}

func (ui *UI) SetStatusViewTextColor(color tcell.Color) {
    ui.statusView.SetTextColor(color)
}

func (ui *UI) SetInputFieldLabel(s string) {
    ui.inputField.SetLabel(s)
}

func (ui *UI) SetTitle(s string) {
    ui.logScreenLayout.SetTitle(s)
}

func (ui *UI) SetTitleAlign(s int) {
    ui.logScreenLayout.SetTitleAlign(s)
}

func (ui *UI) SetTitleColor(s tcell.Color) {
    ui.logScreenLayout.SetTitleColor(s)
}
