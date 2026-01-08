package components

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vaws/internal/model"
	"vaws/internal/ui/theme"
)

// DynamoDBQueryDialog is a dialog for entering DynamoDB query parameters.
type DynamoDBQueryDialog struct {
	tableName       string
	pkName          string
	skName          string
	width           int
	height          int
	active          bool
	isQuery         bool // true = query, false = scan
	focusIndex      int
	pkInput         textinput.Model
	skInput         textinput.Model
	limitInput      textinput.Model
	filterAttrInput textinput.Model
	filterValInput  textinput.Model
	skCondition     int // Index into skConditions
	filterCondition int // Index into filterConditions
}

var skConditions = []struct {
	label string
	value model.SortKeyCondition
}{
	{"= (equals)", model.SortKeyConditionEquals},
	{"begins_with", model.SortKeyConditionBeginsWith},
	{"< (less than)", model.SortKeyConditionLessThan},
	{"<= (less or equal)", model.SortKeyConditionLessEqual},
	{"> (greater than)", model.SortKeyConditionGreater},
	{">= (greater or equal)", model.SortKeyConditionGreaterEq},
}

var filterConditions = []struct {
	label string
	expr  string
}{
	{"= (equals)", "%s = %s"},
	{"<> (not equals)", "%s <> %s"},
	{"< (less than)", "%s < %s"},
	{"<= (less or equal)", "%s <= %s"},
	{"> (greater than)", "%s > %s"},
	{">= (greater or equal)", "%s >= %s"},
	{"contains", "contains(%s, %s)"},
	{"begins_with", "begins_with(%s, %s)"},
}

// NewDynamoDBQueryDialog creates a new query dialog.
func NewDynamoDBQueryDialog() *DynamoDBQueryDialog {
	pkInput := textinput.New()
	pkInput.Placeholder = "partition key value"
	pkInput.CharLimit = 256
	pkInput.Width = 40

	skInput := textinput.New()
	skInput.Placeholder = "sort key value (optional)"
	skInput.CharLimit = 256
	skInput.Width = 40

	limitInput := textinput.New()
	limitInput.Placeholder = "25"
	limitInput.CharLimit = 5
	limitInput.Width = 10

	filterAttrInput := textinput.New()
	filterAttrInput.Placeholder = "attribute name (optional)"
	filterAttrInput.CharLimit = 256
	filterAttrInput.Width = 30

	filterValInput := textinput.New()
	filterValInput.Placeholder = "filter value"
	filterValInput.CharLimit = 256
	filterValInput.Width = 30

	return &DynamoDBQueryDialog{
		pkInput:         pkInput,
		skInput:         skInput,
		limitInput:      limitInput,
		filterAttrInput: filterAttrInput,
		filterValInput:  filterValInput,
	}
}

// SetSize sets the dialog size.
func (d *DynamoDBQueryDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Activate shows the dialog for a query.
func (d *DynamoDBQueryDialog) Activate(tableName, pkName, skName string, isQuery bool) tea.Cmd {
	d.tableName = tableName
	d.pkName = pkName
	d.skName = skName
	d.isQuery = isQuery
	d.active = true
	d.focusIndex = 0
	d.skCondition = 0
	d.filterCondition = 0

	// Reset inputs
	d.pkInput.SetValue("")
	d.skInput.SetValue("")
	d.limitInput.SetValue("")
	d.filterAttrInput.SetValue("")
	d.filterValInput.SetValue("")

	// Focus first input
	d.pkInput.Focus()
	d.skInput.Blur()
	d.limitInput.Blur()
	d.filterAttrInput.Blur()
	d.filterValInput.Blur()

	return textinput.Blink
}

// Deactivate hides the dialog.
func (d *DynamoDBQueryDialog) Deactivate() {
	d.active = false
	d.pkInput.Blur()
	d.skInput.Blur()
	d.limitInput.Blur()
	d.filterAttrInput.Blur()
	d.filterValInput.Blur()
}

// IsActive returns whether the dialog is active.
func (d *DynamoDBQueryDialog) IsActive() bool {
	return d.active
}

// IsQuery returns whether this is a query (vs scan).
func (d *DynamoDBQueryDialog) IsQuery() bool {
	return d.isQuery
}

// QueryDialogResult contains the result of the query dialog.
type QueryDialogResult struct {
	Cancelled   bool
	QueryParams *model.QueryParams
	ScanParams  *model.ScanParams
}

// Update handles input updates.
func (d *DynamoDBQueryDialog) Update(msg tea.Msg) (*QueryDialogResult, tea.Cmd) {
	if !d.active {
		return nil, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Execute query/scan
			result := d.buildResult()
			d.Deactivate()
			return result, nil

		case "esc":
			d.Deactivate()
			return &QueryDialogResult{Cancelled: true}, nil

		case "tab", "down":
			d.nextField()
			return nil, nil

		case "shift+tab", "up":
			d.prevField()
			return nil, nil

		case "left":
			// Change condition selectors
			if d.isOnSKCondition() {
				d.skCondition--
				if d.skCondition < 0 {
					d.skCondition = len(skConditions) - 1
				}
				return nil, nil
			}
			if d.isOnFilterCondition() {
				d.filterCondition--
				if d.filterCondition < 0 {
					d.filterCondition = len(filterConditions) - 1
				}
				return nil, nil
			}

		case "right":
			// Change condition selectors
			if d.isOnSKCondition() {
				d.skCondition++
				if d.skCondition >= len(skConditions) {
					d.skCondition = 0
				}
				return nil, nil
			}
			if d.isOnFilterCondition() {
				d.filterCondition++
				if d.filterCondition >= len(filterConditions) {
					d.filterCondition = 0
				}
				return nil, nil
			}
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	fieldIdx := d.getFieldIndex()
	switch fieldIdx {
	case 0: // PK
		d.pkInput, cmd = d.pkInput.Update(msg)
	case 1: // SK
		d.skInput, cmd = d.skInput.Update(msg)
	case 3: // Limit
		d.limitInput, cmd = d.limitInput.Update(msg)
	case 4: // Filter attribute
		d.filterAttrInput, cmd = d.filterAttrInput.Update(msg)
	case 6: // Filter value
		d.filterValInput, cmd = d.filterValInput.Update(msg)
	}

	return nil, cmd
}

// Field layout for Query with SK:
// 0: PK, 1: SK, 2: SK condition, 3: Limit, 4: Filter attr, 5: Filter condition, 6: Filter value
// Field layout for Query without SK:
// 0: PK, 1: Limit, 2: Filter attr, 3: Filter condition, 4: Filter value

func (d *DynamoDBQueryDialog) maxFields() int {
	if d.skName != "" {
		return 7 // PK, SK, SK condition, Limit, Filter attr, Filter condition, Filter value
	}
	return 5 // PK, Limit, Filter attr, Filter condition, Filter value
}

func (d *DynamoDBQueryDialog) getFieldIndex() int {
	// Maps focusIndex to logical field index
	if d.skName != "" {
		return d.focusIndex
	}
	// No SK - remap indices
	switch d.focusIndex {
	case 0:
		return 0 // PK
	case 1:
		return 3 // Limit
	case 2:
		return 4 // Filter attr
	case 3:
		return 5 // Filter condition
	case 4:
		return 6 // Filter value
	}
	return d.focusIndex
}

func (d *DynamoDBQueryDialog) isOnSKCondition() bool {
	return d.skName != "" && d.focusIndex == 2
}

func (d *DynamoDBQueryDialog) isOnFilterCondition() bool {
	if d.skName != "" {
		return d.focusIndex == 5
	}
	return d.focusIndex == 3
}

func (d *DynamoDBQueryDialog) nextField() {
	d.focusIndex++
	if d.focusIndex >= d.maxFields() {
		d.focusIndex = 0
	}
	d.updateFocus()
}

func (d *DynamoDBQueryDialog) prevField() {
	d.focusIndex--
	if d.focusIndex < 0 {
		d.focusIndex = d.maxFields() - 1
	}
	d.updateFocus()
}

func (d *DynamoDBQueryDialog) updateFocus() {
	d.pkInput.Blur()
	d.skInput.Blur()
	d.limitInput.Blur()
	d.filterAttrInput.Blur()
	d.filterValInput.Blur()

	fieldIdx := d.getFieldIndex()
	switch fieldIdx {
	case 0:
		d.pkInput.Focus()
	case 1:
		d.skInput.Focus()
	case 2:
		// SK condition - no text input focus
	case 3:
		d.limitInput.Focus()
	case 4:
		d.filterAttrInput.Focus()
	case 5:
		// Filter condition - no text input focus
	case 6:
		d.filterValInput.Focus()
	}
}

func (d *DynamoDBQueryDialog) buildResult() *QueryDialogResult {
	limit := int32(25)
	if d.limitInput.Value() != "" {
		if l, err := strconv.Atoi(d.limitInput.Value()); err == nil && l > 0 {
			limit = int32(l)
		}
	}

	// Build filter expression if filter attribute is provided
	filterExpr := ""
	filterAttr := d.filterAttrInput.Value()
	filterVal := d.filterValInput.Value()
	if filterAttr != "" && filterVal != "" {
		filterExpr = fmt.Sprintf(filterConditions[d.filterCondition].expr, "#filterAttr", ":filterVal")
	}

	if d.isQuery {
		params := &model.QueryParams{
			TableName:        d.tableName,
			PartitionKeyName: d.pkName,
			PartitionKeyVal:  d.pkInput.Value(),
			SortKeyName:      d.skName,
			SortKeyVal:       d.skInput.Value(),
			SortKeyCondition: skConditions[d.skCondition].value,
			FilterExpression: filterExpr,
			FilterAttrName:   filterAttr,
			FilterAttrValue:  filterVal,
			Limit:            limit,
			ScanIndexForward: true,
		}
		return &QueryDialogResult{QueryParams: params}
	}

	// Scan
	params := &model.ScanParams{
		TableName:        d.tableName,
		PartitionKeyName: d.pkName,
		SortKeyName:      d.skName,
		FilterExpression: filterExpr,
		FilterAttrName:   filterAttr,
		FilterAttrValue:  filterVal,
		Limit:            limit,
	}
	return &QueryDialogResult{ScanParams: params}
}

// View renders the dialog.
func (d *DynamoDBQueryDialog) View() string {
	if !d.active {
		return ""
	}

	dialogWidth := 60
	if d.width < 70 {
		dialogWidth = d.width - 10
		if dialogWidth < 40 {
			dialogWidth = 40
		}
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderFocus).
		Padding(1, 2).
		Width(dialogWidth)

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Text).
		Width(16)

	focusedLabelStyle := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Width(16)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Italic(true)

	conditionStyle := lipgloss.NewStyle().
		Foreground(theme.Warning)

	var b strings.Builder

	// Title
	if d.isQuery {
		b.WriteString(titleStyle.Render(fmt.Sprintf("Query: %s", d.tableName)))
	} else {
		b.WriteString(titleStyle.Render(fmt.Sprintf("Scan: %s", d.tableName)))
	}
	b.WriteString("\n\n")

	// Partition Key input
	if d.focusIndex == 0 {
		b.WriteString(focusedLabelStyle.Render(d.pkName + ":"))
	} else {
		b.WriteString(labelStyle.Render(d.pkName + ":"))
	}
	b.WriteString(d.pkInput.View())
	b.WriteString("\n\n")

	// Sort Key input (if table has SK)
	if d.skName != "" {
		if d.focusIndex == 1 {
			b.WriteString(focusedLabelStyle.Render(d.skName + ":"))
		} else {
			b.WriteString(labelStyle.Render(d.skName + ":"))
		}
		b.WriteString(d.skInput.View())
		b.WriteString("\n\n")

		// Sort Key condition
		if d.focusIndex == 2 {
			b.WriteString(focusedLabelStyle.Render("Condition:"))
		} else {
			b.WriteString(labelStyle.Render("Condition:"))
		}
		condText := fmt.Sprintf("< %s >", skConditions[d.skCondition].label)
		b.WriteString(conditionStyle.Render(condText))
		b.WriteString("\n\n")
	}

	// Limit input
	fieldIdx := d.getFieldIndex()
	if fieldIdx == 3 {
		b.WriteString(focusedLabelStyle.Render("Limit:"))
	} else {
		b.WriteString(labelStyle.Render("Limit:"))
	}
	b.WriteString(d.limitInput.View())
	b.WriteString("\n\n")

	// Filter section header
	sectionStyle := lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Bold(true)
	b.WriteString(sectionStyle.Render("── Filter (optional) ──"))
	b.WriteString("\n\n")

	// Filter attribute input
	if fieldIdx == 4 {
		b.WriteString(focusedLabelStyle.Render("Filter Attr:"))
	} else {
		b.WriteString(labelStyle.Render("Filter Attr:"))
	}
	b.WriteString(d.filterAttrInput.View())
	b.WriteString("\n\n")

	// Filter condition
	if fieldIdx == 5 {
		b.WriteString(focusedLabelStyle.Render("Filter Cond:"))
	} else {
		b.WriteString(labelStyle.Render("Filter Cond:"))
	}
	condText := fmt.Sprintf("< %s >", filterConditions[d.filterCondition].label)
	b.WriteString(conditionStyle.Render(condText))
	b.WriteString("\n\n")

	// Filter value input
	if fieldIdx == 6 {
		b.WriteString(focusedLabelStyle.Render("Filter Value:"))
	} else {
		b.WriteString(labelStyle.Render("Filter Value:"))
	}
	b.WriteString(d.filterValInput.View())
	b.WriteString("\n\n")

	// Hints
	b.WriteString(hintStyle.Render("Tab: next field | Enter: execute | Esc: cancel"))
	if d.isOnSKCondition() || d.isOnFilterCondition() {
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("Left/Right: change condition"))
	}

	return boxStyle.Render(b.String())
}
