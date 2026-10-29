package mix

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"text/template"
	"text/template/parse"
)

// The most work a template can do, in steps. Each item that seq produces is a step, and so is each call to one of the
// functions here. Each loop iteration and each template invocation costs the number of commands in its body, like
// `.N` or `len .`, at least one. This allows a seq of maxSeqLength items, ranged over with a few commands for each item,
// but stops nested loops, which could otherwise run for minutes without printing anything.
const maxTemplateWork = 100_000

var errTooMuchWork = fmt.Errorf(
	"template does too much work, it can take at most %d steps, like loop iterations, seq items and function calls",
	maxTemplateWork,
)

// workBudget counts the work a single execution of a template does.
type workBudget struct {
	spent int
}

func (b *workBudget) spend(steps int) error {
	b.spent += steps
	if b.spent > maxTemplateWork {
		return errTooMuchWork
	}
	return nil
}

// The function that's called at the start of every loop iteration and template invocation, to charge for its work.
// It's added to the template after parsing, by chargeForWork. A template can call it too, but that only spends more.
const workFuncName = "_work"

// templateFuncs makes the functions for one execution of a template, which spend from the given budget.
func templateFuncs(budget *workBudget) template.FuncMap {
	return template.FuncMap{
		"seq": func(args ...int) ([]SeqItem, error) {
			if err := budget.spend(1); err != nil {
				return nil, err
			}
			return tplFuncSeq(budget, args...)
		},
		"toJSON": func(v any) (string, error) {
			if err := budget.spend(1); err != nil {
				return "", err
			}
			return toJSON(v), nil
		},
		workFuncName: func(steps int) (string, error) {
			return "", budget.spend(max(steps, 1))
		},
	}
}

func toJSON(v any) string {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(v)
	if err != nil {
		log.Printf("Error encoding JSON: %v", err)
		return err.Error()
	}
	return string(bytes.TrimSpace(buffer.Bytes()))
}

// chargeForWork changes the parsed templates, so that the body of every template, and of every range loop, starts by
// spending its cost from the budget. Go templates can't be stopped while executing, and can loop without calling any
// of our functions, like `{{range 100000000}}{{end}}`, or with templates that invoke themselves, so this is the only
// way to bound their work.
func chargeForWork(tpl *template.Template, funcs template.FuncMap) error {
	for _, t := range tpl.Templates() {
		if t.Tree == nil || t.Tree.Root == nil {
			continue
		}
		if err := chargeList(t.Tree.Root, funcs); err != nil {
			return err
		}
	}
	return nil
}

// chargeList adds a charge to the start of the list, for the list's cost, and to the bodies of all range loops in it.
func chargeList(list *parse.ListNode, funcs template.FuncMap) error {
	if err := chargeRangesIn(list, funcs); err != nil {
		return err
	}
	charge, err := template.New("").Funcs(funcs).Parse("{{" + workFuncName + " " + strconv.Itoa(listCost(list)) + "}}")
	if err != nil {
		return err
	}
	list.Nodes = append([]parse.Node{charge.Tree.Root.Nodes[0]}, list.Nodes...)
	return nil
}

// chargeRangesIn adds charges to the bodies of all range loops in the list, including in nested ifs and withs.
func chargeRangesIn(list *parse.ListNode, funcs template.FuncMap) error {
	if list == nil {
		return nil
	}
	for _, node := range list.Nodes {
		var err error
		switch node := node.(type) {
		case *parse.RangeNode:
			if err = chargeList(node.List, funcs); err == nil {
				err = chargeRangesIn(node.ElseList, funcs)
			}
		case *parse.IfNode:
			if err = chargeRangesIn(node.List, funcs); err == nil {
				err = chargeRangesIn(node.ElseList, funcs)
			}
		case *parse.WithNode:
			if err = chargeRangesIn(node.List, funcs); err == nil {
				err = chargeRangesIn(node.ElseList, funcs)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// listCost is the number of commands in the list, including in its ifs and withs, but not in the bodies of its range
// loops, since those charge for themselves on each iteration.
func listCost(list *parse.ListNode) int {
	if list == nil {
		return 0
	}
	cost := 0
	for _, node := range list.Nodes {
		switch node := node.(type) {
		case *parse.ActionNode:
			cost += pipeCost(node.Pipe)
		case *parse.TemplateNode:
			cost += pipeCost(node.Pipe)
		case *parse.RangeNode:
			cost += pipeCost(node.Pipe) + listCost(node.ElseList)
		case *parse.IfNode:
			cost += pipeCost(node.Pipe) + listCost(node.List) + listCost(node.ElseList)
		case *parse.WithNode:
			cost += pipeCost(node.Pipe) + listCost(node.List) + listCost(node.ElseList)
		}
	}
	return cost
}

// pipeCost is the number of commands in a pipeline, including in the parenthesized pipelines in it.
func pipeCost(pipe *parse.PipeNode) int {
	if pipe == nil {
		return 0
	}
	cost := len(pipe.Cmds)
	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			switch arg := arg.(type) {
			case *parse.PipeNode:
				cost += pipeCost(arg)
			case *parse.ChainNode:
				if p, ok := arg.Node.(*parse.PipeNode); ok {
					cost += pipeCost(p)
				}
			}
		}
	}
	return cost
}

type SeqItem struct {
	N       int
	IsFirst bool
	IsLast  bool
}

// Maximum number of items seq can produce, so a template can't exhaust server memory.
const maxSeqLength = 10000

// tplFuncSeq makes a sequence of numbers, spending a step from the budget for each, so that seq calls in nested loops
// stop when they've made too many items in all.
func tplFuncSeq(budget *workBudget, args ...int) ([]SeqItem, error) {
	var start, end, delta int
	switch len(args) {
	case 1:
		start = 0
		end = args[0]
		delta = 1
	case 2:
		start = args[0]
		end = args[1]
		delta = 1
	case 3:
		start = args[0]
		end = args[1]
		delta = args[2]
	default:
		return nil, fmt.Errorf("seq takes one, two or three arguments, got %d", len(args))
	}
	if delta == 0 {
		return nil, errors.New("seq step can't be zero")
	}
	if (start > end && delta > 0) || (start < end && delta < 0) {
		delta = -delta
	}
	var seq []SeqItem
	for i := start; (delta > 0 && i < end) || (delta < 0 && i > end); i += delta {
		if len(seq) == maxSeqLength {
			return nil, fmt.Errorf("seq can't produce more than %d items", maxSeqLength)
		}
		if err := budget.spend(1); err != nil {
			return nil, err
		}
		seq = append(seq, SeqItem{N: i})
	}
	if len(seq) > 0 {
		seq[0].IsFirst = true
		seq[len(seq)-1].IsLast = true
	}
	return seq, nil
}
