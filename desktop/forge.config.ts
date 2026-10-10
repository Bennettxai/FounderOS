import type { ForgeConfig } from "@electron-forge/shared-types";
import { MakerSquirrel } from "@electron-forge/maker-squirrel";
import { MakerZIP } from "@electron-forge/maker-zip";
import { MakerDeb } from "@electron-forge/maker-deb";
import { MakerRpm } from "@electron-forge/maker-rpm";
import { MakerDMG } from "@electron-forge/maker-dmg";
import { VitePlugin } from "@electron-forge/plugin-vite";
import { AutoUnpackNativesPlugin } from "@electron-forge/plugin-auto-unpack-natives";
import { PublisherGithub } from "@electron-forge/publisher-github";
import { rebuild } from "@electron/rebuild";
import fs from "node:fs";
import path from "node:path";

// @electron-forge/plugin-vite bundles app code but does NOT copy node_modules,
// so externalized modules (native better-sqlite3, electron-store) go missing and
// the app crashes at require(). This copies those modules + their full dependency
// trees into the packaged app; AutoUnpackNativesPlugin then unpacks the .node.
function copyModuleTree(
  name: string,
  srcNM: string,
  destNM: string,
  seen = new Set<string>(),
) {
  if (seen.has(name)) return;
  seen.add(name);
  const src = path.join(srcNM, name);
  if (!fs.existsSync(src)) return;
  fs.cpSync(src, path.join(destNM, name), {
    recursive: true,
    dereference: true,
  });
  try {
    const pkg = JSON.parse(
      fs.readFileSync(path.join(src, "package.json"), "utf8"),
    );
    for (const dep of Object.keys(pkg.dependencies || {}))
      copyModuleTree(dep, srcNM, destNM, seen);
  } catch {
    /* no package.json deps to follow */
  }
}

const config: ForgeConfig = {
  hooks: {
    // Copy externalized native/runtime modules into the package (see comment above).
    packageAfterCopy: async (...args: unknown[]) => {
      const [buildPath] = typeof args[0] === "string" ? args : args.slice(1);
      const srcNM = path.join(process.cwd(), "node_modules");
      const destNM = path.join(String(buildPath), "node_modules");
      fs.mkdirSync(destNM, { recursive: true });
      for (const m of ["better-sqlite3", "electron-store", "node-pty"]) {
        copyModuleTree(m, srcNM, destNM);
      }
    },
    packageAfterPrune: async (...args: unknown[]) => {
      const [buildPath, electronVersion, platform, arch] =
        typeof args[0] === "string" ? args : args.slice(1);
      await rebuild({
        buildPath: String(buildPath),
        electronVersion: String(electronVersion),
        arch: String(arch),
        onlyModules: ["better-sqlite3", "node-pty"],
        force: true,
      });
      console.log(
        `[package] rebuilt native modules for Electron ${electronVersion} (${platform}/${arch})`,
      );
    },
  },
  packagerConfig: {
    asar: true,
    icon: "./resources/icons/icon",
    appBundleId: "com.businessos.desktop",
    appCopyright: "Copyright © 2025 BusinessOS",
    extraResource: [
      "./resources/bin",
      "./resources/engine",
      "./resources/icons",
    ],
    // Register the businessos:// deep-link scheme so the OS routes OAuth
    // callbacks (and other deep links) back into the packaged app.
    protocols: [
      {
        name: "BusinessOS",
        schemes: ["businessos"],
      },
    ],
    // Merge additional Info.plist settings (for permission descriptions)
    extendInfo: "./resources/Info.plist",
    // Code signing for macOS (configure via environment variables)
    osxSign: process.env.APPLE_ID
      ? {
          identity: process.env.APPLE_IDENTITY,
          // Use custom entitlements for required permissions
          entitlements: "./resources/entitlements.mac.plist",
          "entitlements-inherit": "./resources/entitlements.mac.plist",
          // Enable hardened runtime (required for notarization)
          hardenedRuntime: true,
          // Gate keeper will check these
          "gatekeeper-assess": false,
        }
      : undefined,
    osxNotarize: process.env.APPLE_ID
      ? {
          appleId: process.env.APPLE_ID,
          appleIdPassword: process.env.APPLE_PASSWORD,
          teamId: process.env.APPLE_TEAM_ID,
        }
      : undefined,
  },
  rebuildConfig: {},
  makers: [
    new MakerSquirrel({
      name: "BusinessOS",
      setupIcon: "./resources/icons/icon.ico",
    }),
    new MakerZIP({}, ["darwin"]),
    new MakerDMG({
      icon: "./resources/icons/icon.icns",
      background: "./resources/dmg-background.png",
      // 720x540 window matches the 2x branded background; icons sit on the
      // "drag arrow" the background draws (app on the left, Applications right).
      additionalDMGOptions: {
        window: { size: { width: 720, height: 540 } },
      },
      contents: (opts: { appPath: string }) => [
        { x: 200, y: 280, type: "file", path: opts.appPath },
        { x: 520, y: 280, type: "link", path: "/Applications" },
      ],
    }),
    new MakerRpm({
      options: {
        icon: "./resources/icons/icon.png",
        bin: "BusinessOS",
        // Adds StartupWMClass so docks match the running window to the entry
        desktopTemplate: "./resources/linux/desktop.ejs",
        // Register the businessos:// deep link (OAuth return path)
        mimeType: ["x-scheme-handler/businessos"],
      },
    }),
    new MakerDeb({
      options: {
        icon: "./resources/icons/icon.png",
        maintainer: "BusinessOS Team",
        homepage: "https://businessos.app",
        // Packaged executable is "BusinessOS" (packagerConfig.name), not the
        // package.json name the maker defaults to.
        bin: "BusinessOS",
        // Adds StartupWMClass so docks match the running window to the entry
        desktopTemplate: "./resources/linux/desktop.ejs",
        // Register the businessos:// deep link (OAuth return path)
        mimeType: ["x-scheme-handler/businessos"],
      },
    }),
  ],
  plugins: [
    new AutoUnpackNativesPlugin({}),
    new VitePlugin({
      // `build` can specify multiple entry builds
      build: [
        {
          // Main process entry point
          entry: "src/main/index.ts",
          config: "vite.main.config.ts",
        },
        {
          // Preload script entry point
          entry: "src/preload/index.ts",
          config: "vite.preload.config.ts",
        },
      ],
      renderer: [
        {
          name: "main_window",
          config: "vite.renderer.config.ts",
        },
      ],
    }),
  ],
  publishers: [
    new PublisherGithub({
      repository: {
        owner: process.env.BUSINESSOS_RELEASE_OWNER || "Miosa-osa",
        name: process.env.BUSINESSOS_RELEASE_REPO || "businessos-5",
      },
      prerelease: false,
      draft: true,
    }),
  ],
};

export default config;
