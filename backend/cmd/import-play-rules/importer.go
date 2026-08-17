package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

var (
	ErrAmbiguousRuleMatch  = errors.New("ambiguous excel rule match")
	ErrUnresolvedRuleMatch = errors.New("unresolved excel rule match")
)

// CatalogCandidate is the stable natural key currently represented by a
// third-party rule id and its complete name in the local play catalogue.
type CatalogCandidate struct {
	RuleID       string
	FullName     string
	TemplateCode string
	TypeID       string
	SubID        string
}

type ExcelRule struct {
	RuleID       string
	FullName     string
	Description  string
	Example      string
	NumberRange  string
	TotalUnits   string
	WinningUnits string
	Odds         string
	SoloUnits    string
}

type DraftRule struct {
	ExcelRule
	CatalogCandidate
}

// ImportFile parses the approved rule workbook and maps every non-empty rule
// to exactly one existing catalogue play. It intentionally does not perform
// any database write; callers decide whether the resulting drafts are stored.
func ImportFile(path string, candidates []CatalogCandidate) ([]DraftRule, error) {
	book, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open workbook: %w", err)
	}
	defer book.Close()

	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("workbook has no sheets")
	}
	rows, err := book.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read workbook rows: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("workbook has no data rows")
	}
	columns := columnsForHeader(rows[0])
	if columns.ruleID < 0 || columns.fullName < 0 {
		return nil, fmt.Errorf("workbook header must contain ID and 规则全名")
	}

	drafts := make([]DraftRule, 0, len(rows)-1)
	for rowNo, row := range rows[1:] {
		rule := ExcelRule{
			RuleID:       valueAt(row, columns.ruleID),
			FullName:     valueAt(row, columns.fullName),
			Description:  valueAt(row, columns.description),
			Example:      valueAt(row, columns.example),
			NumberRange:  valueAt(row, columns.numberRange),
			TotalUnits:   valueAt(row, columns.totalUnits),
			WinningUnits: valueAt(row, columns.winningUnits),
			Odds:         valueAt(row, columns.odds),
			SoloUnits:    valueAt(row, columns.soloUnits),
		}
		if rule.RuleID == "" && rule.FullName == "" {
			continue
		}
		matches := matchCandidates(rule, candidates)
		if len(matches) == 0 {
			return nil, fmt.Errorf("%w at row %d: id=%q name=%q", ErrUnresolvedRuleMatch, rowNo+2, rule.RuleID, rule.FullName)
		}
		if len(matches) != 1 {
			return nil, fmt.Errorf("%w at row %d: id=%q name=%q matched=%d", ErrAmbiguousRuleMatch, rowNo+2, rule.RuleID, rule.FullName, len(matches))
		}
		drafts = append(drafts, DraftRule{ExcelRule: rule, CatalogCandidate: matches[0]})
	}
	return drafts, nil
}

func matchCandidates(rule ExcelRule, candidates []CatalogCandidate) []CatalogCandidate {
	matched := make([]CatalogCandidate, 0, 1)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.RuleID) == rule.RuleID && strings.TrimSpace(candidate.FullName) == rule.FullName {
			matched = append(matched, candidate)
		}
	}
	return matched
}

type workbookColumns struct {
	ruleID, fullName, description, example, numberRange, totalUnits, winningUnits, odds, soloUnits int
}

func columnsForHeader(header []string) workbookColumns {
	columns := workbookColumns{ruleID: -1, fullName: -1, description: -1, example: -1, numberRange: -1, totalUnits: -1, winningUnits: -1, odds: -1, soloUnits: -1}
	for index, value := range header {
		switch strings.TrimSpace(value) {
		case "ID":
			columns.ruleID = index
		case "规则全名":
			columns.fullName = index
		case "描述":
			columns.description = index
		case "示例":
			columns.example = index
		case "号码区间":
			columns.numberRange = index
		case "全注数":
			columns.totalUnits = index
		case "可中注数":
			columns.winningUnits = index
		case "赔率":
			columns.odds = index
		case "单挑注数":
			columns.soloUnits = index
		}
	}
	return columns
}

func valueAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}
