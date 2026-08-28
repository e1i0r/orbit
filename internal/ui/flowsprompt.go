package ui

import (
	"fmt"
	"strings"
)

func generatePhasePrompt(userInput, phaseName, flowName string) string {
	raw := strings.TrimSpace(userInput)
	lower := strings.ToLower(raw)
	phLower := strings.ToLower(phaseName)

	if raw != "" {
		switch {
		case strings.Contains(lower, "valid") || strings.Contains(lower, "test") ||
			strings.Contains(lower, "prob") || strings.Contains(lower, "verif") ||
			strings.Contains(lower, "check"):
			return fmt.Sprintf("Valida exhaustivamente todo el código implementado: ejecuta las "+
				"suites de pruebas automatizadas, verifica casos límite y asegura que no existan "+
				"regresiones. Contexto: %s.", raw)
		case strings.Contains(lower, "sec") || strings.Contains(lower, "segur") ||
			strings.Contains(lower, "audit") || strings.Contains(lower, "vuln"):
			//nolint:misspell // Spanish prompt template
			return fmt.Sprintf("Audita rigurosamente el código en busca de vulnerabilidades de "+
				"seguridad, validación de entradas y manejo seguro de secretos. Contexto: %s.", raw)
		case strings.Contains(lower, "refactor") || strings.Contains(lower, "limp") ||
			strings.Contains(lower, "clean") || strings.Contains(lower, "orden"):
			return fmt.Sprintf("Refactoriza y optimiza la estructura del código para máxima "+
				"claridad, modularidad y rendimiento sin romper contratos existentes. Contexto: %s.", raw)
		case strings.Contains(lower, "fix") || strings.Contains(lower, "correg") ||
			strings.Contains(lower, "repar") || strings.Contains(lower, "bug") ||
			strings.Contains(lower, "error"):
			return fmt.Sprintf("Investiga la causa raíz de los errores reportados, aplica las "+
				"correcciones necesarias y verifica que pasen todos los chequeos. Contexto: %s.", raw)
		case strings.Contains(lower, "doc") || strings.Contains(lower, "coment") ||
			strings.Contains(lower, "readme"):
			return fmt.Sprintf("Genera documentación técnica detallada, clara y concisa explicando "+
				"la arquitectura, configuración y ejemplos de uso. Contexto: %s.", raw)
		default:
			return fmt.Sprintf("Ejecuta con máxima precisión técnica la siguiente instrucción para "+
				"la fase %s: %s. Respeta las directivas de arquitectura y calidad.", phaseName, raw)
		}
	}

	switch {
	case strings.Contains(phLower, "plan") || strings.Contains(phLower, "design") ||
		strings.Contains(phLower, "arch"):
		return "Analiza en detalle los requisitos, examina el código existente y diseña un plan " +
			"técnico estructurado con casos de prueba."
	case strings.Contains(phLower, "impl") || strings.Contains(phLower, "code") ||
		strings.Contains(phLower, "dev") || strings.Contains(phLower, "build"):
		return "Implementa la solución completa siguiendo el plan acordado, asegurando calidad de " +
			"código, modularidad y buenas prácticas."
	case strings.Contains(phLower, "test") || strings.Contains(phLower, "gate") ||
		strings.Contains(phLower, "check") || strings.Contains(phLower, "qa"):
		return "Escribe y ejecuta pruebas automatizadas completas para verificar exhaustivamente " +
			"la funcionalidad implementada."
	case strings.Contains(phLower, "review") || strings.Contains(phLower, "audit") ||
		strings.Contains(phLower, "sec"):
		return "Audita el diff de cambios generados, buscando posibles vulnerabilidades, fugas de " +
			"recursos o regresiones."
	case strings.Contains(phLower, "fix") || strings.Contains(phLower, "patch") ||
		strings.Contains(phLower, "remed"):
		return "Corrige con precisión los errores y hallazgos reportados en la fase anterior hasta " +
			"dejar el sistema impecable."
	default:
		if flowName != "" {
			return fmt.Sprintf("Ejecuta la fase %s para el flujo %s con máxima rigurosidad técnica.",
				phaseName, flowName)
		}

		return fmt.Sprintf("Ejecuta las tareas correspondientes a la fase %s de forma autónoma "+
			"y precisa.", phaseName)
	}
}
