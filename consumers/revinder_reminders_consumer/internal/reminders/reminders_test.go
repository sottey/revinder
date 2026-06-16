package reminders

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sottey/revinder/consumers/revinder_reminders_consumer/internal/bridge"
)

func TestBuildScriptIncludesReminderFields(t *testing.T) {
	dueAt := time.Date(2026, 6, 16, 20, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	notes := "spoken through kitchen Echo"

	script := buildScript(bridge.Item{
		Title:    "replace air filter",
		ListName: "Home",
		Notes:    &notes,
		DueAt:    &dueAt,
	})

	for _, want := range []string{
		`tell application "Reminders"`,
		`at list "Home"`,
		`name:"replace air filter"`,
		`body:"spoken through kitchen Echo"`,
		`due date:date "Tuesday, June 16, 2026 8:00:00 PM"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script = %q, want substring %q", script, want)
		}
	}
}

func TestBuildScriptUsesAllDayDueDate(t *testing.T) {
	dueAt := time.Date(2026, 6, 16, 0, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	script := buildScript(bridge.Item{
		Title:    "replace air filter",
		ListName: "Home",
		DueAt:    &dueAt,
		Metadata: map[string]any{
			"all_day": true,
		},
	})

	for _, want := range []string{
		`allday due date:date "Tuesday, June 16, 2026"`,
		`name:"replace air filter"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script = %q, want substring %q", script, want)
		}
	}
	if strings.Contains(script, `due date:date "Tuesday, June 16, 2026 12:00:00 AM"`) {
		t.Fatalf("script = %q, should not use timed due date for all-day item", script)
	}
}

func TestBuildScriptIgnoresNonBooleanAllDayMetadata(t *testing.T) {
	dueAt := time.Date(2026, 6, 16, 0, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	script := buildScript(bridge.Item{
		Title: "replace air filter",
		DueAt: &dueAt,
		Metadata: map[string]any{
			"all_day": "true",
		},
	})

	if !strings.Contains(script, `due date:date "Tuesday, June 16, 2026 12:00:00 AM"`) {
		t.Fatalf("script = %q, want timed due date", script)
	}
	if strings.Contains(script, "allday due date") {
		t.Fatalf("script = %q, should not use all-day due date", script)
	}
}

func TestProcessReturnsErrorOnNonMacOS(t *testing.T) {
	restoreRemindersTestHooks(t)
	currentGOOS = "linux"

	err := New().Process(context.Background(), bridge.Item{Title: "replace air filter"})
	if err == nil || err.Error() != "apple reminders consumer requires macOS" {
		t.Fatalf("Process() error = %v, want macOS error", err)
	}
}

func TestProcessReturnsErrorWhenOSAScriptMissing(t *testing.T) {
	restoreRemindersTestHooks(t)
	currentGOOS = "darwin"
	lookPath = func(file string) (string, error) {
		if file != "osascript" {
			t.Fatalf("lookPath file = %q, want osascript", file)
		}
		return "", exec.ErrNotFound
	}

	err := New().Process(context.Background(), bridge.Item{Title: "replace air filter"})
	if err == nil || !strings.Contains(err.Error(), "osascript not found") {
		t.Fatalf("Process() error = %v, want osascript not found error", err)
	}
}

func TestProcessReturnsErrorWhenListDoesNotExist(t *testing.T) {
	restoreRemindersTestHooks(t)
	currentGOOS = "darwin"
	lookPath = func(file string) (string, error) {
		return "/usr/bin/osascript", nil
	}
	runScript = func(ctx context.Context, script string) (string, error) {
		if script != `tell application "Reminders" to exists list "Home"` {
			t.Fatalf("script = %q, want list existence check", script)
		}
		return "false\n", nil
	}

	err := New().Process(context.Background(), bridge.Item{Title: "replace air filter"})
	if err == nil || err.Error() != `reminders list "Home" does not exist` {
		t.Fatalf("Process() error = %v, want missing list error", err)
	}
}

func TestProcessReturnsErrorWhenListCheckFails(t *testing.T) {
	restoreRemindersTestHooks(t)
	currentGOOS = "darwin"
	lookPath = func(file string) (string, error) {
		return "/usr/bin/osascript", nil
	}
	wantErr := errors.New("permission denied")
	runScript = func(ctx context.Context, script string) (string, error) {
		return "", wantErr
	}

	err := New().Process(context.Background(), bridge.Item{Title: "replace air filter"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Process() error = %v, want %v", err, wantErr)
	}
}

func TestProcessChecksListThenCreatesReminder(t *testing.T) {
	restoreRemindersTestHooks(t)
	currentGOOS = "darwin"
	lookPath = func(file string) (string, error) {
		return "/usr/bin/osascript", nil
	}
	var scripts []string
	runScript = func(ctx context.Context, script string) (string, error) {
		scripts = append(scripts, script)
		if len(scripts) == 1 {
			return "true\n", nil
		}
		return "", nil
	}

	err := New().Process(context.Background(), bridge.Item{
		Title:    "replace air filter",
		ListName: "Home",
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if len(scripts) != 2 {
		t.Fatalf("script count = %d, want 2", len(scripts))
	}
	if scripts[0] != `tell application "Reminders" to exists list "Home"` {
		t.Fatalf("first script = %q, want list check", scripts[0])
	}
	if !strings.Contains(scripts[1], `make new reminder at list "Home"`) {
		t.Fatalf("second script = %q, want reminder creation", scripts[1])
	}
}

func TestRemindersListExistsReturnsErrorForUnexpectedOutput(t *testing.T) {
	restoreRemindersTestHooks(t)
	runScript = func(ctx context.Context, script string) (string, error) {
		return "maybe\n", nil
	}

	exists, err := remindersListExists(context.Background(), "Home")
	if err == nil || !strings.Contains(err.Error(), `unexpected reminders list check output: "maybe"`) {
		t.Fatalf("remindersListExists() exists = %v, error = %v, want unexpected output error", exists, err)
	}
}

func TestBuildScriptFallsBackToHomeAndText(t *testing.T) {
	script := buildScript(bridge.Item{
		Text: "replace air filter",
	})

	for _, want := range []string{
		`at list "Home"`,
		`name:"replace air filter"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script = %q, want substring %q", script, want)
		}
	}
}

func TestAppleScriptStringEscapesQuotes(t *testing.T) {
	got := appleScriptString(`replace "office" filter`)
	want := `"replace \"office\" filter"`

	if got != want {
		t.Fatalf("appleScriptString() = %q, want %q", got, want)
	}
}

func restoreRemindersTestHooks(t *testing.T) {
	t.Helper()

	oldGOOS := currentGOOS
	oldLookPath := lookPath
	oldRunScript := runScript

	t.Cleanup(func() {
		currentGOOS = oldGOOS
		lookPath = oldLookPath
		runScript = oldRunScript
	})
}

func Example_remindersListExistsScript() {
	fmt.Println(`tell application "Reminders" to exists list ` + appleScriptString("Home"))
	// Output: tell application "Reminders" to exists list "Home"
}
