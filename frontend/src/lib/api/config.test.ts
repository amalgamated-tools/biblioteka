import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  vi,
  type Mock,
} from "vitest";
import {
  getConfigStatus,
  getOidcConfig,
  setOidcConfig,
  getSmtpConfig,
  setSmtpConfig,
  testSmtpConfig,
  getWatchFolderConfig,
  setWatchFolderConfig,
  getLLMConfig,
  setLLMConfig,
  clearToken,
} from "../api";
import type {
  ConfigStatus,
  LLMConfig,
  OIDCConfig,
  SetLLMConfigInput,
  SetOIDCConfigInput,
  SMTPConfig,
  SetSMTPConfigInput,
  WatchFolderConfig,
  SetWatchFolderConfigInput,
} from "../../types";
import { mockFetchResponse as _mockFetchResponse } from "./testUtils";

let fetchMock: Mock;

function mockFetchResponse(body: unknown, status = 200) {
  _mockFetchResponse(fetchMock, body, status);
}

const fakeConfigStatus: ConfigStatus = {
  oidc_configured: true,
  smtp_configured: false,
  is_admin: true,
};

const fakeOidcConfig: OIDCConfig = {
  issuer_url: "https://accounts.example.com",
  client_id: "my-client",
  client_secret_set: true,
  redirect_uri: "https://app.example.com/callback",
};

const fakeSetOidcInput: SetOIDCConfigInput = {
  issuer_url: "https://accounts.example.com",
  client_id: "my-client",
  client_secret: "s3cr3t",
  redirect_uri: "https://app.example.com/callback",
};

const fakeSmtpConfig: SMTPConfig = {
  host: "smtp.example.com",
  port: "587",
  username: "user@example.com",
  password_set: true,
  from: "noreply@example.com",
  tls: "starttls",
  env_override: false,
};

const fakeSetSmtpInput: SetSMTPConfigInput = {
  host: "smtp.example.com",
  port: "587",
  username: "user@example.com",
  password: "hunter2",
  from: "noreply@example.com",
  tls: "starttls",
};

const fakeWatchFolderConfig: WatchFolderConfig = {
  path: "/books",
  library_id: "lib1",
};

const fakeSetWatchFolderInput: SetWatchFolderConfigInput = {
  path: "/books",
  library_id: "lib1",
};

const fakeLLMConfig: LLMConfig = {
  provider: "ollama",
  endpoint: "http://localhost:11434",
  model: "llama3",
  enabled: true,
};

const fakeSetLLMInput: SetLLMConfigInput = {
  provider: "ollama",
  endpoint: "http://localhost:11434",
  model: "llama3",
  enabled: true,
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Config API", () => {
  describe("getConfigStatus", () => {
    it("sends GET /api/config/status and returns the status", async () => {
      mockFetchResponse(fakeConfigStatus);

      const result = await getConfigStatus();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/status");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeConfigStatus);
    });
  });

  describe("getOidcConfig", () => {
    it("sends GET /api/config/oidc and returns the config", async () => {
      mockFetchResponse(fakeOidcConfig);

      const result = await getOidcConfig();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/oidc");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeOidcConfig);
    });
  });

  describe("setOidcConfig", () => {
    it("sends PUT /api/config/oidc with the config body and returns a message", async () => {
      mockFetchResponse({ message: "ok" });

      const result = await setOidcConfig(fakeSetOidcInput);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/oidc");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual(fakeSetOidcInput);
      expect(result).toEqual({ message: "ok" });
    });
  });

  describe("getSmtpConfig", () => {
    it("sends GET /api/config/smtp and returns the config", async () => {
      mockFetchResponse(fakeSmtpConfig);

      const result = await getSmtpConfig();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/smtp");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeSmtpConfig);
    });
  });

  describe("setSmtpConfig", () => {
    it("sends PUT /api/config/smtp with the config body and returns a message", async () => {
      mockFetchResponse({ message: "ok" });

      const result = await setSmtpConfig(fakeSetSmtpInput);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/smtp");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual(fakeSetSmtpInput);
      expect(result).toEqual({ message: "ok" });
    });
  });

  describe("testSmtpConfig", () => {
    it("sends POST /api/config/smtp/test and returns a message", async () => {
      mockFetchResponse({ message: "email sent" });

      const result = await testSmtpConfig();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/smtp/test");
      expect(options.method).toBe("POST");
      expect(result).toEqual({ message: "email sent" });
    });
  });

  describe("getWatchFolderConfig", () => {
    it("sends GET /api/config/watch-folder and returns the config", async () => {
      mockFetchResponse(fakeWatchFolderConfig);

      const result = await getWatchFolderConfig();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/watch-folder");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeWatchFolderConfig);
    });
  });

  describe("setWatchFolderConfig", () => {
    it("sends PUT /api/config/watch-folder with the config body and returns a message", async () => {
      mockFetchResponse({ message: "ok" });

      const result = await setWatchFolderConfig(fakeSetWatchFolderInput);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/watch-folder");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual(fakeSetWatchFolderInput);
      expect(result).toEqual({ message: "ok" });
    });
  });

  describe("getLLMConfig", () => {
    it("sends GET /api/config/llm and returns the config", async () => {
      mockFetchResponse(fakeLLMConfig);

      const result = await getLLMConfig();

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/llm");
      expect(options.method).toBe("GET");
      expect(result).toEqual(fakeLLMConfig);
    });
  });

  describe("setLLMConfig", () => {
    it("sends PUT /api/config/llm with the config body and returns the updated config", async () => {
      const responseWithRestart: LLMConfig = {
        ...fakeLLMConfig,
        restart_required: true,
      };
      mockFetchResponse(responseWithRestart);

      const result = await setLLMConfig(fakeSetLLMInput);

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/config/llm");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual(fakeSetLLMInput);
      expect(result).toEqual(responseWithRestart);
    });
  });
});
