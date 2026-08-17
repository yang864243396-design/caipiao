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

type ImportIssue struct {
	Row     int
	Rule    ExcelRule
	Matches []CatalogCandidate
}

type ImportReport struct {
	Drafts     []DraftRule
	Unresolved []ImportIssue
	Ambiguous  []ImportIssue
}

// ImportFilePartial keeps every exact, unambiguous catalogue match and
// reports legacy rows that cannot be mapped safely. It never writes data.
func ImportFilePartial(path string, candidates []CatalogCandidate) (ImportReport, error) {
	book, err := excelize.OpenFile(path)
	if err != nil {
		return ImportReport{}, fmt.Errorf("open workbook: %w", err)
	}
	defer book.Close()

	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return ImportReport{}, fmt.Errorf("workbook has no sheets")
	}
	rows, err := book.GetRows(sheets[0])
	if err != nil {
		return ImportReport{}, fmt.Errorf("read workbook rows: %w", err)
	}
	if len(rows) < 2 {
		return ImportReport{}, fmt.Errorf("workbook has no data rows")
	}
	columns := columnsForHeader(rows[0])
	if columns.ruleID < 0 || columns.fullName < 0 {
		return ImportReport{}, fmt.Errorf("workbook header must contain ID and 规则全名")
	}

	report := ImportReport{Drafts: make([]DraftRule, 0, len(rows)-1)}
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
		issue := ImportIssue{Row: rowNo + 2, Rule: rule, Matches: matches}
		if len(matches) == 0 {
			report.Unresolved = append(report.Unresolved, issue)
			continue
		}
		if !sameCatalogPlay(matches) {
			report.Ambiguous = append(report.Ambiguous, issue)
			continue
		}
		for _, match := range matches {
			report.Drafts = append(report.Drafts, DraftRule{ExcelRule: rule, CatalogCandidate: match})
		}
	}
	return report, nil
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
			if !sameCatalogPlay(matches) {
				return nil, fmt.Errorf("%w at row %d: id=%q name=%q matched=%d", ErrAmbiguousRuleMatch, rowNo+2, rule.RuleID, rule.FullName, len(matches))
			}
		}
		for _, match := range matches {
			drafts = append(drafts, DraftRule{ExcelRule: rule, CatalogCandidate: match})
		}
	}
	return drafts, nil
}

func sameCatalogPlay(candidates []CatalogCandidate) bool {
	if len(candidates) <= 1 {
		return true
	}
	first := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.TypeID != first.TypeID || candidate.SubID != first.SubID {
			return false
		}
	}
	return true
}

func matchCandidates(rule ExcelRule, candidates []CatalogCandidate) []CatalogCandidate {
	exactMatches := make([]CatalogCandidate, 0, 1)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.FullName) == rule.FullName {
			exactMatches = append(exactMatches, candidate)
		}
	}
	if len(exactMatches) > 0 {
		return exactMatches
	}
	// Workbook rule ids belong to an older third-party catalogue and can be
	// reused with a different meaning in the current catalogue. Match only the
	// human-readable full name; unresolved rows stay out of the import.
	return nil
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
