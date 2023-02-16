package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%2d. %s", index+1, i.label)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

type item struct {
	label, value string
}

func (i item) FilterValue() string { return i.value }

type picker struct {
	dynamicPage

	list   list.Model
	target string
}

func newPicker(listLabel, spinnerLabel, key string, preProcessor tea.Cmd) picker {
	p := picker{}

	l := list.New([]list.Item{}, itemDelegate{}, 0, 20)
	l.Title = listLabel
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	p.list = l

	p.preProcessor = preProcessor
	p.key = key
	p.state = "idle"
	if preProcessor != nil {
		p.state = "querying"
	}

	p.spinnerLabel = spinnerLabel

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	p.spinner = s

	return p
}

func (p picker) Init() tea.Cmd {
	return tea.Batch(p.spinner.Tick, p.preProcessor)
}

func (p picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []list.Item:
		p.state = "displaying"
		items := []list.Item(msg)

		offset := len(p.list.Items())

		for i, v := range items {
			p.list.InsertItem(i+offset, v)
		}
		return p, p.spinner.Tick
	case errMsg:
		p.state = "idle"
		p.err = msg
		p.target = msg.target
		return p, nil
	case successMsg:
		p.state = "idle"
		if !msg.unset {
			p.queue.stack.AddSetting(p.key, p.value)
		}

		return p.queue.next()
	case tea.KeyMsg:
		if p.list.FilterState() == list.Filtering {
			break
		}
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return p, tea.Quit
		case "enter":
			if p.state == "displaying" {
				i, ok := p.list.SelectedItem().(item)
				if ok {
					p.value = string(i.value)
				}
				p.queue.stack.AddSetting(p.key, p.value)

				// TODO: see if you can figure out a test for these untested bits

				if p.postProcessor != nil {
					if p.state != "querying" {
						p.state = "querying"
						p.err = nil

						var cmd tea.Cmd
						var cmdSpin tea.Cmd
						cmd = p.postProcessor(p.value, p.queue)
						p.spinner, cmdSpin = p.spinner.Update(msg)

						return p, tea.Batch(cmd, cmdSpin)
					}

					return p, nil
				}

				return p.queue.next()
			}
			if p.err != nil && p.target != "" {
				p.queue.clear(p.target)
				return p.queue.goToModel(p.target)
			}
		}

	default:
		var cmdList tea.Cmd
		var cmdSpin tea.Cmd
		p.list, cmdList = p.list.Update(msg)
		p.spinner, cmdSpin = p.spinner.Update(msg)
		return p, tea.Batch(cmdSpin, cmdList)
	}

	// If this isn't here, then keyPress events do not get responded to by
	// the list ¯\(°_o)/¯
	if p.state == "displaying" {
		var cmd tea.Cmd
		p.list, cmd = p.list.Update(msg)
		return p, cmd
	}

	return p, nil
}

func (p picker) View() string {
	if p.preViewFunc != nil {
		p.preViewFunc(p.queue)
	}
	doc := strings.Builder{}
	doc.WriteString(p.queue.header.render())

	if p.err != nil {
		doc.WriteString(errorAlert{p.err.(errMsg)}.Render())
		return docStyle.Render(doc.String())
	}

	if len(p.content) > 0 {
		inst := strings.Builder{}
		for _, v := range p.content {
			content := v.render()

			inst.WriteString(content)
		}
		doc.WriteString(instructionStyle.Width(width).Render(inst.String()))
		doc.WriteString("\n")
		doc.WriteString("\n")
	}

	if p.state != "waiting" && p.state != "idle" && p.state != "querying" {
		selectedItemStyle.Width(hardWidthLimit)
		doc.WriteString(componentStyle.Render(p.list.View()))
	}

	if p.state == "querying" {
		doc.WriteString(bodyStyle.Render(fmt.Sprintf("%s %s", p.spinnerLabel, p.spinner.View())))
	}

	return docStyle.Render(doc.String())
}