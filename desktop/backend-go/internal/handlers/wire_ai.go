package handlers

import (
	"github.com/rhl/businessos-backend/internal/feedback"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/subconscious"
)

// SetPedroServices sets the Pedro task services (optional, to avoid breaking existing code)
func (h *Handlers) SetPedroServices(
	documentProcessor *services.DocumentProcessor,
	learningService *services.LearningService,
	autoLearningTriggers *services.AutoLearningTriggers,
	promptPersonalizer *services.PromptPersonalizer,
	appProfilerService *services.AppProfilerService,
	conversationIntelligence *services.ConversationIntelligenceService,
	memoryExtractor *services.MemoryExtractorService,
	blockMapper *services.BlockMapperService,
) {
	h.documentProcessor = documentProcessor
	h.learningService = learningService
	h.autoLearningTriggers = autoLearningTriggers
	h.promptPersonalizer = promptPersonalizer
	h.appProfilerService = appProfilerService
	h.conversationIntelligence = conversationIntelligence
	h.memoryExtractor = memoryExtractor
	h.blockMapper = blockMapper
}

// SetRAGServices sets the RAG services (Day 2)
func (h *Handlers) SetRAGServices(
	hybridSearch *services.HybridSearchService,
	reranker *services.ReRankerService,
	agenticRAG *services.AgenticRAGService,
	memory *services.MemoryService,
) {
	h.hybridSearchService = hybridSearch
	h.rerankerService = reranker
	h.agenticRAGService = agenticRAG
	h.memoryService = memory
}

// SetMultiModalServices sets the multi-modal search services (Feature 7)
func (h *Handlers) SetMultiModalServices(
	multiModalSearch *services.MultiModalSearchService,
	imageEmbedding *services.ImageEmbeddingService,
) {
	h.multiModalHandler = NewMultiModalSearchHandler(multiModalSearch, imageEmbedding)
}

// SetSkillsLoader sets the skills loader (Agent Skills System)
func (h *Handlers) SetSkillsLoader(skillsLoader *services.SkillsLoader) {
	h.skillsLoader = skillsLoader
}

// SetContextTracker sets the context tracker service (per-conversation token budget)
func (h *Handlers) SetContextTracker(svc *services.ContextTrackerService) {
	h.contextTracker = svc
}

// SetModeTransitionService sets the mode transition tracking service
func (h *Handlers) SetModeTransitionService(svc *services.ModeTransitionService) {
	h.modeTransitionSvc = svc
}

// SetSessionHealthService sets the session health metrics service
func (h *Handlers) SetSessionHealthService(svc *services.SessionHealthService) {
	h.sessionHealthSvc = svc
}

// SetSignalHints sets the homeostatic feedback hint provider (Signal Theory).
func (h *Handlers) SetSignalHints(provider feedback.SignalHintProvider) {
	h.signalHints = provider
}

// SetSubconsciousObserver sets the subconscious observer for async pattern detection.
func (h *Handlers) SetSubconsciousObserver(obs *subconscious.Observer) {
	h.subconsciousObserver = obs
}
