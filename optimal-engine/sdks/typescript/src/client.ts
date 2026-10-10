import { HttpClient } from "./http.js";
import { MemoryClient } from "./memory.js";
import { WikiClient } from "./profile.js";
import { RecallClient } from "./recall.js";
import { RenderClient } from "./render.js";
import { RetrievalClient } from "./retrieval.js";
import { SubscriptionClient, SurfaceClient } from "./surface.js";
import type {
  ArchitectureEntry,
  AskOptions,
  AskResult,
  GrepOptions,
  GrepResult,
  OptimalEngineConfig,
  ProfileOptions,
  ProfileResult,
  SearchOptions,
  SearchResult,
  StatusResult,
} from "./types.js";
import { WorkspaceClient } from "./workspace.js";

export class OptimalEngine {
  /** Raw HTTP client — exposed for advanced use; prefer the typed sub-clients. */
  readonly http: HttpClient;

  /** Memory CRUD operations. */
  readonly memory: MemoryClient;

  /** Workspace management. */
  readonly workspaces: WorkspaceClient;

  /** Wiki articles and contradiction detection. */
  readonly wiki: WikiClient;

  /** Subscription management. */
  readonly subscriptions: SubscriptionClient;

  /** Server-sent event surface streams. */
  readonly surface: SurfaceClient;

  /** Recall queries (actions / who / when / where / owns). */
  readonly recall: RecallClient;

  /** Multi-audience render pipeline (1 source → N HTML renders). */
  readonly render: RenderClient;

  private readonly retrieval: RetrievalClient;
  private readonly defaultWorkspace: string | undefined;

  constructor(config: OptimalEngineConfig = {}) {
    const baseUrl = config.baseUrl ?? "http://localhost:4200";
    this.http = new HttpClient(baseUrl, config.apiKey);
    this.defaultWorkspace = config.workspace;

    this.retrieval = new RetrievalClient(this.http, this.defaultWorkspace);
    this.memory = new MemoryClient(this.http, this.defaultWorkspace);
    this.workspaces = new WorkspaceClient(this.http);
    this.wiki = new WikiClient(this.http, this.defaultWorkspace);
    this.subscriptions = new SubscriptionClient(
      this.http,
      this.defaultWorkspace,
    );
    this.surface = new SurfaceClient(this.http, this.defaultWorkspace);
    this.recall = new RecallClient(this.http, this.defaultWorkspace);
    this.render = new RenderClient(this.http, this.defaultWorkspace);
  }

  // ---------------------------------------------------------------------------
  // Top-level retrieval shortcuts (mirror the task's client API spec)
  // ---------------------------------------------------------------------------

  /**
   * Ask the second brain a question. Curated wiki first, hybrid search second.
   * Returns an ACL-scoped, audience-shaped, bandwidth-matched envelope with
   * hot citations.
   */
  ask(query: string, opts: AskOptions = {}): Promise<AskResult> {
    return this.retrieval.ask(query, opts);
  }

  /**
   * Hybrid semantic + keyword search across memory.
   */
  search(query: string, opts: SearchOptions = {}): Promise<SearchResult> {
    return this.retrieval.search(query, opts);
  }

  /**
   * Structured grep across memory with intent, scale, and modality filters.
   */
  grep(query: string, opts: GrepOptions = {}): Promise<GrepResult> {
    return this.retrieval.grep(query, opts);
  }

  /**
   * Get a 4-tier workspace profile: static (ground truth), dynamic (rolling),
   * curated (wiki summary), activity (recent + top entities).
   */
  profile(opts: ProfileOptions = {}): Promise<ProfileResult> {
    return this.retrieval.profile(opts);
  }

  // ---------------------------------------------------------------------------
  // Lifecycle — drive the engine, not just read it.
  // ingest -> source -> signal -> claim -> fact -> memory (+ chunks, embeddings,
  // episodes, semantic edges). assemble -> MCTS tiered context. createNode ->
  // topology gazetteer used for entity extraction.
  // ---------------------------------------------------------------------------

  /**
   * Ingest raw content into a workspace's knowledge. Runs the full intake
   * lifecycle: classify, route, decompose, embed, extract claims, episodes,
   * and semantic graph edges.
   */
  ingest(
    text: string,
    opts: {
      workspace?: string;
      genre?: string;
      title?: string;
      node?: string;
      extractClaims?: boolean;
    } = {},
  ): Promise<{
    ok: boolean;
    signal_id: string;
    genre: string;
    type: string;
    entities: string[];
    source_package_id?: string;
  }> {
    const workspace = opts.workspace ?? this.defaultWorkspace;
    return this.http.post("/api/ingest", {
      text,
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.genre !== undefined ? { genre: opts.genre } : {}),
      ...(opts.title !== undefined ? { title: opts.title } : {}),
      ...(opts.node !== undefined ? { node: opts.node } : {}),
      ...(opts.extractClaims !== undefined
        ? { extract_claims: opts.extractClaims }
        : {}),
    });
  }

  /**
   * Assemble budget-aware, L0-L3 tiered context for a query via MCTS selection.
   * This is what an agent/app should call to get a Context Package instead of
   * loose search hits.
   */
  assemble(
    query: string,
    opts: {
      workspace?: string;
      tierBudgets?: { l0: number; l1: number; l2: number };
    } = {},
  ): Promise<{
    l0: unknown;
    l1: string;
    l2: string;
    l3?: string;
    total_tokens: number;
    sources: unknown[];
    mcts_metadata?: {
      candidate_count: number;
      selected_sources: unknown[];
      mcts_enabled: boolean;
    };
  }> {
    const workspace = opts.workspace ?? this.defaultWorkspace;
    return this.http.post("/api/assemble", {
      query,
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.tierBudgets !== undefined
        ? { tier_budgets: opts.tierBudgets }
        : {}),
    });
  }

  /**
   * Upsert a topology node (person, entity, project, ...). Node names feed the
   * gazetteer that drives entity extraction and semantic graph edges, so seed
   * a workspace's people/orgs here.
   */
  createNode(
    name: string,
    kind: string,
    opts: { workspace?: string; slug?: string; description?: string } = {},
  ): Promise<{ ok: boolean; id: string; slug: string; kind: string }> {
    const workspace = opts.workspace ?? this.defaultWorkspace;
    return this.http.post("/api/nodes", {
      name,
      kind,
      ...(workspace !== undefined ? { workspace } : {}),
      ...(opts.slug !== undefined ? { slug: opts.slug } : {}),
      ...(opts.description !== undefined
        ? { description: opts.description }
        : {}),
    });
  }

  // ---------------------------------------------------------------------------
  // System
  // ---------------------------------------------------------------------------

  status(): Promise<StatusResult> {
    return this.http.get<StatusResult>("/api/status");
  }

  architectures(): Promise<ArchitectureEntry[]> {
    return this.http.get<ArchitectureEntry[]>("/api/architectures");
  }
}
