import { definePluginEntry } from "openclaw/plugin-sdk/plugin-entry";
import {
  CLI_FRESH_WATCHDOG_DEFAULTS,
  CLI_RESUME_WATCHDOG_DEFAULTS,
  type CliBackendPlugin,
} from "openclaw/plugin-sdk/cli-backend";

const FRESH_WATCHDOG = {
  ...CLI_FRESH_WATCHDOG_DEFAULTS,
  timeout: 3_600_000,
  maxMissedHeartbeats: 120,
};

const RESUME_WATCHDOG = {
  ...CLI_RESUME_WATCHDOG_DEFAULTS,
  timeout: 3_600_000,
  maxMissedHeartbeats: 120,
};

function buildPinchpassBackend(): CliBackendPlugin {
  return {
    id: "pinchpass",
    liveTest: {
      defaultModelRef: "pinchpass/request",
      defaultImageProbe: false,
      defaultMcpProbe: false,
      docker: {
        npmPackage: "@pinchpass/cli",
        binaryName: "pinchpass",
      },
    },
    config: {
      command: "pinchpass",
      args: ["request", "-json"],
      output: "json",
      input: "arg",
      modelAliases: {
        request: "pinchpass/request",
      },
      reliability: {
        watchdog: {
          fresh: FRESH_WATCHDOG,
          resume: RESUME_WATCHDOG,
        },
      },
    },
  };
}

export default definePluginEntry({
  id: "pinchpass",
  name: "PinchPass",
  description: "One-time E2E-encrypted secret request links",
  register(api) {
    api.registerCliBackend(buildPinchpassBackend());
  },
});
