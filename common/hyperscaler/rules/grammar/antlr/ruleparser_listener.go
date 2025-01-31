// Code generated from RuleParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // RuleParser

import "github.com/antlr4-go/antlr/v4"

// RuleParserListener is a complete listener for a parse tree produced by RuleParserParser.
type RuleParserListener interface {
	antlr.ParseTreeListener

	// EnterRuleEntry is called when entering the ruleEntry production.
	EnterRuleEntry(c *RuleEntryContext)

	// EnterEntry is called when entering the entry production.
	EnterEntry(c *EntryContext)

	// EnterInputAttrInParen is called when entering the inputAttrInParen production.
	EnterInputAttrInParen(c *InputAttrInParenContext)

	// EnterInputAttrList is called when entering the inputAttrList production.
	EnterInputAttrList(c *InputAttrListContext)

	// EnterOutputAttrList is called when entering the outputAttrList production.
	EnterOutputAttrList(c *OutputAttrListContext)

	// EnterInputAttrVal is called when entering the inputAttrVal production.
	EnterInputAttrVal(c *InputAttrValContext)

	// EnterOutputAttrVal is called when entering the outputAttrVal production.
	EnterOutputAttrVal(c *OutputAttrValContext)

	// EnterPrVal is called when entering the prVal production.
	EnterPrVal(c *PrValContext)

	// EnterHrVal is called when entering the hrVal production.
	EnterHrVal(c *HrValContext)

	// EnterPr is called when entering the pr production.
	EnterPr(c *PrContext)

	// EnterHr is called when entering the hr production.
	EnterHr(c *HrContext)

	// EnterS is called when entering the s production.
	EnterS(c *SContext)

	// EnterEu is called when entering the eu production.
	EnterEu(c *EuContext)

	// EnterVal is called when entering the val production.
	EnterVal(c *ValContext)

	// ExitRuleEntry is called when exiting the ruleEntry production.
	ExitRuleEntry(c *RuleEntryContext)

	// ExitEntry is called when exiting the entry production.
	ExitEntry(c *EntryContext)

	// ExitInputAttrInParen is called when exiting the inputAttrInParen production.
	ExitInputAttrInParen(c *InputAttrInParenContext)

	// ExitInputAttrList is called when exiting the inputAttrList production.
	ExitInputAttrList(c *InputAttrListContext)

	// ExitOutputAttrList is called when exiting the outputAttrList production.
	ExitOutputAttrList(c *OutputAttrListContext)

	// ExitInputAttrVal is called when exiting the inputAttrVal production.
	ExitInputAttrVal(c *InputAttrValContext)

	// ExitOutputAttrVal is called when exiting the outputAttrVal production.
	ExitOutputAttrVal(c *OutputAttrValContext)

	// ExitPrVal is called when exiting the prVal production.
	ExitPrVal(c *PrValContext)

	// ExitHrVal is called when exiting the hrVal production.
	ExitHrVal(c *HrValContext)

	// ExitPr is called when exiting the pr production.
	ExitPr(c *PrContext)

	// ExitHr is called when exiting the hr production.
	ExitHr(c *HrContext)

	// ExitS is called when exiting the s production.
	ExitS(c *SContext)

	// ExitEu is called when exiting the eu production.
	ExitEu(c *EuContext)

	// ExitVal is called when exiting the val production.
	ExitVal(c *ValContext)
}
