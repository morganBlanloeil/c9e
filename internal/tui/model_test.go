package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wescale/claude-dashboard/internal/display"
)

func TestNewModel_Initialization(t *testing.T) {
	m := NewModel("v1.0.0")
	if m.version != "v1.0.0" {
		t.Errorf("version = %q, want %q", m.version, "v1.0.0")
	}
	if m.view != viewList {
		t.Errorf("view = %d, want viewList (%d)", m.view, viewList)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.filtering {
		t.Error("filtering should be false initially")
	}
	if m.filter != "" {
		t.Errorf("filter = %q, want empty", m.filter)
	}
	if m.sortCol != SortByPID {
		t.Errorf("sortCol = %d, want SortByPID (%d)", m.sortCol, SortByPID)
	}
	if !m.sortAsc {
		t.Error("sortAsc should be true initially")
	}
}

func TestWindowSizeMsg_SetsDimensions(t *testing.T) {
	m := NewModel("test")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = updated.(Model)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}

func TestDataMsg_PopulatesRows(t *testing.T) {
	m := newTestModel(t)

	if len(m.rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(m.rows))
	}
	if len(m.filtered) != 3 {
		t.Fatalf("got %d filtered rows, want 3", len(m.filtered))
	}
	if m.rows[0].PID != 1001 {
		t.Errorf("first row PID = %d, want 1001", m.rows[0].PID)
	}
}

func TestDataMsg_Error(t *testing.T) {
	m := NewModel("test")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	testErr := &testError{msg: "test error"}
	updated, _ = m.Update(dataMsg{rows: nil, err: testErr})
	m = updated.(Model)

	if m.err == nil {
		t.Fatal("expected error to be set")
	}
	if m.err.Error() != "test error" {
		t.Errorf("err = %q, want %q", m.err.Error(), "test error")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

func TestNavigation_JDown(t *testing.T) {
	m := newTestModel(t)

	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", m.cursor)
	}

	// Move down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor after j j = %d, want 2", m.cursor)
	}

	// At bottom, should not go past
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor should stay at 2 at bottom, got %d", m.cursor)
	}
}

func TestNavigation_KUp(t *testing.T) {
	m := newTestModel(t)

	// Move to bottom
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Fatalf("cursor after G = %d, want 2", m.cursor)
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("cursor after k = %d, want 1", m.cursor)
	}
}

func TestNavigation_GJumpTop(t *testing.T) {
	m := newTestModel(t)

	// Move down first
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	// Jump to top
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}
}

func TestNavigation_GJumpBottom(t *testing.T) {
	m := newTestModel(t)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor after G = %d, want 2", m.cursor)
	}
}

func TestNavigation_KAtTopStays(t *testing.T) {
	m := newTestModel(t)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor after k at top = %d, want 0", m.cursor)
	}
}

func TestFilter_EnterAndType(t *testing.T) {
	m := newTestModel(t)

	// Enter filter mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	if !m.filtering {
		t.Fatal("expected filtering to be true after /")
	}
	if m.filter != "" {
		t.Errorf("filter = %q, want empty after entering filter mode", m.filter)
	}

	// Type 'b' to filter
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(Model)
	if m.filter != "b" {
		t.Errorf("filter = %q, want %q", m.filter, "b")
	}

	// Type 'e'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	if m.filter != "be" {
		t.Errorf("filter = %q, want %q", m.filter, "be")
	}

	// Should filter to rows containing "be" (project-beta)
	if len(m.filtered) != 1 {
		t.Errorf("filtered = %d rows, want 1 matching 'be'", len(m.filtered))
	}

	// Confirm with enter
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.filtering {
		t.Error("filtering should be false after enter")
	}
	if m.filter != "be" {
		t.Errorf("filter = %q, want %q after enter", m.filter, "be")
	}
}

func TestFilter_EscClears(t *testing.T) {
	m := newTestModel(t)

	// Enter filter mode and type
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Esc should clear filter
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.filtering {
		t.Error("filtering should be false after esc")
	}
	if m.filter != "" {
		t.Errorf("filter = %q, want empty after esc", m.filter)
	}
	if len(m.filtered) != 3 {
		t.Errorf("filtered = %d rows, want 3 after clearing filter", len(m.filtered))
	}
}

func TestFilter_Backspace(t *testing.T) {
	m := newTestModel(t)

	// Enter filter mode and type "ab"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(Model)
	if m.filter != "ab" {
		t.Fatalf("filter = %q, want %q", m.filter, "ab")
	}

	// Backspace
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)
	if m.filter != "a" {
		t.Errorf("filter after backspace = %q, want %q", m.filter, "a")
	}
}

func TestSort_CycleColumn(t *testing.T) {
	m := newTestModel(t)

	if m.sortCol != SortByPID {
		t.Fatalf("initial sortCol = %d, want SortByPID", m.sortCol)
	}

	// Press 's' to cycle
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(Model)
	if m.sortCol != SortByStatus {
		t.Errorf("sortCol after s = %d, want SortByStatus (%d)", m.sortCol, SortByStatus)
	}

	// Press 's' again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(Model)
	if m.sortCol != SortByCPU {
		t.Errorf("sortCol after s s = %d, want SortByCPU (%d)", m.sortCol, SortByCPU)
	}
}

func TestSort_ToggleDirection(t *testing.T) {
	m := newTestModel(t)

	if !m.sortAsc {
		t.Fatal("initial sortAsc should be true")
	}

	// Press 'S' to toggle
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = updated.(Model)
	if m.sortAsc {
		t.Error("sortAsc should be false after S")
	}

	// Press 'S' again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = updated.(Model)
	if !m.sortAsc {
		t.Error("sortAsc should be true after S S")
	}
}

func TestViewTransition_EnterDetail(t *testing.T) {
	m := newTestModel(t)

	if m.view != viewList {
		t.Fatalf("initial view = %d, want viewList", m.view)
	}

	// Press enter to go to detail
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.view != viewDetail {
		t.Errorf("view after enter = %d, want viewDetail (%d)", m.view, viewDetail)
	}
}

func TestViewTransition_DetailBackToList(t *testing.T) {
	m := newTestModel(t)

	// Go to detail
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	// Press esc to go back
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.view != viewList {
		t.Errorf("view after esc = %d, want viewList (%d)", m.view, viewList)
	}
}

func TestViewTransition_DetailBackWithQ(t *testing.T) {
	m := newTestModel(t)

	// Go to detail
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	// Press q to go back
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if m.view != viewList {
		t.Errorf("view after q in detail = %d, want viewList", m.view)
	}
}

func TestViewTransition_EnterOnEmptyList(t *testing.T) {
	m := NewModel("test")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	updated, _ = m.Update(dataMsg{rows: nil, err: nil})
	m = updated.(Model)

	// Press enter on empty list should stay in list view
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.view != viewList {
		t.Errorf("view after enter on empty = %d, want viewList", m.view)
	}
}

func TestDetailView_ShowsCorrectSession(t *testing.T) {
	m := newTestModel(t)

	// Move to second row
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	// Enter detail
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	r := m.selectedRow()
	if r == nil {
		t.Fatal("selectedRow is nil in detail view")
	}
	if r.PID != 2002 {
		t.Errorf("detail PID = %d, want 2002", r.PID)
	}

	// View should contain session info
	view := m.View()
	if !strings.Contains(view, "2002") {
		t.Error("detail view should contain PID 2002")
	}
}

func TestLogView_ToggleFollow(t *testing.T) {
	m := newTestModel(t)
	// Manually set up log view state (since we can't open real logs)
	m.view = viewLogs
	m.logFollow = false
	m.logFrom = viewList

	// Toggle follow with 'f'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(Model)
	if !m.logFollow {
		t.Error("logFollow should be true after f")
	}

	// Toggle again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(Model)
	if m.logFollow {
		t.Error("logFollow should be false after f f")
	}
}

func TestLogView_ToggleThinking(t *testing.T) {
	m := newTestModel(t)
	m.view = viewLogs
	m.logShowThink = false
	m.logFrom = viewList

	// Toggle thinking with 't'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(Model)
	if !m.logShowThink {
		t.Error("logShowThink should be true after t")
	}

	// Toggle again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(Model)
	if m.logShowThink {
		t.Error("logShowThink should be false after t t")
	}
}

func TestLogView_EscGoesBack(t *testing.T) {
	m := newTestModel(t)
	m.view = viewLogs
	m.logFrom = viewDetail

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.view != viewDetail {
		t.Errorf("view after esc in logs = %d, want viewDetail (%d)", m.view, viewDetail)
	}
}

func TestKillConfirmation_DShowsConfirm(t *testing.T) {
	m := newTestModel(t)

	// Press 'd' to start kill confirm
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)

	if m.confirm == nil {
		t.Fatal("confirm should be set after d on alive session")
	}
	if m.confirm.pid != 1001 {
		t.Errorf("confirm pid = %d, want 1001", m.confirm.pid)
	}
}

func TestKillConfirmation_NCancel(t *testing.T) {
	m := newTestModel(t)

	// Press 'd' then 'n'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)

	if m.confirm != nil {
		t.Error("confirm should be nil after n")
	}
}

func TestKillConfirmation_EscCancel(t *testing.T) {
	m := newTestModel(t)

	// Press 'd' then esc
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.confirm != nil {
		t.Error("confirm should be nil after esc")
	}
}

func TestKillConfirmation_DOnDeadSession(t *testing.T) {
	m := newTestModel(t)

	// Move to third row (dead session)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)

	// Press 'd' on dead session should not trigger confirm
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)

	if m.confirm != nil {
		t.Error("confirm should be nil for dead session")
	}
}

func TestTickMsg_TriggersRefresh(t *testing.T) {
	m := newTestModel(t)

	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Error("tickMsg should return a non-nil cmd (batch of fetch + tick)")
	}
}

func TestDataRefresh_UpdatesRows(t *testing.T) {
	m := newTestModel(t)

	// Send new data with different rows
	newRows := []display.Row{
		{
			PID:           9999,
			SessionID:     "newid123",
			FullSessionID: "newid123-full",
			Status:        display.StatusActive,
			CPU:           "5.0",
			Mem:           "2.0",
			Cwd:           "~/new-project",
			RawCwd:        "/home/user/new-project",
			UptimeSec:     100,
			IdleSec:       10,
			LastAction:    "new action",
			Alive:         true,
			LogPath:       "",
			Turns:         1,
			Cost:          "",
			CostValue:     0,
			InputTokens:   0,
			OutputTokens:  0,
			CostModel:     "",
			HasUsageData:  false,
		},
	}
	updated, _ := m.Update(dataMsg{rows: newRows, err: nil})
	m = updated.(Model)

	if len(m.rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(m.rows))
	}
	if m.rows[0].PID != 9999 {
		t.Errorf("row PID = %d, want 9999", m.rows[0].PID)
	}
}

func TestView_LoadingBeforeWindowSize(t *testing.T) {
	m := NewModel("test")
	view := m.View()
	if view != "Loading..." {
		t.Errorf("View before WindowSizeMsg = %q, want %q", view, "Loading...")
	}
}

func TestView_ListContainsSessionData(t *testing.T) {
	m := newTestModel(t)

	view := m.View()
	if !strings.Contains(view, "1001") {
		t.Error("list view should contain PID 1001")
	}
	if !strings.Contains(view, "2002") {
		t.Error("list view should contain PID 2002")
	}
	if !strings.Contains(view, "3003") {
		t.Error("list view should contain PID 3003")
	}
	if !strings.Contains(view, "Claude Code Dashboard") {
		t.Error("list view should contain title")
	}
}

func TestView_ListShowsActiveCount(t *testing.T) {
	m := newTestModel(t)
	view := m.View()
	if !strings.Contains(view, "1 active") {
		t.Error("list view should show '1 active'")
	}
	if !strings.Contains(view, "1 idle") {
		t.Error("list view should show '1 idle'")
	}
	if !strings.Contains(view, "1 dead") {
		t.Error("list view should show '1 dead'")
	}
}

func TestFilter_CursorAdjustsWhenFilteredListShrinks(t *testing.T) {
	m := newTestModel(t)

	// Move cursor to last row
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Fatalf("cursor after G = %d, want 2", m.cursor)
	}

	// Enter filter that matches only 1 row
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)

	// Cursor should be adjusted to fit the filtered list
	if m.cursor >= len(m.filtered) {
		t.Errorf("cursor = %d, filtered len = %d; cursor should be within bounds", m.cursor, len(m.filtered))
	}
}

func TestWithHomeDir(t *testing.T) {
	m := NewModel("test")
	if m.homeDir != "" {
		t.Errorf("default homeDir = %q, want empty", m.homeDir)
	}

	m2 := m.WithHomeDir("/tmp/fake")
	if m2.homeDir != "/tmp/fake" {
		t.Errorf("homeDir after WithHomeDir = %q, want %q", m2.homeDir, "/tmp/fake")
	}
	// Original should be unmodified
	if m.homeDir != "" {
		t.Errorf("original homeDir changed to %q", m.homeDir)
	}
}

func TestFetchRows_WithFakeHome(t *testing.T) {
	homeDir := setupFakeClaudeHome(t)

	rows, err := fetchRows(homeDir)
	if err != nil {
		t.Fatalf("fetchRows returned error: %v", err)
	}

	// We created 2 sessions; both should appear as DEAD since PIDs don't exist
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	for _, r := range rows {
		if r.Status != display.StatusDead {
			t.Errorf("row PID %d status = %s, want DEAD", r.PID, r.Status)
		}
	}
}

func TestFetchRows_EmptyHome(t *testing.T) {
	homeDir := t.TempDir()
	rows, err := fetchRows(homeDir)
	if err != nil {
		t.Fatalf("fetchRows returned error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("got %d rows from empty home, want 0", len(rows))
	}
}

func TestSort_RowOrderChanges(t *testing.T) {
	m := newTestModel(t)

	// Default sort is by PID ascending
	if m.filtered[0].PID != 1001 {
		t.Fatalf("first row PID = %d, want 1001", m.filtered[0].PID)
	}

	// Toggle descending
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = updated.(Model)
	if m.filtered[0].PID != 3003 {
		t.Errorf("first row PID after desc = %d, want 3003", m.filtered[0].PID)
	}
}

func TestNotifyToggle(t *testing.T) {
	m := newTestModel(t)
	initial := m.notifyEnabled

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)
	if m.notifyEnabled == initial {
		t.Error("notifyEnabled should toggle after 'n'")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)
	if m.notifyEnabled != initial {
		t.Error("notifyEnabled should toggle back after second 'n'")
	}
}
