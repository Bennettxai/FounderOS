import { beforeEach, describe, expect, it } from "vitest";
import {
  LOCAL_BACKEND_URL,
  LOCAL_OSA_URL,
  PRODUCTION_BACKEND_URL,
  getApiBaseUrl,
  getBackendUrl,
  getLegacyApiBaseUrl,
  isLocalRuntimeLocation,
  resolveElectronApiBaseUrl,
} from "./runtime";

describe("runtime config", () => {
  beforeEach(() => {
    localStorage.clear();
    Reflect.deleteProperty(window, "electron");
  });

  it("uses relative versioned API URLs on localhost web", () => {
    expect(getApiBaseUrl()).toBe("/api/v1");
  });

  it("uses the local backend as the web backend on localhost", () => {
    expect(getBackendUrl()).toBe(LOCAL_BACKEND_URL);
  });

  it("uses the local OSA API in Electron local mode", () => {
    Object.defineProperty(window, "electron", {
      configurable: true,
      value: {},
    });
    localStorage.setItem("businessos_mode", "local");

    expect(getApiBaseUrl()).toBe(`${LOCAL_OSA_URL}/api/v1`);
    expect(getLegacyApiBaseUrl()).toBe(`${LOCAL_OSA_URL}/api`);
  });

  it("uses the same-origin proxy for Electron cloud mode on localhost", () => {
    Object.defineProperty(window, "electron", {
      configurable: true,
      value: {},
    });
    localStorage.setItem("businessos_mode", "cloud");
    localStorage.setItem("businessos_cloud_url", "https://cloud.example");

    expect(getApiBaseUrl()).toBe("/api/v1");
    expect(getLegacyApiBaseUrl()).toBe("/api");
  });

  it("does not classify the packaged app protocol as a local runtime", () => {
    expect(isLocalRuntimeLocation("app:", "localhost")).toBe(false);
  });

  it("uses the production backend for a packaged app with no stored mode", () => {
    expect(
      resolveElectronApiBaseUrl(
        null,
        null,
        "app:",
        "localhost",
        "v1",
      ),
    ).toBe(`${PRODUCTION_BACKEND_URL}/api/v1`);
    expect(
      resolveElectronApiBaseUrl(null, null, "app:", "localhost", null),
    ).toBe(`${PRODUCTION_BACKEND_URL}/api`);
  });
});
