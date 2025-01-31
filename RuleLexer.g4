// DELETE THIS CONTENT IF YOU PUT COMBINED GRAMMAR IN Parser TAB
lexer grammar RuleLexer;

EQ : '=' ;
COMMA : ',' ;
LPAREN : '('  ;
RPAREN : ')' ;
ARROW : '->' ;
ASTERIX : '*' ;

WS: [ \t\n\r\f]+ -> skip ;

// TODO: fill in missing plans
PLAN: 'azure'
    | 'azure_lite'
    | 'aws'
    | 'gcp'
    | 'trial'
    | 'free'
    | 'sap-converged-cloud'
    ;
    
PR: 'PR';
HR: 'HR';
    
S: 'S';
EU: 'EU';
    
VAL: [a-zA-Z_-]*[a-zA-Z_0-9]+;