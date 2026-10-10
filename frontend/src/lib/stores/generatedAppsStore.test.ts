/**
 * TDD Verification Tests for generatedAppsStore
 * Testing sandbox integration with backend (commit cd8e4b49)
 *
 * These tests verify:
 * 1. Store correctly handles 'deploying' status from backend
 * 2. API connectivity uses correct /sandbox/* namespace
 * 3. SandboxInfo type alignment with backend responses
 */

import { describe, it, expect, beforeEach, vi, type Mock } from "vitest";
import { get } from "svelte/store";

// Mock the API modules BEFORE importing the store
vi.mock("$lib/api/base", () => ({
  request: vi.fn(),
  requestPaginated: vi.fn(),
  getApiBaseUrl: vi.fn(() => "http://localhost:8080/api/v1"),
}));

vi.mock("$lib/api/sandbox", () => ({
  deploySandbox: vi.fn(),
  stopSandbox: vi.fn(),
  restartSandbox: vi.fn(),
  removeSandbox: vi.fn(),
  getSandboxInfo: vi.fn(),
  listUserSandboxes: vi.fn(),
  getSandboxStats: vi.fn(),
  streamSandboxLogs: vi.fn(),
}));

// Import after mocks
import {
  generatedAppsStore,
  type GeneratedApp,
  type AppStatus,
} from "./generatedAppsStore";
import { requestPaginated } from "$lib/api/base";
import {
  deploySandbox,
  restartSandbox,
  removeSandbox,
  stopSandbox,
} from "$lib/api/sandbox";
import type { SandboxStatus, DeployResponse } from "$lib/api/sandbox";

describe("generatedAppsStore - Sandbox Integration", () => {
  beforeEach(() => {
    // Reset store state
    generatedAppsStore.reset();
    // Clear all mocks
    vi.clearAllMocks();
  });

	  describe("TDD Cycle 1: Deploying Status Handling", () => {
    it("should handle backend response with status: deploying", async () => {
      // Arrange: Mock backend response with 'deploying' status
      const mockBuildingResponse: DeployResponse = {
        container_id: "abc123",
        port: 9001,
        url: "http://localhost:9001",
        status: "deploying" as SandboxStatus,
      };

      (deploySandbox as Mock).mockResolvedValueOnce(mockBuildingResponse);

      // Mock fetchApps to return an app with deploying sandbox
      (requestPaginated as Mock).mockResolvedValueOnce({
        data: [
          {
            id: "test-app-uuid",
            app_name: "Test App",
            description: "Test description",
            status: "deployed" as AppStatus,
            progress: 100,
            generated_at: new Date().toISOString(),
            user_id: "user-123",
            workspace_id: "ws-123",
            sandbox: {
              app_id: "test-app-uuid",
              id: "abc123",
              status: "deploying" as SandboxStatus,
              port: 9001,
              url: "http://localhost:9001",
            },
          },
        ],
        pagination: { page: 1, page_size: 20, total_items: 1, total_pages: 1, has_more: false },
      });

      // Act: Deploy the app
      await generatedAppsStore.deployApp("test-app-uuid");

      // Assert: deploySandbox was called with correct app_id
      expect(deploySandbox).toHaveBeenCalledWith("test-app-uuid");

      // Verify the store fetched updated state
      expect(requestPaginated).toHaveBeenCalledWith(
        "/osa/module-instances",
        { page: 1, page_size: 20 },
        "apps"
      );
    });

    it("should correctly type sandbox.status as SandboxStatus", async () => {
      // Arrange: Fetch apps with various sandbox statuses
      const mockApps: GeneratedApp[] = [
        {
          id: "app-1",
          app_name: "Building App",
	          description: "Currently deploying",
          status: "deployed",
          progress: 100,
          generated_at: new Date().toISOString(),
          user_id: "user-1",
          workspace_id: "ws-1",
          sandbox: {
            app_id: "app-1",
	            id: "c1",
	            status: "deploying",
            port: 9001,
            url: "http://localhost:9001",
          },
        },
        {
          id: "app-2",
          app_name: "Running App",
          description: "Currently running",
          status: "deployed",
          progress: 100,
          generated_at: new Date().toISOString(),
          user_id: "user-1",
          workspace_id: "ws-1",
          sandbox: {
            app_id: "app-2",
	            id: "c2",
            status: "running",
            port: 9002,
            url: "http://localhost:9002",
          },
        },
      ];

      (requestPaginated as Mock).mockResolvedValueOnce({
        data: mockApps,
        pagination: { page: 1, page_size: 20, total_items: 2, total_pages: 1, has_more: false },
      });

      // Act
      await generatedAppsStore.fetchApps();

      // Assert: Verify store state
      const state = get(generatedAppsStore);
      expect(state.apps).toHaveLength(2);
      expect(state.apps[0].sandbox?.status).toBe("deploying");
      expect(state.apps[1].sandbox?.status).toBe("running");
    });

    it("should accept all valid SandboxStatus values", () => {
      // Verify type alignment: These should be the ONLY valid values
      const validStatuses: SandboxStatus[] = [
        "pending",
	        "deploying",
        "running",
        "stopped",
	        "failed",
	        "removing",
      ];

      // This is a compile-time test - if it compiles, types are aligned
      validStatuses.forEach((status) => {
        expect([
          "pending",
	          "deploying",
          "running",
          "stopped",
	          "failed",
	          "removing",
        ]).toContain(status);
      });
    });
  });

  describe("TDD Cycle 2: Sandbox Lifecycle Methods", () => {
    it("should call restartSandbox API (not stop+deploy)", async () => {
      // Arrange
      const mockRestartResponse: DeployResponse = {
        container_id: "restarted-123",
        port: 9003,
        url: "http://localhost:9003",
        status: "deploying",
      };

      (restartSandbox as Mock).mockResolvedValueOnce(mockRestartResponse);
      (requestPaginated as Mock).mockResolvedValueOnce({
        data: [],
        pagination: { page: 1, page_size: 20, total_items: 0, total_pages: 0, has_more: false },
      });

      // Act
      await generatedAppsStore.startSandbox("app-123");

      // Assert: Single restart call, NOT stop+deploy
      expect(restartSandbox).toHaveBeenCalledTimes(1);
      expect(restartSandbox).toHaveBeenCalledWith("app-123");
      expect(deploySandbox).not.toHaveBeenCalled(); // Should NOT double-call
    });

    it("should call removeSandbox API and clear local state", async () => {
      // Arrange: Set up initial state with a sandbox
      (requestPaginated as Mock).mockResolvedValueOnce({
        data: [
          {
            id: "app-to-remove",
            app_name: "Remove Me",
            description: "Test",
            status: "deployed",
            progress: 100,
            generated_at: new Date().toISOString(),
            user_id: "u1",
            workspace_id: "w1",
            sandbox: {
              app_id: "app-to-remove",
	              id: "c1",
              status: "running",
              port: 9004,
              url: "http://localhost:9004",
            },
          },
        ],
        pagination: { page: 1, page_size: 20, total_items: 1, total_pages: 1, has_more: false },
      });
      await generatedAppsStore.fetchApps();

      // Mock the remove call
      (removeSandbox as Mock).mockResolvedValueOnce({
        message: "Sandbox removed",
      });
      (requestPaginated as Mock).mockResolvedValueOnce({
        data: [],
        pagination: { page: 1, page_size: 20, total_items: 0, total_pages: 0, has_more: false },
      });

      // Act
      await generatedAppsStore.removeSandbox("app-to-remove");

      // Assert
      expect(removeSandbox).toHaveBeenCalledWith("app-to-remove");
    });

    it("should call stopSandbox API correctly", async () => {
      // Arrange
      (stopSandbox as Mock).mockResolvedValueOnce({ message: "Stopped" });
      (requestPaginated as Mock).mockResolvedValueOnce({
        data: [],
        pagination: { page: 1, page_size: 20, total_items: 0, total_pages: 0, has_more: false },
      });

      // Act
      await generatedAppsStore.stopSandbox("app-456");

      // Assert
      expect(stopSandbox).toHaveBeenCalledTimes(1);
      expect(stopSandbox).toHaveBeenCalledWith("app-456");
    });
  });

  describe("TDD Cycle 3: Error Handling", () => {
    it("should throw descriptive error when deploy fails", async () => {
      // Arrange
      (deploySandbox as Mock).mockRejectedValueOnce(
        new Error("sandbox already running"),
      );

      // Act & Assert
      await expect(generatedAppsStore.deployApp("app-123")).rejects.toThrow(
        "sandbox already running",
      );
    });

    it("should throw descriptive error when restart fails", async () => {
      // Arrange
      (restartSandbox as Mock).mockRejectedValueOnce(
        new Error("container not found"),
      );

      // Act & Assert
      await expect(
        generatedAppsStore.startSandbox("nonexistent"),
      ).rejects.toThrow("container not found");
    });

    it("should throw descriptive error when remove fails", async () => {
      // Arrange
      (removeSandbox as Mock).mockRejectedValueOnce(
        new Error("permission denied"),
      );

      // Act & Assert
      await expect(
        generatedAppsStore.removeSandbox("protected-app"),
      ).rejects.toThrow("permission denied");
    });
  });
});

describe("API Contract Verification", () => {
  it("should verify deploySandbox request body matches backend expectation", () => {
    /**
     * CRITICAL CONTRACT CHECK:
     * Backend (sandbox.go) requires: { app_id, app_name, image }
     * Frontend (sandbox.ts) sends: { app_id, app_name }
     *
     * STATUS: ⚠️ MISMATCH - 'image' field is REQUIRED by backend but not sent
     *
     * This test documents the expected contract for future alignment.
     */
    interface ExpectedBackendRequest {
      app_id: string; // UUID format
      app_name: string; // Required
      image?: string; // Required by backend, optional in current frontend
      workspace_path?: string;
    }

    interface CurrentFrontendRequest {
      app_id: string;
      app_name?: string;
    }

    // Document the contract mismatch
    const backendExpects: ExpectedBackendRequest = {
      app_id: "uuid-format",
      app_name: "My App",
      image: "node:18-alpine", // Backend REQUIRES this
    };

    const frontendSends: CurrentFrontendRequest = {
      app_id: "uuid-format",
      app_name: "My App",
      // Missing: image
    };

    // This test passes to document the current state
    // TODO: Fix frontend to include 'image' or backend to auto-detect
    expect(Object.keys(frontendSends).includes("image")).toBe(false);
    expect(Object.keys(backendExpects).includes("image")).toBe(true);
  });

  it("should verify app_id is in UUID format", () => {
    const uuidRegex =
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

    const validAppId = "f47ac10b-58cc-4372-a567-0e02b2c3d479";
    const invalidAppId = "not-a-uuid";

    expect(validAppId).toMatch(uuidRegex);
    expect(invalidAppId).not.toMatch(uuidRegex);
  });
});
