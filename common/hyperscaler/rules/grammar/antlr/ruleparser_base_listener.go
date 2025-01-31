// Code generated from RuleParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // RuleParser

import "github.com/antlr4-go/antlr/v4"

// BaseRuleParserListener is a complete listener for a parse tree produced by RuleParserParser.
type BaseRuleParserListener struct{}

var _ RuleParserListener = &BaseRuleParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseRuleParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseRuleParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseRuleParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseRuleParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterRuleEntry is called when production ruleEntry is entered.
func (s *BaseRuleParserListener) EnterRuleEntry(ctx *RuleEntryContext) {}

// ExitRuleEntry is called when production ruleEntry is exited.
func (s *BaseRuleParserListener) ExitRuleEntry(ctx *RuleEntryContext) {}

// EnterEntry is called when production entry is entered.
func (s *BaseRuleParserListener) EnterEntry(ctx *EntryContext) {}

// ExitEntry is called when production entry is exited.
func (s *BaseRuleParserListener) ExitEntry(ctx *EntryContext) {}

// EnterInputAttrInParen is called when production inputAttrInParen is entered.
func (s *BaseRuleParserListener) EnterInputAttrInParen(ctx *InputAttrInParenContext) {}

// ExitInputAttrInParen is called when production inputAttrInParen is exited.
func (s *BaseRuleParserListener) ExitInputAttrInParen(ctx *InputAttrInParenContext) {}

// EnterInputAttrList is called when production inputAttrList is entered.
func (s *BaseRuleParserListener) EnterInputAttrList(ctx *InputAttrListContext) {}

// ExitInputAttrList is called when production inputAttrList is exited.
func (s *BaseRuleParserListener) ExitInputAttrList(ctx *InputAttrListContext) {}

// EnterOutputAttrList is called when production outputAttrList is entered.
func (s *BaseRuleParserListener) EnterOutputAttrList(ctx *OutputAttrListContext) {}

// ExitOutputAttrList is called when production outputAttrList is exited.
func (s *BaseRuleParserListener) ExitOutputAttrList(ctx *OutputAttrListContext) {}

// EnterInputAttrVal is called when production inputAttrVal is entered.
func (s *BaseRuleParserListener) EnterInputAttrVal(ctx *InputAttrValContext) {}

// ExitInputAttrVal is called when production inputAttrVal is exited.
func (s *BaseRuleParserListener) ExitInputAttrVal(ctx *InputAttrValContext) {}

// EnterOutputAttrVal is called when production outputAttrVal is entered.
func (s *BaseRuleParserListener) EnterOutputAttrVal(ctx *OutputAttrValContext) {}

// ExitOutputAttrVal is called when production outputAttrVal is exited.
func (s *BaseRuleParserListener) ExitOutputAttrVal(ctx *OutputAttrValContext) {}

// EnterPrVal is called when production prVal is entered.
func (s *BaseRuleParserListener) EnterPrVal(ctx *PrValContext) {}

// ExitPrVal is called when production prVal is exited.
func (s *BaseRuleParserListener) ExitPrVal(ctx *PrValContext) {}

// EnterHrVal is called when production hrVal is entered.
func (s *BaseRuleParserListener) EnterHrVal(ctx *HrValContext) {}

// ExitHrVal is called when production hrVal is exited.
func (s *BaseRuleParserListener) ExitHrVal(ctx *HrValContext) {}

// EnterPr is called when production pr is entered.
func (s *BaseRuleParserListener) EnterPr(ctx *PrContext) {}

// ExitPr is called when production pr is exited.
func (s *BaseRuleParserListener) ExitPr(ctx *PrContext) {}

// EnterHr is called when production hr is entered.
func (s *BaseRuleParserListener) EnterHr(ctx *HrContext) {}

// ExitHr is called when production hr is exited.
func (s *BaseRuleParserListener) ExitHr(ctx *HrContext) {}

// EnterS is called when production s is entered.
func (s *BaseRuleParserListener) EnterS(ctx *SContext) {}

// ExitS is called when production s is exited.
func (s *BaseRuleParserListener) ExitS(ctx *SContext) {}

// EnterEu is called when production eu is entered.
func (s *BaseRuleParserListener) EnterEu(ctx *EuContext) {}

// ExitEu is called when production eu is exited.
func (s *BaseRuleParserListener) ExitEu(ctx *EuContext) {}

// EnterVal is called when production val is entered.
func (s *BaseRuleParserListener) EnterVal(ctx *ValContext) {}

// ExitVal is called when production val is exited.
func (s *BaseRuleParserListener) ExitVal(ctx *ValContext) {}
