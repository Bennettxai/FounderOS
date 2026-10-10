import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ founderosFetch: vi.fn() }));
vi.mock("$lib/founderos/api", async (orig) => ({
  ...(await orig<typeof import("$lib/founderos/api")>()),
  founderosFetch: api.founderosFetch,
}));

import { FounderosApiError } from "$lib/founderos/api";
import ClientsPage from "../../../../routes/(founderos)/os/clients/+page.svelte";
import { busiestDay, statusStyle, type ClientsPayload } from "./types";

const STATUS_ORDER: ClientsPayload["statusOrder"] = [
  {
    key: "launched",
    meter: "Launched",
    chip: "launched",
    tone: "ok",
    hue: "var(--bn-ok)",
  },
  {
    key: "saved",
    meter: "Saved drafts",
    chip: "saved",
    tone: "accent",
    hue: "var(--bn-accent)",
  },
  {
    key: "launching",
    meter: "Launching",
    chip: "launching",
    tone: "warn",
    hue: "var(--bn-warn)",
  },
  {
    key: "needs_attention",
    meter: "Needs attention",
    chip: "needs attention",
    tone: "err",
    hue: "var(--bn-err)",
  },
];

const days = (n: number) =>
  Array.from({ length: n }, (_, i) => ({
    label: `Sep ${11 + i}`,
    count: i === 12 ? 2 : 0,
  }));

function payload(over: Partial<ClientsPayload> = {}): ClientsPayload {
  return {
    clients: [
      {
        id: "silvio-big-mamas",
        name: "Silvio / Big Mama's",
        service: "Marketing",
        projectName: "Silvio-Big Mamas",
        projectId: "2ffe9fec-9041-4577-83c3-72902a6aae98",
        context: "Marketing for Big Mama's two restaurants.",
        sources: ["packages/brand", "marketing/"],
        retainer: "Terms not recorded",
      },
    ],
    work: [
      {
        id: "w1",
        clientId: "silvio-big-mamas",
        brief: "Game day carousel",
        status: "saved",
        createdAt: "2026-09-23T10:00:00.000Z",
        workspaceId: null,
        detail: null,
      },
      {
        id: "w2",
        clientId: "silvio-big-mamas",
        brief: "Menu refresh copy",
        status: "needs_attention",
        createdAt: "2026-09-22T10:00:00.000Z",
        workspaceId: null,
        detail: "Launch could not be confirmed.",
      },
    ],
    volume: {
      headline: 2,
      counts: { saved: 1, launching: 0, launched: 0, needs_attention: 1 },
      chips: [
        { tone: "accent", text: "1 saved" },
        { tone: "err", text: "1 needs attention" },
      ],
      caption: "across 1 confirmed client · 1 with requests",
      meters: STATUS_ORDER.map((s) => ({
        label: `${s.meter} (0)`,
        frac: 0,
        display: "0%",
        hue: s.hue,
      })),
      foot: "draft first · nothing publishes automatically",
      series: days(14),
      requestsInWindow: 2,
      rhythm: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map(
        (label) => ({ label, count: label === "Tue" ? 2 : 0 }),
      ),
      perClient: [{ id: "silvio-big-mamas", count: 2 }],
      insight: {
        value: 1,
        headline: "1 needs attention · 0 launches unconfirmed.",
        body: "Menu refresh copy",
        frac: 0.5,
      },
    },
    windowDays: 14,
    statusOrder: STATUS_ORDER,
    slackBridge: "not activated",
    launchEnabled: false,
    ...over,
  };
}

// mockClear, not mockReset: under vitest 4 a reset mock's later async rejection
// is reported as a test failure even when the page catches it.
beforeEach(() => {
  api.founderosFetch.mockClear();
});
afterEach(() => vi.mocked(window.confirm)?.mockRestore?.());

describe("/os/clients page", () => {
  it("shows loading, then the slab with the hero, second row and board", async () => {
    api.founderosFetch.mockResolvedValue(payload());
    const { container } = render(ClientsPage);
    expect(screen.getByText("Loading client work…")).toBeTruthy();
    await waitFor(() =>
      expect(screen.getByText("Request Volume")).toBeTruthy(),
    );
    expect(api.founderosFetch).toHaveBeenCalledWith("/pages/clients");
    expect(container.querySelector("h1")!.textContent).toBe("Clients");
    expect(screen.getByText("1 confirmed · 2 requests")).toBeTruthy();
    for (const t of [
      "Request Activity",
      "Request Rhythm",
      "Roster",
      "Needs you",
      "Work requests",
      "New request",
    ])
      expect(screen.getAllByText(t).length).toBeGreaterThan(0);
    // the rhythm's busiest day and the roster chip with its request count
    expect(screen.getByText("busiest day · 2 requests")).toBeTruthy();
    expect(screen.getByText("Silvio / Big Mama's · 2")).toBeTruthy();
    expect(screen.getByText("Slack bridge off")).toBeTruthy();
    expect(
      screen.getByText("Menu refresh copy", { selector: "p" }),
    ).toBeTruthy();
  });

  it("an unreachable backend reads as an error, never an empty board", async () => {
    api.founderosFetch.mockImplementation(async () => {
      throw new FounderosApiError(503, "client work is unreadable: db down");
    });
    render(ClientsPage);
    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toContain("db down"),
    );
    expect(screen.queryByText("No requests yet.")).toBeNull();
  });

  it("no requests: honest empty states", async () => {
    const p = payload({ work: [] });
    p.volume = {
      ...p.volume,
      headline: 0,
      chips: [],
      meters: [],
      requestsInWindow: 0,
      series: days(14).map((d) => ({ ...d, count: 0 })),
      rhythm: p.volume.rhythm.map((d) => ({ ...d, count: 0 })),
    };
    api.founderosFetch.mockResolvedValue(p);
    render(ClientsPage);
    await waitFor(() =>
      expect(screen.getByText("No requests yet.")).toBeTruthy(),
    );
    expect(
      screen.getByText("no requests yet, save a brief below"),
    ).toBeTruthy();
    expect(screen.getByText("No requests in the last 14 days.")).toBeTruthy();
    expect(screen.getByText("no requests in 14 days")).toBeTruthy();
  });
});

describe("the client board keeps every action", () => {
  it("saves a brief through /pages/clients/work and reloads", async () => {
    api.founderosFetch.mockResolvedValue(payload());
    render(ClientsPage);
    await waitFor(() => screen.getByText("Save draft request"));
    api.founderosFetch
      .mockResolvedValueOnce({ work: { status: "saved" } })
      .mockResolvedValueOnce(payload());
    await fireEvent.input(screen.getByLabelText("What should we make?"), {
      target: { value: "Draft five posts" },
    });
    await fireEvent.submit(document.getElementById("new-request")!);
    await waitFor(() =>
      expect(screen.getByRole("status").textContent).toContain("Saved."),
    );
    expect(api.founderosFetch).toHaveBeenCalledWith(
      "/pages/clients/work",
      expect.objectContaining({
        method: "POST",
        json: {
          action: "save",
          clientId: "silvio-big-mamas",
          brief: "Draft five posts",
        },
      }),
    );
    expect(api.founderosFetch).toHaveBeenLastCalledWith("/pages/clients");
  });

  it("launch needs the operator token and a confirm, and shows the bridge refusal", async () => {
    api.founderosFetch.mockResolvedValue(payload());
    render(ClientsPage);
    const start = await waitFor(
      () => screen.getByText("Start draft in Superset") as HTMLButtonElement,
    );
    expect(start.disabled).toBe(true);
    expect(screen.getByText(/Launching is off on the bridge/)).toBeTruthy();
    await fireEvent.input(document.querySelector('input[type="password"]')!, {
      target: { value: "t".repeat(32) },
    });
    expect(start.disabled).toBe(false);
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    await fireEvent.click(start);
    expect(confirm).toHaveBeenCalled();
    expect(api.founderosFetch).toHaveBeenCalledTimes(1);
    confirm.mockReturnValue(true);
    api.founderosFetch.mockRejectedValueOnce(
      new FounderosApiError(503, "x", {
        error: "Launching is off on the bridge (FOUNDEROS_WRITES=0).",
      }),
    );
    await fireEvent.click(start);
    await waitFor(() =>
      expect(screen.getByRole("status").textContent).toContain(
        "FOUNDEROS_WRITES=0",
      ),
    );
    const call = api.founderosFetch.mock.calls.find(
      (c) => c[0] === "/pages/clients/work",
    )!;
    expect(call[1]).toMatchObject({
      json: { action: "launch", id: "w1" },
      headers: { Authorization: `Bearer ${"t".repeat(32)}` },
    });
  });

  it("filters requests by status", async () => {
    api.founderosFetch.mockResolvedValue(payload());
    const { container } = render(ClientsPage);
    await waitFor(() => screen.getByText("All 2"));
    expect(container.querySelectorAll('[data-part="request"]').length).toBe(2);
    await fireEvent.click(screen.getByText("Needs attention 1"));
    expect(container.querySelectorAll('[data-part="request"]').length).toBe(1);
    await fireEvent.click(screen.getByText("Launched 0"));
    expect(screen.getByText("Nothing matches that filter.")).toBeTruthy();
  });
});

// FounderOS v1 tests/content-clients-slab.test.ts (the /clients half): the page
// wears the Brand Deals slab.
describe("/clients in the Brand Deals look", () => {
  const view = readFileSync(resolve(__dirname, "ClientsView.svelte"), "utf8");
  const board = readFileSync(
    resolve(__dirname, "ClientWorkspaceBoard.svelte"),
    "utf8",
  );
  const idx = (src: string) =>
    [...src.matchAll(/<(?:SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) =>
      Number(m[1]),
    );

  it("composes the slab kit instead of the console header", () => {
    for (const piece of [
      "<Slab>",
      "<SlabTitle",
      "<SlabCard",
      "<BigStat",
      "<MeterStack",
      "<InsightCard",
    ])
      expect(view, piece).toContain(piece);
    expect(view).not.toContain("<PageHeader");
    expect(view).not.toContain("<h1");
  });

  it("hero is the 2fr/1fr split with the volume card on the right; second row carries the step line and dot matrix", () => {
    expect(view).toContain("grid-cols-[2fr_1fr]");
    expect(view).toContain('title="Request Volume"');
    expect(view).toContain("<StepLine");
    expect(view).toContain("<DotMatrix");
  });

  it("exactly one gradient insight card; distinct stagger; no raw hex", () => {
    expect((view.match(/<InsightCard/g) ?? []).length).toBe(1);
    const i = idx(view);
    expect(i.length).toBeGreaterThan(3);
    expect(new Set(i).size).toBe(i.length);
    expect(view + board).not.toMatch(/#[0-9a-f]{6}\b/i);
    expect(view + board).not.toMatch(/transition-(colors|all)\b/);
  });

  it("the board cards follow the page stagger and keep every action", () => {
    const b = idx(board);
    expect(b.length).toBeGreaterThanOrEqual(3);
    expect(Math.min(...b)).toBeGreaterThan(Math.max(...idx(view)));
    expect(new Set(b).size).toBe(b.length);
    for (const k of [
      "action: 'save'",
      "action: 'launch'",
      "window.confirm",
      "Start draft in Superset",
      "Save draft request",
      'type="password"',
      "Slack bridge",
      'role="status"',
      "/pages/clients/work",
    ])
      expect(board, k).toContain(k);
    // an arrow handler's `=>` ends a naive tag match, so read up to the class attribute
    const buttons = board
      .split("<button")
      .slice(1)
      .map((s) => s.slice(0, s.indexOf("class=") + 160));
    expect(buttons.length).toBeGreaterThan(2);
    for (const btn of buttons) expect(btn).toMatch(/pressable|class={chip\(/);
  });
});

describe("helpers", () => {
  it("busiestDay takes the first max; statusStyle colors by status only", () => {
    expect(
      busiestDay([
        { label: "Mon", count: 1 },
        { label: "Tue", count: 3 },
        { label: "Wed", count: 3 },
      ])?.label,
    ).toBe("Tue");
    expect(busiestDay([])).toBeNull();
    expect(statusStyle(STATUS_ORDER, "launched")).toContain("var(--bn-ok)");
  });
});

// Round 2 (prod 2026-10 snapshot): the whole page in prod's rounded furniture.
describe("/os/clients round 2: prod shapes and copy", () => {
  it("New request is the slab pill; the roster card and Slack box are 12px rounded", async () => {
    api.founderosFetch.mockResolvedValue(payload());
    const { container } = render(ClientsPage);
    const link = await screen.findByRole("link", { name: "New request" });
    expect(link.className).toContain("bn-pill");
    expect(link.className).toContain("rounded-full");
    const pick = screen.getByRole("button", { name: /Silvio \/ Big Mama's Marketing/ });
    expect(pick.className).toMatch(/rounded-\[12px\]/);
    expect(pick.className).toContain("bn-pressable");
    const slack = screen.getByText("Slack bridge").closest("div")!.parentElement as HTMLElement;
    expect(slack.className).toMatch(/rounded-\[12px\]/);
    expect(screen.getByText("not activated").className).toContain("rounded-full");
    expect(container.querySelector("textarea")!.className).toMatch(/rounded-\[12px\]/);
  });

  it("save is the round accent pill; filters are the slab filter chips; status and launch round", async () => {
    api.founderosFetch.mockResolvedValue(payload());
    const { container } = render(ClientsPage);
    await waitFor(() => screen.getByText("All 2"));
    expect(screen.getByRole("button", { name: "Save draft request" }).className).toContain("rounded-full");
    const all = screen.getByRole("button", { name: "All 2" });
    expect(all.className).toContain("bn-filter-chip");
    expect(all.className).toContain("is-on");
    expect(screen.getByRole("button", { name: "Launched 0" }).className).not.toContain("is-on");
    const req = container.querySelector('[data-part="request"] span') as HTMLElement;
    expect(req.className).toContain("rounded-full");
    expect(screen.getByRole("button", { name: "Start draft in Superset" }).className).toContain("rounded-full");
    expect((container.querySelector('input[type="password"]') as HTMLElement).className).toMatch(/rounded-\[12px\]/);
  });

  it("the launch note reads prod's words: the OS server's local Superset installation", async () => {
    api.founderosFetch.mockResolvedValue(payload());
    render(ClientsPage);
    expect(await screen.findByText(/Launches use the OS server’s local Superset installation\./)).toBeTruthy();
  });
});
