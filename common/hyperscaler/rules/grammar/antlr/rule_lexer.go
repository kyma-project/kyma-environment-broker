// Code generated from RuleLexer.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type RuleLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var RuleLexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func rulelexerLexerInit() {
	staticData := &RuleLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'='", "','", "'('", "')'", "'->'", "'*'", "", "", "'PR'", "'HR'",
		"'S'", "'EU'",
	}
	staticData.SymbolicNames = []string{
		"", "EQ", "COMMA", "LPAREN", "RPAREN", "ARROW", "ASTERIX", "WS", "PLAN",
		"PR", "HR", "S", "EU", "VAL",
	}
	staticData.RuleNames = []string{
		"EQ", "COMMA", "LPAREN", "RPAREN", "ARROW", "ASTERIX", "WS", "PLAN",
		"PR", "HR", "S", "EU", "VAL",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 13, 120, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1,
		2, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 6, 4, 6, 42, 8, 6, 11,
		6, 12, 6, 43, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1,
		7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1,
		7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1,
		7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1,
		7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 3, 7, 97, 8, 7, 1, 8, 1, 8, 1, 8, 1, 9,
		1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1, 12, 5, 12, 111, 8, 12,
		10, 12, 12, 12, 114, 9, 12, 1, 12, 4, 12, 117, 8, 12, 11, 12, 12, 12, 118,
		0, 0, 13, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19,
		10, 21, 11, 23, 12, 25, 13, 1, 0, 3, 3, 0, 9, 10, 12, 13, 32, 32, 4, 0,
		45, 45, 65, 90, 95, 95, 97, 122, 4, 0, 48, 57, 65, 90, 95, 95, 97, 122,
		128, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0,
		0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1,
		0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23,
		1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 1, 27, 1, 0, 0, 0, 3, 29, 1, 0, 0, 0, 5,
		31, 1, 0, 0, 0, 7, 33, 1, 0, 0, 0, 9, 35, 1, 0, 0, 0, 11, 38, 1, 0, 0,
		0, 13, 41, 1, 0, 0, 0, 15, 96, 1, 0, 0, 0, 17, 98, 1, 0, 0, 0, 19, 101,
		1, 0, 0, 0, 21, 104, 1, 0, 0, 0, 23, 106, 1, 0, 0, 0, 25, 112, 1, 0, 0,
		0, 27, 28, 5, 61, 0, 0, 28, 2, 1, 0, 0, 0, 29, 30, 5, 44, 0, 0, 30, 4,
		1, 0, 0, 0, 31, 32, 5, 40, 0, 0, 32, 6, 1, 0, 0, 0, 33, 34, 5, 41, 0, 0,
		34, 8, 1, 0, 0, 0, 35, 36, 5, 45, 0, 0, 36, 37, 5, 62, 0, 0, 37, 10, 1,
		0, 0, 0, 38, 39, 5, 42, 0, 0, 39, 12, 1, 0, 0, 0, 40, 42, 7, 0, 0, 0, 41,
		40, 1, 0, 0, 0, 42, 43, 1, 0, 0, 0, 43, 41, 1, 0, 0, 0, 43, 44, 1, 0, 0,
		0, 44, 45, 1, 0, 0, 0, 45, 46, 6, 6, 0, 0, 46, 14, 1, 0, 0, 0, 47, 48,
		5, 97, 0, 0, 48, 49, 5, 122, 0, 0, 49, 50, 5, 117, 0, 0, 50, 51, 5, 114,
		0, 0, 51, 97, 5, 101, 0, 0, 52, 53, 5, 97, 0, 0, 53, 54, 5, 122, 0, 0,
		54, 55, 5, 117, 0, 0, 55, 56, 5, 114, 0, 0, 56, 57, 5, 101, 0, 0, 57, 58,
		5, 95, 0, 0, 58, 59, 5, 108, 0, 0, 59, 60, 5, 105, 0, 0, 60, 61, 5, 116,
		0, 0, 61, 97, 5, 101, 0, 0, 62, 63, 5, 97, 0, 0, 63, 64, 5, 119, 0, 0,
		64, 97, 5, 115, 0, 0, 65, 66, 5, 103, 0, 0, 66, 67, 5, 99, 0, 0, 67, 97,
		5, 112, 0, 0, 68, 69, 5, 116, 0, 0, 69, 70, 5, 114, 0, 0, 70, 71, 5, 105,
		0, 0, 71, 72, 5, 97, 0, 0, 72, 97, 5, 108, 0, 0, 73, 74, 5, 102, 0, 0,
		74, 75, 5, 114, 0, 0, 75, 76, 5, 101, 0, 0, 76, 97, 5, 101, 0, 0, 77, 78,
		5, 115, 0, 0, 78, 79, 5, 97, 0, 0, 79, 80, 5, 112, 0, 0, 80, 81, 5, 45,
		0, 0, 81, 82, 5, 99, 0, 0, 82, 83, 5, 111, 0, 0, 83, 84, 5, 110, 0, 0,
		84, 85, 5, 118, 0, 0, 85, 86, 5, 101, 0, 0, 86, 87, 5, 114, 0, 0, 87, 88,
		5, 103, 0, 0, 88, 89, 5, 101, 0, 0, 89, 90, 5, 100, 0, 0, 90, 91, 5, 45,
		0, 0, 91, 92, 5, 99, 0, 0, 92, 93, 5, 108, 0, 0, 93, 94, 5, 111, 0, 0,
		94, 95, 5, 117, 0, 0, 95, 97, 5, 100, 0, 0, 96, 47, 1, 0, 0, 0, 96, 52,
		1, 0, 0, 0, 96, 62, 1, 0, 0, 0, 96, 65, 1, 0, 0, 0, 96, 68, 1, 0, 0, 0,
		96, 73, 1, 0, 0, 0, 96, 77, 1, 0, 0, 0, 97, 16, 1, 0, 0, 0, 98, 99, 5,
		80, 0, 0, 99, 100, 5, 82, 0, 0, 100, 18, 1, 0, 0, 0, 101, 102, 5, 72, 0,
		0, 102, 103, 5, 82, 0, 0, 103, 20, 1, 0, 0, 0, 104, 105, 5, 83, 0, 0, 105,
		22, 1, 0, 0, 0, 106, 107, 5, 69, 0, 0, 107, 108, 5, 85, 0, 0, 108, 24,
		1, 0, 0, 0, 109, 111, 7, 1, 0, 0, 110, 109, 1, 0, 0, 0, 111, 114, 1, 0,
		0, 0, 112, 110, 1, 0, 0, 0, 112, 113, 1, 0, 0, 0, 113, 116, 1, 0, 0, 0,
		114, 112, 1, 0, 0, 0, 115, 117, 7, 2, 0, 0, 116, 115, 1, 0, 0, 0, 117,
		118, 1, 0, 0, 0, 118, 116, 1, 0, 0, 0, 118, 119, 1, 0, 0, 0, 119, 26, 1,
		0, 0, 0, 5, 0, 43, 96, 112, 118, 1, 6, 0, 0,
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

// RuleLexerInit initializes any static state used to implement RuleLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewRuleLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func RuleLexerInit() {
	staticData := &RuleLexerLexerStaticData
	staticData.once.Do(rulelexerLexerInit)
}

// NewRuleLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewRuleLexer(input antlr.CharStream) *RuleLexer {
	RuleLexerInit()
	l := new(RuleLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &RuleLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "RuleLexer.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// RuleLexer tokens.
const (
	RuleLexerEQ      = 1
	RuleLexerCOMMA   = 2
	RuleLexerLPAREN  = 3
	RuleLexerRPAREN  = 4
	RuleLexerARROW   = 5
	RuleLexerASTERIX = 6
	RuleLexerWS      = 7
	RuleLexerPLAN    = 8
	RuleLexerPR      = 9
	RuleLexerHR      = 10
	RuleLexerS       = 11
	RuleLexerEU      = 12
	RuleLexerVAL     = 13
)
