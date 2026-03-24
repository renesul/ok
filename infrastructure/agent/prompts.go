package agent

import (
	"fmt"
	"strings"

	"github.com/renesul/ok/domain"
)

func BuildPlanningPrompt(goal string, toolDescriptions string, memories []string) string {
	var parts []string

	parts = append(parts, `Voce e um planejador de tarefas. Dado um objetivo, crie um plano estruturado com 2 a 6 passos.

Cada passo deve usar uma das ferramentas disponiveis. Decomponha o objetivo em acoes atomicas e sequenciais.

GUIA: Mapeie a intencao do usuario para a ferramenta correta:
- pesquisar na internet/documentacao → web_search
- executar codigo → repl (respeite a linguagem: JavaScript=node, Python=python)
- navegar/abrir site → browser
- buscar em arquivos do projeto → search
- comando no terminal / git / npm / testes → shell
- ler arquivo → file_read
- criar arquivo → file_write
- editar arquivo → file_edit
- corrigir bug → file_read + file_edit + shell
Respeite TUDO que o usuario pediu explicitamente (linguagem, formato, ferramenta, tom). Nunca substitua sem perguntar.`)

	parts = append(parts, "Ferramentas disponiveis:\n"+toolDescriptions)

	if len(memories) > 0 {
		parts = append(parts, "Memorias relevantes:\n"+strings.Join(memories, "\n"))
	}

	parts = append(parts, fmt.Sprintf(`Objetivo: %s

Responda APENAS com JSON valido no formato:
{"steps":[{"name":"descricao curta","tool":"nome_da_tool","input":"valor","purpose":"por que este passo"}],"reasoning":"raciocinio geral do plano"}

IMPORTANTE: Use apenas tools que existem na lista acima. Cada step deve ter name, tool, input e purpose.`, goal))

	return strings.Join(parts, "\n\n")
}

func BuildReflectionPrompt(goal string, executedSteps []domain.PlannedStep, lastResult string) string {
	var stepsDesc []string
	for i, step := range executedSteps {
		status := step.Status
		if status == "" {
			status = "pending"
		}
		line := fmt.Sprintf("%d. [%s] %s (tool: %s)", i+1, status, step.Name, step.Tool)
		if step.Output != "" {
			output := TruncateWithEllipsis(step.Output, 300)
			line += "\n   Resultado: " + output
		}
		stepsDesc = append(stepsDesc, line)
	}

	return fmt.Sprintf(`Voce e um avaliador de execucao. Analise o progresso e decida o proximo passo.

Objetivo: %s

Passos executados:
%s

Ultimo resultado: %s

Analise:
- O resultado foi valido e util?
- O objetivo foi atingido?
- Precisa ajustar o plano?

Responda APENAS com JSON valido:
{"action":"continue|replan|done|error","reason":"justificativa curta","final_answer":"resposta final se action=done","revised_plan":[{"name":"...","tool":"...","input":"...","purpose":"..."}]}

Acoes:
- "continue": proximo passo do plano
- "replan": substituir passos restantes (fornecer revised_plan)
- "done": objetivo atingido (fornecer final_answer)
- "error": impossivel completar (fornecer reason)`, goal, strings.Join(stepsDesc, "\n"), lastResult)
}
