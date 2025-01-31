// Code generated from RuleParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // RuleParser

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type RuleParserParser struct {
	*antlr.BaseParser
}

var RuleParserParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func ruleparserParserInit() {
	staticData := &RuleParserParserStaticData
	staticData.LiteralNames = []string{
		"", "'='", "','", "'('", "')'", "'->'", "'*'", "", "", "'PR'", "'HR'",
		"'S'", "'EU'",
	}
	staticData.SymbolicNames = []string{
		"", "EQ", "COMMA", "LPAREN", "RPAREN", "ARROW", "ASTERIX", "WS", "PLAN",
		"PR", "HR", "S", "EU", "VAL",
	}
	staticData.RuleNames = []string{
		"ruleEntry", "entry", "inputAttrInParen", "inputAttrList", "outputAttrList",
		"inputAttrVal", "outputAttrVal", "prVal", "hrVal", "pr", "hr", "s",
		"eu", "val",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 13, 93, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 1, 0, 1, 0, 1, 0, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 3, 1, 43, 8, 1,
		1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 3, 2, 51, 8, 2, 1, 3, 1, 3, 1, 3, 1,
		3, 1, 3, 3, 3, 58, 8, 3, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 3, 4, 65, 8, 4,
		1, 5, 1, 5, 3, 5, 69, 8, 5, 1, 6, 1, 6, 3, 6, 73, 8, 6, 1, 7, 1, 7, 1,
		7, 1, 7, 1, 8, 1, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11,
		1, 12, 1, 12, 1, 13, 1, 13, 1, 13, 0, 0, 14, 0, 2, 4, 6, 8, 10, 12, 14,
		16, 18, 20, 22, 24, 26, 0, 1, 2, 0, 6, 6, 13, 13, 86, 0, 28, 1, 0, 0, 0,
		2, 42, 1, 0, 0, 0, 4, 50, 1, 0, 0, 0, 6, 57, 1, 0, 0, 0, 8, 64, 1, 0, 0,
		0, 10, 68, 1, 0, 0, 0, 12, 72, 1, 0, 0, 0, 14, 74, 1, 0, 0, 0, 16, 78,
		1, 0, 0, 0, 18, 82, 1, 0, 0, 0, 20, 84, 1, 0, 0, 0, 22, 86, 1, 0, 0, 0,
		24, 88, 1, 0, 0, 0, 26, 90, 1, 0, 0, 0, 28, 29, 3, 2, 1, 0, 29, 30, 5,
		0, 0, 1, 30, 1, 1, 0, 0, 0, 31, 43, 5, 8, 0, 0, 32, 33, 5, 8, 0, 0, 33,
		34, 5, 5, 0, 0, 34, 43, 3, 8, 4, 0, 35, 36, 5, 8, 0, 0, 36, 43, 3, 4, 2,
		0, 37, 38, 5, 8, 0, 0, 38, 39, 3, 4, 2, 0, 39, 40, 5, 5, 0, 0, 40, 41,
		3, 8, 4, 0, 41, 43, 1, 0, 0, 0, 42, 31, 1, 0, 0, 0, 42, 32, 1, 0, 0, 0,
		42, 35, 1, 0, 0, 0, 42, 37, 1, 0, 0, 0, 43, 3, 1, 0, 0, 0, 44, 45, 5, 3,
		0, 0, 45, 51, 5, 4, 0, 0, 46, 47, 5, 3, 0, 0, 47, 48, 3, 6, 3, 0, 48, 49,
		5, 4, 0, 0, 49, 51, 1, 0, 0, 0, 50, 44, 1, 0, 0, 0, 50, 46, 1, 0, 0, 0,
		51, 5, 1, 0, 0, 0, 52, 58, 3, 10, 5, 0, 53, 54, 3, 10, 5, 0, 54, 55, 5,
		2, 0, 0, 55, 56, 3, 6, 3, 0, 56, 58, 1, 0, 0, 0, 57, 52, 1, 0, 0, 0, 57,
		53, 1, 0, 0, 0, 58, 7, 1, 0, 0, 0, 59, 65, 3, 12, 6, 0, 60, 61, 3, 12,
		6, 0, 61, 62, 5, 2, 0, 0, 62, 63, 3, 8, 4, 0, 63, 65, 1, 0, 0, 0, 64, 59,
		1, 0, 0, 0, 64, 60, 1, 0, 0, 0, 65, 9, 1, 0, 0, 0, 66, 69, 3, 14, 7, 0,
		67, 69, 3, 16, 8, 0, 68, 66, 1, 0, 0, 0, 68, 67, 1, 0, 0, 0, 69, 11, 1,
		0, 0, 0, 70, 73, 3, 22, 11, 0, 71, 73, 3, 24, 12, 0, 72, 70, 1, 0, 0, 0,
		72, 71, 1, 0, 0, 0, 73, 13, 1, 0, 0, 0, 74, 75, 3, 18, 9, 0, 75, 76, 5,
		1, 0, 0, 76, 77, 3, 26, 13, 0, 77, 15, 1, 0, 0, 0, 78, 79, 3, 20, 10, 0,
		79, 80, 5, 1, 0, 0, 80, 81, 3, 26, 13, 0, 81, 17, 1, 0, 0, 0, 82, 83, 5,
		9, 0, 0, 83, 19, 1, 0, 0, 0, 84, 85, 5, 10, 0, 0, 85, 21, 1, 0, 0, 0, 86,
		87, 5, 11, 0, 0, 87, 23, 1, 0, 0, 0, 88, 89, 5, 12, 0, 0, 89, 25, 1, 0,
		0, 0, 90, 91, 7, 0, 0, 0, 91, 27, 1, 0, 0, 0, 6, 42, 50, 57, 64, 68, 72,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// RuleParserParserInit initializes any static state used to implement RuleParserParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewRuleParserParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func RuleParserParserInit() {
	staticData := &RuleParserParserStaticData
	staticData.once.Do(ruleparserParserInit)
}

// NewRuleParserParser produces a new parser instance for the optional input antlr.TokenStream.
func NewRuleParserParser(input antlr.TokenStream) *RuleParserParser {
	RuleParserParserInit()
	this := new(RuleParserParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &RuleParserParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "RuleParser.g4"

	return this
}

// RuleParserParser tokens.
const (
	RuleParserParserEOF     = antlr.TokenEOF
	RuleParserParserEQ      = 1
	RuleParserParserCOMMA   = 2
	RuleParserParserLPAREN  = 3
	RuleParserParserRPAREN  = 4
	RuleParserParserARROW   = 5
	RuleParserParserASTERIX = 6
	RuleParserParserWS      = 7
	RuleParserParserPLAN    = 8
	RuleParserParserPR      = 9
	RuleParserParserHR      = 10
	RuleParserParserS       = 11
	RuleParserParserEU      = 12
	RuleParserParserVAL     = 13
)

// RuleParserParser rules.
const (
	RuleParserParserRULE_ruleEntry        = 0
	RuleParserParserRULE_entry            = 1
	RuleParserParserRULE_inputAttrInParen = 2
	RuleParserParserRULE_inputAttrList    = 3
	RuleParserParserRULE_outputAttrList   = 4
	RuleParserParserRULE_inputAttrVal     = 5
	RuleParserParserRULE_outputAttrVal    = 6
	RuleParserParserRULE_prVal            = 7
	RuleParserParserRULE_hrVal            = 8
	RuleParserParserRULE_pr               = 9
	RuleParserParserRULE_hr               = 10
	RuleParserParserRULE_s                = 11
	RuleParserParserRULE_eu               = 12
	RuleParserParserRULE_val              = 13
)

// IRuleEntryContext is an interface to support dynamic dispatch.
type IRuleEntryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Entry() IEntryContext
	EOF() antlr.TerminalNode

	// IsRuleEntryContext differentiates from other interfaces.
	IsRuleEntryContext()
}

type RuleEntryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyRuleEntryContext() *RuleEntryContext {
	var p = new(RuleEntryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_ruleEntry
	return p
}

func InitEmptyRuleEntryContext(p *RuleEntryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_ruleEntry
}

func (*RuleEntryContext) IsRuleEntryContext() {}

func NewRuleEntryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *RuleEntryContext {
	var p = new(RuleEntryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_ruleEntry

	return p
}

func (s *RuleEntryContext) GetParser() antlr.Parser { return s.parser }

func (s *RuleEntryContext) Entry() IEntryContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEntryContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IEntryContext)
}

func (s *RuleEntryContext) EOF() antlr.TerminalNode {
	return s.GetToken(RuleParserParserEOF, 0)
}

func (s *RuleEntryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *RuleEntryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *RuleEntryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterRuleEntry(s)
	}
}

func (s *RuleEntryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitRuleEntry(s)
	}
}

func (p *RuleParserParser) RuleEntry() (localctx IRuleEntryContext) {
	localctx = NewRuleEntryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, RuleParserParserRULE_ruleEntry)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(28)
		p.Entry()
	}
	{
		p.SetState(29)
		p.Match(RuleParserParserEOF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IEntryContext is an interface to support dynamic dispatch.
type IEntryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PLAN() antlr.TerminalNode
	ARROW() antlr.TerminalNode
	OutputAttrList() IOutputAttrListContext
	InputAttrInParen() IInputAttrInParenContext

	// IsEntryContext differentiates from other interfaces.
	IsEntryContext()
}

type EntryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEntryContext() *EntryContext {
	var p = new(EntryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_entry
	return p
}

func InitEmptyEntryContext(p *EntryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_entry
}

func (*EntryContext) IsEntryContext() {}

func NewEntryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EntryContext {
	var p = new(EntryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_entry

	return p
}

func (s *EntryContext) GetParser() antlr.Parser { return s.parser }

func (s *EntryContext) PLAN() antlr.TerminalNode {
	return s.GetToken(RuleParserParserPLAN, 0)
}

func (s *EntryContext) ARROW() antlr.TerminalNode {
	return s.GetToken(RuleParserParserARROW, 0)
}

func (s *EntryContext) OutputAttrList() IOutputAttrListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOutputAttrListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOutputAttrListContext)
}

func (s *EntryContext) InputAttrInParen() IInputAttrInParenContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInputAttrInParenContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInputAttrInParenContext)
}

func (s *EntryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EntryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EntryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterEntry(s)
	}
}

func (s *EntryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitEntry(s)
	}
}

func (p *RuleParserParser) Entry() (localctx IEntryContext) {
	localctx = NewEntryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, RuleParserParserRULE_entry)
	p.SetState(42)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 0, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(31)
			p.Match(RuleParserParserPLAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(32)
			p.Match(RuleParserParserPLAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(33)
			p.Match(RuleParserParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(34)
			p.OutputAttrList()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(35)
			p.Match(RuleParserParserPLAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(36)
			p.InputAttrInParen()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(37)
			p.Match(RuleParserParserPLAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(38)
			p.InputAttrInParen()
		}
		{
			p.SetState(39)
			p.Match(RuleParserParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(40)
			p.OutputAttrList()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IInputAttrInParenContext is an interface to support dynamic dispatch.
type IInputAttrInParenContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	InputAttrList() IInputAttrListContext

	// IsInputAttrInParenContext differentiates from other interfaces.
	IsInputAttrInParenContext()
}

type InputAttrInParenContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInputAttrInParenContext() *InputAttrInParenContext {
	var p = new(InputAttrInParenContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_inputAttrInParen
	return p
}

func InitEmptyInputAttrInParenContext(p *InputAttrInParenContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_inputAttrInParen
}

func (*InputAttrInParenContext) IsInputAttrInParenContext() {}

func NewInputAttrInParenContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InputAttrInParenContext {
	var p = new(InputAttrInParenContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_inputAttrInParen

	return p
}

func (s *InputAttrInParenContext) GetParser() antlr.Parser { return s.parser }

func (s *InputAttrInParenContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(RuleParserParserLPAREN, 0)
}

func (s *InputAttrInParenContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(RuleParserParserRPAREN, 0)
}

func (s *InputAttrInParenContext) InputAttrList() IInputAttrListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInputAttrListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInputAttrListContext)
}

func (s *InputAttrInParenContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InputAttrInParenContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InputAttrInParenContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterInputAttrInParen(s)
	}
}

func (s *InputAttrInParenContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitInputAttrInParen(s)
	}
}

func (p *RuleParserParser) InputAttrInParen() (localctx IInputAttrInParenContext) {
	localctx = NewInputAttrInParenContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, RuleParserParserRULE_inputAttrInParen)
	p.SetState(50)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 1, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(44)
			p.Match(RuleParserParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(45)
			p.Match(RuleParserParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(46)
			p.Match(RuleParserParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(47)
			p.InputAttrList()
		}
		{
			p.SetState(48)
			p.Match(RuleParserParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IInputAttrListContext is an interface to support dynamic dispatch.
type IInputAttrListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	InputAttrVal() IInputAttrValContext
	COMMA() antlr.TerminalNode
	InputAttrList() IInputAttrListContext

	// IsInputAttrListContext differentiates from other interfaces.
	IsInputAttrListContext()
}

type InputAttrListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInputAttrListContext() *InputAttrListContext {
	var p = new(InputAttrListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_inputAttrList
	return p
}

func InitEmptyInputAttrListContext(p *InputAttrListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_inputAttrList
}

func (*InputAttrListContext) IsInputAttrListContext() {}

func NewInputAttrListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InputAttrListContext {
	var p = new(InputAttrListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_inputAttrList

	return p
}

func (s *InputAttrListContext) GetParser() antlr.Parser { return s.parser }

func (s *InputAttrListContext) InputAttrVal() IInputAttrValContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInputAttrValContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInputAttrValContext)
}

func (s *InputAttrListContext) COMMA() antlr.TerminalNode {
	return s.GetToken(RuleParserParserCOMMA, 0)
}

func (s *InputAttrListContext) InputAttrList() IInputAttrListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInputAttrListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInputAttrListContext)
}

func (s *InputAttrListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InputAttrListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InputAttrListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterInputAttrList(s)
	}
}

func (s *InputAttrListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitInputAttrList(s)
	}
}

func (p *RuleParserParser) InputAttrList() (localctx IInputAttrListContext) {
	localctx = NewInputAttrListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, RuleParserParserRULE_inputAttrList)
	p.SetState(57)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 2, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(52)
			p.InputAttrVal()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(53)
			p.InputAttrVal()
		}
		{
			p.SetState(54)
			p.Match(RuleParserParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(55)
			p.InputAttrList()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IOutputAttrListContext is an interface to support dynamic dispatch.
type IOutputAttrListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	OutputAttrVal() IOutputAttrValContext
	COMMA() antlr.TerminalNode
	OutputAttrList() IOutputAttrListContext

	// IsOutputAttrListContext differentiates from other interfaces.
	IsOutputAttrListContext()
}

type OutputAttrListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOutputAttrListContext() *OutputAttrListContext {
	var p = new(OutputAttrListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_outputAttrList
	return p
}

func InitEmptyOutputAttrListContext(p *OutputAttrListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_outputAttrList
}

func (*OutputAttrListContext) IsOutputAttrListContext() {}

func NewOutputAttrListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OutputAttrListContext {
	var p = new(OutputAttrListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_outputAttrList

	return p
}

func (s *OutputAttrListContext) GetParser() antlr.Parser { return s.parser }

func (s *OutputAttrListContext) OutputAttrVal() IOutputAttrValContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOutputAttrValContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOutputAttrValContext)
}

func (s *OutputAttrListContext) COMMA() antlr.TerminalNode {
	return s.GetToken(RuleParserParserCOMMA, 0)
}

func (s *OutputAttrListContext) OutputAttrList() IOutputAttrListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOutputAttrListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOutputAttrListContext)
}

func (s *OutputAttrListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OutputAttrListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OutputAttrListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterOutputAttrList(s)
	}
}

func (s *OutputAttrListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitOutputAttrList(s)
	}
}

func (p *RuleParserParser) OutputAttrList() (localctx IOutputAttrListContext) {
	localctx = NewOutputAttrListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, RuleParserParserRULE_outputAttrList)
	p.SetState(64)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 3, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(59)
			p.OutputAttrVal()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(60)
			p.OutputAttrVal()
		}
		{
			p.SetState(61)
			p.Match(RuleParserParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(62)
			p.OutputAttrList()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IInputAttrValContext is an interface to support dynamic dispatch.
type IInputAttrValContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PrVal() IPrValContext
	HrVal() IHrValContext

	// IsInputAttrValContext differentiates from other interfaces.
	IsInputAttrValContext()
}

type InputAttrValContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInputAttrValContext() *InputAttrValContext {
	var p = new(InputAttrValContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_inputAttrVal
	return p
}

func InitEmptyInputAttrValContext(p *InputAttrValContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_inputAttrVal
}

func (*InputAttrValContext) IsInputAttrValContext() {}

func NewInputAttrValContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InputAttrValContext {
	var p = new(InputAttrValContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_inputAttrVal

	return p
}

func (s *InputAttrValContext) GetParser() antlr.Parser { return s.parser }

func (s *InputAttrValContext) PrVal() IPrValContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrValContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrValContext)
}

func (s *InputAttrValContext) HrVal() IHrValContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IHrValContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IHrValContext)
}

func (s *InputAttrValContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InputAttrValContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InputAttrValContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterInputAttrVal(s)
	}
}

func (s *InputAttrValContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitInputAttrVal(s)
	}
}

func (p *RuleParserParser) InputAttrVal() (localctx IInputAttrValContext) {
	localctx = NewInputAttrValContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, RuleParserParserRULE_inputAttrVal)
	p.SetState(68)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case RuleParserParserPR:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(66)
			p.PrVal()
		}

	case RuleParserParserHR:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(67)
			p.HrVal()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IOutputAttrValContext is an interface to support dynamic dispatch.
type IOutputAttrValContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	S() ISContext
	Eu() IEuContext

	// IsOutputAttrValContext differentiates from other interfaces.
	IsOutputAttrValContext()
}

type OutputAttrValContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOutputAttrValContext() *OutputAttrValContext {
	var p = new(OutputAttrValContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_outputAttrVal
	return p
}

func InitEmptyOutputAttrValContext(p *OutputAttrValContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_outputAttrVal
}

func (*OutputAttrValContext) IsOutputAttrValContext() {}

func NewOutputAttrValContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OutputAttrValContext {
	var p = new(OutputAttrValContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_outputAttrVal

	return p
}

func (s *OutputAttrValContext) GetParser() antlr.Parser { return s.parser }

func (s *OutputAttrValContext) S() ISContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISContext)
}

func (s *OutputAttrValContext) Eu() IEuContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEuContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IEuContext)
}

func (s *OutputAttrValContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OutputAttrValContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OutputAttrValContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterOutputAttrVal(s)
	}
}

func (s *OutputAttrValContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitOutputAttrVal(s)
	}
}

func (p *RuleParserParser) OutputAttrVal() (localctx IOutputAttrValContext) {
	localctx = NewOutputAttrValContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, RuleParserParserRULE_outputAttrVal)
	p.SetState(72)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case RuleParserParserS:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(70)
			p.S()
		}

	case RuleParserParserEU:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(71)
			p.Eu()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPrValContext is an interface to support dynamic dispatch.
type IPrValContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Pr() IPrContext
	EQ() antlr.TerminalNode
	Val() IValContext

	// IsPrValContext differentiates from other interfaces.
	IsPrValContext()
}

type PrValContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPrValContext() *PrValContext {
	var p = new(PrValContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_prVal
	return p
}

func InitEmptyPrValContext(p *PrValContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_prVal
}

func (*PrValContext) IsPrValContext() {}

func NewPrValContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PrValContext {
	var p = new(PrValContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_prVal

	return p
}

func (s *PrValContext) GetParser() antlr.Parser { return s.parser }

func (s *PrValContext) Pr() IPrContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrContext)
}

func (s *PrValContext) EQ() antlr.TerminalNode {
	return s.GetToken(RuleParserParserEQ, 0)
}

func (s *PrValContext) Val() IValContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IValContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IValContext)
}

func (s *PrValContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PrValContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PrValContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterPrVal(s)
	}
}

func (s *PrValContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitPrVal(s)
	}
}

func (p *RuleParserParser) PrVal() (localctx IPrValContext) {
	localctx = NewPrValContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, RuleParserParserRULE_prVal)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(74)
		p.Pr()
	}
	{
		p.SetState(75)
		p.Match(RuleParserParserEQ)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(76)
		p.Val()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IHrValContext is an interface to support dynamic dispatch.
type IHrValContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Hr() IHrContext
	EQ() antlr.TerminalNode
	Val() IValContext

	// IsHrValContext differentiates from other interfaces.
	IsHrValContext()
}

type HrValContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyHrValContext() *HrValContext {
	var p = new(HrValContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_hrVal
	return p
}

func InitEmptyHrValContext(p *HrValContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_hrVal
}

func (*HrValContext) IsHrValContext() {}

func NewHrValContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *HrValContext {
	var p = new(HrValContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_hrVal

	return p
}

func (s *HrValContext) GetParser() antlr.Parser { return s.parser }

func (s *HrValContext) Hr() IHrContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IHrContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IHrContext)
}

func (s *HrValContext) EQ() antlr.TerminalNode {
	return s.GetToken(RuleParserParserEQ, 0)
}

func (s *HrValContext) Val() IValContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IValContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IValContext)
}

func (s *HrValContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *HrValContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *HrValContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterHrVal(s)
	}
}

func (s *HrValContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitHrVal(s)
	}
}

func (p *RuleParserParser) HrVal() (localctx IHrValContext) {
	localctx = NewHrValContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, RuleParserParserRULE_hrVal)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(78)
		p.Hr()
	}
	{
		p.SetState(79)
		p.Match(RuleParserParserEQ)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(80)
		p.Val()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPrContext is an interface to support dynamic dispatch.
type IPrContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PR() antlr.TerminalNode

	// IsPrContext differentiates from other interfaces.
	IsPrContext()
}

type PrContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPrContext() *PrContext {
	var p = new(PrContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_pr
	return p
}

func InitEmptyPrContext(p *PrContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_pr
}

func (*PrContext) IsPrContext() {}

func NewPrContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PrContext {
	var p = new(PrContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_pr

	return p
}

func (s *PrContext) GetParser() antlr.Parser { return s.parser }

func (s *PrContext) PR() antlr.TerminalNode {
	return s.GetToken(RuleParserParserPR, 0)
}

func (s *PrContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PrContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PrContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterPr(s)
	}
}

func (s *PrContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitPr(s)
	}
}

func (p *RuleParserParser) Pr() (localctx IPrContext) {
	localctx = NewPrContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, RuleParserParserRULE_pr)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(82)
		p.Match(RuleParserParserPR)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IHrContext is an interface to support dynamic dispatch.
type IHrContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	HR() antlr.TerminalNode

	// IsHrContext differentiates from other interfaces.
	IsHrContext()
}

type HrContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyHrContext() *HrContext {
	var p = new(HrContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_hr
	return p
}

func InitEmptyHrContext(p *HrContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_hr
}

func (*HrContext) IsHrContext() {}

func NewHrContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *HrContext {
	var p = new(HrContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_hr

	return p
}

func (s *HrContext) GetParser() antlr.Parser { return s.parser }

func (s *HrContext) HR() antlr.TerminalNode {
	return s.GetToken(RuleParserParserHR, 0)
}

func (s *HrContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *HrContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *HrContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterHr(s)
	}
}

func (s *HrContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitHr(s)
	}
}

func (p *RuleParserParser) Hr() (localctx IHrContext) {
	localctx = NewHrContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, RuleParserParserRULE_hr)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(84)
		p.Match(RuleParserParserHR)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISContext is an interface to support dynamic dispatch.
type ISContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	S() antlr.TerminalNode

	// IsSContext differentiates from other interfaces.
	IsSContext()
}

type SContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySContext() *SContext {
	var p = new(SContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_s
	return p
}

func InitEmptySContext(p *SContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_s
}

func (*SContext) IsSContext() {}

func NewSContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SContext {
	var p = new(SContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_s

	return p
}

func (s *SContext) GetParser() antlr.Parser { return s.parser }

func (s *SContext) S() antlr.TerminalNode {
	return s.GetToken(RuleParserParserS, 0)
}

func (s *SContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterS(s)
	}
}

func (s *SContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitS(s)
	}
}

func (p *RuleParserParser) S() (localctx ISContext) {
	localctx = NewSContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, RuleParserParserRULE_s)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(86)
		p.Match(RuleParserParserS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IEuContext is an interface to support dynamic dispatch.
type IEuContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EU() antlr.TerminalNode

	// IsEuContext differentiates from other interfaces.
	IsEuContext()
}

type EuContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEuContext() *EuContext {
	var p = new(EuContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_eu
	return p
}

func InitEmptyEuContext(p *EuContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_eu
}

func (*EuContext) IsEuContext() {}

func NewEuContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EuContext {
	var p = new(EuContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_eu

	return p
}

func (s *EuContext) GetParser() antlr.Parser { return s.parser }

func (s *EuContext) EU() antlr.TerminalNode {
	return s.GetToken(RuleParserParserEU, 0)
}

func (s *EuContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EuContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EuContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterEu(s)
	}
}

func (s *EuContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitEu(s)
	}
}

func (p *RuleParserParser) Eu() (localctx IEuContext) {
	localctx = NewEuContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, RuleParserParserRULE_eu)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(88)
		p.Match(RuleParserParserEU)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IValContext is an interface to support dynamic dispatch.
type IValContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VAL() antlr.TerminalNode
	ASTERIX() antlr.TerminalNode

	// IsValContext differentiates from other interfaces.
	IsValContext()
}

type ValContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyValContext() *ValContext {
	var p = new(ValContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_val
	return p
}

func InitEmptyValContext(p *ValContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RuleParserParserRULE_val
}

func (*ValContext) IsValContext() {}

func NewValContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ValContext {
	var p = new(ValContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RuleParserParserRULE_val

	return p
}

func (s *ValContext) GetParser() antlr.Parser { return s.parser }

func (s *ValContext) VAL() antlr.TerminalNode {
	return s.GetToken(RuleParserParserVAL, 0)
}

func (s *ValContext) ASTERIX() antlr.TerminalNode {
	return s.GetToken(RuleParserParserASTERIX, 0)
}

func (s *ValContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ValContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ValContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.EnterVal(s)
	}
}

func (s *ValContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RuleParserListener); ok {
		listenerT.ExitVal(s)
	}
}

func (p *RuleParserParser) Val() (localctx IValContext) {
	localctx = NewValContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, RuleParserParserRULE_val)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(90)
		_la = p.GetTokenStream().LA(1)

		if !(_la == RuleParserParserASTERIX || _la == RuleParserParserVAL) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
