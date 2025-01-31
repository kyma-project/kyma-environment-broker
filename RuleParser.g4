grammar RuleParser;
options { tokenVocab=RuleLexer; }

ruleEntry
    : entry EOF
    ;

entry: PLAN
    | PLAN ARROW outputAttrList
    | PLAN inputAttrInParen
    | PLAN inputAttrInParen ARROW outputAttrList
    ;

inputAttrInParen: LPAREN RPAREN
    | LPAREN inputAttrList RPAREN
    ;
    
inputAttrList: inputAttrVal
    | inputAttrVal COMMA inputAttrList
    ;
    
outputAttrList: outputAttrVal
    | outputAttrVal COMMA outputAttrList
    ;
    
inputAttrVal: prVal
    | hrVal
    ;
    
outputAttrVal: s
    | eu
    ;

prVal: pr EQ val;
    
hrVal: hr EQ val;

pr: PR;

hr: HR;

s: S;

eu: EU;

val: VAL 
    | ASTERIX
    ;