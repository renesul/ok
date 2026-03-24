package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/renesul/ok/application/engine"
	"github.com/renesul/ok/domain"
	agentpkg "github.com/renesul/ok/infrastructure/agent"
	"github.com/renesul/ok/infrastructure/llm"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AgentService struct {
	db               *gorm.DB
	llmClient        *llm.Client
	llmHeavyConfig   llm.ClientConfig
	llmFastConfig    llm.ClientConfig
	planner          domain.Planner
	executor         domain.Executor
	memory           *agentpkg.SQLiteMemory
	execRepo         *agentpkg.ExecutionRepository
	configRepo       *agentpkg.ConfigRepository
	soul             string
	identity         string
	userProfile      string
	environmentNotes string
	limits           domain.AgentLimits
	cachedPrompt     string
	log              *zap.Logger
}

func NewAgentService(
	db *gorm.DB,
	llmClient *llm.Client,
	llmHeavyConfig llm.ClientConfig,
	llmFastConfig llm.ClientConfig,
	planner domain.Planner,
	executor domain.Executor,
	memory *agentpkg.SQLiteMemory,
	execRepo *agentpkg.ExecutionRepository,
	configRepo *agentpkg.ConfigRepository,
	log *zap.Logger,
) *AgentService {
	s := &AgentService{
		db:             db,
		llmClient:      llmClient,
		llmHeavyConfig: llmHeavyConfig,
		llmFastConfig:  llmFastConfig,
		planner:        planner,
		executor:       executor,
		memory:         memory,
		execRepo:       execRepo,
		configRepo:     configRepo,
		log:            log.Named("service.agent"),
	}
	s.loadTemplates()
	return s
}

func (s *AgentService) Run(ctx context.Context, input string) (domain.AgentResponse, error) {
	s.log.Debug("agent start", zap.String("input", input))
	eng := s.buildEngine()
	emitter := engine.NewBufferEmitter()
	if err := eng.RunLoop(ctx, input, emitter); err != nil {
		return domain.AgentResponse{}, err
	}
	return emitter.Response(), nil
}

func (s *AgentService) RunStream(ctx context.Context, input string, onEvent domain.EventCallback) error {
	s.log.Debug("agent stream start", zap.String("input", input))
	emit := func(e domain.AgentEvent) {
		if onEvent != nil {
			onEvent(e)
		}
	}
	eng := s.buildEngine()
	emitter := engine.NewCallbackEmitter(emit)
	return eng.RunLoop(ctx, input, emitter)
}

func (s *AgentService) buildEngine() *engine.AgentEngine {
	return engine.NewAgentEngine(
		s.db, s.llmClient, s.llmHeavyConfig, s.llmFastConfig, s.planner, s.executor,
		s.memory, s.execRepo,
		s.limits, s.BuildSystemPrompt, s.log,
	)
}

func (s *AgentService) loadTemplates() {
	if s.configRepo == nil {
		return
	}
	ctx := context.Background()
	if v, err := s.configRepo.Get(ctx, "soul"); err == nil && v != "" {
		s.soul = v
	}
	if v, err := s.configRepo.Get(ctx, "identity"); err == nil && v != "" {
		s.identity = v
	}
	if v, err := s.configRepo.Get(ctx, "user_profile"); err == nil && v != "" {
		s.userProfile = v
	}
	if v, err := s.configRepo.Get(ctx, "environment_notes"); err == nil && v != "" {
		s.environmentNotes = v
	}
	s.limits = domain.DefaultAgentLimits()
	if v, err := s.configRepo.Get(ctx, "agent_limits"); err == nil && v != "" {
		var limits domain.AgentLimits
		if jsonErr := json.Unmarshal([]byte(v), &limits); jsonErr == nil {
			s.limits = limits
		}
	}
}

func (s *AgentService) ReloadSoul() {
	s.loadTemplates()
	s.cachedPrompt = ""
}

func (s *AgentService) GetConfigRepo() *agentpkg.ConfigRepository {
	return s.configRepo
}

func (s *AgentService) BuildSystemPrompt() string {
	if s.cachedPrompt != "" {
		return s.cachedPrompt
	}

	now := time.Now().Format("Monday, 2 January 2006, 15:04")

	var parts []string

	soul := s.soul
	if soul == "" {
		soul = "Voce e um assistente pessoal inteligente e direto."
	}
	parts = append(parts, soul)

	if s.identity != "" {
		parts = append(parts, s.identity)
	}
	if s.userProfile != "" {
		parts = append(parts, "Sobre o usuario: "+s.userProfile)
	}
	if s.environmentNotes != "" {
		parts = append(parts, "Ambiente: "+s.environmentNotes)
	}

	if s.memory != nil {
		rules, _ := s.memory.SearchByCategory("", "rule", 20)
		if len(rules) > 0 {
			var ruleTexts []string
			for _, r := range rules {
				ruleTexts = append(ruleTexts, "- "+r.Content)
			}
			parts = append(parts, "Regras aprendidas (OBEDECA SEMPRE):\n"+strings.Join(ruleTexts, "\n"))
		}
	}

	parts = append(parts, "Data e hora atual: "+now)

	parts = append(parts, fmt.Sprintf(`Ferramentas disponiveis:
%s

GUIA DE SELECAO DE FERRAMENTAS (use a mais especifica para cada pedido):
- Pesquisar na internet / buscar documentacao / solucao de erro → web_search
- Abrir/navegar site especifico → browser
- Buscar texto em arquivos do projeto → search
- Ler arquivo (com paginacao) → file_read
- Criar/escrever arquivo novo → file_write
- Editar trecho de arquivo existente → file_edit
- Executar codigo JS/Python/Bash → repl (language:"node"|"python"|"bash")
- Executar comando no terminal / git / npm / testes → shell
- Corrigir bug → file_read + file_edit + shell (rodar testes)
- Escrever testes → file_read + file_write + shell
- Instalar pacote → shell (npm install / pip install / go get)
- Fazer commit/push → shell (git add/commit/push)
- Calcular expressao → math
- Parsear JSON → json_parse
- Agendar tarefa recorrente → schedule
- Fazer request HTTP para API → http
- Converter base64 → base64
- Extrair texto de HTML → text_extract
- Listar diretorio completo → folder_index
- Tarefa complexa (subdividir em partes) → delegate

REGRAS:
- Respeite EXATAMENTE o que o usuario pediu (linguagem, formato, ferramenta, tom)
- Se pediu JavaScript → repl com language:"node"
- Se pediu CSV → gere CSV, nao JSON
- Se pediu shell → use shell, nao repl
- Nunca substitua a escolha do usuario sem perguntar
- Para conversa normal sem acao → responda direto com done=true

Para usar uma ferramenta, responda APENAS com JSON:
{"tool":"nome","input":"valor","done":false}

Para responder diretamente (sem ferramenta), responda APENAS com JSON:
{"tool":"","input":"sua resposta aqui","done":true}

IMPORTANTE: Responda SEMPRE em JSON valido.`, s.planner.ToolDescriptions()))

	s.cachedPrompt = joinNonEmpty(parts, "\n\n")

	go func() {
		time.Sleep(1 * time.Minute)
		s.cachedPrompt = ""
	}()

	return s.cachedPrompt
}

func joinNonEmpty(parts []string, sep string) string {
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return strings.Join(result, sep)
}

func (s *AgentService) GetExecution(id string) (*domain.ExecutionRecord, error) {
	if s.execRepo == nil {
		return nil, nil
	}
	return s.execRepo.FindByID(id)
}

func (s *AgentService) GetRecentExecutions(limit int) ([]domain.ExecutionRecord, error) {
	if s.execRepo == nil {
		return nil, nil
	}
	return s.execRepo.FindRecent(limit)
}

func (s *AgentService) GetLimits() domain.AgentLimits {
	return s.limits
}

func (s *AgentService) SetLimits(ctx context.Context, limits domain.AgentLimits) error {
	data, err := json.Marshal(limits)
	if err != nil {
		return fmt.Errorf("marshal limits: %w", err)
	}
	if err := s.configRepo.Set(ctx, "agent_limits", string(data)); err != nil {
		return fmt.Errorf("save limits: %w", err)
	}
	s.limits = limits
	return nil
}

func (s *AgentService) GetMetrics() (*domain.ExecutionMetrics, error) {
	if s.execRepo == nil {
		return nil, nil
	}
	return s.execRepo.GetMetrics()
}
