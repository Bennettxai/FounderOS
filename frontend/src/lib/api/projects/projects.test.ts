import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  addProjectFileLink,
  removeProjectFileLink,
  uploadProjectFile,
} from "./projects";
import type { Project } from "./types";

global.fetch = vi.fn();

const project: Project = {
  id: "project-1",
  name: "Project",
  description: null,
  status: "active",
  priority: "medium",
  client_name: null,
  project_type: "standard",
  project_metadata: { quick_links: [] },
  created_at: "2026-01-01T00:00:00.000Z",
  updated_at: "2026-01-01T00:00:00.000Z",
  notes: [],
};

describe("projects API file helpers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it("uploads project files with multipart form data", async () => {
    const file = new File(["content"], "scope.pdf", {
      type: "application/pdf",
    });
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          name: "scope.pdf",
          size: "0.0 MB",
          url: "/api/projects/project-1/files/scope.pdf",
        }),
      ),
    );

    const result = await uploadProjectFile(project.id, file);

    expect(fetch).toHaveBeenCalledWith(
      "http://localhost:8801/projects/project-1/files",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: expect.any(FormData),
      }),
    );
    expect(result).toEqual({
      name: "scope.pdf",
      size: "0.0 MB",
      url: "/api/projects/project-1/files/scope.pdf",
    });
  });

  it("adds uploaded files to project quick links through project metadata", async () => {
    const file = new File(["content"], "brief.txt");
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            name: "brief.txt",
            size: "0.0 MB",
            url: "/api/projects/project-1/files/brief.txt",
          }),
        ),
      )
      .mockResolvedValueOnce(new Response(JSON.stringify(project)));

    await addProjectFileLink(project, file, []);

    expect(fetch).toHaveBeenLastCalledWith(
      "/api/v1/projects/project-1",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          project_metadata: {
            quick_links: [
              {
                name: "brief.txt",
                size: "0.0 MB",
                url: "/api/projects/project-1/files/brief.txt",
              },
            ],
          },
        }),
      }),
    );
  });

  it("removes quick links by index through project metadata", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(new Response(JSON.stringify(project)));

    await removeProjectFileLink(project, 0, [
      { name: "remove.txt", size: "1.0 MB", url: "/remove" },
      { name: "keep.txt", size: "1.0 MB", url: "/keep" },
    ]);

    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/projects/project-1",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({
          project_metadata: {
            quick_links: [{ name: "keep.txt", size: "1.0 MB", url: "/keep" }],
          },
        }),
      }),
    );
  });
});
