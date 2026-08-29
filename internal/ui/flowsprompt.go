package ui

import (
	"fmt"
	"strings"
)

// generatePhasePrompt is the instruction a phase is given when the reader has
// not written one: what they were drafting if there is a draft, and what the
// phase is called if there is not.
//
// The keywords are read in both languages because the draft is whatever the
// operator typed, and what comes back is English because that is the language
// the rest of the prompt speaks to the engine in — the phase instructions land
// under a heading of it, and a prompt that changes language halfway is a
// prompt asking to be read twice.
func generatePhasePrompt(userInput, phaseName, flowName string) string {
	raw := strings.TrimSpace(userInput)
	lower := strings.ToLower(raw)
	phLower := strings.ToLower(phaseName)

	if raw != "" {
		return draftPrompt(raw, lower, phaseName)
	}

	switch {
	case strings.Contains(phLower, "plan") || strings.Contains(phLower, "design") ||
		strings.Contains(phLower, "arch"):
		return "Read the requirements, study the code that is already there, and design a " +
			"technical plan with the cases it has to pass."
	case strings.Contains(phLower, "impl") || strings.Contains(phLower, "code") ||
		strings.Contains(phLower, "dev") || strings.Contains(phLower, "build"):
		return "Implement the agreed plan in full, keeping the code modular and to the standard " +
			"the repository already holds."
	case strings.Contains(phLower, "test") || strings.Contains(phLower, "gate") ||
		strings.Contains(phLower, "check") || strings.Contains(phLower, "qa"):
		return "Write and run the automated tests that prove the implemented behaviour, the edge " +
			"cases included."
	case strings.Contains(phLower, "review") || strings.Contains(phLower, "audit") ||
		strings.Contains(phLower, "sec"):
		return "Audit the diff for vulnerabilities, leaked resources and regressions."
	case strings.Contains(phLower, "fix") || strings.Contains(phLower, "patch") ||
		strings.Contains(phLower, "remed"):
		return "Fix the failures and findings the phase before reported, and leave every check green."
	default:
		if flowName != "" {
			return fmt.Sprintf("Carry out the %s phase of the %s flow.", phaseName, flowName)
		}

		return fmt.Sprintf("Carry out the work the %s phase calls for, on your own.", phaseName)
	}
}

// draftPrompt is the instruction read out of what the reader was drafting.
// The draft is carried into it whole, because it is the one part of the
// sentence the reader wrote and a phase told something they did not write is
// a phase running on this program's guess.
func draftPrompt(raw, lower, phaseName string) string {
	switch {
	case strings.Contains(lower, "valid") || strings.Contains(lower, "test") ||
		strings.Contains(lower, "prob") || strings.Contains(lower, "verif") ||
		strings.Contains(lower, "check"):
		return fmt.Sprintf("Validate what was implemented: run the automated suites, cover the "+
			"edge cases, and prove nothing regressed. Context: %s.", raw)
	case strings.Contains(lower, "sec") || strings.Contains(lower, "segur") ||
		strings.Contains(lower, "audit") || strings.Contains(lower, "vuln"):
		return fmt.Sprintf("Audit the code for security holes: input validation, secret handling, "+
			"and anything that trusts what it should not. Context: %s.", raw)
	case strings.Contains(lower, "refactor") || strings.Contains(lower, "limp") ||
		strings.Contains(lower, "clean") || strings.Contains(lower, "orden"):
		return fmt.Sprintf("Refactor for clarity, modularity and speed, without breaking a "+
			"contract something already depends on. Context: %s.", raw)
	case strings.Contains(lower, "fix") || strings.Contains(lower, "correg") ||
		strings.Contains(lower, "repar") || strings.Contains(lower, "bug") ||
		strings.Contains(lower, "error"):
		return fmt.Sprintf("Find the root cause of the reported failures, fix them, and prove "+
			"every check passes. Context: %s.", raw)
	case strings.Contains(lower, "doc") || strings.Contains(lower, "coment") ||
		strings.Contains(lower, "readme"):
		return fmt.Sprintf("Write the technical documentation: the architecture, how it is "+
			"configured, and worked examples of using it. Context: %s.", raw)
	default:
		return fmt.Sprintf("Carry out this instruction for the %s phase: %s. Keep to the "+
			"architecture and quality rules the repository already sets.", phaseName, raw)
	}
}
