package require_error_cause

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildMissingCauseMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingCause",
		Description: "Thrown Error in catch block must include `{ cause }` from the caught error.",
	}
}

func buildIncorrectCauseMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "incorrectCause",
		Description: "The `cause` property must reference the caught error variable.",
	}
}

// findEnclosingCatchClause walks up the parent chain from the given node,
// looking for the nearest enclosing CatchClause. Returns nil if a function
// boundary is encountered first (nested functions are separate execution contexts).
func findEnclosingCatchClause(node *ast.Node) *ast.Node {
	current := node.Parent
	for current != nil {
		if ast.IsCatchClause(current) {
			return current
		}
		if ast.IsFunctionLikeOrClassStaticBlockDeclaration(current) {
			return nil
		}
		current = current.Parent
	}
	return nil
}

// resolveToNewExpression resolves the thrown expression to a NewExpression,
// either directly or by following a single variable declaration.
func resolveToNewExpression(ctx rule.RuleContext, expr *ast.Node) *ast.Node {
	unwrapped := ast.SkipParentheses(expr)

	if ast.IsNewExpression(unwrapped) {
		return unwrapped
	}

	if ast.IsIdentifier(unwrapped) {
		decl := utils.GetDeclaration(ctx.TypeChecker, unwrapped)
		if decl != nil && ast.IsVariableDeclaration(decl) {
			init := decl.AsVariableDeclaration().Initializer
			if init != nil {
				initUnwrapped := ast.SkipParentheses(init)
				if ast.IsNewExpression(initUnwrapped) {
					return initUnwrapped
				}
			}
		}
	}

	return nil
}

type causeStatus int

const (
	causeNone       causeStatus = iota // no cause property
	causeCorrect                       // cause references the catch variable
	causeIncorrect                     // cause exists but references something else
	causeUnverified                    // second arg is not an object literal; can't verify
)

// checkCauseProperty checks whether a NewExpression's second argument
// is an object literal containing a `cause` property that references the catch variable.
func checkCauseProperty(ctx rule.RuleContext, newExprNode *ast.Node, catchVarDecl *ast.Node) causeStatus {
	newExpr := newExprNode.AsNewExpression()
	if newExpr.Arguments == nil || len(newExpr.Arguments.Nodes) < 2 {
		return causeNone
	}

	optionsArg := ast.SkipParentheses(newExpr.Arguments.Nodes[1])

	if !ast.IsObjectLiteralExpression(optionsArg) {
		return causeUnverified
	}

	objLit := optionsArg.AsObjectLiteralExpression()
	if objLit.Properties == nil {
		return causeNone
	}

	catchSymbol := ctx.TypeChecker.GetSymbolAtLocation(catchVarDecl.Name())

	for _, prop := range objLit.Properties.Nodes {
		var propName *ast.Node
		var propValue *ast.Node

		var isShorthand bool

		if ast.IsPropertyAssignment(prop) {
			propName = prop.Name()
			propValue = prop.AsPropertyAssignment().Initializer
		} else if ast.IsShorthandPropertyAssignment(prop) {
			propName = prop.Name()
			isShorthand = true
		} else {
			continue
		}

		if propName == nil || !ast.IsIdentifier(propName) || propName.Text() != "cause" {
			continue
		}

		// Found a cause property; check if value references the catch variable
		if catchSymbol != nil {
			if isShorthand {
				// For shorthand { cause }, use GetShorthandAssignmentValueSymbol
				// to get the symbol of the value (the variable being referenced)
				valueSymbol := checker.Checker_GetShorthandAssignmentValueSymbol(ctx.TypeChecker, prop)
				if valueSymbol != nil && valueSymbol == catchSymbol {
					return causeCorrect
				}
			} else if propValue != nil {
				valueSymbol := ctx.TypeChecker.GetSymbolAtLocation(propValue)
				if valueSymbol != nil && valueSymbol == catchSymbol {
					return causeCorrect
				}
			}
		}
		return causeIncorrect
	}

	return causeNone
}

var RequireErrorCauseRule = rule.Rule{
	Name: "require-error-cause",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindThrowStatement: func(node *ast.Node) {
				expr := node.Expression()

				catchClause := findEnclosingCatchClause(node)
				if catchClause == nil {
					return
				}

				catchData := catchClause.AsCatchClause()
				if catchData.VariableDeclaration == nil {
					return
				}

				newExprNode := resolveToNewExpression(ctx, expr)
				if newExprNode == nil {
					return
				}

				exprType := ctx.TypeChecker.GetTypeAtLocation(newExprNode)
				if !utils.IsErrorLike(ctx.Program, ctx.TypeChecker, exprType) {
					return
				}

				status := checkCauseProperty(ctx, newExprNode, catchData.VariableDeclaration)
				switch status {
				case causeCorrect, causeUnverified:
					return
				case causeIncorrect:
					ctx.ReportNode(node, buildIncorrectCauseMessage())
				default:
					ctx.ReportNode(node, buildMissingCauseMessage())
				}
			},
		}
	},
}
