package reminders

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sottey/revinder/consumers/revinder_reminders_consumer/internal/bridge"
)

type Creator struct{}

var (
	currentGOOS = runtime.GOOS
	lookPath    = exec.LookPath
	runScript   = runOSAScript
)

func New() *Creator {
	return &Creator{}
}

func (c *Creator) Process(ctx context.Context, item bridge.Item) error {
	listName := reminderListName(item)
	if err := checkReady(ctx, listName); err != nil {
		return err
	}

	script := buildScript(item)
	_, err := runScript(ctx, script)
	return err
}

func checkReady(ctx context.Context, listName string) error {
	if currentGOOS != "darwin" {
		return fmt.Errorf("apple reminders consumer requires macOS")
	}

	if _, err := lookPath("osascript"); err != nil {
		return fmt.Errorf("osascript not found: %w", err)
	}

	exists, err := remindersListExists(ctx, listName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("reminders list %q does not exist", listName)
	}

	return nil
}

func remindersListExists(ctx context.Context, listName string) (bool, error) {
	output, err := runScript(ctx, `tell application "Reminders" to exists list `+appleScriptString(listName))
	if err != nil {
		return false, err
	}

	value := strings.TrimSpace(output)
	if strings.EqualFold(value, "true") {
		return true, nil
	}
	if strings.EqualFold(value, "false") {
		return false, nil
	}

	return false, fmt.Errorf("unexpected reminders list check output: %q", value)
}

func runOSAScript(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("osascript failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func buildScript(item bridge.Item) string {
	listName := reminderListName(item)
	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = strings.TrimSpace(item.Text)
	}

	properties := []string{
		"name:" + appleScriptString(title),
	}
	if item.Notes != nil && strings.TrimSpace(*item.Notes) != "" {
		properties = append(properties, "body:"+appleScriptString(strings.TrimSpace(*item.Notes)))
	}
	if item.DueAt != nil {
		if itemAllDay(item) {
			properties = append(properties, "allday due date:date "+appleScriptString(item.DueAt.Format("Monday, January 2, 2006")))
		} else {
			properties = append(properties, "due date:date "+appleScriptString(item.DueAt.Format("Monday, January 2, 2006 3:04:05 PM")))
		}
	}

	return fmt.Sprintf(
		`tell application "Reminders" to make new reminder at list %s with properties {%s}`,
		appleScriptString(listName),
		strings.Join(properties, ", "),
	)
}

func reminderListName(item bridge.Item) string {
	listName := strings.TrimSpace(item.ListName)
	if listName == "" {
		listName = "Home"
	}
	return listName
}

func itemAllDay(item bridge.Item) bool {
	value, ok := item.Metadata["all_day"].(bool)
	return ok && value
}

func appleScriptString(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
