package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

type helpState struct {
	prevScreen screen
	offset     int
}

func (m Model) openHelp() Model {
	prev := m.screen
	if prev == screenHelp {
		prev = screenList
	}
	m.screen = screenHelp
	m.help = helpState{prevScreen: prev}
	return m
}

func (m Model) abandonHelp() Model {
	target := m.help.prevScreen
	if target == screenHelp {
		target = screenList
	}
	m.help = helpState{}
	m.screen = target
	return m
}

func (m Model) helpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Help), key.Matches(msg, m.keys.Open):
		return m.abandonHelp(), nil
	case key.Matches(msg, m.keys.Up):
		if m.help.offset > 0 {
			m.help.offset--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.help.offset++
		return m, nil
	}
	return m, nil
}

func (m Model) helpRows(h, w int) []string {
	if h <= 0 {
		return nil
	}
	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Bold(true).Render("Ayuda y Atajos de Teclado (Cheat Sheet)"),
		"  " + Paint(Dim).Render("todas las funciones se pueden accionar por teclado o haciendo clic directo"),
		"",
	}

	renderSection := func(title string, items [][2]string) {
		out = append(out, "  "+Paint(Live).Bold(true).Render(title))
		for _, item := range items {
			k := pad(item[0], 28, false)
			line := "    " + Paint(Accent).Render(k) + " " + Paint(Dim).Render(item[1])
			out = append(out, fit(line, w))
		}
		out = append(out, "")
	}

	renderSection("📋 1. TABLERO GENERAL (Board & Queues)", [][2]string{
		{"[↑↓] o [j/k]", "Moverse entre tareas y secciones del tablero"},
		{"[⏎ Enter] o Clic", "Abrir detalle de tarea seleccionada o colapsar cola"},
		{"[n]", "Poner en marcha / Iniciar nueva ejecución de tarea"},
		{"[c]", "Crear una nueva tarea en el repositorio actual"},
		{"[/]", "Búsqueda y filtro en tiempo real (ID, título, repo)"},
		{"[Esc]", "Limpiar filtros activos o volver a la vista anterior"},
		{"[orbit] (clic)", "Resetear todos los filtros y mostrar tablero completo"},
		{"[📋⚡💬🏁] (clic)", "Filtrar para ver únicamente las tareas de esa cola"},
	})

	renderSection("⚡ 2. CONTROL Y CONFIGURACIÓN EN VIVO", [][2]string{
		{"[A] o ⚡ clic", "Alternar Autopilot (ejecución autónoma de tareas en To Do)"},
		{"[M] o 🧠 clic", "Selector de Motor IA (claude, codex, opencode, esfuerzo y thinking)"},
		{"[S]", "Pantalla de Ajustes (idiomas, temas visuales, límites)"},
		{"[R] o 📦 clic", "Selector modal de repositorios conectados"},
		{"🌐 ES / EN (clic)", "Alternar idioma del sistema entre Español e Inglés en vivo"},
	})

	renderSection("🔍 3. DETALLE DE TAREA (11 Tabs)", [][2]string{
		{"[1] Resumen", "Ver estado, duración, modelo, tokens y costes"},
		{"[2] Flujo", "Ver fases del pipeline del ciclo de vida"},
		{"[3] Gates", "Comprobaciones y pruebas de verificación automática"},
		{"[4] Coste", "Gasto financiero real acumulado"},
		{"[6] Cronología", "Línea de tiempo de eventos y logs en tiempo real"},
		{"[7] Informe", "Informe final generado por el agente IA"},
		{"[0] Cambios", "Visor de Git Diff con sintaxis coloreada"},
		{"[w] Thinking", "Cadena de pensamiento profundo del modelo"},
		{"[Tab / Shift+Tab]", "Pestaña siguiente / anterior"},
		{"[p] / [u] / [s]", "Pausar / Desbloquear / Añadir notas de operador"},
	})

	renderSection("⌨️ 4. COMANDOS GLOBALES", [][2]string{
		{"[:]", "Abrir paleta de comandos interactiva (orbit new, flows, set...)"},
		{"[?]", "Abrir o cerrar esta ventana de ayuda"},
		{"[q]", "Salir de Orbit"},
	})

	waysOut := p.T("help.ways_out", "{up_down} scroll · {back} back",
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("back", m.keys.Back.Help().Key))
	out = append(out, fit("  "+Paint(Dim).Render(waysOut), w))

	if m.help.offset > 0 {
		if m.help.offset >= len(out) {
			m.help.offset = len(out) - 1
		}
		out = out[m.help.offset:]
	}
	return fill(out, h)
}
