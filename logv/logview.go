package logv

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"gitlab.com/tslocum/cview"
)

// LogLevel represents the log level
// LogView recognizes three log levels: Info, Warning and Error
// Warning and Error events can be highlighted
type LogLevel uint

const (
	// LogLevelInfo is default log
	LogLevelInfo = LogLevel(iota)
	// LogLevelWarning is the level for warnings
	LogLevelWarning
	// LogLevelError is the level for errors
	LogLevelError
)

// LogEvent that can be added to LogView.
// Contains following fields:
//
// - EventID - a string identifier of the event, used in event handlers, may contain only ASCII characters
//
// - Source - a source that produced the event, may contain only ASCII characters
//
// - Timestamp - an instant when the event was created/ingested
//
// - Level - the severity level of an event. Can be used to highlight errors and warnings
//
// - Message - the event contents
type LogEvent struct {
	EventID   string
	Source    string
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

func NewLogEvent(eventID string, message string) *LogEvent {
	// expand tabs to 4 spaces (not exactly how it should be done, but will work for now)
	msg := strings.Replace(message, "\t", "    ", -1)

	return &LogEvent{
		EventID: eventID,
		Level:   LogLevelInfo,
		Message: msg,
	}
}

type styledSpan struct {
	start int
	end   int
	style tcell.Style
}

type logEventLine struct {
	*LogEvent
	lineID     uint
	previous   *logEventLine
	next       *logEventLine
	styleSpans []styledSpan

	// start and end determine slice of LogEvent.Message this event line covers
	// for unwrapped string this will be the whole length of the message starting at position 0
	// for wrapped strings each line with order != 0 will cover its portion of main event message
	start int
	end   int

	// order indicate whether the single log event is split over multiple lines because of wrapping
	// if the event is not split, then order will be 0
	// otherwise the first line will have the order value of 1, the next line is 2 and so on
	order     int
	lineCount uint
}

func (e logEventLine) getLineCount() uint {
	return e.lineCount
}

func (e *logEventLine) copy() *logEventLine {
	eventCopy := &logEventLine{
		LogEvent:   e.LogEvent,
		lineID:     e.lineID,
		previous:   e.previous,
		next:       e.next,
		styleSpans: e.styleSpans,
		start:      e.start,
		end:        e.end,
		order:      e.order,
		lineCount:  e.lineCount,
	}
	return eventCopy
}

// LogView is a Box that displays log events
//
// LogView doesn't have border or scrollviews to allow easier copy-paste of events
// Events can be appended to a LogView
type LogView struct {
	*cview.Box

	firstEvent *logEventLine
	lastEvent  *logEventLine
	top        *logEventLine
	eventCount uint
	eventLimit uint

	highlightingEnabled bool
	highlightColors     map[string]tcell.Style

	highlightLevels bool
	warningBgColor  tcell.Color
	errorBgColor    tcell.Color

	// as new events are appended, older events are scrolled up, like tail -f
	following bool

	showHeader      bool
	showSource      bool
	showTimestamp   bool
	timestampFormat string
	wrap            bool

	textColor       tcell.Color
	backgroundColor tcell.Color
	defaultStyle    tcell.Style

	highlightPattern *regexp.Regexp

	hasFocus bool

	lastWidth, lastHeight int
	pageHeight, pageWidth int

	sync.RWMutex
}

// NewLogView returns a new log view.
func NewLogView() *LogView {
	logView := &LogView{
		Box:                 cview.NewBox(),
		showHeader:          false,
		showSource:          false,
		showTimestamp:       false,
		timestampFormat:     "15:04:05.000",
		wrap:                true,
		following:           true,
		highlightingEnabled: true,
		textColor:           cview.Styles.PrimaryTextColor,
		backgroundColor:     cview.Styles.PrimitiveBackgroundColor,
		warningBgColor:      tcell.ColorSaddleBrown,
		errorBgColor:        tcell.ColorIndianRed,
		highlightColors:     make(map[string]tcell.Style),
	}
	logView.Box.SetBorder(false)
	logView.updateDefaultStyle()
	return logView
}

// InvalidateHighlights forces recalculation of highlight patterns for all events in the log view.
// LogView calculates highlight spans once for each event when the event is appended. Any changes in highlighting
// will not be applied to the events that are already in the log view.
// To apply changes to all the events call this function.
// Warning: this might be a rather expensive operation
func (lv *LogView) InvalidateHighlights() {
	lv.Lock()
	defer lv.Unlock()

	lv.recolorizeLines()
}

// SetHighlightPattern sets new regular expression pattern to find spans that need to be highlighted
// setting this to nil disables highlighting.
//
// pattern is a regular expression where each matching named capturing group can be highlighted with a different color.
// Colors for any given group name can be set with SetHighlightColor, SetHighlightColorFg, SetHighlightColorBg
//
// Note:
// Updating pattern doesn't changes highlighting for previously appended events
// Call InvalidateHighlights() to force updating highlighting for all events in the log view.
func (lv *LogView) SetHighlightPattern(pattern string) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightPattern = regexp.MustCompile(pattern)
}

// SetHighlighting enables/disables event message highlighting according to the pattern set by SetHighlightPattern.
//
// Events appended when this setting was disabled will not be highlighted until InvalidateHighlights function is called.
func (lv *LogView) SetHighlighting(enable bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightingEnabled = enable
}

// SetHighlightColorFg sets foreground color for a named group "group". Default background color will be used
func (lv *LogView) SetHighlightColorFg(group string, foreground tcell.Color) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightColors[group] = tcell.StyleDefault.Foreground(foreground).Background(lv.backgroundColor)
}

// SetHighlightColorBg sets background color for a named group "group". Default foreground color will be used
func (lv *LogView) SetHighlightColorBg(group string, background tcell.Color) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightColors[group] = tcell.StyleDefault.Foreground(lv.textColor).Background(background)
}

// SetHighlightColor sets both foreground and background colors for a named group "group".
func (lv *LogView) SetHighlightColor(group string, foreground tcell.Color, background tcell.Color) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightColors[group] = tcell.StyleDefault.Foreground(foreground).Background(background)
}

// SetWarningColor sets the background color for events with level == LogLevelWarning.
// Event level highlighting can be turned on and off with SetLevelHighlighting function.
//
// Changing warning color will do nothing to the events that are already in the log view. To update
// highlighting of all events use InvalidateHighlights. Be warned: this is an expensive operation
func (lv *LogView) SetWarningColor(bgColor tcell.Color) {
	lv.Lock()
	defer lv.Unlock()

	lv.warningBgColor = bgColor
}

// SetErrorColor sets the background color for events with level == LogLevelError.
// Event level highlighting can be turned on and off with SetLevelHighlighting function.
//
// Changing error color will do nothing to the events that are already in the log view. To update
// highlighting of all events use InvalidateHighlights. Be warned: this is an expensive operation
func (lv *LogView) SetErrorColor(bgColor tcell.Color) {
	lv.Lock()
	defer lv.Unlock()

	lv.errorBgColor = bgColor
}

// SetLevelHighlighting enables background color highlighting for events based on severity level
func (lv *LogView) SetLevelHighlighting(enabled bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightLevels = enabled
}

// SetBorder does nothing
func (lv *LogView) SetBorder(show bool) {
	// do nothing
}

// Focus is called when this primitive receives focus.
func (lv *LogView) Focus(delegate func(p cview.Primitive)) {
	lv.Lock()
	defer lv.Unlock()

	// Implemented here with locking because this is used by layout primitives.
	lv.hasFocus = true
}

// HasFocus returns whether or not this primitive has focus.
func (lv *LogView) HasFocus() bool {
	lv.RLock()
	defer lv.RUnlock()

	// Implemented here with locking because this may be used in the "changed"
	// callback.
	return lv.hasFocus
}

// Draw draws this primitive onto the screen.
func (lv *LogView) Draw(screen tcell.Screen) {
	if !lv.GetVisible() {
		return
	}

	lv.Box.Draw(screen)

	lv.Lock()
	defer lv.Unlock()

	// Get the available size.
	x, y, width, height := lv.GetInnerRect()
	if height == 0 {
		return
	}

	lv.pageHeight = height
	lv.pageWidth = width
	if width != lv.lastWidth || height != lv.lastHeight && lv.wrap {
		lv.rewrapLines()
		if lv.following {
			// ensure correct top line when resizing
			lv.scrollToEnd()
		}
	}
	lv.lastWidth, lv.lastHeight = width, height

	line := y
	if lv.showHeader {
		lv.pageHeight--
		line = lv.drawHeader(screen, x, line)
	}

	top := lv.top
	for top != nil && line < y+height {
		lv.drawEvent(screen, x, line, top)
		line++
		top = top.next
	}
}

// AppendLogEvent appends an event to the log view
// If possible use AppendEvents to add multiple events at once
func (lv *LogView) AppendLogEvent(logEvent *LogEvent) {
	lv.Lock()
	defer lv.Unlock()

	lv.append(logEvent)
}

// AppendEvents appends multiple events in a single batch improving performance
func (lv *LogView) AppendEvents(events []*LogEvent) {
	lv.Lock()
	defer lv.Unlock()

	for _, e := range events {
		lv.append(e)
	}
}

// ScrollToBottom scrolls the log view to the last event
//
// This does not automatically enables following. User SetFollowing function to enable it
func (lv *LogView) ScrollToBottom() {
	lv.Lock()
	defer lv.Unlock()

	lv.scrollToEnd()
}

// SetFollowing enables/disables following mode. Following mode automatically scrolls log view up
// as new events are appended. Last event is always in the view
//
// Enabling following will automatically scroll to the last event
func (lv *LogView) SetFollowing(follow bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.following = follow
	if follow {
		lv.scrollToEnd()
	}
}

// *******************************
// internal implementation details

func (lv *LogView) scrollToEnd() {
	lv.top = lv.atOffset(lv.lastEvent, -(lv.pageHeight - 1))
}

func (lv *LogView) append(logEvent *LogEvent) {
	event := &logEventLine{
		LogEvent:  logEvent,
		lineCount: 1,
		lineID:    lv.eventCount + 1,
		start:     0,
		end:       len(logEvent.Message),
	}

	lv.insertAfter(lv.lastEvent, event)

	// process event
	lv.colorize(event)
	lv.calculateWrap(event)

	lv.eventCount += event.lineCount

	// if we're in following mode and have enough events to fill the page then update the top position
	if lv.following && lv.eventCount > uint(lv.pageHeight) {
		lv.top = lv.atOffset(lv.top, int(event.lineCount))
	}
}

// atOffset finds event that is at given offset from the starting event
// offset can be positive or negative
// if first or last event is reached then it is returned
func (lv *LogView) atOffset(start *logEventLine, offset int) *logEventLine {
	if offset == 0 {
		return start
	}
	current := start
	var steps int
	if offset > 0 {
		steps = offset
	} else {
		steps = -offset
	}
	for steps > 0 {
		if (current == lv.firstEvent && offset < 0) || (current == lv.lastEvent && offset > 0) {
			break
		}
		if offset < 0 {
			current = current.previous
		} else {
			current = current.next
		}
		steps--
	}
	return current
}

// calculateWrap splits the event line into multiple according to the wrap flag and window width
// for every split event it deletes previous wrapped lines and calculates wrapping from scratch
// new event lines with order >= 1 are created and inserted in the log list
// last event is returned
func (lv *LogView) calculateWrap(event *logEventLine) *logEventLine {
	if !lv.wrap || lv.pageWidth == 0 || len(event.Message) < lv.pageWidth {
		return event
	}
	if event.order != 0 { // first drop extra event lines
		event = lv.deleteWrapLines(event)
	}
	lines := len(event.Message) / lv.pageWidth
	if len(event.Message)%lv.pageWidth != 0 {
		lines++
	}
	event.order = 1
	event.start = 0
	event.end = lv.pageWidth
	event.lineCount = uint(lines)
	currentLine := event
	for i := 1; i < lines; i++ {
		nextLine := event.copy()
		nextLine.start = lv.pageWidth * i
		nextLine.order = i + 1
		if i == lines-1 {
			nextLine.end = nextLine.start + len(event.Message) - lv.pageWidth*i
		} else {
			nextLine.end = nextLine.start + lv.pageWidth
		}
		currentLine = lv.insertAfter(currentLine, nextLine)
	}
	return currentLine
}

func findFirstWrappedLine(event *logEventLine) *logEventLine {
	for event.order > 1 && event.previous != nil {
		event = event.previous
	}
	return event
}

// deleteWrapLines will delete all extra lines for an event
// if the event order is == 0, it will return the event
// otherwise it will fine the first event, change its order to 0
// and delete all subsequent events with order > 0
// it will update event count accordingly
func (lv *LogView) deleteWrapLines(event *logEventLine) *logEventLine {
	if event.order == 0 {
		return event
	}
	event = findFirstWrappedLine(event)
	event.order = 0
	event.start = 0
	event.end = len(event.Message)
	next := event.next
	for next.next != nil && next.next.order > 1 { // find event with order <= 1, that will be
		next = next.next
	}
	event.next = next.next
	next.next.previous = event
	return event
}

func (lv *LogView) colorize(event *logEventLine) {
	if event.order != 0 {
		panic(fmt.Errorf("cannot colorize wrapped line"))
	}
	defaultStyle := lv.defaultStyle
	useSpecialBg := false
	if lv.highlightLevels && event.Level != LogLevelInfo {
		useSpecialBg = true
		if event.Level == LogLevelWarning {
			defaultStyle = defaultStyle.Background(lv.warningBgColor)
		} else {
			defaultStyle = defaultStyle.Background(lv.errorBgColor)
		}
	}
	if lv.highlightingEnabled && lv.highlightPattern != nil {
		groupIndices := lv.highlightPattern.FindStringSubmatchIndex(event.Message)
		if len(groupIndices) == 0 {
			return
		}
		groupIndices = groupIndices[2:]
		names := lv.highlightPattern.SubexpNames()[1:]
		event.styleSpans = lv.buildSpans(event.Message, groupIndices, names, defaultStyle, useSpecialBg)
	} else {
		event.styleSpans = []styledSpan{
			{
				start: 0,
				end:   len(event.Message),
				style: defaultStyle,
			},
		}
	}
}

func (lv *LogView) buildSpans(text string, groupIndices []int, groupNames []string, defaultStyle tcell.Style, useDefaultBg bool) []styledSpan {
	currentPos := 0
	spans := make([]styledSpan, 0)

	_, dbg, _ := defaultStyle.Decompose()

	for i, name := range groupNames {
		st := groupIndices[i*2]
		en := groupIndices[i*2+1]
		if st != -1 && en != -1 {
			if st != currentPos {
				beforeSpan := styledSpan{
					start: currentPos,
					end:   st - 1,
					style: defaultStyle,
				}
				spans = append(spans, beforeSpan)
			}

			var style tcell.Style
			var ok bool
			if style, ok = lv.highlightColors[name]; !ok {
				continue
			}

			if useDefaultBg {
				style = style.Background(dbg)
			}

			matched := styledSpan{
				start: st,
				end:   en - 1,
				style: style,
			}
			spans = append(spans, matched)
			currentPos = en
		}
	}
	if currentPos < len(text)-1 {
		afterSpan := styledSpan{
			start: currentPos,
			end:   len(text) - 1,
			style: defaultStyle,
		}
		spans = append(spans, afterSpan)
	}
	return spans
}

func (lv *LogView) insertAfter(node *logEventLine, new *logEventLine) *logEventLine {
	if node == nil {
		lv.firstEvent = new
		lv.lastEvent = new
		lv.top = new
	} else {
		new.previous = node
		new.next = node.next
		if node.next != nil {
			node.next.previous = new
		}
		node.next = new
		if lv.lastEvent == node {
			lv.lastEvent = new
		}
	}
	return new
}

// unwrapLines removes all wrap lines
func (lv *LogView) unwrapLines() {
	event := lv.firstEvent
	for event != nil {
		event = lv.deleteWrapLines(event)
		event = event.next
	}
}

// recolorizeLines calculates spans for strings
// This must be called after unwrapLines, otherwise it may panic on any wrapped line
func (lv *LogView) recolorizeLines() {
	event := lv.firstEvent
	for event != nil {
		lv.colorize(event)
		event = event.next
	}
}

// rewrapLines recalculates string wrapping
func (lv *LogView) rewrapLines() {
	event := lv.firstEvent
	for event != nil {
		event = lv.calculateWrap(event)
		event = event.next
	}
}

func (lv *LogView) drawHeader(screen tcell.Screen, x int, y int) int {
	return y
}

// drawEvent draws single event on a single line
func (lv *LogView) drawEvent(screen tcell.Screen, x int, y int, event *logEventLine) {
	if lv.highlightingEnabled {
		lv.printLogLine(screen, x, y, event)
	} else {
		lv.printLogLineNoHighlights(screen, x, y, event)
	}
}

// TODO: implement runes and graphemes, now it will be print weird things for unicode characters
func (lv *LogView) printLogLine(screen tcell.Screen, x int, y int, event *logEventLine) {
	// find first styled span for the event
	spanIndex := 0
	for spanIndex < len(event.styleSpans) {
		if event.styleSpans[spanIndex].start <= event.start && event.styleSpans[spanIndex].end >= event.start {
			break
		}
		spanIndex++
	}
	if spanIndex == len(event.styleSpans) { // no colorization needed
		lv.printLogLineNoHighlights(screen, x, y, event)
		return
	}
	textPos := event.start
	i := x
	for textPos < event.end {
		screen.SetCell(i, y, event.styleSpans[spanIndex].style, rune(event.Message[textPos]))
		i++
		textPos++
		if textPos > event.styleSpans[spanIndex].end {
			spanIndex++
		}
	}
}

func (lv *LogView) printLogLineNoHighlights(screen tcell.Screen, x int, y int, event *logEventLine) {
	i := x
	for pos := event.start; pos < event.end; pos++ {
		screen.SetCell(i, y, lv.defaultStyle, rune(event.Message[pos]))
		i++
		if i >= lv.pageWidth {
			break
		}
	}
}

func (lv *LogView) updateDefaultStyle() {
	lv.defaultStyle = tcell.StyleDefault.Foreground(lv.textColor).Background(lv.backgroundColor)
}
