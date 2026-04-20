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
  previewCalibreImport,
  confirmCalibreImport,
  previewCalibreImportFromPath,
  confirmCalibreImportFromPath,
  clearToken,
  setToken,
} from "../api";
import type { CalibrePreview, CalibreImportResult } from "../../types";
import { mockFetchResponse } from "./testUtils";

let fetchMock: Mock;

function mockFetch(body: unknown, status = 200) {
  mockFetchResponse(fetchMock, body, status);
}

const fakePreview: CalibrePreview = {
  total: 2,
  books: [
    {
      calibre_id: 1,
      title: "Foundation",
      authors: ["Isaac Asimov"],
      series: [{ name: "Foundation", position: 1 }],
      formats: ["epub"],
    },
    {
      calibre_id: 2,
      title: "Dune",
      authors: ["Frank Herbert"],
      series: [],
      publisher: "Chilton Books",
      formats: ["epub", "pdf"],
    },
  ],
};

const fakeImportResult: CalibreImportResult = {
  total: 2,
  imported: 1,
  skipped: 1,
  errors: 0,
};

beforeEach(() => {
  clearToken();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Calibre API", () => {
  describe("previewCalibreImport", () => {
    it("sends POST /api/calibre-import/preview with FormData containing the file", async () => {
      mockFetch(fakePreview);
      const file = new File(["db content"], "metadata.db");

      const result = await previewCalibreImport(file);

      expect(fetchMock).toHaveBeenCalledOnce();
      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/calibre-import/preview");
      expect(options.method).toBe("POST");
      expect(options.body).toBeInstanceOf(FormData);
      const fd = options.body as FormData;
      expect(fd.get("metadata_db")).toBe(file);
      expect(result).toEqual(fakePreview);
    });

    it("forwards the Authorization header when a token is set", async () => {
      setToken("test-token");
      mockFetch(fakePreview);
      const file = new File(["db"], "metadata.db");

      await previewCalibreImport(file);

      const options = fetchMock.mock.calls[0][1] as RequestInit;
      expect((options.headers as Record<string, string>)["Authorization"]).toBe(
        "Bearer test-token",
      );
    });
  });

  describe("confirmCalibreImport", () => {
    it("sends POST /api/calibre-import/confirm with FormData containing file and library_id", async () => {
      mockFetch(fakeImportResult);
      const file = new File(["db"], "metadata.db");

      const result = await confirmCalibreImport(file, "lib-123");

      expect(fetchMock).toHaveBeenCalledOnce();
      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/calibre-import/confirm");
      expect(options.method).toBe("POST");
      expect(options.body).toBeInstanceOf(FormData);
      const fd = options.body as FormData;
      expect(fd.get("metadata_db")).toBe(file);
      expect(fd.get("library_id")).toBe("lib-123");
      expect(result).toEqual(fakeImportResult);
    });

    it("omits library_id from FormData when an empty string is provided", async () => {
      mockFetch(fakeImportResult);
      const file = new File(["db"], "metadata.db");

      await confirmCalibreImport(file, "");

      const fd = fetchMock.mock.calls[0][1].body as FormData;
      expect(fd.get("library_id")).toBeNull();
    });

    it("returns the import result with counts", async () => {
      mockFetch(fakeImportResult);
      const file = new File(["db"], "metadata.db");

      const result = await confirmCalibreImport(file, "lib-1");

      expect(result.total).toBe(2);
      expect(result.imported).toBe(1);
      expect(result.skipped).toBe(1);
      expect(result.errors).toBe(0);
    });
  });

  describe("previewCalibreImportFromPath", () => {
    it("sends POST /api/calibre-import/preview with a JSON path body", async () => {
      mockFetch(fakePreview);

      const result = await previewCalibreImportFromPath(
        "/data/calibre/metadata.db",
      );

      expect(fetchMock).toHaveBeenCalledOnce();
      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/calibre-import/preview");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({
        path: "/data/calibre/metadata.db",
      });
      expect(result).toEqual(fakePreview);
    });
  });

  describe("confirmCalibreImportFromPath", () => {
    it("sends POST /api/calibre-import/confirm with path and library_id", async () => {
      mockFetch(fakeImportResult);

      const result = await confirmCalibreImportFromPath(
        "/data/calibre/metadata.db",
        "lib-42",
      );

      expect(fetchMock).toHaveBeenCalledOnce();
      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toBe("/api/calibre-import/confirm");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({
        path: "/data/calibre/metadata.db",
        library_id: "lib-42",
      });
      expect(result).toEqual(fakeImportResult);
    });

    it("omits library_id from the body when an empty string is provided", async () => {
      mockFetch(fakeImportResult);

      await confirmCalibreImportFromPath("/data/calibre/metadata.db", "");

      const body = JSON.parse(fetchMock.mock.calls[0][1].body);
      expect(body).not.toHaveProperty("library_id");
      expect(body.path).toBe("/data/calibre/metadata.db");
    });
  });
});
