package require_error_cause

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireErrorCauseRule(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.minimal.json", t, &RequireErrorCauseRule, []rule_tester.ValidTestCase{
		// Throw outside of catch block
		{Code: "throw new Error('msg');"},
		// Re-throw caught error (not creating new Error)
		{Code: `
try {
} catch (e) {
  throw e;
}
    `},
		// Throw with cause set to caught error
		{Code: `
try {
} catch (e) {
  throw new Error('msg', { cause: e });
}
    `},
		// Error subclass with cause
		{Code: `
try {
} catch (e) {
  throw new TypeError('msg', { cause: e });
}
    `},
		// Shorthand cause property (catch variable named "cause")
		{Code: `
try {
} catch (cause) {
  throw new Error('msg', { cause });
}
    `},
		// Variable indirection with cause
		{Code: `
try {
} catch (e) {
  const err = new Error('msg', { cause: e });
  throw err;
}
    `},
		// Throw non-Error inside catch (not our concern)
		{Code: `
try {
} catch (e) {
  throw 'string';
}
    `},
		// Throw inside nested arrow function (function boundary)
		{Code: `
try {
} catch (e) {
  setTimeout(() => { throw new Error('msg'); }, 0);
}
    `},
		// Throw inside nested function expression (function boundary)
		{Code: `
try {
} catch (e) {
  const fn = function() { throw new Error('msg'); };
}
    `},
		// Catch without binding
		{Code: `
try {
} catch {
  throw new Error('msg');
}
    `},
		// Non-literal second argument (can't verify, conservatively allow)
		{Code: `
try {
} catch (e) {
  const opts = { cause: e };
  throw new Error('msg', opts);
}
    `},
		// Non-Error-like new expression
		{Code: `
try {
} catch (e) {
  throw new Map();
}
    `},
		// Throw function call result (not new expression)
		{Code: `
declare function createError(): Error;
try {
} catch (e) {
  throw createError();
}
    `},
		// Cause with additional properties
		{Code: `
try {
} catch (e) {
  throw new Error('msg', { cause: e, extra: 1 });
}
    `},
		// Nested catch: inner catch with correct cause
		{Code: `
try {
} catch (outer) {
  try {
  } catch (inner) {
    throw new Error('msg', { cause: inner });
  }
}
    `},
	}, []rule_tester.InvalidTestCase{
		// Basic: new Error without cause
		{
			Code: `
try {
} catch (e) {
  throw new Error('msg');
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Empty options object
		{
			Code: `
try {
} catch (e) {
  throw new Error('msg', {});
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Options without cause property
		{
			Code: `
try {
} catch (e) {
  throw new Error('msg', { message: 'other' });
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Cause references wrong variable
		{
			Code: `
declare const someOther: Error;
try {
} catch (e) {
  throw new Error('msg', { cause: someOther });
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "incorrectCause"},
			},
		},
		// Error subclass without cause
		{
			Code: `
try {
} catch (e) {
  throw new TypeError('msg');
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Custom Error subclass without cause
		{
			Code: `
class CustomError extends Error {}
try {
} catch (e) {
  throw new CustomError('msg');
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Variable indirection without cause
		{
			Code: `
try {
} catch (e) {
  const err = new Error('msg');
  throw err;
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Error with no arguments
		{
			Code: `
try {
} catch (e) {
  throw new Error();
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Nested catch: inner throw without cause
		{
			Code: `
try {
} catch (outer) {
  try {
  } catch (inner) {
    throw new Error('msg');
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Parenthesized new expression
		{
			Code: `
try {
} catch (e) {
  throw (new Error('msg'));
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Conditional throw without cause
		{
			Code: `
try {
} catch (e) {
  if (true) {
    throw new Error('msg');
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingCause"},
			},
		},
		// Nested catch: using outer catch variable instead of inner
		{
			Code: `
try {
} catch (outer) {
  try {
  } catch (inner) {
    throw new Error('msg', { cause: outer });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "incorrectCause"},
			},
		},
	})
}
